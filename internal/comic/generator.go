package comic

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"codeberg.org/snonux/comicforge/internal/provider"
)

// GeneratorConfig configures story generation.
type GeneratorConfig struct {
	TextProvider  provider.TextProvider
	Prompts       PromptRenderer
	Language      string
	Script        string
	Theme         string
	Genres        []string
	StoryPages    int
	PanelsPerPage int
}

// Generator produces the comic story text, bible, title, and panel script.
type Generator struct {
	textProvider  provider.TextProvider
	prompts       PromptRenderer
	language      string
	script        string
	theme         string
	genres        []string
	storyPages    int
	panelsPerPage int
	initErr       error
}

// NewGenerator creates a new story generator.
func NewGenerator(cfg *GeneratorConfig) *Generator {
	g := &Generator{
		language:      "Bulgarian",
		script:        "Cyrillic",
		genres:        defaultStoryGenres,
		storyPages:    defaultStoryPagesInScript,
		panelsPerPage: defaultStoryPanelsPerPage,
	}
	if cfg == nil {
		g.initErr = errors.New("generator config is required")
		return g
	}
	g.textProvider = cfg.TextProvider
	g.prompts = cfg.Prompts
	g.language = orDefault(cfg.Language, g.language)
	g.script = orDefault(cfg.Script, g.script)
	g.theme = cfg.Theme
	if cfg.StoryPages > 0 {
		g.storyPages = cfg.StoryPages
	}
	if cfg.PanelsPerPage > 0 {
		g.panelsPerPage = cfg.PanelsPerPage
	}
	if len(cfg.Genres) > 0 {
		g.genres = append([]string(nil), cfg.Genres...)
	}
	if g.textProvider == nil {
		g.initErr = fmt.Errorf("%w: text provider", ErrMissingProvider)
	}
	if g.prompts == nil {
		g.initErr = errors.Join(g.initErr, fmt.Errorf("%w: prompt renderer", ErrMissingProvider))
	}
	return g
}

// Generate produces only the story text.
func (g *Generator) Generate(ctx context.Context, entries []WordEntry) (string, error) {
	if err := g.ready(); err != nil {
		return "", err
	}
	prompt, err := g.renderStoryPrompt(storyPromptTemplate, entries)
	if err != nil {
		return "", err
	}
	ctx, cancel := withTimeout(ctx, storyTimeout)
	defer cancel()
	text, err := g.textProvider.GenerateText(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("generate story: %w", err)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("no story content returned")
	}
	if err := validateTextScript("story text", text, g.script); err != nil {
		return "", err
	}
	if err := validateNoPromptLeakage("story text", text); err != nil {
		return "", err
	}
	return text, nil
}

// GenerateFull produces the combined story, bible, title, and panel script.
func (g *Generator) GenerateFull(ctx context.Context, entries []WordEntry) (GenerateResult, error) {
	if err := g.ready(); err != nil {
		return GenerateResult{}, err
	}
	prompt, err := g.renderStoryPrompt(storyFullPromptTemplate, entries)
	if err != nil {
		return GenerateResult{}, err
	}
	ctx, cancel := withTimeout(ctx, storyTimeout)
	defer cancel()
	text, err := g.textProvider.GenerateText(ctx, prompt)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("generate story bundle: %w", err)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return GenerateResult{}, fmt.Errorf("no content returned")
	}
	result := parseGenerateResultWithDimensions(text, g.storyPages, g.panelsPerPage)
	if err := validateGeneratedResult(result, g.script); err != nil {
		return GenerateResult{}, err
	}
	return result, nil
}

func (g *Generator) ready() error {
	if g == nil {
		return errors.New("generator is nil")
	}
	if g.initErr != nil {
		return g.initErr
	}
	return nil
}

func (g *Generator) renderStoryPrompt(templateName string, entries []WordEntry) (string, error) {
	genre := resolveGenre(g.theme, g.genres)
	data := map[string]any{
		"Language":               g.language,
		"LanguageName":           localizedLanguageName(g.language, g.script),
		"Script":                 g.script,
		"ScriptName":             localizedScriptName(g.script),
		"Genre":                  genre,
		"Words":                  buildWordList(entries, ""),
		"StoryPages":             g.storyPages,
		"PanelsPerPage":          g.panelsPerPage,
		"TotalPanels":            g.storyPages * g.panelsPerPage,
		"PanelLabelsText":        panelLabelsText(g.panelsPerPage),
		"RequiredDialoguePanels": requiredDialoguePanels(g.panelsPerPage),
		"StoryBibleSeparator":    storyBibleSeparator,
		"StoryTitleSeparator":    storyTitleSeparator,
		"StoryPanelSeparator":    storyPanelSeparator,
	}

	systemPrompt, err := g.prompts.RenderPrompt(storySystemPromptTemplate, map[string]any{
		"Language":     g.language,
		"LanguageName": localizedLanguageName(g.language, g.script),
		"Script":       g.script,
		"ScriptName":   localizedScriptName(g.script),
	})
	if err != nil {
		return "", err
	}
	userPrompt, err := g.prompts.RenderPrompt(templateName, data)
	if err != nil {
		return "", err
	}
	return systemPrompt + "\n\n" + userPrompt, nil
}

func validateGeneratedResult(result GenerateResult, script string) error {
	if err := validateTextScript("story text", result.StoryText, script); err != nil {
		return fmt.Errorf("%w (story=%q)", err, compactTextSnippet(result.StoryText))
	}
	if err := validateNoPromptLeakage("story text", result.StoryText); err != nil {
		return fmt.Errorf("%w (story=%q)", err, compactTextSnippet(result.StoryText))
	}
	if err := validateTextScript("character bible", result.Bible, script); err != nil {
		return fmt.Errorf("%w (bible=%q)", err, compactTextSnippet(result.Bible))
	}
	if err := validateNoPromptLeakage("character bible", result.Bible); err != nil {
		return fmt.Errorf("%w (bible=%q)", err, compactTextSnippet(result.Bible))
	}
	if err := validateTextScript("comic title", result.Title, script); err != nil {
		return fmt.Errorf("%w (title=%q)", err, compactTextSnippet(result.Title))
	}
	if err := validateNoPromptLeakage("comic title", result.Title); err != nil {
		return fmt.Errorf("%w (title=%q)", err, compactTextSnippet(result.Title))
	}
	for pageIdx, page := range result.PanelScript {
		for panelIdx, panel := range page {
			label := fmt.Sprintf("panel script P%d-%s", pageIdx+1, panelLabel(panelIdx))
			if err := validateTextScript(label, panel, script); err != nil {
				return fmt.Errorf("%w (panel=%q)", err, compactTextSnippet(panel))
			}
			if err := validateNoPromptLeakage(label, panel); err != nil {
				return fmt.Errorf("%w (panel=%q)", err, compactTextSnippet(panel))
			}
		}
	}
	return nil
}

func compactTextSnippet(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 220 {
		text = text[:220] + "..."
	}
	return text
}
