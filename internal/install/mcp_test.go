package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRegisterMCP_CommandIsPortable_AllTargets is the cross-target guard:
// every MCP-writing target wires the portable `hero` command — not a
// machine-specific absolute path — into a PROJECT-level config file, so
// the wiring travels with the repo. (codex-mcp-binary-path-resolution
// first landed an absolute-path resolver + User-layer move; this replaces
// it — the block belongs in the project layer with a portable command.)
func TestRegisterMCP_CommandIsPortable_AllTargets(t *testing.T) {
	cases := []struct {
		target  Target
		command func(t *testing.T, targetDir string) string
	}{
		{TargetCursor, func(t *testing.T, targetDir string) string {
			return mcpServersCommand(t, filepath.Join(targetDir, ".cursor", "mcp.json"))
		}},
		{TargetClaude, func(t *testing.T, targetDir string) string {
			return mcpServersCommand(t, filepath.Join(targetDir, ".mcp.json"))
		}},
		{TargetOpenCode, func(t *testing.T, targetDir string) string {
			var cfg struct {
				MCP map[string]struct {
					Command []string `json:"command"`
				} `json:"mcp"`
			}
			data, err := os.ReadFile(filepath.Join(targetDir, "opencode.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &cfg); err != nil {
				t.Fatal(err)
			}
			return cfg.MCP["hero"].Command[0]
		}},
		{TargetCodex, func(t *testing.T, targetDir string) string {
			// The codex block lands in the PROJECT config, not ~/.codex.
			return codexHeroCommand(t, filepath.Join(targetDir, ".codex", "config.toml"))
		}},
	}

	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			targetDir := t.TempDir()

			if err := RegisterMCP(tc.target, Options{Mode: ModeProject, TargetDir: targetDir}); err != nil {
				t.Fatalf("RegisterMCP(%s): %v", tc.target, err)
			}

			if cmd := tc.command(t, targetDir); cmd != "hero" {
				t.Errorf("%s command = %q, want portable %q", tc.target, cmd, "hero")
			}
		})
	}
}

// codexHeroCommand extracts the `command = "..."` value from the managed
// hero block in a codex config.toml.
func codexHeroCommand(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "command = ") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "command =")), `"`)
		}
	}
	t.Fatalf("no command line in codex config:\n%s", data)
	return ""
}

// mcpServersCommand extracts mcpServers.hero.command from a
// { "mcpServers": { "hero": { "command": ... } } } JSON config.
func mcpServersCommand(t *testing.T, path string) string {
	t.Helper()
	var cfg struct {
		MCPServers map[string]struct {
			Command string `json:"command"`
		} `json:"mcpServers"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg.MCPServers["hero"].Command
}

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

	if err := upsertCodexConfig(configPath, false, ""); err != nil {
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
	if !strings.Contains(out, `command = "hero"`) {
		t.Errorf("expected portable hero command, got:\n%s", out)
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

	if err := upsertCodexConfig(configPath, false, ""); err != nil {
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
	if !strings.Contains(out, `command = "hero"`) {
		t.Errorf("expected portable hero command:\n%s", out)
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

	// Two writes — the second must not duplicate the managed block.
	if err := upsertCodexConfig(configPath, false, ""); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	if err := upsertCodexConfig(configPath, false, ""); err != nil {
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
	if !strings.Contains(out, `command = "hero"`) {
		t.Errorf("expected portable hero command, got:\n%s", out)
	}
}

// TestUpsertCodexConfig_EmptyFileWritesCleanBlock confirms the fresh-install
// path produces a clean single managed block.
func TestUpsertCodexConfig_EmptyFileWritesCleanBlock(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".codex", "config.toml")

	if err := upsertCodexConfig(configPath, false, ""); err != nil {
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
	if !strings.Contains(out, `command = "hero"`) {
		t.Errorf("expected portable hero command, got:\n%s", out)
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
