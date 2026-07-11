# Handoff: Blog Stories (Instagram-style) — fmert.me (Jekyll Chirpy)

## Overview
Add an Instagram-style "Hikayeler" (Stories) feature to the homepage of fmert.me — a Jekyll blog using the **Chirpy theme** (chirpy-starter) with a custom violet reskin (`assets/css/jekyll-theme-chirpy.scss`). A horizontal row of tall vertical story previews sits at the top of the home page above the post list. Clicking one opens a fullscreen viewer with progress bars and auto-advance.

Repo: https://github.com/fMert/fMert.github.io

## About the Design Files
`Blog Stories.dc.html` is a **design reference created in HTML** — a prototype showing intended look and behavior, NOT production code to copy directly. The task is to **recreate this design inside the Jekyll/Chirpy codebase** using its established patterns: a Jekyll include + a data file + a small custom SCSS block + vanilla JS. Do not introduce React or any build tooling.

## Fidelity
**High-fidelity.** Colors, spacing, typography and interactions are final. Match them exactly. All colors below map to the existing CSS custom properties already defined in `assets/css/jekyll-theme-chirpy.scss` — prefer `var(--accent)`, `var(--accent-gradient)` etc. over hardcoded hex so light mode works automatically.

## Recommended Jekyll Architecture
Stories are authored as a data file, rendered by Liquid, expired client-side by JS (GitHub Pages is static — server-side expiry is impossible; JS compares timestamps at page load).

1. **`_data/stories.yml`** — one entry per story:
   ```yaml
   - id: yeni-hafta
     type: image            # image | text | link
     image: /assets/img/avatar.jpg
     text: "Yeni bir hafta, yeni denemeler 🌱"
     subtext: "Queyntisen için büyük bir güncelleme yolda."
     posted: 2026-07-11 18:00:00 +0300
   - id: queyntisen-yazi
     type: link
     bg: "linear-gradient(160deg, #16171f 0%, #241f3d 60%, #1a1b25 100%)"
     text: "Yeni yazı yayında ✍️"
     subtext: "Queyntisen Artık Bir Obsidian Alternatifi…"
     link: /posts/queyntisen-obsidian-alternatifi/
     link_label: "Yazıyı oku"
     posted: 2026-07-10 21:00:00 +0300
   ```
2. **`_includes/stories.html`** — renders the row + the (hidden) fullscreen viewer markup from `site.data.stories`. Embed each story's `posted` time as `data-posted="{{ story.posted | date_to_xmlschema }}"`.
3. **Override `_layouts/home.html`** — copy the theme's home layout from the chirpy gem (cotes2020/jekyll-theme-chirpy `_layouts/home.html`) into the repo's `_layouts/`, and insert `{% include stories.html %}` immediately before `<div id="post-list" …>`.
4. **`assets/js/stories.js`** — vanilla JS (no dependencies) implementing the expiry rule + viewer. Load it at the end of `stories.html` with `<script src="{{ '/assets/js/stories.js' | relative_url }}" defer></script>`.
5. **SCSS** — append a `/* ----- Stories ----- */` section to the existing custom block in `assets/css/jekyll-theme-chirpy.scss`, reusing the token variables.

## The 24-hour Rule (critical business logic)
At page load, in JS:
- A story is **fresh** if `now - posted < 24h`. Show all fresh stories.
- If NO story is fresh, show ONLY the most recently posted story, in **expired** style (grey ring, dimmed, label "Son hikaye"). It persists until a newer story is published.
- Expired stories other than the newest are hidden entirely.
- If `_data/stories.yml` is empty, hide the whole section (also hide it via Liquid `{% if site.data.stories.size > 0 %}`).

Seen-state: persist viewed story ids in `localStorage` key `fmert-stories-seen` (JSON object `{ "<id>": true }`). Seen fresh stories get the grey "seen" ring.

## Screens / Components

### 1. Stories row (top of home page, above #post-list)
- Section margin-bottom: 28px. Header row: bolt icon `fa-solid fa-bolt` in `var(--accent)` 13px + title "Hikayeler" 15px/700 `var(--heading-color)` + count label 12.5px `var(--text-muted-color)`: "${n} yeni · 24 saat görünür". 14px gap below header.
- Horizontal scroller: `display:flex; gap:14px; overflow-x:auto; padding:4px 2px 10px`.
- **Story card** (button, no border/bg): width 118px, column layout, 8px gap.
  - **Ring wrapper**: 118×196px, border-radius 16px, padding 3px.
    - Unseen fresh ring: `linear-gradient(140deg,#9b7bff 0%,#5aa2ff 55%,#9f8bff 100%)` (= accent gradient family)
    - Seen ring: `#2f3142` (dark) — use a muted token
    - Expired ring: `#3a3b48` / `var(--mask-bg)`
    - Hover: `transform: translateY(-4px)`, transition 0.2s ease.
  - **Inner preview**: fills wrapper, border-radius 13px, `border: 3px solid var(--main-bg)` (gap between ring and content), overflow hidden. Background: story image (object-fit cover) or the story's `bg` gradient, fallback `var(--card-bg)`.
    - Bottom scrim: `linear-gradient(180deg, transparent 40%, rgb(0 0 0/72%) 100%)`.
    - Type badge top-left (8px inset): 10px/700 white, padding 3px 8px, pill, bg `rgb(107 75 240/75%)` (expired: `rgb(58 59 72/80%)`), backdrop-blur 4px. Labels: FOTO / METİN / YAZI / ARŞİV.
    - Title bottom (9px inset): 12px/700 white, line-height 1.25, text-shadow `0 1px 4px rgb(0 0 0/60%)`.
    - Expired: image/content gets `filter: grayscale(0.7) brightness(0.8)` plus an overlay `rgb(8 9 14/35%)` over the whole ring wrapper.
  - **Time label** under card: 11.5px/600 `var(--text-muted-color)` — "Az önce" / "N sa önce" / expired: "Son hikaye" in dimmer color.

### 2. Fullscreen viewer (opened on click)
- Overlay: `position:fixed; inset:0; z-index` above topbar; bg `rgb(4 4 8/92%)`, `backdrop-filter: blur(18px)`, fade-in 0.22s. Click on backdrop closes. Escape closes. Lock body scroll while open.
- **Story canvas**: centered, `width:min(430px,96vw); height:min(764px,94vh)` (9:16), border-radius 18px, overflow hidden, shadow `0 40px 90px -20px rgb(0 0 0/80%), 0 0 0 1px var(--accent-soft-2)`. Background = image (cover, slow 5s scale 1→1.06 "Ken Burns") or story gradient.
  - Image stories get scrims top+bottom: `linear-gradient(180deg, rgb(0 0 0/45%) 0%, transparent 26%, transparent 58%, rgb(0 0 0/62%) 100%)`.
- **Progress bars**: top 10px, 10px side insets, flex gap 5px; each bar flex:1, height 3px, pill, track `rgb(255 255 255/28%)`, fill white. Past = 100%, current animates 0→100% over 5s (JS interval or CSS transition), future = 0%.
- **Header** (top 24px, 14px side insets): 38px round avatar with 2px `var(--accent)` ring; username "fmert" 14px/700 white; time label 12px `rgb(255 255 255/75%)`; if expired, pill badge "SON HİKAYE" with clock icon (10.5px/700, bg `rgb(255 255 255/16%)`); pause/play button; close ✕ button (19px).
- **Tap zones**: invisible left/right areas (34% width each, avoiding header/footer bands) — left = previous, right = next. Arrow keys also navigate.
- **Text content**: centered column, padding 90px 30px 100px. Main text 26–30px/800 white, line-height 1.35, text-shadow `0 2px 16px rgb(0 0 0/45%)`; image stories align it to the bottom (`justify-content:flex-end`). Subtext 15.5px/400 `rgb(255 255 255/82%)`.
- **Link CTA** (type: link): pill bottom-center 26px up: white bg `rgb(255 255 255/94%)`, text `#1b1b23` 14px/700, padding 12px 22px, arrow-up icon, hover scale 1.05 + accent text. Navigates to `story.link`.

## Interactions & Behavior
- Auto-advance: 5000ms per story; on end of last story, close viewer.
- Pause/play toggles the timer (icon swaps `fa-pause`/`fa-play`).
- Prev on first story just restarts its progress.
- Opening/advancing marks the story seen (localStorage) and updates its ring on close.
- All transitions: ring hover 0.2s ease; overlay fade 0.22s; card hover translateY(-4px).
- Mobile: the row already scrolls horizontally; viewer canvas is 96vw × 94vh.

## Design Tokens (dark mode values — map to existing SCSS vars)
- Accent: `#9f8bff`; gradient `linear-gradient(135deg,#9b7bff 0%,#5aa2ff 100%)` — already `--accent`, `--accent-gradient`
- Page bg `#08090e` (`--main-bg`), card `#16171f` (`--card-bg`), borders `#303243`/`#23242f`
- Text: `#dadbe7` (`--text-color`), headings `#f7f8fd`, muted `#9c9dae`, dim `#6a6b7e`
- Radii: ring 16px, inner 13px, viewer 18px, pills 999px
- Font: theme default (Source Sans family in Chirpy); icons: Font Awesome (already loaded by Chirpy)
- Story duration: 5000ms; expiry window: 24h

## Assets
- `/assets/img/avatar.jpg` — already in the repo; used for the demo photo story and viewer header.
- Icons: Font Awesome classes already available in Chirpy (`fa-bolt`, `fa-pause`, `fa-play`, `fa-xmark`, `fa-angle-left/right`, `far fa-clock`, `fas fa-arrow-up`).

## Files in this bundle
- `Blog Stories.dc.html` — the interactive design reference (open in a browser; the stories row, seen-rings, expired state, and full viewer are all functional).
- `assets/img/avatar.jpg` — copy of the site avatar used by the prototype.

## Suggested Claude Code prompt
> Read design_handoff_blog_stories/README.md and implement the Stories feature in this Jekyll Chirpy site exactly as specified: _data/stories.yml, _includes/stories.html, a home layout override that inserts the include above #post-list, assets/js/stories.js, and a Stories SCSS section appended to assets/css/jekyll-theme-chirpy.scss using the existing CSS custom properties. No new dependencies.
