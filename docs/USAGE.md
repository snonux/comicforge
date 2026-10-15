# Usage

[Back to README](../README.md)

ComicForge requires a vocabulary file:

```bash
comicforge --vocab words.txt --config config.yaml --output out
```

## Vocabulary and language

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

## Command-line flags

- `--output` sets the root output directory
- `--prompt` generates a single image from a direct prompt and skips the story flow
- `--prompts-dir` overrides the prompt template directory
- `--style` and `--theme` override story generation hints and are also applied as context in manual prompt mode
- `--page-format` `screen` or `comic` presets aspect ratio (`16:9` vs `2:3`) when PDF mode is `none`; see [Page shape and PDF export](PDF.md)
- `--aspect-ratio` explicit `W:H` for Gemini (highest precedence over `--page-format`, `pdf.presentation`, and `comic.aspect_ratio`)
- `--pdf-jpeg-quality` JPEG quality `0`–`100` for the assembled PDF (`0` = default encoding)
- `--pdf-presentation` `none` (default), `print` (matte + **ISO A4** PDF pages), or `book` (aged art + tilt + shadow + thick edge + **same A4** pages); also sets generation to `3:4` unless `--aspect-ratio` is set
- `--slug` forces the output folder name
- `--narrate` enables narration output
- `--narrator-voice` picks the Gemini narration voice
- `--text-provider`, `--image-provider`, `--tts-provider` override provider names
- `--text-model`, `--image-model`, `--image-text-model`, `--tts-model` override model IDs
- `--ultra-realistic` and `--no-ultra-realistic` force photorealistic or classic comic rendering; without either flag ComicForge uses `story.realistic_weight` (see [Style selection](CONFIGURATION.md#style-selection))
- `--version` prints the application version

## Output and reruns

Generated folder names retain Unicode letters, including Cyrillic titles. A
generation identity in each comic assets folder protects existing packages:
reruns reuse page images only when the complete generated story bundle,
vocabulary, and artwork settings match. A new story with the same title or
`--slug` stops before replacing existing artifacts; choose a different `--slug`
or output directory. Existing assets, PDFs, or gallery files without an identity
also require a different destination. Story generation still runs on each
invocation, so a new model response may prevent a resume. Use an explicit `--style` for a stable
style when retrying an interrupted generation.
Template overrides and provider/model changes are outside this identity check;
choose a new destination when changing those inputs.

## Full comic example

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

For A4 print and book examples, see [Page shape and PDF export](PDF.md#examples).

## Manual prompt mode

```bash
comicforge --prompt "a robot reading a newspaper" --output out --slug manual-robot
```

This writes a single image to `out/comics/assets/manual-robot/prompt.png`. Without `--slug`, ComicForge asks the text model for a short title and uses that as the directory slug, with a short deterministic fallback if title generation fails.

Manual prompt mode can be combined with `--style`, `--theme`, `--page-format`, `--aspect-ratio`, and the ultra-realistic flags to shape the generated image prompt (page-shape flags affect the image API and the manual prompt template; PDF flags have no effect because no PDF is built).
