package image

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockDownloaderProvider struct {
	results       []SearchResult
	searchErr     error
	payload       string
	attribution   string
	searchQueries []string
	downloadURLs  []string
}

func (m *mockDownloaderProvider) Name() string { return "mock" }

func (m *mockDownloaderProvider) Search(_ context.Context, opts *SearchOptions) ([]SearchResult, error) {
	if opts != nil {
		m.searchQueries = append(m.searchQueries, opts.Query)
	}
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return append([]SearchResult(nil), m.results...), nil
}

func (m *mockDownloaderProvider) Download(_ context.Context, url string) (io.ReadCloser, error) {
	m.downloadURLs = append(m.downloadURLs, url)
	return io.NopCloser(strings.NewReader(m.payload)), nil
}

func (m *mockDownloaderProvider) GetAttribution(*SearchResult) string {
	return m.attribution
}

func TestDownloaderGenerateFileName_DataURIUsesPNG(t *testing.T) {
	t.Parallel()

	d := NewDownloader(&mockDownloaderProvider{}, &DownloadOptions{
		FileNamePattern: "{word}_{source}",
	})
	result := &SearchResult{
		URL:    "data:image/png;base64,AAAA",
		Source: Gemini,
	}

	if got := d.generateFileName("ябълка", result, 0); got != "ябълка_gemini.png" {
		t.Fatalf("generateFileName() = %q, want %q", got, "ябълка_gemini.png")
	}
}

func TestDownloadImageWritesAttribution(t *testing.T) {
	t.Parallel()

	provider := &mockDownloaderProvider{
		payload:     "image-bytes",
		attribution: "attribution text",
	}
	d := NewDownloader(provider, &DownloadOptions{
		OutputDir:         t.TempDir(),
		CreateDir:         true,
		OverwriteExisting: false,
		FileNamePattern:   "{word}_{source}",
		MaxSizeBytes:      10 * 1024 * 1024,
	})

	outputPath := filepath.Join(d.options.OutputDir, "ябълка_gemini.png")
	if err := d.DownloadImage(context.Background(), &SearchResult{
		URL:    "https://example.com/image.png",
		Source: Gemini,
		ID:     "1",
	}, outputPath); err != nil {
		t.Fatalf("DownloadImage() error = %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "image-bytes" {
		t.Fatalf("downloaded file = %q, want %q", string(data), "image-bytes")
	}

	attrPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_attribution.txt"
	attr, err := os.ReadFile(attrPath)
	if err != nil {
		t.Fatalf("ReadFile(attribution) error = %v", err)
	}
	if string(attr) != "attribution text" {
		t.Fatalf("attribution = %q, want %q", string(attr), "attribution text")
	}
}

func TestDownloadBestMatchWithOptions(t *testing.T) {
	t.Parallel()

	provider := &mockDownloaderProvider{
		results: []SearchResult{
			{
				ID:     "1",
				URL:    "https://example.com/image1.jpg",
				Source: Gemini,
			},
		},
		payload:     "image-bytes",
		attribution: "attribution text",
	}
	d := NewDownloader(provider, &DownloadOptions{
		OutputDir:         t.TempDir(),
		CreateDir:         true,
		OverwriteExisting: true,
		FileNamePattern:   "{word}_{source}",
		MaxSizeBytes:      10 * 1024 * 1024,
	})

	result, path, err := d.DownloadBestMatchWithOptions(context.Background(), &SearchOptions{Query: "ябълка"})
	if err != nil {
		t.Fatalf("DownloadBestMatchWithOptions() error = %v", err)
	}
	if result == nil || result.ID != "1" {
		t.Fatalf("DownloadBestMatchWithOptions() result = %+v, want ID 1", result)
	}
	if !strings.HasSuffix(path, ".jpg") {
		t.Fatalf("DownloadBestMatchWithOptions() path = %q, want jpg suffix", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("downloaded file missing: %v", err)
	}
}

func TestDownloadImageRejectsNilResult(t *testing.T) {
	t.Parallel()

	d := NewDownloader(&mockDownloaderProvider{}, nil)
	if err := d.DownloadImage(context.Background(), nil, filepath.Join(t.TempDir(), "out.png")); err == nil {
		t.Fatal("expected error for nil result")
	}
}
