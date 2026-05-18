package chat

import (
	"fmt"
	"os"
	"path/filepath"
)

// resolveHeroDir locates the .hero directory for a given workspace
// path. workspace may be the project root or already point at .hero;
// both shapes are accepted. An empty workspace is an error — slashes
// that need filesystem access require an explicit workspace from the
// dispatcher.
func resolveHeroDir(workspace string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("workspace path required (no project context attached)")
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}
	// If the caller passed .hero directly, use it as-is.
	if filepath.Base(abs) == ".hero" {
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs, nil
		}
	}
	candidate := filepath.Join(abs, ".hero")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate, nil
	}
	return "", fmt.Errorf("no .hero directory under %s", abs)
}
