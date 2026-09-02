// Package provider defines capability-specific AI provider interfaces and
// provider-neutral registries for runtime selection.
package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

const (
	// Gemini is the canonical provider name for Google's Gemini backend.
	Gemini = "gemini"

	// OpenAI is the canonical provider name for OpenAI backends.
	OpenAI = "openai"
)

// ErrUnknownProvider indicates that a provider name does not map to a known backend.
var ErrUnknownProvider = errors.New("unknown provider")

// TextProvider generates text from prompts.
type TextProvider interface {
	Name() string
	IsAvailable() error
	GenerateText(ctx context.Context, prompt string) (string, error)
}

// ImageProvider generates images from prompts.
type ImageProvider interface {
	Name() string
	IsAvailable() error
	GenerateImage(ctx context.Context, prompt string, outputFile string) error
}

// AspectRatioImageProvider generates images with an explicit aspect ratio.
type AspectRatioImageProvider interface {
	ImageProvider
	GenerateImageWithAspectRatio(ctx context.Context, prompt string, outputFile string, aspectRatio string) error
}

// TTSProvider generates audio from text.
type TTSProvider interface {
	Name() string
	IsAvailable() error
	GenerateAudio(ctx context.Context, text string, outputFile string) error
}

// TextConfig exposes the configured text provider name.
type TextConfig interface {
	TextProviderName() string
}

// ImageConfig exposes the configured image provider name.
type ImageConfig interface {
	ImageProviderName() string
}

// TTSConfig exposes the configured TTS provider name.
type TTSConfig interface {
	TTSProviderName() string
}

// Factory builds a provider implementation from configuration.
type Factory[T any, C any] func(C) (T, error)

// Registry resolves provider names to factories.
type Registry[T any, C any] struct {
	mu        sync.RWMutex
	factories map[string]Factory[T, C]
}

// NewRegistry creates an empty provider registry.
func NewRegistry[T any, C any]() *Registry[T, C] {
	return &Registry[T, C]{factories: make(map[string]Factory[T, C])}
}

// NewTextRegistry creates an empty text-provider registry.
func NewTextRegistry() *Registry[TextProvider, TextConfig] {
	return NewRegistry[TextProvider, TextConfig]()
}

// NewImageRegistry creates an empty image-provider registry.
func NewImageRegistry() *Registry[ImageProvider, ImageConfig] {
	return NewRegistry[ImageProvider, ImageConfig]()
}

// NewTTSRegistry creates an empty text-to-speech provider registry.
func NewTTSRegistry() *Registry[TTSProvider, TTSConfig] {
	return NewRegistry[TTSProvider, TTSConfig]()
}

// Register associates name with a factory. Later registrations replace earlier ones.
func (r *Registry[T, C]) Register(name string, factory Factory[T, C]) {
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
		r.factories = make(map[string]Factory[T, C])
	}
	r.factories[normalized] = factory
}

// Resolve returns the factory registered for name.
func (r *Registry[T, C]) Resolve(name string) (Factory[T, C], bool) {
	if r == nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.factories[NormalizeName(name)]
	return factory, ok
}

// Has reports whether a factory is registered for name.
func (r *Registry[T, C]) Has(name string) bool {
	_, ok := r.Resolve(name)
	return ok
}

// New constructs a provider for name using cfg.
func (r *Registry[T, C]) New(name string, cfg C) (T, error) {
	var zero T
	factory, ok := r.Resolve(name)
	if !ok {
		return zero, fmt.Errorf("%w: %s", ErrUnknownProvider, name)
	}
	return factory(cfg)
}

// NewFromConfig resolves the provider name from cfg and constructs it.
func (r *Registry[T, C]) NewFromConfig(cfg C, providerName func(C) string) (T, error) {
	var zero T
	if providerName == nil {
		return zero, errors.New("provider name resolver is required")
	}
	return r.New(providerName(cfg), cfg)
}

// NormalizeName returns a canonical lower-case provider name.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// IsKnownName reports whether the name matches a supported provider family.
func IsKnownName(name string) bool {
	switch NormalizeName(name) {
	case Gemini, OpenAI:
		return true
	default:
		return false
	}
}
