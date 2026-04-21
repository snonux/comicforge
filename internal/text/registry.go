package text

import (
	"fmt"
	"reflect"

	"codeberg.org/snonux/comicforge/internal/provider"
)

// Config exposes the configured text provider settings.
type Config interface {
	provider.TextConfig
	GoogleAPIKey() string
	TextModel() string
}

// Registry resolves provider names to factories.
type Registry struct {
	*provider.Registry[provider.TextProvider, Config]
}

// NewRegistry creates an empty text-provider registry.
func NewRegistry() *Registry {
	return &Registry{
		Registry: provider.NewRegistry[provider.TextProvider, Config](),
	}
}

// DefaultRegistry returns the built-in text provider registry.
func DefaultRegistry() *Registry {
	registry := NewRegistry()
	registry.Register(provider.Gemini, func(cfg Config) (provider.TextProvider, error) {
		geminiProvider := NewGeminiProvider(&GeminiConfig{
			APIKey: cfg.GoogleAPIKey(),
			Model:  cfg.TextModel(),
		})
		if err := geminiProvider.IsAvailable(); err != nil {
			return nil, err
		}
		return geminiProvider, nil
	})
	registry.Register(provider.OpenAI, func(Config) (provider.TextProvider, error) {
		return nil, fmt.Errorf("text provider %q is not implemented", provider.OpenAI)
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
func (r *Registry) NewFromConfig(cfg Config) (provider.TextProvider, error) {
	if r == nil || r.Registry == nil {
		return nil, fmt.Errorf("text registry is required")
	}
	if isNilValue(cfg) {
		return nil, fmt.Errorf("text config is required")
	}
	return r.New(cfg.TextProviderName(), cfg)
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
