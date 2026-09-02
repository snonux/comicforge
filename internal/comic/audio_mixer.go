package comic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/snonux/comicforge/internal/ffmpegutil"
)

// AudioMixer performs ffmpeg-backed narration post-processing.
// Defined at the point of use so Narrator can be unit-tested without a real binary.
type AudioMixer interface {
	MixAmbientMusic(ctx context.Context, narrationFile, outputFile, tmpDir string) error
	ConcatenateMP3s(ctx context.Context, chunkPaths []string, outputFile, tmpDir string) error
	ConvertToStereo(ctx context.Context, inputFile, outputFile string) error
}

// FFmpegAudioMixer is the exec.CommandContext adapter for AudioMixer.
type FFmpegAudioMixer struct{}

var _ AudioMixer = (*FFmpegAudioMixer)(nil)

// NewFFmpegAudioMixer returns the default ffmpeg-backed mixer.
func NewFFmpegAudioMixer() *FFmpegAudioMixer {
	return &FFmpegAudioMixer{}
}

// MixAmbientMusic overlays a generated ambient pad under narration.
func (m *FFmpegAudioMixer) MixAmbientMusic(ctx context.Context, narrationFile, outputFile, tmpDir string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ffmpegPath, err := ffmpegutil.LookPath()
	if err != nil {
		return fmt.Errorf("ffmpeg not found")
	}

	musicPath := filepath.Join(tmpDir, "ambient_pad.mp3")
	if err = m.generateAmbientPad(ctx, ffmpegPath, musicPath); err != nil {
		return err
	}

	cmd := ffmpegutil.CommandContext(ctx, ffmpegPath,
		"-y",
		"-i", narrationFile,
		"-i", musicPath,
		"-filter_complex", "[0:a][1:a]amix=inputs=2:weights=1 0.3:duration=first[aout]",
		"-map", "[aout]",
		"-ac", "2",
		"-codec:a", "libmp3lame", "-q:a", "2",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("background music mix failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ConcatenateMP3s joins chunk files into a single MP3 via the ffmpeg concat demuxer.
func (m *FFmpegAudioMixer) ConcatenateMP3s(ctx context.Context, chunkPaths []string, outputFile, tmpDir string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ffmpegPath, err := ffmpegutil.LookPath()
	if err != nil {
		return fmt.Errorf("ffmpeg not found — required for multi-chunk narration: %w", err)
	}

	listPath := filepath.Join(tmpDir, "concat_list.txt")
	var sb strings.Builder
	for _, path := range chunkPaths {
		fmt.Fprintf(&sb, "file '%s'\n", path)
	}
	if err = os.WriteFile(listPath, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("write concat list: %w", err)
	}

	cmd := ffmpegutil.CommandContext(ctx, ffmpegPath,
		"-y",
		"-f", "concat", "-safe", "0",
		"-i", listPath,
		"-codec:a", "copy",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg concat failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ConvertToStereo re-encodes input as stereo MP3. When ffmpeg is missing, copies the file as-is.
func (m *FFmpegAudioMixer) ConvertToStereo(ctx context.Context, inputFile, outputFile string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ffmpegPath, err := ffmpegutil.LookPath()
	if err != nil {
		fmt.Println("    Warning: ffmpeg not found, narration will be mono")
		return copyFile(inputFile, outputFile)
	}

	cmd := ffmpegutil.CommandContext(ctx, ffmpegPath,
		"-y",
		"-i", inputFile,
		"-ac", "2",
		"-codec:a", "libmp3lame", "-q:a", "2",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg stereo conversion failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (m *FFmpegAudioMixer) generateAmbientPad(ctx context.Context, ffmpegPath, outputFile string) error {
	droneExpr := "0.04*sin(65*2*PI*t)+0.03*sin(98*2*PI*t)+0.02*sin(130*2*PI*t)"
	cmd := ffmpegutil.CommandContext(ctx, ffmpegPath,
		"-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("aevalsrc=%s:sample_rate=44100", droneExpr),
		"-f", "lavfi", "-i", "anoisesrc=color=pink:amplitude=0.008",
		"-filter_complex", "[0:a][1:a]amix=inputs=2:duration=first[mixed];[mixed]afade=t=in:st=0:d=4[aout]",
		"-map", "[aout]",
		"-t", "300",
		"-ac", "2",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ambient pad generation failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
