# ComicForge

![ComicForge](assets/comicforge.png)

ComicForge turns a vocabulary list into a comic with Gemini-generated stories and artwork, PDF export, and optional MP3 narration. It also supports generating a single image from a direct prompt.

## Quick start

You need Go (see [`go.mod`](go.mod)), Mage, and a Gemini API key. Install ImageMagick for PDF export and `ffmpeg` for narration; see [installation and prerequisites](./docs/INSTALLATION.md).

Run from the repository root:

```bash
mage build
cp config.yaml.example config.yaml
export GOOGLE_API_KEY="your-api-key"
printf 'apple\ncat\nchair\n' > words.txt
./comicforge --vocab words.txt --config config.yaml --output out
```

Edit `config.yaml` to choose the story language, page count, and styles. The example uses English with Latin script; match these settings to your vocabulary.

Output goes into `out/comics/`: story files and page PNGs under `assets/<slug>/`, gallery copies under `gallery/`, and PDFs under `PDF/` when ImageMagick is available. Add `--narrate` for audio.

## Documentation

- [Generate a comic](./docs/GENERATING.md) — a complete workflow for people and agents, including verification and reruns.
- [Installation and development](./docs/INSTALLATION.md) — dependencies, builds, tests, and Mage tasks.
- [Configuration](./docs/CONFIGURATION.md) — config files, environment variables, providers, and style selection.
- [Usage](./docs/USAGE.md) — vocabulary format, language settings, CLI flags, output, reruns, and manual prompts.
- [Page shape and PDF export](./docs/PDF.md) — aspect ratios, A4 print and book layouts, compression, and examples.
- [Style and theme reference sheet](./docs/STYLEGRID.md) — preview all configured styles and genres with `stylegrid`.
- [Architecture](./docs/ARCHITECTURE.md) — packages, providers, and the generation pipeline.
