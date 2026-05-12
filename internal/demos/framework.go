// Package demos provides pluggable demo recording from Hero spec test files.
package demos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// DemoFramework defines the interface for pluggable demo recording adapters.
type DemoFramework interface {
	// Name returns the framework identifier (e.g. "playwright").
	Name() string

	// Record runs tests with video recording enabled and collects artifacts.
	Record(slug string, testFile string, cfg *config.DemosConfig, testCfg *config.TestingConfig, projectRoot string) (*DemoResult, error)

	// VideoDir returns the path to the demo directory for a spec slug.
	VideoDir(slug string, cfg *config.DemosConfig, projectRoot string) string

	// Clean removes demo recordings for a spec slug.
	Clean(slug string, cfg *config.DemosConfig, projectRoot string) error
}

// DemoResult holds the outcome of a demo recording.
type DemoResult struct {
	Slug       string      `json:"slug"`
	Title      string      `json:"title"`
	RecordedAt time.Time   `json:"recorded_at"`
	Videos     []DemoVideo `json:"videos"`
	TestFile   string      `json:"test_file"`
	Status     string      `json:"status"` // "pass", "fail", "error"
	Error      string      `json:"error,omitempty"`
}

// DemoVideo describes a single recorded video file.
type DemoVideo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

// Manifest is written to each demo directory as manifest.json.
type Manifest struct {
	Slug       string      `json:"slug"`
	Title      string      `json:"title"`
	RecordedAt string      `json:"recorded_at"`
	Videos     []DemoVideo `json:"videos"`
	TestFile   string      `json:"test_file"`
	Status     string      `json:"status"`
}

// registry holds registered demo framework adapters.
var registry = map[string]DemoFramework{}

// Register adds a demo framework adapter to the global registry.
func Register(f DemoFramework) {
	registry[f.Name()] = f
}

// Get returns the demo framework adapter for the given name.
func Get(name string) (DemoFramework, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown demo framework %q (available: %s)", name, availableFrameworks())
	}
	return f, nil
}

func availableFrameworks() string {
	var names []string
	for k := range registry {
		names = append(names, k)
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// WriteManifest writes a manifest.json file to the demo directory.
func WriteManifest(result *DemoResult, dir string) error {
	m := Manifest{
		Slug:       result.Slug,
		Title:      result.Title,
		RecordedAt: result.RecordedAt.UTC().Format(time.RFC3339),
		Videos:     result.Videos,
		TestFile:   result.TestFile,
		Status:     result.Status,
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644)
}

// ReadManifest reads a manifest.json from a demo directory.
func ReadManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &m, nil
}

// ListDemos returns all demo manifests found in the demos output directory.
func ListDemos(projectRoot string, cfg *config.DemosConfig) ([]*Manifest, error) {
	dir := demosBaseDir(projectRoot, cfg)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var manifests []*Manifest
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m, err := ReadManifest(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue // skip directories without valid manifests
		}
		manifests = append(manifests, m)
	}
	return manifests, nil
}

// CleanAll removes all demo recordings.
func CleanAll(projectRoot string, cfg *config.DemosConfig) error {
	dir := demosBaseDir(projectRoot, cfg)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func demosBaseDir(projectRoot string, cfg *config.DemosConfig) string {
	dir := ".hero/demos"
	if cfg != nil {
		dir = cfg.OutputDirOrDefault()
	}
	return filepath.Join(projectRoot, dir)
}

func init() {
	// Register built-in frameworks
	Register(&PlaywrightDemoFramework{})
}
