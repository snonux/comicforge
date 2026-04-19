// Package apicircuit wraps outbound API calls with circuit breakers so repeated
// failures against any provider do not pile up unbounded work.
package apicircuit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

const (
	// CapabilityText identifies text-generation traffic.
	CapabilityText Capability = "text"
	// CapabilityImage identifies image-generation traffic.
	CapabilityImage Capability = "image"
	// CapabilityTTS identifies text-to-speech traffic.
	CapabilityTTS Capability = "tts"
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

// Capability identifies the class of API call protected by a circuit breaker.
type Capability string

// Registry stores lazily-created circuit breakers keyed by provider and capability.
type Registry struct {
	mu       sync.Mutex
	breakers map[string]*gobreaker.CircuitBreaker
}

// NewRegistry creates an empty breaker registry.
func NewRegistry() *Registry {
	return &Registry{breakers: make(map[string]*gobreaker.CircuitBreaker)}
}

// DefaultRegistry is used by package-level helpers.
var DefaultRegistry = NewRegistry()

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

func breakerKey(providerName string, capability Capability) string {
	return NormalizeName(providerName) + ":" + NormalizeName(string(capability))
}

func (r *Registry) breaker(providerName string, capability Capability) *gobreaker.CircuitBreaker {
	key := breakerKey(providerName, capability)

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.breakers == nil {
		r.breakers = make(map[string]*gobreaker.CircuitBreaker)
	}
	if breaker := r.breakers[key]; breaker != nil {
		return breaker
	}

	breaker := newBreaker(key)
	r.breakers[key] = breaker
	return breaker
}

// Execute runs fn through the breaker for providerName and capability.
func Execute[T any](r *Registry, providerName string, capability Capability, fn func() (T, error)) (T, error) {
	var zero T
	if r == nil {
		r = DefaultRegistry
	}

	v, err := r.breaker(providerName, capability).Execute(func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return zero, err
	}
	if v == nil {
		return zero, nil
	}

	result, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("unexpected breaker result type for %s", breakerKey(providerName, capability))
	}

	return result, nil
}

// NormalizeName returns a canonical lower-case name for breaker keys.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
