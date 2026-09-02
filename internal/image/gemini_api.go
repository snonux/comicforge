package image

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"

	"codeberg.org/snonux/comicforge/internal/apicircuit"
)

func (c *GeminiProvider) generateTextImpl(ctx context.Context, model, systemPrompt, userPrompt string, temperature float32, maxOutputTokens int32) (string, error) {
	temp := temperature
	resp, err := apicircuit.Execute(nil, c.Name(), apicircuit.CapabilityText, func() (*genai.GenerateContentResponse, error) {
		return c.client.Models.GenerateContent(ctx, model, []*genai.Content{
			genai.NewContentFromText(userPrompt, genai.RoleUser),
		}, &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
			Temperature:       &temp,
			MaxOutputTokens:   maxOutputTokens,
		})
	})
	if err != nil {
		return "", fmt.Errorf("gemini API error: %w", err)
	}

	text := strings.TrimSpace(resp.Text())
	if text == "" {
		return "", fmt.Errorf("no response received")
	}

	return text, nil
}

func geminiImageConfig(aspectRatio string) *genai.GenerateContentConfig {
	if aspectRatio == "" {
		aspectRatio = geminiAspectRatio
	}

	return &genai.GenerateContentConfig{
		ResponseModalities: []string{string(genai.ModalityImage)},
		ImageConfig: &genai.ImageConfig{
			AspectRatio: aspectRatio,
		},
	}
}

func geminiReferenceParts(prompt string, refs [][]byte) []*genai.Part {
	parts := make([]*genai.Part, 0, len(refs)+1)
	for _, ref := range refs {
		if len(ref) == 0 {
			continue
		}
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{MIMEType: "image/png", Data: ref},
		})
	}

	refNote := fmt.Sprintf(
		"The %d reference image(s) above show the exact character appearance that must be preserved. "+
			"Every character, animal, or object must keep the same identity, face, costume, colours, and markings. "+
			"Use the references for identity only: do not copy their scene, pose, camera angle, panel layout, lighting, action, or background. "+
			"The new image must follow the new prompt's story beat and composition. Now generate:\n\n",
		len(refs),
	)
	parts = append(parts, &genai.Part{Text: refNote + prompt})
	return parts
}

func (c *GeminiProvider) generateImageImpl(ctx context.Context, prompt, aspectRatio string) ([]byte, string, error) {
	cfg := geminiImageConfig(aspectRatio)

	resp, err := apicircuit.Execute(nil, c.Name(), apicircuit.CapabilityImage, func() (*genai.GenerateContentResponse, error) {
		return c.client.Models.GenerateContent(ctx, c.modelName(), []*genai.Content{
			genai.NewContentFromText(prompt, genai.RoleUser),
		}, cfg)
	})
	if err != nil {
		return nil, "", &SearchError{
			Provider: geminiSource,
			Code:     "API_ERROR",
			Message:  fmt.Sprintf("failed to generate image: %v", err),
		}
	}

	imageBytes, mimeType, err := extractGeneratedImage(resp)
	if err != nil {
		return nil, "", &SearchError{
			Provider: geminiSource,
			Code:     "NO_RESULTS",
			Message:  err.Error(),
		}
	}

	return imageBytes, mimeType, nil
}

func (c *GeminiProvider) generateImageWithRefs(ctx context.Context, prompt, aspectRatio string, refs [][]byte) ([]byte, string, error) {
	cfg := geminiImageConfig(aspectRatio)
	parts := geminiReferenceParts(prompt, refs)

	resp, err := apicircuit.Execute(nil, c.Name(), apicircuit.CapabilityImage, func() (*genai.GenerateContentResponse, error) {
		return c.client.Models.GenerateContent(ctx, c.modelName(), []*genai.Content{
			{
				Role:  string(genai.RoleUser),
				Parts: parts,
			},
		}, cfg)
	})
	if err != nil {
		return nil, "", &SearchError{
			Provider: geminiSource,
			Code:     "API_ERROR",
			Message:  fmt.Sprintf("failed to generate image with refs: %v", err),
		}
	}

	imageBytes, mimeType, err := extractGeneratedImage(resp)
	if err != nil {
		return nil, "", &SearchError{
			Provider: geminiSource,
			Code:     "NO_RESULTS",
			Message:  err.Error(),
		}
	}

	return imageBytes, mimeType, nil
}

func extractGeneratedImage(response *genai.GenerateContentResponse) ([]byte, string, error) {
	if response == nil {
		return nil, "", fmt.Errorf("no response from Gemini")
	}

	for _, candidate := range response.Candidates {
		if candidate == nil || candidate.Content == nil {
			continue
		}

		for _, part := range candidate.Content.Parts {
			if part == nil || part.InlineData == nil || len(part.InlineData.Data) == 0 {
				continue
			}

			mimeType := part.InlineData.MIMEType
			if mimeType == "" {
				mimeType = "image/png"
			}

			return append([]byte(nil), part.InlineData.Data...), mimeType, nil
		}
	}

	return nil, "", fmt.Errorf("no image data returned from Gemini")
}
