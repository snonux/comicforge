# Configuration

[Back to README](../README.md)

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

Start from [`config.yaml.example`](../config.yaml.example). The important settings are:

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

language:
  story_language: English
  script: Latin

story:
  # Probability of picking the photorealistic style family when --ultra-realistic
  # / --no-ultra-realistic are unset (0.0–1.0; values outside that range are clamped).
  realistic_weight: 0.4

styles:
  comic:
    - classic comic book with bold ink outlines
    - graphic novel with dramatic shadows
  realistic:
    - ultra-realistic DSLR photography, cinematic 35mm lens
    - cinematic realism with natural light
  # Also supported (see config.yaml.example): cartoon, action_90s, manga, horror,
  # watercolor, pulp, mecha, pixel_art, ink_wash, clay.

narration:
  voices:
    - Charon
    - Fenrir
  chunk_words: 100

prompts_dir: ./prompts
```

`provider.*` currently supports Gemini in this codebase. Set `api.google_api_key`, `COMICFORGE_API_GOOGLE_API_KEY`, or fallback `GOOGLE_API_KEY` for Gemini-backed generation.

## Style selection

When neither `--ultra-realistic` nor `--no-ultra-realistic` is set:

1. With probability `story.realistic_weight`, ComicForge picks the **realistic** (photorealistic) style family.
2. Otherwise it picks uniformly among the **non-realistic** configured families (comic, cartoon, manga, …).
3. Within the chosen family, it picks one style line at random (with a mild comic-family bias when that family is selected).

`--ultra-realistic` forces the realistic family; `--no-ultra-realistic` forces classic comic. `--style` overrides the concrete style line for story and prompt mode.

See [Page shape and PDF export](PDF.md) for aspect-ratio precedence and PDF settings, and [Usage](USAGE.md#vocabulary-and-language) for vocabulary and story-language alignment.
