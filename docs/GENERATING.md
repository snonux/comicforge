# Generate a comic

[Back to README](../README.md)

Use this workflow when a person or agent wants to generate artwork from this checkout. Run commands from the repository root. Generation calls the configured Gemini APIs; a full comic makes text and image requests, and `--narrate` adds TTS requests.

## Prepare the run

1. Build the CLI with `mage build`. Use `./comicforge` from this checkout; `mage install` is optional. Check available flags with `./comicforge --help`.
2. Use an existing config when one is supplied. Otherwise copy [config.yaml.example](../config.yaml.example) to a new run-specific file as in the example below. Pass `--config` explicitly so a config in the user's home directory cannot silently select different settings. Existing `COMICFORGE_*` environment variables still override YAML values; CLI flags override their corresponding settings.
3. Use an available Gemini API key from `api.google_api_key`, `COMICFORGE_API_GOOGLE_API_KEY`, or fallback `GOOGLE_API_KEY`. Keep credentials out of generated docs and commits.
4. Choose the vocabulary, language, script, style, theme, page counts, output root, and a fresh slug. Use the person's requested values; the example below makes a short English comic.
5. For PDFs, check that ImageMagick `convert` is on `PATH`. For narration, also check `ffmpeg`. See [installation](INSTALLATION.md) for all dependencies.

Vocabulary is UTF-8 text, one entry per line. Blank lines are ignored. Supported forms are `word`, `word = translation`, `word == definition`, and `= translation only` (asks the model for the target-language word). There is no comment syntax in vocabulary files.

Set `language.story_language` and `language.script` together, and use vocabulary compatible with the target script. For example, use English + Latin for English words, or Bulgarian + Cyrillic for Bulgarian words. The example config uses English/Latin, while built-in defaults without a config use Bulgarian/Cyrillic. Story language is configured independently of the vocabulary; the CLI does not infer it from the word list.

## Generate a short comic

With the API key already configured in the environment, this example creates a fresh working directory for its input files and output. The shell variable is also used in the verification commands below.

```bash
mage build
comicforge_run_dir=$(mktemp -d "${TMPDIR:-/tmp}/comicforge.XXXXXX")
cp config.yaml.example "$comicforge_run_dir/config.yaml"
printf 'forest\nlantern\nfriendship\n' > "$comicforge_run_dir/words.txt"

COMICFORGE_LANGUAGE_STORY_LANGUAGE=English \
COMICFORGE_LANGUAGE_SCRIPT=Latin \
COMICFORGE_COMIC_STORY_PAGES=2 \
COMICFORGE_COMIC_GALLERY_PAGES=1 \
  ./comicforge \
    --config "$comicforge_run_dir/config.yaml" \
    --vocab "$comicforge_run_dir/words.txt" \
    --output "$comicforge_run_dir/output" \
    --slug forest-adventure \
    --style "classic comic book with bold ink outlines" \
    --theme "friends finding their way through an enchanted forest" \
    --page-format comic
```

This requests five images: a cover, two story pages, one gallery page, and a back cover. Without page-count overrides, the example config requests five story pages and five gallery pages, plus the two covers (12 images). Page counts are config/environment settings; there are no `--story-pages` or `--gallery-pages` flags.

For a permanent destination, replace `--output` with the desired root (for example `--output .` writes into this repository's ignored `comics/` directory). Keep the vocabulary and config if the run needs to be reproduced.

To change the result:

- Set `--style` for an explicit artwork prompt and `--theme` for a story theme. Without a style, family selection uses `story.realistic_weight`; without a theme, the story uses a configured genre. See [configuration](CONFIGURATION.md#style-selection).
- Use `--page-format screen` for `16:9` or `--page-format comic` for `2:3`. Use `--aspect-ratio W:H` for an explicit ratio.
- Use `--pdf-presentation print` for A4 pages with a matte, or `--pdf-presentation book` for aged art, tilt, shadow, and a thick edge on A4 pages. Both force generation to `3:4` unless `--aspect-ratio` is set, even when `--page-format` is present. See [PDF export](PDF.md).
- Add `--narrate` and optionally `--narrator-voice Charon` for MP3 output. Narration is enabled by the CLI flag.

## What the CLI generates

The full-comic pipeline reads the vocabulary, generates and validates a story bundle (story, character bible, title, and panel script), reserves the destination using a generation identity, and saves supporting text. It then renders the cover, story pages, gallery pages, and back cover in order, using reference images for consistency. Finally it copies gallery images, attempts PDF assembly, and optionally narrates the story. Prompt templates come from the configured directory (`./prompts` by default), with embedded templates as a fallback for missing files.

For the example above, the output root contains:

```text
output/comics/
├── assets/forest-adventure/
│   ├── .comicforge-identity
│   ├── forest-adventure_story.txt
│   ├── forest-adventure_comic_vocabulary.txt
│   ├── forest-adventure_theme.txt
│   ├── forest-adventure_cover.png
│   ├── forest-adventure_page_1.png
│   ├── forest-adventure_page_2.png
│   ├── forest-adventure_gallery_1.png
│   └── forest-adventure_back.png
├── gallery/forest-adventure_gallery_1.png
└── PDF/forest-adventure.pdf
```

With narration enabled, the assets directory also receives `forest-adventure_narration.mp3` on success, or `forest-adventure_tts_todo.txt` when narration cannot be produced.

## Verify and report the result

Check the exit status, warnings, and actual files. PDF assembly and narration failures can be reported as warnings while the process still exits successfully. A successful exit alone does not establish that a PDF or MP3 exists.

For the example above:

```bash
ls -lah "$comicforge_run_dir/output/comics/assets/forest-adventure/"
test -s "$comicforge_run_dir/output/comics/PDF/forest-adventure.pdf"
```

Inspect generated page images when image viewing is available. Report the final paths to the assets, PDF, and requested audio, plus any missing outputs or warnings. If dependencies or credentials are missing, explain what is needed to finish.

## Interrupted runs and existing output

Use a fresh slug or output root for a new story. Full-comic runs store `.comicforge-identity` and reject conflicting generations before replacing existing artifacts. Existing assets, PDFs, or gallery copies without an identity also require a fresh destination.

An interrupted run can reuse existing page images only when its complete newly generated story bundle, vocabulary, and artwork settings match the stored identity. The story model is called again on every invocation, so reusing a slug does not guarantee a resume. Explicit `--style` and `--theme` reduce randomness but do not make the story response deterministic. Do not remove the identity to force a resume. Template and provider/model changes are outside the identity check; choose a fresh destination for those changes as well.

## Generate one image from a prompt

Use `--prompt` for a single illustration. It cannot be combined with `--vocab` and does not run the comic, PDF, or narration pipeline.

```bash
./comicforge \
  --config "$comicforge_run_dir/config.yaml" \
  --prompt "a robot reading a newspaper" \
  --style "watercolor storybook illustration" \
  --output "$comicforge_run_dir/output" \
  --slug manual-robot
```

This writes `output/comics/assets/manual-robot/prompt.png` beneath the same run directory. Without `--slug`, the text model generates a title with a deterministic fallback if title generation fails. Prompt mode does not use the full-comic generation identity; choose a fresh destination to retain previous images.

See [Usage](USAGE.md) for the flag reference and [Architecture](ARCHITECTURE.md) for implementation details.
