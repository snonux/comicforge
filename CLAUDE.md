# ComicForge

## Repository Notes

- Go module: `codeberg.org/snonux/comicforge`
- Binary entrypoint: `cmd/comicforge/main.go`
- Build, test, install, and clean tasks live in `Magefile.go`
- Version is defined in `internal/version.go`

## Conventions

- Keep `cmd/comicforge/main.go` small: flag parsing and command wiring only.
- Put application logic under `internal/`.
- Use `mage` for project tasks (`build`, `test` with `-race`, `lint`, `install`, `clean`, `stylegrid`).
- `cmd/stylegrid` + `internal/stylegrid`: montage of all configured `styles.*` lines and `story.genres` (`mage stylegrid` / `go run ./cmd/stylegrid`).
- Style families live in a map (`styles` YAML / `StyleConfig`); `story.realistic_weight` biases photorealistic vs other families when ultra flags are unset.

## Page shape and PDF

- Documented in `README.md` (**Page shape and PDF export** and **Usage**): `--page-format`, `--aspect-ratio`, `comic.aspect_ratio`, `pdf.*` / `--pdf-*` (`print` / `book`: ISO A4 portrait PDF pages; `book` adds aged art, tilt, shadow, thick edge), language env vars for vocab/script alignment, and matching `COMICFORGE_` env vars.
- System deps (ImageMagick, ffmpeg, API key) are listed under **Prerequisites** in `README.md`.
- Architecture overview: `docs/ARCHITECTURE.md`.

