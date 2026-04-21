package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/genai"
)

func TestNewGeminiProvider(t *testing.T) {
	t.Parallel()

	client := mustNewGeminiProvider(t, &GeminiConfig{APIKey: "test-key"})
	if client.config == nil {
		t.Fatal("expected normalized config")
	}
	if client.config.Model != DefaultGeminiImageModel {
		t.Fatalf("expected default model %q, got %q", DefaultGeminiImageModel, client.config.Model)
	}
	if client.config.TextModel != DefaultGeminiTextModel {
		t.Fatalf("expected default text model %q, got %q", DefaultGeminiTextModel, client.config.TextModel)
	}
	if client.Name() != Gemini {
		t.Fatalf("Name() = %q, want %q", client.Name(), Gemini)
	}
}

func mustNewGeminiProvider(t *testing.T, cfg *GeminiConfig) *GeminiProvider {
	t.Helper()

	client := NewGeminiProvider(cfg)
	if client == nil {
		t.Fatal("expected client")
	}
	return client
}

func TestGeminiProvider_NoAPIKey(t *testing.T) {
	client := NewGeminiProvider(&GeminiConfig{})

	_, err := client.Search(context.Background(), DefaultSearchOptions("ябълка"))
	if err == nil {
		t.Fatal("expected error for missing API key")
	}

	searchErr, ok := err.(*SearchError)
	if !ok {
		t.Fatalf("expected SearchError, got %T", err)
	}
	if searchErr.Code != "NO_API_KEY" {
		t.Fatalf("expected NO_API_KEY error, got %s", searchErr.Code)
	}
}

func TestGeminiProvider_Search_CustomPromptSkipsTextGeneration(t *testing.T) {
	originalText := geminiGenerateText
	originalImage := geminiGenerateImage
	t.Cleanup(func() {
		geminiGenerateText = originalText
		geminiGenerateImage = originalImage
	})

	geminiGenerateText = func(context.Context, *GeminiProvider, string, string, string, float32, int32) (string, error) {
		t.Fatal("unexpected text generation for custom prompt")
		return "", nil
	}

	var gotPrompt string
	geminiGenerateImage = func(_ context.Context, _ *GeminiProvider, prompt, _ string) ([]byte, string, error) {
		gotPrompt = prompt
		return mustJPEGBytes(t), "image/jpeg", nil
	}

	client := NewGeminiProvider(&GeminiConfig{APIKey: "test-key"})
	callbackCalled := false
	client.SetPromptCallback(func(prompt string) {
		callbackCalled = true
		if prompt != "custom flashcard prompt" {
			t.Fatalf("callback prompt = %q, want %q", prompt, "custom flashcard prompt")
		}
	})

	results, err := client.Search(context.Background(), &SearchOptions{
		Query:        "ябълка",
		Translation:  "banana",
		CustomPrompt: " custom flashcard prompt ",
	})
	if err != nil {
		t.Fatalf("Search() unexpected error: %v", err)
	}
	if gotPrompt != "custom flashcard prompt" {
		t.Fatalf("image prompt = %q, want %q", gotPrompt, "custom flashcard prompt")
	}
	if !callbackCalled {
		t.Fatal("expected prompt callback to be called")
	}
	if client.LastPrompt() != "custom flashcard prompt" {
		t.Fatalf("LastPrompt() = %q, want %q", client.LastPrompt(), "custom flashcard prompt")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Source != Gemini {
		t.Fatalf("Source = %q, want %q", results[0].Source, Gemini)
	}
	if !strings.HasPrefix(results[0].URL, geminiDataPrefix) {
		t.Fatalf("expected PNG data URI, got %q", results[0].URL)
	}
}

func TestGeminiProvider_Search_CustomPromptIsTruncated(t *testing.T) {
	originalText := geminiGenerateText
	originalImage := geminiGenerateImage
	t.Cleanup(func() {
		geminiGenerateText = originalText
		geminiGenerateImage = originalImage
	})

	geminiGenerateText = func(context.Context, *GeminiProvider, string, string, string, float32, int32) (string, error) {
		t.Fatal("unexpected text generation for truncated prompt")
		return "", nil
	}

	var gotPrompt string
	geminiGenerateImage = func(_ context.Context, _ *GeminiProvider, prompt, _ string) ([]byte, string, error) {
		gotPrompt = prompt
		return mustJPEGBytes(t), "image/jpeg", nil
	}

	client := NewGeminiProvider(&GeminiConfig{APIKey: "test-key"})
	longPrompt := strings.Repeat("a", maxCustomPrompt+128)
	results, err := client.Search(context.Background(), &SearchOptions{
		Query:        "ябълка",
		CustomPrompt: longPrompt,
	})
	if err != nil {
		t.Fatalf("Search() unexpected error: %v", err)
	}
	if len(gotPrompt) != maxCustomPrompt {
		t.Fatalf("prompt length = %d, want %d", len(gotPrompt), maxCustomPrompt)
	}
	if !strings.HasSuffix(gotPrompt, "...") {
		t.Fatalf("prompt = %q, want ellipsis suffix", gotPrompt)
	}
	if client.LastPrompt() != gotPrompt {
		t.Fatalf("LastPrompt() = %q, want %q", client.LastPrompt(), gotPrompt)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestGeminiProvider_GenerateImage_UsesCustomPrompt(t *testing.T) {
	originalText := geminiGenerateText
	originalImage := geminiGenerateImage
	t.Cleanup(func() {
		geminiGenerateText = originalText
		geminiGenerateImage = originalImage
	})

	geminiGenerateText = func(context.Context, *GeminiProvider, string, string, string, float32, int32) (string, error) {
		t.Fatal("unexpected text generation for comic prompt")
		return "", nil
	}

	var gotPrompt string
	geminiGenerateImage = func(_ context.Context, _ *GeminiProvider, prompt, _ string) ([]byte, string, error) {
		gotPrompt = prompt
		return mustJPEGBytes(t), "image/jpeg", nil
	}

	client := NewGeminiProvider(&GeminiConfig{APIKey: "test-key"})
	if err := client.GenerateImage(context.Background(), "Art style: comic. FULL-BLEED SINGLE ILLUSTRATION.", t.TempDir()+"/out.png"); err != nil {
		t.Fatalf("GenerateImage() unexpected error: %v", err)
	}
	if gotPrompt != "Art style: comic. FULL-BLEED SINGLE ILLUSTRATION." {
		t.Fatalf("GenerateImage() prompt = %q, want custom prompt", gotPrompt)
	}
}

func TestGeminiProvider_Search_GeneratedPromptFlow(t *testing.T) {
	originalText := geminiGenerateText
	originalImage := geminiGenerateImage
	originalStyles := append([]string(nil), ArtisticStyles...)
	t.Cleanup(func() {
		geminiGenerateText = originalText
		geminiGenerateImage = originalImage
		ArtisticStyles = originalStyles
	})

	ArtisticStyles = []string{"Photorealism"}

	var sceneCalls int
	var gotPrompt string
	var callbackPrompt string
	geminiGenerateText = func(_ context.Context, _ *GeminiProvider, _, systemPrompt, userPrompt string, temperature float32, maxOutputTokens int32) (string, error) {
		if strings.Contains(systemPrompt, "educational flashcards for language learning") {
			sceneCalls++
			if temperature != 0.7 || maxOutputTokens != 100 {
				t.Fatalf("scene params = %v/%d, want 0.7/100", temperature, maxOutputTokens)
			}
			if !strings.Contains(userPrompt, "apple") {
				t.Fatalf("scene prompt = %q, want English translation", userPrompt)
			}
			return "A bright apple sits centered on a wooden table.", nil
		}
		t.Fatalf("unexpected system prompt: %q", systemPrompt)
		return "", nil
	}

	geminiGenerateImage = func(_ context.Context, _ *GeminiProvider, prompt, _ string) ([]byte, string, error) {
		gotPrompt = prompt
		return mustJPEGBytes(t), "image/jpeg", nil
	}

	client := NewGeminiProvider(&GeminiConfig{APIKey: "test-key"})
	callbackCalled := false
	client.SetPromptCallback(func(prompt string) {
		callbackCalled = true
		callbackPrompt = prompt
	})

	results, err := client.Search(context.Background(), &SearchOptions{
		Query:       "ябълка",
		Translation: "apple",
	})
	if err != nil {
		t.Fatalf("Search() unexpected error: %v", err)
	}
	if sceneCalls != 1 {
		t.Fatalf("sceneCalls = %d, want 1", sceneCalls)
	}
	if !callbackCalled {
		t.Fatal("expected prompt callback to be called")
	}
	if callbackPrompt != gotPrompt {
		t.Fatalf("prompt callback = %q, want %q", callbackPrompt, gotPrompt)
	}
	if client.LastPrompt() != gotPrompt {
		t.Fatalf("LastPrompt() = %q, want %q", client.LastPrompt(), gotPrompt)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Source != Gemini {
		t.Fatalf("Source = %q, want %q", result.Source, Gemini)
	}
	if result.Width != 1 || result.Height != 1 {
		t.Fatalf("Size = %dx%d, want %dx%d", result.Width, result.Height, 1, 1)
	}
	if !strings.Contains(result.Description, "apple") {
		t.Fatalf("Description = %q, want translated word", result.Description)
	}
	if !strings.Contains(gotPrompt, "Generate a Photorealism educational flashcard image illustrating \"apple\".") {
		t.Fatalf("Prompt = %q, want translated subject in generated prompt", gotPrompt)
	}
	if !strings.Contains(gotPrompt, "Scene: A bright apple sits centered on a wooden table.") {
		t.Fatalf("Prompt = %q, want generated scene in prompt", gotPrompt)
	}
	if result.Attribution == "" {
		t.Fatal("expected result attribution to be populated")
	}
	if got := client.GetAttribution(&result); got != result.Attribution {
		t.Fatalf("GetAttribution() = %q, want result attribution %q", got, result.Attribution)
	}

	reader, err := client.Download(context.Background(), result.URL)
	if err != nil {
		t.Fatalf("Download() unexpected error: %v", err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
	})

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() unexpected error: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("Search output was not normalized to PNG")
	}
}

func TestGeminiProvider_Search_InvalidOptions(t *testing.T) {
	t.Parallel()

	client := NewGeminiProvider(&GeminiConfig{APIKey: "test-key"})
	_, err := client.Search(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil options")
	}

	searchErr, ok := err.(*SearchError)
	if !ok {
		t.Fatalf("expected SearchError, got %T", err)
	}
	if searchErr.Code != "INVALID_OPTIONS" {
		t.Fatalf("expected INVALID_OPTIONS, got %s", searchErr.Code)
	}
}

func TestGeminiProvider_DownloadDataURI(t *testing.T) {
	client := &GeminiProvider{}
	payload := mustPNGBytes(t)
	url := geminiDataPrefix + base64.StdEncoding.EncodeToString(payload)

	reader, err := client.Download(context.Background(), url)
	if err != nil {
		t.Fatalf("Download() unexpected error: %v", err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
	})

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() unexpected error: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("Download() = %v, want %v", data, payload)
	}
}

func TestGeminiProvider_DownloadHTTPFallback(t *testing.T) {
	client := &GeminiProvider{}
	payload := []byte("fallback image bytes")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("request method = %s, want GET", r.Method)
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	reader, err := client.Download(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Download() unexpected error: %v", err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
	})

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() unexpected error: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("Download() = %v, want %v", data, payload)
	}
}

func TestExtractGeneratedImage(t *testing.T) {
	t.Parallel()

	want := mustPNGBytes(t)
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{InlineData: &genai.Blob{Data: want, MIMEType: "image/png"}},
					},
				},
			},
		},
	}

	got, mimeType, err := extractGeneratedImage(resp)
	if err != nil {
		t.Fatalf("extractGeneratedImage() error = %v", err)
	}
	if mimeType != "image/png" {
		t.Fatalf("mimeType = %q, want %q", mimeType, "image/png")
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("extractGeneratedImage() = %v, want %v", got, want)
	}
}

func TestEncodeAndNormalizePNG(t *testing.T) {
	t.Parallel()

	jpegBytes := mustJPEGBytes(t)
	pngBytes, err := normalizePNG(jpegBytes, "image/jpeg")
	if err != nil {
		t.Fatalf("normalizePNG() unexpected error: %v", err)
	}
	if !bytes.HasPrefix(pngBytes, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("normalizePNG() did not return PNG data")
	}

	dataURL, err := encodeDataURL(jpegBytes, "image/jpeg")
	if err != nil {
		t.Fatalf("encodeDataURL() unexpected error: %v", err)
	}
	if !strings.HasPrefix(dataURL, geminiDataPrefix) {
		t.Fatalf("encodeDataURL() = %q, want PNG data URI", dataURL)
	}
}

func TestDecodeDataURIErrors(t *testing.T) {
	t.Parallel()

	if _, err := decodeDataURL("not-a-data-uri"); err == nil {
		t.Fatal("expected error for invalid data URI")
	}
}
