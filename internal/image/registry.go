package image

import (
	"fmt"
	"reflect"

	"github.com/snonux/comicforge/internal/provider"
)

// Config exposes the configured image provider settings.
type Config interface {
	provider.ImageConfig
	GoogleAPIKey() string
	ImageModel() string
	ImageTextModel() string
	ComicAspectRatio() string
}

// Registry resolves provider names to factories.
type Registry struct {
	*provider.Registry[Provider, Config]
}

// NewRegistry creates an empty image-provider registry.
func NewRegistry() *Registry {
	return &Registry{
		Registry: provider.NewRegistry[Provider, Config](),
	}
}

// DefaultRegistry returns the built-in image provider registry.
func DefaultRegistry() *Registry {
	registry := NewRegistry()
	registry.Register(Gemini, func(cfg Config) (Provider, error) {
		geminiProvider := NewGeminiProvider(&GeminiConfig{
			APIKey:      cfg.GoogleAPIKey(),
			Model:       cfg.ImageModel(),
			TextModel:   cfg.ImageTextModel(),
			AspectRatio: cfg.ComicAspectRatio(),
		})
		if err := geminiProvider.IsAvailable(); err != nil {
			return nil, err
		}
		return geminiProvider, nil
	})
	registry.Register(OpenAI, func(Config) (Provider, error) {
		return nil, fmt.Errorf("image provider %q is not implemented", OpenAI)
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
func (r *Registry) NewFromConfig(cfg Config) (Provider, error) {
	if r == nil || r.Registry == nil {
		return nil, fmt.Errorf("image registry is required")
	}
	if isNilValue(cfg) {
		return nil, fmt.Errorf("image config is required")
	}
	return r.New(cfg.ImageProviderName(), cfg)
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
