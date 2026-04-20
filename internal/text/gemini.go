package text

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"

	"codeberg.org/snonux/comicforge/internal/httpctx"
	"codeberg.org/snonux/comicforge/internal/provider"
)

const (
	// DefaultModel is the Gemini text model used for story generation.
	DefaultModel = "gemini-2.5-flash"
)

// GeminiConfig holds the settings needed to build a Gemini-backed text provider.
type GeminiConfig struct {
	APIKey string
	Model  string
}

// GeminiProvider implements TextProvider for Google Gemini text generation.
type GeminiProvider struct {
	client *genai.Client
	model  string
	err    error
}

var _ provider.TextProvider = (*GeminiProvider)(nil)

var newGeminiClient = httpctx.NewGenAIClient
var geminiGenerateText = func(ctx context.Context, p *GeminiProvider, prompt string) (string, error) {
	return p.generateText(ctx, prompt)
}

// NewGeminiProvider creates a Gemini text provider.
func NewGeminiProvider(cfg *GeminiConfig) *GeminiProvider {
	g := &GeminiProvider{model: DefaultModel}
	if cfg == nil {
		g.err = fmt.Errorf("text config is required")
		return g
	}
	g.model = defaultOr(cfg.Model, DefaultModel)
	if strings.TrimSpace(cfg.APIKey) == "" {
		g.err = fmt.Errorf("google API key is required for text generation")
		return g
	}
	client, err := newGeminiClient(context.Background(), &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		g.err = fmt.Errorf("create Gemini client: %w", err)
		return g
	}
	g.client = client
	return g
}

// Name returns the provider name.
func (g *GeminiProvider) Name() string { return provider.Gemini }

// IsAvailable reports whether the provider was initialized successfully.
func (g *GeminiProvider) IsAvailable() error {
	if g == nil {
		return fmt.Errorf("text provider is nil")
	}
	return g.err
}

// GenerateText generates a text response for the provided prompt.
func (g *GeminiProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	if g == nil {
		return "", fmt.Errorf("text provider is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if g.err != nil {
		return "", g.err
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt is required")
	}
	return geminiGenerateText(ctx, g, prompt)
}

func (g *GeminiProvider) generateText(ctx context.Context, prompt string) (string, error) {
	resp, err := g.client.Models.GenerateContent(ctx, g.model, []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("generate text: %w", err)
	}
	text := strings.TrimSpace(resp.Text())
	if text == "" {
		return "", fmt.Errorf("no text content returned")
	}
	return text, nil
}

func defaultOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
