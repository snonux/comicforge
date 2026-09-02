// Package stylegrid builds a single montage image comparing all configured art styles and story themes.
package stylegrid

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"codeberg.org/snonux/comicforge/internal/config"
	"codeberg.org/snonux/comicforge/internal/image"
	"codeberg.org/snonux/comicforge/internal/provider"
)

// DefaultScene is the shared composition so style differences stay comparable.
const DefaultScene = "A lone traveler with a backpack pauses on a rocky hill at golden sunset, wind moving their coat; a distant ruined stone tower on the horizon; wide scenic shot, clear silhouette, atmospheric sky."

// Job is one cell in the reference grid.
type Job struct {
	Kind   string // "style" or "theme"
	Header string // shown in the header bar above the image
	Prompt string // full image generation prompt
}

// MontageOptions tunes the output sheet.
type MontageOptions struct {
	CellW   int
	CellH   int
	HeaderH int
	TileX   int
	Gutter  int
	BG      string
}

// BuildJobs expands config style pools and story genres into generation jobs.
func BuildJobs(cfg *config.Config, scene string) []Job {
	if cfg == nil {
		return nil
	}
	scene = strings.TrimSpace(scene)
	if scene == "" {
		scene = DefaultScene
	}
	base := "Single full-frame illustration only. No text, lettering, captions, logos, or watermarks inside the artwork. No borders inside the image. 16:9 composition.\n\nScene (keep this layout consistent for comparison):\n" + scene + "\n\n"

	families := cfg.Styles.Keys()
	var jobs []Job
	for _, key := range families {
		lines := cfg.Styles[key]
		title := config.StyleCategoryTitle(key)
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			header := fmt.Sprintf("Style · %s (%d/%d)", title, i+1, len(lines))
			prompt := base + "Rendering — apply this visual style precisely:\n" + line
			jobs = append(jobs, Job{Kind: "style", Header: header, Prompt: prompt})
		}
	}

	for i, g := range cfg.Story.Genres {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		header := fmt.Sprintf("Theme / genre (%d/%d): %s", i+1, len(cfg.Story.Genres), truncateRunes(g, 72))
		prompt := base + "Rendering — interpret the same scene with mood, palette, and staging strongly guided by this story genre (illustrated comic or graphic-novel finish, readable shapes):\n" + g
		jobs = append(jobs, Job{Kind: "theme", Header: header, Prompt: prompt})
	}
	return jobs
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func defaultMontageOptions(nCells int) MontageOptions {
	// ~5 columns for the default grid (~27 cells).
	tileX := 5
	if nCells <= 4 {
		tileX = 2
	}
	return MontageOptions{
		CellW:   640,
		CellH:   360,
		HeaderH: 56,
		TileX:   tileX,
		Gutter:  6,
		BG:      "#1a1a1a",
	}
}

// GenerateMontage renders each job with img, labels tiles, and runs ImageMagick montage.
func GenerateMontage(ctx context.Context, img provider.ImageProvider, aspectRatio string, jobs []Job, outPath string, opts MontageOptions) error {
	if len(jobs) == 0 {
		return fmt.Errorf("no style/theme jobs (empty config lists?)")
	}
	if _, err := exec.LookPath("convert"); err != nil {
		return fmt.Errorf("ImageMagick convert not found: %w", err)
	}
	if _, err := exec.LookPath("montage"); err != nil {
		return fmt.Errorf("ImageMagick montage not found: %w", err)
	}
	arImg, ok := img.(provider.AspectRatioImageProvider)
	if !ok {
		return fmt.Errorf("image provider %q does not support aspect ratio", img.Name())
	}
	if err := img.IsAvailable(); err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "comicforge-stylegrid-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	for i := range jobs {
		raw := filepath.Join(tmpDir, fmt.Sprintf("raw_%03d.png", i))
		if err := arImg.GenerateImageWithAspectRatio(ctx, jobs[i].Prompt, raw, aspectRatio); err != nil {
			return fmt.Errorf("job %d (%s): %w", i, jobs[i].Header, err)
		}
		norm := filepath.Join(tmpDir, fmt.Sprintf("norm_%03d.png", i))
		if err := normalizeCell(raw, norm, opts.CellW, opts.CellH); err != nil {
			return fmt.Errorf("normalize %d: %w", i, err)
		}
		lblPath := filepath.Join(tmpDir, fmt.Sprintf("caption_%03d.txt", i))
		if err := os.WriteFile(lblPath, []byte(jobs[i].Header), 0o644); err != nil {
			return err
		}
		labeled := filepath.Join(tmpDir, fmt.Sprintf("labeled_%03d.png", i))
		if err := addCaptionHeader(norm, labeled, lblPath, opts.CellW, opts.HeaderH); err != nil {
			return fmt.Errorf("header %d: %w", i, err)
		}
	}

	labeledPaths, err := filepath.Glob(filepath.Join(tmpDir, "labeled_*.png"))
	if err != nil {
		return err
	}
	sort.Strings(labeledPaths)

	args := []string{}
	for _, p := range labeledPaths {
		args = append(args, p)
	}
	args = append(args,
		"-tile", fmt.Sprintf("%dx", opts.TileX),
		"-geometry", fmt.Sprintf("+%d+%d", opts.Gutter, opts.Gutter),
		"-background", opts.BG,
		outPath,
	)
	cmd := exec.CommandContext(ctx, "montage", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("montage: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func normalizeCell(src, dst string, w, h int) error {
	// Cover crop to exact WxH.
	args := []string{
		src,
		"-resize", fmt.Sprintf("%dx%d^", w, h),
		"-gravity", "center",
		"-extent", fmt.Sprintf("%dx%d", w, h),
		dst,
	}
	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert resize: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func addCaptionHeader(imagePath, outPath, captionFile string, cellW, headerH int) error {
	// caption:@file wraps text; header bar above the art.
	args := []string{
		"(",
		"-background", "#252525",
		"-fill", "#f2f2f2",
		"-size", fmt.Sprintf("%dx%d", cellW-20, headerH-8),
		"caption:@" + captionFile,
		")",
		imagePath,
		"-append",
		outPath,
	}
	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert caption: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Run loads config from path, builds jobs, and writes outPath.
func Run(ctx context.Context, configPath, outPath, scene, aspectRatio string) error {
	cfg, err := config.Load(configPath, config.WithProviderNameChecks(
		nil,
		image.DefaultRegistry().Has,
		nil,
	))
	if err != nil {
		return err
	}
	jobs := BuildJobs(cfg, scene)
	opts := defaultMontageOptions(len(jobs))

	img, err := buildImageProvider(cfg)
	if err != nil {
		return err
	}
	if strings.TrimSpace(aspectRatio) == "" {
		aspectRatio = cfg.Comic.AspectRatio
	}
	if strings.TrimSpace(aspectRatio) == "" {
		aspectRatio = "16:9"
	}
	return GenerateMontage(ctx, img, aspectRatio, jobs, outPath, opts)
}

func buildImageProvider(cfg *config.Config) (provider.ImageProvider, error) {
	p, err := image.DefaultRegistry().NewFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	ip, ok := p.(provider.ImageProvider)
	if !ok {
		return nil, fmt.Errorf("image provider %q does not satisfy ImageProvider", cfg.Provider.Image)
	}
	return ip, nil
}
