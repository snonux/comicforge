package stylegrid

import (
	"strings"
	"testing"

	"codeberg.org/snonux/comicforge/internal/config"
)

func TestBuildJobsDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	jobs := BuildJobs(cfg, "test scene")
	// 12 families × catalog lines (34) + 3 genres = 37
	if want := 37; len(jobs) != want {
		t.Fatalf("len(jobs) = %d, want %d", len(jobs), want)
	}
	if jobs[0].Kind != "style" || jobs[0].Header == "" || jobs[0].Prompt == "" {
		t.Fatalf("first job = %#v", jobs[0])
	}
	last := jobs[len(jobs)-1]
	if last.Kind != "theme" {
		t.Fatalf("last job kind = %q, want theme", last.Kind)
	}
}

func TestBuildJobsSkipsEmptyStrings(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Styles["comic"] = []string{"", "  ", "only-valid"}
	cfg.Story.Genres = nil
	jobs := BuildJobs(cfg, "x")
	// 1 comic line + 28 from other families + 0 themes
	if want := 29; len(jobs) != want {
		t.Fatalf("len(jobs) = %d, want %d", len(jobs), want)
	}
}

func TestBuildJobsIncludesCustomCategory(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Styles["neon_noir"] = []string{"neon glow noir"}
	cfg.Story.Genres = nil
	jobs := BuildJobs(cfg, "x")
	found := false
	for _, job := range jobs {
		if strings.Contains(job.Header, "Neon Noir") && strings.Contains(job.Prompt, "neon glow noir") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("BuildJobs missing custom neon_noir category")
	}
}
