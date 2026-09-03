package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	version "github.com/snonux/comicforge/internal"
	"github.com/snonux/comicforge/internal/comic"
	"github.com/snonux/comicforge/internal/config"
	"github.com/snonux/comicforge/internal/provider"
)

func TestRootCommandAppliesConfigAndFlagOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(`
provider:
  text: openai
  image: openai
  tts: openai
models:
  text: config-text
  image: config-image
  image_text: config-image-text
  tts: config-tts
comic:
  story_pages: 7
  gallery_pages: 3
  panels_per_page: 2
language:
  story_language: Bulgarian
  script: Cyrillic
narration:
  voices:
    - Fenrir
    - Charon
prompts_dir: ./config-prompts
`)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	vocabPath := filepath.Join(tmpDir, "vocab.txt")
	if err := os.WriteFile(vocabPath, []byte("ябълка = apple\nкнига = book\n"), 0o644); err != nil {
		t.Fatalf("write vocab: %v", err)
	}

	var gotTextCfg *config.Config
	var gotImageCfg *config.Config
	var gotTTSCfg *config.Config
	var gotTTSVoice string
	var gotRunnerCfg *comic.RunnerConfig
	runner := &recordingRunner{}

	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(path string) (*config.Config, error) {
			return config.Load(path)
		},
		newTextProvider: func(cfg *config.Config) (provider.TextProvider, error) {
			gotTextCfg = cfg
			return noopProvider{}, nil
		},
		newImageProvider: func(cfg *config.Config) (provider.ImageProvider, error) {
			gotImageCfg = cfg
			return noopProvider{}, nil
		},
		newTTSProvider: func(cfg *config.Config, voice string) (provider.TTSProvider, error) {
			gotTTSCfg = cfg
			gotTTSVoice = voice
			return noopProvider{}, nil
		},
		newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner {
			gotRunnerCfg = cfg
			runner.cfg = cfg
			return runner
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--config", configPath,
		"--vocab", vocabPath,
		"--text-provider", provider.Gemini,
		"--image-provider", provider.Gemini,
		"--tts-provider", provider.Gemini,
		"--text-model", "flag-text",
		"--image-model", "flag-image",
		"--image-text-model", "flag-image-text",
		"--tts-model", "flag-tts",
		"--prompts-dir", filepath.Join(tmpDir, "prompts"),
		"--output", filepath.Join(tmpDir, "out"),
		"--style", "noir",
		"--theme", "mystery",
		"--slug", "forced-slug",
		"--narrate",
		"--narrator-voice", "Aoede",
		"--ultra-realistic",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v\noutput:\n%s", err, buf.String())
	}

	if runner.runs != 1 {
		t.Fatalf("runner.runs = %d, want 1", runner.runs)
	}
	if got, want := runner.batchFile, vocabPath; got != want {
		t.Fatalf("runner.batchFile = %q, want %q", got, want)
	}
	if gotRunnerCfg == nil {
		t.Fatal("runner config was not captured")
	}
	if gotTextCfg == nil || gotImageCfg == nil || gotTTSCfg == nil {
		t.Fatal("provider configs were not captured")
	}
	if got, want := gotTextCfg.Provider.Text, provider.Gemini; got != want {
		t.Fatalf("text provider = %q, want %q", got, want)
	}
	if got, want := gotImageCfg.Provider.Image, provider.Gemini; got != want {
		t.Fatalf("image provider = %q, want %q", got, want)
	}
	if got, want := gotTTSCfg.Provider.TTS, provider.Gemini; got != want {
		t.Fatalf("tts provider = %q, want %q", got, want)
	}
	if got, want := gotTextCfg.Models.Text, "flag-text"; got != want {
		t.Fatalf("text model = %q, want %q", got, want)
	}
	if got, want := gotImageCfg.Models.Image, "flag-image"; got != want {
		t.Fatalf("image model = %q, want %q", got, want)
	}
	if got, want := gotImageCfg.Models.ImageText, "flag-image-text"; got != want {
		t.Fatalf("image text model = %q, want %q", got, want)
	}
	if got, want := gotTTSCfg.Models.TTS, "flag-tts"; got != want {
		t.Fatalf("tts model = %q, want %q", got, want)
	}
	if got, want := gotTextCfg.PromptsDir, filepath.Join(tmpDir, "prompts"); got != want {
		t.Fatalf("prompts dir = %q, want %q", got, want)
	}
	if got, want := gotTTSVoice, "Aoede"; got != want {
		t.Fatalf("tts voice = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.OutputDir, filepath.Join(tmpDir, "out"); got != want {
		t.Fatalf("output dir = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.Style, "noir"; got != want {
		t.Fatalf("style = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.Theme, "mystery"; got != want {
		t.Fatalf("theme = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.Slug, "forced-slug"; got != want {
		t.Fatalf("slug = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.NarrateEnabled, true; got != want {
		t.Fatalf("narrate = %t, want %t", got, want)
	}
	if got, want := gotRunnerCfg.NarratorVoice, "Aoede"; got != want {
		t.Fatalf("narrator voice = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.Language, "Bulgarian"; got != want {
		t.Fatalf("language = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.Script, "Cyrillic"; got != want {
		t.Fatalf("script = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.StoryPages, 7; got != want {
		t.Fatalf("story pages = %d, want %d", got, want)
	}
	if got, want := gotRunnerCfg.GalleryPages, 3; got != want {
		t.Fatalf("gallery pages = %d, want %d", got, want)
	}
	if got, want := gotRunnerCfg.PanelsPerPage, 2; got != want {
		t.Fatalf("panels per page = %d, want %d", got, want)
	}
	if gotRunnerCfg.UltraRealistic == nil || !*gotRunnerCfg.UltraRealistic {
		t.Fatalf("ultra realistic = %#v, want true", gotRunnerCfg.UltraRealistic)
	}
}

func TestPDFPresentationPrintAndBookForcesA4AspectRatio(t *testing.T) {
	for _, presentation := range []string{"print", "book"} {
		t.Run(presentation, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(strings.TrimSpace(`
comic:
  aspect_ratio: "16:9"
pdf:
  presentation: none
`)), 0o644); err != nil {
				t.Fatalf("write config: %v", err)
			}
			vocabPath := filepath.Join(tmpDir, "vocab.txt")
			if err := os.WriteFile(vocabPath, []byte("ябълка = apple\n"), 0o644); err != nil {
				t.Fatalf("write vocab: %v", err)
			}

			var gotRunnerCfg *comic.RunnerConfig
			cmd := newRootCommandWithDeps(commandDeps{
				loadConfig: func(path string) (*config.Config, error) {
					return config.Load(path)
				},
				newTextProvider:  func(*config.Config) (provider.TextProvider, error) { return noopProvider{}, nil },
				newImageProvider: func(*config.Config) (provider.ImageProvider, error) { return noopProvider{}, nil },
				newTTSProvider:   func(*config.Config, string) (provider.TTSProvider, error) { return noopProvider{}, nil },
				newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner {
					gotRunnerCfg = cfg
					return &recordingRunner{}
				},
			})
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{
				"--config", configPath,
				"--vocab", vocabPath,
				"--pdf-presentation", presentation,
			})

			if err := cmd.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("ExecuteContext() error = %v\n%s", err, buf.String())
			}
			if gotRunnerCfg == nil {
				t.Fatal("runner config was not captured")
			}
			if got, want := gotRunnerCfg.AspectRatio, comic.BookPageAspectRatio; got != want {
				t.Fatalf("aspect ratio = %q, want %q (A4-class 3:4)", got, want)
			}
			if got, want := gotRunnerCfg.PDF.Presentation, presentation; got != want {
				t.Fatalf("pdf presentation = %q, want %q", got, want)
			}
		})
	}
}

func TestPDFPresentationExplicitAspectRatioOverridesA4Default(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(`
comic:
  aspect_ratio: "16:9"
`)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	vocabPath := filepath.Join(tmpDir, "vocab.txt")
	if err := os.WriteFile(vocabPath, []byte("ябълка = apple\n"), 0o644); err != nil {
		t.Fatalf("write vocab: %v", err)
	}

	var gotRunnerCfg *comic.RunnerConfig
	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(path string) (*config.Config, error) {
			return config.Load(path)
		},
		newTextProvider:  func(*config.Config) (provider.TextProvider, error) { return noopProvider{}, nil },
		newImageProvider: func(*config.Config) (provider.ImageProvider, error) { return noopProvider{}, nil },
		newTTSProvider:   func(*config.Config, string) (provider.TTSProvider, error) { return noopProvider{}, nil },
		newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner {
			gotRunnerCfg = cfg
			return &recordingRunner{}
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--config", configPath,
		"--vocab", vocabPath,
		"--pdf-presentation", "print",
		"--aspect-ratio", "21:9",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v\n%s", err, buf.String())
	}
	if gotRunnerCfg == nil {
		t.Fatal("runner config was not captured")
	}
	if got, want := gotRunnerCfg.AspectRatio, "21:9"; got != want {
		t.Fatalf("aspect ratio = %q, want %q", got, want)
	}
}

func TestRootCommandUsesRealisticWeightWhenUltraModeUnset(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(`
story:
  realistic_weight: 0
`)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var gotRunnerCfg *comic.RunnerConfig
	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(path string) (*config.Config, error) {
			return config.Load(path)
		},
		newTextProvider:  func(*config.Config) (provider.TextProvider, error) { return noopProvider{}, nil },
		newImageProvider: func(*config.Config) (provider.ImageProvider, error) { return noopProvider{}, nil },
		newTTSProvider:   func(*config.Config, string) (provider.TTSProvider, error) { return noopProvider{}, nil },
		newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner {
			gotRunnerCfg = cfg
			return &recordingRunner{}
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	vocabPath := filepath.Join(tmpDir, "vocab.txt")
	if err := os.WriteFile(vocabPath, []byte("ябълка = apple\n"), 0o644); err != nil {
		t.Fatalf("write vocab: %v", err)
	}
	cmd.SetArgs([]string{"--config", configPath, "--vocab", vocabPath})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v\noutput:\n%s", err, buf.String())
	}
	if gotRunnerCfg == nil {
		t.Fatal("runner config was not captured")
	}
	if got, want := gotRunnerCfg.RealisticWeight, 0.0; got != want {
		t.Fatalf("realistic weight = %v, want %v", got, want)
	}
	if gotRunnerCfg.UltraRealistic != nil {
		t.Fatalf("ultra realistic override = %#v, want nil when flags are unset", gotRunnerCfg.UltraRealistic)
	}
}

func TestRootCommandVersionSkipsRequiredFlags(t *testing.T) {
	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(string) (*config.Config, error) {
			t.Fatal("loadConfig should not be called for --version")
			return nil, nil
		},
		newTextProvider: func(*config.Config) (provider.TextProvider, error) {
			t.Fatal("newTextProvider should not be called for --version")
			return noopProvider{}, nil
		},
		newImageProvider: func(*config.Config) (provider.ImageProvider, error) {
			t.Fatal("newImageProvider should not be called for --version")
			return noopProvider{}, nil
		},
		newTTSProvider: func(*config.Config, string) (provider.TTSProvider, error) {
			t.Fatal("newTTSProvider should not be called for --version")
			return noopProvider{}, nil
		},
		newRunner: func(*comic.RunnerConfig) comic.StoryRunner {
			t.Fatal("newRunner should not be called for --version")
			return &recordingRunner{}
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if got, want := strings.TrimSpace(buf.String()), version.Version; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRootCommandRejectsConflictingUltraFlags(t *testing.T) {
	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(string) (*config.Config, error) {
			return config.DefaultConfig(), nil
		},
		newTextProvider: func(*config.Config) (provider.TextProvider, error) {
			return noopProvider{}, nil
		},
		newImageProvider: func(*config.Config) (provider.ImageProvider, error) {
			return noopProvider{}, nil
		},
		newTTSProvider: func(*config.Config, string) (provider.TTSProvider, error) {
			return noopProvider{}, nil
		},
		newRunner: func(*comic.RunnerConfig) comic.StoryRunner {
			t.Fatal("newRunner should not be called when ultra-realistic flags conflict")
			return &recordingRunner{}
		},
	})
	cmd.SetArgs([]string{
		"--vocab", "words.txt",
		"--ultra-realistic",
		"--no-ultra-realistic",
	})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext() error = nil, want conflict error")
	}
	if !strings.Contains(err.Error(), "only one of --ultra-realistic and --no-ultra-realistic may be set") {
		t.Fatalf("ExecuteContext() error = %v, want conflict error", err)
	}
}

func TestRootCommandRejectsConflictingUltraFlagsInPromptMode(t *testing.T) {
	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(string) (*config.Config, error) {
			t.Fatal("loadConfig should not be called when ultra-realistic flags conflict")
			return nil, nil
		},
		newTextProvider: func(*config.Config) (provider.TextProvider, error) {
			t.Fatal("newTextProvider should not be called when ultra-realistic flags conflict")
			return noopProvider{}, nil
		},
		newImageProvider: func(*config.Config) (provider.ImageProvider, error) {
			t.Fatal("newImageProvider should not be called when ultra-realistic flags conflict")
			return noopProvider{}, nil
		},
		newTTSProvider: func(*config.Config, string) (provider.TTSProvider, error) {
			t.Fatal("newTTSProvider should not be called when ultra-realistic flags conflict")
			return noopProvider{}, nil
		},
		newRunner: func(*comic.RunnerConfig) comic.StoryRunner {
			t.Fatal("newRunner should not be called when ultra-realistic flags conflict")
			return &recordingRunner{}
		},
	})
	cmd.SetArgs([]string{
		"--prompt", "draw a robot",
		"--ultra-realistic",
		"--no-ultra-realistic",
	})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext() error = nil, want conflict error")
	}
	if !strings.Contains(err.Error(), "only one of --ultra-realistic and --no-ultra-realistic may be set") {
		t.Fatalf("ExecuteContext() error = %v, want conflict error", err)
	}
}

func TestRootCommandProviderFlagsDefaultToGemini(t *testing.T) {
	cmd := newRootCommand()
	for _, flagName := range []string{"text-provider", "image-provider", "tts-provider"} {
		flag := mustLookupFlag(t, cmd, flagName)
		if got, want := flag.DefValue, provider.Gemini; got != want {
			t.Fatalf("%s default = %q, want %q", flagName, got, want)
		}
	}
}

func TestRootCommandSkipsTTSProviderWhenNarrationDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	vocabPath := filepath.Join(tmpDir, "vocab.txt")
	if err := os.WriteFile(vocabPath, []byte("ябълка = apple\n"), 0o644); err != nil {
		t.Fatalf("write vocab: %v", err)
	}

	var ttsCalled bool
	var gotRunnerCfg *comic.RunnerConfig
	runner := &recordingRunner{}

	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(string) (*config.Config, error) {
			return config.DefaultConfig(), nil
		},
		newTextProvider: func(cfg *config.Config) (provider.TextProvider, error) {
			return noopProvider{}, nil
		},
		newImageProvider: func(cfg *config.Config) (provider.ImageProvider, error) {
			return noopProvider{}, nil
		},
		newTTSProvider: func(*config.Config, string) (provider.TTSProvider, error) {
			ttsCalled = true
			return nil, fmt.Errorf("tts provider should not be resolved when narration is disabled")
		},
		newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner {
			gotRunnerCfg = cfg
			runner.cfg = cfg
			return runner
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--vocab", vocabPath})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v\noutput:\n%s", err, buf.String())
	}
	if ttsCalled {
		t.Fatal("newTTSProvider was called, want narration-disabled path to skip TTS setup")
	}
	if gotRunnerCfg == nil {
		t.Fatal("runner config was not captured")
	}
	if gotRunnerCfg.MainTTSProvider != nil || gotRunnerCfg.ConclusionTTSProvider != nil {
		t.Fatalf("runner TTS providers = %#v / %#v, want nil / nil", gotRunnerCfg.MainTTSProvider, gotRunnerCfg.ConclusionTTSProvider)
	}
	if runner.runs != 1 {
		t.Fatalf("runner.runs = %d, want 1", runner.runs)
	}
}

func TestRootCommandPromptModeSkipsVocabFlow(t *testing.T) {
	tmpDir := t.TempDir()

	var ttsCalled bool
	var gotRunnerCfg *comic.RunnerConfig
	runner := &recordingRunner{}

	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(string) (*config.Config, error) {
			return config.DefaultConfig(), nil
		},
		newTextProvider: func(*config.Config) (provider.TextProvider, error) {
			return noopProvider{}, nil
		},
		newImageProvider: func(cfg *config.Config) (provider.ImageProvider, error) {
			return noopProvider{}, nil
		},
		newTTSProvider: func(*config.Config, string) (provider.TTSProvider, error) {
			ttsCalled = true
			return noopProvider{}, nil
		},
		newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner {
			gotRunnerCfg = cfg
			runner.cfg = cfg
			return runner
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--prompt", "a robot reading a newspaper",
		"--output", filepath.Join(tmpDir, "out"),
		"--slug", "manual-robot",
		"--style", "noir",
		"--theme", "mystery",
		"--ultra-realistic",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v\noutput:\n%s", err, buf.String())
	}
	if ttsCalled {
		t.Fatal("newTTSProvider was called, want prompt mode to skip narration setup")
	}
	if gotRunnerCfg == nil {
		t.Fatal("runner config was not captured")
	}
	if got, want := runner.prompt, "a robot reading a newspaper"; got != want {
		t.Fatalf("prompt = %q, want %q", got, want)
	}
	if got, want := runner.promptRuns, 1; got != want {
		t.Fatalf("prompt runs = %d, want %d", got, want)
	}
	if got, want := gotRunnerCfg.OutputDir, filepath.Join(tmpDir, "out"); got != want {
		t.Fatalf("output dir = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.Slug, "manual-robot"; got != want {
		t.Fatalf("slug = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.Style, "noir"; got != want {
		t.Fatalf("style = %q, want %q", got, want)
	}
	if got, want := gotRunnerCfg.Theme, "mystery"; got != want {
		t.Fatalf("theme = %q, want %q", got, want)
	}
	if gotRunnerCfg.UltraRealistic == nil || !*gotRunnerCfg.UltraRealistic {
		t.Fatalf("ultra realistic = %#v, want true", gotRunnerCfg.UltraRealistic)
	}
}

func TestRootCommandRejectsPromptAndVocabTogether(t *testing.T) {
	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(string) (*config.Config, error) {
			return config.DefaultConfig(), nil
		},
		newTextProvider: func(*config.Config) (provider.TextProvider, error) {
			t.Fatal("newTextProvider should not be called when flags conflict")
			return noopProvider{}, nil
		},
		newImageProvider: func(*config.Config) (provider.ImageProvider, error) {
			t.Fatal("newImageProvider should not be called when flags conflict")
			return noopProvider{}, nil
		},
		newTTSProvider: func(*config.Config, string) (provider.TTSProvider, error) {
			t.Fatal("newTTSProvider should not be called when flags conflict")
			return noopProvider{}, nil
		},
		newRunner: func(*comic.RunnerConfig) comic.StoryRunner {
			t.Fatal("newRunner should not be called when flags conflict")
			return &recordingRunner{}
		},
	})
	cmd.SetArgs([]string{
		"--vocab", "words.txt",
		"--prompt", "draw a robot",
	})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext() error = nil, want conflict error")
	}
	if !strings.Contains(err.Error(), "only one of --prompt and --vocab may be set") {
		t.Fatalf("ExecuteContext() error = %v, want conflict error", err)
	}
}

func TestRootCommandRejectsEmptyPromptFlag(t *testing.T) {
	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(string) (*config.Config, error) {
			t.Fatal("loadConfig should not be called for empty prompt validation")
			return nil, nil
		},
		newTextProvider: func(*config.Config) (provider.TextProvider, error) {
			t.Fatal("newTextProvider should not be called for empty prompt validation")
			return noopProvider{}, nil
		},
		newImageProvider: func(*config.Config) (provider.ImageProvider, error) {
			t.Fatal("newImageProvider should not be called for empty prompt validation")
			return noopProvider{}, nil
		},
		newTTSProvider: func(*config.Config, string) (provider.TTSProvider, error) {
			t.Fatal("newTTSProvider should not be called for empty prompt validation")
			return noopProvider{}, nil
		},
		newRunner: func(*comic.RunnerConfig) comic.StoryRunner {
			t.Fatal("newRunner should not be called for empty prompt validation")
			return &recordingRunner{}
		},
	})
	cmd.SetArgs([]string{"--prompt", ""})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext() error = nil, want empty prompt error")
	}
	if !strings.Contains(err.Error(), "--prompt is required when set") {
		t.Fatalf("ExecuteContext() error = %v, want empty prompt error", err)
	}
}

func TestBuildRunnerConfigMapsSharedFields(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Language.Story = "Bulgarian"
	cfg.Language.Script = "Cyrillic"
	cfg.Comic.StoryPages = 7
	cfg.Comic.GalleryPages = 3
	cfg.Comic.PanelsPerPage = 2
	cfg.Comic.AspectRatio = "2:3"
	cfg.Comic.PromptMaxChars = 900
	cfg.Comic.PageMaxRetries = 4
	cfg.Comic.PageRetryBaseSeconds = 3
	cfg.Narration.ChunkWords = 42
	cfg.Story.RealisticWeight = 0.25
	cfg.PDF.Density = 150
	cfg.PDF.JPEGQuality = 85
	cfg.PDF.Presentation = "print"
	cfg.Styles = config.StyleConfig{"noir": {"high contrast"}}

	flags := cliFlags{
		outputDir:      "/tmp/out",
		style:          "noir",
		theme:          "mystery",
		slug:           "forced-slug",
		ultraRealistic: true,
		narrateEnabled: true,
		narratorVoice:  "Aoede",
	}
	textProv := noopProvider{}
	imageProv := noopProvider{}

	got := buildRunnerConfig(cfg, flags, textProv, imageProv)
	if got == nil {
		t.Fatal("buildRunnerConfig returned nil")
	}
	if got.TextProvider != textProv {
		t.Fatalf("TextProvider = %#v, want injected text provider", got.TextProvider)
	}
	if got.ImageProvider != imageProv {
		t.Fatalf("ImageProvider = %#v, want injected image provider", got.ImageProvider)
	}
	if got.Prompts != cfg {
		t.Fatalf("Prompts = %#v, want cfg", got.Prompts)
	}
	if got, want := got.OutputDir, flags.outputDir; got != want {
		t.Fatalf("OutputDir = %q, want %q", got, want)
	}
	if got, want := got.Style, flags.style; got != want {
		t.Fatalf("Style = %q, want %q", got, want)
	}
	if got, want := got.Theme, flags.theme; got != want {
		t.Fatalf("Theme = %q, want %q", got, want)
	}
	if got, want := got.Slug, flags.slug; got != want {
		t.Fatalf("Slug = %q, want %q", got, want)
	}
	if got, want := got.Language, "Bulgarian"; got != want {
		t.Fatalf("Language = %q, want %q", got, want)
	}
	if got, want := got.Script, "Cyrillic"; got != want {
		t.Fatalf("Script = %q, want %q", got, want)
	}
	if got, want := got.StoryPages, 7; got != want {
		t.Fatalf("StoryPages = %d, want %d", got, want)
	}
	if got, want := got.GalleryPages, 3; got != want {
		t.Fatalf("GalleryPages = %d, want %d", got, want)
	}
	if got, want := got.PanelsPerPage, 2; got != want {
		t.Fatalf("PanelsPerPage = %d, want %d", got, want)
	}
	if got, want := got.ChunkWords, 42; got != want {
		t.Fatalf("ChunkWords = %d, want %d", got, want)
	}
	if got, want := got.AspectRatio, "2:3"; got != want {
		t.Fatalf("AspectRatio = %q, want %q", got, want)
	}
	if got, want := got.PromptMaxChars, 900; got != want {
		t.Fatalf("PromptMaxChars = %d, want %d", got, want)
	}
	if got, want := got.PageMaxRetries, 4; got != want {
		t.Fatalf("PageMaxRetries = %d, want %d", got, want)
	}
	if got, want := got.PageRetryBase, 3*time.Second; got != want {
		t.Fatalf("PageRetryBase = %v, want %v", got, want)
	}
	if got, want := got.RealisticWeight, 0.25; got != want {
		t.Fatalf("RealisticWeight = %v, want %v", got, want)
	}
	if got.UltraRealistic == nil || !*got.UltraRealistic {
		t.Fatalf("UltraRealistic = %#v, want true", got.UltraRealistic)
	}
	if got, want := got.PDF.Density, 150; got != want {
		t.Fatalf("PDF.Density = %d, want %d", got, want)
	}
	if got, want := got.PDF.JPEGQuality, 85; got != want {
		t.Fatalf("PDF.JPEGQuality = %d, want %d", got, want)
	}
	if got, want := got.PDF.Presentation, "print"; got != want {
		t.Fatalf("PDF.Presentation = %q, want %q", got, want)
	}
	if got, want := len(got.Styles["noir"]), 1; got != want {
		t.Fatalf("Styles[noir] len = %d, want %d", got, want)
	}
}

func TestBuildRunnerConfigLeavesNarrationFieldsUnset(t *testing.T) {
	cfg := config.DefaultConfig()
	flags := cliFlags{narrateEnabled: true, narratorVoice: "Aoede"}

	got := buildRunnerConfig(cfg, flags, noopProvider{}, noopProvider{})
	if got.NarrateEnabled {
		t.Fatal("NarrateEnabled = true, want false (caller must set)")
	}
	if got.NarratorVoice != "" {
		t.Fatalf("NarratorVoice = %q, want empty (caller must set)", got.NarratorVoice)
	}
	if got.MainTTSProvider != nil || got.ConclusionTTSProvider != nil {
		t.Fatalf("TTS providers = %#v / %#v, want nil / nil (caller must set)", got.MainTTSProvider, got.ConclusionTTSProvider)
	}
}

func TestBuildRunnerConfigNilConfigUsesZeroValues(t *testing.T) {
	flags := cliFlags{
		outputDir:        "/out",
		style:            "ink",
		theme:            "quiet",
		slug:             "nil-cfg",
		noUltraRealistic: true,
	}

	got := buildRunnerConfig(nil, flags, noopProvider{}, noopProvider{})
	if got == nil {
		t.Fatal("buildRunnerConfig(nil, ...) returned nil")
	}
	if got, want := got.OutputDir, "/out"; got != want {
		t.Fatalf("OutputDir = %q, want %q", got, want)
	}
	if got, want := got.Style, "ink"; got != want {
		t.Fatalf("Style = %q, want %q", got, want)
	}
	if got, want := got.Theme, "quiet"; got != want {
		t.Fatalf("Theme = %q, want %q", got, want)
	}
	if got, want := got.Slug, "nil-cfg"; got != want {
		t.Fatalf("Slug = %q, want %q", got, want)
	}
	if got.Language != "" || got.Script != "" {
		t.Fatalf("Language/Script = %q/%q, want empty for nil cfg", got.Language, got.Script)
	}
	if got.StoryPages != 0 || got.ChunkWords != 0 {
		t.Fatalf("StoryPages/ChunkWords = %d/%d, want 0/0 for nil cfg", got.StoryPages, got.ChunkWords)
	}
	if got.UltraRealistic == nil || *got.UltraRealistic {
		t.Fatalf("UltraRealistic = %#v, want false", got.UltraRealistic)
	}
	if got.Prompts == nil {
		t.Fatal("Prompts = nil, want non-nil empty config substitute")
	}
}

func TestRootCommandPromptModeGetsSharedComicFields(t *testing.T) {
	tmpDir := t.TempDir()
	var gotRunnerCfg *comic.RunnerConfig

	cmd := newRootCommandWithDeps(commandDeps{
		loadConfig: func(string) (*config.Config, error) {
			cfg := config.DefaultConfig()
			cfg.Comic.StoryPages = 9
			cfg.Comic.GalleryPages = 4
			cfg.Comic.PanelsPerPage = 3
			cfg.Narration.ChunkWords = 55
			return cfg, nil
		},
		newTextProvider:  func(*config.Config) (provider.TextProvider, error) { return noopProvider{}, nil },
		newImageProvider: func(*config.Config) (provider.ImageProvider, error) { return noopProvider{}, nil },
		newTTSProvider: func(*config.Config, string) (provider.TTSProvider, error) {
			t.Fatal("TTS should not be built in prompt mode")
			return nil, nil
		},
		newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner {
			gotRunnerCfg = cfg
			return &recordingRunner{}
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--prompt", "shared fields check", "--output", tmpDir})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v\n%s", err, buf.String())
	}
	if gotRunnerCfg == nil {
		t.Fatal("runner config was not captured")
	}
	if got, want := gotRunnerCfg.StoryPages, 9; got != want {
		t.Fatalf("StoryPages = %d, want %d (shared via buildRunnerConfig)", got, want)
	}
	if got, want := gotRunnerCfg.GalleryPages, 4; got != want {
		t.Fatalf("GalleryPages = %d, want %d", got, want)
	}
	if got, want := gotRunnerCfg.PanelsPerPage, 3; got != want {
		t.Fatalf("PanelsPerPage = %d, want %d", got, want)
	}
	if got, want := gotRunnerCfg.ChunkWords, 55; got != want {
		t.Fatalf("ChunkWords = %d, want %d", got, want)
	}
	if gotRunnerCfg.NarrateEnabled || gotRunnerCfg.NarratorVoice != "" {
		t.Fatalf("narration fields set in prompt mode: enabled=%t voice=%q", gotRunnerCfg.NarrateEnabled, gotRunnerCfg.NarratorVoice)
	}
}

func mustLookupFlag(t *testing.T, cmd *cobra.Command, name string) *pflag.Flag {
	t.Helper()

	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		t.Fatalf("flag %q not found", name)
	}
	return flag
}

type recordingRunner struct {
	cfg        *comic.RunnerConfig
	batchFile  string
	prompt     string
	runs       int
	promptRuns int
}

func (r *recordingRunner) Run(_ context.Context, batchFile string) error {
	r.batchFile = batchFile
	r.runs++
	return nil
}

func (r *recordingRunner) RunPrompt(_ context.Context, prompt string) error {
	r.prompt = prompt
	r.promptRuns++
	return nil
}

type noopProvider struct{}

func (noopProvider) Name() string { return "noop" }
func (noopProvider) IsAvailable() error {
	return nil
}
func (noopProvider) GenerateText(context.Context, string) (string, error) {
	return "noop", nil
}
func (noopProvider) GenerateImage(context.Context, string, string) error {
	return nil
}
func (noopProvider) GenerateAudio(context.Context, string, string) error {
	return nil
}
