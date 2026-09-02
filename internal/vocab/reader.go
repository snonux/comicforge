// Package vocab reads language-agnostic vocabulary files.
package vocab

import (
	"fmt"
	"os"
	"strings"
)

// CardType identifies the semantic shape of a vocabulary entry.
type CardType string

const (
	// CardTypeTranslation marks a standard word-to-translation entry.
	CardTypeTranslation CardType = "translation"
	// CardTypeDefinition marks a same-language word-to-definition entry.
	CardTypeDefinition CardType = "definition"
)

// WordEntry represents a vocabulary entry with an optional translation.
type WordEntry struct {
	Word             string
	Translation      string
	NeedsTranslation bool
	CardType         CardType
}

// ReadVocabularyFile reads vocabulary entries from a file and returns the parsed slice.
func ReadVocabularyFile(filename string) ([]WordEntry, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read vocabulary file: %w", err)
	}

	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	entries := make([]WordEntry, 0, len(lines))
	for _, line := range lines {
		if entry := parseLine(line); entry != nil {
			entries = append(entries, *entry)
		}
	}

	if len(entries) == 0 {
		return nil, nil
	}

	return entries, nil
}

func parseLine(line string) *WordEntry {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}
	if strings.Contains(trimmed, "==") {
		return parseDefinitionEntry(trimmed)
	}
	if strings.Contains(trimmed, "=") {
		return parseTranslationEntry(trimmed)
	}
	return &WordEntry{
		Word:             trimmed,
		Translation:      "",
		NeedsTranslation: false,
		CardType:         CardTypeTranslation,
	}
}

func parseDefinitionEntry(trimmed string) *WordEntry {
	parts := strings.SplitN(trimmed, "==", 2)
	if len(parts) != 2 {
		return nil
	}
	word := strings.TrimSpace(parts[0])
	translation := strings.TrimSpace(parts[1])
	if word == "" || translation == "" {
		return nil
	}
	return &WordEntry{
		Word:             word,
		Translation:      translation,
		NeedsTranslation: false,
		CardType:         CardTypeDefinition,
	}
}

func parseTranslationEntry(trimmed string) *WordEntry {
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return nil
	}
	word := strings.TrimSpace(parts[0])
	translation := strings.TrimSpace(parts[1])
	if word == "" && translation != "" {
		return &WordEntry{
			Word:             "",
			Translation:      translation,
			NeedsTranslation: true,
			CardType:         CardTypeTranslation,
		}
	}
	if word != "" && translation != "" {
		return &WordEntry{
			Word:             word,
			Translation:      translation,
			NeedsTranslation: false,
			CardType:         CardTypeTranslation,
		}
	}
	return nil
}
