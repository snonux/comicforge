package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"google.golang.org/genai"

	"github.com/snonux/comicforge/prompts"
)

type promptTestTransport func(*http.Request) (*http.Response, error)

func (f promptTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGeminiProvider_GenerateImage_PreservesRenderedPromptsAtAPI(t *testing.T) {
	for _, name := range []string{
		"cover_prompt.md", "story_page_prompt.md", "gallery_page_prompt.md",
		"back_cover_prompt.md", "manual_prompt.md",
	} {
		t.Run(name, func(t *testing.T) {
			prompt := renderLongImagePrompt(t, name)
			for _, withRefs := range []bool{false, true} {
				label := "without_references"
				if withRefs {
					label = "with_references"
				}
				t.Run(label, func(t *testing.T) {
					checkRenderedPromptRequest(t, prompt, withRefs)
				})
			}
		})
	}
}

func TestGeminiProvider_GenerateImage_ReportsPromptRejection(t *testing.T) {
	for _, refs := range [][][]byte{nil, {mustPNGBytes(t)}} {
		calls := 0
		prompt := strings.Repeat("Дълга история с точен диалог. ", 1000) + "FINAL STYLE LOCK"
		client := newPromptTestProvider(t, func(req *http.Request) (*http.Response, error) {
			calls++
			checkImageRequest(t, req, prompt, refs)
			return promptTestResponse(http.StatusBadRequest, `{"error":{"code":400,"message":"model input limit exceeded","status":"INVALID_ARGUMENT"}}`), nil
		})
		output := filepath.Join(t.TempDir(), "rejected.png")
		err := client.GenerateImageWithReferences(context.Background(), prompt, output, refs)
		var searchErr *SearchError
		if !errors.As(err, &searchErr) || searchErr.Code != "API_ERROR" || !strings.Contains(err.Error(), "model input limit exceeded") {
			t.Fatalf("GenerateImageWithReferences() error = %v, want model rejection", err)
		}
		if calls != 1 {
			t.Fatalf("API calls = %d, want one request with the full prompt", calls)
		}
		if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("rejected request created output: %v", err)
		}
	}
}

func TestGeminiProvider_GenerateImage_RejectsBlankPrompt(t *testing.T) {
	client := newPromptTestProvider(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("blank prompt reached API")
		return nil, nil
	})
	for _, prompt := range []string{"", " \n\t "} {
		output := filepath.Join(t.TempDir(), "blank.png")
		if err := client.GenerateImage(context.Background(), prompt, output); err == nil || !strings.Contains(err.Error(), "prompt is required") {
			t.Fatalf("GenerateImage(%q) error = %v, want required prompt error", prompt, err)
		}
		if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("blank prompt created output: %v", err)
		}
	}
}

func newPromptTestProvider(t *testing.T, transport promptTestTransport) *GeminiProvider {
	t.Helper()
	client := NewGeminiProvider(&GeminiConfig{
		APIKey: "test-key",
		newClient: func(ctx context.Context, cfg *genai.ClientConfig) (*genai.Client, error) {
			cfg.HTTPClient = &http.Client{Transport: transport}
			return genai.NewClient(ctx, cfg)
		},
	})
	if err := client.IsAvailable(); err != nil {
		t.Fatalf("initialize provider: %v", err)
	}
	return client
}

func promptTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func checkRenderedPromptRequest(t *testing.T, prompt string, withRefs bool) {
	t.Helper()
	payload := mustPNGBytes(t)
	var refs [][]byte
	if withRefs {
		refs = [][]byte{payload}
	}
	calls := 0
	client := newPromptTestProvider(t, func(req *http.Request) (*http.Response, error) {
		calls++
		checkImageRequest(t, req, prompt, refs)
		body := `{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"` + base64.StdEncoding.EncodeToString(payload) + `"}}]}}]}`
		return promptTestResponse(http.StatusOK, body), nil
	})
	var callbackPrompt string
	client.SetPromptCallback(func(prompt string) { callbackPrompt = prompt })
	output := filepath.Join(t.TempDir(), "page.png")
	if err := client.GenerateImageWithReferencesAndAspectRatio(context.Background(), prompt, output, refs, "2:3"); err != nil {
		t.Fatalf("GenerateImageWithReferencesAndAspectRatio() error = %v", err)
	}
	if calls != 1 || client.LastPrompt() != prompt || callbackPrompt != prompt {
		t.Fatalf("prompt changed or API calls = %d, want 1", calls)
	}
	got, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("generated image differs from API response: %v", err)
	}
}

func checkImageRequest(t *testing.T, req *http.Request, prompt string, refs [][]byte) {
	t.Helper()
	var request struct {
		Contents []*genai.Content `json:"contents"`
	}
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		t.Fatalf("decode image API request: %v", err)
	}
	if req.Method != http.MethodPost || !strings.Contains(req.URL.Path, ":generateContent") || len(request.Contents) != 1 {
		t.Fatalf("unexpected image API request: %s %s, %d contents", req.Method, req.URL.Path, len(request.Contents))
	}
	parts := request.Contents[0].Parts
	if len(parts) != len(refs)+1 {
		t.Fatalf("request parts = %d, want %d", len(parts), len(refs)+1)
	}
	for i, ref := range refs {
		if parts[i].InlineData == nil || !bytes.Equal(parts[i].InlineData.Data, ref) {
			t.Fatalf("reference image %d changed", i)
		}
	}
	text := parts[len(parts)-1].Text
	if len(refs) > 0 {
		_, text, _ = strings.Cut(text, "Now generate:\n\n")
	}
	if text != prompt {
		t.Fatalf("API prompt changed: got %d bytes, want all %d bytes including story, identity, dialog and style lock", len(text), len(prompt))
	}
}

func renderLongImagePrompt(t *testing.T, name string) string {
	t.Helper()
	read := func(name string) string {
		t.Helper()
		data, err := prompts.Read(name)
		if err != nil {
			t.Fatalf("read embedded prompt %q: %v", name, err)
		}
		return string(data)
	}
	story := strings.Repeat("Мира върви през гората и намира изгубената книга. ", 80) + "КРАЙ НА ИСТОРИЯТА"
	bible := strings.Repeat("Мира носи синя дреха и има зелени очи. ", 80) + "КРАЙ НА ОПИСАНИЕТО"
	data := map[string]any{
		"LanguageName": "български", "ScriptName": "кирилица",
		"Style": "cinematic realism", "PageFrame": "portrait 2:3", "DINA4PDF": false,
		"Bible": bible, "StoryText": story, "Prompt": story,
		"RenderingRequirement":    read("rendering_requirement.md"),
		"RenderingRequirementEnd": read("rendering_requirement_end.md"),
		"Subtitle":                "Приключение", "Words": "книга, гора", "PageNum": 1, "TotalPages": 3,
		"PanelLayout": "Мира казва: „Намерих книгата!“", "PanelsPerPage": 4, "RequiredDialoguePanels": 3,
		"GalleryNum": 1, "TotalGalleryPages": 2, "Pose": "Мира тича през гората.",
		"BlurbBox": "История за приятелството", "SeriesTitle": "Приключения", "Theme": "mystery",
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(read(name))
	if err != nil {
		t.Fatalf("parse prompt: %v", err)
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, data); err != nil {
		t.Fatalf("render prompt: %v", err)
	}
	prompt := strings.TrimSpace(out.String())
	if len(prompt) <= 4000 || !strings.Contains(prompt, "FINAL STYLE LOCK") {
		t.Fatal("fixture must exceed the former byte limit and include the final style lock")
	}
	return prompt
}
