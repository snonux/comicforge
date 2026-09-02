package ffmpegutil

import (
	"errors"
	"strings"
	"testing"
)

func TestLookPathMissingBinary(t *testing.T) {
	restore := OverrideLookPath(func(string) (string, error) {
		return "", errors.New("not in PATH")
	})
	t.Cleanup(restore)

	_, err := LookPath()
	if err == nil {
		t.Fatal("LookPath() error = nil, want missing binary")
	}
	if !strings.Contains(err.Error(), "ffmpeg not found") {
		t.Fatalf("LookPath() error = %v, want ffmpeg not found wrapper", err)
	}
}

func TestLookPathSuccess(t *testing.T) {
	restore := OverrideLookPath(func(name string) (string, error) {
		if name != "ffmpeg" {
			t.Fatalf("lookPath(%q), want ffmpeg", name)
		}
		return "/usr/bin/ffmpeg", nil
	})
	t.Cleanup(restore)

	got, err := LookPath()
	if err != nil {
		t.Fatalf("LookPath() error = %v", err)
	}
	if got != "/usr/bin/ffmpeg" {
		t.Fatalf("LookPath() = %q, want /usr/bin/ffmpeg", got)
	}
}

func TestCommandPrependsQuietFlags(t *testing.T) {
	cmd := Command("/usr/bin/ffmpeg", "-y", "-i", "in.mp3", "out.mp3")
	if cmd.Path != "/usr/bin/ffmpeg" {
		t.Fatalf("Path = %q, want /usr/bin/ffmpeg", cmd.Path)
	}
	want := append(append([]string{}, QuietFlags...), "-y", "-i", "in.mp3", "out.mp3")
	if len(cmd.Args) != len(want)+1 { // Args[0] is the program name
		t.Fatalf("Args len = %d, want %d (incl. program)", len(cmd.Args), len(want)+1)
	}
	for i, arg := range want {
		if cmd.Args[i+1] != arg {
			t.Fatalf("Args[%d] = %q, want %q (full Args=%v)", i+1, cmd.Args[i+1], arg, cmd.Args)
		}
	}
}
