# ComicForge

## Repository Notes

- Go module: `codeberg.org/snonux/comicforge`
- Binary entrypoint: `cmd/comicforge/main.go`
- Build, test, install, and clean tasks live in `Magefile.go`
- Version is defined in `internal/version.go`

## Conventions

- Keep `cmd/comicforge/main.go` small: flag parsing and command wiring only.
- Put application logic under `internal/`.
- Use `mage` for project tasks.
- `cmd/stylegrid` + `internal/stylegrid`: montage of all configured `styles.*` lines and `story.genres` (`mage stylegrid` / `go run ./cmd/stylegrid`).

## Page shape and PDF

- Documented in `README.md` (**Page shape and PDF export** and **Usage**): `--page-format`, `--aspect-ratio`, `comic.aspect_ratio`, `pdf.*` / `--pdf-*` (`print` / `book`: ISO A4 portrait PDF pages; `book` adds aged art, tilt, shadow, thick edge), language env vars for vocab/script alignment, and matching `COMICFORGE_` env vars.

