package stylegrid

import (
	"testing"

	"github.com/snonux/comicforge/internal/config"
)

func TestBuildJobsDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	jobs := BuildJobs(cfg, "test scene")
	// 12 families × 2 lines + 3 genres = 27
	if want := 27; len(jobs) != want {
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
	cfg.Styles.Comic = []string{"", "  ", "only-valid"}
	cfg.Story.Genres = nil
	jobs := BuildJobs(cfg, "x")
	// 1 comic line + 22 from other families + 0 themes
	if want := 23; len(jobs) != want {
		t.Fatalf("len(jobs) = %d, want %d", len(jobs), want)
	}
}
