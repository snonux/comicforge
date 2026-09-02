package comic

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/snonux/comicforge/internal/vocab"
)

func TestFSAssetStorePrepareOutputDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewFSAssetStore()
	assetsDir, err := store.PrepareOutputDirs(root, "my-slug")
	if err != nil {
		t.Fatalf("PrepareOutputDirs() error = %v", err)
	}
	wantAssets := filepath.Join(root, "comics", "assets", "my-slug")
	if assetsDir != wantAssets {
		t.Fatalf("PrepareOutputDirs() = %q, want %q", assetsDir, wantAssets)
	}
	if _, err := os.Stat(wantAssets); err != nil {
		t.Fatalf("assets dir missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "comics", "PDF")); err != nil {
		t.Fatalf("pdf dir missing: %v", err)
	}
}

func TestFSAssetStorePrepareOutputDirsWriteError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	blocker := filepath.Join(root, "comics")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewFSAssetStore().PrepareOutputDirs(root, "slug")
	if err == nil {
		t.Fatal("PrepareOutputDirs() error = nil, want mkdir failure")
	}
	if !strings.Contains(err.Error(), "create comics assets dir") {
		t.Fatalf("PrepareOutputDirs() error = %v, want assets dir failure", err)
	}
}

func TestFSAssetStoreSaveStoryText(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := NewFSAssetStore().SaveStoryText("hello story", "slug", dir); err != nil {
		t.Fatalf("SaveStoryText() error = %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "slug_story.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "hello story\n" {
		t.Fatalf("story content = %q, want %q", got, "hello story\n")
	}
}

func TestFSAssetStoreSaveStoryTextWriteError(t *testing.T) {
	t.Parallel()

	err := NewFSAssetStore().SaveStoryText("x", "slug", filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("SaveStoryText() error = nil, want write failure")
	}
	if !strings.Contains(err.Error(), "failed to write story file") {
		t.Fatalf("SaveStoryText() error = %v, want write failure", err)
	}
}

func TestFSAssetStoreSaveVocabularyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	entries := []vocab.WordEntry{
		{Word: "ябълка", Translation: "apple"},
		{Word: "книга"},
		{Translation: "orphan"},
	}
	if err := NewFSAssetStore().SaveVocabularyFile("story body", entries, "slug", dir); err != nil {
		t.Fatalf("SaveVocabularyFile() error = %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "slug_comic_vocabulary.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{
		"# Vocabulary Words",
		"ябълка - apple",
		"книга",
		"orphan",
		"# Story Text",
		"story body",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("vocabulary file missing %q:\n%s", want, got)
		}
	}
}

func TestFSAssetStoreSaveVocabularyFileWriteError(t *testing.T) {
	t.Parallel()

	err := NewFSAssetStore().SaveVocabularyFile("s", nil, "slug", filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("SaveVocabularyFile() error = nil, want write failure")
	}
	if !strings.Contains(err.Error(), "failed to write vocabulary file") {
		t.Fatalf("SaveVocabularyFile() error = %v, want write failure", err)
	}
}

func TestFSAssetStoreSaveThemeFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := NewFSAssetStore().SaveThemeFile("forest adventure", "slug", dir); err != nil {
		t.Fatalf("SaveThemeFile() error = %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "slug_theme.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "forest adventure\n" {
		t.Fatalf("theme content = %q", got)
	}
}

func TestFSAssetStoreSaveThemeFileWriteError(t *testing.T) {
	t.Parallel()

	err := NewFSAssetStore().SaveThemeFile("t", "slug", filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("SaveThemeFile() error = nil, want write failure")
	}
	if !strings.Contains(err.Error(), "failed to write theme file") {
		t.Fatalf("SaveThemeFile() error = %v, want write failure", err)
	}
}

func TestFSAssetStoreSaveTTSPlaceholder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := NewFSAssetStore().SaveTTSPlaceholder("slug", dir); err != nil {
		t.Fatalf("SaveTTSPlaceholder() error = %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "slug_tts_todo.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Fallback Placeholder") {
		t.Fatalf("tts placeholder content unexpected: %q", b)
	}
}

func TestFSAssetStoreSaveTTSPlaceholderWriteError(t *testing.T) {
	t.Parallel()

	err := NewFSAssetStore().SaveTTSPlaceholder("slug", filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("SaveTTSPlaceholder() error = nil, want write failure")
	}
	if !strings.Contains(err.Error(), "failed to write TTS placeholder") {
		t.Fatalf("SaveTTSPlaceholder() error = %v, want write failure", err)
	}
}

func TestFSAssetStoreCopyGalleryPNGs(t *testing.T) {
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
	if err := NewFSAssetStore().CopyGalleryPNGs(comicsRoot, comicDir); err != nil {
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

func TestFSAssetStoreCopyGalleryPNGsNoMatches(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	comicDir := filepath.Join(root, "assets")
	if err := os.MkdirAll(comicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := NewFSAssetStore().CopyGalleryPNGs(root, comicDir); err != nil {
		t.Fatalf("CopyGalleryPNGs() error = %v, want nil for no matches", err)
	}
	if _, err := os.Stat(filepath.Join(root, "gallery")); err != nil {
		t.Fatalf("gallery dir should still be created: %v", err)
	}
}

func TestFSAssetStoreCopyGalleryPNGsCopyError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	comicDir := filepath.Join(root, "assets")
	if err := os.MkdirAll(comicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A directory matching the gallery glob cannot be copied as a file.
	bad := filepath.Join(comicDir, "slug_gallery_1.png")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	err := NewFSAssetStore().CopyGalleryPNGs(root, comicDir)
	if err == nil {
		t.Fatal("CopyGalleryPNGs() error = nil, want copy failure")
	}
}

func TestNewRunnerDefaultsAssetStore(t *testing.T) {
	t.Parallel()

	r := NewRunner(&RunnerConfig{})
	if r.assetStore == nil {
		t.Fatal("assetStore = nil, want default FSAssetStore")
	}
	if _, ok := r.assetStore.(*FSAssetStore); !ok {
		t.Fatalf("assetStore type = %T, want *FSAssetStore", r.assetStore)
	}
}

func TestNewRunnerUsesInjectedAssetStore(t *testing.T) {
	t.Parallel()

	fake := &stubAssetStore{assetsDir: t.TempDir()}
	r := NewRunner(&RunnerConfig{AssetStore: fake})
	if r.assetStore != fake {
		t.Fatalf("assetStore = %v, want injected fake", r.assetStore)
	}
}

func TestRunnerRunPropagatesAssetStorePrepareError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "vocab.txt"), []byte("ябълка = apple\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(&RunnerConfig{
		TextProvider: &scriptedTextProvider{responses: []string{
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
		}},
		ImageProvider: fakeImageProvider{t: t},
		Prompts:       fakePromptRenderer{},
		OutputDir:     tmpDir,
		AssetStore: &stubAssetStore{
			prepareErr: errors.New("prepare failed"),
		},
		PDFAssembler: funcPDFAssembler{fn: func(context.Context, string, string, []string, PDFAssembleOptions) (string, error) {
			return "x.pdf", nil
		}},
	})
	err := runner.Run(context.Background(), filepath.Join(tmpDir, "vocab.txt"))
	if err == nil {
		t.Fatal("Run() error = nil, want prepare failure")
	}
	if !strings.Contains(err.Error(), "prepare failed") {
		t.Fatalf("Run() error = %v, want prepare failed", err)
	}
}

func TestRunnerRunPropagatesAssetStoreStoryWriteError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	assetsDir := filepath.Join(tmpDir, "assets-out")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "vocab.txt"), []byte("ябълка = apple\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(&RunnerConfig{
		TextProvider: &scriptedTextProvider{responses: []string{
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
		}},
		ImageProvider: fakeImageProvider{t: t},
		Prompts:       fakePromptRenderer{},
		OutputDir:     tmpDir,
		AssetStore: &stubAssetStore{
			assetsDir: assetsDir,
			storyErr:  errors.New("story write failed"),
		},
		PDFAssembler: funcPDFAssembler{fn: func(context.Context, string, string, []string, PDFAssembleOptions) (string, error) {
			return "x.pdf", nil
		}},
	})
	err := runner.Run(context.Background(), filepath.Join(tmpDir, "vocab.txt"))
	if err == nil {
		t.Fatal("Run() error = nil, want story write failure")
	}
	if !strings.Contains(err.Error(), "story write failed") {
		t.Fatalf("Run() error = %v, want story write failed", err)
	}
}

// stubAssetStore is a test double for AssetStore.
type stubAssetStore struct {
	assetsDir  string
	prepareErr error
	storyErr   error
}

func (s *stubAssetStore) PrepareOutputDirs(string, string) (string, error) {
	if s.prepareErr != nil {
		return "", s.prepareErr
	}
	return s.assetsDir, nil
}

func (s *stubAssetStore) EnsureAssetsDir(string, string) (string, error) {
	return s.assetsDir, nil
}

func (s *stubAssetStore) SaveStoryText(string, string, string) error {
	return s.storyErr
}

func (s *stubAssetStore) SaveVocabularyFile(string, []vocab.WordEntry, string, string) error {
	return nil
}

func (s *stubAssetStore) SaveThemeFile(string, string, string) error { return nil }

func (s *stubAssetStore) SaveTTSPlaceholder(string, string) error { return nil }

func (s *stubAssetStore) CopyGalleryPNGs(string, string) error { return nil }
