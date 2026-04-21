package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
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

	reader, err := d.provider.Download(ctx, result.URL)
	if err != nil {
		return fmt.Errorf("download %q: %w", result.URL, err)
	}
	defer func() {
		_ = reader.Close()
	}()

	store := newSecureOutputStore(d.options)
	file, parentFD, finalName, err := store.openSecureOutputFile(outputPath)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", outputPath, err)
	}
	defer func() {
		_ = unix.Close(parentFD)
	}()
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close output file %q: %w", outputPath, closeErr)
		}
	}()

	if d.options != nil && d.options.MaxSizeBytes > 0 {
		written, copyErr := io.CopyN(file, reader, d.options.MaxSizeBytes)
		if copyErr != nil && copyErr != io.EOF {
			_ = unix.Unlinkat(parentFD, finalName, 0)
			return fmt.Errorf("write output file %q: %w", outputPath, copyErr)
		}

		if written == d.options.MaxSizeBytes {
			var probe [1]byte
			if n, probeErr := reader.Read(probe[:]); n > 0 || probeErr != io.EOF {
				_ = unix.Unlinkat(parentFD, finalName, 0)
				return fmt.Errorf("image exceeds max size %d bytes", d.options.MaxSizeBytes)
			}
		}
	} else {
		if _, err = io.Copy(file, reader); err != nil {
			_ = unix.Unlinkat(parentFD, finalName, 0)
			return fmt.Errorf("write output file %q: %w", outputPath, err)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync output file %q: %w", outputPath, err)
	}

	if attribution := strings.TrimSpace(result.Attribution); attribution != "" {
		attrRelPath := strings.TrimSuffix(finalName, filepath.Ext(finalName)) + "_attribution.txt"
		if err := store.writeSecureRelativeFile(outputPath, attrRelPath, []byte(attribution), 0o644); err != nil {
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
	if searchOpts.PerPage == 0 {
		searchOpts.PerPage = 5
	}

	results, err := d.provider.Search(ctx, &searchOpts)
	if err != nil {
		return nil, "", fmt.Errorf("search images: %w", err)
	}
	if len(results) == 0 {
		return nil, "", fmt.Errorf("no images found for %q", opts.Query)
	}

	policy := newDownloadPathPolicy(d.options)
	for i, result := range results {
		filename := policy.generateFileName(opts.Query, &result, i)
		outputPath, err := policy.resolveOutputPath(filename)
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
