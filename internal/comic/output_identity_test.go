package comic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

type identityImageProvider struct {
	calls int
}

func (p *identityImageProvider) Name() string       { return "identity-image" }
func (p *identityImageProvider) IsAvailable() error { return nil }
func (p *identityImageProvider) GenerateImage(_ context.Context, _ string, path string) error {
	p.calls++
	return os.WriteFile(path, fmt.Appendf(nil, "image %d", p.calls), 0o644)
}

func TestSlugifyUnicodeAndLongTitles(t *testing.T) {
	for _, tc := range []struct{ title, want string }{
		{"Горска тайна", "горска-тайна"},
		{"Морско пътешествие", "морско-пътешествие"},
		{"../Горска/тайна", "горскатайна"},
		{"東京 2026", "東京-2026"},
	} {
		t.Run(tc.title, func(t *testing.T) {
			if got := slugify(tc.title); got != tc.want {
				t.Fatalf("slugify() = %q, want %q", got, tc.want)
			}
		})
	}
	first := slugify(strings.Repeat("дълго", 80) + "а")
	second := slugify(strings.Repeat("дълго", 80) + "б")
	if first == second || len(first) > maxComicSlugLength || !utf8.ValidString(first) {
		t.Fatalf("long slugs must be distinct, bounded UTF-8: %q, %q", first, second)
	}
	short := shortPromptSlugFromText(strings.Repeat("я", 50))
	if len(short) > maxPromptSlugLength || !utf8.ValidString(short) {
		t.Fatalf("short slug must be bounded UTF-8: %q", short)
	}
}

func TestRunnerDistinctCyrillicTitlesPreserveCompletePackages(t *testing.T) {
	root, batch := identityBatch(t)
	images := &identityImageProvider{}
	for _, title := range []string{"Горска тайна", "Морско пътешествие"} {
		runner := identityRunner(t, root, title, "история за "+title, "", images)
		if err := runner.Run(context.Background(), batch); err != nil {
			t.Fatal(err)
		}
		slug := slugify(title)
		for _, name := range []string{comicIdentityFile, slug + "_story.txt", slug + "_cover.png", slug + "_page_1.png", slug + "_back.png"} {
			if _, err := os.Stat(filepath.Join(comicAssetsDir(root, slug), name)); err != nil {
				t.Fatalf("missing %s package artifact: %v", title, err)
			}
		}
		story, err := os.ReadFile(filepath.Join(comicAssetsDir(root, slug), slug+"_story.txt"))
		if err != nil || !strings.Contains(string(story), title) {
			t.Fatalf("story for %s = %q, %v", title, story, err)
		}
		if _, err := os.Stat(filepath.Join(comicsPDFDir(root), slug+".pdf")); err != nil {
			t.Fatal(err)
		}
	}
	if images.calls != 6 {
		t.Fatalf("image calls = %d, want 6", images.calls)
	}
}

func TestRunnerSameSlugRejectsDifferentGenerationBeforeWrites(t *testing.T) {
	for _, forced := range []string{"", "chosen-slug"} {
		t.Run("forced="+forced, func(t *testing.T) {
			root, batch := identityBatch(t)
			images := &identityImageProvider{}
			if err := identityRunner(t, root, "Горска тайна", "първа история", forced, images).Run(context.Background(), batch); err != nil {
				t.Fatal(err)
			}
			slug := resolveComicSlug("Горска тайна", forced)
			before := identitySnapshot(t, comicAssetsDir(root, slug))
			secondTitle := "Горска тайна"
			if forced != "" {
				secondTitle = "Морско пътешествие"
			}
			runner := identityRunner(t, root, secondTitle, "друга история", forced, images)
			err := runner.Run(context.Background(), batch)
			if err == nil || !strings.Contains(err.Error(), "different generation") {
				t.Fatalf("Run() error = %v, want identity mismatch", err)
			}
			if got := identitySnapshot(t, comicAssetsDir(root, slug)); got != before || images.calls != 3 {
				t.Fatalf("rejected run changed original package or generated artwork (calls=%d)", images.calls)
			}
		})
	}
}

func TestRunnerRejectsLegacyAssetsBeforeStoryWrites(t *testing.T) {
	root, batch := identityBatch(t)
	dir, err := NewFSAssetStore().PrepareOutputDirs(root, "chosen")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "chosen_story.txt"), []byte("original story"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := identitySnapshot(t, dir)
	images := &identityImageProvider{}
	err = identityRunner(t, root, "Горска тайна", "нова история", "chosen", images).Run(context.Background(), batch)
	if err == nil || !strings.Contains(err.Error(), "no generation identity") {
		t.Fatalf("Run() error = %v, want legacy identity rejection", err)
	}
	if got := identitySnapshot(t, dir); got != before || images.calls != 0 {
		t.Fatalf("legacy package was modified (image calls=%d)", images.calls)
	}
}

func TestRunnerRejectsLegacyExternalArtifactsBeforeWrites(t *testing.T) {
	for _, artifact := range []string{"PDF/chosen.pdf", "gallery/chosen_gallery_1.png"} {
		for _, emptyAssetsDir := range []bool{false, true} {
			name := fmt.Sprintf("%s/empty-assets-dir=%t", artifact, emptyAssetsDir)
			t.Run(name, func(t *testing.T) {
				root, batch := identityBatch(t)
				path := filepath.Join(comicsRootDir(root), artifact)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("original artifact"), 0o644); err != nil {
					t.Fatal(err)
				}
				assetsDir := comicAssetsDir(root, "chosen")
				if emptyAssetsDir {
					if err := os.MkdirAll(assetsDir, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				images := &identityImageProvider{}
				runner := identityRunner(t, root, "Горска тайна", "нова история", "chosen", images)
				runner.artist.galleryPages = 1
				err := runner.Run(context.Background(), batch)
				if err == nil || !strings.Contains(err.Error(), "no generation identity") {
					t.Fatalf("Run() error = %v, want legacy artifact rejection", err)
				}
				got, err := os.ReadFile(path)
				if err != nil || string(got) != "original artifact" {
					t.Fatalf("existing artifact changed: %q, %v", got, err)
				}
				if images.calls != 0 || identitySnapshot(t, assetsDir) != "" {
					t.Fatalf("rejected run wrote assets (image calls=%d)", images.calls)
				}
			})
		}
	}
}

func TestRunnerMatchingIdentityResumesOnlyMissingArtwork(t *testing.T) {
	root, batch := identityBatch(t)
	images := &identityImageProvider{}
	for i := range 3 {
		if i == 2 {
			if err := os.Remove(filepath.Join(comicAssetsDir(root, "chosen"), "chosen_page_1.png")); err != nil {
				t.Fatal(err)
			}
		}
		runner := identityRunner(t, root, "Горска тайна", "история", "chosen", images)
		runner.artist.galleryPages = 1
		if err := runner.Run(context.Background(), batch); err != nil {
			t.Fatal(err)
		}
		want := 4
		if i == 2 {
			want = 5
		}
		if images.calls != want {
			t.Fatalf("run %d image calls = %d, want %d", i, images.calls, want)
		}
	}
}

func TestClaimComicIdentityRejectsLegacyAndInvalidIdentity(t *testing.T) {
	for _, filename := range []string{"old_cover.png", comicIdentityFile} {
		t.Run(filename, func(t *testing.T) {
			root := t.TempDir()
			dir, err := NewFSAssetStore().EnsureAssetsDir(root, "chosen")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, filename)
			if err = os.WriteFile(path, []byte("original"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err = NewFSAssetStore().ClaimComicIdentity(root, "chosen", "new"); err == nil {
				t.Fatal("expected identity rejection")
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != "original" {
				t.Fatalf("existing artifact changed: %q, %v", got, err)
			}
		})
	}
}

func TestComicIdentityIncludesArtworkInputs(t *testing.T) {
	root, _ := identityBatch(t)
	for _, field := range []string{"bible", "panels", "vocabulary", "style", "aspect-ratio", "page-count", "ultra-realistic"} {
		t.Run(field, func(t *testing.T) {
			r := identityRunner(t, root, "title", "story", "slug", &identityImageProvider{})
			result := GenerateResult{StoryText: "story", Bible: "bible", Title: "title", PanelScript: [][]string{{"panel"}}}
			entries := []WordEntry{{Word: "word"}}
			before, err := r.comicIdentity(result, entries)
			if err != nil {
				t.Fatal(err)
			}
			switch field {
			case "bible":
				result.Bible = "other"
			case "panels":
				result.PanelScript[0][0] = "other"
			case "vocabulary":
				entries[0].Word = "other"
			case "style":
				r.artist.style = "other"
			case "aspect-ratio":
				r.artist.aspectRatio = "2:3"
			case "page-count":
				r.artist.storyPages++
			case "ultra-realistic":
				r.artist.ultraRealistic = !r.artist.ultraRealistic
			}
			after, err := r.comicIdentity(result, entries)
			if err != nil || before == after {
				t.Fatalf("changed %s must change identity: %v", field, err)
			}
		})
	}
}

func identityBatch(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	batch := filepath.Join(root, "vocab.txt")
	if err := os.WriteFile(batch, []byte("ябълка = apple\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, batch
}

func identityRunner(t *testing.T, root, title, story, forced string, images *identityImageProvider) *Runner {
	t.Helper()
	bundle := strings.Join([]string{story, storyBibleSeparator, "библия", storyTitleSeparator, title, storyPanelSeparator, "P1-A: герой"}, "\n")
	r := NewRunner(&RunnerConfig{
		TextProvider:  &scriptedTextProvider{responses: []string{bundle, "описание"}},
		ImageProvider: images, Prompts: fakePromptRenderer{}, OutputDir: root,
		Slug: forced, Style: englishStyleLines(defaultStyles[styleModeComic])[0], StyleMode: styleModeComic, StoryPages: 1, PanelsPerPage: 1,
		PDFAssembler: funcPDFAssembler{fn: func(_ context.Context, dir, slug string, _ []string, _ PDFAssembleOptions) (string, error) {
			path := filepath.Join(dir, slug+".pdf")
			return path, os.WriteFile(path, []byte("pdf"), 0o644)
		}},
	})
	stubRunnerArtistSeams(r)
	return r
}

func identitySnapshot(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot strings.Builder
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&snapshot, "%q:%q\n", entry.Name(), data)
	}
	return snapshot.String()
}
