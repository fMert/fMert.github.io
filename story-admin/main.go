package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Story struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Image     string `json:"image,omitempty"`
	BG        string `json:"bg,omitempty"`
	Title     string `json:"title,omitempty"`
	Text      string `json:"text"`
	Subtext   string `json:"subtext,omitempty"`
	Link      string `json:"link,omitempty"`
	LinkLabel string `json:"link_label,omitempty"`
	Posted    string `json:"posted"`
}

type app struct {
	dataDir  string
	password string
	mu       sync.Mutex
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		response, err := http.Get("http://127.0.0.1:8081/health")
		if err != nil || response.StatusCode != http.StatusNoContent {
			os.Exit(1)
		}
		response.Body.Close()
		return
	}
	a := &app{dataDir: env("DATA_DIR", "/data"), password: os.Getenv("STORY_PASSWORD")}
	if a.password == "" {
		log.Fatal("STORY_PASSWORD is required")
	}
	if err := os.MkdirAll(filepath.Join(a.dataDir, "media"), 0755); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /stories-api", a.list)
	mux.HandleFunc("POST /stories-api", a.create)
	mux.HandleFunc("POST /stories-api/delete", a.delete)
	mux.HandleFunc("GET /stories-admin", a.admin)
	mux.HandleFunc("POST /stories-login", a.login)
	mux.HandleFunc("POST /stories-logout", a.logout)
	mux.Handle("GET /story-media/", http.StripPrefix("/story-media/", http.FileServer(http.Dir(filepath.Join(a.dataDir, "media")))))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	log.Println("story admin listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}

func (a *app) list(w http.ResponseWriter, _ *http.Request) {
	stories, err := a.load()
	if err != nil {
		http.Error(w, "Could not load stories", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(stories)
}

func (a *app) admin(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		loginPage.Execute(w, r.URL.Query().Has("error"))
		return
	}
	stories, err := a.load()
	if err != nil {
		http.Error(w, "Could not load stories", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	adminPage.Execute(w, stories)
}

func (a *app) create(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(r) {
		http.Redirect(w, r, "/stories-admin", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		http.Error(w, "Upload is too large (maximum 10 MB)", http.StatusBadRequest)
		return
	}

	s := Story{
		Type:      r.FormValue("type"),
		BG:        strings.TrimSpace(r.FormValue("bg")),
		Title:     strings.TrimSpace(r.FormValue("title")),
		Text:      strings.TrimSpace(r.FormValue("text")),
		Subtext:   strings.TrimSpace(r.FormValue("subtext")),
		Link:      strings.TrimSpace(r.FormValue("link")),
		LinkLabel: strings.TrimSpace(r.FormValue("link_label")),
		Posted:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := validate(s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.ID = newID()

	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		ext, err := saveImage(filepath.Join(a.dataDir, "media"), s.ID, file, header.Size)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.Image = "/story-media/" + s.ID + ext
	} else if !errors.Is(err, http.ErrMissingFile) {
		http.Error(w, "Could not read image", http.StatusBadRequest)
		return
	}
	if s.Type == "image" && s.Image == "" {
		http.Error(w, "Image stories need an image", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	stories, err := a.loadUnlocked()
	if err == nil {
		stories = append([]Story{s}, stories...)
		err = a.saveUnlocked(stories)
	}
	if err != nil {
		http.Error(w, "Could not save story", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/stories-admin", http.StatusSeeOther)
}

func (a *app) delete(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(r) {
		http.Redirect(w, r, "/stories-admin", http.StatusSeeOther)
		return
	}
	id := r.FormValue("id")
	a.mu.Lock()
	defer a.mu.Unlock()
	stories, err := a.loadUnlocked()
	if err != nil {
		http.Error(w, "Could not load stories", http.StatusInternalServerError)
		return
	}
	kept := stories[:0]
	for _, s := range stories {
		if s.ID == id {
			if strings.HasPrefix(s.Image, "/story-media/") {
				os.Remove(filepath.Join(a.dataDir, "media", filepath.Base(s.Image)))
			}
			continue
		}
		kept = append(kept, s)
	}
	if err := a.saveUnlocked(kept); err != nil {
		http.Error(w, "Could not delete story", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/stories-admin", http.StatusSeeOther)
}

func (a *app) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil || subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(a.password)) != 1 {
		http.Redirect(w, r, "/stories-admin?error=1", http.StatusSeeOther)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "story_session", Value: a.sessionToken(), Path: "/", MaxAge: 7 * 24 * 60 * 60, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/stories-admin", http.StatusSeeOther)
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "story_session", Path: "/", MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/stories-admin", http.StatusSeeOther)
}

func (a *app) sessionToken() string {
	expires := strconv.FormatInt(time.Now().Add(7*24*time.Hour).Unix(), 10)
	mac := hmac.New(sha256.New, []byte(a.password))
	mac.Write([]byte(expires))
	return expires + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *app) authorized(r *http.Request) bool {
	cookie, err := r.Cookie("story_session")
	if err != nil {
		return false
	}
	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	expires, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || expires < time.Now().Unix() {
		return false
	}
	mac := hmac.New(sha256.New, []byte(a.password))
	mac.Write([]byte(parts[0]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	return err == nil && hmac.Equal(signature, mac.Sum(nil))
}

func (a *app) load() ([]Story, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.loadUnlocked()
}

func (a *app) loadUnlocked() ([]Story, error) {
	b, err := os.ReadFile(filepath.Join(a.dataDir, "stories.json"))
	if errors.Is(err, os.ErrNotExist) {
		return []Story{}, nil
	}
	if err != nil {
		return nil, err
	}
	var stories []Story
	return stories, json.Unmarshal(b, &stories)
}

func (a *app) saveUnlocked(stories []Story) error {
	b, err := json.MarshalIndent(stories, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(a.dataDir, "stories.json.tmp")
	if err := os.WriteFile(tmp, append(b, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(a.dataDir, "stories.json"))
}

func validate(s Story) error {
	if s.Type != "text" && s.Type != "image" && s.Type != "link" {
		return errors.New("invalid story type")
	}
	if s.Text == "" || len(s.Text) > 1000 || len(s.Title) > 120 || len(s.Subtext) > 1000 || len(s.BG) > 500 || len(s.LinkLabel) > 80 {
		return errors.New("required text is missing or a field is too long")
	}
	if s.Link != "" {
		u, err := url.Parse(s.Link)
		if err != nil || strings.HasPrefix(s.Link, "//") || (!strings.HasPrefix(s.Link, "/") && u.Scheme != "http" && u.Scheme != "https") {
			return errors.New("link must start with /, http://, or https://")
		}
	}
	return nil
}

func saveImage(dir, id string, src io.Reader, size int64) (string, error) {
	if size > 10<<20 {
		return "", errors.New("image is too large (maximum 10 MB)")
	}
	b, err := io.ReadAll(io.LimitReader(src, 10<<20+1))
	if err != nil || len(b) > 10<<20 {
		return "", errors.New("image is too large (maximum 10 MB)")
	}
	extensions := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}
	ext, ok := extensions[http.DetectContentType(b)]
	if !ok {
		return "", errors.New("image must be JPEG, PNG, WebP, or GIF")
	}
	return ext, os.WriteFile(filepath.Join(dir, id+ext), b, 0644)
}

func newID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return time.Now().UTC().Format("20060102-150405-") + hex.EncodeToString(b)
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

var loginPage = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Hikâye girişi</title><style>
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;background:#101018;color:#eee;font:16px system-ui,sans-serif}main{width:min(92%,400px);background:#1c1c28;padding:26px;border-radius:16px}h1{margin-top:0}label{display:grid;gap:8px;color:#bbb}input,button{width:100%;font:inherit;color:#fff;background:#29293a;border:1px solid #45455d;border-radius:9px;padding:13px}button{margin-top:16px;border:0;background:#745bea;font-weight:700}.error{color:#ff8d9d}
</style></head><body><main><h1>Hikâye yayınla</h1><p>Devam etmek için şifreni gir.</p>{{if .}}<p class="error">Şifre yanlış.</p>{{end}}<form method="post" action="/stories-login"><label>Şifre<input name="password" type="password" required autofocus autocomplete="current-password"></label><button>Giriş yap</button></form></main></body></html>`))

var adminPage = template.Must(template.New("admin").Parse(`<!doctype html>
<html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Hikâye yayınla</title><style>
*{box-sizing:border-box}body{margin:0;background:#101018;color:#eee;font:16px system-ui,sans-serif}main{max-width:620px;margin:auto;padding:24px 16px}h1{font-size:25px}form,.story{display:grid;gap:12px;background:#1c1c28;padding:18px;border-radius:14px;margin:16px 0}label{display:grid;gap:6px;font-size:14px;color:#bbb}input,textarea,select,button{width:100%;font:inherit;color:#fff;background:#29293a;border:1px solid #45455d;border-radius:9px;padding:12px}textarea{min-height:90px;resize:vertical}button{border:0;background:#745bea;font-weight:700;cursor:pointer}.delete{width:auto;background:#9c3344;padding:8px 12px}.story{display:flex;align-items:center;justify-content:space-between}.story form{display:block;background:none;padding:0;margin:0}.story small{color:#aaa}.hint{color:#aaa;font-size:13px}
</style></head><body><main><form method="post" action="/stories-logout" style="float:right;background:none;padding:0;margin:0"><button class="delete">Çıkış</button></form><h1>Yeni hikâye</h1>
<form method="post" action="/stories-api" enctype="multipart/form-data">
<label>Tür<select name="type"><option value="text">Metin</option><option value="image">Fotoğraf</option><option value="link">Bağlantı</option></select></label>
<label>Ana metin<textarea name="text" required maxlength="1000"></textarea></label>
<label>Kısa kart başlığı<input name="title" maxlength="120"></label>
<label>Alt metin<textarea name="subtext" maxlength="1000"></textarea></label>
<label>Fotoğraf (en fazla 10 MB)<input name="image" type="file" accept="image/jpeg,image/png,image/webp,image/gif"></label>
<label>Bağlantı<input name="link" placeholder="/posts/… veya https://…"></label>
<label>Bağlantı düğmesi<input name="link_label" maxlength="80" value="Aç"></label>
<label>Arka plan CSS'i<input name="bg" value="linear-gradient(160deg, #16171f, #241f3d)"></label>
<button>Şimdi yayınla</button><div class="hint">Yayın saati otomatik eklenir ve hikâye 24 saat yeni görünür.</div>
</form><h1>Yayınlananlar</h1>
{{range .}}<div class="story"><div><strong>{{.Title}}</strong><br><small>{{.Text}} · {{.Posted}}</small></div><form method="post" action="/stories-api/delete"><input type="hidden" name="id" value="{{.ID}}"><button class="delete">Sil</button></form></div>{{else}}<p class="hint">Henüz hikâye yok.</p>{{end}}
</main></body></html>`))
