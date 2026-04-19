package tts

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/genai"

	"codeberg.org/snonux/comicforge/internal/provider"
)

const (
	// DefaultModel is the Gemini TTS model used for narration.
	DefaultModel = "gemini-2.5-flash-preview-tts"
)

// GeminiConfig configures the Gemini TTS provider.
type GeminiConfig struct {
	APIKey string
	Model  string
	Voice  string
}

// GeminiProvider generates MP3 audio with Gemini TTS.
type GeminiProvider struct {
	client *genai.Client
	model  string
	voice  string
	err    error
}

var _ provider.TTSProvider = (*GeminiProvider)(nil)

// NewGeminiProvider creates a Gemini TTS provider.
func NewGeminiProvider(cfg *GeminiConfig) *GeminiProvider {
	g := &GeminiProvider{model: DefaultModel}
	if cfg == nil {
		g.err = fmt.Errorf("tts config is required")
		return g
	}
	g.model = defaultOr(cfg.Model, DefaultModel)
	g.voice = cfg.Voice
	if strings.TrimSpace(cfg.APIKey) == "" {
		g.err = fmt.Errorf("Google API key is required for TTS")
		return g
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
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
func (g *GeminiProvider) Name() string { return "gemini" }

// IsAvailable reports whether the provider was initialized successfully.
func (g *GeminiProvider) IsAvailable() error {
	if g == nil {
		return fmt.Errorf("tts provider is nil")
	}
	return g.err
}

// GenerateAudio writes MP3 audio for the provided text to outputFile.
func (g *GeminiProvider) GenerateAudio(ctx context.Context, text, outputFile string) error {
	if g == nil {
		return fmt.Errorf("tts provider is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if g.err != nil {
		return g.err
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("text is required")
	}
	if strings.TrimSpace(outputFile) == "" {
		return fmt.Errorf("output file is required")
	}
	voiceName := g.voice
	if strings.TrimSpace(voiceName) == "" {
		voiceName = "Aoede"
	}

	speechCfg := &genai.SpeechConfig{
		VoiceConfig: &genai.VoiceConfig{
			PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{VoiceName: voiceName},
		},
		LanguageCode: "bg-BG",
	}
	resp, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(text), &genai.GenerateContentConfig{
		ResponseModalities: []string{"AUDIO"},
		SpeechConfig:       speechCfg,
	})
	if err != nil {
		return fmt.Errorf("generate audio: %w", err)
	}
	data, mimeType, err := extractAudio(resp)
	if err != nil {
		return err
	}
	if strings.TrimSpace(outputFile) == "" {
		return fmt.Errorf("output file is required")
	}
	if err := os.WriteFile(outputFile, data, 0o644); err != nil {
		return fmt.Errorf("write audio: %w", err)
	}
	if mimeType != "" && !strings.HasPrefix(mimeType, "audio/") {
		return fmt.Errorf("unexpected audio mime type %q", mimeType)
	}
	return nil
}

func extractAudio(resp *genai.GenerateContentResponse) ([]byte, string, error) {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0] == nil || resp.Candidates[0].Content == nil {
		return nil, "", fmt.Errorf("no audio returned")
	}
	for _, part := range resp.Candidates[0].Content.Parts {
		if part != nil && part.InlineData != nil && len(part.InlineData.Data) > 0 {
			return part.InlineData.Data, part.InlineData.MIMEType, nil
		}
	}
	return nil, "", fmt.Errorf("audio payload missing")
}

func defaultOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
