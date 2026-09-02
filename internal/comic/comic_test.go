package comic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
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

func TestShortPromptSlugFromText(t *testing.T) {
	t.Parallel()

	longPrompt := "generate a logo for ComicForge which is a comic book generator displaying a superhero fighting a bad guy and speech bubbles"

	got := shortPromptSlugFromText(longPrompt)
	if got == "" {
		t.Fatal("shortPromptSlugFromText() returned empty slug")
	}
	if len(got) > maxPromptSlugLength {
		t.Fatalf("shortPromptSlugFromText() length = %d, want <= %d (%q)", len(got), maxPromptSlugLength, got)
	}
	if strings.Contains(got, " ") {
		t.Fatalf("shortPromptSlugFromText() = %q, want path-safe slug", got)
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

	got := buildPanelLayout("one two three four five", nil, 2, 900)
	if !strings.Contains(got, "exactly 2 distinct panels") {
		t.Fatalf("buildPanelLayout() = %q, want 2-panel layout instruction", got)
	}
	if !strings.Contains(got, "Story excerpt") {
		t.Fatalf("buildPanelLayout() = %q", got)
	}
}

func TestPanelLayoutLeadFourPanelsUsesAtLeastThreeRows(t *testing.T) {
	t.Parallel()
	got := panelLayoutLead(4)
	if !strings.Contains(strings.ToLower(got), "three horizontal rows") {
		t.Fatalf("panelLayoutLead(4) = %q, want at least three horizontal rows", got)
	}
	// Reject the old two-row prescription, not the phrase "do not use 2×2".
	if strings.Contains(got, "in a 2×2 grid") || strings.Contains(got, "in a 2x2 grid") {
		t.Fatalf("panelLayoutLead(4) = %q, must not prescribe a 2×2 grid layout", got)
	}
}

func TestSplitIntoSectionsUsesRuneBoundaries(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("a", 99) + "б" + strings.Repeat("c", 300)
	sections := splitIntoSections(text, 4)
	if len(sections) != 4 {
		t.Fatalf("splitIntoSections() sections = %d, want 4", len(sections))
	}
	for i, section := range sections {
		if !utf8.ValidString(section) {
			t.Fatalf("splitIntoSections() section %d is not valid UTF-8: %q", i, section)
		}
	}
	if !strings.Contains(sections[0], "б") {
		t.Fatalf("splitIntoSections() first section = %q, want mixed-width rune preserved", sections[0])
	}
}

func TestBuildPanelLayoutTruncatesCyrillicOnRuneBoundaries(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("a", 899) + "б" + strings.Repeat("c", 100)
	got := buildPanelLayout(text, nil, 2, 900)
	if !utf8.ValidString(got) {
		t.Fatalf("buildPanelLayout() returned invalid UTF-8: %q", got)
	}
	if !strings.Contains(got, "…") {
		t.Fatalf("buildPanelLayout() = %q, want truncation ellipsis", got)
	}
	if !strings.Contains(got, "б") {
		t.Fatalf("buildPanelLayout() = %q, want mixed-width rune preserved", got)
	}
}

func TestBuildPanelLayoutRespectsPromptMaxChars(t *testing.T) {
	t.Parallel()

	got := buildPanelLayout(strings.Repeat("a", 80), nil, 2, 12)
	if !strings.Contains(got, "…") {
		t.Fatalf("buildPanelLayout() = %q, want truncation ellipsis", got)
	}
	if strings.Contains(got, strings.Repeat("a", 20)) {
		t.Fatalf("buildPanelLayout() = %q, want configured truncation limit", got)
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

func TestValidateGeneratedResultUsesExcelStylePanelLabels(t *testing.T) {
	t.Parallel()

	page := make([]string, 27)
	for i := range page {
		page[i] = "а"
	}
	page[26] = "latin"

	err := validateGeneratedResult(GenerateResult{
		StoryText: "история",
		Bible:     "библия",
		Title:     "Заглавие",
		PanelScript: [][]string{
			page,
		},
	}, "Cyrillic")
	if err == nil {
		t.Fatal("validateGeneratedResult() error = nil, want panel label failure")
	}
	if !strings.Contains(err.Error(), "panel script P1-AA") {
		t.Fatalf("validateGeneratedResult() error = %v, want Excel-style panel label", err)
	}
}

func TestComicAssetAndPDFDirs(t *testing.T) {
	t.Parallel()

	if got, want := comicAssetsDir("comics", "slug"), filepath.Join("comics", "assets", "slug"); got != want {
		t.Fatalf("comicAssetsDir() = %q, want %q", got, want)
	}
	if got, want := comicAssetsDir(".", "slug"), filepath.Join(".", "comics", "assets", "slug"); got != want {
		t.Fatalf("comicAssetsDir() = %q, want %q", got, want)
	}
	if got, want := comicAssetsDir("/tmp/out", "slug"), filepath.Join("/tmp/out", "comics", "assets", "slug"); got != want {
		t.Fatalf("comicAssetsDir() = %q, want %q", got, want)
	}
	if got, want := comicsPDFDir("comics"), filepath.Join("comics", "PDF"); got != want {
		t.Fatalf("comicsPDFDir() = %q, want %q", got, want)
	}
	if got, want := comicsPDFDir("."), filepath.Join(".", "comics", "PDF"); got != want {
		t.Fatalf("comicsPDFDir() = %q, want %q", got, want)
	}
	if got, want := comicsPDFDir("/tmp/out"), filepath.Join("/tmp/out", "comics", "PDF"); got != want {
		t.Fatalf("comicsPDFDir() = %q, want %q", got, want)
	}
}

func TestCopyGalleryPNGsToComicsGallery(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	comicsRoot := filepath.Join(root, "comics")
	comicDir := filepath.Join(comicsRoot, "my-slug")
	if err := os.MkdirAll(comicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comicDir, "my-slug_gallery_1.png"), []byte("png1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyGalleryPNGsToComicsGallery(comicsRoot, comicDir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(comicsRoot, "gallery", "my-slug_gallery_1.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "png1" {
		t.Fatalf("copied file = %q", b)
	}
	if _, err := os.Stat(filepath.Join(root, "comics", "comics", "gallery", "my-slug_gallery_1.png")); !os.IsNotExist(err) {
		t.Fatalf("nested gallery path exists or stat failed unexpectedly: %v", err)
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
	originalSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() {
		sleep = originalSleep
	})

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
		GalleryPages:          1,
	})
	runner.assemblePDF = func(outputDir, titleSlug string, imagePaths []string, _ PDFAssembleOptions) (string, error) {
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
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "PDF", "forced-slug.pdf")); err != nil {
		t.Fatalf("pdf missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "assets", "forced-slug", "forced-slug_story.txt")); err != nil {
		t.Fatalf("story missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "gallery", "forced-slug_gallery_1.png")); err != nil {
		t.Fatalf("gallery image missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "comics", "gallery", "forced-slug_gallery_1.png")); !os.IsNotExist(err) {
		t.Fatalf("nested gallery path exists or stat failed unexpectedly: %v", err)
	}
}

func TestRunnerRunPromptWritesSingleAsset(t *testing.T) {
	originalLeakValidation := validateImagePromptLeakageFn
	validateImagePromptLeakageFn = func(context.Context, string, string, string) error { return nil }
	t.Cleanup(func() {
		validateImagePromptLeakageFn = originalLeakValidation
	})

	tmpDir := t.TempDir()
	runner := NewRunner(&RunnerConfig{
		ImageProvider:  fakeImageProvider{t: t},
		TextProvider:   fakeTextProvider{text: "ComicForge Superhero Logo"},
		Prompts:        fakePromptRenderer{},
		OutputDir:      tmpDir,
		UltraRealistic: boolPtr(false),
		PageMaxRetries: 1,
		PageRetryBase:  time.Second,
	})

	if err := runner.RunPrompt(context.Background(), "a robot reading a newspaper"); err != nil {
		t.Fatalf("RunPrompt() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "assets", "comicforge-superhero-logo", "prompt.png")); err != nil {
		t.Fatalf("prompt image missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "PDF")); !os.IsNotExist(err) {
		t.Fatalf("PDF directory exists unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "gallery")); !os.IsNotExist(err) {
		t.Fatalf("gallery directory exists unexpectedly: %v", err)
	}
}

func TestRunnerRunPromptUsesShortFallbackSlugWithoutTextProvider(t *testing.T) {
	originalLeakValidation := validateImagePromptLeakageFn
	validateImagePromptLeakageFn = func(context.Context, string, string, string) error { return nil }
	t.Cleanup(func() {
		validateImagePromptLeakageFn = originalLeakValidation
	})

	tmpDir := t.TempDir()
	runner := NewRunner(&RunnerConfig{
		ImageProvider:  fakeImageProvider{t: t},
		Prompts:        fakePromptRenderer{},
		OutputDir:      tmpDir,
		UltraRealistic: boolPtr(false),
		PageMaxRetries: 1,
		PageRetryBase:  time.Second,
	})

	longPrompt := "generate a logo for ComicForge which is a comic book generator displaying a superhero fighting a bad guy and speech bubbles"
	if err := runner.RunPrompt(context.Background(), longPrompt); err != nil {
		t.Fatalf("RunPrompt() error = %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(tmpDir, "comics", "assets"))
	if err != nil {
		t.Fatalf("read assets dir: %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("asset dirs = %d, want %d", got, want)
	}
	slug := entries[0].Name()
	if len(slug) > maxPromptSlugLength {
		t.Fatalf("prompt slug length = %d, want <= %d (%q)", len(slug), maxPromptSlugLength, slug)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "comics", "assets", slug, "prompt.png")); err != nil {
		t.Fatalf("prompt image missing: %v", err)
	}
}

func TestRunnerRunPromptValidatesImageOutput(t *testing.T) {
	originalLeakValidation := validateImagePromptLeakageFn
	defer func() {
		validateImagePromptLeakageFn = originalLeakValidation
	}()

	var calls int
	var gotLabel string
	var gotScript string
	validateImagePromptLeakageFn = func(_ context.Context, outputFile, label, script string) error {
		calls++
		if outputFile == "" {
			t.Fatal("validator received empty output file")
		}
		gotLabel = label
		gotScript = script
		return nil
	}

	tmpDir := t.TempDir()
	runner := NewRunner(&RunnerConfig{
		ImageProvider:  fakeImageProvider{t: t},
		Prompts:        fakePromptRenderer{},
		OutputDir:      tmpDir,
		Slug:           "manual-robot",
		UltraRealistic: boolPtr(false),
		PageMaxRetries: 1,
		PageRetryBase:  time.Second,
	})

	if err := runner.RunPrompt(context.Background(), "a robot reading a newspaper"); err != nil {
		t.Fatalf("RunPrompt() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("validator calls = %d, want 1", calls)
	}
	if gotLabel != "manual prompt image" {
		t.Fatalf("validator label = %q, want %q", gotLabel, "manual prompt image")
	}
	if gotScript == "" {
		t.Fatal("validator script was empty")
	}
}

func TestRunnerRunPromptRejectsEmptyPrompt(t *testing.T) {
	t.Parallel()

	runner := NewRunner(&RunnerConfig{
		ImageProvider:  fakeImageProvider{t: t},
		Prompts:        fakePromptRenderer{},
		OutputDir:      t.TempDir(),
		UltraRealistic: boolPtr(false),
	})

	if err := runner.RunPrompt(context.Background(), "   "); err == nil {
		t.Fatal("RunPrompt() error = nil, want validation failure")
	}
}

func TestRunnerRunPromptReturnsErrorWhenPromptRendererMissing(t *testing.T) {
	runner := NewRunner(&RunnerConfig{
		ImageProvider:  fakeImageProvider{t: t},
		OutputDir:      t.TempDir(),
		Slug:           "manual-robot",
		UltraRealistic: boolPtr(false),
		PageMaxRetries: 1,
		PageRetryBase:  time.Second,
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RunPrompt() panicked: %v", r)
		}
	}()

	if err := runner.RunPrompt(context.Background(), "a robot reading a newspaper"); err == nil {
		t.Fatal("RunPrompt() error = nil, want readiness failure")
	}
}

func TestRunnerRunPromptAppliesStyleThemeAndUltraContext(t *testing.T) {
	t.Parallel()

	originalLeakValidation := validateImagePromptLeakageFn
	validateImagePromptLeakageFn = func(context.Context, string, string, string) error { return nil }
	t.Cleanup(func() {
		validateImagePromptLeakageFn = originalLeakValidation
	})

	provider := &capturingImageProvider{t: t}
	renderer := &recordingPromptRenderer{}
	runner := NewRunner(&RunnerConfig{
		ImageProvider:  provider,
		Prompts:        renderer,
		OutputDir:      t.TempDir(),
		Slug:           "manual-robot",
		Style:          "noir",
		Theme:          "mystery",
		Language:       "English",
		Script:         "Latin",
		UltraRealistic: boolPtr(true),
		PageMaxRetries: 1,
		PageRetryBase:  time.Second,
	})

	if err := runner.RunPrompt(context.Background(), "a robot reading a newspaper"); err != nil {
		t.Fatalf("RunPrompt() error = %v", err)
	}
	if provider.lastPrompt == "" {
		t.Fatal("image prompt was not captured")
	}
	if !strings.Contains(provider.lastPrompt, "Create a single image based on this prompt") {
		t.Fatalf("prompt = %q, want manual prompt template", provider.lastPrompt)
	}
	if !strings.Contains(provider.lastPrompt, "a robot reading a newspaper") {
		t.Fatalf("prompt = %q, want raw user prompt", provider.lastPrompt)
	}
	if !strings.Contains(provider.lastPrompt, "Visual style: noir.") {
		t.Fatalf("prompt = %q, want style context", provider.lastPrompt)
	}
	if !strings.Contains(provider.lastPrompt, "Theme context: mystery.") {
		t.Fatalf("prompt = %q, want theme context", provider.lastPrompt)
	}
	if !strings.Contains(provider.lastPrompt, "FINAL STYLE LOCK — PHOTOREALISM") {
		t.Fatalf("prompt = %q, want ultra-realistic context", provider.lastPrompt)
	}
}

func TestNewRunnerUsesUltraFlagsForStyleMode(t *testing.T) {
	t.Parallel()

	t.Run("comic", func(t *testing.T) {
		ultra := false
		runner := NewRunner(&RunnerConfig{
			TextProvider:   fakeTextProvider{text: "story"},
			ImageProvider:  fakeImageProvider{t: t},
			Prompts:        fakePromptRenderer{},
			UltraRealistic: &ultra,
			StoryPages:     1,
			GalleryPages:   0,
			PanelsPerPage:  1,
			Styles: map[string][]string{
				styleModeComic:     {"comic-ink"},
				styleModeRealistic: {"photo-real"},
			},
			AspectRatio:    "1:1",
			PromptMaxChars: 12,
			PageMaxRetries: 1,
			PageRetryBase:  time.Second,
			ChunkWords:     2,
		})
		if runner.artist == nil {
			t.Fatal("artist is nil")
		}
		if runner.artist.ultraRealistic {
			t.Fatal("ultraRealistic = true, want false when explicitly disabled")
		}
		if got, want := runner.artist.styleMode, styleModeComic; got != want {
			t.Fatalf("styleMode = %q, want %q", got, want)
		}
	})

	t.Run("realistic", func(t *testing.T) {
		ultra := true
		runner := NewRunner(&RunnerConfig{
			TextProvider:   fakeTextProvider{text: "story"},
			ImageProvider:  fakeImageProvider{t: t},
			Prompts:        fakePromptRenderer{},
			UltraRealistic: &ultra,
			StoryPages:     1,
			GalleryPages:   0,
			PanelsPerPage:  1,
			Styles: map[string][]string{
				styleModeComic:     {"comic-ink"},
				styleModeRealistic: {"photo-real"},
			},
			AspectRatio:    "1:1",
			PromptMaxChars: 12,
			PageMaxRetries: 1,
			PageRetryBase:  time.Second,
			ChunkWords:     2,
		})
		if runner.artist == nil {
			t.Fatal("artist is nil")
		}
		if !runner.artist.ultraRealistic {
			t.Fatal("ultraRealistic = false, want true when explicitly enabled")
		}
		if got, want := runner.artist.styleMode, styleModeRealistic; got != want {
			t.Fatalf("styleMode = %q, want %q", got, want)
		}
	})
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
	runner.assemblePDF = func(string, string, []string, PDFAssembleOptions) (string, error) {
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

func TestDrawComicPagesUsesConfiguredStylePools(t *testing.T) {
	originalLeakValidation := validateImagePromptLeakageFn
	validateImagePromptLeakageFn = func(context.Context, string, string, string) error { return nil }
	t.Cleanup(func() {
		validateImagePromptLeakageFn = originalLeakValidation
	})

	tests := []struct {
		name      string
		styleMode string
		wantStyle string
	}{
		{name: "comic", styleMode: styleModeComic, wantStyle: "comic-ink"},
		{name: "realistic", styleMode: styleModeRealistic, wantStyle: "photo-real"},
		{name: "cartoon", styleMode: styleModeCartoon, wantStyle: "cartoon-ink"},
		{name: "action90s", styleMode: styleModeAction90s, wantStyle: "action-ink"},
		{name: "manga", styleMode: styleModeManga, wantStyle: "manga-ink"},
		{name: "horror", styleMode: styleModeHorror, wantStyle: "horror-ink"},
		{name: "watercolor", styleMode: styleModeWatercolor, wantStyle: "watercolor-ink"},
		{name: "pulp", styleMode: styleModePulp, wantStyle: "pulp-ink"},
		{name: "mecha", styleMode: styleModeMecha, wantStyle: "mecha-ink"},
		{name: "pixel-art", styleMode: styleModePixelArt, wantStyle: "pixel-ink"},
		{name: "ink-wash", styleMode: styleModeInkWash, wantStyle: "inkwash-ink"},
		{name: "clay", styleMode: styleModeClay, wantStyle: "clay-ink"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := &recordingPromptRenderer{}
			artist := NewArtist(&ArtistConfig{
				ImageProvider: fakeImageProvider{t: t},
				Prompts:       renderer,
				OutputDir:     t.TempDir(),
				StyleMode:     tt.styleMode,
				Styles: map[string][]string{
					styleModeComic:      {"comic-ink"},
					styleModeRealistic:  {"photo-real"},
					styleModeCartoon:    {"cartoon-ink"},
					styleModeAction90s:  {"action-ink"},
					styleModeManga:      {"manga-ink"},
					styleModeHorror:     {"horror-ink"},
					styleModeWatercolor: {"watercolor-ink"},
					styleModePulp:       {"pulp-ink"},
					styleModeMecha:      {"mecha-ink"},
					styleModePixelArt:   {"pixel-ink"},
					styleModeInkWash:    {"inkwash-ink"},
					styleModeClay:       {"clay-ink"},
				},
				Language:      "English",
				Script:        "Latin",
				PanelsPerPage: 2,
			})

			if _, err := artist.DrawComicPages(context.Background(), "story", "bible", "slug", []WordEntry{{Word: "ябълка"}}, nil); err != nil {
				t.Fatalf("DrawComicPages() error = %v", err)
			}

			var gotStyle string
			for _, call := range renderer.calls {
				if call.name != storyPagePromptTemplate {
					continue
				}
				gotStyle, _ = call.data["Style"].(string)
				break
			}
			if gotStyle != tt.wantStyle {
				t.Fatalf("Style prompt = %q, want %q", gotStyle, tt.wantStyle)
			}
		})
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

func TestGenerateWithRetryUsesConfiguredRetriesAndBackoff(t *testing.T) {
	originalSleep := sleep
	var pauses []time.Duration
	sleep = func(d time.Duration) {
		pauses = append(pauses, d)
	}
	t.Cleanup(func() {
		sleep = originalSleep
	})

	artist := NewArtist(&ArtistConfig{
		ImageProvider:  failingImageProvider{},
		Prompts:        fakePromptRenderer{},
		OutputDir:      t.TempDir(),
		UltraRealistic: false,
		PageMaxRetries: 3,
		PageRetryBase:  2 * time.Second,
	})

	err := artist.generateWithRetry(context.Background(), "prompt", filepath.Join(t.TempDir(), "out.png"), "cover page", nil)
	if err == nil {
		t.Fatal("generateWithRetry() error = nil, want failure")
	}
	if got, want := len(pauses), 2; got != want {
		t.Fatalf("sleep calls = %d, want %d", got, want)
	}
	if pauses[0] != 2*time.Second || pauses[1] != 4*time.Second {
		t.Fatalf("sleep pauses = %v, want [2s 4s]", pauses)
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

func TestNarratorUsesConfiguredChunkWords(t *testing.T) {
	t.Parallel()

	text := strings.Join([]string{
		"едно две",
		"три четири",
		"пет шест",
	}, "\n\n")

	n := NewNarrator(&NarratorConfig{
		MainProvider: fakeTTSProvider{},
		Prompts:      fakePromptRenderer{},
		ChunkWords:   2,
	})
	paths, err := n.narrateMainStory(context.Background(), text, t.TempDir())
	if err != nil {
		t.Fatalf("narrateMainStory() error = %v", err)
	}
	if got, want := len(paths), 3; got != want {
		t.Fatalf("chunk count = %d, want %d", got, want)
	}
}

type fakePromptRenderer struct{}

func (fakePromptRenderer) RenderPrompt(name string, data any) (string, error) {
	switch name {
	case storySystemPromptTemplate:
		return "system prompt", nil
	case storyPromptTemplate, storyFullPromptTemplate:
		return "prompt", nil
	case renderingRequirementPrompt:
		return "ULTRA-REALISTIC MODE for this image:\n  • The image must look like a real photograph or a frame from a high-budget live-action film.\n  • Skin, hair, fabric, metal, and environments must look natural and real, with no drawn or painterly effect.\n  • Avoid cartoon, anime, manga, comic-line-art, or obviously illustrated rendering.\n  • Text elements, if required by the composition, should look like part of a photographed physical cover, sign, or poster.\n  • For gallery images, the whole image must be entirely photographic.\n", nil
	case renderingRequirementEndPrompt:
		return "FINAL STYLE LOCK — PHOTOREALISM: the entire image must look camera-captured. If anything looks drawn or painted, the result is wrong. Do not drift toward comic art between panels or in gallery images.", nil
	case manualPromptTemplate:
		return renderManualPromptForTest(data), nil
	case manualTitlePromptTemplate:
		return "short title prompt", nil
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
	case renderingRequirementPrompt:
		return "ULTRA-REALISTIC MODE for this image:\n  • The image must look like a real photograph or a frame from a high-budget live-action film.\n  • Skin, hair, fabric, metal, and environments must look natural and real, with no drawn or painterly effect.\n  • Avoid cartoon, anime, manga, comic-line-art, or obviously illustrated rendering.\n  • Text elements, if required by the composition, should look like part of a photographed physical cover, sign, or poster.\n  • For gallery images, the whole image must be entirely photographic.\n", nil
	case renderingRequirementEndPrompt:
		return "FINAL STYLE LOCK — PHOTOREALISM: the entire image must look camera-captured. If anything looks drawn or painted, the result is wrong. Do not drift toward comic art between panels or in gallery images.", nil
	case manualPromptTemplate:
		return renderManualPromptForTest(data), nil
	case manualTitlePromptTemplate:
		return "short title prompt", nil
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

func renderManualPromptForTest(data any) string {
	m, _ := data.(map[string]any)
	if m == nil {
		return "Create a single image based on this prompt:"
	}
	var sb strings.Builder
	sb.WriteString("Create a single image based on this prompt:\n\n")
	sb.WriteString(fmt.Sprint(m["Prompt"]))
	sb.WriteString("\n\n")
	if style, _ := m["Style"].(string); strings.TrimSpace(style) != "" {
		fmt.Fprintf(&sb, "Visual style: %s.\n", style)
	}
	if theme, _ := m["Theme"].(string); strings.TrimSpace(theme) != "" {
		fmt.Fprintf(&sb, "Theme context: %s.\n", theme)
	}
	if requirement, _ := m["RenderingRequirement"].(string); strings.TrimSpace(requirement) != "" {
		sb.WriteString(requirement)
		sb.WriteString("\n")
	}
	sb.WriteString("No text, captions, logos, borders, panels, or UI elements.")
	if requirementEnd, _ := m["RenderingRequirementEnd"].(string); strings.TrimSpace(requirementEnd) != "" {
		sb.WriteString(requirementEnd)
	}
	return sb.String()
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

type capturingImageProvider struct {
	t          *testing.T
	lastPrompt string
}

func (p *capturingImageProvider) Name() string       { return "capturing-image" }
func (p *capturingImageProvider) IsAvailable() error { return nil }
func (p *capturingImageProvider) GenerateImage(_ context.Context, prompt, outputFile string) error {
	p.lastPrompt = prompt
	if err := os.WriteFile(outputFile, []byte("png"), 0o644); err != nil {
		p.t.Fatal(err)
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

func boolPtr(v bool) *bool {
	return &v
}
