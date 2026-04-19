package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/snonux/comicforge/internal/provider"
)

func TestLoadReturnsDefaultsWhenConfigMissing(t *testing.T) {
	t.Parallel()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.Provider.Text, provider.Gemini; got != want {
		t.Fatalf("Provider.Text = %q, want %q", got, want)
	}
	if got, want := cfg.PromptsDir, DefaultPromptsDir; got != want {
		t.Fatalf("PromptsDir = %q, want %q", got, want)
	}
}

func TestLoadReadsFileAndEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(`
provider:
  text: openai
  image: openai
  tts: openai
prompts_dir: ./custom-prompts
comic:
  story_pages: 7
`)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("COMICFORGE_PROVIDER_TEXT", "gemini")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.Provider.Text, provider.Gemini; got != want {
		t.Fatalf("Provider.Text = %q, want %q", got, want)
	}
	if got, want := cfg.Provider.Image, provider.OpenAI; got != want {
		t.Fatalf("Provider.Image = %q, want %q", got, want)
	}
	if got, want := cfg.Comic.StoryPages, 7; got != want {
		t.Fatalf("Comic.StoryPages = %d, want %d", got, want)
	}
	if got, want := cfg.PromptsDir, "./custom-prompts"; got != want {
		t.Fatalf("PromptsDir = %q, want %q", got, want)
	}
}

func TestRenderPrompt(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.PromptsDir = tmpDir

	if err := os.WriteFile(filepath.Join(tmpDir, "story.md"), []byte("{{.Word}} -> {{.Translation}}"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	got, err := cfg.RenderPrompt("story.md", map[string]string{
		"Word":        "ябълка",
		"Translation": "apple",
	})
	if err != nil {
		t.Fatalf("RenderPrompt() error = %v", err)
	}
	if got != "ябълка -> apple" {
		t.Fatalf("RenderPrompt() = %q, want %q", got, "ябълка -> apple")
	}
}

func TestRenderPromptMissingKeyReturnsError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.PromptsDir = tmpDir

	if err := os.WriteFile(filepath.Join(tmpDir, "story.md"), []byte("{{.Word}} -> {{.Translation}}"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	_, err := cfg.RenderPrompt("story.md", map[string]string{"Word": "ябълка"})
	if err == nil {
		t.Fatal("RenderPrompt() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "render prompt") {
		t.Fatalf("RenderPrompt() error = %v, want wrapped render error", err)
	}
}

func TestLoadRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(`
provider:
  text: mystery
`)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown text provider") {
		t.Fatalf("Load() error = %v, want unknown provider error", err)
	}
}

func TestHomeDirReturnsFallbackWhenResolutionFails(t *testing.T) {
	t.Parallel()

	oldUserHomeDir := userHomeDir
	t.Cleanup(func() {
		userHomeDir = oldUserHomeDir
	})

	userHomeDir = func() (string, error) {
		return "", errors.New("boom")
	}

	homeDir, err := HomeDir()
	if err == nil {
		t.Fatal("HomeDir() error = nil, want error")
	}
	if homeDir != "." {
		t.Fatalf("HomeDir() homeDir = %q, want %q", homeDir, ".")
	}
}
