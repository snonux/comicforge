// Package httpctx provides HTTP client defaults and context helpers for outbound
// Gemini API calls and remote asset downloads.
package httpctx

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/genai"
)

const (
	// GenAIHTTPTimeout bounds each Google GenAI SDK HTTP request.
	GenAIHTTPTimeout = 30 * time.Minute

	// ImageDownloadTimeout limits fetches of remote image URLs.
	ImageDownloadTimeout = 60 * time.Second

	// OperationTimeoutDefault caps a full high-level operation when the caller did not set a deadline.
	OperationTimeoutDefault = 15 * time.Minute

	// ListModelsTimeout bounds model-listing CLI calls.
	ListModelsTimeout = 3 * time.Minute

	// StoryPageImageTimeout bounds a single comic page image pipeline when no parent deadline exists.
	StoryPageImageTimeout = 25 * time.Minute

	// VeoCLIPerVideoTimeout bounds one gallery-to-video run when the CLI passes Background.
	VeoCLIPerVideoTimeout = 25 * time.Minute

	// SingleWordProcessTimeout caps a single vocabulary processing operation when the caller uses Background.
	SingleWordProcessTimeout = 10 * time.Minute
)

// GenAIHTTPClient returns an http.Client for google.golang.org/genai.
func GenAIHTTPClient() *http.Client {
	return &http.Client{Timeout: GenAIHTTPTimeout}
}

// ImageDownloadHTTPClient returns a client for generic image URL downloads.
func ImageDownloadHTTPClient() *http.Client {
	return &http.Client{Timeout: ImageDownloadTimeout}
}

// NewGenAIClient wraps genai.NewClient, setting HTTPClient when the config does
// not supply one so outbound requests never rely on an unbounded default.
func NewGenAIClient(ctx context.Context, cfg *genai.ClientConfig) (*genai.Client, error) {
	if cfg == nil {
		cfg = &genai.ClientConfig{}
	}

	merged := *cfg
	if merged.HTTPClient == nil {
		merged.HTTPClient = GenAIHTTPClient()
	}

	client, err := genai.NewClient(ctx, &merged)
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	return client, nil
}

// WithTimeoutUnlessSet returns a child context with timeout d when ctx has no
// deadline. If ctx already has a deadline, it returns ctx and a no-op cancel.
func WithTimeoutUnlessSet(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, d)
}
