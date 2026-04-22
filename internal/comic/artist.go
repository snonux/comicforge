package comic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"codeberg.org/snonux/comicforge/internal/provider"
)

// ArtistConfig configures comic page generation.
type ArtistConfig struct {
	ImageProvider   provider.ImageProvider
	TextProvider    provider.TextProvider
	Prompts         PromptRenderer
	OutputDir       string
	Style           string
	ComicStyles     []string
	RealisticStyles []string
	Theme           string
	AspectRatio     string
	Language        string
	Script          string
	UltraRealistic  bool
	StoryPages      int
	GalleryPages    int
	PanelsPerPage   int
	PromptMaxChars  int
	PageMaxRetries  int
	PageRetryBase   time.Duration
}

// Artist generates comic-book pages.
type Artist struct {
	imageProvider   provider.ImageProvider
	textProvider    provider.TextProvider
	prompts         PromptRenderer
	outputDir       string
	style           string
	comicStyles     []string
	realisticStyles []string
	theme           string
	aspectRatio     string
	language        string
	script          string
	ultraRealistic  bool
	storyPages      int
	galleryPages    int
	panelsPerPage   int
	promptMaxChars  int
	pageMaxRetries  int
	pageRetryBase   time.Duration
	initErr         error
}

type referenceImageGenerator interface {
	GenerateImageWithReferences(context.Context, string, string, [][]byte) error
}

type referenceAspectRatioImageGenerator interface {
	GenerateImageWithReferencesAndAspectRatio(context.Context, string, string, [][]byte, string) error
}

var sleep = time.Sleep

// NewArtist creates an Artist.
func NewArtist(cfg *ArtistConfig) *Artist {
	a := &Artist{
		outputDir:      ".",
		language:       "Bulgarian",
		script:         "Cyrillic",
		storyPages:     defaultStoryPagesInScript,
		galleryPages:   5,
		panelsPerPage:  defaultStoryPanelsPerPage,
		ultraRealistic: true,
	}
	if cfg == nil {
		a.initErr = fmt.Errorf("artist config is required")
		return a
	}

	a.imageProvider = cfg.ImageProvider
	a.textProvider = cfg.TextProvider
	a.prompts = cfg.Prompts
	a.outputDir = orDefault(cfg.OutputDir, a.outputDir)
	a.style = cfg.Style
	a.comicStyles = append([]string(nil), cfg.ComicStyles...)
	a.realisticStyles = append([]string(nil), cfg.RealisticStyles...)
	a.theme = cfg.Theme
	a.aspectRatio = orDefault(cfg.AspectRatio, comicPageAspectRatio)
	a.language = orDefault(cfg.Language, a.language)
	a.script = orDefault(cfg.Script, a.script)
	a.ultraRealistic = cfg.UltraRealistic
	if cfg.StoryPages > 0 {
		a.storyPages = cfg.StoryPages
	}
	if cfg.GalleryPages > 0 {
		a.galleryPages = cfg.GalleryPages
	}
	if cfg.PanelsPerPage > 0 {
		a.panelsPerPage = cfg.PanelsPerPage
	}
	a.promptMaxChars = normalizePositive(cfg.PromptMaxChars, comicPromptMaxChars)
	a.pageMaxRetries = normalizePositive(cfg.PageMaxRetries, pageMaxRetries)
	if cfg.PageRetryBase > 0 {
		a.pageRetryBase = cfg.PageRetryBase
	} else {
		a.pageRetryBase = pageRetryBase
	}

	if a.imageProvider == nil {
		a.initErr = fmt.Errorf("%w: image provider", ErrMissingProvider)
	}
	if a.prompts == nil {
		a.initErr = errorsJoin(a.initErr, fmt.Errorf("%w: prompt renderer", ErrMissingProvider))
	}
	return a
}

// DrawComicPages renders the cover, story pages, gallery pages, and back cover.
func (a *Artist) DrawComicPages(ctx context.Context, storyText, bible, titleSlug string, entries []WordEntry, panelScript [][]string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	style := a.style
	if style == "" {
		style = pickStyle(a.comicStyles, a.realisticStyles, a.ultraRealistic)
	}
	fmt.Printf("  Comic style: %s\n", style)

	resolvedBible, blurb, err := a.resolveHelperTexts(ctx, storyText, bible)
	if err != nil {
		return nil, err
	}

	var paths []string
	var recentRefs [][]byte
	if p, err := a.renderPage(ctx, titleSlug+"_cover", coverPromptTemplate, a.coverPromptData(storyText, style, resolvedBible), "cover page", nil); err != nil {
		return nil, err
	} else if p != "" {
		paths = append(paths, p)
		if coverBytes, readErr := os.ReadFile(p); readErr == nil {
			recentRefs = appendRef(recentRefs, coverBytes)
		}
	}

	sections := splitIntoSections(storyText, a.storyPages)
	for i, section := range sections {
		pageNum := i + 1
		data := a.storyPagePromptData(section, pageNum, style, resolvedBible, entries, panelScript)
		if p, err := a.renderPage(ctx, fmt.Sprintf("%s_page_%d", titleSlug, pageNum), storyPagePromptTemplate, data, fmt.Sprintf("story page %d", pageNum), recentRefs); err != nil {
			return nil, err
		} else if p != "" {
			paths = append(paths, p)
			if pageBytes, readErr := os.ReadFile(p); readErr == nil {
				recentRefs = appendRef(recentRefs, pageBytes)
			}
		}
	}

	for i := 0; i < a.galleryPages; i++ {
		galleryNum := i + 1
		data := a.galleryPromptData(style, resolvedBible, galleryNum)
		if p, err := a.renderPage(ctx, fmt.Sprintf("%s_gallery_%d", titleSlug, galleryNum), galleryPagePromptTemplate, data, fmt.Sprintf("gallery page %d/%d", galleryNum, a.galleryPages), recentRefs); err != nil {
			return nil, err
		} else if p != "" {
			paths = append(paths, p)
			if galleryBytes, readErr := os.ReadFile(p); readErr == nil {
				recentRefs = appendRef(recentRefs, galleryBytes)
			}
		}
	}

	if p, err := a.renderPage(ctx, titleSlug+"_back", backCoverPromptTemplate, a.backPromptData(storyText, style, resolvedBible, blurb), "back cover", recentRefs); err != nil {
		return nil, err
	} else if p != "" {
		paths = append(paths, p)
	}
	return paths, nil
}

func (a *Artist) ready() error {
	if a == nil {
		return fmt.Errorf("artist is nil")
	}
	if a.initErr != nil {
		return a.initErr
	}
	return nil
}

func (a *Artist) renderPage(ctx context.Context, fileName, templateName string, data map[string]any, label string, refs [][]byte) (string, error) {
	path := filepath.Join(a.outputDir, fileName+".png")
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  Skipping %s (already exists)\n", filepath.Base(path))
		return path, nil
	}
	prompt, err := a.prompts.RenderPrompt(templateName, data)
	if err != nil {
		return "", fmt.Errorf("render %s prompt: %w", label, err)
	}
	if err := a.generateWithRetry(ctx, prompt, path, label, refs); err != nil {
		return "", err
	}
	return path, nil
}

func (a *Artist) generateWithRetry(ctx context.Context, prompt, outputFile, label string, refs [][]byte) error {
	return a.generateWithRetryAndValidation(ctx, prompt, outputFile, label, refs, validateImagePromptLeakageFn)
}

func (a *Artist) generatePromptImage(ctx context.Context, prompt, outputFile string) error {
	renderedPrompt, err := a.prompts.RenderPrompt(manualPromptTemplate, a.manualPromptData(prompt))
	if err != nil {
		return fmt.Errorf("render manual prompt: %w", err)
	}
	return a.generateWithRetryAndValidation(ctx, renderedPrompt, outputFile, "manual prompt image", nil, validateImagePromptLeakageFn)
}

type imageOutputValidator func(context.Context, string, string, string) error

func (a *Artist) generateWithRetryAndValidation(ctx context.Context, prompt, outputFile, label string, refs [][]byte, validator imageOutputValidator) error {
	attempts := a.pageMaxRetries
	for attempt := 1; attempt <= attempts; attempt++ {
		callCtx, cancel := withTimeout(ctx, helperTimeout)
		err := a.generateImage(callCtx, prompt, outputFile, refs)
		if err == nil && validator != nil {
			if leakErr := validator(callCtx, outputFile, label, a.script); leakErr != nil {
				_ = os.Remove(outputFile)
				err = leakErr
			}
		}
		cancel()
		if err == nil {
			return nil
		}
		if attempt < attempts {
			pause := a.pageRetryBase * time.Duration(attempt)
			fmt.Printf("  Warning: %s attempt %d/%d failed (%v), retrying in %s...\n", label, attempt, attempts, err, pause)
			sleep(pause)
			continue
		}
		return fmt.Errorf("%s failed after %d attempts: %w", label, attempts, err)
	}
	return nil
}

func (a *Artist) generateImage(ctx context.Context, prompt, outputFile string, refs [][]byte) error {
	if withRefs, ok := a.imageProvider.(referenceAspectRatioImageGenerator); ok {
		return withRefs.GenerateImageWithReferencesAndAspectRatio(ctx, prompt, outputFile, refs, a.aspectRatio)
	}
	if len(refs) > 0 {
		if withRefs, ok := a.imageProvider.(referenceImageGenerator); ok {
			return withRefs.GenerateImageWithReferences(ctx, prompt, outputFile, refs)
		}
	}
	if withAspectRatio, ok := a.imageProvider.(provider.AspectRatioImageProvider); ok {
		return withAspectRatio.GenerateImageWithAspectRatio(ctx, prompt, outputFile, a.aspectRatio)
	}
	return a.imageProvider.GenerateImage(ctx, prompt, outputFile)
}

func appendRef(refs [][]byte, imgBytes []byte) [][]byte {
	if len(imgBytes) == 0 {
		return refs
	}
	refs = append(refs, imgBytes)
	if len(refs) > 2 {
		refs = [][]byte{refs[0], refs[len(refs)-1]}
	}
	return refs
}

func (a *Artist) resolveHelperTexts(ctx context.Context, storyText, prebuiltBible string) (string, string, error) {
	bible := strings.TrimSpace(prebuiltBible)
	if bible != "" {
		fmt.Printf("  Character bible ready (%d chars)\n", len(bible))
	}
	blurb := ""
	if a.textProvider == nil {
		return bible, blurb, nil
	}
	systemPrompt, err := a.prompts.RenderPrompt(blurbSystemTemplate, map[string]any{
		"StoryText":    storyText,
		"Language":     a.language,
		"LanguageName": localizedLanguageName(a.language, a.script),
		"Script":       a.script,
		"ScriptName":   localizedScriptName(a.script),
	})
	if err != nil {
		return "", "", fmt.Errorf("render blurb prompt: %w", err)
	}
	prompt := systemPrompt + "\n\n" + storyText
	callCtx, cancel := withTimeout(ctx, helperTimeout)
	defer cancel()
	text, err := a.textProvider.GenerateText(callCtx, prompt)
	if err != nil {
		fmt.Printf("  Warning: back-cover blurb generation failed: %v\n", err)
		return bible, blurb, nil
	}
	blurb = strings.TrimSpace(text)
	if err := validateTextScript("back-cover blurb", blurb, a.script); err != nil {
		return "", "", fmt.Errorf("generate back-cover blurb: %w", err)
	}
	if err := validateNoPromptLeakage("back-cover blurb", blurb); err != nil {
		return "", "", fmt.Errorf("generate back-cover blurb: %w", err)
	}
	if blurb != "" {
		fmt.Printf("  Back-cover blurb ready (%d chars)\n", len(blurb))
	}
	return bible, blurb, nil
}

func (a *Artist) coverPromptData(storyText, style, bible string) map[string]any {
	return map[string]any{
		"Language":                a.language,
		"LanguageName":            localizedLanguageName(a.language, a.script),
		"Script":                  a.script,
		"ScriptName":              localizedScriptName(a.script),
		"Style":                   localizedStylePrompt(style, a.language, a.script),
		"Bible":                   bible,
		"Subtitle":                localizedBrandName(a.language, a.script),
		"StoryText":               storyText,
		"RenderingRequirement":    a.renderingRequirement(),
		"RenderingRequirementEnd": a.renderingRequirementEnd(),
	}
}

func (a *Artist) storyPagePromptData(section string, pageNum int, style, bible string, entries []WordEntry, panelScript [][]string) map[string]any {
	return map[string]any{
		"Language":                a.language,
		"LanguageName":            localizedLanguageName(a.language, a.script),
		"Script":                  a.script,
		"ScriptName":              localizedScriptName(a.script),
		"Style":                   localizedStylePrompt(style, a.language, a.script),
		"Bible":                   bible,
		"Words":                   buildWordList(entries, ""),
		"PageNum":                 pageNum,
		"TotalPages":              a.storyPages,
		"PanelsPerPage":           a.panelsPerPage,
		"RequiredDialoguePanels":  requiredDialoguePanels(a.panelsPerPage),
		"TotalPanels":             a.storyPages * a.panelsPerPage,
		"PanelLabelsText":         panelLabelsText(a.panelsPerPage),
		"PanelLayout":             buildPanelLayout(section, pageScriptForPage(panelScript, pageNum-1), a.panelsPerPage, a.promptMaxChars),
		"RenderingRequirement":    a.renderingRequirement(),
		"RenderingRequirementEnd": a.renderingRequirementEnd(),
	}
}

func (a *Artist) galleryPromptData(style, bible string, galleryNum int) map[string]any {
	return map[string]any{
		"Language":                a.language,
		"LanguageName":            localizedLanguageName(a.language, a.script),
		"Script":                  a.script,
		"ScriptName":              localizedScriptName(a.script),
		"Style":                   localizedStylePrompt(style, a.language, a.script),
		"Bible":                   bible,
		"GalleryNum":              galleryNum,
		"TotalGalleryPages":       a.galleryPages,
		"Pose":                    galleryPoses[(galleryNum-1)%len(galleryPoses)],
		"RenderingRequirement":    a.renderingRequirement(),
		"RenderingRequirementEnd": a.renderingRequirementEnd(),
	}
}

func (a *Artist) backPromptData(storyText, style, bible, blurb string) map[string]any {
	return map[string]any{
		"Language":                a.language,
		"LanguageName":            localizedLanguageName(a.language, a.script),
		"Script":                  a.script,
		"ScriptName":              localizedScriptName(a.script),
		"Style":                   localizedStylePrompt(style, a.language, a.script),
		"Bible":                   bible,
		"BlurbBox":                blurbBoxInstruction(blurb),
		"SeriesTitle":             localizedBrandName(a.language, a.script),
		"StoryText":               storyText,
		"RenderingRequirement":    a.renderingRequirement(),
		"RenderingRequirementEnd": a.renderingRequirementEnd(),
	}
}

func (a *Artist) manualPromptData(prompt string) map[string]any {
	return map[string]any{
		"Prompt":                  strings.TrimSpace(prompt),
		"Style":                   localizedStylePrompt(a.style, a.language, a.script),
		"Theme":                   a.theme,
		"RenderingRequirement":    a.renderingRequirement(),
		"RenderingRequirementEnd": a.renderingRequirementEnd(),
	}
}

func (a *Artist) renderingRequirement() string {
	if a.ultraRealistic {
		text, err := a.prompts.RenderPrompt(renderingRequirementPrompt, nil)
		if err == nil {
			return text
		}
	}
	return ""
}

func (a *Artist) renderingRequirementEnd() string {
	if a.ultraRealistic {
		text, err := a.prompts.RenderPrompt(renderingRequirementEndPrompt, nil)
		if err == nil {
			return text
		}
	}
	return ""
}

func pageScriptForPage(panelScript [][]string, idx int) []string {
	if idx < 0 || idx >= len(panelScript) {
		return nil
	}
	return panelScript[idx]
}

func buildPanelLayout(section string, pagePanels []string, panelCount, promptMaxChars int) string {
	panelCount = normalizePositive(panelCount, defaultStoryPanelsPerPage)
	if len(pagePanels) == panelCount && allPanelsPresent(pagePanels) {
		var sb strings.Builder
		sb.WriteString(panelLayoutLead(panelCount))
		sb.WriteString(" The panels must tell the scene in sequence and remain visually distinct from each other.\n")
		for i, panel := range pagePanels {
			if panel == "" {
				continue
			}
			fmt.Fprintf(&sb, "Panel %s: %s\n", panelLabel(i), panel)
		}
		return sb.String()
	}

	excerpt := strings.TrimSpace(section)
	promptMaxChars = normalizePositive(promptMaxChars, comicPromptMaxChars)
	if utf8.RuneCountInString(excerpt) > promptMaxChars {
		excerpt = string([]rune(excerpt)[:promptMaxChars])
		if idx := strings.LastIndex(excerpt, " "); idx > 0 {
			excerpt = excerpt[:idx]
		}
		excerpt += "…"
	}
	return panelLayoutLead(panelCount) + " The panels must tell the story in sequence from beginning to end and remain visually distinct.\n" +
		"Story excerpt:\n\n" + excerpt + "\n"
}

func allPanelsPresent(pagePanels []string) bool {
	for _, panel := range pagePanels {
		if strings.TrimSpace(panel) == "" {
			return false
		}
	}
	return true
}

func panelLayoutLead(panelCount int) string {
	switch panelCount {
	case 1:
		return "Divide the image into exactly 1 distinct panel."
	case 2:
		return "Divide the image into exactly 2 distinct panels in a balanced two-panel layout."
	case 3:
		return "Divide the image into exactly 3 distinct panels in a balanced three-panel layout."
	case 4:
		return "Divide the image into exactly 4 distinct panels in a 2x2 grid."
	default:
		return fmt.Sprintf("Divide the image into exactly %d distinct panels in a balanced grid.", panelCount)
	}
}

func blurbBoxInstruction(blurb string) string {
	if strings.TrimSpace(blurb) == "" {
		return "a rectangular text box near the bottom with a white or cream background and a thin black border, styled like a classic back-cover synopsis box"
	}
	return fmt.Sprintf("a rectangular text box near the bottom with a white or cream background and a thin black border, displaying this italic text:\n    %q", blurb)
}

func errorsJoin(errs ...error) error {
	var filtered []error
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return fmt.Errorf("%v; %w", filtered[0], filtered[1])
	}
}
