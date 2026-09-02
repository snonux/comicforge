// Package ffmpegutil centralizes ffmpeg binary lookup and the quiet CLI flags
// shared by the comic narration mixer and TTS PCM encoder.
package ffmpegutil

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
)

// QuietFlags are the non-interactive, low-noise ffmpeg flags used across the project.
var QuietFlags = []string{"-nostdin", "-hide_banner", "-loglevel", "error"}

var (
	lookPath atomic.Value // func(string) (string, error)
	// overrideMu serializes OverrideLookPath so parallel packages cannot interleave stubs.
	overrideMu sync.Mutex
)

func init() {
	lookPath.Store(exec.LookPath)
}

func currentLookPath() func(string) (string, error) {
	return lookPath.Load().(func(string) (string, error))
}

// LookPath resolves the ffmpeg binary on PATH.
func LookPath() (string, error) {
	path, err := currentLookPath()("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found: %w", err)
	}
	return path, nil
}

// Command builds an ffmpeg command with QuietFlags prepended to args.
// ffmpegPath must already be resolved (typically via LookPath).
func Command(ffmpegPath string, args ...string) *exec.Cmd {
	return CommandContext(context.Background(), ffmpegPath, args...)
}

// CommandContext builds an ffmpeg command bound to ctx, with QuietFlags prepended to args.
// ffmpegPath must already be resolved (typically via LookPath).
func CommandContext(ctx context.Context, ffmpegPath string, args ...string) *exec.Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	full := make([]string, 0, len(QuietFlags)+len(args))
	full = append(full, QuietFlags...)
	full = append(full, args...)
	return exec.CommandContext(ctx, ffmpegPath, full...)
}

// OverrideLookPath replaces binary lookup for tests. Call the returned restore func in cleanup.
// Concurrent overrides are serialized so parallel packages cannot race on the stub.
func OverrideLookPath(fn func(string) (string, error)) (restore func()) {
	overrideMu.Lock()
	prev := currentLookPath()
	lookPath.Store(fn)
	return func() {
		lookPath.Store(prev)
		overrideMu.Unlock()
	}
}
