package tts

import (
	"fmt"
	"reflect"

	"github.com/snonux/comicforge/internal/provider"
)

// Config exposes the configured TTS provider settings.
type Config interface {
	provider.TTSConfig
	GoogleAPIKey() string
	TTSModel() string
}

type registryConfig struct {
	Config
	voice string
}

// Registry resolves provider names to factories.
type Registry struct {
	*provider.Registry[provider.TTSProvider, registryConfig]
}

// NewRegistry creates an empty TTS-provider registry.
func NewRegistry() *Registry {
	return &Registry{
		Registry: provider.NewRegistry[provider.TTSProvider, registryConfig](),
	}
}

// DefaultRegistry returns the built-in TTS provider registry.
func DefaultRegistry() *Registry {
	registry := NewRegistry()
	registry.Register(provider.Gemini, func(cfg registryConfig) (provider.TTSProvider, error) {
		geminiProvider := NewGeminiProvider(&GeminiConfig{
			APIKey: cfg.GoogleAPIKey(),
			Model:  cfg.TTSModel(),
			Voice:  cfg.voice,
		})
		if err := geminiProvider.IsAvailable(); err != nil {
			return nil, err
		}
		return geminiProvider, nil
	})
	registry.Register(provider.OpenAI, func(registryConfig) (provider.TTSProvider, error) {
		return nil, fmt.Errorf("tts provider %q is not implemented", provider.OpenAI)
	})
	return registry
}

// Has reports whether a factory is registered for name.
func (r *Registry) Has(name string) bool {
	if r == nil || r.Registry == nil {
		return false
	}
	_, ok := r.Resolve(name)
	return ok
}

// NewFromConfig resolves the provider name from cfg and constructs it.
func (r *Registry) NewFromConfig(cfg Config, voice string) (provider.TTSProvider, error) {
	if r == nil || r.Registry == nil {
		return nil, fmt.Errorf("tts registry is required")
	}
	if isNilValue(cfg) {
		return nil, fmt.Errorf("tts config is required")
	}
	return r.New(cfg.TTSProviderName(), registryConfig{
		Config: cfg,
		voice:  voice,
	})
}

func isNilValue[T any](value T) bool {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
