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
	TextProvider provider.TextProvider
	Prompts      PromptRenderer
	Language     string
	Script       string
	Theme        string
	Genres       []string
}

// Generator produces the comic story text, bible, title, and panel script.
type Generator struct {
	textProvider provider.TextProvider
	prompts      PromptRenderer
	language     string
	script       string
	theme        string
	genres       []string
	initErr      error
}

var _ = (*Generator)(nil)

// NewGenerator creates a new story generator.
func NewGenerator(cfg *GeneratorConfig) *Generator {
	g := &Generator{
		language: "Bulgarian",
		script:   "Cyrillic",
		genres:   defaultStoryGenres,
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
	result := parseGenerateResult(text)
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
		"Language":            g.language,
		"Script":              g.script,
		"Genre":               genre,
		"Words":               buildWordList(entries, ""),
		"StoryBibleSeparator": storyBibleSeparator,
		"StoryTitleSeparator": storyTitleSeparator,
		"StoryPanelSeparator": storyPanelSeparator,
	}

	systemPrompt, err := g.prompts.RenderPrompt(storySystemPromptTemplate, map[string]any{
		"Language": g.language,
		"Script":   g.script,
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
		return err
	}
	if err := validateNoPromptLeakage("story text", result.StoryText); err != nil {
		return err
	}
	if err := validateTextScript("character bible", result.Bible, script); err != nil {
		return err
	}
	if err := validateNoPromptLeakage("character bible", result.Bible); err != nil {
		return err
	}
	if err := validateTextScript("comic title", result.Title, script); err != nil {
		return err
	}
	if err := validateNoPromptLeakage("comic title", result.Title); err != nil {
		return err
	}
	for pageIdx, page := range result.PanelScript {
		for panelIdx, panel := range page {
			if err := validateTextScript(fmt.Sprintf("panel script P%d-%c", pageIdx+1, 'A'+panelIdx), panel, script); err != nil {
				return err
			}
			if err := validateNoPromptLeakage(fmt.Sprintf("panel script P%d-%c", pageIdx+1, 'A'+panelIdx), panel); err != nil {
				return err
			}
		}
	}
	return nil
}
