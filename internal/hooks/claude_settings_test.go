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

// TestInstall_MissingSettingsFile_CreatesIt — install on a fresh root
// with no .claude/ at all should create the file (and its directory)
// from scratch. The first existing test already covers this implicitly;
// this one asserts the directory was newly created.
func TestInstall_MissingSettingsFile_CreatesIt(t *testing.T) {
	root := t.TempDir()
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("precondition: .claude should not yet exist (err=%v)", err)
	}
	installed, err := InstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !installed {
		t.Fatal("expected installed=true on missing-file path")
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "settings.json")); err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}
	settings := readSettings(t, root)
	entries := settings["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Errorf("expected 1 SessionStart entry; got %d", len(entries))
	}
}

// TestInstall_InvalidJSON_ErrorsCleanlyNoMutate — when settings.json
// exists but isn't valid JSON, install returns an error and leaves the
// file unchanged. We must not silently overwrite a user's broken file.
func TestInstall_InvalidJSON_ErrorsCleanlyNoMutate(t *testing.T) {
	root := t.TempDir()
	original := "{ this is not valid json"
	writeSettings(t, root, original)

	installed, err := InstallClaudeCompactHandoff(root)
	if err == nil {
		t.Fatal("expected install to error on invalid JSON")
	}
	if installed {
		t.Error("expected installed=false on parse error")
	}
	data, readErr := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if readErr != nil {
		t.Fatalf("read settings.json: %v", readErr)
	}
	if string(data) != original {
		t.Errorf("settings.json was mutated despite parse error:\ngot:  %q\nwant: %q", string(data), original)
	}
}

// TestInstall_SessionStartArrayExistsNoCompact_AddsCompactEntry — when
// hooks.SessionStart already contains a user entry with a different
// matcher (e.g. "startup"), our compact entry must be appended without
// touching the existing one.
func TestInstall_SessionStartArrayExistsNoCompact_AddsCompactEntry(t *testing.T) {
	root := t.TempDir()
	seed := `{
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
	writeSettings(t, root, seed)
	installed, err := InstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !installed {
		t.Fatal("expected installed=true")
	}
	settings := readSettings(t, root)
	entries := settings["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected 2 SessionStart entries (startup + compact); got %d: %+v", len(entries), entries)
	}
	// Order is not contractually required; just assert both matchers present.
	matchers := map[string]bool{}
	for _, e := range entries {
		matchers[e.(map[string]any)["matcher"].(string)] = true
	}
	if !matchers["startup"] {
		t.Error("user-authored startup entry was dropped")
	}
	if !matchers["compact"] {
		t.Error("hero compact entry was not added")
	}
}

// TestInstall_CompactMatcherHasUserEntry_HeroEntryAddedAlongside — when
// a user has already authored a SessionStart{matcher:"compact"} entry
// with a *different* command, Hero's install adds its own entry
// alongside without merging or overwriting the user's.
func TestInstall_CompactMatcherHasUserEntry_HeroEntryAddedAlongside(t *testing.T) {
	root := t.TempDir()
	seed := `{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "compact",
        "hooks": [{"type": "command", "command": "my-custom-compact-tool"}]
      }
    ]
  }
}
`
	writeSettings(t, root, seed)
	installed, err := InstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !installed {
		t.Fatal("expected installed=true")
	}
	entries := readSettings(t, root)["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected 2 compact entries (user + hero); got %d: %+v", len(entries), entries)
	}
	// Find the user-authored entry and verify it survives unmodified.
	foundUser, foundHero := false, false
	for _, e := range entries {
		em := e.(map[string]any)
		inner := em["hooks"].([]any)
		cmd := inner[0].(map[string]any)["command"].(string)
		if cmd == "my-custom-compact-tool" {
			foundUser = true
			if _, marked := em[HeroMarkerField]; marked {
				t.Error("user entry must not carry hero marker")
			}
		}
		if cmd == "hero next compact-handoff --json" {
			foundHero = true
			if em[HeroMarkerField] != true {
				t.Error("hero entry must carry added_by_hero=true")
			}
		}
	}
	if !foundUser {
		t.Error("user's custom-compact-tool entry was lost")
	}
	if !foundHero {
		t.Error("hero entry was not added alongside user's")
	}
}

// TestUninstall_NoHeroEntries_IsNoOp — when no Hero-marked entries
// exist, uninstall reports (false, nil) and the file is unmodified.
func TestUninstall_NoHeroEntries_IsNoOp(t *testing.T) {
	root := t.TempDir()
	// Seed with a non-Hero SessionStart entry.
	seed := `{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "compact",
        "hooks": [{"type": "command", "command": "user-only-tool"}]
      }
    ]
  }
}
`
	writeSettings(t, root, seed)
	original, _ := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))

	removed, err := UninstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if removed {
		t.Error("expected removed=false when nothing to remove")
	}
	after, _ := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if string(after) != string(original) {
		t.Errorf("file was mutated by a no-op uninstall:\nbefore: %s\nafter:  %s", original, after)
	}
}

// TestRemoveThenReinstall_PreservesIdempotency — install → uninstall →
// install should land at the same single-entry state as a single
// install. Tests that the marker convention survives a removal cycle.
func TestRemoveThenReinstall_PreservesIdempotency(t *testing.T) {
	root := t.TempDir()

	if _, err := InstallClaudeCompactHandoff(root); err != nil {
		t.Fatalf("install#1: %v", err)
	}
	removed, err := UninstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !removed {
		t.Fatal("uninstall: expected removed=true")
	}
	installed, err := InstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("install#2: %v", err)
	}
	if !installed {
		t.Fatal("second install should treat as fresh (installed=true after uninstall)")
	}
	// Re-running install once more is a no-op.
	again, err := InstallClaudeCompactHandoff(root)
	if err != nil {
		t.Fatalf("install#3: %v", err)
	}
	if again {
		t.Error("third install should be a no-op")
	}
	// Exactly one entry survives.
	entries := readSettings(t, root)["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Errorf("expected 1 SessionStart entry after cycle; got %d", len(entries))
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
