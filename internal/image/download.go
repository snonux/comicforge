package image

import (
	"context"
	"errors"
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

	file, parentFD, finalName, err := d.openSecureOutputFile(outputPath)
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
		if err := d.writeSecureRelativeFile(outputPath, attrRelPath, []byte(attribution), 0o644); err != nil {
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
		outputPath, err := d.resolveOutputPath(filename)
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

func (d *Downloader) resolveOutputPath(name string) (string, error) {
	baseDir := "./images"
	if d != nil && d.options != nil && d.options.OutputDir != "" {
		baseDir = d.options.OutputDir
	}

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

func (d *Downloader) openSecureOutputFile(outputPath string) (*os.File, int, string, error) {
	baseDir := "./images"
	if d != nil && d.options != nil && d.options.OutputDir != "" {
		baseDir = d.options.OutputDir
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, 0, "", fmt.Errorf("resolve base dir: %w", err)
	}
	targetAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("resolve output path: %w", err)
	}
	relPath, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return nil, 0, "", fmt.Errorf("relativize output path: %w", err)
	}
	if relPath == "." || relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return nil, 0, "", fmt.Errorf("path escapes base dir")
	}

	relPath = filepath.Clean(relPath)
	parentRel := filepath.Dir(relPath)
	finalName := filepath.Base(relPath)

	parentFD, err := openDirPathNoFollow(baseAbs, parentRel, d.options != nil && d.options.CreateDir)
	if err != nil {
		return nil, 0, "", err
	}

	flags := unix.O_WRONLY | unix.O_CREAT | unix.O_CLOEXEC | unix.O_NOFOLLOW
	if d.options != nil && d.options.OverwriteExisting {
		flags |= unix.O_TRUNC
	} else {
		flags |= unix.O_EXCL
	}

	fd, err := unix.Openat(parentFD, finalName, flags, 0o644)
	if err != nil {
		_ = unix.Close(parentFD)
		return nil, 0, "", fmt.Errorf("open output file: %w", err)
	}

	return os.NewFile(uintptr(fd), targetAbs), parentFD, finalName, nil
}

func (d *Downloader) writeSecureRelativeFile(outputPath, relPath string, content []byte, perm os.FileMode) error {
	file, parentFD, finalName, err := d.openSecureRelativeFile(outputPath, relPath, perm)
	if err != nil {
		return err
	}
	defer func() {
		_ = unix.Close(parentFD)
	}()
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Write(content); err != nil {
		_ = unix.Unlinkat(parentFD, finalName, 0)
		return fmt.Errorf("write attribution file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = unix.Unlinkat(parentFD, finalName, 0)
		return fmt.Errorf("sync attribution file: %w", err)
	}
	return nil
}

func (d *Downloader) openSecureRelativeFile(outputPath, relPath string, perm os.FileMode) (*os.File, int, string, error) {
	baseDir := "./images"
	if d != nil && d.options != nil && d.options.OutputDir != "" {
		baseDir = d.options.OutputDir
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, 0, "", fmt.Errorf("resolve base dir: %w", err)
	}
	targetAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("resolve output path: %w", err)
	}
	baseRelPath, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return nil, 0, "", fmt.Errorf("relativize output path: %w", err)
	}
	if baseRelPath == "." || baseRelPath == ".." || strings.HasPrefix(baseRelPath, ".."+string(os.PathSeparator)) {
		return nil, 0, "", fmt.Errorf("path escapes base dir")
	}

	parentRel := filepath.Dir(filepath.Clean(baseRelPath))
	parentFD, err := openDirPathNoFollow(baseAbs, parentRel, true)
	if err != nil {
		return nil, 0, "", err
	}

	flags := unix.O_WRONLY | unix.O_CREAT | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_TRUNC
	fd, err := unix.Openat(parentFD, relPath, flags, uint32(perm.Perm()))
	if err != nil {
		_ = unix.Close(parentFD)
		return nil, 0, "", fmt.Errorf("open attribution file: %w", err)
	}

	return os.NewFile(uintptr(fd), targetAbs), parentFD, relPath, nil
}

func openDirPathNoFollow(baseAbs, relPath string, createDirs bool) (int, error) {
	dirFD, err := openPathDirNoFollow(baseAbs, createDirs)
	if err != nil {
		return 0, err
	}

	relPath = filepath.Clean(relPath)
	if relPath == "." {
		return dirFD, nil
	}

	parts := strings.Split(relPath, string(os.PathSeparator))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if createDirs {
			if err := unix.Mkdirat(dirFD, part, 0o755); err != nil && !errors.Is(err, unix.EEXIST) {
				_ = unix.Close(dirFD)
				return 0, fmt.Errorf("create dir %q: %w", part, err)
			}
		}

		nextFD, err := unix.Openat(dirFD, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			_ = unix.Close(dirFD)
			return 0, fmt.Errorf("open dir %q: %w", part, err)
		}
		_ = unix.Close(dirFD)
		dirFD = nextFD
	}

	return dirFD, nil
}

func openPathDirNoFollow(absPath string, createDirs bool) (int, error) {
	absPath = filepath.Clean(absPath)
	if absPath == string(os.PathSeparator) {
		return unix.Open(absPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	}

	parentAbs := filepath.Dir(absPath)
	leaf := filepath.Base(absPath)
	parentFD, err := openPathDirNoFollow(parentAbs, createDirs)
	if err != nil {
		return 0, err
	}

	if createDirs {
		if err := unix.Mkdirat(parentFD, leaf, 0o755); err != nil && !errors.Is(err, unix.EEXIST) {
			_ = unix.Close(parentFD)
			return 0, fmt.Errorf("create dir %q: %w", leaf, err)
		}
	}

	dirFD, err := unix.Openat(parentFD, leaf, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	_ = unix.Close(parentFD)
	if err != nil {
		return 0, fmt.Errorf("open dir %q: %w", leaf, err)
	}

	return dirFD, nil
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
