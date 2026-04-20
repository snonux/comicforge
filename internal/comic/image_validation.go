package comic

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var imageLeakMarkers = []string{
	"mandatory language rule",
	"this is a bulgarian comic book",
	"bulgarian comic book",
	"mandatory panel layout",
	"mandatory speech bubbles",
	"ultra-realistic rendering",
	"final lock",
	"photorealism",
	"character & setting reference",
	"this is a text-free character gallery page",
	"story page",
	"gallery page",
	"back cover",
	"cover lines",
	"story teaser",
	"story ending hint",
	"art style:",
	"no panel grid",
	"no speech bubbles",
	"no text of any kind",
}

var englishLeakWords = []string{
	"page", "comic", "photograph", "photography", "panel", "panels",
	"cover", "title", "subtitle", "blurb", "story", "gallery",
	"rendering", "language", "mandatory", "speech", "bubble", "caption",
	"close-up", "medium-long", "ultra-realistic", "illustration", "text",
}

var validateImagePromptLeakageFn = validateImagePromptLeakage

func validateImagePromptLeakage(ctx context.Context, outputFile, label, script string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	tesseractPath, err := exec.LookPath("tesseract")
	if err != nil {
		return nil
	}

	cmd := exec.CommandContext(ctx, tesseractPath, outputFile, "stdout", "-l", "eng", "--psm", "11")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s OCR failed: %w", label, err)
	}

	ocr := strings.ToLower(out.String())
	if marker, ok := findImageLeakMarker(ocr); ok {
		return fmt.Errorf("%s contains prompt leakage marker %q (ocr=%q)", label, marker, compactOCRSnippet(ocr))
	}
	if strings.EqualFold(script, "Cyrillic") {
		if word, ok := findEnglishLeakWord(ocr); ok {
			return fmt.Errorf("%s contains English leakage word %q despite %s script (ocr=%q)", label, word, script, compactOCRSnippet(ocr))
		}
	}
	return nil
}

func findImageLeakMarker(text string) (string, bool) {
	for _, marker := range imageLeakMarkers {
		if strings.Contains(text, marker) {
			return marker, true
		}
	}
	return "", false
}

func findEnglishLeakWord(text string) (string, bool) {
	for _, word := range englishLeakWords {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
		if pattern.MatchString(text) {
			return word, true
		}
	}
	return "", false
}

func compactOCRSnippet(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 240 {
		text = text[:240] + "..."
	}
	return text
}
