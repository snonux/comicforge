package image

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPromptSubjectUsesTranslationFirst(t *testing.T) {
	t.Parallel()

	if got := promptSubject(" apple ", "ябълка"); got != "apple" {
		t.Fatalf("promptSubject() = %q, want %q", got, "apple")
	}
}

func TestPromptSubjectFallsBackToOriginalWord(t *testing.T) {
	t.Parallel()

	if got := promptSubject("   ", "ябълка"); got != "ябълка" {
		t.Fatalf("promptSubject() = %q, want %q", got, "ябълка")
	}
}

func TestSanitizeSceneDescriptionRemovesLabelsAndFences(t *testing.T) {
	t.Parallel()

	scene := "```text\nScene: A bright apple sits centered on a wooden table.\n```"
	if got := sanitizeSceneDescription(scene); got != "A bright apple sits centered on a wooden table" {
		t.Fatalf("sanitizeSceneDescription() = %q", got)
	}
}

func TestUsableSceneDescriptionRejectsTrivialContent(t *testing.T) {
	t.Parallel()

	if usableSceneDescription("A") {
		t.Fatal("usableSceneDescription() unexpectedly accepted trivial scene")
	}
	if usableSceneDescription("A single, perfectly") {
		t.Fatal("usableSceneDescription() unexpectedly accepted incomplete scene fragment")
	}
}

func TestFallbackVisualDirectionForPhrase(t *testing.T) {
	t.Parallel()

	got := fallbackVisualDirection("to indulge someone")
	if got == "" || !usableSceneDescription(got) {
		t.Fatalf("fallbackVisualDirection() returned unusable phrase direction: %q", got)
	}
}

func TestFallbackVisualDirectionForSingleWord(t *testing.T) {
	t.Parallel()

	got := fallbackVisualDirection("apple")
	if got == "" || !strings.Contains(got, "single apple") {
		t.Fatalf("fallbackVisualDirection() = %q", got)
	}
}

func TestBuildEducationalPromptTruncatesLongSceneWithoutBreakingUTF8(t *testing.T) {
	t.Parallel()

	style := "Photorealism"
	subject := "ябълка"
	var maxSceneLen int
	for i := 0; i < 3; i++ {
		candidateSubject := subject + strings.Repeat("x", i)
		template := fmt.Sprintf(
			"Generate a %s flashcard image illustrating \"%s\". Scene:  "+
				"The image should be educational and suitable for language learning flashcards. "+
				"Requirements: The main subject or concept must be clearly visible, centered, well lit, and easy to identify.",
			style, candidateSubject,
		)
		maxSceneLen = maxImagePromptChars - len(template)
		if maxSceneLen%3 != 0 {
			subject = candidateSubject
			break
		}
	}

	scene := "abc" + strings.Repeat("界", maxSceneLen+32)
	if len(scene) <= maxSceneLen {
		t.Fatalf("test setup failed: scene length = %d, want > %d", len(scene), maxSceneLen)
	}
	if utf8.ValidString(scene[:maxSceneLen]) {
		t.Fatalf("test setup failed: raw byte slice unexpectedly remained valid UTF-8 at cut position %d", maxSceneLen)
	}
	got := buildEducationalPrompt(style, scene, subject)

	if !utf8.ValidString(got) {
		t.Fatalf("buildEducationalPrompt() returned invalid UTF-8: %q", got)
	}
	if len(got) > maxImagePromptChars {
		t.Fatalf("buildEducationalPrompt() length = %d, want <= %d", len(got), maxImagePromptChars)
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("buildEducationalPrompt() = %q, want truncated scene marker", got)
	}
	if !strings.Contains(got, "abc") {
		t.Fatalf("buildEducationalPrompt() = %q, want ASCII scene prefix to survive truncation", got)
	}
}
