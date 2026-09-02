package comic

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/snonux/comicforge/internal/ffmpegutil"
)

func TestImageMagickPDFAssemblerMissingConvert(t *testing.T) {
	t.Parallel()

	a := NewImageMagickPDFAssembler()
	a.lookPath = func(string) (string, error) {
		return "", errors.New("not in PATH")
	}

	_, err := a.Assemble(t.TempDir(), "title", []string{"page.png"}, PDFAssembleOptions{})
	if err == nil {
		t.Fatal("Assemble() error = nil, want missing convert")
	}
	if !strings.Contains(err.Error(), "convert' not found") {
		t.Fatalf("Assemble() error = %v, want convert not found", err)
	}
}

func TestImageMagickPDFAssemblerNoImages(t *testing.T) {
	t.Parallel()

	_, err := NewImageMagickPDFAssembler().Assemble(t.TempDir(), "title", nil, PDFAssembleOptions{})
	if err == nil {
		t.Fatal("Assemble() error = nil, want no images")
	}
	if !strings.Contains(err.Error(), "no comic images") {
		t.Fatalf("Assemble() error = %v, want no comic images", err)
	}
}

func TestFFmpegAudioMixerConcatenateMissingBinary(t *testing.T) {
	restore := ffmpegutil.OverrideLookPath(func(string) (string, error) {
		return "", errors.New("missing")
	})
	t.Cleanup(restore)

	err := NewFFmpegAudioMixer().ConcatenateMP3s(
		[]string{"a.mp3", "b.mp3"},
		filepath.Join(t.TempDir(), "out.mp3"),
		t.TempDir(),
	)
	if err == nil {
		t.Fatal("ConcatenateMP3s() error = nil, want missing ffmpeg")
	}
	if !strings.Contains(err.Error(), "ffmpeg not found") {
		t.Fatalf("ConcatenateMP3s() error = %v, want ffmpeg not found", err)
	}
}

func TestFFmpegAudioMixerMixAmbientMissingBinary(t *testing.T) {
	restore := ffmpegutil.OverrideLookPath(func(string) (string, error) {
		return "", errors.New("missing")
	})
	t.Cleanup(restore)

	err := NewFFmpegAudioMixer().MixAmbientMusic("narration.mp3", "out.mp3", t.TempDir())
	if err == nil {
		t.Fatal("MixAmbientMusic() error = nil, want missing ffmpeg")
	}
	if !strings.Contains(err.Error(), "ffmpeg not found") {
		t.Fatalf("MixAmbientMusic() error = %v, want ffmpeg not found", err)
	}
}

func TestNewNarratorDefaultsAudioMixer(t *testing.T) {
	t.Parallel()

	n := NewNarrator(&NarratorConfig{
		MainProvider: fakeTTSProvider{},
		Prompts:      fakePromptRenderer{},
	})
	if n.mixer == nil {
		t.Fatal("mixer = nil, want default FFmpegAudioMixer")
	}
	if _, ok := n.mixer.(*FFmpegAudioMixer); !ok {
		t.Fatalf("mixer type = %T, want *FFmpegAudioMixer", n.mixer)
	}
}

func TestNewRunnerDefaultsPDFAssembler(t *testing.T) {
	t.Parallel()

	r := NewRunner(&RunnerConfig{})
	if r.pdfAssembler == nil {
		t.Fatal("pdfAssembler = nil, want default ImageMagickPDFAssembler")
	}
	if _, ok := r.pdfAssembler.(*ImageMagickPDFAssembler); !ok {
		t.Fatalf("pdfAssembler type = %T, want *ImageMagickPDFAssembler", r.pdfAssembler)
	}
}

func TestNewRunnerUsesInjectedPDFAssembler(t *testing.T) {
	t.Parallel()

	fake := funcPDFAssembler{fn: func(string, string, []string, PDFAssembleOptions) (string, error) {
		return "injected.pdf", nil
	}}
	r := NewRunner(&RunnerConfig{PDFAssembler: fake})
	got, err := r.pdfAssembler.Assemble("", "", []string{"x.png"}, PDFAssembleOptions{})
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if got != "injected.pdf" {
		t.Fatalf("Assemble() = %q, want injected.pdf", got)
	}
}
