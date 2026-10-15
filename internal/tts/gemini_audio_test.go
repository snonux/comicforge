package tts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/genai"

	"github.com/snonux/comicforge/internal/ffmpegutil"
)

// TestMain also lets the test executable act as a controlled ffmpeg process.
// Handle its arguments before testing parses ffmpeg's flags.
func TestMain(m *testing.M) {
	if mode := os.Getenv("COMICFORGE_TTS_TEST_PROCESS"); mode != "" {
		if err := runPCMTestProcess(mode); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runPCMTestProcess(mode string) error {
	args, err := json.Marshal(os.Args[1:])
	if err != nil {
		return err
	}
	if err := os.WriteFile(os.Getenv("COMICFORGE_TTS_TEST_STARTED"), args, 0o600); err != nil {
		return err
	}
	if mode == "fail" {
		return errors.New("controlled encoder failure")
	}
	// Bounded even if the implementation stops honoring cancellation.
	time.Sleep(10 * time.Second)
	return nil
}

func TestGenerateAudioCancellationAfterAPIResponse(t *testing.T) {
	for _, mimeType := range []string{"audio/l16;rate=24000", "audio/mpeg"} {
		t.Run(mimeType, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stubAudioResponse(t, func() { cancel() }, mimeType)
			dir := t.TempDir()
			err := (&GeminiProvider{model: DefaultModel}).GenerateAudio(ctx, "hello", filepath.Join(dir, "out.mp3"))
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("GenerateAudio() error = %v, want context.Canceled", err)
			}
			assertNoAudioFiles(t, dir)
		})
	}
}

func TestGenerateAudioCancelsRunningPCMConversion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		timeout time.Duration
		want    error
	}{
		{"cancel", 5 * time.Second, context.Canceled},
		{"deadline", 2 * time.Second, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := stubPCMProcess(t, "block")
			stubAudioResponse(t, nil, "audio/l16;rate=16000")
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			dir := t.TempDir()
			out := filepath.Join(dir, "out.mp3")
			done := make(chan error, 1)
			go func() {
				done <- (&GeminiProvider{model: DefaultModel}).GenerateAudio(ctx, "hello", out)
			}()

			args := waitForPCMProcess(t, started, done)
			wantPrefix := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-f", "s16le", "-ar", "16000", "-ac", "1", "-i"}
			if len(args) != 18 || !reflect.DeepEqual(args[:12], wantPrefix) || args[17] != out {
				t.Fatalf("ffmpeg arguments = %q", args)
			}
			if data, err := os.ReadFile(args[12]); err != nil || len(data) != 4800 {
				t.Fatalf("temporary PCM: bytes = %d, error = %v", len(data), err)
			}
			if errors.Is(tc.want, context.Canceled) {
				cancel()
			}
			select {
			case err := <-done:
				if !errors.Is(err, tc.want) {
					t.Fatalf("GenerateAudio() error = %v, want %v", err, tc.want)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("PCM conversion did not stop after cancellation")
			}
			assertNoAudioFiles(t, dir)
		})
	}
}

func TestEncodePCMToMP3ConversionFailureCleansTemporaryPCM(t *testing.T) {
	stubPCMProcess(t, "fail")
	dir := t.TempDir()
	err := encodePCMToMP3(context.Background(), []byte{0, 0}, "", filepath.Join(dir, "out.mp3"))
	if err == nil || !strings.Contains(err.Error(), "controlled encoder failure") {
		t.Fatalf("encodePCMToMP3() error = %v, want subprocess diagnostic", err)
	}
	assertNoAudioFiles(t, dir)
}

func TestWriteAudioFileMP3AndErrors(t *testing.T) {
	// Error cases must reach the filesystem even on hosts without ffmpeg.
	t.Cleanup(ffmpegutil.OverrideLookPath(func(string) (string, error) { return "unused-ffmpeg", nil }))
	for _, mimeType := range []string{"audio/mpeg", " AUDIO/MP3 "} {
		t.Run(mimeType, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out.mp3")
			if err := writeAudioFile(context.Background(), []byte("ID3audio"), mimeType, out); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(out); err != nil || string(got) != "ID3audio" {
				t.Fatalf("MP3 data = %q, error = %v", got, err)
			}
		})
	}
	for _, tc := range []struct {
		name, mimeType, output, want string
	}{
		{"empty output", "audio/mpeg", "", "output file is required"},
		{"unsupported", "audio/ogg", filepath.Join(t.TempDir(), "out.mp3"), "unsupported audio mime type"},
		{"write failure", "audio/mpeg", t.TempDir(), "write audio"},
		{"temporary file failure", "audio/l16", filepath.Join(t.TempDir(), "missing", "out.mp3"), "create temporary audio file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := writeAudioFile(context.Background(), []byte("audio"), tc.mimeType, tc.output)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("writeAudioFile() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestWriteAudioFilePCMConversion(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	for _, mimeType := range []string{"audio/l16;rate=16000", "audio/raw;codec=pcm;rate=24000", ""} {
		t.Run(mimeType, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "out.mp3")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := writeAudioFile(ctx, make([]byte, 4800), mimeType, out); err != nil {
				t.Fatal(err)
			}
			if data, err := os.ReadFile(out); err != nil || len(data) == 0 {
				t.Fatalf("MP3 bytes = %d, error = %v", len(data), err)
			}
			files, err := os.ReadDir(dir)
			if err != nil || len(files) != 1 || files[0].Name() != "out.mp3" {
				t.Fatalf("conversion output files = %v, error = %v", files, err)
			}
		})
	}
}

func TestEncodePCMToMP3ExpiredDeadline(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	dir := t.TempDir()
	err := encodePCMToMP3(ctx, []byte{0, 0}, "", filepath.Join(dir, "out.mp3"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("encodePCMToMP3() error = %v, want context.DeadlineExceeded", err)
	}
	assertNoAudioFiles(t, dir)
}

func stubAudioResponse(t *testing.T, beforeReturn func(), mimeType string) {
	t.Helper()
	original := geminiGenerateContent
	t.Cleanup(func() { geminiGenerateContent = original })
	geminiGenerateContent = func(context.Context, *genai.Client, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		if beforeReturn != nil {
			beforeReturn()
		}
		return &genai.GenerateContentResponse{Candidates: []*genai.Candidate{{Content: &genai.Content{
			Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: mimeType, Data: make([]byte, 4800)}}},
		}}}}, nil
	}
}

func stubPCMProcess(t *testing.T, mode string) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	started := filepath.Join(t.TempDir(), "started.json")
	t.Setenv("COMICFORGE_TTS_TEST_PROCESS", mode)
	t.Setenv("COMICFORGE_TTS_TEST_STARTED", started)
	t.Cleanup(ffmpegutil.OverrideLookPath(func(string) (string, error) { return executable, nil }))
	return started
}

func waitForPCMProcess(t *testing.T, started string, done <-chan error) []string {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case <-ticker.C:
			data, err := os.ReadFile(started)
			var args []string
			if err == nil && json.Unmarshal(data, &args) == nil {
				return args
			}
		case err := <-done:
			t.Fatalf("conversion exited before cancellation: %v", err)
		case <-timeout.C:
			t.Fatal("conversion subprocess did not start")
		}
	}
}

func assertNoAudioFiles(t *testing.T, dir string) {
	t.Helper()
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 0 {
		t.Fatalf("audio files after cancellation/failure = %v, error = %v", files, err)
	}
}
