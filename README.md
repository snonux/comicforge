# ComicForge

![ComicForge](assets/comicforge.png)

ComicForge turns a vocabulary file into a generated comic package. It uses Gemini-backed providers to write a story, draw comic pages, and optionally produce narration. The CLI writes comic assets into `./comics/assets/<slug>/`, gallery copies into `./comics/gallery/`, and final PDFs into `./comics/PDF/`.

## What It Does

ComicForge reads a vocabulary list, generates a story from those words, renders comic pages, and saves supporting files alongside the comic. By default it uses Gemini for text, image, and text-to-speech generation. When no explicit style is provided, it randomly chooses between photorealistic, classic comic, rubber-hose cartoon, 90s action, manga, horror, and watercolor storybook style families.

It also supports a manual prompt mode for generating a single image directly from a prompt, without the vocabulary/story pipeline.

Generated output includes:

- story text in `comics/assets/<slug>/`
- vocabulary recap in `comics/assets/<slug>/`
- theme file in `comics/assets/<slug>/`
- comic page PNGs in `comics/assets/<slug>/`
- gallery PNG copies in `comics/gallery/`
- PDF output in `comics/PDF/` when page rendering succeeds
- narration MP3 in `comics/assets/<slug>/` when narration is enabled and a TTS provider is available

## Installation

Build and test with Mage:

```bash
mage build
mage test
mage install
```

Or build directly:

```bash
go build -o comicforge ./cmd/comicforge
```

The `mage install` target copies the binary into `$GOPATH/bin` and falls back to `~/go/bin` when `GOPATH` is not set.

## Configuration

ComicForge loads configuration from:

1. `--config <file>` when provided
2. `~/.config/comicforge/config.yaml`
3. `~/config.yaml`
4. `./config.yaml`

Environment variables also work. `COMICFORGE_` is the prefix, and dots in config keys become underscores. Examples:

- `COMICFORGE_API_GOOGLE_API_KEY`
- `COMICFORGE_PROVIDER_TEXT`
- `COMICFORGE_MODELS_IMAGE`
- `COMICFORGE_COMIC_STORY_PAGES`
- `COMICFORGE_COMIC_GALLERY_PAGES`
- `COMICFORGE_COMIC_ASPECT_RATIO`
- `COMICFORGE_LANGUAGE_STORY_LANGUAGE`
- `COMICFORGE_LANGUAGE_SCRIPT`
- `COMICFORGE_PDF_DENSITY`
- `COMICFORGE_PDF_JPEG_QUALITY`
- `COMICFORGE_PDF_PRESENTATION`

If `COMICFORGE_API_GOOGLE_API_KEY` is not set, ComicForge falls back to `GOOGLE_API_KEY`.

Start from [`config.yaml.example`](config.yaml.example). The important settings are:

```yaml
provider:
  text: gemini
  image: gemini
  tts: gemini

api:
  google_api_key: ""

models:
  text: gemini-2.5-flash
  image: gemini-3.1-flash-image-preview
  image_text: gemini-2.5-flash
  tts: gemini-2.5-flash-preview-tts

comic:
  story_pages: 5
  gallery_pages: 5
  # Layout prompts: 4 panels require at least three horizontal rows (no 2×2-only grid).
  panels_per_page: 4
  # Gemini image aspect ratio (e.g. 16:9 widescreen, 2:3 portrait comic page).
  aspect_ratio: "16:9"
  prompt_max_chars: 900
  page_max_retries: 5
  page_retry_base_seconds: 15

# Final PDF assembly via ImageMagick `convert` (requires ImageMagick on PATH).
pdf:
  density: 150
  # 0 = default PDF encoding; 1–100 = JPEG compression (smaller files).
  jpeg_quality: 0
  # none = full-bleed; print/book = ISO A4 portrait PDF pages (print = matte; book = aged + tilt + shadow + thick edge).
  presentation: none

story:
  realistic_weight: 0.4

styles:
  comic:
    - classic comic book with bold ink outlines
    - graphic novel with dramatic shadows
  realistic:
    - ultra-realistic DSLR photography, cinematic 35mm lens
    - cinematic realism with natural light

narration:
  chunk_words: 100

prompts_dir: ./prompts
```

`provider.*` currently supports Gemini in this codebase. Set `api.google_api_key`, `COMICFORGE_API_GOOGLE_API_KEY`, or fallback `GOOGLE_API_KEY` for Gemini-backed generation.

### Page shape and PDF export

**Image aspect ratio** controls both the Gemini generation API and the wording in prompt templates (widescreen vs tall comic page). It comes from configuration unless overridden on the command line.

| Source | Effect |
|--------|--------|
| `comic.aspect_ratio` in YAML / env | Default ratio for the run (default `16:9`). Overridden when `pdf.presentation` is `print` or `book` (see next row), unless you set `--aspect-ratio`. |
| `pdf.presentation` `print` or `book` (YAML / `COMICFORGE_PDF_PRESENTATION` / `--pdf-presentation`) | Unless `--aspect-ratio` is set on the CLI, forces `3:4` (closest Gemini ratio to **ISO A4** portrait, matching the PDF page size). This wins over `--page-format`. |
| `--page-format screen` | Sets aspect ratio to `16:9` only when PDF mode is `none` and `--aspect-ratio` is unset. No effect when `pdf.presentation` is `print` or `book` (those modes force `3:4` unless you pass `--aspect-ratio`). |
| `--page-format comic` | Sets aspect ratio to `2:3` under the same conditions as `screen` (not applied for `print` / `book` unless you override with `--aspect-ratio`). |
| `--aspect-ratio <W:H>` | Explicit ratio (e.g. `16:9`, `2:3`, `3:4`, `21:9`). **Highest precedence** over `--page-format` and `pdf.presentation`. |

**PDF assembly** (`pdf` in YAML or the flags below) only affects the multi-page comic PDF produced after all PNGs are rendered. It does not apply to manual `--prompt` mode (single PNG, no PDF). ImageMagick’s `convert` must be available to build the PDF.

| Key / flag | Meaning |
|------------|---------|
| `pdf.density` / `COMICFORGE_PDF_DENSITY` | Passed to ImageMagick `-density` (default `150`). |
| `pdf.jpeg_quality` / `COMICFORGE_PDF_JPEG_QUALITY` | `0` keeps the default encoding for the assembled PDF. Values `1`–`100` enable JPEG compression inside the PDF (smaller files, similar to a “compressed” comic PDF). |
| `pdf.presentation` / `COMICFORGE_PDF_PRESENTATION` | `none` (default): full-bleed PDF pages (size follows source rasters). `print`: cream matte around the art, then **each page is fitted to ISO A4 portrait** (210×297 mm at `pdf.density`); cover, story, gallery, and back pages each become one A4 PDF page. Prompts add extra “single full sheet” instructions for cover / gallery / back. `book`: same **A4 page size** and the same per-page layout, after art is **aged** (mild sepia / desaturation, grain, vignette), then **tilt**, **drop shadow**, and a **thick page-edge** strip. |
| `--pdf-jpeg-quality` | Same as `pdf.jpeg_quality` when set on the CLI. |
| `--pdf-presentation` | `none`, `print`, or `book`; same as `pdf.presentation` when set on the CLI. |

## Usage

ComicForge requires a vocabulary file:

```bash
comicforge --vocab words.txt --config config.yaml --output out
```

Vocabulary lines can be:

- `ябълка = apple`
- `котка == домашно животно`
- `стол`
- `= translation only`

**Story language vs vocabulary script:** The story and visible text in panels must match `language.story_language` and `language.script` in config (e.g. Bulgarian + Cyrillic). If your word list is **English (Latin letters)** but the story is generated in **Cyrillic**, validation will fail when those Latin words appear in the story. For English stories with English/Latin vocabulary, set for example `story_language: English` and `script: Latin` in YAML, or for one run:

```bash
COMICFORGE_LANGUAGE_STORY_LANGUAGE=English COMICFORGE_LANGUAGE_SCRIPT=Latin \
  comicforge --vocab english-words.txt --config config.yaml --output out
```

Useful flags:

- `--output` sets the root output directory
- `--prompt` generates a single image from a direct prompt and skips the story flow
- `--prompts-dir` overrides the prompt template directory
- `--style` and `--theme` override story generation hints and are also applied as context in manual prompt mode
- `--page-format` `screen` or `comic` presets aspect ratio (`16:9` vs `2:3`) when PDF mode is `none`; see [Page shape and PDF export](#page-shape-and-pdf-export)
- `--aspect-ratio` explicit `W:H` for Gemini (highest precedence over `--page-format`, `pdf.presentation`, and `comic.aspect_ratio`)
- `--pdf-jpeg-quality` JPEG quality `0`–`100` for the assembled PDF (`0` = default encoding)
- `--pdf-presentation` `none` (default), `print` (matte + **ISO A4** PDF pages), or `book` (aged art + tilt + shadow + thick edge + **same A4** pages); also sets generation to `3:4` unless `--aspect-ratio` is set
- `--slug` forces the output folder name
- `--narrate` enables narration output
- `--narrator-voice` picks the Gemini narration voice
- `--text-provider`, `--image-provider`, `--tts-provider` override provider names
- `--text-model`, `--image-model`, `--image-text-model`, `--tts-model` override model IDs
- `--ultra-realistic` and `--no-ultra-realistic` force photorealistic or classic comic rendering; without either flag ComicForge randomly chooses from the configured style families
- `--version` prints the application version

Example (default PDF: full-bleed, aspect ratio from config):

```bash
comicforge \
  --vocab vocab.txt \
  --config config.yaml \
  --output out \
  --slug demo-comic \
  --narrate
```

The generated files are written under `out/comics/assets/demo-comic/`, with the gallery copied to `out/comics/gallery/` and the PDF written to `out/comics/PDF/`.

**ISO A4 print PDF** (`print` fits every page to 210×297 mm at `pdf.density`; Gemini uses `3:4` unless you pass `--aspect-ratio`):

```bash
comicforge \
  --vocab vocab.txt \
  --config config.yaml \
  --output out \
  --slug my-a4-comic \
  --pdf-presentation print
```

Shorter test run (fewer story and gallery pages) via env:

```bash
COMICFORGE_COMIC_STORY_PAGES=2 COMICFORGE_COMIC_GALLERY_PAGES=2 \
  comicforge --vocab vocab.txt --config config.yaml --output out \
  --slug a4-smoke --pdf-presentation print
```

**Book-style A4 PDF** (same page dimensions as `print`, plus aged paper, tilt, shadow, and page-edge treatment):

```bash
comicforge \
  --vocab vocab.txt \
  --config config.yaml \
  --output out \
  --slug my-a4-book \
  --pdf-presentation book
```

For manual prompt mode:

```bash
comicforge --prompt "a robot reading a newspaper" --output out --slug manual-robot
```

This writes a single image to `out/comics/assets/manual-robot/prompt.png`. Without `--slug`, ComicForge asks the text model for a short title and uses that as the directory slug, with a short deterministic fallback if title generation fails.

Manual prompt mode can be combined with `--style`, `--theme`, `--page-format`, `--aspect-ratio`, and the ultra-realistic flags to shape the generated image prompt (page-shape flags affect the image API and the manual prompt template; PDF flags have no effect because no PDF is built).

### Style and theme reference sheet (`stylegrid`)

To compare **every** line in `styles.*` plus **every** `story.genres` entry on one montage (shared test scene, header above each cell), use the helper command. It calls the configured image provider once per cell and requires ImageMagick **`montage`** as well as **`convert`**.

```bash
go run ./cmd/stylegrid --config config.yaml --output stylegrid.png
```

Flags: `--scene` (override the shared scene), `--aspect-ratio` (default `16:9`), `--timeout` (default 45m). With default `config` this produces **14** style variants + **3** genre panels = **17** images. Regenerate after editing style or genre lists in YAML.

`mage stylegrid` writes `stylegrid.png` in the repo root via the same command.
