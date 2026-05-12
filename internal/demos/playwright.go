package demos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// PlaywrightDemoFramework implements DemoFramework for Playwright video recording.
type PlaywrightDemoFramework struct{}

func (p *PlaywrightDemoFramework) Name() string { return "playwright" }

func (p *PlaywrightDemoFramework) VideoDir(slug string, cfg *config.DemosConfig, projectRoot string) string {
	base := demosBaseDir(projectRoot, cfg)
	return filepath.Join(base, slug)
}

func (p *PlaywrightDemoFramework) Clean(slug string, cfg *config.DemosConfig, projectRoot string) error {
	dir := p.VideoDir(slug, cfg, projectRoot)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (p *PlaywrightDemoFramework) Record(slug string, testFile string, cfg *config.DemosConfig, testCfg *config.TestingConfig, projectRoot string) (*DemoResult, error) {
	result := &DemoResult{
		Slug:       slug,
		RecordedAt: time.Now().UTC(),
		TestFile:   testFile,
	}

	// Check test file exists
	absTestFile := filepath.Join(projectRoot, testFile)
	if _, err := os.Stat(absTestFile); os.IsNotExist(err) {
		result.Status = "error"
		result.Error = fmt.Sprintf("test file not found: %s", testFile)
		return result, fmt.Errorf("test file not found: %s", testFile)
	}

	// Create output directory
	outDir := p.VideoDir(slug, cfg, projectRoot)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("creating output directory: %s", err)
		return result, err
	}

	// Build the playwright command
	runner, args := p.buildCommand(testFile, testCfg)

	cmd := exec.Command(runner, args...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Set PWVIDEO=1 so generated tests enable video recording
	cmd.Env = append(os.Environ(), "PWVIDEO=1")

	runErr := cmd.Run()

	if runErr != nil {
		result.Status = "fail"
		result.Error = runErr.Error()
	} else {
		result.Status = "pass"
	}

	// Collect video files from Playwright's test-results directory
	collected, collectErr := p.collectVideos(projectRoot, outDir, slug)
	if collectErr != nil {
		// Non-fatal: recording may have worked but collection failed
		if result.Error == "" {
			result.Error = fmt.Sprintf("collecting videos: %s", collectErr)
		}
	}
	result.Videos = collected

	// Write manifest
	if writeErr := WriteManifest(result, outDir); writeErr != nil {
		return result, fmt.Errorf("writing manifest: %w", writeErr)
	}

	if runErr != nil {
		return result, fmt.Errorf("tests failed: %w", runErr)
	}

	return result, nil
}

func (p *PlaywrightDemoFramework) buildCommand(testFile string, testCfg *config.TestingConfig) (string, []string) {
	runner := "npx"
	args := []string{"playwright", "test", testFile, "--reporter=list"}

	if testCfg != nil && testCfg.RunnerCommand != "" {
		parts := strings.Fields(testCfg.RunnerCommand)
		if len(parts) > 0 {
			runner = parts[0]
			args = append(parts[1:], testFile, "--reporter=list")
		}
	}

	if testCfg != nil && testCfg.ConfigPath != "" {
		args = append(args, "--config", testCfg.ConfigPath)
	}

	return runner, args
}

// collectVideos moves WebM files from Playwright's test-results/ to the demo directory.
func (p *PlaywrightDemoFramework) collectVideos(projectRoot, outDir, slug string) ([]DemoVideo, error) {
	testResultsDir := filepath.Join(projectRoot, "test-results")

	var videos []DemoVideo
	err := filepath.Walk(testResultsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".webm") {
			return nil
		}

		// Move the video to the demo output directory
		name := sanitizeVideoName(info.Name(), slug)
		destPath := filepath.Join(outDir, name)

		if moveErr := moveFile(path, destPath); moveErr != nil {
			return nil // skip files we can't move
		}

		destInfo, statErr := os.Stat(destPath)
		var size int64
		if statErr == nil {
			size = destInfo.Size()
		}

		videos = append(videos, DemoVideo{
			Name:      strings.TrimSuffix(name, ".webm"),
			Path:      name,
			SizeBytes: size,
		})
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return videos, err
	}

	return videos, nil
}

// sanitizeVideoName creates a clean filename for a collected video.
func sanitizeVideoName(originalName, slug string) string {
	// Playwright generates random hashes for video names.
	// We prefix with the slug for clarity.
	if strings.HasPrefix(originalName, slug) {
		return originalName
	}
	return slug + "-" + originalName
}

// moveFile moves a file from src to dst, falling back to copy+delete if rename fails.
func moveFile(src, dst string) error {
	// Try rename first (fast, same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fall back to copy + delete
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return err
	}
	return os.Remove(src)
}
