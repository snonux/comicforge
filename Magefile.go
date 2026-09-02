//go:build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const binaryName = "comicforge"

// Default builds the project.
var Default = Build

// Build compiles the comicforge binary.
func Build() error {
	return sh.RunV("go", "build", "-o", binaryName, "./cmd/comicforge")
}

// Test runs the Go test suite.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// Lint runs golangci-lint (install: https://golangci-lint.run/welcome/install/).
func Lint() error {
	if _, err := sh.Output("golangci-lint", "version"); err != nil {
		return fmt.Errorf("golangci-lint not found in PATH; install from https://golangci-lint.run/welcome/install/: %w", err)
	}
	return sh.RunV("golangci-lint", "run")
}

// Install copies the built binary into GOPATH/bin.
func Install() error {
	mg.Deps(Build)

	gopath, err := resolveGOPATH()
	if err != nil {
		return err
	}

	binDir := filepath.Join(gopath, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create bin dir: %w", err)
	}

	return sh.RunV("cp", "-v", binaryName, filepath.Join(binDir, binaryName))
}

// Clean removes build outputs from the repository root.
func Clean() error {
	return sh.RunV("rm", "-f", binaryName)
}

// Stylegrid renders stylegrid.png (all style lines + story genres from config) via Gemini; requires ImageMagick montage.
func Stylegrid() error {
	return sh.RunV("go", "run", "./cmd/stylegrid", "--output", "stylegrid.png")
}

func resolveGOPATH() (string, error) {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return gopath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(homeDir, "go"), nil
}
