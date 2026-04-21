package image

import (
	"fmt"
	"path/filepath"
	"strings"
)

type downloadPathPolicy struct {
	options *DownloadOptions
}

func newDownloadPathPolicy(options *DownloadOptions) downloadPathPolicy {
	return downloadPathPolicy{options: options}
}

func (p downloadPathPolicy) generateFileName(word string, result *SearchResult, index int) string {
	filename := ""
	if p.options != nil {
		filename = p.options.FileNamePattern
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

func (p downloadPathPolicy) resolveOutputPath(name string) (string, error) {
	baseDir := "./images"
	if p.options != nil && p.options.OutputDir != "" {
		baseDir = p.options.OutputDir
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
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes base dir")
	}

	return fullPath, nil
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
