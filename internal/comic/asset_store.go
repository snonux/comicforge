package comic

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/snonux/comicforge/internal/vocab"
)

const ttsTodoContent = `# Story Narration - Fallback Placeholder
#
# Comic narration was not produced (missing provider or generation error).
#
# To generate narration manually, run again with a configured TTS provider, or
# use a text-to-speech backend of your choice and save the result as story_narration.mp3.
`

// AssetStore writes comic pipeline artifacts (story text, vocabulary, theme,
// TTS placeholders, gallery copies, and output directories).
// Defined at the point of use so Runner can be unit-tested without real I/O.
type AssetStore interface {
	PrepareOutputDirs(outputRoot, slug string) (assetsDir string, err error)
	EnsureAssetsDir(outputRoot, slug string) (assetsDir string, err error)
	SaveStoryText(text, titleSlug, dir string) error
	SaveVocabularyFile(storyText string, entries []vocab.WordEntry, titleSlug, dir string) error
	SaveThemeFile(theme, titleSlug, dir string) error
	SaveTTSPlaceholder(titleSlug, dir string) error
	CopyGalleryPNGs(outputRoot, comicDir string) error
}

// FSAssetStore is the filesystem-backed AssetStore.
type FSAssetStore struct{}

var _ AssetStore = (*FSAssetStore)(nil)

// NewFSAssetStore returns the default filesystem AssetStore.
func NewFSAssetStore() *FSAssetStore {
	return &FSAssetStore{}
}

// PrepareOutputDirs creates the comic assets and PDF directories for a slug.
func (s *FSAssetStore) PrepareOutputDirs(outputRoot, slug string) (string, error) {
	assetsDir, err := s.EnsureAssetsDir(outputRoot, slug)
	if err != nil {
		return "", err
	}
	pdfDir := comicsPDFDir(outputRoot)
	if err := os.MkdirAll(pdfDir, 0o755); err != nil {
		return "", fmt.Errorf("create comics pdf dir %s: %w", pdfDir, err)
	}
	return assetsDir, nil
}

// EnsureAssetsDir creates the per-comic assets directory and returns its path.
func (s *FSAssetStore) EnsureAssetsDir(outputRoot, slug string) (string, error) {
	assetsDir := comicAssetsDir(outputRoot, slug)
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return "", fmt.Errorf("create comics assets dir %s: %w", assetsDir, err)
	}
	return assetsDir, nil
}

// SaveStoryText writes the story text artifact.
func (s *FSAssetStore) SaveStoryText(text, titleSlug, dir string) error {
	path := filepath.Join(dir, titleSlug+"_story.txt")
	if err := os.WriteFile(path, []byte(text+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write story file: %w", err)
	}
	fmt.Printf("Story saved: %s\n", path)
	return nil
}

// SaveVocabularyFile writes the vocabulary + story summary artifact.
func (s *FSAssetStore) SaveVocabularyFile(storyText string, entries []vocab.WordEntry, titleSlug, dir string) error {
	path := filepath.Join(dir, titleSlug+"_comic_vocabulary.txt")
	var sb strings.Builder
	sb.WriteString("# Vocabulary Words\n\n")
	for _, entry := range entries {
		word := strings.TrimSpace(entry.Word)
		if word == "" {
			word = strings.TrimSpace(entry.Translation)
		}
		if entry.Translation != "" && entry.Word != "" {
			fmt.Fprintf(&sb, "  %s - %s\n", entry.Word, entry.Translation)
			continue
		}
		fmt.Fprintf(&sb, "  %s\n", word)
	}
	sb.WriteString("\n# Story Text\n\n")
	sb.WriteString(strings.TrimSpace(storyText))
	sb.WriteString("\n")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write vocabulary file: %w", err)
	}
	fmt.Printf("Vocabulary saved: %s\n", path)
	return nil
}

// SaveThemeFile writes the theme artifact.
func (s *FSAssetStore) SaveThemeFile(theme, titleSlug, dir string) error {
	path := filepath.Join(dir, titleSlug+"_theme.txt")
	if err := os.WriteFile(path, []byte(theme+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write theme file: %w", err)
	}
	fmt.Printf("Theme saved: %s\n", path)
	return nil
}

// SaveTTSPlaceholder writes the fallback TTS todo artifact.
func (s *FSAssetStore) SaveTTSPlaceholder(titleSlug, dir string) error {
	path := filepath.Join(dir, titleSlug+"_tts_todo.txt")
	if err := os.WriteFile(path, []byte(ttsTodoContent), 0o644); err != nil {
		return fmt.Errorf("failed to write TTS placeholder: %w", err)
	}
	fmt.Printf("TTS placeholder saved: %s\n", path)
	return nil
}

// CopyGalleryPNGs copies *_gallery_*.png files from comicDir into outputRoot/gallery.
func (s *FSAssetStore) CopyGalleryPNGs(outputRoot, comicDir string) error {
	destDir := filepath.Join(outputRoot, "gallery")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir gallery: %w", err)
	}
	matches, err := filepath.Glob(filepath.Join(comicDir, "*_gallery_*.png"))
	if err != nil {
		return err
	}
	for _, src := range matches {
		dst := filepath.Join(destDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("%s -> %s: %w", src, dst, err)
		}
	}
	if len(matches) > 0 {
		fmt.Printf("Gallery images copied to %s (%d files)\n", destDir, len(matches))
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	_, err = io.Copy(out, in)
	return err
}
