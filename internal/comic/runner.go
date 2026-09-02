package comic

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	StyleMode             string
	Theme                 string
	Language              string
	Script                string
	NarratorVoice         string
	Slug                  string
	NarrateEnabled        bool
	UltraRealistic        *bool
	RealisticWeight       float64
	Styles                map[string][]string
	AspectRatio           string
	PromptMaxChars        int
	PageMaxRetries        int
	PageRetryBase         time.Duration
	ChunkWords            int
	StoryPages            int
	GalleryPages          int
	PanelsPerPage         int
	PDF                   PDFAssembleOptions
}

// Runner orchestrates the full pipeline.
type Runner struct {
	config      *RunnerConfig
	generator   *Generator
	artist      *Artist
	narrator    *Narrator
	assemblePDF func(outputDir, titleSlug string, imagePaths []string, opts PDFAssembleOptions) (string, error)
}

// NewRunner wires together the generator, artist, and narrator.
func NewRunner(cfg *RunnerConfig) *Runner {
	r := &Runner{config: cfg}
	r.assemblePDF = AssembleComicPDF
	if cfg == nil {
		return r
	}

	styleMode := strings.TrimSpace(cfg.StyleMode)
	if styleMode == "" {
		styleMode = pickStyleMode(cfg.UltraRealistic, cfg.Styles, cfg.RealisticWeight)
	}
	ultra := styleMode == styleModeRealistic

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
		StyleMode:      styleMode,
		Styles:         cfg.Styles,
		Theme:          cfg.Theme,
		AspectRatio:    cfg.AspectRatio,
		Language:       cfg.Language,
		Script:         cfg.Script,
		UltraRealistic: ultra,
		PromptMaxChars: cfg.PromptMaxChars,
		PageMaxRetries: cfg.PageMaxRetries,
		PageRetryBase:  cfg.PageRetryBase,
		StoryPages:     cfg.StoryPages,
		GalleryPages:   cfg.GalleryPages,
		PanelsPerPage:  cfg.PanelsPerPage,
		DINA4PDF:       IsDINA4ClassPDFPresentation(cfg.PDF.Presentation),
	})
	r.narrator = NewNarrator(&NarratorConfig{
		TextProvider:       cfg.TextProvider,
		MainProvider:       cfg.MainTTSProvider,
		ConclusionProvider: cfg.ConclusionTTSProvider,
		Prompts:            cfg.Prompts,
		VoiceName:          cfg.NarratorVoice,
		Language:           cfg.Language,
		Script:             cfg.Script,
		ChunkWords:         cfg.ChunkWords,
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

	assetsDir := comicAssetsDir(dir, slug)
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return fmt.Errorf("create comics assets dir %s: %w", assetsDir, err)
	}
	if err := os.MkdirAll(comicsPDFDir(dir), 0o755); err != nil {
		return fmt.Errorf("create comics pdf dir %s: %w", comicsPDFDir(dir), err)
	}
	r.artist.outputDir = assetsDir

	if err := r.saveStoryText(result.StoryText, slug, assetsDir); err != nil {
		return err
	}
	if err := r.saveVocabularyFile(result.StoryText, entries, slug, assetsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write vocabulary file: %v\n", err)
	}
	if err := r.saveThemeFile(slug, assetsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write theme file: %v\n", err)
	}

	paths, err := r.artist.DrawComicPages(ctx, result.StoryText, result.Bible, slug, entries, result.PanelScript)
	if err != nil {
		return fmt.Errorf("comic page generation failed: %w", err)
	}
	for _, path := range paths {
		fmt.Printf("Comic page saved: %s\n", path)
	}
	rootDir := comicsRootDir(dir)
	if err := copyGalleryPNGsToComicsGallery(rootDir, r.artist.outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not copy gallery images to comics/gallery: %v\n", err)
	}
	if len(paths) > 0 {
		pdfPath, err := r.assemblePDF(comicsPDFDir(dir), slug, paths, r.config.PDF)
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
	return r.handleNarration(ctx, result.StoryText, slug, assetsDir)
}

// RunPrompt renders a single image from a direct user prompt.
func (r *Runner) RunPrompt(ctx context.Context, prompt string) error {
	if r == nil || r.config == nil {
		return fmt.Errorf("runner config is required")
	}
	if r.artist == nil {
		return fmt.Errorf("runner providers are not configured")
	}
	if err := r.artist.ready(); err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	dir := orDefault(r.config.OutputDir, ".")
	slug, err := r.manualPromptSlug(ctx, prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not generate prompt title: %v\n", err)
	}

	assetsDir := comicAssetsDir(dir, slug)
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return fmt.Errorf("create comics assets dir %s: %w", assetsDir, err)
	}

	r.artist.outputDir = assetsDir
	outputFile := filepath.Join(assetsDir, "prompt.png")
	if err := r.artist.generatePromptImage(ctx, prompt, outputFile); err != nil {
		return fmt.Errorf("prompt image generation failed: %w", err)
	}
	fmt.Printf("Prompt image saved: %s\n", outputFile)
	return nil
}

func (r *Runner) manualPromptSlug(ctx context.Context, prompt string) (string, error) {
	if forced := strings.TrimSpace(r.config.Slug); forced != "" {
		return forced, nil
	}
	fallback := shortPromptSlugFromText(prompt)
	if r.config.TextProvider == nil || r.config.Prompts == nil {
		return fallback, nil
	}
	title, err := r.generateManualPromptTitle(ctx, prompt)
	if err != nil {
		return fallback, err
	}
	if titleSlug := shortPromptSlugFromText(title); titleSlug != "manual-prompt" {
		fmt.Printf("  Prompt title: %q (slug: %s)\n", title, titleSlug)
		return titleSlug, nil
	}
	return fallback, nil
}

func (r *Runner) generateManualPromptTitle(ctx context.Context, prompt string) (string, error) {
	renderedPrompt, err := r.config.Prompts.RenderPrompt(manualTitlePromptTemplate, map[string]any{
		"Prompt": prompt,
	})
	if err != nil {
		return "", fmt.Errorf("render manual title prompt: %w", err)
	}
	callCtx, cancel := withTimeout(ctx, helperTimeout)
	defer cancel()
	text, err := r.config.TextProvider.GenerateText(callCtx, renderedPrompt)
	if err != nil {
		return "", fmt.Errorf("generate manual title: %w", err)
	}
	title := cleanManualPromptTitle(text)
	if title == "" {
		return "", fmt.Errorf("manual title was empty")
	}
	if err := validateNoPromptLeakage("manual prompt title", title); err != nil {
		return "", err
	}
	return title, nil
}

func cleanManualPromptTitle(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)
	text = strings.Trim(text, `"'“”‘’.:;`)
	return strings.Join(strings.Fields(text), " ")
}

func comicsRootDir(outputRoot string) string {
	root := orDefault(outputRoot, ".")
	if filepath.Base(filepath.Clean(root)) == "comics" {
		return root
	}
	return filepath.Join(root, "comics")
}

func comicAssetsDir(outputRoot, slug string) string {
	return filepath.Join(comicsRootDir(outputRoot), "assets", slug)
}

func comicsPDFDir(outputRoot string) string {
	return filepath.Join(comicsRootDir(outputRoot), "PDF")
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
