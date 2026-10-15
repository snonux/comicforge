package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type speechRequestTransport func(*http.Request) (*http.Response, error)

func (f speechRequestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestBuildTTSProviderUsesConfiguredStoryLanguage(t *testing.T) {
	// Not parallel: the normal provider's HTTP client uses http.DefaultTransport.
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	t.Setenv("COMICFORGE_LANGUAGE_STORY_LANGUAGE", "")

	for _, tc := range []struct {
		name     string
		language string
		want     string
	}{
		{"English name", "English", "en-US"},
		{"English code", "en-GB", "en-GB"},
		{"Bulgarian name", "Bulgarian", "bg-BG"},
		{"Bulgarian code", "bg", "bg-BG"},
		{"Chinese name", "Chinese", "cmn-CN"},
		{"Mandarin name", "Mandarin", "cmn-CN"},
		{"Chinese alias", "zh-CN", "cmn-CN"},
		{"Mandarin code", "cmn-CN", "cmn-CN"},
		{"Arabic name", "Arabic", "ar-XA"},
		{"Arabic code", "ar-XA", "ar-XA"},
		{"unsupported Arabic locale", "ar-EG", ""},
		{"unsupported Portuguese locale", "pt-PT", ""},
		{"unknown", "Unknown language", ""},
		{"empty", "", "bg-BG"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls int
			http.DefaultTransport = speechRequestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var body struct {
					GenerationConfig struct {
						SpeechConfig map[string]json.RawMessage `json:"speechConfig"`
					} `json:"generationConfig"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					return nil, fmt.Errorf("decode speech request: %w", err)
				}
				speechConfig := body.GenerationConfig.SpeechConfig
				if speechConfig == nil {
					t.Fatal("request has no speechConfig")
				}
				language, present := speechConfig["languageCode"]
				if tc.want == "" {
					if present {
						t.Fatalf("languageCode = %s, want omitted hint", language)
					}
				} else if string(language) != fmt.Sprintf("%q", tc.want) {
					t.Fatalf("languageCode = %s, want %q", language, tc.want)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"audio/mpeg","data":"SUQz"}}]}}]}`)),
				}, nil
			})

			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			yaml := fmt.Sprintf("api:\n  google_api_key: test-key\nlanguage:\n  story_language: %q\n", tc.language)
			if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadAppConfig(path)
			if err != nil {
				t.Fatalf("loadAppConfig() error = %v", err)
			}
			p, err := buildTTSProvider(cfg, "Aoede")
			if err != nil {
				t.Fatalf("buildTTSProvider() error = %v", err)
			}
			if err := p.GenerateAudio(context.Background(), "Narration", filepath.Join(dir, "out.mp3")); err != nil {
				t.Fatalf("GenerateAudio() error = %v", err)
			}
			if calls != 1 {
				t.Fatalf("speech requests = %d, want 1", calls)
			}
		})
	}
}
