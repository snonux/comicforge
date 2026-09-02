package comic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/snonux/comicforge/internal/provider"
)

// NarratorConfig configures narration generation.
type NarratorConfig struct {
	TextProvider       provider.TextProvider
	MainProvider       provider.TTSProvider
	ConclusionProvider provider.TTSProvider
	Prompts            PromptRenderer
	AudioMixer         AudioMixer
	VoiceName          string
	Language           string
	Script             string
	ChunkWords         int
}

// Narrator generates intro, story, and conclusion narration.
type Narrator struct {
	textProvider       provider.TextProvider
	mainProvider       provider.TTSProvider
	conclusionProvider provider.TTSProvider
	prompts            PromptRenderer
	mixer              AudioMixer
	voiceName          string
	language           string
	script             string
	chunkWords         int
	initErr            error
}

// NewNarrator creates a narration pipeline.
func NewNarrator(cfg *NarratorConfig) *Narrator {
	n := &Narrator{
		language: "Bulgarian",
		script:   "Cyrillic",
	}
	if cfg == nil {
		n.initErr = fmt.Errorf("narrator config is required")
		return n
	}
	n.textProvider = cfg.TextProvider
	n.mainProvider = cfg.MainProvider
	n.conclusionProvider = cfg.ConclusionProvider
	n.prompts = cfg.Prompts
	n.mixer = cfg.AudioMixer
	if n.mixer == nil {
		n.mixer = NewFFmpegAudioMixer()
	}
	n.voiceName = cfg.VoiceName
	n.language = orDefault(cfg.Language, n.language)
	n.script = orDefault(cfg.Script, n.script)
	n.chunkWords = normalizePositive(cfg.ChunkWords, narratorChunkWords)
	if n.conclusionProvider == nil {
		n.conclusionProvider = n.mainProvider
	}
	if n.mainProvider == nil {
		n.initErr = fmt.Errorf("%w: main TTS provider", ErrMissingProvider)
	}
	if n.prompts == nil {
		n.initErr = errorsJoin(n.initErr, fmt.Errorf("%w: prompt renderer", ErrMissingProvider))
	}
	return n
}

// Narrate generates a cinematic MP3 narration of storyText and saves it to outputFile.
func (n *Narrator) Narrate(ctx context.Context, storyText, outputFile string) error {
	if err := n.ready(); err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "comicforge-narration-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	var allPaths []string
	if introPath, ok := n.narrateIntro(ctx, storyText, tmpDir); ok {
		allPaths = append(allPaths, introPath)
	}
	chunkPaths, err := n.narrateMainStory(ctx, storyText, tmpDir)
	if err != nil {
		return err
	}
	allPaths = append(allPaths, chunkPaths...)
	if conclusionPath, ok := n.narrateConclusion(ctx, storyText, tmpDir); ok {
		allPaths = append(allPaths, conclusionPath)
	}

	combinedPath := filepath.Join(tmpDir, "combined.mp3")
	if len(allPaths) == 1 {
		combinedPath = allPaths[0]
	} else if err := n.mixer.ConcatenateMP3s(ctx, allPaths, combinedPath, tmpDir); err != nil {
		return err
	}
	return n.mixer.ConvertToStereo(ctx, combinedPath, outputFile)
}

func (n *Narrator) ready() error {
	if n == nil {
		return fmt.Errorf("narrator is nil")
	}
	if n.initErr != nil {
		return n.initErr
	}
	return nil
}

func (n *Narrator) narrateMainStory(ctx context.Context, storyText, tmpDir string) ([]string, error) {
	chunks := splitIntoNarrationChunks(storyText, n.chunkWords)
	fmt.Printf("    Splitting narration into %d chunks for consistent voice quality...\n", len(chunks))

	var paths []string
	for i, chunk := range chunks {
		path := filepath.Join(tmpDir, fmt.Sprintf("chunk_%03d.mp3", i+1))
		fmt.Printf("    Narrating chunk %d/%d...\n", i+1, len(chunks))
		if err := n.narrateChunkWith(ctx, n.mainProvider, cinematicInstruction+chunk, path); err != nil {
			return nil, fmt.Errorf("narrate chunk %d: %w", i+1, err)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func (n *Narrator) narrateIntro(ctx context.Context, storyText, tmpDir string) (string, bool) {
	intro := n.buildIntro(ctx, storyText)
	if intro == "" {
		return "", false
	}
	introRaw := filepath.Join(tmpDir, "intro_narration.mp3")
	if err := n.narrateChunkWith(ctx, n.conclusionProvider, cinematicInstruction+intro, introRaw); err != nil {
		fmt.Printf("    Warning: intro narration failed: %v\n", err)
		return "", false
	}

	introWithMusic := filepath.Join(tmpDir, "intro_with_music.mp3")
	if err := n.mixer.MixAmbientMusic(ctx, introRaw, introWithMusic, tmpDir); err != nil {
		fmt.Printf("    Warning: intro music mix failed (%v) — using narration only\n", err)
		return introRaw, true
	}
	return introWithMusic, true
}

func (n *Narrator) buildIntro(ctx context.Context, storyText string) string {
	return n.buildTeaser(ctx, introSystemTemplate, storyText)
}

func (n *Narrator) narrateConclusion(ctx context.Context, storyText, tmpDir string) (string, bool) {
	conclusion := n.buildConclusion(ctx, storyText)
	if conclusion == "" {
		return "", false
	}

	chunks := splitIntoNarrationChunks(conclusion, n.chunkWords)
	var paths []string
	for i, chunk := range chunks {
		path := filepath.Join(tmpDir, fmt.Sprintf("conclusion_%03d.mp3", i+1))
		if err := n.narrateChunkWith(ctx, n.conclusionProvider, cinematicInstruction+chunk, path); err != nil {
			fmt.Printf("    Warning: conclusion narration failed: %v\n", err)
			return "", false
		}
		paths = append(paths, path)
	}

	conclusionNarration := filepath.Join(tmpDir, "conclusion_narration.mp3")
	if len(paths) == 1 {
		conclusionNarration = paths[0]
	} else if err := n.mixer.ConcatenateMP3s(ctx, paths, conclusionNarration, tmpDir); err != nil {
		fmt.Printf("    Warning: conclusion concat failed: %v\n", err)
		return "", false
	}

	conclusionWithMusic := filepath.Join(tmpDir, "conclusion_with_music.mp3")
	if err := n.mixer.MixAmbientMusic(ctx, conclusionNarration, conclusionWithMusic, tmpDir); err != nil {
		fmt.Printf("    Warning: background music mix failed (%v) — using narration only\n", err)
		return conclusionNarration, true
	}
	return conclusionWithMusic, true
}

func (n *Narrator) buildConclusion(ctx context.Context, storyText string) string {
	return n.buildTeaser(ctx, conclusionSystemTemplate, storyText)
}

func (n *Narrator) buildTeaser(ctx context.Context, templateName, storyText string) string {
	if n.textProvider == nil {
		return ""
	}
	systemPrompt, err := n.prompts.RenderPrompt(templateName, map[string]any{
		"StoryText":    storyText,
		"Language":     n.language,
		"LanguageName": localizedLanguageName(n.language, n.script),
		"Script":       n.script,
		"ScriptName":   localizedScriptName(n.script),
	})
	if err != nil {
		fmt.Printf("    Warning: text prompt render failed: %v\n", err)
		return ""
	}
	prompt := systemPrompt + "\n\n" + storyText
	callCtx, cancel := withTimeout(ctx, helperTimeout)
	defer cancel()
	text, err := n.textProvider.GenerateText(callCtx, prompt)
	if err != nil {
		fmt.Printf("    Warning: teaser generation failed: %v\n", err)
		return ""
	}
	teaser := strings.TrimSpace(text)
	if err := validateTextScript("narration teaser", teaser, n.script); err != nil {
		fmt.Printf("    Warning: teaser validation failed: %v\n", err)
		return ""
	}
	return teaser
}

func (n *Narrator) narrateChunkWith(ctx context.Context, provider provider.TTSProvider, text, outputFile string) error {
	callCtx, cancel := withTimeout(ctx, narratorTimeout)
	defer cancel()
	return provider.GenerateAudio(callCtx, text, outputFile)
}

const (
	cinematicInstruction = `You are a dramatic cinematic narrator performing the provided story.
Respect the story's language exactly. Do not translate it and do not drift into another language or accent family.
Deliver this as a professional movie trailer narrator would: deep, resonant, and commanding.
Use long dramatic pauses before key moments. Build tension with slower, deliberate pacing,
then accelerate through action. Drop your voice low and gravelly for mysterious or serious
passages; let warmth and energy rise for joyful or triumphant ones. Breathe life into every
sentence — this should sound like an epic film, not a reading exercise.

`

	introSystemTemplate      = "narrator_intro_system.md"
	conclusionSystemTemplate = "narrator_conclusion_system.md"
)

func splitIntoNarrationChunks(text string, targetWords int) []string {
	paragraphs := splitParagraphs(text)
	if len(paragraphs) == 0 {
		return []string{strings.TrimSpace(text)}
	}

	var chunks []string
	var current strings.Builder
	currentWords := 0
	for _, paragraph := range paragraphs {
		paraWords := len(strings.Fields(paragraph))
		if currentWords > 0 && currentWords+paraWords > targetWords {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
			currentWords = 0
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(paragraph)
		currentWords += paraWords
	}
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}
