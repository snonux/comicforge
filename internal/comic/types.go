package comic

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"codeberg.org/snonux/comicforge/internal/vocab"
)

const (
	storySystemPromptTemplate     = "story_system.md"
	storyPromptTemplate           = "story_prompt.md"
	storyFullPromptTemplate       = "story_full_prompt.md"
	coverPromptTemplate           = "cover_prompt.md"
	storyPagePromptTemplate       = "story_page_prompt.md"
	galleryPagePromptTemplate     = "gallery_page_prompt.md"
	backCoverPromptTemplate       = "back_cover_prompt.md"
	panelScriptPromptTemplate     = "panel_script_prompt.md"
	bibleSystemTemplate           = "bible_system.md"
	blurbSystemTemplate           = "blurb_system.md"
	renderingRequirementPrompt    = "rendering_requirement.md"
	renderingRequirementEndPrompt = "rendering_requirement_end.md"
)

const (
	storyGeminiModel = "gemini-2.5-flash"

	storyTimeout = 120 * time.Second

	// 8192 tokens for story-only generation (thinking + visible story).
	storyMaxTokens = int32(8192)

	// 16384 total for the combined story+bible call, with thinking capped.
	storyFullMaxTokens = int32(16384)

	// Caps the internal chain-of-thought so visible output is still produced.
	storyFullThinkingBudget = int32(8192)

	storyBibleSeparator = "---CHARACTER GUIDE---"
	storyTitleSeparator = "---COMIC TITLE---"
	storyPanelSeparator = "---PANEL SCRIPT---"

	storyPagesInScript = 5
	storyPanelsPerPage = 4

	pageMaxRetries       = 5
	pageRetryBase        = 15 * time.Second
	comicPageAspectRatio = "16:9"
	comicPromptMaxChars  = 900
	helperTimeout        = 90 * time.Second
	helperMaxTokens      = int32(8192)
	helperRetryPause     = 15 * time.Second
	narratorTimeout      = 3 * time.Minute
	narratorChunkWords   = 100
)

var (
	// ErrMissingProvider is returned when a required provider is not configured.
	ErrMissingProvider = errors.New("missing provider")

	defaultStoryGenres = []string{
		"a warm realistic slice-of-life story",
		"a heartfelt family drama",
		"an exciting science-fiction adventure",
		"a thrilling action-adventure story",
		"a mystery with a surprising twist",
		"a funny comedy with silly misunderstandings",
		"a fantasy quest in a magical world",
		"a spooky but kid-friendly horror story",
		"a space exploration adventure",
		"a superhero origin story",
	}

	realisticStyles = []string{
		"ultra-realistic DSLR photography, cinematic 35mm lens, natural lighting, hyper-detailed textures",
		"cinematic still photography, golden-hour lighting, shallow depth of field, photojournalism quality",
		"hyper-realistic photography, studio-quality lighting, sharp focus, true-to-life colours and textures",
	}

	comicStyles = []string{
		"ultra realistic comic strip with photographic detail and dramatic lighting",
		"classic American comic book with bold ink outlines, halftone dots, and primary colors",
		"Japanese manga with clean linework, expressive eyes, and speed lines",
		"retro 1960s pop art in the style of Roy Lichtenstein with thick outlines and Ben-Day dots",
		"watercolor illustration with soft washes, delicate linework, and pastel tones",
		"European bande dessinée with detailed backgrounds, clear lines, and rich flat colors",
		"noir black-and-white graphic novel with heavy shadows and high contrast",
		"children's picture book with bright, friendly illustrations and thick outlines",
		"painterly oil-on-canvas comic with loose brushwork and vivid impressionist colors",
		"cyberpunk neon art with glowing outlines, dark backgrounds, and electric accent colors",
	}

	galleryPoses = []string{
		"extreme close-up portrait: face and shoulders filling the entire frame, dramatic three-quarter lighting, intense gaze directly at the viewer, fine detail on eyes and expression",
		"dynamic action pose: full body, low-angle shot looking up at the main character against the sky or setting backdrop, confident stance, hair and clothing caught in motion",
		"atmospheric mid-shot: waist-up, the main character silhouetted or lit by the ambient environment (bioluminescence, sunset, neon glow), looking off into the distance with a sense of wonder or resolve",
		"profile close-up: side view of face and upper body, soft rim lighting tracing the jawline and hair, contemplative expression, rich background bokeh",
		"power stance full-body: the main character seen from the front at eye level, arms relaxed but ready, environment filling the frame behind them, golden-hour or dramatic storm light",
	}
)

// PromptRenderer renders external or embedded prompt templates.
type PromptRenderer interface {
	RenderPrompt(name string, data any) (string, error)
}

// WordEntry is the vocabulary input consumed by the comic pipeline.
type WordEntry = vocab.WordEntry

// GenerateResult holds the story text, character bible, comic title, and panel script.
type GenerateResult struct {
	StoryText   string
	Bible       string
	Title       string
	PanelScript [][]string
}

// StoryRunner runs the comic generation pipeline from a vocabulary file.
type StoryRunner interface {
	Run(ctx context.Context, batchFile string) error
}

func pickStoryGenre(genres []string) string {
	if len(genres) == 0 {
		genres = defaultStoryGenres
	}
	if len(genres) == 1 {
		return genres[0]
	}
	if rand.Float64() < 0.4 {
		return genres[rand.IntN(min(2, len(genres)))]
	}
	if len(genres) <= 2 {
		return genres[rand.IntN(len(genres))]
	}
	return genres[2+rand.IntN(len(genres)-2)]
}

func resolveGenre(theme string, genres []string) string {
	if strings.TrimSpace(theme) != "" {
		return theme
	}
	return pickStoryGenre(genres)
}

func pickUltraRealistic() bool {
	return rand.Float64() < 0.5
}

func pickStyle(styles []string, ultraRealistic bool) string {
	if len(styles) == 0 {
		if ultraRealistic {
			styles = realisticStyles
		} else {
			styles = comicStyles
		}
	}
	if ultraRealistic {
		return styles[rand.IntN(len(styles))]
	}
	if len(styles) == 1 {
		return styles[0]
	}
	if rand.Float64() < 0.9 {
		return styles[0]
	}
	return styles[1+rand.IntN(len(styles)-1)]
}

func parseGenerateResult(combined string) GenerateResult {
	bibleIdx := strings.Index(combined, storyBibleSeparator)
	if bibleIdx < 0 {
		return GenerateResult{StoryText: strings.TrimSpace(combined)}
	}

	story := strings.TrimSpace(combined[:bibleIdx])
	afterBible := strings.TrimSpace(combined[bibleIdx+len(storyBibleSeparator):])

	titleIdx := strings.Index(afterBible, storyTitleSeparator)
	if titleIdx < 0 {
		return GenerateResult{StoryText: story, Bible: strings.TrimSpace(afterBible)}
	}

	bible := strings.TrimSpace(afterBible[:titleIdx])
	afterTitle := strings.TrimSpace(afterBible[titleIdx+len(storyTitleSeparator):])

	panelIdx := strings.Index(afterTitle, storyPanelSeparator)
	title := afterTitle
	panelText := ""
	if panelIdx >= 0 {
		title = afterTitle[:panelIdx]
		panelText = afterTitle[panelIdx+len(storyPanelSeparator):]
	}
	if nl := strings.IndexByte(title, '\n'); nl >= 0 {
		title = title[:nl]
	}
	title = strings.TrimSpace(title)

	return GenerateResult{
		StoryText:   story,
		Bible:       bible,
		Title:       title,
		PanelScript: parsePanelScript(panelText),
	}
}

func parsePanelScript(text string) [][]string {
	script := make([][]string, storyPagesInScript)
	for i := range script {
		script[i] = make([]string, storyPanelsPerPage)
	}

	panelIndex := map[byte]int{'A': 0, 'B': 1, 'C': 2, 'D': 3}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 7 || line[0] != 'P' || line[2] != '-' || line[4] != ':' {
			continue
		}
		page := int(line[1] - '1')
		panel, ok := panelIndex[line[3]]
		if !ok || page < 0 || page >= storyPagesInScript {
			continue
		}
		script[page][panel] = strings.TrimSpace(line[5:])
	}
	return script
}

func buildWordList(entries []WordEntry, header string) string {
	var sb strings.Builder
	sb.WriteString(header)
	if header != "" {
		sb.WriteString("Words to include:\n")
	}
	for i, entry := range entries {
		word := strings.TrimSpace(entry.Word)
		if word == "" {
			word = strings.TrimSpace(entry.Translation)
		}
		if entry.Translation != "" && entry.Word != "" {
			sb.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, entry.Word, entry.Translation))
			continue
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, word))
	}
	return sb.String()
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, timeout)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
