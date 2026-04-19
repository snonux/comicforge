package apicircuit

import (
	"context"
	"errors"
	"testing"
)

func TestExecute_Success(t *testing.T) {
	t.Parallel()

	v, err := Execute(NewRegistry(), "gemini", CapabilityText, func() (string, error) {
		return "ok", nil
	})
	if err != nil || v != "ok" {
		t.Fatalf("Execute() = %q, %v; want ok, nil", v, err)
	}
}

func TestExecute_ContextCanceled(t *testing.T) {
	t.Parallel()

	_, err := Execute(NewRegistry(), "gemini", CapabilityText, func() (string, error) {
		return "", context.Canceled
	})
	if err == nil {
		t.Fatal("expected cancellation to be returned")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestIsSuccessful_ContextCanceled(t *testing.T) {
	t.Parallel()

	if !isSuccessful(context.Canceled) {
		t.Fatal("context.Canceled should not count as breaker failure")
	}
	if isSuccessful(errors.New("api error")) {
		t.Fatal("arbitrary errors must count as failure")
	}
}
