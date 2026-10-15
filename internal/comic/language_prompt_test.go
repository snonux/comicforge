package comic

import (
	"context"
	"strings"
	"testing"
)

type languagePromptTextProvider func(context.Context, string) (string, error)

func (languagePromptTextProvider) Name() string       { return "language-prompt-text" }
func (languagePromptTextProvider) IsAvailable() error { return nil }

func (f languagePromptTextProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}

func TestLocalizedLanguageNameIndependentOfScript(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		language string
		want     string
	}{
		{"Russian", "Russian"},
		{"Ukrainian", "Ukrainian"},
		{"Bulgarian", "български"},
		{"bUlGaRiAn", "български"},
		{"English", "English"},
		{"german", "German"},
		{"french", "French"},
		{"Serbian", "Serbian"},
		{"Українська", "Українська"},
		{"", ""},
	} {
		for _, script := range []string{"Cyrillic", "cyrillic", "Latin", ""} {
			t.Run(tt.language+"/"+script, func(t *testing.T) {
				if got := localizedLanguageName(tt.language, script); got != tt.want {
					t.Errorf("localizedLanguageName(%q, %q) = %q, want %q", tt.language, script, got, tt.want)
				}
			})
		}
	}
}

func TestGeneratedPromptsRetainSelectedLanguage(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		language string
		script   string
		name     string
		text     string
	}{
		{"Russian", "Cyrillic", "Russian", "История."},
		{"Ukrainian", "Cyrillic", "Ukrainian", "Історія."},
		{"Bulgarian", "Cyrillic", "български", "История."},
		{"English", "Latin", "English", "Story."},
		{"Serbian", "Cyrillic", "Serbian", "Прича."},
	} {
		t.Run(tt.language, func(t *testing.T) {
			renderer := embeddedStylePromptRenderer{}
			checkPrompt := func(prompt, want string) {
				t.Helper()
				if !strings.Contains(prompt, want) {
					t.Errorf("prompt missing selected language instruction %q: %s", want, prompt)
				}
				if tt.language != "Bulgarian" && strings.Contains(prompt, "български") {
					t.Error("prompt replaces selected language with Bulgarian")
				}
			}
			var helperPrompt string
			textProvider := languagePromptTextProvider(func(_ context.Context, prompt string) (string, error) {
				helperPrompt = prompt
				return tt.text, nil
			})
			generator := NewGenerator(&GeneratorConfig{
				TextProvider: textProvider, Prompts: renderer, Language: tt.language, Script: tt.script,
			})
			for _, name := range []string{storyPromptTemplate, storyFullPromptTemplate} {
				prompt, err := generator.renderStoryPrompt(name, nil)
				if err != nil {
					t.Fatal(err)
				}
				checkPrompt(prompt, "All story output must be in "+tt.name+" using "+tt.script+" script only")
				checkPrompt(prompt, "Write a story of about 250 words in "+tt.name)
			}

			artist := NewArtist(&ArtistConfig{
				ImageProvider: fakeImageProvider{t: t}, TextProvider: textProvider, Prompts: renderer,
				Language: tt.language, Script: tt.script,
			})
			pages := []struct {
				name string
				data map[string]any
			}{
				{coverPromptTemplate, artist.coverPromptData(tt.text, "watercolor", tt.text)},
				{storyPagePromptTemplate, artist.storyPagePromptData(tt.text, 1, "watercolor", tt.text, nil, nil)},
				{backCoverPromptTemplate, artist.backPromptData(tt.text, "watercolor", tt.text, tt.text)},
			}
			for _, page := range pages {
				prompt, err := renderer.RenderPrompt(page.name, page.data)
				if err != nil {
					t.Fatal(err)
				}
				checkPrompt(prompt, "All visible text must use "+tt.name+" and "+tt.script+" script only")
			}
			if _, blurb, err := artist.resolveHelperTexts(context.Background(), tt.text, tt.text); err != nil || blurb != tt.text {
				t.Fatalf("resolveHelperTexts() blurb = %q, error = %v", blurb, err)
			}
			checkPrompt(helperPrompt, "sentences in "+tt.name+" using "+tt.script+" script")

			narrator := NewNarrator(&NarratorConfig{
				TextProvider: textProvider, MainProvider: fakeTTSProvider{}, Prompts: renderer,
				Language: tt.language, Script: tt.script,
			})
			for _, build := range []func(context.Context, string) string{narrator.buildIntro, narrator.buildConclusion} {
				if got := build(context.Background(), tt.text); got != tt.text {
					t.Fatalf("narration helper = %q, want %q", got, tt.text)
				}
				checkPrompt(helperPrompt, "in "+tt.name+" using "+tt.script+" script")
			}
		})
	}
}
