package comic

import (
	"image"
	_ "image/png"
	"math"
	"os"
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
