package comic

import (
	"strings"
	"testing"
)

func TestLocalizedBrandName(t *testing.T) {
	t.Parallel()

	if got, want := localizedBrandName("Bulgarian", "Cyrillic"), "КомиксФордж Приключения"; got != want {
		t.Fatalf("localizedBrandName() = %q, want %q", got, want)
	}
	if got, want := localizedBrandName("English", "Latin"), "ComicForge Adventures"; got != want {
		t.Fatalf("localizedBrandName() = %q, want %q", got, want)
	}
}

func TestValidateTextScript(t *testing.T) {
	t.Parallel()

	if err := validateTextScript("title", "Заглавие", "Cyrillic"); err != nil {
		t.Fatalf("validateTextScript() error = %v", err)
	}
	if err := validateTextScript("title", "Title", "Cyrillic"); err == nil {
		t.Fatal("validateTextScript() error = nil, want Latin rejection")
	}
	if err := validateTextScript("title", "Title", "Latin"); err != nil {
		t.Fatalf("validateTextScript() error = %v", err)
	}
	if err := validateTextScript("title", "Заглавие", "Latin"); err == nil {
		t.Fatal("validateTextScript() error = nil, want Cyrillic rejection")
	}
}

func TestValidateNoPromptLeakage(t *testing.T) {
	t.Parallel()

	if err := validateNoPromptLeakage("story", "A calm story without artifacts."); err != nil {
		t.Fatalf("validateNoPromptLeakage() error = %v", err)
	}
	if err := validateNoPromptLeakage("story", "Words to include:\n- ябълка"); err == nil {
		t.Fatal("validateNoPromptLeakage() error = nil, want prompt leakage rejection")
	}
}

func TestFindImageLeakMarker(t *testing.T) {
	t.Parallel()

	if marker, ok := findImageLeakMarker("calm text"); ok || marker != "" {
		t.Fatalf("findImageLeakMarker() = %q, %v, want no match", marker, ok)
	}
	if marker, ok := findImageLeakMarker("mandatory language rule"); !ok || marker != "mandatory language rule" {
		t.Fatalf("findImageLeakMarker() = %q, %v, want mandatory language rule match", marker, ok)
	}
}

func TestFindEnglishLeakWord(t *testing.T) {
	t.Parallel()

	if word, ok := findEnglishLeakWord("спокойна сцена"); ok || word != "" {
		t.Fatalf("findEnglishLeakWord() = %q, %v, want no match", word, ok)
	}
	if word, ok := findEnglishLeakWord("page 1 of 5"); !ok || word != "page" {
		t.Fatalf("findEnglishLeakWord() = %q, %v, want page match", word, ok)
	}
}

func TestComicStylesStayComic(t *testing.T) {
	t.Parallel()

	for _, style := range defaultStyles[styleModeComic] {
		if strings.Contains(strings.ToLower(style), "ultra realistic") || strings.Contains(strings.ToLower(style), "photograph") {
			t.Fatalf("comic styles contain hybrid or photorealistic style: %q", style)
		}
	}
}

func TestContainsLatinLetters(t *testing.T) {
	t.Parallel()

	if containsLatinLetters("Здравей") {
		t.Fatal("containsLatinLetters() = true for Cyrillic text")
	}
	if !containsLatinLetters("Page 4 of 5") {
		t.Fatal("containsLatinLetters() = false for Latin text")
	}
}

func TestLocalizedStylePrompt(t *testing.T) {
	t.Parallel()

	got := localizedStylePrompt("classic American comic book with bold ink outlines, halftone dots, and primary colors", "Bulgarian", "Cyrillic")
	if containsLatinLetters(got) {
		t.Fatalf("localizedStylePrompt() = %q, want no Latin letters", got)
	}
	if got == "" {
		t.Fatal("localizedStylePrompt() returned empty string")
	}
}
