package prompts

import (
	"embed"
	"path/filepath"
	"strings"
)

// embeddedTemplates stores the built-in prompt templates shipped with the binary.
//
//go:embed *.md
var embeddedTemplates embed.FS

// Read returns the embedded prompt template named by name.
func Read(name string) ([]byte, error) {
	return embeddedTemplates.ReadFile(filepath.Base(strings.TrimSpace(name)))
}
