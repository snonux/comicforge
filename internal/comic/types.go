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
	storyTimeout = 120 * time.Second

	storyBibleSeparator = "---CHARACTER GUIDE---"
	storyTitleSeparator = "---COMIC TITLE---"
	storyPanelSeparator = "---PANEL SCRIPT---"

	defaultStoryPagesInScript = 5
	defaultStoryPanelsPerPage = 4

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
		"close-up portrait of face and shoulders, dramatic three-quarter lighting, clear gaze toward the viewer, sharp eyes and expressive face",
		"dynamic full-body action pose, low camera angle looking up at the hero against the sky or environment, confident stance, hair and clothing in motion",
		"atmospheric medium shot, the hero rim-lit by the environment (bioluminescence, sunset, neon), looking into the distance with wonder or resolve",
		"profile close-up of face and upper body, soft rim light along jaw and hair, thoughtful expression, softly blurred background",
		"confident full-body stance from eye level, arms relaxed but ready, environment filling the frame, golden or dramatic light",
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
	return parseGenerateResultWithDimensions(combined, defaultStoryPagesInScript, defaultStoryPanelsPerPage)
}

func parseGenerateResultWithDimensions(combined string, storyPages, panelsPerPage int) GenerateResult {
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
		PanelScript: parsePanelScript(panelText, storyPages, panelsPerPage),
	}
}

func parsePanelScript(text string, storyPages, panelsPerPage int) [][]string {
	storyPages = normalizePositive(storyPages, defaultStoryPagesInScript)
	panelsPerPage = normalizePositive(panelsPerPage, defaultStoryPanelsPerPage)

	script := make([][]string, storyPages)
	for i := range script {
		script[i] = make([]string, panelsPerPage)
	}

	panelIndex := make(map[byte]int, panelsPerPage)
	for i := 0; i < panelsPerPage; i++ {
		panelIndex[byte('A'+i)] = i
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 7 || line[0] != 'P' || line[2] != '-' || line[4] != ':' {
			continue
		}
		page := int(line[1] - '1')
		panel, ok := panelIndex[line[3]]
		if !ok || page < 0 || page >= storyPages {
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
		sb.WriteString("Required words:\n")
	}
	for _, entry := range entries {
		word := strings.TrimSpace(entry.Word)
		if word == "" {
			word = strings.TrimSpace(entry.Translation)
		}
		fmt.Fprintf(&sb, "- %s\n", word)
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

func normalizePositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func panelLabel(idx int) string {
	if idx < 0 {
		return ""
	}
	return string(rune('A' + idx))
}

func panelLabelsText(count int) string {
	count = normalizePositive(count, defaultStoryPanelsPerPage)
	labels := make([]string, 0, count)
	for i := 0; i < count; i++ {
		labels = append(labels, panelLabel(i))
	}
	return joinEnglishList(labels)
}

func requiredDialoguePanels(count int) int {
	count = normalizePositive(count, defaultStoryPanelsPerPage)
	if count <= 1 {
		return 1
	}
	return (count + 1) / 2
}

func joinEnglishList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " or " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
	}
}
