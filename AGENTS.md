# ComicForge

## Repository Notes

- Go module: `github.com/snonux/comicforge`
- Binary entrypoint: `cmd/comicforge/main.go`
- Build, test, install, and clean tasks live in `Magefile.go`
- Version is defined in `internal/version.go`

## Conventions

- Keep `cmd/comicforge/main.go` small: flag parsing and command wiring only.
- Put application logic under `internal/`.
- Use `mage` for project tasks (`build`, `test` with `-race`, `lint`, `install`, `clean`, `stylegrid`).
- `cmd/stylegrid` + `internal/stylegrid`: montage of all configured `styles.*` lines and `story.genres` (`mage stylegrid` / `go run ./cmd/stylegrid`).
- Style families live in a map (`styles` YAML / `StyleConfig`); `story.realistic_weight` biases photorealistic vs other families when ultra flags are unset.

## Generating comics

- Before generating a comic or image, read [the generation workflow](docs/GENERATING.md). It covers setup, exact CLI commands, language/script settings, page counts, output files, verification, and reruns.
- Build with `mage build` and run `./comicforge` from the repository root. Use an explicit `--config`, the intended vocabulary, and a fresh `--slug` or output root for each new comic.
- Full comics use `--vocab`; single images use `--prompt`. Generation calls Gemini APIs. Narration requires `--narrate`; verify the actual PDF/MP3 files because optional output failures may only produce warnings.
- Detailed references: [configuration](docs/CONFIGURATION.md), [usage](docs/USAGE.md), [PDF export](docs/PDF.md), and [style previews](docs/STYLEGRID.md).

## Page shape and PDF

- Documented in `docs/PDF.md`, `docs/USAGE.md`, and `docs/CONFIGURATION.md`: `--page-format`, `--aspect-ratio`, `comic.aspect_ratio`, `pdf.*` / `--pdf-*` (`print` / `book`: ISO A4 portrait PDF pages; `book` adds aged art, tilt, shadow, thick edge), language env vars for vocab/script alignment, and matching `COMICFORGE_` env vars.
- System deps (ImageMagick, ffmpeg, API key) are listed under **Prerequisites** in `docs/INSTALLATION.md`.
- Architecture overview: `docs/ARCHITECTURE.md`.

