package install

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// addTargetSubdir populates a target's content tree under root with
// stub agents/commands/skills so the satellite materializer treats it
// as installed.
func addTargetSubdir(t *testing.T, root, subdir string) {
	t.Helper()
	for _, sub := range SymlinkedDirs {
		path := filepath.Join(root, subdir, sub)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "stub.md"), []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTargetLayoutsCoverage(t *testing.T) {
	want := map[Target]struct {
		subDir string
		marker string
	}{
		TargetClaude:   {".claude", "CLAUDE.md"},
		TargetCodex:    {".codex", "AGENTS.md"},
		TargetOpenCode: {".opencode", "AGENTS.md"},
		TargetCursor:   {filepath.Join(".cursor", "rules"), ""},
		TargetCopilot:  {filepath.Join(".github", "copilot"), ""},
		TargetGeneric:  {".ai", "AGENTS.md"},
	}
	for tgt, w := range want {
		layout := LayoutFor(tgt)
		if layout == nil {
			t.Errorf("%s: missing from targetLayouts", tgt)
			continue
		}
		if layout.SubDir != w.subDir {
			t.Errorf("%s: SubDir = %q, want %q", tgt, layout.SubDir, w.subDir)
		}
		if layout.MarkerFile != w.marker {
			t.Errorf("%s: MarkerFile = %q, want %q", tgt, layout.MarkerFile, w.marker)
		}
	}
}

func TestDetectInstalledTargetsAllSupported(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	addTargetSubdir(t, root, ".claude")
	addTargetSubdir(t, root, ".codex")
	addTargetSubdir(t, root, ".opencode")
	addTargetSubdir(t, root, filepath.Join(".cursor", "rules"))
	addTargetSubdir(t, root, ".ai")

	got := DetectInstalledTargets(root)
	gotSet := map[Target]bool{}
	for _, tgt := range got {
		gotSet[tgt] = true
	}
	for _, expected := range []Target{TargetClaude, TargetCodex, TargetOpenCode, TargetCursor, TargetGeneric} {
		if !gotSet[expected] {
			t.Errorf("expected %s in detected set; got %v", expected, got)
		}
	}
}

func TestMaterializeOpenCodeOnlyWritesAgentsMd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may be unavailable on Windows")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	addTargetSubdir(t, root, ".opencode")

	sat := filepath.Join(root, "engines", "mlx")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(SatelliteOptions{
		RootDir: root, SatelliteDir: sat, Scope: "engines/mlx", Version: "test",
	}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	// AGENTS.md should be written for OpenCode.
	data, err := os.ReadFile(filepath.Join(sat, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md missing: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "<!-- hero:satellite -->") {
		t.Errorf("AGENTS.md missing hero:satellite marker:\n%s", body)
	}
	if !strings.Contains(body, "Target:") || !strings.Contains(body, "opencode") {
		t.Errorf("expected single-Target opencode line:\n%s", body)
	}
}

func TestMaterializeCodexAndOpenCodeShareAgentsMd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may be unavailable on Windows")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	addTargetSubdir(t, root, ".codex")
	addTargetSubdir(t, root, ".opencode")

	sat := filepath.Join(root, "engines", "mlx")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Materialize(SatelliteOptions{
		RootDir: root, SatelliteDir: sat, Scope: "engines/mlx", Version: "test",
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}

	// Both targets should appear in the result.
	gotTargets := map[Target]bool{}
	for _, tgt := range res.Targets {
		gotTargets[tgt] = true
	}
	if !gotTargets[TargetCodex] || !gotTargets[TargetOpenCode] {
		t.Errorf("expected both codex+opencode in Targets, got %v", res.Targets)
	}

	// AGENTS.md should be written exactly once but list both targets.
	body, err := os.ReadFile(filepath.Join(sat, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md missing: %v", err)
	}
	if !strings.Contains(string(body), "Targets:") {
		t.Errorf("expected plural Targets: line for shared marker:\n%s", body)
	}
	if !strings.Contains(string(body), "codex") || !strings.Contains(string(body), "opencode") {
		t.Errorf("expected both targets named:\n%s", body)
	}

	// Symlinks should exist under both .codex/ and .opencode/.
	for _, sub := range []string{filepath.Join(".codex", "agents"), filepath.Join(".opencode", "agents")} {
		linkPath := filepath.Join(sat, sub)
		info, err := os.Lstat(linkPath)
		if err != nil {
			t.Errorf("expected symlink at %s: %v", linkPath, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", linkPath)
		}
	}

	// Created list shouldn't contain two AGENTS.md entries.
	seen := 0
	for _, p := range res.Created {
		if strings.HasSuffix(p, "AGENTS.md") {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("expected exactly one AGENTS.md in Created, got %d (entries: %v)", seen, res.Created)
	}
}

func TestMaterializeCursorWritesSymlinksNoMarker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may be unavailable on Windows")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	addTargetSubdir(t, root, filepath.Join(".cursor", "rules"))

	sat := filepath.Join(root, "engines", "mlx")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(SatelliteOptions{
		RootDir: root, SatelliteDir: sat, Scope: "engines/mlx", Version: "test",
	}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	// Symlinks should exist.
	if info, err := os.Lstat(filepath.Join(sat, ".cursor", "rules", "agents")); err != nil {
		t.Errorf("expected cursor agents symlink: %v", err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("cursor agents is not a symlink")
	}
	// No CLAUDE.md or AGENTS.md should be written for cursor.
	if _, err := os.Stat(filepath.Join(sat, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("cursor-only satellite shouldn't have CLAUDE.md")
	}
	if _, err := os.Stat(filepath.Join(sat, "AGENTS.md")); !os.IsNotExist(err) {
		t.Errorf("cursor-only satellite shouldn't have AGENTS.md")
	}
}

func TestPerTargetMarkerSingleVsMultiple(t *testing.T) {
	root := t.TempDir()
	sat := filepath.Join(root, "engines", "mlx")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}

	single := perTargetMarker(root, sat, "engines/mlx", []Target{TargetClaude}, true)
	if !strings.Contains(single, "**Target:** claude") {
		t.Errorf("single-target marker missing singular Target line:\n%s", single)
	}

	multi := perTargetMarker(root, sat, "engines/mlx", []Target{TargetCodex, TargetOpenCode}, true)
	if !strings.Contains(multi, "**Targets:** codex, opencode") {
		t.Errorf("multi-target marker missing plural Targets line:\n%s", multi)
	}
}
