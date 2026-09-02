package config

import (
	"fmt"
	"os"
)

// HomeDir returns the user's home directory.
//
// It falls back to "." when the home directory cannot be resolved so callers
// can still build a safe relative path instead of joining against an empty
// string.
func HomeDir() (string, error) {
	return resolveHomeDir(os.UserHomeDir)
}

func resolveHomeDir(userHomeDir func() (string, error)) (string, error) {
	if userHomeDir == nil {
		userHomeDir = os.UserHomeDir
	}
	homeDir, err := userHomeDir()
	if err != nil {
		return ".", fmt.Errorf("resolve home directory: %w", err)
	}

	return homeDir, nil
}
