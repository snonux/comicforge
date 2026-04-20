package httpctx

import (
	"context"
	"net/http"
	"testing"
	"time"

	"google.golang.org/genai"
)

func TestWithTimeoutUnlessSet_AlreadyHasDeadline(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	ctx, childCancel := WithTimeoutUnlessSet(parent, time.Nanosecond)
	defer childCancel()

	if ctx != parent {
		t.Fatal("expected same context when parent already has deadline")
	}
}

func TestWithTimeoutUnlessSet_NoDeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := WithTimeoutUnlessSet(context.Background(), 50*time.Millisecond)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}
	if time.Until(deadline) > time.Second {
		t.Fatalf("deadline too far: %v", deadline)
	}
}

func TestWithTimeoutUnlessSet_NilUsesBackground(t *testing.T) {
	t.Parallel()

	ctx, cancel := WithTimeoutUnlessSet(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := ctx.Err(); err != nil {
		t.Fatalf("context should not be done: %v", err)
	}
}

func TestNewGenAIClientAppliesDefaultHTTPClient(t *testing.T) {
	t.Parallel()

	customClient := &http.Client{Timeout: time.Second}
	cfg := &genai.ClientConfig{
		APIKey: "test-key",
	}

	client, err := NewGenAIClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewGenAIClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewGenAIClient() client = nil")
	}
	if cfg.HTTPClient != nil {
		t.Fatalf("NewGenAIClient() mutated input config HTTPClient = %#v, want nil", cfg.HTTPClient)
	}

	cfg2 := &genai.ClientConfig{
		APIKey:     "test-key",
		HTTPClient: customClient,
	}
	if _, err := NewGenAIClient(context.Background(), cfg2); err != nil {
		t.Fatalf("NewGenAIClient() error = %v", err)
	}
	if cfg2.HTTPClient != customClient {
		t.Fatal("NewGenAIClient() should preserve caller HTTPClient")
	}
}
