# Installation and development

[Back to README](../README.md)

Run these commands from the repository root, where the default prompt templates live.

## Prerequisites

| Dependency | Needed for |
|------------|------------|
| Go (see [`go.mod`](../go.mod)) | Build and tests |
| Mage | Project build and development tasks |
| Gemini API key (`api.google_api_key`, `COMICFORGE_API_GOOGLE_API_KEY`, or `GOOGLE_API_KEY`) | Text, image, and TTS generation |
| [ImageMagick](https://imagemagick.org/) `convert` | Assembling `comics/PDF/*.pdf` |
| ImageMagick `montage` | `mage stylegrid` / `cmd/stylegrid` montages |
| `ffmpeg` | Narration MP3 encoding and ambient/audio mixing |
| [golangci-lint](https://golangci-lint.run/) | `mage lint` only |

Without ImageMagick, page PNGs and story files are still written; PDF assembly is skipped with a warning. Without `ffmpeg`, narration fails when `--narrate` is enabled.

## Build and install

Build and test with Mage:

```bash
mage build
mage test
mage lint
mage install
mage stylegrid   # optional; needs Gemini + ImageMagick montage
mage clean
```

`mage lint` runs [golangci-lint](https://golangci-lint.run/) using [`.golangci.yml`](../.golangci.yml) (install the tool separately if it is not already on your `PATH`).

Or build directly:

```bash
go build -o comicforge ./cmd/comicforge
```

The `mage install` target copies the binary into `$GOPATH/bin` and falls back to `~/go/bin` when `GOPATH` is not set.
