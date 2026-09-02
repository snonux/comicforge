package comic

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PDFAssembler builds a comic PDF from page images.
// Defined at the point of use so Runner can be unit-tested without ImageMagick.
type PDFAssembler interface {
	Assemble(outputDir, titleSlug string, imagePaths []string, opts PDFAssembleOptions) (string, error)
}

// ImageMagickPDFAssembler is the exec.Command adapter for PDFAssembler.
type ImageMagickPDFAssembler struct {
	lookPath func(file string) (string, error)
	command  func(name string, arg ...string) *exec.Cmd
}

var _ PDFAssembler = (*ImageMagickPDFAssembler)(nil)

// NewImageMagickPDFAssembler returns the default ImageMagick-backed assembler.
func NewImageMagickPDFAssembler() *ImageMagickPDFAssembler {
	return &ImageMagickPDFAssembler{
		lookPath: exec.LookPath,
		command:  exec.Command,
	}
}

// AssembleComicPDF combines comic pages into a PDF using the default ImageMagick assembler.
func AssembleComicPDF(outputDir, titleSlug string, imagePaths []string, opts PDFAssembleOptions) (string, error) {
	return NewImageMagickPDFAssembler().Assemble(outputDir, titleSlug, imagePaths, opts)
}

// Assemble combines comic pages into a PDF using ImageMagick convert.
func (a *ImageMagickPDFAssembler) Assemble(outputDir, titleSlug string, imagePaths []string, opts PDFAssembleOptions) (string, error) {
	if len(imagePaths) == 0 {
		return "", fmt.Errorf("no comic images to assemble into PDF")
	}
	if _, err := a.resolveConvert(); err != nil {
		return "", fmt.Errorf("ImageMagick 'convert' not found — install ImageMagick to generate the PDF")
	}

	opts = normalizePDFAssembleOptions(opts)
	pdfPath := filepath.Join(outputDir, titleSlug+".pdf")

	inputs := imagePaths
	if opts.Presentation == pdfPresentationPrint || opts.Presentation == pdfPresentationBook {
		tmpDir, err := os.MkdirTemp("", "comicforge-pdf-*")
		if err != nil {
			return "", fmt.Errorf("temp dir for pdf frames: %w", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		inputs = make([]string, len(imagePaths))
		for i, src := range imagePaths {
			dst := filepath.Join(tmpDir, fmt.Sprintf("frame_%04d.png", i))
			var ferr error
			switch opts.Presentation {
			case pdfPresentationPrint:
				ferr = a.renderPrintFrame(src, dst)
			case pdfPresentationBook:
				ferr = a.renderBookFrame(src, dst, tmpDir, i)
			}
			if ferr != nil {
				return "", ferr
			}
			inputs[i] = dst
		}

		fitBG := printMatteColor
		if opts.Presentation == pdfPresentationBook {
			fitBG = bookDeskColor
		}
		fitted := make([]string, len(inputs))
		for i, p := range inputs {
			out := filepath.Join(tmpDir, fmt.Sprintf("din_a4_%04d.png", i))
			if err := a.fitRasterPageToDINA4(p, out, opts.Density, fitBG); err != nil {
				return "", err
			}
			fitted[i] = out
		}
		inputs = fitted
	}

	args := []string{"-density", fmt.Sprintf("%d", opts.Density)}
	args = append(args, inputs...)
	if opts.JPEGQuality > 0 {
		args = append(args, "-compress", "JPEG", "-quality", fmt.Sprintf("%d", opts.JPEGQuality))
	}
	args = append(args, pdfPath)

	out, err := a.runConvert(args...)
	if err != nil {
		return "", fmt.Errorf("convert failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return pdfPath, nil
}

func (a *ImageMagickPDFAssembler) resolveConvert() (string, error) {
	lp := a.lookPath
	if lp == nil {
		lp = exec.LookPath
	}
	return lp("convert")
}

func (a *ImageMagickPDFAssembler) runConvert(args ...string) ([]byte, error) {
	path, err := a.resolveConvert()
	if err != nil {
		return nil, err
	}
	cmdFn := a.command
	if cmdFn == nil {
		cmdFn = exec.Command
	}
	return cmdFn(path, args...).CombinedOutput()
}

func (a *ImageMagickPDFAssembler) renderPrintFrame(src, dst string) error {
	args := []string{
		src,
		"-background", printMatteColor,
		"-gravity", "center",
		"-extent", "108%x108%",
		dst,
	}
	out, err := a.runConvert(args...)
	if err != nil {
		return fmt.Errorf("convert matte frame: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (a *ImageMagickPDFAssembler) renderBookAgedPage(src, dst string) error {
	args := []string{
		src,
		"-modulate", bookAgedModulate,
		"-sepia-tone", bookAgedSepiaPercent,
		"-attenuate", bookAgedNoiseAtten,
		"+noise", "Gaussian",
		"(", "+clone", "-colorspace", "gray", "-blur", bookAgedVignetteBlur, "+level", bookAgedVignetteLevel, ")",
		"-compose", "multiply",
		"-composite",
		dst,
	}
	out, err := a.runConvert(args...)
	if err != nil {
		return fmt.Errorf("convert book aged page: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (a *ImageMagickPDFAssembler) renderBookFrame(src, dst, tmpDir string, pageIndex int) error {
	aged := filepath.Join(tmpDir, fmt.Sprintf("book_aged_%04d.png", pageIndex))
	if err := a.renderBookAgedPage(src, aged); err != nil {
		return err
	}

	step1 := filepath.Join(tmpDir, fmt.Sprintf("book_step1_%04d.png", pageIndex))
	angle := 2.1
	if pageIndex%2 == 1 {
		angle = -2.1
	}
	args := []string{
		aged,
		"-background", printMatteColor,
		"-gravity", "center",
		"-extent", "125%x125%",
		"-background", printMatteColor,
		"-rotate", fmt.Sprintf("%.1f", angle),
		"(", "+clone", "-background", "black", "-shadow", "78x5+14+22", ")",
		"+swap",
		"-background", bookDeskColor,
		"-layers", "merge",
		"+repage",
		step1,
	}
	out, err := a.runConvert(args...)
	if err != nil {
		return fmt.Errorf("convert book frame (tilt/shadow): %w\n%s", err, strings.TrimSpace(string(out)))
	}

	h, err := imageHeight(step1)
	if err != nil {
		return fmt.Errorf("book frame dimensions: %w", err)
	}
	if h < 1 {
		h = 1
	}
	grad := filepath.Join(tmpDir, fmt.Sprintf("book_grad_%04d.png", pageIndex))
	out, err = a.runConvert("-size", fmt.Sprintf("12x%d", h), bookEdgeGradient, grad)
	if err != nil {
		return fmt.Errorf("convert book edge gradient: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	out, err = a.runConvert(step1, grad, "-gravity", "east", "-compose", "over", "-composite", dst)
	if err != nil {
		return fmt.Errorf("convert book frame (edge): %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (a *ImageMagickPDFAssembler) fitRasterPageToDINA4(src, dst string, density int, background string) error {
	w, h := dinAPortraitPixels(density)
	if strings.TrimSpace(background) == "" {
		background = printMatteColor
	}
	args := []string{
		src,
		"-resize", fmt.Sprintf("%dx%d", w, h),
		"-background", background,
		"-gravity", "center",
		"-extent", fmt.Sprintf("%dx%d", w, h),
		dst,
	}
	out, err := a.runConvert(args...)
	if err != nil {
		return fmt.Errorf("convert DIN A4 page: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
