package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetVegaDir returns the vega-missile directory, auto-detecting if not specified
func GetVegaDir() (string, error) {
	if VegaDir != "" {
		return VegaDir, nil
	}

	// Auto-detect: look for .claude/ and goals/ directory walking up from cwd
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory: %w", err)
	}

	for dir != "/" && dir != "." {
		claudeDir := filepath.Join(dir, ".claude")
		goalsDir := filepath.Join(dir, "goals")

		if _, err := os.Stat(claudeDir); err == nil {
			if _, err := os.Stat(goalsDir); err == nil {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not auto-detect vega-missile directory (no .claude/ and goals/ found)")
}
