// Package provider defines capability-specific AI provider interfaces and shared
// provider naming helpers. The concrete Gemini implementations will satisfy
// these interfaces once the comic pipeline is wired up.
package provider

import (
	"context"
	"errors"
	"strings"
)

const (
	// Gemini is the canonical provider name for Google's Gemini backend.
	Gemini = "gemini"

	// OpenAI is the canonical provider name for OpenAI backends.
	OpenAI = "openai"
)

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

// ErrUnknownProvider indicates that a provider name does not map to a known backend.
var ErrUnknownProvider = errors.New("unknown provider")
