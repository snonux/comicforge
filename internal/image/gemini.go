package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"google.golang.org/genai"

	"github.com/snonux/comicforge/internal/httpctx"
)

const (
	// DefaultGeminiImageModel is the Gemini image model used for image generation.
	DefaultGeminiImageModel = "gemini-3.1-flash-image-preview"

	// DefaultGeminiTextModel is the Gemini text model used for scene generation.
	DefaultGeminiTextModel = "gemini-2.5-flash"

	geminiAspectRatio = "4:3"
	geminiDataPrefix  = "data:image/png;base64,"
	geminiSource      = Gemini
)

// GeminiConfig holds the settings needed to build a Gemini-backed image provider.
type GeminiConfig struct {
	APIKey      string
	Model       string
	TextModel   string
	AspectRatio string

	// newClient overrides the GenAI client factory; nil uses httpctx.NewGenAIClient.
	newClient func(context.Context, *genai.ClientConfig) (*genai.Client, error)
}

// GeminiProvider implements Provider for Google Gemini image generation.
type GeminiProvider struct {
	client     *genai.Client
	initErr    error
	config     *GeminiConfig
	lastPrompt string

	httpClient    httpDoer
	generateText  geminiTextGenerator
	generateImage geminiImageGenerator

	// PromptCallback runs after the prompt is generated and before image creation.
	PromptCallback func(prompt string)
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type geminiTextGenerator func(ctx context.Context, c *GeminiProvider, model, systemPrompt, userPrompt string, temperature float32, maxOutputTokens int32) (string, error)

type geminiImageGenerator func(ctx context.Context, c *GeminiProvider, prompt, aspectRatio string) ([]byte, string, error)

var _ Provider = (*GeminiProvider)(nil)

func defaultGeminiGenerateText(ctx context.Context, c *GeminiProvider, model, systemPrompt, userPrompt string, temperature float32, maxOutputTokens int32) (string, error) {
	return c.generateTextImpl(ctx, model, systemPrompt, userPrompt, temperature, maxOutputTokens)
}

func defaultGeminiGenerateImage(ctx context.Context, c *GeminiProvider, prompt, aspectRatio string) ([]byte, string, error) {
	return c.generateImageImpl(ctx, prompt, aspectRatio)
}

// NewGeminiProvider creates a new Gemini image provider.
func NewGeminiProvider(config *GeminiConfig) *GeminiProvider {
	normalized := normalizeGeminiConfig(config)
	client := &GeminiProvider{
		config:        normalized,
		httpClient:    httpctx.ImageDownloadHTTPClient(),
		generateText:  defaultGeminiGenerateText,
		generateImage: defaultGeminiGenerateImage,
	}

	if normalized.APIKey == "" {
		return client
	}

	newClient := httpctx.NewGenAIClient
	if normalized.newClient != nil {
		newClient = normalized.newClient
	}
	genaiClient, err := newClient(context.Background(), &genai.ClientConfig{
		APIKey:  normalized.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		client.initErr = err
		return client
	}

	client.client = genaiClient
	return client
}

// Search generates an educational image for the requested word or phrase.
func (c *GeminiProvider) Search(ctx context.Context, opts *SearchOptions) ([]SearchResult, error) {
	ctx, cancel := httpctx.WithTimeoutUnlessSet(ctx, httpctx.OperationTimeoutDefault)
	defer cancel()

	if err := c.ensureReady(); err != nil {
		return nil, err
	}
	if opts == nil {
		return nil, &SearchError{
			Provider: geminiSource,
			Code:     "INVALID_OPTIONS",
			Message:  "search options are required",
		}
	}

	prompt, translatedWord, err := c.buildPrompt(ctx, opts)
	if err != nil {
		return nil, err
	}
	c.lastPrompt = prompt
	if c.PromptCallback != nil {
		c.PromptCallback(prompt)
	}

	aspectRatio := c.resolveSearchAspectRatio(opts)
	fmt.Printf("Gemini Image Generation Prompt (%d chars): %s\n", len(prompt), prompt)
	fmt.Printf("Gemini Image Generation: Using model %q with aspect ratio %q\n", c.modelName(), aspectRatio)

	imageBytes, mimeType, err := c.generateSearchImage(ctx, prompt, aspectRatio, opts.ReferenceImages)
	if err != nil {
		return nil, err
	}
	return c.assembleSearchResult(opts.Query, translatedWord, prompt, imageBytes, mimeType)
}

func (c *GeminiProvider) resolveSearchAspectRatio(opts *SearchOptions) string {
	aspectRatio := geminiAspectRatio
	if c.config != nil && strings.TrimSpace(c.config.AspectRatio) != "" {
		aspectRatio = strings.TrimSpace(c.config.AspectRatio)
	}
	if opts != nil && opts.AspectRatio != "" {
		aspectRatio = opts.AspectRatio
	}
	return aspectRatio
}

func (c *GeminiProvider) generateSearchImage(ctx context.Context, prompt, aspectRatio string, refs [][]byte) ([]byte, string, error) {
	var imageBytes []byte
	var mimeType string
	var err error
	if len(refs) > 0 {
		imageBytes, mimeType, err = c.generateImageWithRefs(ctx, prompt, aspectRatio, refs)
	} else {
		genImage := c.generateImage
		if genImage == nil {
			genImage = defaultGeminiGenerateImage
		}
		imageBytes, mimeType, err = genImage(ctx, c, prompt, aspectRatio)
	}
	if err != nil {
		var searchErr *SearchError
		if errors.As(err, &searchErr) {
			return nil, "", searchErr
		}
		return nil, "", &SearchError{
			Provider: geminiSource,
			Code:     "API_ERROR",
			Message:  fmt.Sprintf("failed to generate image: %v", err),
		}
	}
	return imageBytes, mimeType, nil
}

func (c *GeminiProvider) assembleSearchResult(query, translatedWord, prompt string, imageBytes []byte, mimeType string) ([]SearchResult, error) {
	dataURL, err := encodeDataURL(imageBytes, mimeType)
	if err != nil {
		return nil, err
	}
	width, height, err := decodedImageDimensions(imageBytes)
	if err != nil {
		return nil, err
	}

	description := fmt.Sprintf("Generated educational image for %s", query)
	if translatedWord != "" {
		description = fmt.Sprintf("%s (%s)", description, translatedWord)
	}

	result := SearchResult{
		ID:           c.generateImageID(query),
		URL:          dataURL,
		ThumbnailURL: dataURL,
		Width:        width,
		Height:       height,
		Description:  description,
		Source:       geminiSource,
	}
	result.Attribution = c.buildAttribution(&result, prompt)
	return []SearchResult{result}, nil
}

// IsAvailable reports whether the provider was initialized successfully.
func (c *GeminiProvider) IsAvailable() error {
	return c.ensureReady()
}

// GenerateImage renders the first generated image to outputFile.
func (c *GeminiProvider) GenerateImage(ctx context.Context, prompt, outputFile string) error {
	return c.GenerateImageWithAspectRatio(ctx, prompt, outputFile, "")
}

// GenerateImageWithAspectRatio renders the first generated image to outputFile
// using the configured aspect ratio or the supplied override.
func (c *GeminiProvider) GenerateImageWithAspectRatio(ctx context.Context, prompt, outputFile string, aspectRatio string) error {
	return c.GenerateImageWithReferencesAndAspectRatio(ctx, prompt, outputFile, nil, aspectRatio)
}

// GenerateImageWithReferences renders the first generated image to outputFile,
// optionally conditioning the model on prior page images so comic pages stay
// visually consistent across the full PDF.
func (c *GeminiProvider) GenerateImageWithReferences(ctx context.Context, prompt, outputFile string, refs [][]byte) error {
	return c.GenerateImageWithReferencesAndAspectRatio(ctx, prompt, outputFile, refs, "")
}

// GenerateImageWithReferencesAndAspectRatio renders the first generated image
// to outputFile, optionally conditioning on reference images and overriding the
// aspect ratio.
func (c *GeminiProvider) GenerateImageWithReferencesAndAspectRatio(ctx context.Context, prompt, outputFile string, refs [][]byte, aspectRatio string) error {
	if c == nil {
		return fmt.Errorf("image provider is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if strings.TrimSpace(outputFile) == "" {
		return fmt.Errorf("output file is required")
	}
	opts := &SearchOptions{CustomPrompt: prompt}
	if len(refs) > 0 {
		opts.ReferenceImages = refs
	}
	if strings.TrimSpace(aspectRatio) != "" {
		opts.AspectRatio = strings.TrimSpace(aspectRatio)
	}
	results, err := c.Search(ctx, opts)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no image results returned")
	}
	rc, err := c.Download(ctx, results[0].URL)
	if err != nil {
		return err
	}
	defer func() {
		_ = rc.Close()
	}()
	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("read image data: %w", err)
	}
	if err := os.WriteFile(outputFile, data, 0o644); err != nil {
		return fmt.Errorf("write image: %w", err)
	}
	return nil
}

// Download returns the image bytes for a data URI or a remote URL.
func (c *GeminiProvider) Download(ctx context.Context, url string) (io.ReadCloser, error) {
	if strings.HasPrefix(url, geminiDataPrefix) {
		return decodeDataURL(url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = httpctx.ImageDownloadHTTPClient()
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return nil, fmt.Errorf("HTTP %d: %s (failed to close response body: %w)", resp.StatusCode, resp.Status, closeErr)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}

// GetAttribution returns attribution text for the generated image.
func (c *GeminiProvider) GetAttribution(result *SearchResult) string {
	if result == nil {
		return ""
	}

	return result.Attribution
}

// Name returns the provider name.
func (c *GeminiProvider) Name() string {
	return geminiSource
}

// LastPrompt returns the most recent image prompt.
func (c *GeminiProvider) LastPrompt() string {
	return c.lastPrompt
}

// SetPromptCallback registers a callback that runs after prompt generation.
func (c *GeminiProvider) SetPromptCallback(callback func(prompt string)) {
	c.PromptCallback = callback
}

func (c *GeminiProvider) ensureReady() error {
	if c == nil || c.config == nil {
		return &SearchError{
			Provider: geminiSource,
			Code:     "NO_CONFIG",
			Message:  "Gemini client not initialized",
		}
	}
	if c.config.APIKey == "" {
		return &SearchError{
			Provider: geminiSource,
			Code:     "NO_API_KEY",
			Message:  "Google API key not configured",
		}
	}
	if c.initErr != nil {
		return &SearchError{
			Provider: geminiSource,
			Code:     "CLIENT_INIT_FAILED",
			Message:  fmt.Sprintf("failed to initialize client: %v", c.initErr),
		}
	}
	if c.client == nil {
		return &SearchError{
			Provider: geminiSource,
			Code:     "CLIENT_NOT_READY",
			Message:  "Gemini client not initialized",
		}
	}

	return nil
}

func (c *GeminiProvider) modelName() string {
	if c == nil || c.config == nil || strings.TrimSpace(c.config.Model) == "" {
		return DefaultGeminiImageModel
	}

	return c.config.Model
}

func (c *GeminiProvider) textModelName() string {
	if c == nil || c.config == nil || strings.TrimSpace(c.config.TextModel) == "" {
		return DefaultGeminiTextModel
	}

	return c.config.TextModel
}

func normalizeGeminiConfig(config *GeminiConfig) *GeminiConfig {
	normalized := &GeminiConfig{}
	if config != nil {
		*normalized = *config
	}

	normalized.APIKey = strings.TrimSpace(normalized.APIKey)
	normalized.Model = strings.TrimSpace(normalized.Model)
	normalized.TextModel = strings.TrimSpace(normalized.TextModel)

	if normalized.Model == "" {
		normalized.Model = DefaultGeminiImageModel
	}
	if normalized.TextModel == "" {
		normalized.TextModel = DefaultGeminiTextModel
	}

	return normalized
}
