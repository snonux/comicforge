package apicircuit

import (
	"context"
	"errors"
	"testing"
)

func TestGeminiTTS_Success(t *testing.T) {
	t.Parallel()

	v, err := GeminiTTS(func() (string, error) {
		return "ok", nil
	})
	if err != nil || v != "ok" {
		t.Fatalf("GeminiTTS() = %q, %v; want ok, nil", v, err)
	}
}

func TestGeminiImage_Success(t *testing.T) {
	t.Parallel()

	v, err := GeminiImage(func() (int, error) {
		return 42, nil
	})
	if err != nil || v != 42 {
		t.Fatalf("GeminiImage() = %d, %v; want 42, nil", v, err)
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
