package image

import (
	"context"
	"errors"
	"io"
	"strings"
)

const (
	// Gemini is the canonical provider name for Google's Gemini backend.
	Gemini = "gemini"

	// OpenAI is the canonical provider name for OpenAI backends.
	OpenAI = "openai"
)

const (
	defaultSearchLanguage    = "bg"
	defaultSearchPerPage     = 10
	defaultSearchPage        = 1
	defaultSearchImageType   = "photo"
	defaultSearchOrientation = "all"
)

// SearchResult represents a single image result.
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

// SearchOptions configures image search and generation.
type SearchOptions struct {
	Query           string
	Translation     string
	Language        string
	SafeSearch      bool
	PerPage         int
	Page            int
	ImageType       string
	Orientation     string
	CustomPrompt    string
	AspectRatio     string
	ReferenceImages [][]byte
}

// DefaultSearchOptions returns sensible defaults for language-learning queries.
func DefaultSearchOptions(query string) *SearchOptions {
	return &SearchOptions{
		Query:       query,
		Language:    defaultSearchLanguage,
		SafeSearch:  true,
		PerPage:     defaultSearchPerPage,
		Page:        defaultSearchPage,
		ImageType:   defaultSearchImageType,
		Orientation: defaultSearchOrientation,
	}
}

// ImageProvider generates, downloads, and describes images.
type ImageProvider interface {
	Name() string
	Search(ctx context.Context, opts *SearchOptions) ([]SearchResult, error)
	Download(ctx context.Context, url string) (io.ReadCloser, error)
	GetAttribution(result *SearchResult) string
}

// ImageClient is kept as a compatibility alias for ImageProvider.
type ImageClient = ImageProvider

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
var ErrUnknownProvider = errors.New("unknown image provider")

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
