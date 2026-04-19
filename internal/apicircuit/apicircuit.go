// Package apicircuit wraps outbound Gemini API calls with sony/gobreaker
// circuit breakers so repeated failures do not pile up unbounded work.
package apicircuit

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

const (
	// breakerInterval clears rolling failure counts in the closed state so stale
	// errors do not keep the breaker sensitive forever.
	breakerInterval = 2 * time.Minute
	// breakerOpenTimeout is how long the breaker stays open before trying half-open.
	breakerOpenTimeout = 45 * time.Second
	// breakerMaxHalfOpenRequests limits trial traffic while recovering.
	breakerMaxHalfOpenRequests = 3
	// breakerTripAfterConsecutiveFailures opens the circuit after this many
	// consecutive failed requests in the closed state.
	breakerTripAfterConsecutiveFailures uint32 = 5
)

var (
	geminiTTSOnce      sync.Once
	geminiTTSBreaker   *gobreaker.CircuitBreaker
	geminiImageOnce    sync.Once
	geminiImageBreaker *gobreaker.CircuitBreaker
)

// isSuccessful counts only real API outcomes: nil is success; context.Canceled is
// treated as success so user abort does not trip the breaker. Timeouts and
// remote errors still count as failures.
func isSuccessful(err error) bool {
	if err == nil {
		return true
	}

	return errors.Is(err, context.Canceled)
}

func readyToTrip(counts gobreaker.Counts) bool {
	return counts.ConsecutiveFailures >= breakerTripAfterConsecutiveFailures
}

func newBreaker(name string) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:         name,
		MaxRequests:  breakerMaxHalfOpenRequests,
		Interval:     breakerInterval,
		Timeout:      breakerOpenTimeout,
		ReadyToTrip:  readyToTrip,
		IsSuccessful: isSuccessful,
	})
}

func geminiBreaker(name string, slot **gobreaker.CircuitBreaker, once *sync.Once) *gobreaker.CircuitBreaker {
	once.Do(func() {
		*slot = newBreaker(name)
	})

	return *slot
}

func runValue[T any](cb *gobreaker.CircuitBreaker, fn func() (T, error)) (T, error) {
	var zero T

	v, err := cb.Execute(func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return zero, err
	}
	if v == nil {
		return zero, nil
	}

	return v.(T), nil
}

// GeminiTTS runs one Gemini TTS GenerateContent call through its circuit breaker.
func GeminiTTS[T any](fn func() (T, error)) (T, error) {
	return runValue(geminiBreaker("gemini-tts", &geminiTTSBreaker, &geminiTTSOnce), fn)
}

// GeminiImage runs one Gemini image-generation call through its circuit breaker.
func GeminiImage[T any](fn func() (T, error)) (T, error) {
	return runValue(geminiBreaker("gemini-image", &geminiImageBreaker, &geminiImageOnce), fn)
}
