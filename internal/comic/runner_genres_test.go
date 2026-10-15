package comic

import (
	"context"
	"slices"
	"testing"
)

func TestNewRunnerStoryGenres(t *testing.T) {
	for _, tt := range []struct {
		name   string
		genres []string
		theme  string
		want   []string
	}{
		{name: "configured", genres: []string{"configured genre"}, want: []string{"configured genre"}},
		{name: "nil uses defaults", want: defaultStoryGenres},
		{name: "empty uses defaults", genres: []string{}, want: defaultStoryGenres},
		{name: "theme overrides defaults", theme: "explicit theme", want: []string{"explicit theme"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prompts := &recordingPromptRenderer{}
			r := NewRunner(&RunnerConfig{
				TextProvider: fakeTextProvider{text: "История."},
				Prompts:      prompts,
				Genres:       tt.genres,
				Theme:        tt.theme,
			})
			if _, err := r.generator.Generate(context.Background(), []WordEntry{{Word: "ябълка"}}); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if tt.theme == "" && !slices.Equal(r.generator.genres, tt.want) {
				t.Fatalf("generator genres = %q, want %q", r.generator.genres, tt.want)
			}
			genre := prompts.calls[1].data["Genre"].(string)
			if !slices.Contains(tt.want, genre) {
				t.Fatalf("story genre = %q, want one of %q", genre, tt.want)
			}
		})
	}
}

func TestNewRunnerOwnsStoryGenres(t *testing.T) {
	prompts := &recordingPromptRenderer{}
	cfg := &RunnerConfig{
		TextProvider: fakeTextProvider{text: "История."},
		Prompts:      prompts,
		Genres:       []string{"original genre"},
	}
	r := NewRunner(cfg)
	cfg.Genres[0] = "changed genre"
	if _, err := r.generator.Generate(context.Background(), []WordEntry{{Word: "ябълка"}}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if genre := prompts.calls[1].data["Genre"]; genre != "original genre" {
		t.Fatalf("story genre after config mutation = %q, want original genre", genre)
	}
}
