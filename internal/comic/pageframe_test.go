package comic

import "testing"

func TestDescribePageFrame(t *testing.T) {
	tests := []struct {
		ratio string
		want  string
	}{
		{"16:9", "landscape 16:9 format (widescreen)"},
		{"2:3", "portrait 2:3 format (tall comic book page proportions)"},
		{"3:2", "landscape 3:2 format (widescreen)"},
		{"1:1", "square 1:1 format"},
		{"", "landscape 16:9 format (widescreen)"},
		{"bogus", "bogus format"},
	}
	for _, tt := range tests {
		if got := DescribePageFrame(tt.ratio); got != tt.want {
			t.Fatalf("DescribePageFrame(%q) = %q, want %q", tt.ratio, got, tt.want)
		}
	}
}
