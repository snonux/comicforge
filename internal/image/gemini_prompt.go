package image

import (
	"context"
	"fmt"
	"strings"
)

func normalizeCustomPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if len(prompt) > maxCustomPrompt {
		prompt = prompt[:maxCustomPrompt-3] + "..."
	}
	return prompt
}

func (c *GeminiProvider) resolveTranslation(_ context.Context, opts *SearchOptions, translation string) (string, error) {
	if translation != "" {
		fmt.Printf("Using provided translation: %s -> %s\n", opts.Query, translation)
		return translation, nil
	}

	return opts.Query, nil
}

func (c *GeminiProvider) resolvePrompt(ctx context.Context, opts *SearchOptions, translatedWord string) (string, error) {
	if customPrompt := normalizeCustomPrompt(opts.CustomPrompt); customPrompt != "" {
		fmt.Printf("Using custom prompt: %s\n", customPrompt)
		return customPrompt, nil
	}

	return c.createEducationalPrompt(ctx, opts.Query, translatedWord), nil
}

func (c *GeminiProvider) buildPrompt(ctx context.Context, opts *SearchOptions) (string, string, error) {
	if opts == nil {
		return "", "", &SearchError{
			Provider: geminiSource,
			Code:     "INVALID_OPTIONS",
			Message:  "search options are required",
		}
	}

	translation := strings.TrimSpace(opts.Translation)
	if customPrompt := normalizeCustomPrompt(opts.CustomPrompt); customPrompt != "" {
		fmt.Printf("Using custom prompt: %s\n", customPrompt)
		return customPrompt, translation, nil
	}

	translatedWord, err := c.resolveTranslation(ctx, opts, translation)
	if err != nil {
		return "", "", err
	}

	prompt, err := c.resolvePrompt(ctx, opts, translatedWord)
	if err != nil {
		return "", "", err
	}

	return prompt, translatedWord, nil
}

// createEducationalPrompt generates a prompt optimized for image generation.
func (c *GeminiProvider) createEducationalPrompt(ctx context.Context, query, translation string) string {
	subject := promptSubject(translation, query)

	scene, err := c.generateSceneDescription(ctx, query, translation)
	if err != nil {
		fmt.Printf("  Failed to generate scene: %v, using basic prompt\n", err)
		scene = ""
	}
	if scene != "" {
		scene = sanitizeSceneDescription(scene)
		if !usableSceneDescription(scene) {
			fmt.Printf("  Scene response was too short or generic, using basic prompt\n")
			scene = ""
		}
	}

	selectedStyle := chooseArtisticStyle()
	if selectedStyle == defaultArtisticStyle {
		fmt.Printf("  No artistic styles available, using generic prompt\n")
	}
	fmt.Printf("  Using image style: %s\n", selectedStyle)

	return buildEducationalPrompt(selectedStyle, scene, subject)
}

func (c *GeminiProvider) generateSceneDescription(ctx context.Context, query, translation string) (string, error) {
	fmt.Printf("Gemini Scene Generation: Creating scene for %q (%s)\n", query, translation)

	scene, err := geminiGenerateText(
		ctx,
		c,
		c.textModelName(),
		"You are helping create educational flashcards for language learning. Generate a brief, vivid scene description that incorporates the given English word in a memorable, contextual way. The scene should be visually interesting and help with memory retention. Keep it to 1-2 sentences, focusing on visual elements that can be illustrated. The subject (the English word) should be the clear focal point of the image, prominent and centered.",
		fmt.Sprintf("Create a scene description for the English word %q that would make a memorable flashcard image. Make sure %q is the main focus and most prominent element in the scene.", translation, translation),
		0.7,
		100,
	)
	if err != nil {
		return "", fmt.Errorf("scene generation failed: %w", err)
	}
	scene = sanitizeSceneDescription(scene)
	if !usableSceneDescription(scene) {
		return "", fmt.Errorf("scene generation returned unusable content")
	}

	fmt.Printf("Generated scene: %s\n", scene)
	return scene, nil
}
