package install

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hero-engine/hero/internal/workspace"
)

func TestRepairDropsMissingFolder(t *testing.T) {
	root := setupRoot(t)
	heroDir := filepath.Join(root, ".hero")

	// Record a satellite for a folder that does not exist.
	local := &SatellitesLocal{Version: 1}
	local.Upsert(SatelliteEntry{Path: "engines/ghost", Targets: []string{"claude"}})
	if err := SaveSatellitesLocal(heroDir, local); err != nil {
		t.Fatal(err)
	}

	res, err := Repair(RepairOptions{HeroDir: heroDir, RootDir: root, DryRun: false})
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if len(res.Findings) == 0 {
		t.Errorf("expected at least one finding")
	}
	// Verify entry was dropped.
	loaded, _ := LoadSatellitesLocal(heroDir)
	if loaded.Find("engines/ghost") != nil {
		t.Errorf("ghost entry should be dropped")
	}
}

func TestRepairFlagsDeclaredNotMaterialized(t *testing.T) {
	root := setupRoot(t)
	heroDir := filepath.Join(root, ".hero")
	subs := &SubprojectsManifest{}
	subs.AddSubproject(Subproject{Path: "engines/mlx", Scope: "engines/mlx"})
	if err := SaveSubprojects(heroDir, subs); err != nil {
		t.Fatal(err)
	}

	res, err := Repair(RepairOptions{HeroDir: heroDir, RootDir: root, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	hit := false
	for _, f := range res.Findings {
		if f.Kind == DriftDeclaredNotLocal && f.Path == "engines/mlx" {
			hit = true
		}
	}
	if !hit {
		t.Errorf("expected DriftDeclaredNotLocal finding, got %+v", res.Findings)
	}
}

func TestRepairFixesBrokenSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may be unavailable on Windows in CI")
	}
	root := setupRoot(t)
	heroDir := filepath.Join(root, ".hero")
	sat := filepath.Join(root, "engines", "mlx")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	// Materialize then break a symlink.
	if _, err := Materialize(SatelliteOptions{
		RootDir:      root,
		SatelliteDir: sat,
		Scope:        "engines/mlx",
		Version:      "test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := RecordSatellite(heroDir, SatelliteEntry{
		Path:    "engines/mlx",
		Targets: []string{"claude"},
	}); err != nil {
		t.Fatal(err)
	}

	// Break the agents symlink.
	agentsLink := filepath.Join(sat, ".claude", "agents")
	if err := os.Remove(agentsLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/nonexistent", agentsLink); err != nil {
		t.Fatal(err)
	}

	res, err := Repair(RepairOptions{
		HeroDir: heroDir,
		RootDir: root,
		Version: "test",
		DryRun:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Repaired) == 0 {
		t.Errorf("expected repair action, got %+v", res)
	}
	// Verify symlink is now correct.
	target, err := os.Readlink(agentsLink)
	if err != nil {
		t.Fatal(err)
	}
	resolved := filepath.Clean(filepath.Join(filepath.Dir(agentsLink), target))
	want := filepath.Join(root, ".claude", "agents")
	if resolved != want {
		t.Errorf("symlink resolved to %s, want %s", resolved, want)
	}
}

func TestRepairRewritesMissingMarker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may be unavailable on Windows in CI")
	}
	root := setupRoot(t)
	heroDir := filepath.Join(root, ".hero")
	sat := filepath.Join(root, "x")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(SatelliteOptions{
		RootDir:      root,
		SatelliteDir: sat,
		Scope:        "x",
		Version:      "test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := RecordSatellite(heroDir, SatelliteEntry{Path: "x", Targets: []string{"claude"}}); err != nil {
		t.Fatal(err)
	}

	// Remove the marker.
	if err := os.Remove(filepath.Join(sat, workspace.SatelliteMarker)); err != nil {
		t.Fatal(err)
	}
	res, err := Repair(RepairOptions{
		HeroDir: heroDir,
		RootDir: root,
		Version: "test",
		DryRun:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(sat, workspace.SatelliteMarker)); err != nil {
		t.Errorf("marker should be rewritten: %v", err)
	}
	if len(res.Repaired) == 0 {
		t.Errorf("expected repair action")
	}
}
