package comic

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	if got, want := slugify("  The Clockwork Dragon! "), "the-clockwork-dragon"; got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
	if got, want := slugify("!!!"), "comic"; got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
}

func TestParseGenerateResult(t *testing.T) {
	t.Parallel()

	combined := strings.Join([]string{
		"story text",
		storyBibleSeparator,
		"bible text",
		storyTitleSeparator,
		"My Comic",
		storyPanelSeparator,
		"P1-A: first",
		"P1-B: second",
	}, "\n")
	got := parseGenerateResult(combined)
	if got.StoryText != "story text" || got.Bible != "bible text" || got.Title != "My Comic" {
		t.Fatalf("parseGenerateResult() = %#v", got)
	}
	if got.PanelScript[0][0] != "first" || got.PanelScript[0][1] != "second" {
		t.Fatalf("parseGenerateResult() panel script = %#v", got.PanelScript)
	}
}

func TestBuildPanelLayoutUsesFallbackExcerpt(t *testing.T) {
	t.Parallel()

	got := buildPanelLayout("one two three four five", nil)
	if !strings.Contains(got, "Story excerpt") {
		t.Fatalf("buildPanelLayout() = %q", got)
	}
}

func TestCopyGalleryPNGsToComicsGallery(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	comicDir := filepath.Join(root, "comics", "my-slug")
	if err := os.MkdirAll(comicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comicDir, "my-slug_gallery_1.png"), []byte("png1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyGalleryPNGsToComicsGallery(root, comicDir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "comics", "gallery", "my-slug_gallery_1.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "png1" {
		t.Fatalf("copied file = %q", b)
	}
}

func TestGeneratorGenerateFull(t *testing.T) {
	t.Parallel()

	text := strings.Join([]string{
		"story",
		storyBibleSeparator,
		"bible",
		storyTitleSeparator,
		"Title",
		storyPanelSeparator,
		"P1-A: a",
		"P1-B: b",
		"P1-C: c",
		"P1-D: d",
		"P2-A: e",
		"P2-B: f",
		"P2-C: g",
		"P2-D: h",
		"P3-A: i",
		"P3-B: j",
		"P3-C: k",
		"P3-D: l",
		"P4-A: m",
		"P4-B: n",
		"P4-C: o",
		"P4-D: p",
		"P5-A: q",
		"P5-B: r",
		"P5-C: s",
		"P5-D: t",
	}, "\n")
	generator := NewGenerator(&GeneratorConfig{
		TextProvider: fakeTextProvider{text: text},
		Prompts:      fakePromptRenderer{},
	})
	got, err := generator.GenerateFull(context.Background(), []WordEntry{{Word: "ябълка"}})
	if err != nil {
		t.Fatalf("GenerateFull() error = %v", err)
	}
	if got.Title != "Title" || got.StoryText != "story" || got.Bible != "bible" {
		t.Fatalf("GenerateFull() = %#v", got)
	}
}

func TestArtistAndRunnerEndToEndWithFakes(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	img := fakeImageProvider{t: t}
	genText := fakeTextProvider{text: strings.Join([]string{
		"story",
		storyBibleSeparator,
		"bible",
		storyTitleSeparator,
		"Title",
		storyPanelSeparator,
		"P1-A: a", "P1-B: b", "P1-C: c", "P1-D: d",
		"P2-A: e", "P2-B: f", "P2-C: g", "P2-D: h",
		"P3-A: i", "P3-B: j", "P3-C: k", "P3-D: l",
		"P4-A: m", "P4-B: n", "P4-C: o", "P4-D: p",
		"P5-A: q", "P5-B: r", "P5-C: s", "P5-D: t",
	}, "\n")}
	narr := fakeTTSProvider{}
	runner := NewRunner(&RunnerConfig{
		TextProvider:          genText,
		ImageProvider:         img,
		MainTTSProvider:       narr,
		ConclusionTTSProvider: narr,
		Prompts:               fakePromptRenderer{},
		OutputDir:             tmpDir,
		Slug:                  "forced-slug",
		NarrateEnabled:        true,
	})
	runner.assemblePDF = func(outputDir, titleSlug string, imagePaths []string) (string, error) {
		path := filepath.Join(outputDir, titleSlug+".pdf")
		return path, os.WriteFile(path, []byte("pdf"), 0o644)
	}
	if err := runner.Run(context.Background(), filepath.Join(tmpDir, "vocab.txt")); err == nil {
		t.Fatal("expected vocab read error for missing file")
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "vocab.txt"), []byte("ябълка = apple\nкнига = book\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runner.Run(context.Background(), filepath.Join(tmpDir, "vocab.txt")); err != nil {
		t.Fatalf("Runner.Run() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "forced-slug", "forced-slug.pdf")); err != nil {
		t.Fatalf("pdf missing: %v", err)
	}
}

type fakePromptRenderer struct{}

func (fakePromptRenderer) RenderPrompt(name string, data any) (string, error) {
	switch name {
	case storySystemPromptTemplate:
		return "system prompt", nil
	case storyPromptTemplate, storyFullPromptTemplate:
		return "prompt", nil
	case coverPromptTemplate, storyPagePromptTemplate, galleryPagePromptTemplate, backCoverPromptTemplate:
		return "image prompt", nil
	case blurbSystemTemplate, introSystemTemplate, conclusionSystemTemplate:
		return "teaser prompt", nil
	default:
		return "", errors.New("unexpected template")
	}
}

type fakeTextProvider struct{ text string }

func (f fakeTextProvider) Name() string       { return "fake-text" }
func (f fakeTextProvider) IsAvailable() error { return nil }
func (f fakeTextProvider) GenerateText(_ context.Context, _ string) (string, error) {
	return f.text, nil
}

type fakeImageProvider struct{ t *testing.T }

func (f fakeImageProvider) Name() string       { return "fake-image" }
func (f fakeImageProvider) IsAvailable() error { return nil }
func (f fakeImageProvider) GenerateImage(_ context.Context, _ string, outputFile string) error {
	if err := os.WriteFile(outputFile, []byte("png"), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return nil
}

type fakeTTSProvider struct{}

func (fakeTTSProvider) Name() string       { return "fake-tts" }
func (fakeTTSProvider) IsAvailable() error { return nil }
func (fakeTTSProvider) GenerateAudio(_ context.Context, _ string, outputFile string) error {
	return os.WriteFile(outputFile, []byte("mp3"), 0o644)
}
