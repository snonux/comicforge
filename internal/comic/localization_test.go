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
		lower := strings.ToLower(style.English)
		if strings.Contains(lower, "ultra realistic") || strings.Contains(lower, "photograph") {
			t.Fatalf("comic styles contain hybrid or photorealistic style: %q", style.English)
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

func TestLocalizedStylePromptAllDefaults(t *testing.T) {
	t.Parallel()

	for mode, prompts := range defaultStyles {
		for _, p := range prompts {
			got := localizedStylePrompt(p.English, "Bulgarian", "Cyrillic")
			if got != p.Localized {
				t.Fatalf("mode %s style %q: got %q, want Localized field", mode, p.English, got)
			}
			if containsLatinLetters(got) {
				t.Fatalf("mode %s style Localized has Latin letters: %q", mode, got)
			}
			if !containsCyrillicLetters(got) {
				t.Fatalf("mode %s style Localized has no Cyrillic letters: %q", mode, got)
			}
		}
	}
}

func TestLocalizedStylePromptLatinPassthrough(t *testing.T) {
	t.Parallel()

	style := "custom neon noir with glowing alleys"
	if got := localizedStylePrompt(style, "English", "Latin"); got != style {
		t.Fatalf("Latin passthrough = %q, want %q", got, style)
	}
}

func TestLocalizedStylePromptMissingPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("localizedStylePrompt() did not panic for missing Cyrillic localization")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "no Cyrillic localization") {
			t.Fatalf("panic = %#v, want missing-localization message", r)
		}
	}()
	_ = localizedStylePrompt("totally unknown custom style line", "Bulgarian", "Cyrillic")
}

func TestDefaultStylesHaveLocalizations(t *testing.T) {
	t.Parallel()

	if len(defaultStyles) == 0 {
		t.Fatal("defaultStyles empty")
	}
	if len(cyrillicStyleByEnglish) == 0 {
		t.Fatal("cyrillicStyleByEnglish empty")
	}
	total := 0
	for mode, prompts := range defaultStyles {
		if len(prompts) == 0 {
			t.Fatalf("mode %s has no style prompts", mode)
		}
		for _, p := range prompts {
			total++
			if p.English == "" || p.Localized == "" {
				t.Fatalf("mode %s has empty English or Localized", mode)
			}
			if got, ok := cyrillicStyleByEnglish[p.English]; !ok || got != p.Localized {
				t.Fatalf("lookup missing or mismatched for %q", p.English)
			}
		}
	}
	if total < 30 {
		t.Fatalf("expected 30+ built-in styles, got %d", total)
	}
}

func TestConfigDefaultStylesLocalize(t *testing.T) {
	t.Parallel()

	for mode, lines := range DefaultStyleLines() {
		for _, line := range lines {
			got := localizedStylePrompt(line, "Bulgarian", "Cyrillic")
			if containsLatinLetters(got) {
				t.Fatalf("config default mode %s leaked Latin: %q", mode, got)
			}
		}
	}
}
