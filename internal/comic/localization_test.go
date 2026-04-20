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

func TestComicStylesStayComic(t *testing.T) {
	t.Parallel()

	for _, style := range comicStyles {
		if strings.Contains(strings.ToLower(style), "ultra realistic") || strings.Contains(strings.ToLower(style), "photograph") {
			t.Fatalf("comicStyles contains hybrid or photorealistic style: %q", style)
		}
	}
}
