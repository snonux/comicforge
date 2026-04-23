package comic

import (
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PDFAssembleOptions configures ImageMagick conversion when building the comic PDF.
type PDFAssembleOptions struct {
	// Density is passed to ImageMagick -density (DPI for the vector/page coordinate system).
	Density int
	// JPEGQuality is 0 for default lossless-style output, or 1–100 for JPEG compression
	// inside the PDF (smaller files, similar to a "compressed" comic PDF).
	JPEGQuality int
	// Presentation is one of:
	//   none  — full bleed
	//   print — light matte border, then each page is fitted to ISO A4 portrait at pdf.density
	//   book  — aged newsprint, tilt, shadow, thick page-edge strip, then ISO A4 portrait pages
	Presentation string
}

// BookPageAspectRatio is the Gemini image aspect ratio for pdf.presentation print or book.
// API-supported ratio closest to ISO 216 A4 portrait (210×297 mm, 1:√2).
const BookPageAspectRatio = "3:4"

const (
	pdfPresentationNone  = "none"
	pdfPresentationPrint = "print"
	pdfPresentationBook  = "book"

	printMatteColor = "#f4efe6"
	bookDeskColor   = "#ebe6dc"
	// bookEdgeGradient is drawn on the outer vertical edge after tilt (paper → block shadow).
	bookEdgeGradient = "gradient:#ddd7cd-#2c2926"

	// bookAged* tune the “yellowed / shelf-worn” pass (ImageMagick) before tilt/shadow.
	bookAgedModulate      = "100,86,100" // brightness,saturation,hue
	bookAgedSepiaPercent  = "16%"
	bookAgedNoiseAtten    = "0.4"
	bookAgedVignetteBlur  = "0x22"
	bookAgedVignetteLevel = "50x100%" // edge darkening strength
)

// IsBookPDFPresentation reports whether p selects the book (aged / tilt / shadow) PDF pipeline.
func IsBookPDFPresentation(p string) bool {
	return strings.ToLower(strings.TrimSpace(p)) == pdfPresentationBook
}

// IsDINA4ClassPDFPresentation reports whether p assembles each PDF page at ISO A4 portrait
// (210×297 mm at the configured density): print or book.
func IsDINA4ClassPDFPresentation(p string) bool {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case pdfPresentationPrint, pdfPresentationBook:
		return true
	default:
		return false
	}
}

// AssembleComicPDF combines comic pages into a PDF using ImageMagick.
func AssembleComicPDF(outputDir, titleSlug string, imagePaths []string, opts PDFAssembleOptions) (string, error) {
	if len(imagePaths) == 0 {
		return "", fmt.Errorf("no comic images to assemble into PDF")
	}
	if _, err := exec.LookPath("convert"); err != nil {
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
				ferr = renderPrintFrame(src, dst)
			case pdfPresentationBook:
				ferr = renderBookFrame(src, dst, tmpDir, i)
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
			if err := fitRasterPageToDINA4(p, out, opts.Density, fitBG); err != nil {
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

	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("convert failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return pdfPath, nil
}

func renderPrintFrame(src, dst string) error {
	args := []string{
		src,
		"-background", printMatteColor,
		"-gravity", "center",
		"-extent", "108%x108%",
		dst,
	}
	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert matte frame: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func renderBookAgedPage(src, dst string) error {
	// Sepia + desaturation + fine grain + vignette ≈ old newsprint / stored comic.
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
	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert book aged page: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func renderBookFrame(src, dst, tmpDir string, pageIndex int) error {
	aged := filepath.Join(tmpDir, fmt.Sprintf("book_aged_%04d.png", pageIndex))
	if err := renderBookAgedPage(src, aged); err != nil {
		return err
	}

	step1 := filepath.Join(tmpDir, fmt.Sprintf("book_step1_%04d.png", pageIndex))
	angle := 2.1
	if pageIndex%2 == 1 {
		angle = -2.1
	}
	// Room for shadow after rotation; cream fill for empty corners.
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
	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
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
	cmd = exec.Command("convert", "-size", fmt.Sprintf("12x%d", h), bookEdgeGradient, grad)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert book edge gradient: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	cmd = exec.Command("convert", step1, grad, "-gravity", "east", "-compose", "over", "-composite", dst)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert book frame (edge): %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func dinAPortraitPixels(dpi int) (w, h int) {
	const mmW, mmH = 210.0, 297.0
	w = int(math.Round(float64(dpi) * mmW / 25.4))
	h = int(math.Round(float64(dpi) * mmH / 25.4))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// fitRasterPageToDINA4 letterboxes or fits raster art into exact ISO A4 portrait pixels at the given DPI.
// background is the letterbox / pad color (print matte vs book desk).
func fitRasterPageToDINA4(src, dst string, density int, background string) error {
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
	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert DIN A4 page: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func imageHeight(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, err
	}
	return cfg.Height, nil
}

func normalizePDFAssembleOptions(o PDFAssembleOptions) PDFAssembleOptions {
	if o.Density <= 0 {
		o.Density = 150
	}
	if o.JPEGQuality < 0 {
		o.JPEGQuality = 0
	}
	if o.JPEGQuality > 100 {
		o.JPEGQuality = 100
	}
	o.Presentation = strings.ToLower(strings.TrimSpace(o.Presentation))
	if o.Presentation == "" {
		o.Presentation = pdfPresentationNone
	}
	return o
}
