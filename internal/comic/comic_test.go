package comic

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if !strings.Contains(got, "Откъс от историята") {
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
		"история",
		storyBibleSeparator,
		"библия",
		storyTitleSeparator,
		"Заглавие",
		storyPanelSeparator,
		"P1-A: а",
		"P1-B: б",
		"P1-C: в",
		"P1-D: г",
		"P2-A: д",
		"P2-B: е",
		"P2-C: ж",
		"P2-D: з",
		"P3-A: и",
		"P3-B: й",
		"P3-C: к",
		"P3-D: л",
		"P4-A: м",
		"P4-B: н",
		"P4-C: о",
		"P4-D: п",
		"P5-A: р",
		"P5-B: с",
		"P5-C: т",
		"P5-D: у",
	}, "\n")
	generator := NewGenerator(&GeneratorConfig{
		TextProvider: fakeTextProvider{text: text},
		Prompts:      fakePromptRenderer{},
	})
	got, err := generator.GenerateFull(context.Background(), []WordEntry{{Word: "ябълка"}})
	if err != nil {
		t.Fatalf("GenerateFull() error = %v", err)
	}
	if got.Title != "Заглавие" || got.StoryText != "история" || got.Bible != "библия" {
		t.Fatalf("GenerateFull() = %#v", got)
	}
}

func TestDrawComicPagesReturnsErrorWhenRenderFails(t *testing.T) {
	originalSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() {
		sleep = originalSleep
	})

	artist := NewArtist(&ArtistConfig{
		ImageProvider:  failingImageProvider{},
		Prompts:        fakePromptRenderer{},
		OutputDir:      t.TempDir(),
		UltraRealistic: false,
	})

	_, err := artist.DrawComicPages(context.Background(), "story", "bible", "slug", []WordEntry{{Word: "ябълка"}}, nil)
	if err == nil {
		t.Fatal("DrawComicPages() error = nil, want render failure")
	}
}

func TestArtistAndRunnerEndToEndWithFakes(t *testing.T) {
	t.Parallel()

	originalLeakValidation := validateImagePromptLeakageFn
	validateImagePromptLeakageFn = func(context.Context, string, string) error { return nil }
	t.Cleanup(func() {
		validateImagePromptLeakageFn = originalLeakValidation
	})

	tmpDir := t.TempDir()
	img := fakeImageProvider{t: t}
	genText := &scriptedTextProvider{responses: []string{
		strings.Join([]string{
			"история",
			storyBibleSeparator,
			"библия",
			storyTitleSeparator,
			"Заглавие",
			storyPanelSeparator,
			"P1-A: а", "P1-B: б", "P1-C: в", "P1-D: г",
			"P2-A: д", "P2-B: е", "P2-C: ж", "P2-D: з",
			"P3-A: и", "P3-B: й", "P3-C: к", "P3-D: л",
			"P4-A: м", "P4-B: н", "P4-C: о", "P4-D: п",
			"P5-A: р", "P5-B: с", "P5-C: т", "P5-D: у",
		}, "\n"),
		"Кратко рекламно описание.",
		"Пъстра уводна реплика за разказа.",
		"Топъл заключителен епилог за историята.",
	}}
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

func TestRunnerPropagatesRenderFailures(t *testing.T) {
	originalSleep := sleep
	sleep = func(time.Duration) {}
	originalLeakValidation := validateImagePromptLeakageFn
	validateImagePromptLeakageFn = func(context.Context, string, string) error { return nil }
	t.Cleanup(func() {
		sleep = originalSleep
		validateImagePromptLeakageFn = originalLeakValidation
	})

	runner := NewRunner(&RunnerConfig{
		TextProvider: &scriptedTextProvider{responses: []string{
			strings.Join([]string{"история", storyBibleSeparator, "библия", storyTitleSeparator, "Заглавие", storyPanelSeparator, "P1-A: а", "P1-B: б", "P1-C: в", "P1-D: г", "P2-A: д", "P2-B: е", "P2-C: ж", "P2-D: з", "P3-A: и", "P3-B: й", "P3-C: к", "P3-D: л", "P4-A: м", "P4-B: н", "P4-C: о", "P4-D: п", "P5-A: р", "P5-B: с", "P5-C: т", "P5-D: у"}, "\n"),
			"Кратко рекламно описание.",
		}},
		ImageProvider:         failingImageProvider{},
		MainTTSProvider:       fakeTTSProvider{},
		ConclusionTTSProvider: fakeTTSProvider{},
		Prompts:               fakePromptRenderer{},
		OutputDir:             t.TempDir(),
		Slug:                  "forced-slug",
		NarrateEnabled:        false,
	})
	runner.assemblePDF = func(string, string, []string) (string, error) {
		t.Fatal("assemblePDF should not be called on render failure")
		return "", nil
	}

	vocabFile := filepath.Join(t.TempDir(), "vocab.txt")
	if err := os.WriteFile(vocabFile, []byte("ябълка = apple\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runner.Run(context.Background(), vocabFile); err == nil {
		t.Fatal("Runner.Run() error = nil, want render failure")
	}
}

func TestDrawComicPagesUsesOneStyleAcrossTheWholePDF(t *testing.T) {
	originalLeakValidation := validateImagePromptLeakageFn
	validateImagePromptLeakageFn = func(context.Context, string, string) error { return nil }
	t.Cleanup(func() {
		validateImagePromptLeakageFn = originalLeakValidation
	})

	renderer := &recordingPromptRenderer{}
	artist := NewArtist(&ArtistConfig{
		ImageProvider:  fakeImageProvider{t: t},
		Prompts:        renderer,
		OutputDir:      t.TempDir(),
		UltraRealistic: false,
	})

	if _, err := artist.DrawComicPages(context.Background(), "история", "библия", "slug", []WordEntry{{Word: "ябълка"}}, nil); err != nil {
		t.Fatalf("DrawComicPages() error = %v", err)
	}

	var style string
	var promptCount int
	for _, call := range renderer.calls {
		if !isImagePromptTemplate(call.name) {
			continue
		}
		promptCount++
		gotStyle, ok := call.data["Style"].(string)
		if !ok || gotStyle == "" {
			t.Fatalf("image prompt %q missing style: %#v", call.name, call.data)
		}
		if style == "" {
			style = gotStyle
			continue
		}
		if gotStyle != style {
			t.Fatalf("image prompt styles diverged: first=%q later=%q in %q", style, gotStyle, call.name)
		}
	}
	if promptCount == 0 {
		t.Fatal("no image prompts recorded")
	}
}

func TestConvertToStereoFallsBackToCopyWhenFFmpegMissing(t *testing.T) {
	originalLookPath := lookPath
	lookPath = func(string) (string, error) {
		return "", errors.New("missing ffmpeg")
	}
	t.Cleanup(func() {
		lookPath = originalLookPath
	})

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.mp3")
	outputFile := filepath.Join(tmpDir, "nested", "output.mp3")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputFile, []byte("audio-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := convertToStereo(inputFile, outputFile); err != nil {
		t.Fatalf("convertToStereo() error = %v", err)
	}
	got, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "audio-bytes" {
		t.Fatalf("convertToStereo() copied %q, want %q", got, "audio-bytes")
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

type recordingPromptRenderer struct {
	calls []recordedPromptCall
}

type recordedPromptCall struct {
	name string
	data map[string]any
}

func (r *recordingPromptRenderer) RenderPrompt(name string, data any) (string, error) {
	if m, ok := data.(map[string]any); ok {
		copyData := make(map[string]any, len(m))
		for k, v := range m {
			copyData[k] = v
		}
		r.calls = append(r.calls, recordedPromptCall{name: name, data: copyData})
	} else {
		r.calls = append(r.calls, recordedPromptCall{name: name})
	}
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

func isImagePromptTemplate(name string) bool {
	switch name {
	case coverPromptTemplate, storyPagePromptTemplate, galleryPagePromptTemplate, backCoverPromptTemplate:
		return true
	default:
		return false
	}
}

type fakeTextProvider struct{ text string }

func (f fakeTextProvider) Name() string       { return "fake-text" }
func (f fakeTextProvider) IsAvailable() error { return nil }
func (f fakeTextProvider) GenerateText(_ context.Context, _ string) (string, error) {
	return f.text, nil
}

type scriptedTextProvider struct {
	responses []string
	index     int
}

func (p *scriptedTextProvider) Name() string       { return "scripted-text" }
func (p *scriptedTextProvider) IsAvailable() error { return nil }
func (p *scriptedTextProvider) GenerateText(_ context.Context, _ string) (string, error) {
	if len(p.responses) == 0 {
		return "", nil
	}
	if p.index >= len(p.responses) {
		return p.responses[len(p.responses)-1], nil
	}
	out := p.responses[p.index]
	p.index++
	return out, nil
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

type failingImageProvider struct{}

func (failingImageProvider) Name() string       { return "failing-image" }
func (failingImageProvider) IsAvailable() error { return nil }
func (failingImageProvider) GenerateImage(_ context.Context, _ string, _ string) error {
	return errors.New("image render failed")
}

type fakeTTSProvider struct{}

func (fakeTTSProvider) Name() string       { return "fake-tts" }
func (fakeTTSProvider) IsAvailable() error { return nil }
func (fakeTTSProvider) GenerateAudio(_ context.Context, _ string, outputFile string) error {
	return os.WriteFile(outputFile, []byte("mp3"), 0o644)
}
