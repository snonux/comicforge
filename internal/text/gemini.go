package text

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"

	"codeberg.org/snonux/comicforge/internal/apicircuit"
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

	// newClient overrides the GenAI client factory; nil uses httpctx.NewGenAIClient.
	newClient func(context.Context, *genai.ClientConfig) (*genai.Client, error)
}

// GeminiProvider implements TextProvider for Google Gemini text generation.
type GeminiProvider struct {
	client       *genai.Client
	model        string
	err          error
	generateText geminiTextGenerator
}

type geminiTextGenerator func(ctx context.Context, p *GeminiProvider, prompt string) (string, error)

var _ provider.TextProvider = (*GeminiProvider)(nil)

func defaultGeminiGenerateText(ctx context.Context, p *GeminiProvider, prompt string) (string, error) {
	return p.generateTextImpl(ctx, prompt)
}

var geminiGenerateContent = func(ctx context.Context, client *genai.Client, model string, contents []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return client.Models.GenerateContent(ctx, model, contents, cfg)
}

// circuitRegistry overrides the breaker registry in tests; nil uses apicircuit.DefaultRegistry.
var circuitRegistry *apicircuit.Registry

// NewGeminiProvider creates a Gemini text provider.
func NewGeminiProvider(cfg *GeminiConfig) *GeminiProvider {
	g := &GeminiProvider{
		model:        DefaultModel,
		generateText: defaultGeminiGenerateText,
	}
	if cfg == nil {
		g.err = fmt.Errorf("text config is required")
		return g
	}
	g.model = defaultOr(cfg.Model, DefaultModel)
	if strings.TrimSpace(cfg.APIKey) == "" {
		g.err = fmt.Errorf("google API key is required for text generation")
		return g
	}
	newClient := httpctx.NewGenAIClient
	if cfg.newClient != nil {
		newClient = cfg.newClient
	}
	client, err := newClient(context.Background(), &genai.ClientConfig{
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
	fn := g.generateText
	if fn == nil {
		fn = defaultGeminiGenerateText
	}
	return fn(ctx, g, prompt)
}

func (g *GeminiProvider) generateTextImpl(ctx context.Context, prompt string) (string, error) {
	resp, err := apicircuit.Execute(circuitRegistry, g.Name(), apicircuit.CapabilityText, func() (*genai.GenerateContentResponse, error) {
		return geminiGenerateContent(ctx, g.client, g.model, []*genai.Content{
			genai.NewContentFromText(prompt, genai.RoleUser),
		}, nil)
	})
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
