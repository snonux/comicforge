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

	got := buildPanelLayout("one two three four five", nil, 2)
	if !strings.Contains(got, "exactly 2 distinct panels") {
		t.Fatalf("buildPanelLayout() = %q, want 2-panel layout instruction", got)
	}
	if !strings.Contains(got, "Story excerpt") {
		t.Fatalf("buildPanelLayout() = %q", got)
	}
}

func TestParseGenerateResultUsesConfiguredDimensions(t *testing.T) {
	t.Parallel()

	lines := []string{
		"story text",
		storyBibleSeparator,
		"bible text",
		storyTitleSeparator,
		"My Comic",
		storyPanelSeparator,
		"P1-A: first",
		"P1-B: second",
		"P10-A: last",
		"P10-B: end",
		"P10-C: ignored",
	}
	got := parseGenerateResultWithDimensions(strings.Join(lines, "\n"), 10, 2)
	if got.StoryText != "story text" || got.Bible != "bible text" || got.Title != "My Comic" {
		t.Fatalf("parseGenerateResultWithDimensions() = %#v", got)
	}
	if got.PanelScript == nil || len(got.PanelScript) != 10 {
		t.Fatalf("panel script pages = %d, want 10", len(got.PanelScript))
	}
	for i, page := range got.PanelScript {
		if len(page) != 2 {
			t.Fatalf("page %d panels = %d, want 2", i+1, len(page))
		}
	}
	if got.PanelScript[0][0] != "first" || got.PanelScript[0][1] != "second" {
		t.Fatalf("first page panel script = %#v", got.PanelScript[0])
	}
	if got.PanelScript[9][0] != "last" || got.PanelScript[9][1] != "end" {
		t.Fatalf("last page panel script = %#v", got.PanelScript[9])
	}
}

func TestPanelLabelSupportsMultiLetterSequence(t *testing.T) {
	t.Parallel()

	if got, want := panelLabel(0), "A"; got != want {
		t.Fatalf("panelLabel(0) = %q, want %q", got, want)
	}
	if got, want := panelLabel(25), "Z"; got != want {
		t.Fatalf("panelLabel(25) = %q, want %q", got, want)
	}
	if got, want := panelLabel(26), "AA"; got != want {
		t.Fatalf("panelLabel(26) = %q, want %q", got, want)
	}
	if got, want := panelLabel(27), "AB"; got != want {
		t.Fatalf("panelLabel(27) = %q, want %q", got, want)
	}
	if got := panelLabelsText(28); !strings.Contains(got, "AA") || !strings.Contains(got, "AB") {
		t.Fatalf("panelLabelsText(28) = %q, want multi-letter labels", got)
	}
}

func TestComicOutputDirDoesNotDuplicateComicsSegment(t *testing.T) {
	t.Parallel()

	if got, want := comicOutputDir("comics", "slug"), filepath.Join("comics", "slug"); got != want {
		t.Fatalf("comicOutputDir() = %q, want %q", got, want)
	}
	if got, want := comicOutputDir(".", "slug"), filepath.Join(".", "comics", "slug"); got != want {
		t.Fatalf("comicOutputDir() = %q, want %q", got, want)
	}
	if got, want := comicOutputDir("/tmp/out", "slug"), filepath.Join("/tmp/out", "comics", "slug"); got != want {
		t.Fatalf("comicOutputDir() = %q, want %q", got, want)
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

func TestGeneratorGenerateFullUsesConfiguredDimensions(t *testing.T) {
	t.Parallel()

	renderer := &recordingPromptRenderer{}
	text := strings.Join([]string{
		"история",
		storyBibleSeparator,
		"библия",
		storyTitleSeparator,
		"Заглавие",
		storyPanelSeparator,
		"P1-A: а",
		"P1-B: б",
		"P2-A: в",
		"P2-B: г",
		"P3-A: д",
		"P3-B: е",
		"P4-A: ж",
		"P4-B: з",
		"P5-A: и",
		"P5-B: й",
		"P6-A: к",
		"P6-B: л",
		"P7-A: м",
		"P7-B: н",
	}, "\n")

	generator := NewGenerator(&GeneratorConfig{
		TextProvider:  fakeTextProvider{text: text},
		Prompts:       renderer,
		StoryPages:    7,
		PanelsPerPage: 2,
	})
	got, err := generator.GenerateFull(context.Background(), []WordEntry{{Word: "ябълка"}})
	if err != nil {
		t.Fatalf("GenerateFull() error = %v", err)
	}
	if len(got.PanelScript) != 7 {
		t.Fatalf("panel script pages = %d, want 7", len(got.PanelScript))
	}
	for i, page := range got.PanelScript {
		if len(page) != 2 {
			t.Fatalf("page %d panels = %d, want 2", i+1, len(page))
		}
	}
	var promptData map[string]any
	for _, call := range renderer.calls {
		if call.name == storyFullPromptTemplate {
			promptData = call.data
			break
		}
	}
	if promptData == nil {
		t.Fatal("story full prompt render was not recorded")
	}
	if got, want := promptData["StoryPages"], 7; got != want {
		t.Fatalf("StoryPages prompt data = %#v, want %d", got, want)
	}
	if got, want := promptData["PanelsPerPage"], 2; got != want {
		t.Fatalf("PanelsPerPage prompt data = %#v, want %d", got, want)
	}
	if got, want := promptData["TotalPanels"], 14; got != want {
		t.Fatalf("TotalPanels prompt data = %#v, want %d", got, want)
	}
	if got, want := promptData["PanelLabelsText"], "A or B"; got != want {
		t.Fatalf("PanelLabelsText prompt data = %#v, want %q", got, want)
	}
	if got, want := promptData["RequiredDialoguePanels"], 1; got != want {
		t.Fatalf("RequiredDialoguePanels prompt data = %#v, want %d", got, want)
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
	validateImagePromptLeakageFn = func(context.Context, string, string, string) error { return nil }
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
	validateImagePromptLeakageFn = func(context.Context, string, string, string) error { return nil }
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
	validateImagePromptLeakageFn = func(context.Context, string, string, string) error { return nil }
	t.Cleanup(func() {
		validateImagePromptLeakageFn = originalLeakValidation
	})

	renderer := &recordingPromptRenderer{}
	artist := NewArtist(&ArtistConfig{
		ImageProvider:  fakeImageProvider{t: t},
		Prompts:        renderer,
		OutputDir:      t.TempDir(),
		UltraRealistic: false,
		PanelsPerPage:  2,
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
	var pagePrompt map[string]any
	for _, call := range renderer.calls {
		if call.name == storyPagePromptTemplate {
			pagePrompt = call.data
			break
		}
	}
	if pagePrompt == nil {
		t.Fatal("story page prompt was not recorded")
	}
	if got, want := pagePrompt["PanelsPerPage"], 2; got != want {
		t.Fatalf("PanelsPerPage prompt data = %#v, want %d", got, want)
	}
	if got, want := pagePrompt["RequiredDialoguePanels"], 1; got != want {
		t.Fatalf("RequiredDialoguePanels prompt data = %#v, want %d", got, want)
	}
	if got, want := pagePrompt["PanelLabelsText"], "A or B"; got != want {
		t.Fatalf("PanelLabelsText prompt data = %#v, want %q", got, want)
	}
	if layout, ok := pagePrompt["PanelLayout"].(string); !ok || !strings.Contains(layout, "exactly 2 distinct panels") {
		t.Fatalf("PanelLayout prompt data = %#v, want 2-panel layout", pagePrompt["PanelLayout"])
	}
}

func TestDrawComicPagesChainsReferenceImages(t *testing.T) {
	originalLeakValidation := validateImagePromptLeakageFn
	validateImagePromptLeakageFn = func(context.Context, string, string, string) error { return nil }
	t.Cleanup(func() {
		validateImagePromptLeakageFn = originalLeakValidation
	})

	provider := &refTrackingImageProvider{t: t}
	artist := NewArtist(&ArtistConfig{
		ImageProvider:  provider,
		Prompts:        fakePromptRenderer{},
		OutputDir:      t.TempDir(),
		UltraRealistic: false,
	})

	if _, err := artist.DrawComicPages(context.Background(), "история", "библия", "slug", []WordEntry{{Word: "ябълка"}}, nil); err != nil {
		t.Fatalf("DrawComicPages() error = %v", err)
	}
	if len(provider.referenceCounts) < 3 {
		t.Fatalf("referenceCounts = %v, want multiple page generations", provider.referenceCounts)
	}
	if provider.referenceCounts[0] != 0 {
		t.Fatalf("cover refs = %d, want 0", provider.referenceCounts[0])
	}
	if provider.referenceCounts[1] != 1 {
		t.Fatalf("first story page refs = %d, want 1", provider.referenceCounts[1])
	}
	if provider.referenceCounts[2] < 1 {
		t.Fatalf("second page refs = %d, want chained references", provider.referenceCounts[2])
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

func TestNarrateConclusionDoesNotAcceptPartialConcatFallback(t *testing.T) {
	originalLookPath := lookPath
	lookPath = func(string) (string, error) {
		return "", errors.New("missing ffmpeg")
	}
	t.Cleanup(func() {
		lookPath = originalLookPath
	})

	provider := &recordingTTSProvider{}
	n := NewNarrator(&NarratorConfig{
		TextProvider:       fakeTextProvider{text: strings.Join([]string{strings.Repeat("алфа ", 60), strings.Repeat("бета ", 60)}, "\n\n")},
		MainProvider:       fakeTTSProvider{},
		ConclusionProvider: provider,
		Prompts:            fakePromptRenderer{},
	})

	path, ok := n.narrateConclusion(context.Background(), "story", t.TempDir())
	if ok {
		t.Fatalf("narrateConclusion() ok = true, want false")
	}
	if path != "" {
		t.Fatalf("narrateConclusion() path = %q, want empty", path)
	}
	if got, want := len(provider.calls), 2; got != want {
		t.Fatalf("conclusion provider calls = %d, want %d", got, want)
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

type refTrackingImageProvider struct {
	t               *testing.T
	referenceCounts []int
}

func (p *refTrackingImageProvider) Name() string       { return "ref-tracking-image" }
func (p *refTrackingImageProvider) IsAvailable() error { return nil }
func (p *refTrackingImageProvider) GenerateImage(ctx context.Context, prompt, outputFile string) error {
	return p.GenerateImageWithReferences(ctx, prompt, outputFile, nil)
}
func (p *refTrackingImageProvider) GenerateImageWithReferences(_ context.Context, _ string, outputFile string, refs [][]byte) error {
	p.referenceCounts = append(p.referenceCounts, len(refs))
	if err := os.WriteFile(outputFile, []byte("png"), 0o644); err != nil {
		p.t.Fatal(err)
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

type recordingTTSProvider struct {
	calls []string
}

func (p *recordingTTSProvider) Name() string       { return "recording-tts" }
func (p *recordingTTSProvider) IsAvailable() error { return nil }
func (p *recordingTTSProvider) GenerateAudio(_ context.Context, text, outputFile string) error {
	p.calls = append(p.calls, text)
	return os.WriteFile(outputFile, []byte("mp3"), 0o644)
}
