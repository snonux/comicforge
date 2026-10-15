package tts

import (
	"errors"
	"strings"
	"testing"

	"github.com/snonux/comicforge/internal/provider"
)

func TestDefaultRegistryNewFromConfig(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry()
	gotProvider, err := registry.NewFromConfig(fakeConfig{
		name:   provider.Gemini,
		apiKey: "test-key",
		model:  "gemini-2.5-flash-preview-tts",
	}, "Aoede")
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if got, want := gotProvider.Name(), provider.Gemini; got != want {
		t.Fatalf("provider.Name() = %q, want %q", got, want)
	}
}

func TestDefaultRegistryUnsupportedProvider(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry()
	_, err := registry.NewFromConfig(fakeConfig{
		name:   provider.OpenAI,
		apiKey: "test-key",
		model:  "unused",
	}, "Aoede")
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if !strings.Contains(err.Error(), "tts provider \"openai\" is not implemented") {
		t.Fatalf("error = %v, want unsupported provider error", err)
	}
}

func TestRegistryUnknownProvider(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry()
	_, err := registry.New("missing", registryConfig{Config: fakeConfig{}})
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
	if !errors.Is(err, provider.ErrUnknownProvider) {
		t.Fatalf("error = %v, want ErrUnknownProvider", err)
	}
}

type fakeConfig struct {
	name   string
	apiKey string
	model  string
}

func (f fakeConfig) TTSProviderName() string { return f.name }
func (f fakeConfig) GoogleAPIKey() string    { return f.apiKey }
func (f fakeConfig) TTSModel() string        { return f.model }
func (f fakeConfig) StoryLanguage() string   { return "Bulgarian" }
