package herotest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// Generate creates a test file for a spec using the configured framework and mode.
// Returns the path to the generated test file.
func Generate(projectRoot string, s *spec.Spec, cfg *config.TestingConfig, modeOverride string) (string, error) {
	frameworkName := "playwright"
	if cfg != nil {
		frameworkName = cfg.FrameworkOrDefault()
	}

	fw, err := Get(frameworkName)
	if err != nil {
		return "", err
	}

	criteria := ExtractCriteria(s)
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria to generate tests from", s.Slug)
	}

	mode := "autonomous"
	if modeOverride != "" {
		mode = modeOverride
	} else if cfg != nil {
		mode = cfg.ModeOrDefault()
	}

	var content string
	switch mode {
	case "agent":
		// For agent mode, we return the context string as content and let the
		// caller decide how to use it (e.g. inject into MCP context).
		content = fw.AgentContext(s, criteria, cfg)
	case "assisted":
		content, err = fw.GenerateAssisted(s, criteria, cfg)
	case "autonomous":
		content, err = fw.GenerateAutonomous(s, criteria, cfg)
	default:
		return "", fmt.Errorf("unknown test generation mode %q (expected: agent, assisted, autonomous)", mode)
	}

	if err != nil {
		return "", err
	}

	testFile := fw.TestFilePath(s.Slug, cfg)
	absPath := filepath.Join(projectRoot, testFile)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("creating test directory: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing test file: %w", err)
	}

	return testFile, nil
}

// TestFileExists checks if a generated test file exists for a spec.
func TestFileExists(projectRoot string, slug string, cfg *config.TestingConfig) bool {
	frameworkName := "playwright"
	if cfg != nil {
		frameworkName = cfg.FrameworkOrDefault()
	}

	fw, err := Get(frameworkName)
	if err != nil {
		return false
	}

	testFile := fw.TestFilePath(slug, cfg)
	absPath := filepath.Join(projectRoot, testFile)
	_, err = os.Stat(absPath)
	return err == nil
}

// TestFilePath returns the expected test file path for a spec slug.
func TestFilePath(slug string, cfg *config.TestingConfig) string {
	frameworkName := "playwright"
	if cfg != nil {
		frameworkName = cfg.FrameworkOrDefault()
	}

	fw, err := Get(frameworkName)
	if err != nil {
		return filepath.Join("e2e", slug+".spec.ts")
	}

	return fw.TestFilePath(slug, cfg)
}

// RunArgs returns the command and arguments to run tests for a spec.
func RunArgs(slug string, cfg *config.TestingConfig) (string, []string, error) {
	frameworkName := "playwright"
	if cfg != nil {
		frameworkName = cfg.FrameworkOrDefault()
	}

	fw, err := Get(frameworkName)
	if err != nil {
		return "", nil, err
	}

	testFile := fw.TestFilePath(slug, cfg)
	cmd, args := fw.RunCommand(testFile, cfg)
	return cmd, args, nil
}
