package demos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

func TestRegistryGetPlaywright(t *testing.T) {
	fw, err := Get("playwright")
	if err != nil {
		t.Fatalf("Get(playwright) failed: %v", err)
	}
	if fw.Name() != "playwright" {
		t.Errorf("Name() = %q, want %q", fw.Name(), "playwright")
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	_, err := Get("ffmpeg")
	if err == nil {
		t.Fatal("Get(ffmpeg) should fail, got nil error")
	}
	if !strings.Contains(err.Error(), "unknown demo framework") {
		t.Errorf("error = %q, want to contain 'unknown demo framework'", err.Error())
	}
}

func TestWriteAndReadManifest(t *testing.T) {
	tmpDir := t.TempDir()

	result := &DemoResult{
		Slug:       "login-flow",
		Title:      "Login Flow",
		RecordedAt: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		Videos: []DemoVideo{
			{Name: "login-flow-video1", Path: "login-flow-video1.webm", SizeBytes: 1024},
		},
		TestFile: "e2e/login-flow.spec.ts",
		Status:   "pass",
	}

	if err := WriteManifest(result, tmpDir); err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	// Check file exists
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatal("manifest.json not created")
	}

	// Read it back
	m, err := ReadManifest(tmpDir)
	if err != nil {
		t.Fatalf("ReadManifest failed: %v", err)
	}

	if m.Slug != "login-flow" {
		t.Errorf("Slug = %q, want %q", m.Slug, "login-flow")
	}
	if m.Title != "Login Flow" {
		t.Errorf("Title = %q, want %q", m.Title, "Login Flow")
	}
	if m.Status != "pass" {
		t.Errorf("Status = %q, want %q", m.Status, "pass")
	}
	if len(m.Videos) != 1 {
		t.Fatalf("Videos count = %d, want 1", len(m.Videos))
	}
	if m.Videos[0].Name != "login-flow-video1" {
		t.Errorf("Video name = %q, want %q", m.Videos[0].Name, "login-flow-video1")
	}
	if m.TestFile != "e2e/login-flow.spec.ts" {
		t.Errorf("TestFile = %q, want %q", m.TestFile, "e2e/login-flow.spec.ts")
	}
}

func TestReadManifestMissing(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ReadManifest(tmpDir)
	if err == nil {
		t.Fatal("ReadManifest should fail for missing manifest")
	}
}

func TestReadManifestInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte("not json"), 0o644)
	_, err := ReadManifest(tmpDir)
	if err == nil {
		t.Fatal("ReadManifest should fail for invalid JSON")
	}
}

func TestListDemos(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two demo directories with manifests
	for _, slug := range []string{"feature-a", "feature-b"} {
		dir := filepath.Join(tmpDir, ".hero", "demos", slug)
		os.MkdirAll(dir, 0o755)
		m := Manifest{
			Slug:       slug,
			Title:      slug,
			RecordedAt: time.Now().UTC().Format(time.RFC3339),
			Status:     "pass",
		}
		data, _ := json.MarshalIndent(m, "", "  ")
		os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644)
	}

	// Also create a directory without a manifest (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, ".hero", "demos", "no-manifest"), 0o755)

	cfg := &config.DemosConfig{OutputDir: ".hero/demos"}
	manifests, err := ListDemos(tmpDir, cfg)
	if err != nil {
		t.Fatalf("ListDemos failed: %v", err)
	}
	if len(manifests) != 2 {
		t.Errorf("got %d manifests, want 2", len(manifests))
	}
}

func TestListDemosEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Directory doesn't exist — should return nil, nil
	cfg := &config.DemosConfig{OutputDir: ".hero/demos"}
	manifests, err := ListDemos(tmpDir, cfg)
	if err != nil {
		t.Fatalf("ListDemos failed: %v", err)
	}
	if manifests != nil {
		t.Errorf("expected nil manifests for non-existent dir, got %v", manifests)
	}
}

func TestCleanAll(t *testing.T) {
	tmpDir := t.TempDir()
	demosDir := filepath.Join(tmpDir, ".hero", "demos", "test-demo")
	os.MkdirAll(demosDir, 0o755)
	os.WriteFile(filepath.Join(demosDir, "video.webm"), []byte("fake video"), 0o644)

	cfg := &config.DemosConfig{OutputDir: ".hero/demos"}
	if err := CleanAll(tmpDir, cfg); err != nil {
		t.Fatalf("CleanAll failed: %v", err)
	}

	baseDir := filepath.Join(tmpDir, ".hero", "demos")
	if _, err := os.Stat(baseDir); !os.IsNotExist(err) {
		t.Error("demos directory should be removed after CleanAll")
	}
}

func TestCleanAllNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.DemosConfig{OutputDir: ".hero/demos"}
	// Should not error when directory doesn't exist
	if err := CleanAll(tmpDir, cfg); err != nil {
		t.Fatalf("CleanAll on non-existent dir should not error: %v", err)
	}
}

// --- PlaywrightDemoFramework tests ---

func TestPlaywrightVideoDir(t *testing.T) {
	fw := &PlaywrightDemoFramework{}

	got := fw.VideoDir("my-feature", nil, "/project")
	want := filepath.Join("/project", ".hero", "demos", "my-feature")
	if got != want {
		t.Errorf("VideoDir = %q, want %q", got, want)
	}

	cfg := &config.DemosConfig{OutputDir: "custom/demos"}
	got = fw.VideoDir("my-feature", cfg, "/project")
	want = filepath.Join("/project", "custom", "demos", "my-feature")
	if got != want {
		t.Errorf("VideoDir(custom) = %q, want %q", got, want)
	}
}

func TestPlaywrightClean(t *testing.T) {
	tmpDir := t.TempDir()
	fw := &PlaywrightDemoFramework{}

	// Create a demo directory
	demoDir := filepath.Join(tmpDir, ".hero", "demos", "cleanup-test")
	os.MkdirAll(demoDir, 0o755)
	os.WriteFile(filepath.Join(demoDir, "video.webm"), []byte("data"), 0o644)

	if err := fw.Clean("cleanup-test", nil, tmpDir); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	if _, err := os.Stat(demoDir); !os.IsNotExist(err) {
		t.Error("demo directory should be removed after Clean")
	}
}

func TestPlaywrightCleanNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	fw := &PlaywrightDemoFramework{}
	// Should not error
	if err := fw.Clean("nonexistent", nil, tmpDir); err != nil {
		t.Fatalf("Clean on non-existent should not error: %v", err)
	}
}

func TestPlaywrightRecordMissingTestFile(t *testing.T) {
	tmpDir := t.TempDir()
	fw := &PlaywrightDemoFramework{}

	_, err := fw.Record("test-slug", "e2e/nonexistent.spec.ts", nil, nil, tmpDir)
	if err == nil {
		t.Fatal("Record should fail for missing test file")
	}
	if !strings.Contains(err.Error(), "test file not found") {
		t.Errorf("error = %q, want to contain 'test file not found'", err.Error())
	}
}

func TestSanitizeVideoName(t *testing.T) {
	tests := []struct {
		original string
		slug     string
		want     string
	}{
		{"abc123.webm", "my-feat", "my-feat-abc123.webm"},
		{"my-feat-abc.webm", "my-feat", "my-feat-abc.webm"}, // already prefixed
	}
	for _, tt := range tests {
		got := sanitizeVideoName(tt.original, tt.slug)
		if got != tt.want {
			t.Errorf("sanitizeVideoName(%q, %q) = %q, want %q", tt.original, tt.slug, got, tt.want)
		}
	}
}

func TestMoveFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	os.WriteFile(src, []byte("hello"), 0o644)

	if err := moveFile(src, dst); err != nil {
		t.Fatalf("moveFile failed: %v", err)
	}

	// Source should be gone
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source file should be removed after move")
	}

	// Dest should exist with correct content
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("dest content = %q, want %q", string(data), "hello")
	}
}

func TestDemosBaseDir(t *testing.T) {
	got := demosBaseDir("/project", nil)
	want := filepath.Join("/project", ".hero", "demos")
	if got != want {
		t.Errorf("demosBaseDir(nil) = %q, want %q", got, want)
	}

	cfg := &config.DemosConfig{OutputDir: "output/demos"}
	got = demosBaseDir("/project", cfg)
	want = filepath.Join("/project", "output", "demos")
	if got != want {
		t.Errorf("demosBaseDir(custom) = %q, want %q", got, want)
	}
}

func TestBuildCommand(t *testing.T) {
	fw := &PlaywrightDemoFramework{}

	// Default
	runner, args := fw.buildCommand("e2e/test.spec.ts", nil)
	if runner != "npx" {
		t.Errorf("runner = %q, want npx", runner)
	}
	if len(args) < 3 {
		t.Fatalf("args too short: %v", args)
	}
	if args[0] != "playwright" || args[1] != "test" {
		t.Errorf("unexpected args: %v", args)
	}
	// Should contain reporter
	foundReporter := false
	for _, a := range args {
		if a == "--reporter=list" {
			foundReporter = true
		}
	}
	if !foundReporter {
		t.Errorf("args %v missing --reporter=list", args)
	}

	// Custom runner
	cfg := &config.TestingConfig{
		RunnerCommand: "pnpm playwright test",
		ConfigPath:    "pw.config.ts",
	}
	runner, args = fw.buildCommand("e2e/test.spec.ts", cfg)
	if runner != "pnpm" {
		t.Errorf("runner = %q, want pnpm", runner)
	}
	foundConfig := false
	for i, a := range args {
		if a == "--config" && i+1 < len(args) && args[i+1] == "pw.config.ts" {
			foundConfig = true
		}
	}
	if !foundConfig {
		t.Errorf("args %v missing --config pw.config.ts", args)
	}
}

func TestCollectVideosNoDir(t *testing.T) {
	tmpDir := t.TempDir()
	fw := &PlaywrightDemoFramework{}

	// test-results/ doesn't exist — should return empty, no error
	videos, err := fw.collectVideos(tmpDir, tmpDir, "slug")
	if err != nil {
		t.Fatalf("collectVideos should not error for missing dir: %v", err)
	}
	if len(videos) != 0 {
		t.Errorf("got %d videos, want 0", len(videos))
	}
}

func TestCollectVideos(t *testing.T) {
	tmpDir := t.TempDir()
	fw := &PlaywrightDemoFramework{}

	// Create fake test-results with a webm
	testResultsDir := filepath.Join(tmpDir, "test-results", "test-chromium")
	os.MkdirAll(testResultsDir, 0o755)
	os.WriteFile(filepath.Join(testResultsDir, "abc123.webm"), []byte("fake video data"), 0o644)
	// Also a non-webm file (should be skipped)
	os.WriteFile(filepath.Join(testResultsDir, "screenshot.png"), []byte("img"), 0o644)

	outDir := filepath.Join(tmpDir, "demos-out")
	os.MkdirAll(outDir, 0o755)

	videos, err := fw.collectVideos(tmpDir, outDir, "my-feat")
	if err != nil {
		t.Fatalf("collectVideos failed: %v", err)
	}
	if len(videos) != 1 {
		t.Fatalf("got %d videos, want 1", len(videos))
	}
	if !strings.HasSuffix(videos[0].Path, ".webm") {
		t.Errorf("video path = %q, want .webm suffix", videos[0].Path)
	}
	if videos[0].SizeBytes == 0 {
		t.Error("video size should be > 0")
	}

	// Verify the video was moved to outDir
	movedPath := filepath.Join(outDir, videos[0].Path)
	if _, err := os.Stat(movedPath); os.IsNotExist(err) {
		t.Errorf("video not moved to %s", movedPath)
	}

	// Original should be gone
	origPath := filepath.Join(testResultsDir, "abc123.webm")
	if _, err := os.Stat(origPath); !os.IsNotExist(err) {
		t.Error("original video should be removed after move")
	}
}
