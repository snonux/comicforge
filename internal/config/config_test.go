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

func TestRenderPromptFallsBackToEmbeddedTemplate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PromptsDir = t.TempDir()

	got, err := cfg.RenderPrompt("story_prompt.md", map[string]any{
		"Language": "Bulgarian",
		"Genre":    "a mystery with a surprising twist",
		"Words":    "- ябълка\n- книга\n",
	})
	if err != nil {
		t.Fatalf("RenderPrompt() error = %v", err)
	}
	if !strings.Contains(got, "Write a ~250-word story in Bulgarian") {
		t.Fatalf("RenderPrompt() = %q, want embedded story prompt", got)
	}
}

func TestRenderPromptPrefersExternalTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.PromptsDir = tmpDir

	if err := os.WriteFile(filepath.Join(tmpDir, "story_prompt.md"), []byte("external {{.Word}}"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	got, err := cfg.RenderPrompt("story_prompt.md", map[string]any{
		"Word": "override",
	})
	if err != nil {
		t.Fatalf("RenderPrompt() error = %v", err)
	}
	if got != "external override" {
		t.Fatalf("RenderPrompt() = %q, want %q", got, "external override")
	}
}

func TestRenderPromptMissingKeyReturnsError(t *testing.T) {
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

func TestEmbeddedPromptTemplatesRender(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PromptsDir = t.TempDir()

	data := map[string]any{
		"Language":                "Bulgarian",
		"Script":                  "Cyrillic",
		"Genre":                   "a mystery with a surprising twist",
		"Style":                   "cinematic realism",
		"Words":                   "- ябълка\n- книга\n",
		"Bible":                   "Мира: млада жена, кафява коса, сини очи, червено палто.\n",
		"Subtitle":                "КомиксФордж Приключения",
		"StoryText":               "A short story teaser.",
		"RenderingRequirement":    "ULTRA-REALISTIC RENDERING",
		"RenderingRequirementEnd": "FINAL LOCK — PHOTOREALISM",
		"StoryBibleSeparator":     "---CHARACTER GUIDE---",
		"StoryTitleSeparator":     "---COMIC TITLE---",
		"StoryPanelSeparator":     "---PANEL SCRIPT---",
		"PageNum":                 1,
		"TotalPages":              5,
		"PanelLayout":             "MANDATORY PANEL LAYOUT — divide the image into exactly 4 panels in a 2×2 grid.\n",
		"BlurbBox":                "правоъгълно текстово поле с кратък текст",
		"SeriesTitle":             "КомиксФордж Приключения",
		"Pose":                    "extreme close-up portrait",
	}

	for _, name := range []string{
		"story_system.md",
		"story_prompt.md",
		"story_full_prompt.md",
		"bible_system.md",
		"blurb_system.md",
		"cover_prompt.md",
		"story_page_prompt.md",
		"gallery_page_prompt.md",
		"back_cover_prompt.md",
		"panel_script_prompt.md",
		"narrator_cinematic.md",
		"narrator_intro_system.md",
		"narrator_conclusion_system.md",
		"rendering_requirement.md",
		"rendering_requirement_end.md",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := cfg.RenderPrompt(name, data)
			if err != nil {
				t.Fatalf("RenderPrompt() error = %v", err)
			}
			if strings.TrimSpace(got) == "" {
				t.Fatal("RenderPrompt() = empty string, want rendered prompt")
			}
		})
	}
}

func TestRenderPromptMissingTemplateReturnsError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PromptsDir = t.TempDir()

	_, err := cfg.RenderPrompt("missing_prompt.md", map[string]any{})
	if err == nil {
		t.Fatal("RenderPrompt() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "read embedded prompt") {
		t.Fatalf("RenderPrompt() error = %v, want embedded read error", err)
	}
}

func TestLoadRejectsUnknownProvider(t *testing.T) {
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
