package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// preExistingSettings is a settings.json mirroring the live Hero repo's
// own .claude/settings.json layout — PreCompact + Stop entries plus a
// permissions block. The compact-handoff installer must add its own
// entry without disturbing any of these.
const preExistingSettings = `{
  "hooks": {
    "PreCompact": [
      {
        "hooks": [
          {
            "command": "hero next checkpoint --quiet",
            "type": "command"
          }
        ],
        "matcher": ""
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "command": "hero next checkpoint --quiet",
            "type": "command"
          }
        ],
        "matcher": ""
      }
    ]
  },
  "permissions": {
    "allow": [
      "Bash(*)",
      "Edit(*)"
    ]
  }
}
`

func writeSettings(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readSettings(t *testing.T, root string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse settings.json: %v\n%s", err, data)
	}
	return out
}

func TestInstallClaudeCompactHandoff_FreshRepoCreatesFile(t *testing.T) {
	root := t.TempDir()
	installed, err := InstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !installed {
		t.Fatal("expected installed=true on fresh repo")
	}
	settings := readSettings(t, root)
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks key missing: %#v", settings)
	}
	sessionStart, ok := hooks["SessionStart"].([]any)
	if !ok || len(sessionStart) != 1 {
		t.Fatalf("SessionStart entry missing: %#v", hooks)
	}
	entry := sessionStart[0].(map[string]any)
	if entry["matcher"] != "compact" {
		t.Errorf("matcher = %v", entry["matcher"])
	}
	if entry[HeroMarkerField] != true {
		t.Errorf("added_by_hero marker missing: %#v", entry)
	}
	inner := entry["hooks"].([]any)
	if cmd := inner[0].(map[string]any)["command"]; cmd != "hero next compact-handoff --json" {
		t.Errorf("unexpected command: %v", cmd)
	}
}

func TestInstallClaudeCompactHandoff_PreservesExistingEntries(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, root, preExistingSettings)
	if _, err := InstallClaudeCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	settings := readSettings(t, root)
	hooks := settings["hooks"].(map[string]any)

	// PreCompact + Stop entries must be intact and unmodified.
	for _, ev := range []string{"PreCompact", "Stop"} {
		entries := hooks[ev].([]any)
		if len(entries) != 1 {
			t.Errorf("%s entry count = %d, want 1", ev, len(entries))
			continue
		}
		entry := entries[0].(map[string]any)
		if _, marked := entry[HeroMarkerField]; marked {
			t.Errorf("%s entry should not carry hero marker (we did not install it)", ev)
		}
		inner := entry["hooks"].([]any)
		cmd := inner[0].(map[string]any)["command"]
		if cmd != "hero next checkpoint --quiet" {
			t.Errorf("%s command = %v", ev, cmd)
		}
	}

	// Permissions block must survive.
	perms, ok := settings["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permissions missing: %#v", settings)
	}
	if allow, ok := perms["allow"].([]any); !ok || len(allow) != 2 {
		t.Errorf("permissions.allow lost: %#v", perms)
	}
}

func TestInstallClaudeCompactHandoff_Idempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := InstallClaudeCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	second, err := InstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if second {
		t.Fatal("second install should be a no-op (installed=false)")
	}
	// Verify only one entry, not two.
	settings := readSettings(t, root)
	entries := settings["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Errorf("SessionStart entry count = %d, want 1", len(entries))
	}
}

func TestUninstallClaudeCompactHandoff_RemovesOnlyHeroEntry(t *testing.T) {
	root := t.TempDir()
	// Seed with both a user-authored SessionStart entry and the Hero one.
	custom := `{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [{"type": "command", "command": "echo hi"}]
      }
    ]
  }
}
`
	writeSettings(t, root, custom)
	if _, err := InstallClaudeCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	// Should now have 2 entries.
	if entries := readSettings(t, root)["hooks"].(map[string]any)["SessionStart"].([]any); len(entries) != 2 {
		t.Fatalf("expected 2 SessionStart entries; got %d", len(entries))
	}

	removed, err := UninstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !removed {
		t.Fatal("expected removed=true")
	}
	// Only the user-authored entry should remain.
	entries := readSettings(t, root)["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected 1 remaining entry; got %d", len(entries))
	}
	if matcher := entries[0].(map[string]any)["matcher"]; matcher != "startup" {
		t.Errorf("wrong entry remained: %v", matcher)
	}
}

func TestClaudeCompactHandoffStatus(t *testing.T) {
	root := t.TempDir()
	ok, err := ClaudeCompactHandoffStatus(root)
	if err != nil {
		t.Fatalf("status before install: %v", err)
	}
	if ok {
		t.Fatal("expected status=false on fresh repo")
	}
	if _, err := InstallClaudeCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	ok, err = ClaudeCompactHandoffStatus(root)
	if err != nil {
		t.Fatalf("status after install: %v", err)
	}
	if !ok {
		t.Fatal("expected status=true after install")
	}
}

func TestSettingsRemainValidJSONAfterInstall(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, root, preExistingSettings)
	if _, err := InstallClaudeCompactHandoff(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("post-install settings.json is not valid JSON: %v\n%s", err, data)
	}
}
