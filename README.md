# ComicForge

![ComicForge logo](assets/comicforge-logo.png)

ComicForge turns a vocabulary file into a generated comic package. It uses Gemini-backed providers to write a story, draw comic pages, and optionally produce narration. The CLI writes comic assets into `./comics/assets/<slug>/`, gallery copies into `./comics/gallery/`, and final PDFs into `./comics/PDF/`.

## What It Does

ComicForge reads a vocabulary list, generates a story from those words, renders comic pages, and saves supporting files alongside the comic. By default it uses Gemini for text, image, and text-to-speech generation.

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
  panels_per_page: 4
  aspect_ratio: "16:9"
  prompt_max_chars: 900
  page_max_retries: 5
  page_retry_base_seconds: 15

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

`provider.*` currently supports Gemini in this codebase. Set `api.google_api_key` or `COMICFORGE_API_GOOGLE_API_KEY` for Gemini-backed generation.

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

Useful flags:

- `--output` sets the root output directory
- `--prompt` generates a single image from a direct prompt and skips the story flow
- `--prompts-dir` overrides the prompt template directory
- `--style` and `--theme` override story generation hints
- `--slug` forces the output folder name
- `--narrate` enables narration output
- `--narrator-voice` picks the Gemini narration voice
- `--text-provider`, `--image-provider`, `--tts-provider` override provider names
- `--text-model`, `--image-model`, `--image-text-model`, `--tts-model` override model IDs
- `--ultra-realistic` and `--no-ultra-realistic` control the rendering mode
- `--version` prints the application version

Example:

```bash
comicforge \
  --vocab vocab.txt \
  --config config.yaml \
  --output out \
  --slug demo-comic \
  --narrate
```

The generated files are written under `out/comics/assets/demo-comic/`, with the gallery copied to `out/comics/gallery/` and the PDF written to `out/comics/PDF/`.

For manual prompt mode:

```bash
comicforge --prompt "a robot reading a newspaper" --output out --slug manual-robot
```

This writes a single image to `out/comics/assets/manual-robot/prompt.png`.
