package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrustCodex(t *testing.T) {
	_ = newTestEnvEmpty(t)

	out, err := runCmd("trust", "codex")
	if err != nil {
		t.Fatalf("trust codex returned error: %v", err)
	}

	assertCodexTrustHint(t, out)
}

func TestTrustUnknownTarget(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("trust", "vscode")
	if err == nil {
		t.Fatal("trust vscode should fail")
	}
	if !strings.Contains(err.Error(), `unsupported trust target "vscode"`) {
		t.Fatalf("error should mention the bad target: %v", err)
	}
	if !strings.Contains(err.Error(), "codex") || !strings.Contains(err.Error(), "claude") {
		t.Fatalf("error should list both supported targets, got: %v", err)
	}
}

func TestTrustClaudeAppliesAllowlist(t *testing.T) {
	env := newTestEnvEmpty(t)

	out, err := runCmd("trust", "claude")
	if err != nil {
		t.Fatalf("trust claude returned error: %v", err)
	}

	if !strings.Contains(out, "added Bash(hero:*)") {
		t.Errorf("expected 'added' message, got:\n%s", out)
	}

	allow := readClaudeAllowlist(t, env.dir)
	if !containsString(allow, "Bash(hero:*)") {
		t.Errorf("Bash(hero:*) not in permissions.allow: %v", allow)
	}
}

func TestTrustClaudeIdempotent(t *testing.T) {
	env := newTestEnvEmpty(t)

	// Pre-seed settings.json with the entry already present plus an
	// unrelated user entry that must survive.
	settingsPath := filepath.Join(env.dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"permissions": {"allow": ["Bash(ls)", "Bash(hero:*)"]}}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd("trust", "claude")
	if err != nil {
		t.Fatalf("trust claude returned error: %v", err)
	}

	if !strings.Contains(out, "already present") {
		t.Errorf("expected 'already present' message, got:\n%s", out)
	}

	allow := readClaudeAllowlist(t, env.dir)
	heroCount := 0
	for _, e := range allow {
		if e == "Bash(hero:*)" {
			heroCount++
		}
	}
	if heroCount != 1 {
		t.Errorf("Bash(hero:*) must appear exactly once, got %d (allow=%v)", heroCount, allow)
	}
	if !containsString(allow, "Bash(ls)") {
		t.Errorf("user entry Bash(ls) was lost: %v", allow)
	}
}

func TestTrustClaudeProjectExplicit(t *testing.T) {
	env := newTestEnvEmpty(t)

	out, err := runCmd("trust", "claude", "project")
	if err != nil {
		t.Fatalf("trust claude project returned error: %v", err)
	}

	if !strings.Contains(out, "added Bash(hero:*)") {
		t.Errorf("expected 'added' message, got:\n%s", out)
	}
	if strings.Contains(out, "(global)") {
		t.Errorf("project scope must not render the (global) label, got:\n%s", out)
	}

	allow := readClaudeAllowlist(t, env.dir)
	if !containsString(allow, "Bash(hero:*)") {
		t.Errorf("Bash(hero:*) not in permissions.allow: %v", allow)
	}
}

func TestTrustClaudeGlobal(t *testing.T) {
	_ = newTestEnvEmpty(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	out, err := runCmd("trust", "claude", "global")
	if err != nil {
		t.Fatalf("trust claude global returned error: %v", err)
	}

	if !strings.Contains(out, "Claude Code (global): added") {
		t.Errorf("expected 'Claude Code (global): added' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Bash(hero:*)") {
		t.Errorf("expected Bash(hero:*) in output, got:\n%s", out)
	}

	allow := readClaudeAllowlist(t, home)
	if !containsString(allow, "Bash(hero:*)") {
		t.Errorf("Bash(hero:*) not in global permissions.allow: %v", allow)
	}
}

func TestTrustClaudeGlobalIdempotent(t *testing.T) {
	_ = newTestEnvEmpty(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"permissions": {"allow": ["Bash(ls)", "Bash(hero:*)"]}}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd("trust", "claude", "global")
	if err != nil {
		t.Fatalf("trust claude global returned error: %v", err)
	}

	if !strings.Contains(out, "already present") {
		t.Errorf("expected 'already present' message, got:\n%s", out)
	}
	if !strings.Contains(out, "Claude Code (global):") {
		t.Errorf("expected '(global)' label in idempotent path, got:\n%s", out)
	}

	allow := readClaudeAllowlist(t, home)
	heroCount := 0
	for _, e := range allow {
		if e == "Bash(hero:*)" {
			heroCount++
		}
	}
	if heroCount != 1 {
		t.Errorf("Bash(hero:*) must appear exactly once, got %d (allow=%v)", heroCount, allow)
	}
	if !containsString(allow, "Bash(ls)") {
		t.Errorf("user entry Bash(ls) was lost: %v", allow)
	}
}

func TestTrustClaudeUnknownScope(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("trust", "claude", "workspace")
	if err == nil {
		t.Fatal("trust claude workspace should fail")
	}
	if !strings.Contains(err.Error(), "scope must be 'project' or 'global'") {
		t.Fatalf("error should call out the scope contract, got: %v", err)
	}
}

func TestTrustCodexWithScopeNote(t *testing.T) {
	_ = newTestEnvEmpty(t)

	out, err := runCmd("trust", "codex", "global")
	if err != nil {
		t.Fatalf("trust codex global returned error: %v", err)
	}

	assertCodexTrustHint(t, out)
	if !strings.Contains(out, "Note: scope `global` has no effect for Codex") {
		t.Errorf("expected no-effect note, got:\n%s", out)
	}
}

func TestInstallCodexPrintsTrustHint(t *testing.T) {
	_ = newTestEnvEmpty(t)
	targetDir := t.TempDir()

	out, err := runCmd("install", "project", targetDir, "--target", "codex")
	if err != nil {
		t.Fatalf("install codex returned error: %v", err)
	}

	assertCodexTrustHint(t, out)
	if _, err := os.Stat(filepath.Join(targetDir, ".codex", "hooks.json")); err != nil {
		t.Fatalf("expected codex hooks to be installed: %v", err)
	}
}

func assertCodexTrustHint(t *testing.T, out string) {
	t.Helper()

	want := []string{
		"Codex permissions: optional one-time setup",
		"Hero cannot grant Codex permissions itself; Codex owns the approval.",
		"Please run `hero status` and request persistent approval for the `hero` command prefix.",
		"You can show this again with `hero trust codex`.",
	}
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Fatalf("expected output to contain %q, got:\n%s", s, out)
		}
	}
}

func readClaudeAllowlist(t *testing.T, projectDir string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	perms, _ := settings["permissions"].(map[string]interface{})
	rawAllow, _ := perms["allow"].([]interface{})
	out := make([]string, 0, len(rawAllow))
	for _, e := range rawAllow {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
