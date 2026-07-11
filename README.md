# fmert.me

Personal blog of **Furkan Mert Bağcı** — posts about my projects (Queyntisen, Aria), software, and the occasional field research. Built with [Jekyll](https://jekyllrb.com/) and the [Chirpy](https://github.com/cotes2020/jekyll-theme-chirpy) theme, deployed to GitHub Pages.

**Live site:** <https://fmert.me>

## What's customized

- **Violet reskin** — the entire theme is recolored through CSS custom properties in `assets/css/jekyll-theme-chirpy.scss`. Both light and dark modes work; the theme's own toggle is untouched.
- **Stories** — an Instagram-style "Hikayeler" row on the homepage with a fullscreen viewer (progress bars, auto-advance, tap zones, keyboard navigation). Vanilla JS, no dependencies.

## Stories

Stories live in [`_data/stories.yml`](_data/stories.yml). To publish one, add an entry:

```yaml
- id: my-story            # unique, used for seen-state
  type: image             # image | text | link
  image: /assets/img/example.jpg   # type: image
  bg: "linear-gradient(...)"       # type: text | link
  title: "Short card caption"      # optional, falls back to text
  text: "Main story text"
  subtext: "Smaller line below"    # optional
  link: /posts/some-post/          # type: link
  link_label: "Yazıyı oku"         # type: link
  posted: 2026-07-11 18:00:00 +0300
```

Rules (enforced client-side in `assets/js/stories.js`, since GitHub Pages is static):

- A story is visible for **24 hours** after `posted`.
- When nothing is fresh, only the newest story stays, shown dimmed as **"Son hikaye"** — until a newer one is published.
- Viewed stories get a grey ring, persisted in `localStorage`.
- An empty `stories.yml` hides the section entirely.

Implementation: `_includes/stories.html` (markup), `assets/js/stories.js` (logic), the `/* ----- Stories ----- */` section of `assets/css/jekyll-theme-chirpy.scss` (styles), and `_layouts/home.html` (Chirpy home layout override that inserts the row above the post list).

> Note: story images are CSS backgrounds rather than `<img>` tags, because Chirpy's `refactor-content.html` rewrites raw `<img>` markup in layout content.

## Running locally

```bash
bundle install
bash tools/run.sh          # serves at http://127.0.0.1:4000 with live reload
```

Or a one-off production build:

```bash
JEKYLL_ENV=production bundle exec jekyll build   # output in _site/
```

Requires Ruby ≥ 3.1.

## Deployment

Pushing to `main` triggers `.github/workflows/pages-deploy.yml`, which builds the site and publishes it to GitHub Pages.

## License

Content © Furkan Mert Bağcı. Theme code under the [MIT License](https://github.com/cotes2020/chirpy-starter/blob/master/LICENSE) (Chirpy starter).
