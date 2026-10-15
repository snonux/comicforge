package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/snonux/comicforge/internal/comic"
	"github.com/snonux/comicforge/internal/config"
	"github.com/snonux/comicforge/internal/provider"
)

type genrePromptCapture struct {
	noopProvider
	prompt string
	calls  int
	err    error
}

func (p *genrePromptCapture) GenerateText(_ context.Context, prompt string) (string, error) {
	p.prompt = prompt
	p.calls++
	return "", p.err
}

func TestRootCommandStoryGenrePrompt(t *testing.T) {
	const configuredGenre = "UNIQUE_CONFIGURED_GENRE"
	const themeOverride = "UNIQUE_THEME_OVERRIDE"
	for _, tt := range []struct {
		name      string
		storyYAML string
		theme     string
		want      []string
		absent    string
	}{
		{name: "configured", storyYAML: "story:\n  genres: [" + configuredGenre + "]\n", want: []string{configuredGenre}},
		{name: "theme override", storyYAML: "story:\n  genres: [" + configuredGenre + "]\n", theme: themeOverride, want: []string{themeOverride}, absent: configuredGenre},
		{name: "configuration defaults", want: config.DefaultConfig().Story.Genres, absent: configuredGenre},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte("language:\n  story_language: English\n  script: Latin\n"+tt.storyYAML), 0o600); err != nil {
				t.Fatal(err)
			}
			vocabPath := filepath.Join(tmpDir, "vocab.txt")
			if err := os.WriteFile(vocabPath, []byte("apple = fruit\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			stop := errors.New("stop after capturing story prompt")
			capture := &genrePromptCapture{err: stop}
			cmd := newRootCommandWithDeps(commandDeps{
				loadConfig: func(path string) (*config.Config, error) { return config.Load(path) },
				newTextProvider: func(*config.Config) (provider.TextProvider, error) {
					return capture, nil
				},
				newImageProvider: func(*config.Config) (provider.ImageProvider, error) {
					return noopProvider{}, nil
				},
				newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner { return comic.NewRunner(cfg) },
			})
			cmd.SetArgs([]string{"--config", configPath, "--vocab", vocabPath, "--output", tmpDir, "--prompts-dir", filepath.Join(tmpDir, "embedded-prompts"), "--theme", tt.theme})
			if err := cmd.ExecuteContext(context.Background()); !errors.Is(err, stop) {
				t.Fatalf("ExecuteContext() error = %v, want capture stop", err)
			}
			if capture.calls != 1 {
				t.Fatalf("text calls = %d, want one story prompt", capture.calls)
			}
			if !slices.ContainsFunc(tt.want, func(genre string) bool { return strings.Contains(capture.prompt, genre) }) {
				t.Fatalf("story prompt does not contain any expected genre %q:\n%s", tt.want, capture.prompt)
			}
			if tt.absent != "" && strings.Contains(capture.prompt, tt.absent) {
				t.Fatalf("story prompt contains overridden or unconfigured genre %q", tt.absent)
			}
		})
	}
}

func TestBuildRunnerConfigCopiesStoryGenres(t *testing.T) {
	for _, tt := range []struct {
		name string
		cfg  *config.Config
	}{
		{name: "defaults", cfg: config.DefaultConfig()},
		{name: "configured", cfg: &config.Config{Story: config.StoryConfig{Genres: []string{"original", "second"}}}},
		{name: "empty", cfg: &config.Config{Story: config.StoryConfig{Genres: []string{}}}},
		{name: "nil genres", cfg: &config.Config{}},
		{name: "nil config"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRunnerConfig(tt.cfg, cliFlags{}, noopProvider{}, noopProvider{})
			var want []string
			if tt.cfg != nil {
				want = slices.Clone(tt.cfg.Story.Genres)
			}
			if !slices.Equal(got.Genres, want) {
				t.Fatalf("Genres = %q, want %q", got.Genres, want)
			}
			if len(want) == 0 {
				return
			}
			tt.cfg.Story.Genres[0] = "changed source"
			if !slices.Equal(got.Genres, want) {
				t.Fatal("mutating source config changed runner genres")
			}
			got.Genres[0] = "changed runner"
			if tt.cfg.Story.Genres[0] != "changed source" {
				t.Fatal("mutating runner genres changed source config")
			}
		})
	}
}
