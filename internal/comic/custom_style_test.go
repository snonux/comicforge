package comic

import (
	"context"
	"strings"
	"testing"
	"text/template"

	"github.com/snonux/comicforge/prompts"
	"gopkg.in/yaml.v3"
)

type embeddedStylePromptRenderer struct{}

type customStyleImageProvider struct {
	fakeImageProvider
	prompts []string
}

func TestRunPromptCustomStylesInDefaultLanguage(t *testing.T) {
	t.Parallel()

	for _, style := range []string{"watercolor", "акварел с меки линии"} {
		t.Run(style, func(t *testing.T) {
			images := &customStyleImageProvider{fakeImageProvider: fakeImageProvider{t: t}}
			runner := NewRunner(&RunnerConfig{
				ImageProvider: images,
				Prompts:       embeddedStylePromptRenderer{},
				OutputDir:     t.TempDir(),
				Slug:          "custom-style",
				Style:         style,
			})
			stubRunnerArtistSeams(runner)
			if err := runner.RunPrompt(context.Background(), "котка до прозореца"); err != nil {
				t.Fatalf("RunPrompt() error = %v", err)
			}
			if len(images.prompts) != 1 {
				t.Fatalf("image calls = %d, want 1", len(images.prompts))
			}
			for _, want := range []string{"Visual style: " + style, "No text, captions, logos"} {
				if !strings.Contains(images.prompts[0], want) {
					t.Errorf("image prompt missing %q", want)
				}
			}
		})
	}
}

func TestDrawComicPagesCustomStylesInDefaultLanguage(t *testing.T) {
	t.Parallel()

	var configured struct {
		Styles map[string][]string `yaml:"styles"`
	}
	if err := yaml.Unmarshal([]byte("styles:\n  comic:\n    - custom charcoal sketch\n  soft_watercolor:\n    - акварел с меки линии\n"), &configured); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		style string
		mode  string
		want  string
	}{
		{name: "English override", style: "watercolor", want: "watercolor"},
		{name: "Cyrillic override", style: "акварел с меки линии", want: "акварел с меки линии"},
		{name: "edited YAML family", mode: "comic", want: "custom charcoal sketch"},
		{name: "custom YAML category", mode: "soft_watercolor", want: "акварел с меки линии"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := &customStyleImageProvider{fakeImageProvider: fakeImageProvider{t: t}}
			artist := NewArtist(&ArtistConfig{
				ImageProvider: images,
				Prompts:       embeddedStylePromptRenderer{},
				OutputDir:     t.TempDir(),
				Style:         tt.style,
				StyleMode:     tt.mode,
				Styles:        configured.Styles,
				StoryPages:    1,
				GalleryPages:  1,
			})
			stubArtistSeams(artist)
			paths, err := artist.DrawComicPages(context.Background(), "Котка до прозореца.", "Котката е сива.", "custom-style", nil, nil)
			if err != nil {
				t.Fatalf("DrawComicPages() error = %v", err)
			}
			if len(paths) != 4 || len(images.prompts) != 4 {
				t.Fatalf("paths = %d, image calls = %d, want cover, story, gallery, back", len(paths), len(images.prompts))
			}
			for i, prompt := range images.prompts {
				if !strings.Contains(prompt, "Style: "+tt.want+".") {
					t.Errorf("page %d prompt missing custom style %q", i, tt.want)
				}
				want := "Cyrillic script only"
				if i == 2 {
					want = "No text of any kind"
				}
				if !strings.Contains(prompt, want) {
					t.Errorf("page %d prompt missing visible-text restriction %q", i, want)
				}
			}
		})
	}
}

func (embeddedStylePromptRenderer) RenderPrompt(name string, data any) (string, error) {
	content, err := prompts.Read(name)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return "", err
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (p *customStyleImageProvider) GenerateImage(ctx context.Context, prompt, outputFile string) error {
	p.prompts = append(p.prompts, prompt)
	return p.fakeImageProvider.GenerateImage(ctx, prompt, outputFile)
}
