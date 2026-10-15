# Page shape and PDF export

[Back to README](../README.md)

**Image aspect ratio** controls both the Gemini generation API and the wording in prompt templates (widescreen vs tall comic page). It comes from configuration unless overridden on the command line.

| Source | Effect |
|--------|--------|
| `comic.aspect_ratio` in YAML / env | Default ratio for the run (default `16:9`). Overridden when `pdf.presentation` is `print` or `book` (see next row), unless you set `--aspect-ratio`. |
| `pdf.presentation` `print` or `book` (YAML / `COMICFORGE_PDF_PRESENTATION` / `--pdf-presentation`) | Unless `--aspect-ratio` is set on the CLI, forces `3:4` (closest Gemini ratio to **ISO A4** portrait, matching the PDF page size). This wins over `--page-format`. |
| `--page-format screen` | Sets aspect ratio to `16:9` only when PDF mode is `none` and `--aspect-ratio` is unset. No effect when `pdf.presentation` is `print` or `book` (those modes force `3:4` unless you pass `--aspect-ratio`). |
| `--page-format comic` | Sets aspect ratio to `2:3` under the same conditions as `screen` (not applied for `print` / `book` unless you override with `--aspect-ratio`). |
| `--aspect-ratio <W:H>` | Explicit ratio (e.g. `16:9`, `2:3`, `3:4`, `21:9`). **Highest precedence** over `--page-format` and `pdf.presentation`. |

**PDF assembly** (`pdf` in YAML or the flags below) only affects the multi-page comic PDF produced after all PNGs are rendered. It does not apply to manual `--prompt` mode (single PNG, no PDF). ImageMagick’s `convert` must be available to build the PDF; if it is missing, ComicForge warns on stderr and exits successfully without writing a PDF.

| Key / flag | Meaning |
|------------|---------|
| `pdf.density` / `COMICFORGE_PDF_DENSITY` | Passed to ImageMagick `-density` (default `150`). |
| `pdf.jpeg_quality` / `COMICFORGE_PDF_JPEG_QUALITY` | `0` keeps the default encoding for the assembled PDF. Values `1`–`100` enable JPEG compression inside the PDF (smaller files, similar to a “compressed” comic PDF). |
| `pdf.presentation` / `COMICFORGE_PDF_PRESENTATION` | `none` (default): full-bleed PDF pages (size follows source rasters). `print`: cream matte around the art, then **each page is fitted to ISO A4 portrait** (210×297 mm at `pdf.density`); cover, story, gallery, and back pages each become one A4 PDF page. Prompts add extra “single full sheet” instructions for cover / gallery / back. `book`: same **A4 page size** and the same per-page layout, after art is **aged** (mild sepia / desaturation, grain, vignette), then **tilt**, **drop shadow**, and a **thick page-edge** strip. |
| `--pdf-jpeg-quality` | Same as `pdf.jpeg_quality` when set on the CLI. |
| `--pdf-presentation` | `none`, `print`, or `book`; same as `pdf.presentation` when set on the CLI. |

## Examples

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
