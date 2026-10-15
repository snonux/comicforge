package stylegrid

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/snonux/comicforge/internal/provider"
)

type tileImageProvider struct {
	outputs []string
	cancel  context.CancelFunc
}

var _ provider.AspectRatioImageProvider = (*tileImageProvider)(nil)

func TestGenerateMontageDeadlineCancelsConversion(t *testing.T) {
	for _, stage := range []string{"normalize", "caption"} {
		t.Run(stage, func(t *testing.T) {
			logPath, pidPath := installMontageCommands(t, stage)
			img := &tileImageProvider{}
			outPath := filepath.Join(t.TempDir(), "montage.png")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			start := time.Now()
			err := GenerateMontage(ctx, img, "16:9", []Job{{Header: "first"}, {Header: "second"}}, outPath, defaultMontageOptions(2))
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("GenerateMontage() error = %v, want deadline exceeded", err)
			}
			if elapsed := time.Since(start); elapsed > 3*time.Second {
				t.Fatalf("GenerateMontage() took %s; converter was not interrupted promptly", elapsed)
			}
			wantLog := "normalize\n"
			if stage == "caption" {
				wantLog += "caption\n"
			}
			assertCommandLog(t, logPath, wantLog)
			assertTileCleanup(t, img, 1)
			if _, err := os.Stat(outPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("montage output exists or stat failed: %v", err)
			}
			assertConverterStopped(t, pidPath)
		})
	}
}

func TestGenerateMontageCanceledBeforeConversion(t *testing.T) {
	for _, beforeJob := range []bool{true, false} {
		name := "after_generation"
		if beforeJob {
			name = "before_generation"
		}
		t.Run(name, func(t *testing.T) {
			logPath, _ := installMontageCommands(t, "")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			img := &tileImageProvider{cancel: cancel}
			wantJobs := 1
			if beforeJob {
				cancel()
				wantJobs = 0
			}
			err := GenerateMontage(ctx, img, "16:9", []Job{{Header: "first"}, {Header: "second"}}, filepath.Join(t.TempDir(), "montage.png"), defaultMontageOptions(2))
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("GenerateMontage() error = %v, want canceled", err)
			}
			assertCommandLog(t, logPath, "")
			assertTileCleanup(t, img, wantJobs)
		})
	}
}

func TestGenerateMontageCompletesAllStages(t *testing.T) {
	logPath, _ := installMontageCommands(t, "")
	img := &tileImageProvider{}
	outPath := filepath.Join(t.TempDir(), "montage.png")
	err := GenerateMontage(context.Background(), img, "16:9", []Job{{Header: "first"}, {Header: "second"}}, outPath, defaultMontageOptions(2))
	if err != nil {
		t.Fatalf("GenerateMontage(): %v", err)
	}
	assertCommandLog(t, logPath, "normalize\ncaption\nnormalize\ncaption\nmontage\n")
	assertTileCleanup(t, img, 2)
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("montage output missing: %v", err)
	}
}

func (p *tileImageProvider) Name() string { return "tile-test" }

func (p *tileImageProvider) IsAvailable() error { return nil }

func (p *tileImageProvider) GenerateImage(ctx context.Context, prompt, outputFile string) error {
	return p.GenerateImageWithAspectRatio(ctx, prompt, outputFile, "16:9")
}

func (p *tileImageProvider) GenerateImageWithAspectRatio(_ context.Context, _, outputFile, _ string) error {
	p.outputs = append(p.outputs, outputFile)
	if err := os.WriteFile(outputFile, []byte("raw tile"), 0o600); err != nil {
		return err
	}
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func installMontageCommands(t *testing.T, blockingStage string) (string, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fixtures require a POSIX shell")
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "commands.log")
	pidPath := filepath.Join(dir, "converter.pid")
	// exec replaces the shell with sleep, so cancellation tests the converter
	// process itself without leaving a child holding its output pipe open.
	script := `#!/bin/sh
stage=normalize
case "$0" in *montage) stage=montage ;; esac
if [ "$1" = "(" ]; then stage=caption; fi
printf '%s\n' "$stage" >> "$STYLEGRID_COMMAND_LOG"
for output do :; done
printf 'partial output' > "$output"
if [ "$stage" = "$STYLEGRID_BLOCK_STAGE" ]; then
    printf '%s\n' "$$" > "$STYLEGRID_CONVERTER_PID"
    exec sleep 10
fi
`
	for _, command := range []string{"convert", "montage"} {
		if err := os.WriteFile(filepath.Join(dir, command), []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("STYLEGRID_COMMAND_LOG", logPath)
	t.Setenv("STYLEGRID_CONVERTER_PID", pidPath)
	t.Setenv("STYLEGRID_BLOCK_STAGE", blockingStage)
	return logPath, pidPath
}

func assertCommandLog(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if want == "" && errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("command log = %q, want %q", got, want)
	}
}

func assertTileCleanup(t *testing.T, img *tileImageProvider, wantJobs int) {
	t.Helper()
	if len(img.outputs) != wantJobs {
		t.Fatalf("generated %d jobs, want %d", len(img.outputs), wantJobs)
	}
	for _, output := range img.outputs {
		if _, err := os.Stat(filepath.Dir(output)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("temporary tiles remain or stat failed: %v", err)
		}
	}
}

func assertConverterStopped(t *testing.T, pidPath string) {
	t.Helper()
	pid, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("blocking converter did not start: %v", err)
	}
	cmd := exec.Command("sh", "-c", `kill -0 "$1" 2>/dev/null`, "sh", strings.TrimSpace(string(pid)))
	if err := cmd.Run(); err == nil {
		t.Fatalf("converter process %s still exists", strings.TrimSpace(string(pid)))
	} else {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("check converter process: %v", err)
		}
	}
}
