package comic

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/snonux/comicforge/internal/provider"
	"codeberg.org/snonux/comicforge/internal/vocab"
)

const ttsTodoContent = `# Story Narration - Fallback Placeholder
#
# Comic narration was not produced (missing provider or generation error).
#
# To generate narration manually, run again with a configured TTS provider, or
# use a text-to-speech backend of your choice and save the result as story_narration.mp3.
`

// RunnerConfig holds orchestration settings for the comic pipeline.
type RunnerConfig struct {
	TextProvider          provider.TextProvider
	ImageProvider         provider.ImageProvider
	MainTTSProvider       provider.TTSProvider
	ConclusionTTSProvider provider.TTSProvider
	Prompts               PromptRenderer
	OutputDir             string
	Style                 string
	Theme                 string
	Language              string
	Script                string
	NarratorVoice         string
	Slug                  string
	NarrateEnabled        bool
	UltraRealistic        *bool
	StoryPages            int
	GalleryPages          int
	PanelsPerPage         int
}

// Runner orchestrates the full pipeline.
type Runner struct {
	config      *RunnerConfig
	generator   *Generator
	artist      *Artist
	narrator    *Narrator
	assemblePDF func(outputDir, titleSlug string, imagePaths []string) (string, error)
}

// NewRunner wires together the generator, artist, and narrator.
func NewRunner(cfg *RunnerConfig) *Runner {
	r := &Runner{config: cfg}
	r.assemblePDF = AssembleComicPDF
	if cfg == nil {
		return r
	}

	ultra := pickUltraRealistic()
	if cfg.UltraRealistic != nil {
		ultra = *cfg.UltraRealistic
	}

	r.generator = NewGenerator(&GeneratorConfig{
		TextProvider:  cfg.TextProvider,
		Prompts:       cfg.Prompts,
		Language:      cfg.Language,
		Script:        cfg.Script,
		Theme:         cfg.Theme,
		StoryPages:    cfg.StoryPages,
		PanelsPerPage: cfg.PanelsPerPage,
	})
	r.artist = NewArtist(&ArtistConfig{
		ImageProvider:  cfg.ImageProvider,
		TextProvider:   cfg.TextProvider,
		Prompts:        cfg.Prompts,
		OutputDir:      cfg.OutputDir,
		Style:          cfg.Style,
		Theme:          cfg.Theme,
		Language:       cfg.Language,
		Script:         cfg.Script,
		UltraRealistic: ultra,
		StoryPages:     cfg.StoryPages,
		GalleryPages:   cfg.GalleryPages,
		PanelsPerPage:  cfg.PanelsPerPage,
	})
	r.narrator = NewNarrator(&NarratorConfig{
		TextProvider:       cfg.TextProvider,
		MainProvider:       cfg.MainTTSProvider,
		ConclusionProvider: cfg.ConclusionTTSProvider,
		Prompts:            cfg.Prompts,
		VoiceName:          cfg.NarratorVoice,
		Language:           cfg.Language,
		Script:             cfg.Script,
	})
	return r
}

// Run reads the batch file, generates a story, renders comic pages, and optionally narrates it.
func (r *Runner) Run(ctx context.Context, batchFile string) error {
	if r == nil || r.config == nil {
		return fmt.Errorf("runner config is required")
	}
	if r.generator == nil || r.artist == nil {
		return fmt.Errorf("runner providers are not configured")
	}

	dir := orDefault(r.config.OutputDir, ".")
	entries, err := vocab.ReadVocabularyFile(batchFile)
	if err != nil {
		return fmt.Errorf("failed to read batch file: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("batch file %q contains no words", batchFile)
	}

	fmt.Printf("Generating story for %d words...\n", len(entries))
	result, err := r.generator.GenerateFull(ctx, entries)
	if err != nil {
		return fmt.Errorf("story generation failed: %w", err)
	}

	slug := slugify(result.Title)
	if strings.TrimSpace(r.config.Slug) != "" {
		slug = r.config.Slug
		fmt.Printf("  Comic title: %q (slug forced: %s)\n", result.Title, slug)
	} else if result.Title != "" {
		fmt.Printf("  Comic title: %q (slug: %s)\n", result.Title, slug)
	}

	comicsDir := comicOutputDir(dir, slug)
	if err := os.MkdirAll(comicsDir, 0o755); err != nil {
		return fmt.Errorf("create comics dir %s: %w", comicsDir, err)
	}
	r.artist.outputDir = comicsDir

	if err := r.saveStoryText(result.StoryText, slug, comicsDir); err != nil {
		return err
	}
	if err := r.saveVocabularyFile(result.StoryText, entries, slug, comicsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write vocabulary file: %v\n", err)
	}
	if err := r.saveThemeFile(slug, comicsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write theme file: %v\n", err)
	}

	paths, err := r.artist.DrawComicPages(ctx, result.StoryText, result.Bible, slug, entries, result.PanelScript)
	if err != nil {
		return fmt.Errorf("comic page generation failed: %w", err)
	}
	for _, path := range paths {
		fmt.Printf("Comic page saved: %s\n", path)
	}
	rootDir := filepath.Dir(filepath.Dir(r.artist.outputDir))
	if err := copyGalleryPNGsToComicsGallery(rootDir, r.artist.outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not copy gallery images to comics/gallery: %v\n", err)
	}
	if len(paths) > 0 {
		pdfPath, err := r.assemblePDF(r.artist.outputDir, slug, paths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: PDF assembly failed: %v\n", err)
		} else {
			fmt.Printf("Comic PDF saved: %s\n", pdfPath)
		}
	}

	if !r.config.NarrateEnabled {
		fmt.Println("Narration skipped (enable narration in config to produce audio).")
		return nil
	}
	return r.handleNarration(ctx, result.StoryText, slug, comicsDir)
}

func comicOutputDir(outputRoot, slug string) string {
	root := orDefault(outputRoot, ".")
	if filepath.Base(filepath.Clean(root)) == "comics" {
		return filepath.Join(root, slug)
	}
	return filepath.Join(root, "comics", slug)
}

var _ StoryRunner = (*Runner)(nil)

func (r *Runner) handleNarration(ctx context.Context, storyText, titleSlug, dir string) error {
	if r.narrator == nil || r.narrator.mainProvider == nil || r.narrator.initErr != nil {
		return r.saveTTSPlaceholder(titleSlug, dir)
	}
	mp3Path := filepath.Join(dir, titleSlug+"_narration.mp3")
	fmt.Printf("Generating cinematic narration (voice: %s)...\n", r.narrator.voiceName)
	if err := r.narrator.Narrate(ctx, storyText, mp3Path); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: narration failed: %v\n", err)
		return r.saveTTSPlaceholder(titleSlug, dir)
	}
	fmt.Printf("Narration saved: %s\n", mp3Path)
	return nil
}

func (r *Runner) saveStoryText(text, titleSlug, dir string) error {
	path := filepath.Join(dir, titleSlug+"_story.txt")
	if err := os.WriteFile(path, []byte(text+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write story file: %w", err)
	}
	fmt.Printf("Story saved: %s\n", path)
	return nil
}

func (r *Runner) saveVocabularyFile(storyText string, entries []vocab.WordEntry, titleSlug, dir string) error {
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

func (r *Runner) saveThemeFile(titleSlug, dir string) error {
	path := filepath.Join(dir, titleSlug+"_theme.txt")
	theme := ""
	if r.config != nil {
		theme = r.config.Theme
	}
	if err := os.WriteFile(path, []byte(theme+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write theme file: %w", err)
	}
	fmt.Printf("Theme saved: %s\n", path)
	return nil
}

func (r *Runner) saveTTSPlaceholder(titleSlug, dir string) error {
	path := filepath.Join(dir, titleSlug+"_tts_todo.txt")
	if err := os.WriteFile(path, []byte(ttsTodoContent), 0o644); err != nil {
		return fmt.Errorf("failed to write TTS placeholder: %w", err)
	}
	fmt.Printf("TTS placeholder saved: %s\n", path)
	return nil
}

func copyGalleryPNGsToComicsGallery(outputRoot, comicDir string) error {
	destDir := filepath.Join(outputRoot, "comics", "gallery")
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
