package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpsertCodexConfig_StripsPreExistingUnmanagedHeroTable is the
// regression test for the "duplicate [mcp_servers.hero] entries break
// codex parsing" bug. Pre-v0.14.2 installs and hand-written codex
// configs could leave an unmanaged [mcp_servers.hero] table on disk;
// the subsequent install added the managed block alongside it,
// producing two `[mcp_servers.hero]` tables in the same file, which
// is invalid TOML.
func TestUpsertCodexConfig_StripsPreExistingUnmanagedHeroTable(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed: unmanaged hero block left behind by an older install (or
	// hand-written by the user).
	seed := `[mcp_servers.hero]
args = ["mcp"]
command = "/old/path/to/hero"
`
	if err := os.WriteFile(configPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := upsertCodexConfig(configPath, "/new/path/to/hero", false, ""); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)

	if count := strings.Count(out, "[mcp_servers.hero]"); count != 1 {
		t.Errorf("expected exactly one [mcp_servers.hero] table, got %d:\n%s", count, out)
	}
	if !strings.Contains(out, codexMCPMarker) || !strings.Contains(out, codexMCPEndMarker) {
		t.Errorf("expected managed markers, got:\n%s", out)
	}
	if !strings.Contains(out, `"/new/path/to/hero"`) {
		t.Errorf("expected new hero path, got:\n%s", out)
	}
	if strings.Contains(out, `"/old/path/to/hero"`) {
		t.Errorf("old hero path should have been stripped, got:\n%s", out)
	}
}

// TestUpsertCodexConfig_PreservesOtherTables confirms that stripping the
// unmanaged hero table does not touch user-defined tables before or
// after it.
func TestUpsertCodexConfig_PreservesOtherTables(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}

	seed := `model = "gpt-5.5"

[mcp_servers.other]
command = "/usr/local/bin/other"

[mcp_servers.hero]
command = "/old/hero"
args = ["mcp"]

[mcp_servers.another]
command = "/usr/local/bin/another"
`
	if err := os.WriteFile(configPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := upsertCodexConfig(configPath, "/new/hero", false, ""); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)

	if !strings.Contains(out, `model = "gpt-5.5"`) {
		t.Errorf("top-level model setting was lost:\n%s", out)
	}
	if !strings.Contains(out, "[mcp_servers.other]") || !strings.Contains(out, `"/usr/local/bin/other"`) {
		t.Errorf("[mcp_servers.other] was lost:\n%s", out)
	}
	if !strings.Contains(out, "[mcp_servers.another]") || !strings.Contains(out, `"/usr/local/bin/another"`) {
		t.Errorf("[mcp_servers.another] was lost:\n%s", out)
	}
	if count := strings.Count(out, "[mcp_servers.hero]"); count != 1 {
		t.Errorf("expected exactly one [mcp_servers.hero], got %d:\n%s", count, out)
	}
	if strings.Contains(out, `"/old/hero"`) {
		t.Errorf("old hero command should be stripped:\n%s", out)
	}
}

// TestUpsertCodexConfig_IdempotentOnManagedOnly confirms that running
// upsert against a file that already has only the managed block (no
// duplicates) leaves a single managed block — just refreshing path/args.
func TestUpsertCodexConfig_IdempotentOnManagedOnly(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// First write.
	if err := upsertCodexConfig(configPath, "/hero/v1", false, ""); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	// Second write with a different path — simulates an upgrade.
	if err := upsertCodexConfig(configPath, "/hero/v2", false, ""); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)

	if count := strings.Count(out, "[mcp_servers.hero]"); count != 1 {
		t.Errorf("expected exactly one [mcp_servers.hero] after re-upsert, got %d:\n%s", count, out)
	}
	if !strings.Contains(out, `"/hero/v2"`) {
		t.Errorf("expected v2 hero path, got:\n%s", out)
	}
	if strings.Contains(out, `"/hero/v1"`) {
		t.Errorf("v1 hero path should be gone, got:\n%s", out)
	}
}

// TestUpsertCodexConfig_EmptyFileWritesCleanBlock confirms the fresh-install
// path produces a clean single managed block.
func TestUpsertCodexConfig_EmptyFileWritesCleanBlock(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".codex", "config.toml")

	if err := upsertCodexConfig(configPath, "/hero", false, ""); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)

	if count := strings.Count(out, "[mcp_servers.hero]"); count != 1 {
		t.Errorf("expected one [mcp_servers.hero], got %d:\n%s", count, out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), codexMCPMarker) {
		t.Errorf("expected file to start with managed marker, got:\n%s", out)
	}
}

// TestStripUnmanagedCodexHeroTable_LeavesManagedBlockIntact is a focused
// unit test for the strip helper — ensures it does not touch the managed
// block even though the managed block also contains the [mcp_servers.hero]
// table header.
func TestStripUnmanagedCodexHeroTable_LeavesManagedBlockIntact(t *testing.T) {
	input := codexMCPMarker + "\n[mcp_servers.hero]\ncommand = \"/hero\"\nargs = [\"mcp\"]\n" + codexMCPEndMarker + "\n"

	out := stripUnmanagedCodexHeroTable(input)

	if out != input {
		t.Errorf("managed block was modified by strip helper.\nin:  %q\nout: %q", input, out)
	}
}

// TestStripUnmanagedCodexHeroTable_StripsTrailingFields confirms that
// removal of an unmanaged hero table includes its key/value lines and
// stops at the next table header.
func TestStripUnmanagedCodexHeroTable_StripsTrailingFields(t *testing.T) {
	input := `[mcp_servers.hero]
args = ["mcp"]
command = "/old/hero"
startup_timeout_sec = 30

[mcp_servers.other]
command = "/other"
`

	out := stripUnmanagedCodexHeroTable(input)

	if strings.Contains(out, "[mcp_servers.hero]") {
		t.Errorf("unmanaged hero table not stripped:\n%s", out)
	}
	if strings.Contains(out, "/old/hero") {
		t.Errorf("hero command line should be stripped:\n%s", out)
	}
	if strings.Contains(out, "startup_timeout_sec") {
		t.Errorf("hero-table key/value lines should be stripped:\n%s", out)
	}
	if !strings.Contains(out, "[mcp_servers.other]") || !strings.Contains(out, "/other") {
		t.Errorf("subsequent table should be preserved:\n%s", out)
	}
}
