package comic

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/snonux/comicforge/internal/vocab"
)

const comicIdentityFile = ".comicforge-identity"

// ClaimComicIdentity reserves an empty directory for a generation, or verifies
// that existing artifacts belong to exactly the same generation before resume.
// Legacy assets, PDFs, and gallery files without an identity are never adopted
// implicitly.
func (s *FSAssetStore) ClaimComicIdentity(outputRoot, slug, identity string) error {
	dir := comicAssetsDir(outputRoot, slug)
	path := filepath.Join(dir, comicIdentityFile)
	stored, err := os.ReadFile(path)
	if err == nil {
		return matchComicIdentity(dir, string(stored), identity)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read comic identity: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("inspect comic assets: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("comic assets %q have no generation identity; choose a different --slug or output directory", dir)
	}
	if err = checkUnclaimedExternalArtifacts(outputRoot, slug); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		// Another run reserved the directory after our initial read.
		stored, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read claimed comic identity: %w", err)
		}
		return matchComicIdentity(dir, string(stored), identity)
	}
	if err != nil {
		return fmt.Errorf("claim comic identity: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(identity + "\n"); err != nil {
		return fmt.Errorf("write comic identity: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close comic identity: %w", err)
	}
	return nil
}

func checkUnclaimedExternalArtifacts(outputRoot, slug string) error {
	pdf := filepath.Join(comicsPDFDir(outputRoot), slug+".pdf")
	if _, err := os.Lstat(pdf); err == nil {
		return unclaimedArtifactError(pdf)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect comic PDF: %w", err)
	}
	gallery := filepath.Join(comicsRootDir(outputRoot), "gallery")
	entries, err := os.ReadDir(gallery)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect comic gallery: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, slug+"_gallery_") && strings.HasSuffix(name, ".png") {
			return unclaimedArtifactError(filepath.Join(gallery, name))
		}
	}
	return nil
}

func unclaimedArtifactError(path string) error {
	return fmt.Errorf("comic artifact %q has no generation identity; choose a different --slug or output directory", path)
}

func (r *Runner) comicIdentity(result GenerateResult, entries []vocab.WordEntry) (string, error) {
	a := r.artist
	// Include the entire generated bundle, not just its title or story text:
	// different panel scripts and character guides require different artwork.
	data := struct {
		Generation     GenerateResult
		Vocabulary     []vocab.WordEntry
		Style          string
		Theme          string
		Language       string
		Script         string
		AspectRatio    string
		StoryPages     int
		GalleryPages   int
		PanelsPerPage  int
		PromptMaxChars int
		DINA4PDF       bool
		UltraRealistic bool
	}{result, entries, a.style, a.theme, a.language, a.script, a.aspectRatio,
		a.storyPages, a.galleryPages, a.panelsPerPage, a.promptMaxChars, a.dina4PDF, a.ultraRealistic}
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("encode comic identity: %w", err)
	}
	return fmt.Sprintf("v1:%x", sha256.Sum256(encoded)), nil
}

func matchComicIdentity(dir, stored, identity string) error {
	if strings.TrimSpace(stored) != identity {
		return fmt.Errorf("comic assets %q belong to a different generation; choose a different --slug or output directory", dir)
	}
	return nil
}
