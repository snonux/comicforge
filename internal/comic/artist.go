package comic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codeberg.org/snonux/comicforge/internal/provider"
)

// ArtistConfig configures comic page generation.
type ArtistConfig struct {
	ImageProvider  provider.ImageProvider
	TextProvider   provider.TextProvider
	Prompts        PromptRenderer
	OutputDir      string
	Style          string
	Theme          string
	Language       string
	Script         string
	UltraRealistic bool
	StoryPages     int
	GalleryPages   int
	PanelsPerPage  int
}

// Artist generates comic-book pages.
type Artist struct {
	imageProvider  provider.ImageProvider
	textProvider   provider.TextProvider
	prompts        PromptRenderer
	outputDir      string
	style          string
	theme          string
	language       string
	script         string
	ultraRealistic bool
	storyPages     int
	galleryPages   int
	panelsPerPage  int
	initErr        error
}

var sleep = time.Sleep

// NewArtist creates an Artist.
func NewArtist(cfg *ArtistConfig) *Artist {
	a := &Artist{
		outputDir:      ".",
		language:       "Bulgarian",
		script:         "Cyrillic",
		storyPages:     storyPagesInScript,
		galleryPages:   5,
		panelsPerPage:  storyPanelsPerPage,
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
	a.theme = cfg.Theme
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
		style = pickStyle(nil, a.ultraRealistic)
	}
	fmt.Printf("  Comic style: %s\n", style)

	resolvedBible, blurb, err := a.resolveHelperTexts(ctx, storyText, bible)
	if err != nil {
		return nil, err
	}

	var paths []string
	if p, err := a.renderPage(ctx, titleSlug+"_cover", coverPromptTemplate, a.coverPromptData(storyText, style, resolvedBible), "cover page"); err != nil {
		return nil, err
	} else if p != "" {
		paths = append(paths, p)
	}

	sections := splitIntoSections(storyText, a.storyPages)
	for i, section := range sections {
		pageNum := i + 1
		data := a.storyPagePromptData(section, pageNum, style, resolvedBible, entries, panelScript)
		if p, err := a.renderPage(ctx, fmt.Sprintf("%s_page_%d", titleSlug, pageNum), storyPagePromptTemplate, data, fmt.Sprintf("story page %d", pageNum)); err != nil {
			return nil, err
		} else if p != "" {
			paths = append(paths, p)
		}
	}

	for i := 0; i < a.galleryPages; i++ {
		galleryNum := i + 1
		data := a.galleryPromptData(style, resolvedBible, galleryNum)
		if p, err := a.renderPage(ctx, fmt.Sprintf("%s_gallery_%d", titleSlug, galleryNum), galleryPagePromptTemplate, data, fmt.Sprintf("gallery page %d/%d", galleryNum, a.galleryPages)); err != nil {
			return nil, err
		} else if p != "" {
			paths = append(paths, p)
		}
	}

	if p, err := a.renderPage(ctx, titleSlug+"_back", backCoverPromptTemplate, a.backPromptData(storyText, style, resolvedBible, blurb), "back cover"); err != nil {
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

func (a *Artist) renderPage(ctx context.Context, fileName, templateName string, data map[string]any, label string) (string, error) {
	path := filepath.Join(a.outputDir, fileName+".png")
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  Skipping %s (already exists)\n", filepath.Base(path))
		return path, nil
	}
	prompt, err := a.prompts.RenderPrompt(templateName, data)
	if err != nil {
		return "", fmt.Errorf("render %s prompt: %w", label, err)
	}
	if err := a.generateWithRetry(ctx, prompt, path, label); err != nil {
		return "", err
	}
	return path, nil
}

func (a *Artist) generateWithRetry(ctx context.Context, prompt, outputFile, label string) error {
	attempts := pageMaxRetries
	for attempt := 1; attempt <= attempts; attempt++ {
		callCtx, cancel := withTimeout(ctx, helperTimeout)
		err := a.imageProvider.GenerateImage(callCtx, prompt, outputFile)
		if err == nil {
			if leakErr := validateImagePromptLeakageFn(callCtx, outputFile, label); leakErr != nil {
				_ = os.Remove(outputFile)
				err = leakErr
			}
		}
		cancel()
		if err == nil {
			return nil
		}
		if attempt < attempts {
			pause := pageRetryBase * time.Duration(attempt)
			fmt.Printf("  Warning: %s attempt %d/%d failed (%v), retrying in %s...\n", label, attempt, attempts, err, pause)
			sleep(pause)
			continue
		}
		return fmt.Errorf("%s failed after %d attempts: %w", label, attempts, err)
	}
	return nil
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
		"StoryText": storyText,
		"Language":  a.language,
		"Script":    a.script,
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
		"Script":                  a.script,
		"Style":                   style,
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
		"Script":                  a.script,
		"Style":                   style,
		"Bible":                   bible,
		"Words":                   buildWordList(entries, ""),
		"PageNum":                 pageNum,
		"TotalPages":              a.storyPages,
		"PanelLayout":             buildPanelLayout(section, pageScriptForPage(panelScript, pageNum-1)),
		"RenderingRequirement":    a.renderingRequirement(),
		"RenderingRequirementEnd": a.renderingRequirementEnd(),
	}
}

func (a *Artist) galleryPromptData(style, bible string, galleryNum int) map[string]any {
	return map[string]any{
		"Language":                a.language,
		"Script":                  a.script,
		"Style":                   style,
		"Bible":                   bible,
		"Pose":                    galleryPoses[(galleryNum-1)%len(galleryPoses)],
		"RenderingRequirement":    a.renderingRequirement(),
		"RenderingRequirementEnd": a.renderingRequirementEnd(),
	}
}

func (a *Artist) backPromptData(storyText, style, bible, blurb string) map[string]any {
	return map[string]any{
		"Language":                a.language,
		"Script":                  a.script,
		"Style":                   style,
		"Bible":                   bible,
		"BlurbBox":                blurbBoxInstruction(blurb),
		"SeriesTitle":             localizedBrandName(a.language, a.script),
		"StoryText":               storyText,
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

func buildPanelLayout(section string, pagePanels []string) string {
	labels := [4]string{"TOP-LEFT", "TOP-RIGHT", "BOTTOM-LEFT", "BOTTOM-RIGHT"}
	if len(pagePanels) == 4 && pagePanels[0] != "" && pagePanels[1] != "" && pagePanels[2] != "" && pagePanels[3] != "" {
		var sb strings.Builder
		sb.WriteString("ЗАДЪЛЖИТЕЛНО ОФОРМЛЕНИЕ НА ПАНЕЛИТЕ — раздели изображението точно на 4 панела в решетка 2×2.\n")
		sb.WriteString("Оформи всеки панел ТОЧНО както е описано по-долу:\n")
		for i, label := range labels {
			sb.WriteString(fmt.Sprintf("  • %s панел: %s\n", label, pagePanels[i]))
		}
		return sb.String()
	}

	excerpt := strings.TrimSpace(section)
	if len(excerpt) > comicPromptMaxChars {
		excerpt = excerpt[:comicPromptMaxChars]
		if idx := strings.LastIndex(excerpt, " "); idx > 0 {
			excerpt = excerpt[:idx]
		}
		excerpt += "…"
	}
	return "ЗАДЪЛЖИТЕЛНО ОФОРМЛЕНИЕ НА ПАНЕЛИТЕ — раздели изображението точно на 4 панела в решетка 2×2:\n" +
		"  • TOP-LEFT панел: сцена 1 от откъса\n" +
		"  • TOP-RIGHT панел: сцена 2 от откъса\n" +
		"  • BOTTOM-LEFT панел: сцена 3 от откъса\n" +
		"  • BOTTOM-RIGHT панел: сцена 4 от откъса\n" +
		"Откъс от историята (всички панели трябва да илюстрират САМО този откъс):\n\n" + excerpt + "\n"
}

func blurbBoxInstruction(blurb string) string {
	if strings.TrimSpace(blurb) == "" {
		return "правоъгълно текстово поле (бял или кремав фон, тънка черна рамка) близо до дъното, в стил на класически синопсис на задната корица"
	}
	return fmt.Sprintf("правоъгълно текстово поле (бял или кремав фон, тънка черна рамка) близо до дъното, което показва този текст в курсив:\n    %q", blurb)
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
