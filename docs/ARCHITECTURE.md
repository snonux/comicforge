# ComicForge Architecture

This document describes the architecture of **ComicForge**, a Go CLI application that generates AI-powered comic books from vocabulary word lists or direct prompts.

## Table of Contents

1. [High-Level Overview](#high-level-overview)
2. [System Architecture](#system-architecture)
3. [Package Structure](#package-structure)
4. [Data Flow](#data-flow)
5. [Provider Architecture](#provider-architecture)
6. [Configuration System](#configuration-system)
7. [Comic Generation Pipeline](#comic-generation-pipeline)
8. [Key Design Patterns](#key-design-patterns)

---

## High-Level Overview

ComicForge is a command-line tool that orchestrates multiple AI providers to produce complete comic books. Given a vocabulary file containing words in a target language, it:

1. Generates a story incorporating those words
2. Creates a character bible for visual consistency
3. Renders comic pages (cover, story pages, gallery, back cover)
4. Assembles a PDF
5. Optionally generates cinematic audio narration

---

## System Architecture

```mermaid
graph TB
    subgraph CLI["Command Layer"]
        CMD["cmd/comicforge/main.go<br/>Minimal bootstrap"]
        CLI_GO["cmd/comicforge/cli.go<br/>Cobra flags & wiring"]
    end

    subgraph Internal["Internal Packages"]
        CFG["internal/config<br/>Configuration loading"]
        PRV["internal/provider<br/>Provider interfaces"]
        COM["internal/comic<br/>Generation pipeline"]
        IMG["internal/image<br/>Image providers"]
        TXT["internal/text<br/>Text providers"]
        TTS["internal/tts<br/>TTS providers"]
        VOC["internal/vocab<br/>Vocabulary parsing"]
        HTP["internal/httpctx<br/>HTTP/GenAI clients"]
    end

    subgraph External["External Services"]
        GEMINI["Google Gemini API<br/>(Text, Image, TTS)"]
        FS["Local Filesystem<br/>(Output, PDFs, Audio)"]
    end

    CMD --> CLI_GO
    CLI_GO --> CFG
    CLI_GO --> COM
    CLI_GO --> PRV
    PRV --> TXT
    PRV --> IMG
    PRV --> TTS
    COM --> TXT
    COM --> IMG
    COM --> TTS
    COM --> VOC
    TXT --> HTP
    IMG --> HTP
    TTS --> HTP
    HTP --> GEMINI
    COM --> FS
```

---

## Package Structure

```mermaid
graph LR
    subgraph Entry["Entry Point"]
        M[main.go]
        C[cli.go]
    end

    subgraph Config["Configuration"]
        CFG[config.go]
        HOME[home.go]
    end

    subgraph Providers["Provider Layer"]
        P[provider.go]
        TR[text/registry.go]
        IR[image/registry.go]
        TTR[tts/registry.go]
        TG[text/gemini.go]
        IG[image/gemini.go]
        TTG[tts/gemini.go]
    end

    subgraph Pipeline["Comic Pipeline"]
        RUN[runner.go]
        GEN[generator.go]
        ART[artist.go]
        NAR[narrator.go]
        TYP[types.go]
    end

    subgraph Support["Supporting Packages"]
        VOC[vocab/reader.go]
        HTP[httpctx/httpctx.go]
        CIR[apicircuit/apicircuit.go]
    end

    M --> C
    C --> CFG
    C --> RUN
    C --> TR
    C --> IR
    C --> TTR
    RUN --> GEN
    RUN --> ART
    RUN --> NAR
    RUN --> VOC
    GEN --> TG
    ART --> IG
    ART --> TG
    NAR --> TTG
    NAR --> TG
    TR --> P
    IR --> P
    TTR --> P
    TG --> HTP
    IG --> HTP
    TTG --> HTP
```

---

## Data Flow

### Vocabulary Mode (Full Comic)

```mermaid
sequenceDiagram
    participant User
    participant CLI as cli.go
    participant Run as Runner
    participant Gen as Generator
    participant Art as Artist
    participant Nar as Narrator
    participant TP as TextProvider
    participant IP as ImageProvider
    participant TTS as TTSProvider
    participant FS as Filesystem

    User->>CLI: comicforge --vocab words.txt
    CLI->>Run: NewRunner(cfg)
    Run->>Gen: NewGenerator()
    Run->>Art: NewArtist()
    Run->>Nar: NewNarrator()

    Run->>Gen: GenerateFull(entries)
    Gen->>TP: GenerateText(storyPrompt)
    TP-->>Gen: story + bible + title + panelScript
    Gen-->>Run: GenerateResult

    Run->>Art: DrawComicPages(story, bible, title, entries, panelScript)
    loop For each page
        Art->>Art: Render prompt template
        Art->>IP: GenerateImageWithReferences(prompt, refs)
        IP-->>Art: image.png
        Art->>FS: Write page file
    end
    Art-->>Run: []pagePaths

    Run->>FS: Save story.txt
    Run->>FS: Save vocabulary.txt
    Run->>FS: Save theme.txt
    Run->>FS: Assemble PDF

    opt Narration enabled
        Run->>Nar: Narrate(storyText)
        Nar->>TP: GenerateText(intro prompt)
        Nar->>TTS: GenerateAudio(chunks)
        Nar->>FS: Save narration.mp3
    end

    Run-->>CLI: success
    CLI-->>User: Done
```

### Prompt Mode (Single Image)

```mermaid
sequenceDiagram
    participant User
    participant CLI as cli.go
    participant Run as Runner
    participant Art as Artist
    participant TP as TextProvider
    participant IP as ImageProvider
    participant FS as Filesystem

    User->>CLI: comicforge --prompt "dragon in space"
    CLI->>Run: NewRunner(cfg)
    Run->>Art: NewArtist()

    Run->>Run: Generate title slug via TP
    Run->>Art: generatePromptImage(prompt)
    Art->>Art: Render manual prompt template
    Art->>IP: GenerateImage(renderedPrompt)
    IP-->>Art: image.png
    Art->>FS: Write prompt.png

    Run-->>CLI: success
    CLI-->>User: Done
```

---

## Provider Architecture

ComicForge uses a **registry-based provider system** that abstracts over different AI backends. Currently, only **Google Gemini** is fully implemented; OpenAI is stubbed for future extension.

```mermaid
classDiagram
    class TextProvider {
        <<interface>>
        +Name() string
        +IsAvailable() error
        +GenerateText(ctx, prompt) (string, error)
    }

    class ImageProvider {
        <<interface>>
        +Name() string
        +IsAvailable() error
        +GenerateImage(ctx, prompt, outputFile) error
    }

    class AspectRatioImageProvider {
        <<interface>>
        +GenerateImageWithAspectRatio(ctx, prompt, outputFile, aspectRatio) error
    }

    class TTSProvider {
        <<interface>>
        +Name() string
        +IsAvailable() error
        +GenerateAudio(ctx, text, outputFile) error
    }

    class GeminiTextProvider {
        +client *genai.Client
        +model string
        +GenerateText(ctx, prompt) (string, error)
    }

    class GeminiImageProvider {
        +client *genai.Client
        +config *GeminiConfig
        +GenerateImage(ctx, prompt, outputFile) error
        +GenerateImageWithReferences(ctx, prompt, outputFile, refs) error
    }

    class GeminiTTSProvider {
        +client *genai.Client
        +model string
        +voice string
        +GenerateAudio(ctx, text, outputFile) error
    }

    class Registry~T, C~ {
        +Register(name, factory)
        +Resolve(name) Factory
        +New(name, cfg) T
    }

    TextProvider <|.. GeminiTextProvider
    ImageProvider <|.. GeminiImageProvider
    ImageProvider <|-- AspectRatioImageProvider
    AspectRatioImageProvider <|.. GeminiImageProvider
    TTSProvider <|.. GeminiTTSProvider
    Registry --> TextProvider
    Registry --> ImageProvider
    Registry --> TTSProvider
```

### Provider Registry Flow

```mermaid
graph LR
    A[CLI Flags<br/>--text-provider gemini] --> B[Config.Normalize]
    B --> C[Registry.Resolve"gemini"]
    C --> D[Factory Function]
    D --> E[GeminiProvider]
    E --> F[IsAvailable Check]
    F --> G[Inject into Runner]
```

---

## Configuration System

Configuration is loaded via **Viper** with a three-layer priority:

1. **Defaults** (code-defined)
2. **Config file** (`config.yaml`)
3. **Environment variables** (`COMICFORGE_*`)
4. **CLI flags** (highest priority)

```mermaid
graph TD
    A[DefaultConfig] --> B[Viper SetDefaults]
    C[config.yaml] --> D[Viper ReadInConfig]
    E[COMICFORGE_API_GOOGLE_API_KEY] --> F[Viper AutomaticEnv]
    G[CLI Flags<br/>--text-model, --style] --> H[Apply Overrides]

    B --> I[Loaded Config]
    D --> I
    F --> I
    H --> I
    I --> J[Validate Providers]
    J --> K[Return *Config]
```

The `Config` struct implements multiple small interfaces so each package only depends on what it needs:

```mermaid
graph TD
    CFG[Config] --> TC[TextConfig]
    CFG --> IC[ImageConfig]
    CFG --> TTC[TTSConfig]
    CFG --> TXTC[text.Config]
    CFG --> IMGC[image.Config]
    CFG --> TTSC[tts.Config]

    TC --> TXT[internal/text]
    IC --> IMG[internal/image]
    TTC --> TTS[internal/tts]
```

---

## Comic Generation Pipeline

The pipeline is divided into three specialized stages:

```mermaid
graph LR
    subgraph Stage1["Stage 1: Story Generation"]
        direction TB
        VOC[Vocabulary Words] --> GEN
        GEN[Generator] --> STORY[Story Text]
        GEN --> BIBLE[Character Bible]
        GEN --> TITLE[Comic Title]
        GEN --> PANEL[Panel Script]
    end

    subgraph Stage2["Stage 2: Visual Art"]
        direction TB
        STORY --> ART[Artist]
        BIBLE --> ART
        PANEL --> ART
        ART --> COVER[Cover Page]
        ART --> PAGES[Story Pages]
        ART --> GALLERY[Gallery Pages]
        ART --> BACK[Back Cover]
    end

    subgraph Stage3["Stage 3: Assembly"]
        direction TB
        COVER --> PDF[PDF Assembly]
        PAGES --> PDF
        GALLERY --> PDF
        BACK --> PDF
        STORY --> NAR[Narrator]
        NAR --> AUDIO[MP3 Narration]
    end
```

### Reference Image Chaining

To maintain visual consistency across comic pages, the Artist passes previous page images as **reference images** to the image provider:

```mermaid
graph LR
    P1[Page 1] -->|ref| P2[Page 2]
    P2 -->|ref| P3[Page 3]
    P3 -->|ref| P4[...]

    style P1 fill:#e1f5fe
    style P2 fill:#e1f5fe
    style P3 fill:#e1f5fe
```

The `appendRef` function keeps only the **first and most recent** reference images to avoid exceeding API limits.

---

## Key Design Patterns

### Dependency Injection

The CLI uses a `commandDeps` struct to inject provider factories, making the entire command layer testable without real API calls:

```go
type commandDeps struct {
    loadConfig       func(string) (*config.Config, error)
    newTextProvider  func(*config.Config) (provider.TextProvider, error)
    newImageProvider func(*config.Config) (provider.ImageProvider, error)
    newTTSProvider   func(*config.Config, string) (provider.TTSProvider, error)
    newRunner        func(*comic.RunnerConfig) comic.StoryRunner
}
```

### Template-Driven Prompts

All LLM prompts are **Go text/template files** rendered with structured data. Templates live in `prompts/` and can be overridden at runtime via `--prompts-dir`:

```mermaid
graph LR
    DATA[Template Data<br/>Language, Style, Bible] --> RENDER[config.RenderPrompt]
    TPL[Template File<br/>story_full_prompt.md] --> RENDER
    RENDER --> PROMPT[Final Prompt]
    PROMPT --> LLM[Gemini API]
```

### Circuit Breaker

The `internal/apicircuit` package provides circuit breaker protection for API calls to prevent cascading failures when AI providers are unavailable.

### Defensive Defaults

Every component uses the **functional options / config struct** pattern with sensible defaults:

```mermaid
graph TD
    A[User Config] --> B{Field Set?}
    B -->|Yes| C[Use User Value]
    B -->|No| D[Use Default Value]
    C --> E[Initialized Component]
    D --> E
```

---

## File Output Structure

When running with `--output .` and a comic titled "Space Adventure":

```
comics/
├── assets/
│   └── space-adventure/
│       ├── space-adventure_cover.png
│       ├── space-adventure_page_1.png
│       ├── space-adventure_page_2.png
│       ├── space-adventure_gallery_1.png
│       ├── space-adventure_back.png
│       ├── space-adventure_story.txt
│       ├── space-adventure_comic_vocabulary.txt
│       ├── space-adventure_theme.txt
│       └── space-adventure_narration.mp3
├── PDF/
│   └── space-adventure.pdf
└── gallery/
    ├── space-adventure_gallery_1.png
    └── ...
```

---

## Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23+ |
| CLI Framework | Cobra |
| Configuration | Viper |
| AI Provider SDK | `google.golang.org/genai` |
| PDF Generation | Custom assembly via `internal/comic/pdf.go` |
| Audio Processing | `ffmpeg` (external dependency) |
| Testing | Go standard testing + test tables |
| Module Path | `codeberg.org/snonux/comicforge` |

---

## Future Extension Points

- **New AI Providers**: Implement `TextProvider`/`ImageProvider`/`TTSProvider` and register in the respective registries
- **New Style Modes**: Add to `StyleConfig` and update `pickStyle()` in `types.go`
- **Custom Output Formats**: The `assemblePDF` field in `Runner` is injectable for alternative exporters; the default implementation takes `PDFAssembleOptions` (density, JPEG quality, print-style matting)
- **Additional Languages**: Extend `localization.go` with new script/language mappings
