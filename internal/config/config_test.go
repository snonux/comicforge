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
  panels_per_page: 2
  aspect_ratio: "1:1"
  prompt_max_chars: 321
  page_max_retries: 2
  page_retry_base_seconds: 9
story:
  realistic_weight: 0.8
styles:
  comic:
    - custom comic
  realistic:
    - custom realistic
narration:
  chunk_words: 42
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
	if got, want := cfg.Comic.PanelsPerPage, 2; got != want {
		t.Fatalf("Comic.PanelsPerPage = %d, want %d", got, want)
	}
	if got, want := cfg.Comic.AspectRatio, "1:1"; got != want {
		t.Fatalf("Comic.AspectRatio = %q, want %q", got, want)
	}
	if got, want := cfg.Comic.PromptMaxChars, 321; got != want {
		t.Fatalf("Comic.PromptMaxChars = %d, want %d", got, want)
	}
	if got, want := cfg.Comic.PageMaxRetries, 2; got != want {
		t.Fatalf("Comic.PageMaxRetries = %d, want %d", got, want)
	}
	if got, want := cfg.Comic.PageRetryBaseSeconds, 9; got != want {
		t.Fatalf("Comic.PageRetryBaseSeconds = %d, want %d", got, want)
	}
	if got, want := cfg.Story.RealisticWeight, 0.8; got != want {
		t.Fatalf("Story.RealisticWeight = %v, want %v", got, want)
	}
	if got, want := len(cfg.Styles.Comic), 1; got != want || cfg.Styles.Comic[0] != "custom comic" {
		t.Fatalf("Styles.Comic = %#v, want %q", cfg.Styles.Comic, "custom comic")
	}
	if got, want := len(cfg.Styles.Realistic), 1; got != want || cfg.Styles.Realistic[0] != "custom realistic" {
		t.Fatalf("Styles.Realistic = %#v, want %q", cfg.Styles.Realistic, "custom realistic")
	}
	if got, want := cfg.Narration.ChunkWords, 42; got != want {
		t.Fatalf("Narration.ChunkWords = %d, want %d", got, want)
	}
	if got, want := cfg.PromptsDir, "./custom-prompts"; got != want {
		t.Fatalf("PromptsDir = %q, want %q", got, want)
	}
}

func TestLoadFallsBackToGoogleAPIKeyEnv(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "fallback-key")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.GoogleAPIKey(), "fallback-key"; got != want {
		t.Fatalf("GoogleAPIKey() = %q, want %q", got, want)
	}
}

func TestLoadPrefersComicForgeGoogleAPIKeyEnv(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "fallback-key")
	t.Setenv("COMICFORGE_API_GOOGLE_API_KEY", "comicforge-key")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.GoogleAPIKey(), "comicforge-key"; got != want {
		t.Fatalf("GoogleAPIKey() = %q, want %q", got, want)
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
		"Language":     "Bulgarian",
		"LanguageName": "български",
		"ScriptName":   "кирилица",
		"Genre":        "a mystery with a surprising twist",
		"Words":        "- ябълка\n- книга\n",
	})
	if err != nil {
		t.Fatalf("RenderPrompt() error = %v", err)
	}
	if !strings.Contains(got, "Write a story of about 250 words in български") {
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
		"LanguageName":            "български",
		"Script":                  "Cyrillic",
		"ScriptName":              "кирилица",
		"Genre":                   "a mystery with a surprising twist",
		"Style":                   "cinematic realism",
		"Theme":                   "mystery",
		"Prompt":                  "a robot reading a newspaper",
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
		"StoryPages":              7,
		"PanelsPerPage":           2,
		"TotalPages":              7,
		"TotalPanels":             14,
		"PanelLabelsText":         "A or B",
		"RequiredDialoguePanels":  1,
		"PanelLayout":             "MANDATORY PANEL LAYOUT — divide the image into exactly 2 distinct panels in a balanced two-panel layout.\n",
		"BlurbBox":                "правоъгълно текстово поле с кратък текст",
		"SeriesTitle":             "КомиксФордж Приключения",
		"Pose":                    "extreme close-up portrait",
		"GalleryNum":              1,
		"TotalGalleryPages":       5,
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
		"manual_prompt.md",
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
