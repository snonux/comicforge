package image

import (
	"fmt"
	"reflect"
	"sync"
)

// Factory builds a provider instance.
type Factory[C any] func(C) (ImageProvider, error)

// Config exposes the configured image provider name.
type Config interface {
	ImageProviderName() string
	GoogleAPIKey() string
	ImageModel() string
	ImageTextModel() string
}

// Registry resolves provider names to factories.
type Registry[C Config] struct {
	mu        sync.RWMutex
	factories map[string]Factory[C]
}

// NewRegistry creates an empty provider registry.
func NewRegistry[C Config]() *Registry[C] {
	return &Registry[C]{factories: make(map[string]Factory[C])}
}

// Register associates name with a factory. Later registrations replace earlier ones.
func (r *Registry[C]) Register(name string, factory Factory[C]) {
	if r == nil || factory == nil {
		return
	}

	normalized := NormalizeName(name)
	if normalized == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.factories == nil {
		r.factories = make(map[string]Factory[C])
	}
	r.factories[normalized] = factory
}

// Resolve returns the factory registered for name.
func (r *Registry[C]) Resolve(name string) (Factory[C], bool) {
	if r == nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.factories[NormalizeName(name)]
	return factory, ok
}

// Has reports whether a factory is registered for name.
func (r *Registry[C]) Has(name string) bool {
	_, ok := r.Resolve(name)
	return ok
}

// New constructs a provider for name.
func (r *Registry[C]) New(name string, cfg C) (ImageProvider, error) {
	var zero ImageProvider
	factory, ok := r.Resolve(name)
	if !ok {
		return zero, fmt.Errorf("%w: %s", ErrUnknownProvider, name)
	}
	return factory(cfg)
}

// NewFromConfig resolves the provider name from cfg and constructs it.
func (r *Registry[C]) NewFromConfig(cfg C) (ImageProvider, error) {
	if isNilValue(cfg) {
		return nil, fmt.Errorf("image config is required")
	}
	return r.New(cfg.ImageProviderName(), cfg)
}

// DefaultRegistry returns the built-in image provider registry.
func DefaultRegistry() *Registry[Config] {
	registry := NewRegistry[Config]()
	registry.Register(Gemini, func(cfg Config) (ImageProvider, error) {
		geminiProvider := NewGeminiProvider(&GeminiConfig{
			APIKey:    cfg.GoogleAPIKey(),
			Model:     cfg.ImageModel(),
			TextModel: cfg.ImageTextModel(),
		})
		if err := geminiProvider.IsAvailable(); err != nil {
			return nil, err
		}
		return geminiProvider, nil
	})
	registry.Register(OpenAI, func(Config) (ImageProvider, error) {
		return nil, fmt.Errorf("image provider %q is not implemented", OpenAI)
	})
	return registry
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
