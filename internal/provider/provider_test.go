package provider

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	t.Parallel()

	if got, want := NormalizeName(" Gemini "), Gemini; got != want {
		t.Fatalf("NormalizeName() = %q, want %q", got, want)
	}
}

func TestIsKnownName(t *testing.T) {
	t.Parallel()

	if !IsKnownName("gemini") {
		t.Fatal("expected gemini to be recognized")
	}
	if IsKnownName("bogus") {
		t.Fatal("expected bogus provider to be rejected")
	}
}

func TestRegistryNewFromConfig(t *testing.T) {
	t.Parallel()

	registry := NewTextRegistry()
	registry.Register(Gemini, func(cfg TextConfig) (TextProvider, error) {
		return fakeTextProvider{name: cfg.TextProviderName()}, nil
	})

	provider, err := registry.NewFromConfig(fakeTextConfig{name: "gemini"}, func(cfg TextConfig) string {
		return cfg.TextProviderName()
	})
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if got, want := provider.Name(), "gemini"; got != want {
		t.Fatalf("provider.Name() = %q, want %q", got, want)
	}
}

func TestRegistryUnknownProvider(t *testing.T) {
	t.Parallel()

	registry := NewTTSRegistry()
	_, err := registry.New("missing", fakeTTSConfig{})
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
	if !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("error = %v, want ErrUnknownProvider", err)
	}
}

type fakeTextConfig struct {
	name string
}

func (f fakeTextConfig) TextProviderName() string { return f.name }

type fakeTTSConfig struct{}

func (fakeTTSConfig) TTSProviderName() string { return "missing" }

type fakeTextProvider struct {
	name string
}

func (f fakeTextProvider) Name() string       { return f.name }
func (f fakeTextProvider) IsAvailable() error { return nil }
func (f fakeTextProvider) GenerateText(_ context.Context, _ string) (string, error) {
	return "", nil
}
