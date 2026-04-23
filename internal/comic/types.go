package comic

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"codeberg.org/snonux/comicforge/internal/vocab"
)

const (
	storySystemPromptTemplate     = "story_system.md"
	storyPromptTemplate           = "story_prompt.md"
	storyFullPromptTemplate       = "story_full_prompt.md"
	manualPromptTemplate          = "manual_prompt.md"
	manualTitlePromptTemplate     = "manual_title_prompt.md"
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

	defaultRealisticStyles = []string{
		"ultra-realistic DSLR photography, cinematic 35mm lens, natural lighting, hyper-detailed textures",
		"cinematic still photography, golden-hour lighting, shallow depth of field, photojournalism quality",
		"hyper-realistic photography, studio-quality lighting, sharp focus, true-to-life colours and textures",
	}

	defaultComicStyles = []string{
		"classic American comic book with bold ink outlines, halftone dots, and primary colors",
		"retro 1960s pop art in the style of Roy Lichtenstein with thick outlines and Ben-Day dots",
		"European bande dessinée with detailed backgrounds, clear lines, and rich flat colors",
		"noir black-and-white graphic novel with heavy shadows and high contrast",
		"children's picture book with bright, friendly illustrations and thick outlines",
		"painterly oil-on-canvas comic with loose brushwork and vivid impressionist colors",
	}

	defaultCartoonStyles = []string{
		"extremely cartoonish 1930s rubber-hose funny-animal comic style, original mouse-and-duck-era mascot energy, pie-cut eyes, button noses or beaks, white gloves, oversized shoes, noodle arms, squash-and-stretch bodies, round heads, huge expressions, flat candy colors, simple gag staging, absolutely no realism",
		"vintage theatrical funny-animal cartoon comic style, redesign people as original anthropomorphic mascot characters with beaks or round snouts, pie-cut eyes, white gloves, oversized shoes, elastic limbs, cheerful slapstick poses, bold clean outlines, bright flat backgrounds, no realistic anatomy",
		"old black-and-white short cartoon style translated into comic panels, original rubber-hose mascot characters, pie eyes, rubber limbs, bouncing motion arcs, simple rounded shapes, comic mischief, exaggerated face acting, hand-inked animation cels",
	}

	defaultAction90sStyles = []string{
		"1990s superhero comic splash-page style like a bold caped hero punching a masked villain in a city, huge muscles, spandex costumes, flowing cape, explosive impact burst, speed lines, smoke, rubble, halftone print texture, heavy black inks, saturated primary colors",
		"classic 90s action superhero cover style, oversized yellow masthead energy, caped spandex hero, armored villain, clenched fists, dramatic foreshortening, city skyline, explosions, speech balloons, thick ink outlines, Ben-Day dots",
		"extreme 1990s superhero action comic style, heroic anatomy, masked villain, motion streaks, lightning impact shapes, cross-hatching, smoky city destruction, glossy saturated colors, bold comic-book lettering spaces",
	}

	defaultMangaStyles = []string{
		"true monochrome Japanese shonen manga page, black ink only with grey screentone, no western color rendering, large expressive anime eyes, sharp spiky hair silhouettes, sweat drops, impact bursts, speed-line backgrounds, exaggerated emotion close-ups, dynamic diagonal manga panel language",
		"authentic serialized manga action art, crisp black linework, dense screentone gradients, white paper highlights, hand-drawn Japanese comic energy, chibi reaction inset where appropriate, dramatic sound-effect shapes, cinematic close-ups, no painterly western comic coloring",
		"modern black-and-white manga fantasy style, anime character proportions, big eyes, angular hair, thin-to-thick ink strokes, patterned screentones, motion streaks, expressive face acting, explosive action frames, minimal color accents only if unavoidable",
	}

	defaultHorrorStyles = []string{
		"vintage horror comic style, eerie shadows, gothic atmosphere, unsettling monsters, dramatic candlelit contrast, suspenseful panel staging",
		"creepy supernatural horror comic style, fog, haunted architecture, anxious expressions, heavy inks, sickly green and crimson accents",
		"old anthology horror comic style, graveyard mist, warped shadows, tense faces, scratchy ink texture, ominous colored lighting",
	}

	defaultWatercolorStyles = []string{
		"pure transparent watercolor storybook comic style, visible cold-press paper grain, watery pigment blooms, uneven wash edges, granulation, soft bleeding colors, pale layered washes, loose pencil underdrawing, no digital airbrush, no oil-paint texture, no glossy comic coloring",
		"hand-painted watercolor children's adventure style, translucent pastel washes, wet-on-wet skies, dry-brush texture, white paper highlights, delicate ink-and-pencil outlines, puddled pigment edges, airy negative space, gentle low-contrast rendering",
		"storybook watercolor fantasy comic style, washed paper texture, transparent layered color, soft backruns and blossoms, feathered brush edges, muted handmade palette, tender expressions, dreamy illustrated settings, visibly painted on paper",
	}
	defaultPulpStyles = []string{
		"retro pulp adventure cover style, painted poster energy, dramatic rim light, saturated inks, worn paper grain, bold composition, vintage print imperfections",
		"mid-century pulp sci-fi comic style, chromatic airbrushed highlights, atomic-age props, bold diagonals, distressed halftone texture, high adventure mood",
	}
	defaultMechaStyles = []string{
		"high-detail mecha comic style, hard-surface armor plates, mechanical joints, cockpit glass reflections, technical line art, industrial lighting",
		"anime mecha action page style, towering robots, dynamic perspective, panel-like motion energy, sparks and smoke, crisp cel-shaded rendering",
	}
	defaultPixelArtStyles = []string{
		"16-bit pixel-art comic scene, limited palette, clean pixel clusters, deliberate dithering, retro game-era perspective, sharp pixel edges",
		"retro pixel-art storyboard style, chunky sprites, high-contrast palette ramps, tiled-environment texture, no anti-aliasing, classic console look",
	}
	defaultInkWashStyles = []string{
		"ink-wash brush painting style, sumi-e inspired strokes, expressive black ink gradients, controlled bleed on paper, minimal but dramatic composition",
		"monochrome ink-and-wash comic style, calligraphic brush lines, soft gray washes, negative-space storytelling, textured rice-paper feel",
	}
	defaultClayStyles = []string{
		"stop-motion clay diorama style, hand-sculpted clay characters, thumbprint texture, miniature set lighting, tactile handcrafted look",
		"plasticine comic tableau style, sculpted figures, soft studio shadows, practical miniature props, playful clay surface detail",
	}

	realisticStyles  = defaultRealisticStyles
	comicStyles      = defaultComicStyles
	cartoonStyles    = defaultCartoonStyles
	action90sStyles  = defaultAction90sStyles
	mangaStyles      = defaultMangaStyles
	horrorStyles     = defaultHorrorStyles
	watercolorStyles = defaultWatercolorStyles
	pulpStyles       = defaultPulpStyles
	mechaStyles      = defaultMechaStyles
	pixelArtStyles   = defaultPixelArtStyles
	inkWashStyles    = defaultInkWashStyles
	clayStyles       = defaultClayStyles

	galleryPoses = []string{
		"close-up portrait of face and shoulders, dramatic three-quarter lighting, clear gaze toward the viewer, sharp eyes and expressive face",
		"dynamic full-body action pose, low camera angle looking up at the hero against the sky or environment, confident stance, hair and clothing in motion",
		"atmospheric medium shot, the hero rim-lit by the environment (bioluminescence, sunset, neon), looking into the distance with wonder or resolve",
		"profile close-up of face and upper body, soft rim light along jaw and hair, thoughtful expression, softly blurred background",
		"confident full-body stance from eye level, arms relaxed but ready, environment filling the frame, golden or dramatic light",
	}
)

const (
	styleModeRealistic  = "realistic"
	styleModeComic      = "comic"
	styleModeCartoon    = "cartoon"
	styleModeAction90s  = "action_90s"
	styleModeManga      = "manga"
	styleModeHorror     = "horror"
	styleModeWatercolor = "watercolor"
	styleModePulp       = "pulp"
	styleModeMecha      = "mecha"
	styleModePixelArt   = "pixel_art"
	styleModeInkWash    = "ink_wash"
	styleModeClay       = "clay"
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

// StoryRunner runs the comic generation pipeline from either a vocabulary file
// or a direct manual prompt.
type StoryRunner interface {
	Run(ctx context.Context, batchFile string) error
	RunPrompt(ctx context.Context, prompt string) error
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

func pickStyleMode(forcedUltra *bool) string {
	if forcedUltra != nil {
		if *forcedUltra {
			return styleModeRealistic
		}
		return styleModeComic
	}
	modes := []string{
		styleModeRealistic,
		styleModeComic,
		styleModeCartoon,
		styleModeAction90s,
		styleModeManga,
		styleModeHorror,
		styleModeWatercolor,
		styleModePulp,
		styleModeMecha,
		styleModePixelArt,
		styleModeInkWash,
		styleModeClay,
	}
	return modes[rand.IntN(len(modes))]
}

func pickStyle(
	comicStyles,
	realisticStyles,
	cartoonStyles,
	action90sStyles,
	mangaStyles,
	horrorStyles,
	watercolorStyles,
	pulpStyles,
	mechaStyles,
	pixelArtStyles,
	inkWashStyles,
	clayStyles []string,
	mode string,
) string {
	switch mode {
	case styleModeRealistic:
		if len(realisticStyles) == 0 {
			realisticStyles = defaultRealisticStyles
		}
		if len(realisticStyles) == 0 {
			return ""
		}
		return realisticStyles[rand.IntN(len(realisticStyles))]
	case styleModeCartoon:
		if len(cartoonStyles) == 0 {
			cartoonStyles = defaultCartoonStyles
		}
		if len(cartoonStyles) == 0 {
			return ""
		}
		return cartoonStyles[rand.IntN(len(cartoonStyles))]
	case styleModeAction90s:
		if len(action90sStyles) == 0 {
			action90sStyles = defaultAction90sStyles
		}
		if len(action90sStyles) == 0 {
			return ""
		}
		return action90sStyles[rand.IntN(len(action90sStyles))]
	case styleModeManga:
		if len(mangaStyles) == 0 {
			mangaStyles = defaultMangaStyles
		}
		if len(mangaStyles) == 0 {
			return ""
		}
		return mangaStyles[rand.IntN(len(mangaStyles))]
	case styleModeHorror:
		if len(horrorStyles) == 0 {
			horrorStyles = defaultHorrorStyles
		}
		if len(horrorStyles) == 0 {
			return ""
		}
		return horrorStyles[rand.IntN(len(horrorStyles))]
	case styleModeWatercolor:
		if len(watercolorStyles) == 0 {
			watercolorStyles = defaultWatercolorStyles
		}
		if len(watercolorStyles) == 0 {
			return ""
		}
		return watercolorStyles[rand.IntN(len(watercolorStyles))]
	case styleModePulp:
		if len(pulpStyles) == 0 {
			pulpStyles = defaultPulpStyles
		}
		if len(pulpStyles) == 0 {
			return ""
		}
		return pulpStyles[rand.IntN(len(pulpStyles))]
	case styleModeMecha:
		if len(mechaStyles) == 0 {
			mechaStyles = defaultMechaStyles
		}
		if len(mechaStyles) == 0 {
			return ""
		}
		return mechaStyles[rand.IntN(len(mechaStyles))]
	case styleModePixelArt:
		if len(pixelArtStyles) == 0 {
			pixelArtStyles = defaultPixelArtStyles
		}
		if len(pixelArtStyles) == 0 {
			return ""
		}
		return pixelArtStyles[rand.IntN(len(pixelArtStyles))]
	case styleModeInkWash:
		if len(inkWashStyles) == 0 {
			inkWashStyles = defaultInkWashStyles
		}
		if len(inkWashStyles) == 0 {
			return ""
		}
		return inkWashStyles[rand.IntN(len(inkWashStyles))]
	case styleModeClay:
		if len(clayStyles) == 0 {
			clayStyles = defaultClayStyles
		}
		if len(clayStyles) == 0 {
			return ""
		}
		return clayStyles[rand.IntN(len(clayStyles))]
	}
	if len(comicStyles) == 0 {
		comicStyles = defaultComicStyles
	}
	if len(comicStyles) == 0 {
		return ""
	}
	if rand.Float64() < 0.9 {
		return comicStyles[0]
	}
	if len(comicStyles) == 1 {
		return comicStyles[0]
	}
	return comicStyles[1+rand.IntN(len(comicStyles)-1)]
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

	panelIndexByLabel := make(map[string]int, panelsPerPage)
	for i := 0; i < panelsPerPage; i++ {
		panelIndexByLabel[panelLabel(i)] = i
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "P") {
			continue
		}
		pagePart, rest, ok := strings.Cut(line[1:], "-")
		if !ok {
			continue
		}
		panelPart, textPart, ok := strings.Cut(rest, ":")
		if !ok {
			continue
		}
		page, err := strconv.Atoi(strings.TrimSpace(pagePart))
		if err != nil {
			continue
		}
		panelPart = strings.ToUpper(strings.TrimSpace(panelPart))
		panel, ok := panelIndexByLabel[panelPart]
		if !ok || page <= 0 || page > storyPages {
			continue
		}
		script[page-1][panel] = strings.TrimSpace(textPart)
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

func clampRealisticWeight(weight float64) float64 {
	switch {
	case weight < 0:
		return 0
	case weight > 1:
		return 1
	default:
		return weight
	}
}

func panelLabel(idx int) string {
	if idx < 0 {
		return ""
	}
	idx++
	var out []byte
	for idx > 0 {
		idx--
		out = append([]byte{byte('A' + idx%26)}, out...)
		idx /= 26
	}
	return string(out)
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
