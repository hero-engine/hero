package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hero-engine/hero/internal/version"
)

// testContentFS creates an in-memory fs.FS with test agent/command/skill files.
func testContentFS() fstest.MapFS {
	return fstest.MapFS{
		"agents/hero.md":      {Data: []byte("# Test Hero Agent v2\n")},
		"agents/engineer.md":  {Data: []byte("# Test Engineer v2\n")},
		"commands/design.md":  {Data: []byte("# Test Design Command v2\n")},
		"skills/go-stack.md":  {Data: []byte("# Test Go Stack v2\n")},
	}
}

func TestUpgradeNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("upgrade")
	if err == nil {
		t.Fatal("upgrade should error without workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace found") {
		t.Errorf("error should mention no workspace: %v", err)
	}
}

func TestUpgradeAlreadyAtVersion(t *testing.T) {
	env := newTestEnv(t)

	// Stamp workspace at 1.0.0 and set binary version to 1.0.0
	if err := version.StampInit(env.heroDir, "1.0.0"); err != nil {
		t.Fatalf("StampInit: %v", err)
	}
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	output, err := runCmd("upgrade")
	if err != nil {
		t.Fatalf("upgrade returned error: %v", err)
	}

	if !strings.Contains(output, "already at v1.0.0") {
		t.Errorf("should say already at version: %q", output)
	}
}

func TestUpgradeDryRun(t *testing.T) {
	env := newTestEnv(t)

	// Set up test content FS
	upgradeContentFS = testContentFS()
	defer func() { upgradeContentFS = nil }()

	// Stamp workspace at 0.9.0, binary at 1.0.0
	if err := version.StampInit(env.heroDir, "0.9.0"); err != nil {
		t.Fatalf("StampInit: %v", err)
	}
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	// Create opencode target directory so detectInstalledTarget finds it
	if err := os.MkdirAll(filepath.Join(env.dir, ".opencode", "agents"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	output, err := runCmd("upgrade", "--dry-run")
	if err != nil {
		t.Fatalf("upgrade --dry-run returned error: %v", err)
	}

	if !strings.Contains(output, "dry run") {
		t.Errorf("should indicate dry run: %q", output)
	}

	// Version file should NOT be updated
	info, _ := version.Read(env.heroDir)
	if info != nil && info.HeroVersion != "0.9.0" {
		t.Errorf("dry run should not update version.json, got: %s", info.HeroVersion)
	}
}

func TestUpgradeUpdatesFiles(t *testing.T) {
	env := newTestEnv(t)

	// Set up test content FS
	upgradeContentFS = testContentFS()
	defer func() { upgradeContentFS = nil }()

	// Create opencode target with an old version of the file
	agentRelPath := filepath.Join(".opencode", "agents", "hero.md")
	agentDestPath := filepath.Join(env.dir, agentRelPath)

	if err := os.MkdirAll(filepath.Dir(agentDestPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	oldContent := []byte("# Old Hero Agent\n")
	if err := os.WriteFile(agentDestPath, oldContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Get checksum of the installed file (simulating a proper install)
	oldChecksum, err := version.FileChecksum(agentDestPath)
	if err != nil {
		t.Fatalf("FileChecksum: %v", err)
	}

	// Stamp workspace at 0.9.0 with the installed file checksum
	info := &version.Info{
		HeroVersion: "0.9.0",
		InstalledFiles: map[string]string{
			agentRelPath: oldChecksum,
		},
	}
	if err := version.Write(env.heroDir, info); err != nil {
		t.Fatalf("Write version: %v", err)
	}

	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	output, err := runCmd("upgrade")
	if err != nil {
		t.Fatalf("upgrade returned error: %v", err)
	}

	if !strings.Contains(output, "update") {
		t.Errorf("should show updated file: %q", output)
	}

	// The file should now have content from the test content FS
	data, err := os.ReadFile(agentDestPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "# Test Hero Agent v2\n" {
		t.Errorf("file content should be updated: %q", string(data))
	}

	// Version should be updated
	info, err = version.Read(env.heroDir)
	if err != nil {
		t.Fatalf("Read version: %v", err)
	}
	if info.HeroVersion != "1.0.0" {
		t.Errorf("version should be 1.0.0, got: %s", info.HeroVersion)
	}

	// Checksums should be updated for the new file
	newChecksum, _ := version.FileChecksum(agentDestPath)
	if info.InstalledFiles[agentRelPath] != newChecksum {
		t.Errorf("checksum should be updated after upgrade")
	}
}

func TestUpgradeSkipsCustomizedFiles(t *testing.T) {
	env := newTestEnv(t)

	// Set up test content FS
	upgradeContentFS = testContentFS()
	defer func() { upgradeContentFS = nil }()

	// Stamp workspace at 0.9.0 with known checksum for the agent file
	agentRelPath := filepath.Join(".opencode", "agents", "hero.md")
	agentDestPath := filepath.Join(env.dir, agentRelPath)

	if err := os.MkdirAll(filepath.Dir(agentDestPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write the original installed file
	originalContent := []byte("# Original Hero Agent\n")
	if err := os.WriteFile(agentDestPath, originalContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Get its checksum
	originalChecksum, err := version.FileChecksum(agentDestPath)
	if err != nil {
		t.Fatalf("FileChecksum: %v", err)
	}

	// Now write a customized version (simulate user edit)
	if err := os.WriteFile(agentDestPath, []byte("# My Custom Hero Agent\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Stamp version with the original checksum
	info := &version.Info{
		HeroVersion: "0.9.0",
		InstalledFiles: map[string]string{
			agentRelPath: originalChecksum,
		},
	}
	if err := version.Write(env.heroDir, info); err != nil {
		t.Fatalf("Write version: %v", err)
	}

	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	output, err := runCmd("upgrade")
	if err != nil {
		t.Fatalf("upgrade returned error: %v", err)
	}

	if !strings.Contains(output, "skip") {
		t.Errorf("should show skipped customized file: %q", output)
	}

	// The file should still have the user's customization
	data, err := os.ReadFile(agentDestPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "# My Custom Hero Agent\n" {
		t.Errorf("customized file should not be overwritten: %q", string(data))
	}
}

func TestUpgradeForceOverwritesCustomized(t *testing.T) {
	env := newTestEnv(t)

	// Set up test content FS
	upgradeContentFS = testContentFS()
	defer func() { upgradeContentFS = nil }()

	// Same setup as TestUpgradeSkipsCustomizedFiles
	agentRelPath := filepath.Join(".opencode", "agents", "hero.md")
	agentDestPath := filepath.Join(env.dir, agentRelPath)

	if err := os.MkdirAll(filepath.Dir(agentDestPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	originalContent := []byte("# Original Hero Agent\n")
	if err := os.WriteFile(agentDestPath, originalContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	originalChecksum, err := version.FileChecksum(agentDestPath)
	if err != nil {
		t.Fatalf("FileChecksum: %v", err)
	}

	// Customize the file
	if err := os.WriteFile(agentDestPath, []byte("# My Custom Hero Agent\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	info := &version.Info{
		HeroVersion: "0.9.0",
		InstalledFiles: map[string]string{
			agentRelPath: originalChecksum,
		},
	}
	if err := version.Write(env.heroDir, info); err != nil {
		t.Fatalf("Write version: %v", err)
	}

	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	output, err := runCmd("upgrade", "--force")
	if err != nil {
		t.Fatalf("upgrade --force returned error: %v", err)
	}

	if !strings.Contains(output, "update") {
		t.Errorf("force should show updated file: %q", output)
	}

	// The file should now have the new content from the test FS
	data, err := os.ReadFile(agentDestPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "# Test Hero Agent v2\n" {
		t.Errorf("force should overwrite customized file: %q", string(data))
	}
}

func TestUpgradeUpdatesAllInstalledTargetsByDefault(t *testing.T) {
	env := newTestEnv(t)
	upgradeContentFS = testContentFS()
	defer func() { upgradeContentFS = nil }()

	// Pre-create both .opencode/ and .claude/ with one agent file each,
	// and stamp the existing checksums so the upgrade treats the files
	// as unmodified (otherwise they'd be skipped as "customized").
	installedFiles := map[string]string{}
	for _, dir := range []string{".opencode/agents", ".claude/agents"} {
		if err := os.MkdirAll(filepath.Join(env.dir, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", dir, err)
		}
		path := filepath.Join(env.dir, dir, "hero.md")
		if err := os.WriteFile(path, []byte("# old\n"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", dir, err)
		}
		cs, err := version.FileChecksum(path)
		if err != nil {
			t.Fatalf("FileChecksum: %v", err)
		}
		rel, _ := filepath.Rel(env.dir, path)
		installedFiles[rel] = cs
	}

	info := &version.Info{HeroVersion: "0.9.0", InstalledFiles: installedFiles}
	if err := version.Write(env.heroDir, info); err != nil {
		t.Fatalf("Write version: %v", err)
	}
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	output, err := runCmd("upgrade")
	if err != nil {
		t.Fatalf("upgrade error: %v", err)
	}

	// Output should mention both targets.
	for _, want := range []string{"opencode", "claude", "across 2 targets"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}

	// Both files should be rewritten to v2 content.
	for _, dir := range []string{".opencode/agents", ".claude/agents"} {
		path := filepath.Join(env.dir, dir, "hero.md")
		data, _ := os.ReadFile(path)
		if string(data) != "# Test Hero Agent v2\n" {
			t.Errorf("%s not upgraded:\n%s", dir, data)
		}
	}
}

func TestUpgradeNarrowsViaTargetFlag(t *testing.T) {
	env := newTestEnv(t)
	upgradeContentFS = testContentFS()
	defer func() { upgradeContentFS = nil }()

	// Pre-create both targets with the SAME old content, stamp the
	// checksums so neither shows as "customized", then upgrade only
	// claude. opencode should be untouched.
	installedFiles := map[string]string{}
	for _, dir := range []string{".opencode/agents", ".claude/agents"} {
		if err := os.MkdirAll(filepath.Join(env.dir, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(env.dir, dir, "hero.md")
		if err := os.WriteFile(path, []byte("# old\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cs, _ := version.FileChecksum(path)
		rel, _ := filepath.Rel(env.dir, path)
		installedFiles[rel] = cs
	}

	info := &version.Info{HeroVersion: "0.9.0", InstalledFiles: installedFiles}
	if err := version.Write(env.heroDir, info); err != nil {
		t.Fatal(err)
	}
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	if _, err := runCmd("upgrade", "--target", "claude"); err != nil {
		t.Fatalf("upgrade --target claude: %v", err)
	}

	claude, _ := os.ReadFile(filepath.Join(env.dir, ".claude/agents/hero.md"))
	opencode, _ := os.ReadFile(filepath.Join(env.dir, ".opencode/agents/hero.md"))

	if string(claude) != "# Test Hero Agent v2\n" {
		t.Errorf("claude target not upgraded:\n%s", claude)
	}
	if string(opencode) != "# old\n" {
		t.Errorf("opencode target was modified despite narrow:\n%s", opencode)
	}
}

func TestUpgradeRejectsUnknownTarget(t *testing.T) {
	env := newTestEnv(t)
	upgradeContentFS = testContentFS()
	defer func() { upgradeContentFS = nil }()

	if err := os.MkdirAll(filepath.Join(env.dir, ".opencode/agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := version.StampInit(env.heroDir, "0.9.0"); err != nil {
		t.Fatal(err)
	}
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	_, err := runCmd("upgrade", "--target", "vimscript")
	if err == nil {
		t.Fatal("expected error for unknown --target value")
	}
	if !strings.Contains(err.Error(), "unknown --target") {
		t.Errorf("error message: %v", err)
	}
}

func TestDetectInstalledTargets_DedupsAndOrdersStably(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{".opencode", ".claude"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := detectInstalledTargets(dir, nil)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %v", len(got), got)
	}
	if string(got[0]) != "opencode" || string(got[1]) != "claude" {
		t.Errorf("order = %v, want [opencode claude]", got)
	}
}

func TestDetectInstalledTargets_FallsBackToVersionInfoWhenEmpty(t *testing.T) {
	dir := t.TempDir() // no install dirs
	info := &version.Info{
		LastInstall: &version.InstallRecord{Target: "claude"},
	}
	got := detectInstalledTargets(dir, info)
	if len(got) != 1 || string(got[0]) != "claude" {
		t.Errorf("got %v, want [claude] from LastInstall fallback", got)
	}
}

func TestUpgradeRejectsDowngrade(t *testing.T) {
	env := newTestEnv(t)

	// Stamp workspace at 1.0.0, binary at 0.9.0 (older)
	if err := version.StampInit(env.heroDir, "1.0.0"); err != nil {
		t.Fatalf("StampInit: %v", err)
	}
	rootCmd.Version = "0.9.0"
	defer func() { rootCmd.Version = "" }()

	_, err := runCmd("upgrade")
	if err == nil {
		t.Fatal("upgrade should reject downgrade attempts")
	}
	if !strings.Contains(err.Error(), "cannot downgrade") {
		t.Errorf("error should mention downgrade rejection: %v", err)
	}

	// Version file should NOT have been updated
	info, _ := version.Read(env.heroDir)
	if info.HeroVersion != "1.0.0" {
		t.Errorf("version should remain 1.0.0, got %s", info.HeroVersion)
	}
}
