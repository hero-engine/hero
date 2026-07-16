package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testExecutable returns the running test binary's path with symlinks
// resolved — the value findHeroBinary must produce.
func testExecutable(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe
}

// TestFindHeroBinary_PrefersRunningBinaryOverPATH is the wrong-hero guard
// (codex-mcp-binary-path-resolution): a decoy `hero` earlier on PATH must
// NOT win over the binary that is actually running the install.
func TestFindHeroBinary_PrefersRunningBinaryOverPATH(t *testing.T) {
	decoyDir := t.TempDir()
	decoy := filepath.Join(decoyDir, "hero")
	if err := os.WriteFile(decoy, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", decoyDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := findHeroBinary()
	if err != nil {
		t.Fatalf("findHeroBinary: %v", err)
	}

	want := testExecutable(t)
	if got != want {
		t.Errorf("findHeroBinary = %q, want the running binary %q", got, want)
	}
	if got == decoy {
		t.Errorf("findHeroBinary resolved the decoy PATH hero %q", decoy)
	}
}

// TestFindHeroBinary_FallsBackToPATH — when os.Executable fails, PATH
// lookup is the fallback.
func TestFindHeroBinary_FallsBackToPATH(t *testing.T) {
	origExec := osExecutable
	osExecutable = func() (string, error) { return "", errors.New("no executable") }
	t.Cleanup(func() { osExecutable = origExec })

	decoyDir := t.TempDir()
	decoy := filepath.Join(decoyDir, "hero")
	if err := os.WriteFile(decoy, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", decoyDir)

	got, err := findHeroBinary()
	if err != nil {
		t.Fatalf("findHeroBinary: %v", err)
	}
	if got != decoy {
		t.Errorf("findHeroBinary = %q, want PATH fallback %q", got, decoy)
	}
}

// TestFindHeroBinary_ErrorWhenUnresolvable — when neither the running
// binary nor PATH resolves, the failure surfaces as an error instead of
// the old silent `"hero", nil` fallback.
func TestFindHeroBinary_ErrorWhenUnresolvable(t *testing.T) {
	origExec := osExecutable
	origLook := execLookPath
	osExecutable = func() (string, error) { return "", errors.New("no executable") }
	execLookPath = func(string) (string, error) { return "", errors.New("not on PATH") }
	t.Cleanup(func() {
		osExecutable = origExec
		execLookPath = origLook
	})

	got, err := findHeroBinary()
	if err == nil {
		t.Fatalf("findHeroBinary = %q, nil; want error when resolution fails", got)
	}
}

// TestRegisterMCPCodex_MigratesProjectBlockToUserLayer — a project-mode
// codex install finding Hero's managed block in the project's
// .codex/config.toml removes exactly that span (user bytes outside it
// untouched) and writes the block to ~/.codex/config.toml instead.
func TestRegisterMCPCodex_MigratesProjectBlockToUserLayer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	targetDir := t.TempDir()

	projectConfig := filepath.Join(targetDir, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(projectConfig), 0o755); err != nil {
		t.Fatal(err)
	}

	userBefore := "model = \"gpt-5.5\"\n\n"
	heroBlock := codexMCPMarker + "\n[mcp_servers.hero]\ncommand = \"/old/hero\"\nargs = [\"mcp\"]\n" + codexMCPEndMarker + "\n"
	userAfter := "\n[mcp_servers.other]\ncommand = \"/other\"\n"
	if err := os.WriteFile(projectConfig, []byte(userBefore+heroBlock+userAfter), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{Mode: ModeProject, TargetDir: targetDir}
	if err := RegisterMCP(TargetCodex, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	// Project file: hero span gone, user content byte-identical.
	got, err := os.ReadFile(projectConfig)
	if err != nil {
		t.Fatalf("project config must be left in place: %v", err)
	}
	if want := userBefore + userAfter; string(got) != want {
		t.Errorf("project config after migration:\n%q\nwant user content byte-identical:\n%q", got, want)
	}

	// User layer carries the block, pointed at the running binary.
	userData, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("~/.codex/config.toml not written: %v", err)
	}
	if !strings.Contains(string(userData), "[mcp_servers.hero]") {
		t.Errorf("user layer missing hero block:\n%s", userData)
	}
	if !strings.Contains(string(userData), fmt.Sprintf("command = %q", testExecutable(t))) {
		t.Errorf("user layer command should be the running binary, got:\n%s", userData)
	}
}

// TestRegisterMCPCodex_MigrationBlockOnlyFileLeftInPlace — when the
// project .codex/config.toml contains ONLY Hero's block, migration leaves
// the (now empty/whitespace) file in place. Deleting a git-tracked file
// from an installer would be a surprise.
func TestRegisterMCPCodex_MigrationBlockOnlyFileLeftInPlace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	targetDir := t.TempDir()

	projectConfig := filepath.Join(targetDir, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(projectConfig), 0o755); err != nil {
		t.Fatal(err)
	}
	heroBlock := codexMCPMarker + "\n[mcp_servers.hero]\ncommand = \"/old/hero\"\nargs = [\"mcp\"]\n" + codexMCPEndMarker + "\n"
	if err := os.WriteFile(projectConfig, []byte(heroBlock), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{Mode: ModeProject, TargetDir: targetDir}
	if err := RegisterMCP(TargetCodex, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	got, err := os.ReadFile(projectConfig)
	if err != nil {
		t.Fatalf("block-only project config must be left in place, not deleted: %v", err)
	}
	if strings.TrimSpace(string(got)) != "" {
		t.Errorf("expected only whitespace left after migration, got:\n%q", got)
	}

	// Second run: idempotent — no block reappears in the project file,
	// user layer still has exactly one block.
	if err := RegisterMCP(TargetCodex, opts); err != nil {
		t.Fatalf("RegisterMCP second run: %v", err)
	}
	got2, err := os.ReadFile(projectConfig)
	if err != nil {
		t.Fatalf("project config missing after second run: %v", err)
	}
	if string(got2) != string(got) {
		t.Errorf("project config changed on second run:\n%q\nwas:\n%q", got2, got)
	}
	userData, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if count := strings.Count(string(userData), "[mcp_servers.hero]"); count != 1 {
		t.Errorf("expected exactly one hero block in user layer after two runs, got %d:\n%s", count, userData)
	}
}

// TestRegisterMCP_CommandPointsAtRunningBinary_AllTargets is the
// cross-target guard: every MCP-writing target must wire the command to
// an existing executable — specifically the binary running the install.
func TestRegisterMCP_CommandPointsAtRunningBinary_AllTargets(t *testing.T) {
	want := testExecutable(t)

	cases := []struct {
		target  Target
		command func(t *testing.T, home, targetDir string) string
	}{
		{TargetCursor, func(t *testing.T, home, targetDir string) string {
			return mcpServersCommand(t, filepath.Join(targetDir, ".cursor", "mcp.json"))
		}},
		{TargetClaude, func(t *testing.T, home, targetDir string) string {
			return mcpServersCommand(t, filepath.Join(targetDir, ".mcp.json"))
		}},
		{TargetOpenCode, func(t *testing.T, home, targetDir string) string {
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
		{TargetCodex, func(t *testing.T, home, targetDir string) string {
			data, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "command = ") {
					var cmd string
					if _, err := fmt.Sscanf(line, "command = %q", &cmd); err != nil {
						t.Fatalf("parsing %q: %v", line, err)
					}
					return cmd
				}
			}
			t.Fatalf("no command line in codex config:\n%s", data)
			return ""
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

			cmd := tc.command(t, home, targetDir)
			if cmd != want {
				t.Errorf("%s command = %q, want running binary %q", tc.target, cmd, want)
			}
			if info, err := os.Stat(cmd); err != nil {
				t.Errorf("%s command %q does not exist: %v", tc.target, cmd, err)
			} else if info.Mode()&0o111 == 0 {
				t.Errorf("%s command %q is not executable", tc.target, cmd)
			}
		})
	}
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
