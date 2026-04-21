package image

import (
	"context"
	"errors"
	"io"
	"strings"
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

func TestDefaultSearchOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultSearchOptions("ябълка")
	if opts.Query != "ябълка" {
		t.Fatalf("Query = %q, want %q", opts.Query, "ябълка")
	}
	if opts.Language != defaultSearchLanguage || !opts.SafeSearch || opts.PerPage != defaultSearchPerPage || opts.Page != defaultSearchPage {
		t.Fatalf("DefaultSearchOptions() = %+v", opts)
	}
}

func TestSearchError(t *testing.T) {
	t.Parallel()

	err := &SearchError{Provider: "test", Code: "404", Message: "Not found"}
	if got, want := err.Error(), "test: Not found"; got != want {
		t.Fatalf("SearchError.Error() = %q, want %q", got, want)
	}
}

func TestRegistryNewFromConfig(t *testing.T) {
	t.Parallel()

	registry := NewRegistry[fakeConfig]()
	registry.Register(Gemini, func(cfg fakeConfig) (ImageProvider, error) {
		return fakeProvider{
			name:  cfg.name,
			token: cfg.token,
		}, nil
	})

	gotProvider, err := registry.NewFromConfig(fakeConfig{name: Gemini, token: "secret"})
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if got, want := gotProvider.Name(), Gemini; got != want {
		t.Fatalf("provider.Name() = %q, want %q", got, want)
	}
	if got, want := gotProvider.(fakeProvider).token, "secret"; got != want {
		t.Fatalf("provider token = %q, want %q", got, want)
	}
}

func TestDefaultRegistryNewFromConfig(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry()
	gotProvider, err := registry.NewFromConfig(fakeConfig{
		name:      Gemini,
		apiKey:    "test-key",
		model:     DefaultGeminiImageModel,
		textModel: DefaultGeminiTextModel,
	})
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if got, want := gotProvider.Name(), Gemini; got != want {
		t.Fatalf("provider.Name() = %q, want %q", got, want)
	}
}

func TestDefaultRegistryUnsupportedProvider(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry()
	_, err := registry.NewFromConfig(fakeConfig{
		name:      OpenAI,
		apiKey:    "test-key",
		model:     "unused",
		textModel: "unused",
	})
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if !strings.Contains(err.Error(), "image provider \"openai\" is not implemented") {
		t.Fatalf("error = %v, want unsupported provider error", err)
	}
}

func TestRegistryUnknownProvider(t *testing.T) {
	t.Parallel()

	registry := NewRegistry[fakeConfig]()
	_, err := registry.New("missing", fakeConfig{})
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
	if !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("error = %v, want ErrUnknownProvider", err)
	}
}

type fakeConfig struct {
	name      string
	token     string
	apiKey    string
	model     string
	textModel string
}

func (f fakeConfig) ImageProviderName() string { return f.name }
func (f fakeConfig) GoogleAPIKey() string      { return f.apiKey }
func (f fakeConfig) ImageModel() string        { return f.model }
func (f fakeConfig) ImageTextModel() string    { return f.textModel }

type fakeProvider struct {
	name  string
	token string
}

func (f fakeProvider) Name() string { return f.name }

func (fakeProvider) Search(context.Context, *SearchOptions) ([]SearchResult, error) {
	return nil, nil
}

func (fakeProvider) Download(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (fakeProvider) GetAttribution(*SearchResult) string { return "" }
