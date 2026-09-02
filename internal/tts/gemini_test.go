package tts

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sony/gobreaker"
	"google.golang.org/genai"

	"codeberg.org/snonux/comicforge/internal/apicircuit"
	"codeberg.org/snonux/comicforge/internal/httpctx"
)

func TestNewGeminiProviderNilConfig(t *testing.T) {
	t.Parallel()

	p := NewGeminiProvider(nil)
	if err := p.IsAvailable(); err == nil {
		t.Fatal("IsAvailable() error = nil, want tts config required")
	} else if !strings.Contains(err.Error(), "tts config is required") {
		t.Fatalf("IsAvailable() error = %v, want tts config required", err)
	}
}

func TestNewGeminiProviderEmptyAPIKey(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "whitespace", key: "   "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewGeminiProvider(&GeminiConfig{APIKey: tc.key, Model: "custom-model", Voice: "Aoede"})
			if err := p.IsAvailable(); err == nil {
				t.Fatal("IsAvailable() error = nil, want google API key required")
			} else if !strings.Contains(err.Error(), "google API key is required") {
				t.Fatalf("IsAvailable() error = %v, want google API key required", err)
			}
			if p.model != "custom-model" {
				t.Fatalf("model = %q, want custom-model", p.model)
			}
		})
	}
}

func TestNewGeminiProviderSuccess(t *testing.T) {
	t.Parallel()

	p := NewGeminiProvider(&GeminiConfig{
		APIKey: "test-key",
		Model:  "tts-model",
		Voice:  "Kore",
	})
	if err := p.IsAvailable(); err != nil {
		t.Fatalf("IsAvailable() error = %v", err)
	}
	if p.client == nil {
		t.Fatal("client = nil, want non-nil")
	}
	if p.model != "tts-model" {
		t.Fatalf("model = %q, want tts-model", p.model)
	}
	if p.voice != "Kore" {
		t.Fatalf("voice = %q, want Kore", p.voice)
	}
	if got, want := p.Name(), "gemini"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}

func TestNewGeminiClientDefaultsToHTTPCtxWrapper(t *testing.T) {
	// Not parallel: asserts the package-level factory binding.
	if newGeminiClient == nil {
		t.Fatal("newGeminiClient = nil")
	}
	got := reflect.ValueOf(newGeminiClient).Pointer()
	want := reflect.ValueOf(httpctx.NewGenAIClient).Pointer()
	if got != want {
		t.Fatal("newGeminiClient is not httpctx.NewGenAIClient")
	}
}

func TestNewGeminiProviderClientFactoryError(t *testing.T) {
	orig := newGeminiClient
	t.Cleanup(func() { newGeminiClient = orig })

	wantErr := errors.New("factory boom")
	var sawKey string
	var sawBackend genai.Backend
	newGeminiClient = func(_ context.Context, cfg *genai.ClientConfig) (*genai.Client, error) {
		if cfg == nil {
			t.Fatal("cfg = nil")
		}
		sawKey = cfg.APIKey
		sawBackend = cfg.Backend
		return nil, wantErr
	}

	p := NewGeminiProvider(&GeminiConfig{APIKey: "injected-key"})
	err := p.IsAvailable()
	if err == nil {
		t.Fatal("IsAvailable() error = nil, want create Gemini client error")
	}
	if !strings.Contains(err.Error(), "create Gemini client") {
		t.Fatalf("IsAvailable() error = %v, want create Gemini client wrap", err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("IsAvailable() error = %v, want wrapped %v", err, wantErr)
	}
	if sawKey != "injected-key" {
		t.Fatalf("factory APIKey = %q, want injected-key", sawKey)
	}
	if sawBackend != genai.BackendGeminiAPI {
		t.Fatalf("factory Backend = %v, want BackendGeminiAPI", sawBackend)
	}
	if p.client != nil {
		t.Fatal("client != nil after factory failure")
	}
}

func TestGenerateAudioRejectsEmptyInputs(t *testing.T) {
	t.Parallel()

	p := NewGeminiProvider(&GeminiConfig{APIKey: "test-key"})
	if err := p.IsAvailable(); err != nil {
		t.Fatalf("IsAvailable() error = %v", err)
	}

	if err := p.GenerateAudio(context.Background(), "", "out.mp3"); err == nil {
		t.Fatal("GenerateAudio() empty text: error = nil")
	} else if !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("GenerateAudio() empty text: error = %v", err)
	}

	if err := p.GenerateAudio(context.Background(), "hello", ""); err == nil {
		t.Fatal("GenerateAudio() empty output: error = nil")
	} else if !strings.Contains(err.Error(), "output file is required") {
		t.Fatalf("GenerateAudio() empty output: error = %v", err)
	}
}

func TestGenerateAudioPropagatesInitError(t *testing.T) {
	t.Parallel()

	p := NewGeminiProvider(nil)
	err := p.GenerateAudio(context.Background(), "hello", "out.mp3")
	if err == nil {
		t.Fatal("GenerateAudio() error = nil, want init error")
	}
	if !strings.Contains(err.Error(), "tts config is required") {
		t.Fatalf("GenerateAudio() error = %v, want tts config required", err)
	}
}

func TestGenerateAudioSuccessUsesCircuit(t *testing.T) {
	origGen := geminiGenerateContent
	t.Cleanup(func() { geminiGenerateContent = origGen })

	geminiGenerateContent = func(_ context.Context, _ *genai.Client, model string, contents []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		if model != "tts-model" {
			t.Fatalf("model = %q, want tts-model", model)
		}
		if cfg == nil || len(cfg.ResponseModalities) != 1 || cfg.ResponseModalities[0] != "AUDIO" {
			t.Fatalf("unexpected config: %#v", cfg)
		}
		if len(contents) == 0 {
			t.Fatal("contents empty")
		}
		return &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				Content: &genai.Content{
					Parts: []*genai.Part{{
						InlineData: &genai.Blob{MIMEType: "audio/mpeg", Data: []byte("ID3fake")},
					}},
				},
			}},
		}, nil
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "out.mp3")
	p := &GeminiProvider{model: "tts-model", voice: "Kore"}
	if err := p.GenerateAudio(context.Background(), "hello", out); err != nil {
		t.Fatalf("GenerateAudio() error = %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "ID3fake" {
		t.Fatalf("audio bytes = %q, want ID3fake", data)
	}
}

func TestGenerateAudioCircuitOpenSkipsAPI(t *testing.T) {
	circuitRegistry = apicircuit.NewRegistry()
	t.Cleanup(func() { circuitRegistry = nil })

	origGen := geminiGenerateContent
	t.Cleanup(func() { geminiGenerateContent = origGen })

	var calls int
	apiErr := errors.New("tts api down")
	geminiGenerateContent = func(context.Context, *genai.Client, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		calls++
		return nil, apiErr
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "out.mp3")
	p := &GeminiProvider{model: DefaultModel, voice: "Aoede"}
	for i := 0; i < 5; i++ {
		err := p.GenerateAudio(context.Background(), "hello", out)
		if err == nil {
			t.Fatalf("failure %d: error = nil", i+1)
		}
		if !errors.Is(err, apiErr) {
			t.Fatalf("failure %d: error = %v, want wrapped %v", i+1, err, apiErr)
		}
	}

	callsBeforeOpen := calls
	err := p.GenerateAudio(context.Background(), "hello", out)
	if err == nil {
		t.Fatal("open circuit: error = nil")
	}
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("open circuit: error = %v, want ErrOpenState", err)
	}
	if calls != callsBeforeOpen {
		t.Fatalf("open circuit still called API: calls = %d, want %d", calls, callsBeforeOpen)
	}
}
