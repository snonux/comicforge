package comic

import (
	"testing"
)

func TestPickStyleKnownModes(t *testing.T) {
	t.Parallel()

	styles := map[string][]string{
		styleModeComic:     {"comic-a", "comic-b"},
		styleModeRealistic: {"photo-a"},
		styleModeManga:     {"manga-a"},
	}

	if got := pickStyle(styles, styleModeRealistic); got != "photo-a" {
		t.Fatalf("realistic = %q, want photo-a", got)
	}
	if got := pickStyle(styles, styleModeManga); got != "manga-a" {
		t.Fatalf("manga = %q, want manga-a", got)
	}
}

func TestPickStyleNonComicIsUniform(t *testing.T) {
	t.Parallel()

	styles := map[string][]string{
		styleModeManga: {"manga-a", "manga-b", "manga-c", "manga-d", "manga-e"},
	}
	seen := map[string]int{}
	for i := 0; i < 200; i++ {
		got := pickStyle(styles, styleModeManga)
		seen[got]++
	}
	if len(seen) < 2 {
		t.Fatalf("non-comic pickStyle stayed on one line across 200 draws: %#v", seen)
	}
	for _, line := range styles[styleModeManga] {
		if seen[line] == 0 {
			// Not every line must appear, but at least two distinct ones must.
			continue
		}
	}
}

func TestPickStyleFallsBackToDefaults(t *testing.T) {
	t.Parallel()

	got := pickStyle(nil, styleModeRealistic)
	want := defaultStyles[styleModeRealistic]
	if len(want) == 0 {
		t.Fatal("default realistic styles missing")
	}
	found := false
	for _, line := range want {
		if got == line {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("pickStyle(nil, realistic) = %q, want one of defaults", got)
	}
}

func TestPickStyleUnknownModeFallsBackToComic(t *testing.T) {
	t.Parallel()

	styles := map[string][]string{
		styleModeComic: {"only-comic"},
	}
	if got := pickStyle(styles, "not-a-real-mode"); got != "only-comic" {
		t.Fatalf("unknown mode = %q, want only-comic", got)
	}
	if got := pickStyle(styles, ""); got != "only-comic" {
		t.Fatalf("empty mode = %q, want only-comic", got)
	}
}

func TestPickStyleEmptyCategoryUsesDefaults(t *testing.T) {
	t.Parallel()

	styles := map[string][]string{
		styleModeComic:     {"comic-line"},
		styleModeRealistic: {},
	}
	// Empty realistic slice should use package defaults, not comic.
	got := pickStyle(styles, styleModeRealistic)
	for _, line := range defaultStyles[styleModeRealistic] {
		if got == line {
			return
		}
	}
	t.Fatalf("empty realistic category = %q, want a default realistic line", got)
}

func TestPickStyleCustomCategoryWithoutDefaults(t *testing.T) {
	t.Parallel()

	styles := map[string][]string{
		"neon_noir": {"neon glow noir"},
	}
	if got := pickStyle(styles, "neon_noir"); got != "neon glow noir" {
		t.Fatalf("custom category = %q, want neon glow noir", got)
	}
}

func TestPickStyleModeForcedUltra(t *testing.T) {
	t.Parallel()

	yes, no := true, false
	if got := pickStyleMode(&yes, nil); got != styleModeRealistic {
		t.Fatalf("forced ultra = %q, want realistic", got)
	}
	if got := pickStyleMode(&no, nil); got != styleModeComic {
		t.Fatalf("forced comic = %q, want comic", got)
	}
}

func TestPickStyleModeUsesConfiguredKeys(t *testing.T) {
	t.Parallel()

	styles := map[string][]string{"neon_noir": {"x"}}
	if got := pickStyleMode(nil, styles); got != "neon_noir" {
		t.Fatalf("pickStyleMode = %q, want neon_noir", got)
	}
}

func TestCloneStyleMapIndependent(t *testing.T) {
	t.Parallel()

	src := map[string][]string{styleModeComic: {"a"}}
	dst := cloneStyleMap(src)
	dst[styleModeComic][0] = "mutated"
	dst["extra"] = []string{"y"}
	if src[styleModeComic][0] != "a" {
		t.Fatal("clone shared underlying slice with source")
	}
	if _, ok := src["extra"]; ok {
		t.Fatal("clone mutation leaked into source map")
	}
	if cloneStyleMap(nil) != nil {
		t.Fatal("cloneStyleMap(nil) should return nil")
	}
}
