package comic

import (
	"fmt"
	"strings"
	"unicode"
)

var promptLeakMarkers = []string{
	"words to include",
	"required words",
	"character guide",
	"comic title",
	"panel script",
	"mandatory language rule",
	"mandatory panel layout",
	"mandatory speech bubbles",
	"strict consistency rules",
	"story excerpt",
	"story teaser",
	"story ending hint",
	"---character guide---",
	"---comic title---",
	"---panel script---",
}

func validateTextScript(label, text, script string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("%s is empty", label)
	}

	switch {
	case strings.EqualFold(script, "Cyrillic"):
		if containsLatinLetters(text) {
			return fmt.Errorf("%s contains Latin letters despite %s script", label, script)
		}
	case strings.EqualFold(script, "Latin"):
		if containsCyrillicLetters(text) {
			return fmt.Errorf("%s contains Cyrillic letters despite %s script", label, script)
		}
	}
	return nil
}

func containsLatinLetters(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Latin, r) {
			return true
		}
	}
	return false
}

func containsCyrillicLetters(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

func validateNoPromptLeakage(label, text string) error {
	lower := strings.ToLower(text)
	for _, marker := range promptLeakMarkers {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("%s contains prompt leakage marker %q", label, marker)
		}
	}
	return nil
}
