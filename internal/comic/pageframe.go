package comic

import (
	"fmt"
	"strconv"
	"strings"
)

// DescribePageFrame returns human-readable page shape text for image prompts,
// derived from a Gemini-style aspect ratio such as "16:9" or "2:3".
func DescribePageFrame(aspectRatio string) string {
	aspectRatio = strings.TrimSpace(aspectRatio)
	if aspectRatio == "" {
		aspectRatio = comicPageAspectRatio
	}
	parts := strings.Split(aspectRatio, ":")
	if len(parts) != 2 {
		return fmt.Sprintf("%s format", aspectRatio)
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return fmt.Sprintf("%s format", aspectRatio)
	}
	switch {
	case w < h:
		if w == 3 && h == 4 {
			return "portrait 3:4 format (ISO A4–class sheet: 210×297 mm target; image API uses 3:4 as the closest standard ratio)"
		}
		return fmt.Sprintf("portrait %s format (tall comic book page proportions)", aspectRatio)
	case w > h:
		return fmt.Sprintf("landscape %s format (widescreen)", aspectRatio)
	default:
		return fmt.Sprintf("square %s format", aspectRatio)
	}
}
