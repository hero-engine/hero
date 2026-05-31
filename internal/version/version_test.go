package version

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStampInit(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	if err := StampInit(heroDir, "0.2.3"); err != nil {
		t.Fatalf("StampInit: %v", err)
	}

	info, err := Read(heroDir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.HeroVersion != "0.2.3" {
		t.Errorf("HeroVersion = %q, want %q", info.HeroVersion, "0.2.3")
	}
	if info.InitializedAt.IsZero() {
		t.Error("InitializedAt should not be zero")
	}
}

func TestStampInstall(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	StampInit(heroDir, "0.2.0")

	files := map[string]string{
		"agents/engineer.md": "sha256:abc123",
		"commands/prime.md":  "sha256:def456",
	}
	if err := StampInstall(heroDir, "0.2.3", "opencode", "project", files); err != nil {
		t.Fatalf("StampInstall: %v", err)
	}

	info, err := Read(heroDir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if info.HeroVersion != "0.2.3" {
		t.Errorf("HeroVersion = %q, want %q", info.HeroVersion, "0.2.3")
	}
	if info.LastInstall == nil {
		t.Fatal("LastInstall should not be nil")
	}
	if info.LastInstall.Target != "opencode" {
		t.Errorf("Target = %q, want opencode", info.LastInstall.Target)
	}
	if info.LastInstall.Mode != "project" {
		t.Errorf("Mode = %q, want project", info.LastInstall.Mode)
	}
	if len(info.InstalledFiles) != 2 {
		t.Errorf("InstalledFiles count = %d, want 2", len(info.InstalledFiles))
	}
	if info.InstalledFiles["agents/engineer.md"] != "sha256:abc123" {
		t.Errorf("checksum mismatch for agents/engineer.md")
	}
}

func TestStampUpgrade(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	StampInit(heroDir, "0.2.0")

	if err := StampUpgrade(heroDir, "0.2.0", "0.3.0", 5, 2); err != nil {
		t.Fatalf("StampUpgrade: %v", err)
	}

	info, err := Read(heroDir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if info.HeroVersion != "0.3.0" {
		t.Errorf("HeroVersion = %q, want %q", info.HeroVersion, "0.3.0")
	}
	if info.LastUpgrade == nil {
		t.Fatal("LastUpgrade should not be nil")
	}
	if info.LastUpgrade.FromVersion != "0.2.0" {
		t.Errorf("FromVersion = %q, want 0.2.0", info.LastUpgrade.FromVersion)
	}
	if info.LastUpgrade.Updated != 5 {
		t.Errorf("Updated = %d, want 5", info.LastUpgrade.Updated)
	}
	if info.LastUpgrade.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", info.LastUpgrade.Skipped)
	}
}

func TestRead_NoFile(t *testing.T) {
	dir := t.TempDir()
	info, err := Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Error("expected nil info for missing file")
	}
}

func TestMismatch_Match(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	StampInit(heroDir, "0.2.3")

	msg := Mismatch(heroDir, "0.2.3")
	if msg != "" {
		t.Errorf("expected empty mismatch, got %q", msg)
	}
}

func TestMismatch_Different(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	StampInit(heroDir, "0.2.0")

	msg := Mismatch(heroDir, "0.3.0")
	if msg == "" {
		t.Error("expected mismatch warning")
	}
	if !contains(msg, "hero upgrade") {
		t.Errorf("warning should mention 'hero upgrade', got %q", msg)
	}
	// Regression for upgrade-strands-install-layout — make sure the
	// warning surfaces that the recommended action clears legacy layout,
	// so users grasp that ignoring it leaves Claude/Codex sessions in
	// a silently-broken state.
	if !contains(msg, "legacy install layout") {
		t.Errorf("warning should explain cleanup behaviour, got %q", msg)
	}
}

func TestMismatch_DevBuild(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	StampInit(heroDir, "0.2.0")

	// Dev builds should not warn
	if msg := Mismatch(heroDir, "dev"); msg != "" {
		t.Errorf("dev build should not warn, got %q", msg)
	}
	if msg := Mismatch(heroDir, ""); msg != "" {
		t.Errorf("empty version should not warn, got %q", msg)
	}
}

func TestMismatch_NoVersionFile(t *testing.T) {
	dir := t.TempDir()
	msg := Mismatch(dir, "0.3.0")
	if msg != "" {
		t.Errorf("missing version file should not warn, got %q", msg)
	}
}

func TestFileChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world\n"), 0o644)

	cs, err := FileChecksum(path)
	if err != nil {
		t.Fatalf("FileChecksum: %v", err)
	}
	if cs == "" {
		t.Error("expected non-empty checksum")
	}
	if !contains(cs, "sha256:") {
		t.Errorf("checksum should start with sha256:, got %q", cs)
	}

	// Same content should produce same checksum
	path2 := filepath.Join(dir, "test2.txt")
	os.WriteFile(path2, []byte("hello world\n"), 0o644)
	cs2, _ := FileChecksum(path2)
	if cs != cs2 {
		t.Errorf("same content should produce same checksum")
	}

	// Different content should differ
	path3 := filepath.Join(dir, "test3.txt")
	os.WriteFile(path3, []byte("different\n"), 0o644)
	cs3, _ := FileChecksum(path3)
	if cs == cs3 {
		t.Errorf("different content should produce different checksum")
	}
}

func TestIsFileModified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("original content"), 0o644)

	cs, _ := FileChecksum(path)

	info := &Info{
		InstalledFiles: map[string]string{
			"test.md": cs,
		},
	}

	// Unmodified
	if IsFileModified(info, "test.md", path) {
		t.Error("file should not be marked as modified")
	}

	// Modify the file
	os.WriteFile(path, []byte("changed content"), 0o644)
	if !IsFileModified(info, "test.md", path) {
		t.Error("file should be marked as modified after change")
	}

	// Unknown file
	if !IsFileModified(info, "unknown.md", path) {
		t.Error("unknown file should be treated as modified")
	}

	// Nil info
	if !IsFileModified(nil, "test.md", path) {
		t.Error("nil info should treat files as modified")
	}
}

func TestWorkspaceVersion(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")

	// No version file
	if v := WorkspaceVersion(heroDir); v != "unknown" {
		t.Errorf("WorkspaceVersion = %q, want unknown", v)
	}

	// With version file
	StampInit(heroDir, "0.2.3")
	if v := WorkspaceVersion(heroDir); v != "0.2.3" {
		t.Errorf("WorkspaceVersion = %q, want 0.2.3", v)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.2.0", "0.2.0", 0},
		{"0.3.0", "0.2.0", 1},
		{"0.2.0", "0.3.0", -1},
		{"1.0.0", "0.9.0", 1},
		{"0.9.0", "1.0.0", -1},
		{"0.2.3", "0.2.3", 0},
		{"0.2.4", "0.2.3", 1},
		{"0.2.3", "0.2.4", -1},
		{"0.3", "0.3.0", 0},
		{"0.3.0", "0.3", 0},
		{"1.0", "0.9.9", 1},
		{"0.9.9", "1.0", -1},
		{"v0.3.0", "0.2.5", 1},
		{"0.2.5", "v0.3.0", -1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestMismatch_BinaryNewer(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	StampInit(heroDir, "0.2.0")

	msg := Mismatch(heroDir, "0.3.0")
	if msg == "" {
		t.Fatal("expected mismatch warning when binary is newer")
	}
	if !contains(msg, "hero upgrade") {
		t.Errorf("binary-newer warning should mention 'hero upgrade', got %q", msg)
	}
}

func TestMismatch_BinaryOlder(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	StampInit(heroDir, "0.3.0")

	msg := Mismatch(heroDir, "0.2.5")
	if msg == "" {
		t.Fatal("expected mismatch warning when binary is older")
	}
	if !contains(msg, "downgrade detected") {
		t.Errorf("binary-older warning should mention 'downgrade detected', got %q", msg)
	}
	if contains(msg, "hero upgrade") {
		t.Errorf("binary-older warning should NOT suggest 'hero upgrade', got %q", msg)
	}
}

func TestStampInit_Idempotent(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")

	StampInit(heroDir, "0.2.0")
	info1, _ := Read(heroDir)
	initTime := info1.InitializedAt

	time.Sleep(10 * time.Millisecond)

	// Re-stamp should preserve InitializedAt
	StampInit(heroDir, "0.2.3")
	info2, _ := Read(heroDir)

	if info2.HeroVersion != "0.2.3" {
		t.Errorf("version should be updated to 0.2.3")
	}
	if !info2.InitializedAt.Equal(initTime) {
		t.Error("InitializedAt should be preserved on re-init")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
