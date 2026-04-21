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
	searchQueries []string
	downloadURLs  []string
	getAttrCalls  int
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
	m.getAttrCalls++
	return "provider attribution"
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

	if got := newDownloadPathPolicy(d.options).generateFileName("ябълка", result, 0); got != "ябълка_gemini.png" {
		t.Fatalf("generateFileName() = %q, want %q", got, "ябълка_gemini.png")
	}
}

func TestDownloadImageWritesResultAttribution(t *testing.T) {
	t.Parallel()

	provider := &mockDownloaderProvider{
		payload: "image-bytes",
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
		URL:         "https://example.com/image.png",
		Source:      Gemini,
		ID:          "1",
		Attribution: "result attribution text",
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
	if provider.getAttrCalls != 0 {
		t.Fatalf("GetAttribution() calls = %d, want 0", provider.getAttrCalls)
	}

	attrPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_attribution.txt"
	attr, err := os.ReadFile(attrPath)
	if err != nil {
		t.Fatalf("ReadFile(attribution) error = %v", err)
	}
	if string(attr) != "result attribution text" {
		t.Fatalf("attribution = %q, want %q", string(attr), "result attribution text")
	}
}

func TestDownloadBestMatchWithOptions(t *testing.T) {
	t.Parallel()

	provider := &mockDownloaderProvider{
		results: []SearchResult{
			{
				ID:          "1",
				URL:         "https://example.com/image1.jpg",
				Source:      Gemini,
				Attribution: "result attribution text",
			},
		},
		payload: "image-bytes",
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

func TestDownloadBestMatchWithOptions_SanitizesUnsafeProviderFields(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	provider := &mockDownloaderProvider{
		results: []SearchResult{
			{
				ID:          "../escape/..//id",
				URL:         "https://example.com/image1.jpg",
				Source:      "../../outside/path",
				Attribution: "safe attribution",
			},
		},
		payload: "image-bytes",
	}
	d := NewDownloader(provider, &DownloadOptions{
		OutputDir:         outputDir,
		CreateDir:         true,
		OverwriteExisting: true,
		FileNamePattern:   "{word}_{source}_{id}",
		MaxSizeBytes:      10 * 1024 * 1024,
	})

	_, path, err := d.DownloadBestMatchWithOptions(context.Background(), &SearchOptions{Query: "ябълка"})
	if err != nil {
		t.Fatalf("DownloadBestMatchWithOptions() error = %v", err)
	}

	rel, err := filepath.Rel(outputDir, path)
	if err != nil {
		t.Fatalf("filepath.Rel() error = %v", err)
	}
	if strings.HasPrefix(rel, "..") {
		t.Fatalf("path escaped output dir: %q (rel=%q)", path, rel)
	}
	if strings.Contains(path, "..") {
		t.Fatalf("path contains path traversal elements: %q", path)
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

func TestDownloadImageRejectsSymlinkAncestor(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	outsideDir := t.TempDir()
	linkDir := filepath.Join(baseDir, "link")
	if err := os.Symlink(outsideDir, linkDir); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	provider := &mockDownloaderProvider{payload: "image-bytes"}
	d := NewDownloader(provider, &DownloadOptions{
		OutputDir:         baseDir,
		CreateDir:         true,
		OverwriteExisting: true,
	})

	outputPath := filepath.Join(linkDir, "escape.png")
	err := d.DownloadImage(context.Background(), &SearchResult{
		URL:         "https://example.com/image.png",
		Source:      Gemini,
		Attribution: "safe attribution",
	}, outputPath)
	if err == nil {
		t.Fatal("expected error for symlink ancestor")
	}
	if !strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("DownloadImage() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(outsideDir, "escape.png")); !os.IsNotExist(statErr) {
		t.Fatalf("unexpected file write outside output dir: %v", statErr)
	}
}

func TestDownloadImageRejectsSymlinkSwapDuringDownload(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	outsideDir := t.TempDir()
	nestedDir := filepath.Join(baseDir, "nested")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	downloadReady := make(chan struct{})
	releaseDownload := make(chan struct{})
	provider := &swapBlockingProvider{
		payload:         "image-bytes",
		downloadReady:   downloadReady,
		releaseDownload: releaseDownload,
	}
	d := NewDownloader(provider, &DownloadOptions{
		OutputDir:         baseDir,
		CreateDir:         true,
		OverwriteExisting: true,
	})

	result := &SearchResult{
		URL:         "https://example.com/image.png",
		Source:      Gemini,
		Attribution: "safe attribution",
	}
	outputPath := filepath.Join(nestedDir, "escape.png")
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.DownloadImage(context.Background(), result, outputPath)
	}()

	<-downloadReady
	if err := os.RemoveAll(nestedDir); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}
	if err := os.Symlink(outsideDir, nestedDir); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	close(releaseDownload)

	err := <-errCh
	if err == nil {
		t.Fatal("expected error for symlink swap during download")
	}
	if !strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "open dir") {
		t.Fatalf("DownloadImage() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(outsideDir, "escape.png")); !os.IsNotExist(statErr) {
		t.Fatalf("unexpected file write outside output dir: %v", statErr)
	}
}

type swapBlockingProvider struct {
	payload         string
	downloadReady   chan struct{}
	releaseDownload chan struct{}
}

func (p *swapBlockingProvider) Name() string { return "swap-blocking" }

func (p *swapBlockingProvider) Search(context.Context, *SearchOptions) ([]SearchResult, error) {
	return nil, nil
}

func (p *swapBlockingProvider) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	close(p.downloadReady)
	<-p.releaseDownload
	return io.NopCloser(strings.NewReader(p.payload)), nil
}

func (p *swapBlockingProvider) GetAttribution(*SearchResult) string { return "" }
