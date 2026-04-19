package image

import (
	"fmt"
	"sync"
)

// Factory builds a provider instance.
type Factory func() (ImageProvider, error)

// Config exposes the configured image provider name.
type Config interface {
	ImageProviderName() string
}

// Registry resolves provider names to factories.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register associates name with a factory. Later registrations replace earlier ones.
func (r *Registry) Register(name string, factory Factory) {
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
		r.factories = make(map[string]Factory)
	}
	r.factories[normalized] = factory
}

// Resolve returns the factory registered for name.
func (r *Registry) Resolve(name string) (Factory, bool) {
	if r == nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.factories[NormalizeName(name)]
	return factory, ok
}

// New constructs a provider for name.
func (r *Registry) New(name string) (ImageProvider, error) {
	var zero ImageProvider
	factory, ok := r.Resolve(name)
	if !ok {
		return zero, fmt.Errorf("%w: %s", ErrUnknownProvider, name)
	}
	return factory()
}

// NewFromConfig resolves the provider name from cfg and constructs it.
func (r *Registry) NewFromConfig(cfg Config) (ImageProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("image config is required")
	}
	return r.New(cfg.ImageProviderName())
}
