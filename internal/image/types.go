package image

import (
	"context"
	"io"
	"strings"

	"codeberg.org/snonux/comicforge/internal/provider"
)

const (
	// Gemini is the canonical provider name for Google's Gemini backend.
	Gemini = "gemini"

	// OpenAI is the canonical provider name for OpenAI backends.
	OpenAI = "openai"
)

// SearchResult represents a single generated image result.
type SearchResult struct {
	ID           string
	URL          string
	ThumbnailURL string
	Width        int
	Height       int
	Description  string
	Attribution  string
	Source       string
}

// SearchOptions configures image generation for Gemini (and compatible providers).
// Fields are limited to what the generator consumes; stock-photo search knobs
// (pagination, SafeSearch, orientation, etc.) are intentionally absent.
type SearchOptions struct {
	Query           string
	Translation     string
	CustomPrompt    string
	AspectRatio     string
	ReferenceImages [][]byte
}

// DefaultSearchOptions returns generation options for a language-learning query.
func DefaultSearchOptions(query string) *SearchOptions {
	return &SearchOptions{
		Query: query,
	}
}

// Provider generates, downloads, and describes images.
type Provider interface {
	Name() string
	Search(ctx context.Context, opts *SearchOptions) ([]SearchResult, error)
	Download(ctx context.Context, url string) (io.ReadCloser, error)
	GetAttribution(result *SearchResult) string
}

// SearchError represents a provider failure.
type SearchError struct {
	Provider string
	Code     string
	Message  string
}

func (e *SearchError) Error() string {
	switch {
	case e == nil:
		return ""
	case e.Provider == "":
		return e.Message
	case e.Message == "":
		return e.Provider
	default:
		return e.Provider + ": " + e.Message
	}
}

// ErrUnknownProvider indicates that no provider is registered for a name.
var ErrUnknownProvider = provider.ErrUnknownProvider

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
