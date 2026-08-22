// Command stylegrid renders one comparison image for every style line and story genre in config.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/snonux/comicforge/internal/stylegrid"
)

func main() {
	configPath := flag.String("config", "", "config file (default: same search paths as comicforge)")
	outPath := flag.String("output", "stylegrid.png", "output PNG path")
	scene := flag.String("scene", stylegrid.DefaultScene, "shared scene description for all cells")
	aspect := flag.String("aspect-ratio", "16:9", "Gemini aspect ratio for each cell")
	timeout := flag.Duration("timeout", 45*time.Minute, "overall timeout for all generations + montage")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if err := stylegrid.Run(ctx, *configPath, *outPath, *scene, *aspect); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\n", *outPath)
}
