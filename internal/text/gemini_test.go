package text

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sony/gobreaker"
	"google.golang.org/genai"

	"codeberg.org/snonux/comicforge/internal/apicircuit"
)

func TestGenerateTextSuccess(t *testing.T) {
	origGen := geminiGenerateContent
	t.Cleanup(func() { geminiGenerateContent = origGen })

	geminiGenerateContent = func(_ context.Context, _ *genai.Client, model string, contents []*genai.Content, _ *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		if model != "test-model" {
			t.Fatalf("model = %q, want test-model", model)
		}
		if len(contents) != 1 || contents[0] == nil || len(contents[0].Parts) != 1 {
			t.Fatalf("unexpected contents: %#v", contents)
		}
		if got := contents[0].Parts[0].Text; got != "hello" {
			t.Fatalf("prompt = %q, want hello", got)
		}
		return &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: "  story text  "}},
				},
			}},
		}, nil
	}

	p := &GeminiProvider{model: "test-model"}
	got, err := p.GenerateText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("GenerateText() error = %v", err)
	}
	if got != "story text" {
		t.Fatalf("GenerateText() = %q, want story text", got)
	}
}

func TestGenerateTextRejectsEmptyPrompt(t *testing.T) {
	t.Parallel()

	p := &GeminiProvider{model: DefaultModel}
	_, err := p.GenerateText(context.Background(), "  ")
	if err == nil {
		t.Fatal("GenerateText() error = nil, want prompt required")
	}
	if !strings.Contains(err.Error(), "prompt is required") {
		t.Fatalf("GenerateText() error = %v, want prompt required", err)
	}
}

func TestGenerateTextEmptyResponse(t *testing.T) {
	origGen := geminiGenerateContent
	t.Cleanup(func() { geminiGenerateContent = origGen })

	geminiGenerateContent = func(context.Context, *genai.Client, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		return &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				Content: &genai.Content{Parts: []*genai.Part{{Text: "   "}}},
			}},
		}, nil
	}

	p := &GeminiProvider{model: DefaultModel}
	_, err := p.generateText(context.Background(), "hello")
	if err == nil {
		t.Fatal("generateText() error = nil, want no text content")
	}
	if !strings.Contains(err.Error(), "no text content returned") {
		t.Fatalf("generateText() error = %v, want no text content", err)
	}
}

func TestGenerateTextCircuitOpenSkipsAPI(t *testing.T) {
	circuitRegistry = apicircuit.NewRegistry()
	t.Cleanup(func() { circuitRegistry = nil })

	origGen := geminiGenerateContent
	t.Cleanup(func() { geminiGenerateContent = origGen })

	var calls int
	apiErr := errors.New("api down")
	geminiGenerateContent = func(context.Context, *genai.Client, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		calls++
		return nil, apiErr
	}

	p := &GeminiProvider{model: DefaultModel}
	for i := 0; i < 5; i++ {
		_, err := p.generateText(context.Background(), "hello")
		if err == nil {
			t.Fatalf("failure %d: error = nil", i+1)
		}
		if !errors.Is(err, apiErr) {
			t.Fatalf("failure %d: error = %v, want wrapped %v", i+1, err, apiErr)
		}
	}

	callsBeforeOpen := calls
	_, err := p.generateText(context.Background(), "hello")
	if err == nil {
		t.Fatal("open circuit: error = nil")
	}
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("open circuit: error = %v, want ErrOpenState", err)
	}
	if calls != callsBeforeOpen {
		t.Fatalf("open circuit still called API: calls = %d, want %d", calls, callsBeforeOpen)
	}
}
