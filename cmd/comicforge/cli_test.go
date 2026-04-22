package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	version "codeberg.org/snonux/comicforge/internal"
	"codeberg.org/snonux/comicforge/internal/comic"
	"codeberg.org/snonux/comicforge/internal/config"
	"codeberg.org/snonux/comicforge/internal/provider"
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
		loadConfig: config.Load,
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
