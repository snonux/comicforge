package image

import (
	"context"
	"errors"
	"io"
	"reflect"
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
	if opts == nil {
		t.Fatal("DefaultSearchOptions() returned nil")
	}
	if opts.Query != "ябълка" {
		t.Fatalf("Query = %q, want %q", opts.Query, "ябълка")
	}
	if opts.Translation != "" || opts.CustomPrompt != "" || opts.AspectRatio != "" || len(opts.ReferenceImages) != 0 {
		t.Fatalf("DefaultSearchOptions() set unexpected fields: %+v", opts)
	}
}

func TestDefaultSearchOptions_EmptyQuery(t *testing.T) {
	t.Parallel()

	opts := DefaultSearchOptions("")
	if opts == nil {
		t.Fatal("DefaultSearchOptions() returned nil")
	}
	if opts.Query != "" {
		t.Fatalf("Query = %q, want empty", opts.Query)
	}
}

func TestSearchOptions_OnlyGenerationFields(t *testing.T) {
	t.Parallel()

	// SearchOptions must stay trimmed to Gemini generation inputs.
	// Reintroducing stock-photo knobs is a regression against task t0.
	allowed := map[string]struct{}{
		"Query":           {},
		"Translation":     {},
		"CustomPrompt":    {},
		"AspectRatio":     {},
		"ReferenceImages": {},
	}
	typ := reflect.TypeOf(SearchOptions{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if _, ok := allowed[name]; !ok {
			t.Errorf("unexpected SearchOptions field %q (stock-photo search fields must stay removed)", name)
		}
	}
	for name := range allowed {
		if _, ok := typ.FieldByName(name); !ok {
			t.Errorf("missing expected SearchOptions field %q", name)
		}
	}

	forbidden := []string{
		"SafeSearch", "PerPage", "Page", "ImageType", "Orientation", "Language",
	}
	for _, name := range forbidden {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("SearchOptions still has stock-photo field %q", name)
		}
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

func TestRegistryNewFromConfigNilRegistry(t *testing.T) {
	t.Parallel()

	var registry *Registry[fakeConfig]
	_, err := registry.NewFromConfig(fakeConfig{name: Gemini})
	if err == nil {
		t.Fatal("expected nil registry error")
	}
	if !strings.Contains(err.Error(), "image registry is required") {
		t.Fatalf("error = %v, want nil registry error", err)
	}
}

func TestRegistrySupportsCustomConfigWithoutGeminiFields(t *testing.T) {
	t.Parallel()

	registry := NewRegistry[customConfig]()
	registry.Register(Gemini, func(cfg customConfig) (ImageProvider, error) {
		return fakeProvider{name: cfg.name}, nil
	})

	gotProvider, err := registry.NewFromConfig(customConfig{name: Gemini})
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if got, want := gotProvider.Name(), Gemini; got != want {
		t.Fatalf("provider.Name() = %q, want %q", got, want)
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
func (f fakeConfig) ComicAspectRatio() string  { return "" }

type customConfig struct {
	name string
}

func (f customConfig) ImageProviderName() string { return f.name }

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
