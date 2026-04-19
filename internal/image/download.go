package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DownloadOptions configures image download behavior.
type DownloadOptions struct {
	OutputDir         string
	OverwriteExisting bool
	CreateDir         bool
	FileNamePattern   string
	MaxSizeBytes      int64
}

// DefaultDownloadOptions returns sensible defaults for image downloads.
func DefaultDownloadOptions() *DownloadOptions {
	return &DownloadOptions{
		OutputDir:         "./images",
		OverwriteExisting: false,
		CreateDir:         true,
		FileNamePattern:   "{word}_{source}",
		MaxSizeBytes:      10 * 1024 * 1024,
	}
}

// Downloader handles image downloads from search results.
type Downloader struct {
	provider ImageProvider
	options  *DownloadOptions
}

// NewDownloader creates a new image downloader.
func NewDownloader(provider ImageProvider, options *DownloadOptions) *Downloader {
	if options == nil {
		options = DefaultDownloadOptions()
	}
	return &Downloader{
		provider: provider,
		options:  options,
	}
}

// DownloadImage downloads a single image to the specified path.
func (d *Downloader) DownloadImage(ctx context.Context, result *SearchResult, outputPath string) (err error) {
	if d == nil || d.provider == nil {
		return fmt.Errorf("image provider is required")
	}
	if result == nil {
		return fmt.Errorf("search result is required")
	}

	dir := filepath.Dir(outputPath)
	if d.options != nil && d.options.CreateDir && dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output dir %q: %w", dir, err)
		}
	}
	if err := ensureNoSymlinkAncestors(outputPath); err != nil {
		return fmt.Errorf("validate output path %q: %w", outputPath, err)
	}

	if d.options != nil && !d.options.OverwriteExisting {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("output file exists: %s", outputPath)
		}
	}

	reader, err := d.provider.Download(ctx, result.URL)
	if err != nil {
		return fmt.Errorf("download %q: %w", result.URL, err)
	}
	defer func() {
		_ = reader.Close()
	}()

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", outputPath, err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close output file %q: %w", outputPath, closeErr)
		}
	}()

	if d.options != nil && d.options.MaxSizeBytes > 0 {
		written, copyErr := io.CopyN(file, reader, d.options.MaxSizeBytes)
		if copyErr != nil && copyErr != io.EOF {
			_ = os.Remove(outputPath)
			return fmt.Errorf("write output file %q: %w", outputPath, copyErr)
		}

		if written == d.options.MaxSizeBytes {
			var probe [1]byte
			if n, probeErr := reader.Read(probe[:]); n > 0 || probeErr != io.EOF {
				_ = os.Remove(outputPath)
				return fmt.Errorf("image exceeds max size %d bytes", d.options.MaxSizeBytes)
			}
		}
	} else {
		if _, err = io.Copy(file, reader); err != nil {
			_ = os.Remove(outputPath)
			return fmt.Errorf("write output file %q: %w", outputPath, err)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync output file %q: %w", outputPath, err)
	}

	if attribution := strings.TrimSpace(result.Attribution); attribution != "" {
		attrPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_attribution.txt"
		if err := os.WriteFile(attrPath, []byte(attribution), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save attribution: %v\n", err)
		}
	}

	return nil
}

// DownloadBestMatch downloads the best matching image for a query.
func (d *Downloader) DownloadBestMatch(ctx context.Context, query string) (*SearchResult, string, error) {
	opts := DefaultSearchOptions(query)
	opts.PerPage = 5
	return d.DownloadBestMatchWithOptions(ctx, opts)
}

// DownloadBestMatchWithOptions downloads the best matching image for given search options.
func (d *Downloader) DownloadBestMatchWithOptions(ctx context.Context, opts *SearchOptions) (*SearchResult, string, error) {
	if d == nil || d.provider == nil {
		return nil, "", fmt.Errorf("image provider is required")
	}
	if opts == nil {
		return nil, "", fmt.Errorf("search options are required")
	}

	searchOpts := *opts
	searchOpts.PerPage = 5

	results, err := d.provider.Search(ctx, &searchOpts)
	if err != nil {
		return nil, "", fmt.Errorf("search images: %w", err)
	}
	if len(results) == 0 {
		return nil, "", fmt.Errorf("no images found for %q", opts.Query)
	}

	for i, result := range results {
		filename := d.generateFileName(opts.Query, &result, i)
		outputDir := "./images"
		if d.options != nil && d.options.OutputDir != "" {
			outputDir = d.options.OutputDir
		}
		outputPath, err := joinWithinBaseDir(outputDir, filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: refusing unsafe output path %q: %v\n", filename, err)
			continue
		}

		if err := d.DownloadImage(ctx, &result, outputPath); err == nil {
			return &result, outputPath, nil
		} else {
			fmt.Fprintf(os.Stderr, "Warning: failed to download image %d: %v\n", i+1, err)
		}
	}

	return nil, "", fmt.Errorf("no downloadable images found for %q", opts.Query)
}

func (d *Downloader) generateFileName(word string, result *SearchResult, index int) string {
	filename := ""
	if d != nil && d.options != nil {
		filename = d.options.FileNamePattern
	}
	if filename == "" {
		filename = "{word}_{source}"
	}

	filename = strings.ReplaceAll(filename, "{word}", sanitizeFileName(word))
	if result != nil {
		filename = strings.ReplaceAll(filename, "{source}", sanitizeFileName(result.Source))
		filename = strings.ReplaceAll(filename, "{id}", sanitizeFileName(result.ID))
	}
	filename = strings.ReplaceAll(filename, "{index}", fmt.Sprintf("%d", index))

	ext := ""
	if result != nil {
		ext = filepath.Ext(result.URL)
		if strings.HasPrefix(result.URL, geminiDataPrefix) {
			ext = ".png"
		} else if ext == "" || len(ext) > 5 {
			ext = ".jpg"
		}
	}

	if filepath.Ext(filename) == "" {
		filename += ext
	}

	return filename
}

func joinWithinBaseDir(baseDir, name string) (string, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}

	cleanBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve base dir: %w", err)
	}

	fullPath, err := filepath.Abs(filepath.Join(cleanBase, name))
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}

	rel, err := filepath.Rel(cleanBase, fullPath)
	if err != nil {
		return "", fmt.Errorf("relativize output path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes base dir")
	}

	return fullPath, nil
}

func ensureNoSymlinkAncestors(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	absPath = filepath.Clean(absPath)
	dir := filepath.Dir(absPath)
	root := filepath.VolumeName(absPath) + string(os.PathSeparator)
	if filepath.VolumeName(absPath) == "" {
		root = string(os.PathSeparator)
	}

	for {
		if dir == root || dir == "." || dir == string(os.PathSeparator) {
			break
		}

		if info, err := os.Lstat(dir); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("path component %q is a symlink", dir)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat path component %q: %w", dir, err)
		}

		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}

	if info, err := os.Lstat(absPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path %q is a symlink", absPath)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat output path %q: %w", absPath, err)
	}

	return nil
}

func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
		".", "_",
	)

	sanitized := replacer.Replace(name)
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}

	return sanitized
}
