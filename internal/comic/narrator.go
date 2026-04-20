package comic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"codeberg.org/snonux/comicforge/internal/provider"
)

var lookPath = exec.LookPath

// NarratorConfig configures narration generation.
type NarratorConfig struct {
	TextProvider       provider.TextProvider
	MainProvider       provider.TTSProvider
	ConclusionProvider provider.TTSProvider
	Prompts            PromptRenderer
	VoiceName          string
	Language           string
	Script             string
}

// Narrator generates intro, story, and conclusion narration.
type Narrator struct {
	textProvider       provider.TextProvider
	mainProvider       provider.TTSProvider
	conclusionProvider provider.TTSProvider
	prompts            PromptRenderer
	voiceName          string
	language           string
	script             string
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
	n.voiceName = cfg.VoiceName
	n.language = orDefault(cfg.Language, n.language)
	n.script = orDefault(cfg.Script, n.script)
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
	} else if err := concatenateMP3s(allPaths, combinedPath, tmpDir); err != nil {
		return err
	}
	return convertToStereo(combinedPath, outputFile)
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
	chunks := splitIntoNarrationChunks(storyText, narratorChunkWords)
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
	if err := mixAmbientMusic(introRaw, introWithMusic, tmpDir); err != nil {
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

	chunks := splitIntoNarrationChunks(conclusion, narratorChunkWords)
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
	} else if err := concatenateMP3s(paths, conclusionNarration, tmpDir); err != nil {
		fmt.Printf("    Warning: conclusion concat failed: %v\n", err)
		return paths[len(paths)-1], true
	}

	conclusionWithMusic := filepath.Join(tmpDir, "conclusion_with_music.mp3")
	if err := mixAmbientMusic(conclusionNarration, conclusionWithMusic, tmpDir); err != nil {
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
	cinematicInstruction = `You are a dramatic cinematic narrator performing a story written in BULGARIAN.
IMPORTANT: This text is in the BULGARIAN language — NOT Russian, NOT Serbian, NOT any other Slavic language.
Pronounce every word using authentic BULGARIAN phonology and accent. Bulgarian vowels are clear and distinct;
do not apply Russian stress patterns or Russian vowel reduction. The letter 'ъ' in Bulgarian is a mid-central
vowel (like the 'u' in "but"), not the Russian reduced schwa.
Deliver this as a professional movie trailer narrator would: deep, resonant, and commanding.
Use long dramatic pauses before key moments. Build tension with slower, deliberate pacing,
then accelerate through action. Drop your voice low and gravelly for mysterious or serious
passages; let warmth and energy rise for joyful or triumphant ones. Breathe life into every
sentence — this should sound like an epic Bulgarian film, not a reading exercise.

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

func mixAmbientMusic(narrationFile, outputFile, tmpDir string) error {
	ffmpegPath, err := lookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found")
	}

	musicPath := filepath.Join(tmpDir, "ambient_pad.mp3")
	if err := generateAmbientPad(ffmpegPath, musicPath); err != nil {
		return err
	}

	cmd := exec.Command(ffmpegPath,
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-i", narrationFile,
		"-i", musicPath,
		"-filter_complex", "[0:a][1:a]amix=inputs=2:weights=1 0.3:duration=first[aout]",
		"-map", "[aout]",
		"-ac", "2",
		"-codec:a", "libmp3lame", "-q:a", "2",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("background music mix failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func generateAmbientPad(ffmpegPath, outputFile string) error {
	droneExpr := "0.04*sin(65*2*PI*t)+0.03*sin(98*2*PI*t)+0.02*sin(130*2*PI*t)"
	cmd := exec.Command(ffmpegPath,
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("aevalsrc=%s:sample_rate=44100", droneExpr),
		"-f", "lavfi", "-i", "anoisesrc=color=pink:amplitude=0.008",
		"-filter_complex", "[0:a][1:a]amix=inputs=2:duration=first[mixed];[mixed]afade=t=in:st=0:d=4[aout]",
		"-map", "[aout]",
		"-t", "300",
		"-ac", "2",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ambient pad generation failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func concatenateMP3s(chunkPaths []string, outputFile, tmpDir string) error {
	ffmpegPath, err := lookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found — required for multi-chunk narration: %w", err)
	}

	listPath := filepath.Join(tmpDir, "concat_list.txt")
	var sb strings.Builder
	for _, path := range chunkPaths {
		fmt.Fprintf(&sb, "file '%s'\n", path)
	}
	if err := os.WriteFile(listPath, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("write concat list: %w", err)
	}

	cmd := exec.Command(ffmpegPath,
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-y",
		"-f", "concat", "-safe", "0",
		"-i", listPath,
		"-codec:a", "copy",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg concat failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func convertToStereo(inputFile, outputFile string) error {
	ffmpegPath, err := lookPath("ffmpeg")
	if err != nil {
		fmt.Println("    Warning: ffmpeg not found, narration will be mono")
		return copyFile(inputFile, outputFile)
	}

	cmd := exec.Command(ffmpegPath,
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-y",
		"-i", inputFile,
		"-ac", "2",
		"-codec:a", "libmp3lame", "-q:a", "2",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg stereo conversion failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
