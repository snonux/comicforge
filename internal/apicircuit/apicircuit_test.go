package apicircuit

import (
	"context"
	"errors"
	"testing"

	"github.com/sony/gobreaker"
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

func TestExecute_OpenStateAfterConsecutiveFailures(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	apiErr := errors.New("remote failure")
	var calls int
	fn := func() (string, error) {
		calls++
		return "", apiErr
	}

	for i := 0; i < int(breakerTripAfterConsecutiveFailures); i++ {
		_, err := Execute(reg, "gemini", CapabilityTTS, fn)
		if !errors.Is(err, apiErr) {
			t.Fatalf("failure %d: error = %v, want %v", i+1, err, apiErr)
		}
	}

	callsBeforeOpen := calls
	_, err := Execute(reg, "gemini", CapabilityTTS, fn)
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("open circuit: error = %v, want ErrOpenState", err)
	}
	if calls != callsBeforeOpen {
		t.Fatalf("open circuit still invoked fn: calls = %d, want %d", calls, callsBeforeOpen)
	}
}

func TestExecute_CapabilitiesAreIsolated(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	apiErr := errors.New("text down")
	for i := 0; i < int(breakerTripAfterConsecutiveFailures); i++ {
		_, err := Execute(reg, "gemini", CapabilityText, func() (string, error) {
			return "", apiErr
		})
		if !errors.Is(err, apiErr) {
			t.Fatalf("text failure %d: error = %v", i+1, err)
		}
	}
	_, err := Execute(reg, "gemini", CapabilityText, func() (string, error) {
		return "should-not-run", nil
	})
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("text open: error = %v, want ErrOpenState", err)
	}

	v, err := Execute(reg, "gemini", CapabilityTTS, func() (string, error) {
		return "tts-ok", nil
	})
	if err != nil || v != "tts-ok" {
		t.Fatalf("tts Execute() = %q, %v; want tts-ok, nil", v, err)
	}
}
