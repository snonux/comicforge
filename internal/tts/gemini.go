package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"google.golang.org/genai"

	"codeberg.org/snonux/comicforge/internal/apicircuit"
	"codeberg.org/snonux/comicforge/internal/httpctx"
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

var newGeminiClient = httpctx.NewGenAIClient
var geminiGenerateContent = func(ctx context.Context, client *genai.Client, model string, contents []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return client.Models.GenerateContent(ctx, model, contents, cfg)
}

// circuitRegistry overrides the breaker registry in tests; nil uses apicircuit.DefaultRegistry.
var circuitRegistry *apicircuit.Registry

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
		g.err = fmt.Errorf("google API key is required for TTS")
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
	resp, err := apicircuit.Execute(circuitRegistry, g.Name(), apicircuit.CapabilityTTS, func() (*genai.GenerateContentResponse, error) {
		return geminiGenerateContent(ctx, g.client, g.model, genai.Text(text), &genai.GenerateContentConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig:       speechCfg,
		})
	})
	if err != nil {
		return fmt.Errorf("generate audio: %w", err)
	}
	data, mimeType, err := extractAudio(resp)
	if err != nil {
		return err
	}
	if err := writeAudioFile(data, mimeType, outputFile); err != nil {
		return err
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

func writeAudioFile(data []byte, mimeType, outputFile string) error {
	if strings.TrimSpace(outputFile) == "" {
		return fmt.Errorf("output file is required")
	}
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(lowerMime, "audio/mpeg"), strings.HasPrefix(lowerMime, "audio/mp3"):
		if err := os.WriteFile(outputFile, data, 0o644); err != nil {
			return fmt.Errorf("write audio: %w", err)
		}
		return nil
	case strings.Contains(lowerMime, "audio/l16"), strings.Contains(lowerMime, "codec=pcm"), lowerMime == "":
		return encodePCMToMP3(data, lowerMime, outputFile)
	default:
		return fmt.Errorf("unsupported audio mime type %q", mimeType)
	}
}

func encodePCMToMP3(data []byte, mimeType, outputFile string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found for audio conversion: %w", err)
	}

	rate := 24000
	if idx := strings.Index(mimeType, "rate="); idx >= 0 {
		value := mimeType[idx+len("rate="):]
		for i, r := range value {
			if r < '0' || r > '9' {
				value = value[:i]
				break
			}
		}
		if parsed, parseErr := strconv.Atoi(value); parseErr == nil && parsed > 0 {
			rate = parsed
		}
	}

	tmpDir := filepath.Dir(outputFile)
	rawFile, err := os.CreateTemp(tmpDir, "comicforge-tts-*.pcm")
	if err != nil {
		return fmt.Errorf("create temporary audio file: %w", err)
	}
	rawPath := rawFile.Name()
	if _, err := rawFile.Write(data); err != nil {
		_ = rawFile.Close()
		_ = os.Remove(rawPath)
		return fmt.Errorf("write temporary audio file: %w", err)
	}
	if err := rawFile.Close(); err != nil {
		_ = os.Remove(rawPath)
		return fmt.Errorf("close temporary audio file: %w", err)
	}
	defer func() {
		_ = os.Remove(rawPath)
	}()

	cmd := exec.Command(ffmpegPath,
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "s16le",
		"-ar", fmt.Sprintf("%d", rate),
		"-ac", "1",
		"-i", rawPath,
		"-codec:a", "libmp3lame",
		"-q:a", "2",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert PCM audio to mp3: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func defaultOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
