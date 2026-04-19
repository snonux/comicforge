package comic

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// AssembleComicPDF combines comic pages into a PDF using ImageMagick.
func AssembleComicPDF(outputDir, titleSlug string, imagePaths []string) (string, error) {
	if len(imagePaths) == 0 {
		return "", fmt.Errorf("no comic images to assemble into PDF")
	}
	if _, err := exec.LookPath("convert"); err != nil {
		return "", fmt.Errorf("ImageMagick 'convert' not found — install ImageMagick to generate the PDF")
	}

	pdfPath := filepath.Join(outputDir, titleSlug+".pdf")
	args := []string{"-density", "150"}
	args = append(args, imagePaths...)
	args = append(args, pdfPath)

	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("convert failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return pdfPath, nil
}
