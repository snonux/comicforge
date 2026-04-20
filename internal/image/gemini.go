package image

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"

	"codeberg.org/snonux/comicforge/internal/apicircuit"
	"codeberg.org/snonux/comicforge/internal/httpctx"
)

const (
	// DefaultGeminiImageModel is the Gemini image model used for image generation.
	DefaultGeminiImageModel = "gemini-3.1-flash-image-preview"

	// DefaultGeminiTextModel is the Gemini text model used for scene generation.
	DefaultGeminiTextModel = "gemini-2.5-flash"

	geminiAspectRatio = "4:3"
	geminiDataPrefix  = "data:image/png;base64,"
	geminiSource      = Gemini
	maxCustomPrompt   = 4000
)

// GeminiConfig holds the settings needed to build a Gemini-backed image provider.
type GeminiConfig struct {
	APIKey    string
	Model     string
	TextModel string
}

// GeminiProvider implements ImageProvider for Google Gemini image generation.
type GeminiProvider struct {
	client     *genai.Client
	initErr    error
	config     *GeminiConfig
	lastPrompt string

	// PromptCallback runs after the prompt is generated and before image creation.
	PromptCallback func(prompt string)
}

var _ ImageProvider = (*GeminiProvider)(nil)

var imageHTTPClient = httpctx.ImageDownloadHTTPClient()
var newGeminiClient = httpctx.NewGenAIClient
var geminiGenerateText = func(ctx context.Context, c *GeminiProvider, model, systemPrompt, userPrompt string, temperature float32, maxOutputTokens int32) (string, error) {
	return c.generateText(ctx, model, systemPrompt, userPrompt, temperature, maxOutputTokens)
}
var geminiGenerateImage = func(ctx context.Context, c *GeminiProvider, prompt, aspectRatio string) ([]byte, string, error) {
	return c.generateImage(ctx, prompt, aspectRatio)
}

// NewGeminiProvider creates a new Gemini image provider.
func NewGeminiProvider(config *GeminiConfig) *GeminiProvider {
	normalized := normalizeGeminiConfig(config)
	client := &GeminiProvider{config: normalized}

	if normalized.APIKey == "" {
		return client
	}

	genaiClient, err := newGeminiClient(context.Background(), &genai.ClientConfig{
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

	aspectRatio := geminiAspectRatio
	if opts.AspectRatio != "" {
		aspectRatio = opts.AspectRatio
	}

	fmt.Printf("Gemini Image Generation Prompt (%d chars): %s\n", len(prompt), prompt)
	fmt.Printf("Gemini Image Generation: Using model %q with aspect ratio %q\n", c.modelName(), aspectRatio)

	var imageBytes []byte
	var mimeType string
	if len(opts.ReferenceImages) > 0 {
		imageBytes, mimeType, err = c.generateImageWithRefs(ctx, prompt, aspectRatio, opts.ReferenceImages)
	} else {
		imageBytes, mimeType, err = geminiGenerateImage(ctx, c, prompt, aspectRatio)
	}
	if err != nil {
		if searchErr, ok := err.(*SearchError); ok {
			return nil, searchErr
		}
		return nil, &SearchError{
			Provider: geminiSource,
			Code:     "API_ERROR",
			Message:  fmt.Sprintf("failed to generate image: %v", err),
		}
	}

	dataURL, err := encodeDataURL(imageBytes, mimeType)
	if err != nil {
		return nil, err
	}
	width, height, err := decodedImageDimensions(imageBytes)
	if err != nil {
		return nil, err
	}

	description := fmt.Sprintf("Generated educational image for %s", opts.Query)
	if translatedWord != "" {
		description = fmt.Sprintf("%s (%s)", description, translatedWord)
	}

	result := SearchResult{
		ID:           c.generateImageID(opts.Query),
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
	return c.GenerateImageWithReferences(ctx, prompt, outputFile, nil)
}

// GenerateImageWithReferences renders the first generated image to outputFile,
// optionally conditioning the model on prior page images so comic pages stay
// visually consistent across the full PDF.
func (c *GeminiProvider) GenerateImageWithReferences(ctx context.Context, prompt, outputFile string, refs [][]byte) error {
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
	defer rc.Close()
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

	resp, err := imageHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return nil, fmt.Errorf("HTTP %d: %s (failed to close response body: %v)", resp.StatusCode, resp.Status, closeErr)
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

func (c *GeminiProvider) resolveTranslation(_ context.Context, opts *SearchOptions, translation string) (string, error) {
	if translation != "" {
		fmt.Printf("Using provided translation: %s -> %s\n", opts.Query, translation)
		return translation, nil
	}

	return opts.Query, nil
}

func (c *GeminiProvider) resolvePrompt(ctx context.Context, opts *SearchOptions, translatedWord string) (string, error) {
	if customPrompt := strings.TrimSpace(opts.CustomPrompt); customPrompt != "" {
		if len(customPrompt) > maxCustomPrompt {
			customPrompt = customPrompt[:maxCustomPrompt-3] + "..."
		}
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
	if customPrompt := strings.TrimSpace(opts.CustomPrompt); customPrompt != "" {
		if len(customPrompt) > maxCustomPrompt {
			customPrompt = customPrompt[:maxCustomPrompt-3] + "..."
		}
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

func (c *GeminiProvider) generateText(ctx context.Context, model, systemPrompt, userPrompt string, temperature float32, maxOutputTokens int32) (string, error) {
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

func (c *GeminiProvider) generateImage(ctx context.Context, prompt, aspectRatio string) ([]byte, string, error) {
	if aspectRatio == "" {
		aspectRatio = geminiAspectRatio
	}

	cfg := &genai.GenerateContentConfig{
		ResponseModalities: []string{string(genai.ModalityImage)},
		ImageConfig: &genai.ImageConfig{
			AspectRatio: aspectRatio,
		},
	}

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
	if aspectRatio == "" {
		aspectRatio = geminiAspectRatio
	}

	cfg := &genai.GenerateContentConfig{
		ResponseModalities: []string{string(genai.ModalityImage)},
		ImageConfig:        &genai.ImageConfig{AspectRatio: aspectRatio},
	}

	parts := make([]*genai.Part, 0, len(refs)+1)
	for _, ref := range refs {
		if len(ref) > 0 {
			parts = append(parts, &genai.Part{
				InlineData: &genai.Blob{MIMEType: "image/png", Data: ref},
			})
		}
	}
	refNote := fmt.Sprintf(
		"The %d reference image(s) above show the exact character appearance that must be preserved. "+
			"Every character, animal, or object must look identical in the new image. Now generate:\n\n",
		len(refs),
	)
	parts = append(parts, &genai.Part{Text: refNote + prompt})

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

func encodeDataURL(imageBytes []byte, mimeType string) (string, error) {
	if len(imageBytes) == 0 {
		return "", fmt.Errorf("no image bytes returned")
	}

	normalizedBytes, err := normalizePNG(imageBytes, mimeType)
	if err != nil {
		return "", err
	}

	return geminiDataPrefix + base64.StdEncoding.EncodeToString(normalizedBytes), nil
}

func decodeDataURL(url string) (io.ReadCloser, error) {
	header, payload, ok := strings.Cut(url, ",")
	if !ok || !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return nil, fmt.Errorf("unsupported data URI: %s", url)
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode data URI: %w", err)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func normalizePNG(imageBytes []byte, mimeType string) ([]byte, error) {
	if strings.EqualFold(strings.TrimSpace(mimeType), "image/png") {
		return append([]byte(nil), imageBytes...), nil
	}

	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("decode generated image: %w", err)
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return nil, fmt.Errorf("encode generated image as png: %w", err)
	}

	return buffer.Bytes(), nil
}

func decodedImageDimensions(imageBytes []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		return 0, 0, fmt.Errorf("decode generated image dimensions: %w", err)
	}

	return cfg.Width, cfg.Height, nil
}

func (c *GeminiProvider) generateImageID(word string) string {
	hash := md5.Sum([]byte(word))
	return hex.EncodeToString(hash[:])[:8]
}

func (c *GeminiProvider) buildAttribution(result *SearchResult, prompt string) string {
	if result == nil {
		return ""
	}

	var attribution strings.Builder
	attribution.WriteString("Image generated by Google Gemini Nano Banana\n\n")
	fmt.Fprintf(&attribution, "Model: %s\n", c.modelName())
	fmt.Fprintf(&attribution, "Text model: %s\n", c.textModelName())
	fmt.Fprintf(&attribution, "Aspect ratio: %s\n", geminiAspectRatio)
	fmt.Fprintf(&attribution, "Size: %dx%d\n", result.Width, result.Height)
	if result.Description != "" {
		fmt.Fprintf(&attribution, "Result: %s\n", result.Description)
	}
	fmt.Fprintf(&attribution, "\nPrompt used:\n%s\n", prompt)
	fmt.Fprintf(&attribution, "\nGenerated at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	return attribution.String()
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
