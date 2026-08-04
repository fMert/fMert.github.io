package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationAndImage(t *testing.T) {
	if validate(Story{Type: "text", Text: "hello"}) != nil {
		t.Fatal("valid text story rejected")
	}
	if validate(Story{Type: "text", Text: "hello", Link: "javascript:alert(1)"}) == nil {
		t.Fatal("unsafe link accepted")
	}
	if validate(Story{Type: "text", Text: "hello", Link: "//evil.example"}) == nil {
		t.Fatal("protocol-relative link accepted")
	}
	dir := t.TempDir()
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 504)...)
	ext, err := saveImage(dir, "test", bytes.NewReader(png), int64(len(png)))
	if err != nil || ext != ".png" {
		t.Fatalf("PNG rejected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "test.png")); err != nil {
		t.Fatal("PNG was not saved")
	}
}

func TestSession(t *testing.T) {
	a := &app{password: "secret"}
	r := httptest.NewRequest("GET", "/stories-admin", nil)
	if a.authorized(r) {
		t.Fatal("request without a session was authorized")
	}
	r.AddCookie(&http.Cookie{Name: "story_session", Value: a.sessionToken()})
	if !a.authorized(r) {
		t.Fatal("valid session was rejected")
	}
}

func TestAdminPageIncludesComposerAndPreview(t *testing.T) {
	a := &app{dataDir: t.TempDir(), password: "secret"}
	r := httptest.NewRequest("GET", "/stories-admin", nil)
	r.AddCookie(&http.Cookie{Name: "story_session", Value: a.sessionToken()})
	w := httptest.NewRecorder()

	a.admin(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("admin page returned %d", w.Code)
	}
	body := w.Body.String()
	for _, expected := range []string{`id="story-form"`, `id="preview"`, `Canlı önizleme`, `Yayınlanan hikâyeler`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("admin page is missing %q", expected)
		}
	}
}
