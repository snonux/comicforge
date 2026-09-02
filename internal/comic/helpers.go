package comic

import (
	"strings"
)

const maxPromptSlugLength = 48

// slugify converts a comic title into a safe file-name component.
func slugify(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	prevHyphen := false
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}

	slug := strings.TrimRight(b.String(), "-")
	if slug == "" {
		return "comic"
	}
	return slug
}

func shortSlug(title, fallback string) string {
	slug := slugify(title)
	if slug == "comic" {
		slug = fallback
	}
	if slug == "" {
		slug = fallback
	}
	if len(slug) <= maxPromptSlugLength {
		return slug
	}
	cut := strings.LastIndex(slug[:maxPromptSlugLength+1], "-")
	if cut < 12 {
		cut = maxPromptSlugLength
	}
	slug = strings.TrimRight(slug[:cut], "-")
	if slug == "" {
		return fallback
	}
	return slug
}

func shortPromptSlugFromText(text string) string {
	return shortSlug(text, "manual-prompt")
}

// splitIntoSections divides text into n sections, preferring paragraph boundaries.
func splitIntoSections(text string, n int) []string {
	paragraphs := splitParagraphs(text)
	if len(paragraphs) >= n {
		return distributeParagraphs(paragraphs, n)
	}
	return splitByChars(text, n)
}

// splitParagraphs returns non-empty paragraphs separated by blank lines.
func splitParagraphs(text string) []string {
	var out []string
	for _, paragraph := range strings.Split(text, "\n\n") {
		if paragraph = strings.TrimSpace(paragraph); paragraph != "" {
			out = append(out, paragraph)
		}
	}
	return out
}

func distributeParagraphs(paragraphs []string, n int) []string {
	sections := make([]string, n)
	size, rem, idx := len(paragraphs)/n, len(paragraphs)%n, 0
	for i := range n {
		count := size
		if i < rem {
			count++
		}
		sections[i] = strings.Join(paragraphs[idx:idx+count], "\n\n")
		idx += count
	}
	return sections
}

func splitByChars(text string, n int) []string {
	runes := []rune(text)
	size := len(runes) / n
	sections := make([]string, n)
	for i := range n {
		start := i * size
		end := start + size
		if i == n-1 {
			end = len(runes)
		}
		sections[i] = strings.TrimSpace(string(runes[start:end]))
	}
	return sections
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func cloneStyleMap(styles map[string][]string) map[string][]string {
	if styles == nil {
		return nil
	}
	out := make(map[string][]string, len(styles))
	for mode, lines := range styles {
		out[mode] = append([]string(nil), lines...)
	}
	return out
}
