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

