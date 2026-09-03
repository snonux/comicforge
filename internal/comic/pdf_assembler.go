package comic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PDFAssembler builds a comic PDF from page images.
// Defined at the point of use so Runner can be unit-tested without ImageMagick.
type PDFAssembler interface {
	Assemble(ctx context.Context, outputDir, titleSlug string, imagePaths []string, opts PDFAssembleOptions) (string, error)
}

// ImageMagickPDFAssembler is the exec.CommandContext adapter for PDFAssembler.
type ImageMagickPDFAssembler struct {
	lookPath func(file string) (string, error)
	command  func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

var _ PDFAssembler = (*ImageMagickPDFAssembler)(nil)

// NewImageMagickPDFAssembler returns the default ImageMagick-backed assembler.
func NewImageMagickPDFAssembler() *ImageMagickPDFAssembler {
	return &ImageMagickPDFAssembler{
		lookPath: exec.LookPath,
		command:  exec.CommandContext,
	}
}

// AssembleComicPDF combines comic pages into a PDF using the default ImageMagick assembler.
func AssembleComicPDF(ctx context.Context, outputDir, titleSlug string, imagePaths []string, opts PDFAssembleOptions) (string, error) {
	return NewImageMagickPDFAssembler().Assemble(ctx, outputDir, titleSlug, imagePaths, opts)
}

// Assemble combines comic pages into a PDF using ImageMagick convert.
func (a *ImageMagickPDFAssembler) Assemble(ctx context.Context, outputDir, titleSlug string, imagePaths []string, opts PDFAssembleOptions) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(imagePaths) == 0 {
		return "", fmt.Errorf("no comic images to assemble into PDF")
	}
	if _, err := a.resolveConvert(); err != nil {
		return "", fmt.Errorf("ImageMagick 'convert' not found — install ImageMagick to generate the PDF")
	}

	opts = normalizePDFAssembleOptions(opts)
	pdfPath := filepath.Join(outputDir, titleSlug+".pdf")

	inputs, cleanup, err := a.preparePresentationInputs(ctx, imagePaths, opts)
	if err != nil {
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}

	args := []string{"-density", fmt.Sprintf("%d", opts.Density)}
	args = append(args, inputs...)
	if opts.JPEGQuality > 0 {
		args = append(args, "-compress", "JPEG", "-quality", fmt.Sprintf("%d", opts.JPEGQuality))
	}
	args = append(args, pdfPath)

	out, err := a.runConvert(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("convert failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return pdfPath, nil
}

// preparePresentationInputs frames/fits pages for print or book presentation.
// cleanup removes any temp dir; callers must defer it when non-nil.
func (a *ImageMagickPDFAssembler) preparePresentationInputs(ctx context.Context, imagePaths []string, opts PDFAssembleOptions) ([]string, func(), error) {
	if opts.Presentation != pdfPresentationPrint && opts.Presentation != pdfPresentationBook {
		return imagePaths, nil, nil
	}

	tmpDir, err := os.MkdirTemp("", "comicforge-pdf-*")
	if err != nil {
		return nil, nil, fmt.Errorf("temp dir for pdf frames: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	framed, err := a.renderPresentationFrames(ctx, imagePaths, opts, tmpDir)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	fitted, err := a.fitFramesToDINA4(ctx, framed, opts, tmpDir)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return fitted, cleanup, nil
}

func (a *ImageMagickPDFAssembler) renderPresentationFrames(ctx context.Context, imagePaths []string, opts PDFAssembleOptions, tmpDir string) ([]string, error) {
	inputs := make([]string, len(imagePaths))
	for i, src := range imagePaths {
		dst := filepath.Join(tmpDir, fmt.Sprintf("frame_%04d.png", i))
		var ferr error
		switch opts.Presentation {
		case pdfPresentationPrint:
			ferr = a.renderPrintFrame(ctx, src, dst)
		case pdfPresentationBook:
			ferr = a.renderBookFrame(ctx, src, dst, tmpDir, i)
		}
		if ferr != nil {
			return nil, ferr
		}
		inputs[i] = dst
	}
	return inputs, nil
}

func (a *ImageMagickPDFAssembler) fitFramesToDINA4(ctx context.Context, inputs []string, opts PDFAssembleOptions, tmpDir string) ([]string, error) {
	fitBG := printMatteColor
	if opts.Presentation == pdfPresentationBook {
		fitBG = bookDeskColor
	}
	fitted := make([]string, len(inputs))
	for i, p := range inputs {
		out := filepath.Join(tmpDir, fmt.Sprintf("din_a4_%04d.png", i))
		if err := a.fitRasterPageToDINA4(ctx, p, out, opts.Density, fitBG); err != nil {
			return nil, err
		}
		fitted[i] = out
	}
	return fitted, nil
}

func (a *ImageMagickPDFAssembler) resolveConvert() (string, error) {
	lp := a.lookPath
	if lp == nil {
		lp = exec.LookPath
	}
	return lp("convert")
}

func (a *ImageMagickPDFAssembler) runConvert(ctx context.Context, args ...string) ([]byte, error) {
	path, err := a.resolveConvert()
	if err != nil {
		return nil, err
	}
	cmdFn := a.command
	if cmdFn == nil {
		cmdFn = exec.CommandContext
	}
	return cmdFn(ctx, path, args...).CombinedOutput()
}

func (a *ImageMagickPDFAssembler) renderPrintFrame(ctx context.Context, src, dst string) error {
	args := []string{
		src,
		"-background", printMatteColor,
		"-gravity", "center",
		"-extent", "108%x108%",
		dst,
	}
	out, err := a.runConvert(ctx, args...)
	if err != nil {
		return fmt.Errorf("convert matte frame: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (a *ImageMagickPDFAssembler) renderBookAgedPage(ctx context.Context, src, dst string) error {
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
	out, err := a.runConvert(ctx, args...)
	if err != nil {
		return fmt.Errorf("convert book aged page: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (a *ImageMagickPDFAssembler) renderBookFrame(ctx context.Context, src, dst, tmpDir string, pageIndex int) error {
	aged := filepath.Join(tmpDir, fmt.Sprintf("book_aged_%04d.png", pageIndex))
	if err := a.renderBookAgedPage(ctx, src, aged); err != nil {
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
	out, err := a.runConvert(ctx, args...)
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
	out, err = a.runConvert(ctx, "-size", fmt.Sprintf("12x%d", h), bookEdgeGradient, grad)
	if err != nil {
		return fmt.Errorf("convert book edge gradient: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	out, err = a.runConvert(ctx, step1, grad, "-gravity", "east", "-compose", "over", "-composite", dst)
	if err != nil {
		return fmt.Errorf("convert book frame (edge): %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (a *ImageMagickPDFAssembler) fitRasterPageToDINA4(ctx context.Context, src, dst string, density int, background string) error {
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
	out, err := a.runConvert(ctx, args...)
	if err != nil {
		return fmt.Errorf("convert DIN A4 page: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
