package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// preExistingCodexSettings is a hooks.json mirroring a plausible
// pre-existing user setup: a Stop hook unrelated to Hero. The
// compact-handoff installer must add its own entry without disturbing
// the user's.
const preExistingCodexSettings = `{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "echo user-stop-hook"
          }
        ]
      }
    ]
  }
}
`

func writeCodexSettingsTest(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readCodexSettingsTest(t *testing.T, root string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".codex", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse hooks.json: %v\n%s", err, data)
	}
	return out
}

// isolateHome points HOME at a fresh temp dir so the global
// ~/.codex/config.toml lookup is sandboxed per-test.
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeCodexFeatureFlag(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCodexCompactHandoffSupported(t *testing.T) {
	if !CodexCompactHandoffSupported() {
		t.Fatal("expected CodexCompactHandoffSupported=true")
	}
}

func TestInstallCodex_FreshRepoCreatesFile(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	installed, err := InstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !installed {
		t.Fatal("expected installed=true on fresh repo")
	}
	settings := readCodexSettingsTest(t, root)
	hooksMap, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks key missing: %#v", settings)
	}
	sessionStart, ok := hooksMap["SessionStart"].([]any)
	if !ok || len(sessionStart) != 1 {
		t.Fatalf("SessionStart entry missing: %#v", hooksMap)
	}
	entry := sessionStart[0].(map[string]any)
	if entry["matcher"] != "compact" {
		t.Errorf("matcher = %v", entry["matcher"])
	}
	inner, _ := entry["hooks"].([]any)
	if len(inner) != 1 {
		t.Fatalf("expected 1 inner hook; got %d", len(inner))
	}
	hm := inner[0].(map[string]any)
	if hm[HeroMarkerField] != true {
		t.Errorf("added_by_hero marker missing: %#v", hm)
	}
	if hm["command"] != "hero next compact-handoff --json" {
		t.Errorf("unexpected command: %v", hm["command"])
	}
	if hm["type"] != "command" {
		t.Errorf("unexpected type: %v", hm["type"])
	}
	// timeout is JSON-decoded as float64
	if to, ok := hm["timeout"].(float64); !ok || int(to) != 30 {
		t.Errorf("expected timeout=30; got %#v", hm["timeout"])
	}
}

func TestInstallCodex_PreservesExistingEntries(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	writeCodexSettingsTest(t, root, preExistingCodexSettings)
	if _, err := InstallCodexCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	settings := readCodexSettingsTest(t, root)
	hooksMap := settings["hooks"].(map[string]any)

	stop := hooksMap["Stop"].([]any)
	if len(stop) != 1 {
		t.Fatalf("Stop entries count = %d, want 1", len(stop))
	}
	stopEntry := stop[0].(map[string]any)
	stopInner := stopEntry["hooks"].([]any)
	if cmd := stopInner[0].(map[string]any)["command"]; cmd != "echo user-stop-hook" {
		t.Errorf("Stop command lost: %v", cmd)
	}
	if _, marked := stopInner[0].(map[string]any)[HeroMarkerField]; marked {
		t.Error("user Stop entry must not carry Hero marker")
	}
}

func TestInstallCodex_Idempotent(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	if _, err := InstallCodexCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	second, err := InstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if second {
		t.Fatal("second install should be a no-op (installed=false)")
	}
	settings := readCodexSettingsTest(t, root)
	entries := settings["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Errorf("SessionStart entries = %d, want 1", len(entries))
	}
	inner := entries[0].(map[string]any)["hooks"].([]any)
	if len(inner) != 1 {
		t.Errorf("inner hooks = %d, want 1", len(inner))
	}
}

func TestUninstallCodex_RemovesOnlyHeroEntry(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	// Seed a user-authored SessionStart{startup} entry, then install Hero.
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
	writeCodexSettingsTest(t, root, seed)
	if _, err := InstallCodexCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	if entries := readCodexSettingsTest(t, root)["hooks"].(map[string]any)["SessionStart"].([]any); len(entries) != 2 {
		t.Fatalf("expected 2 SessionStart entries; got %d", len(entries))
	}

	removed, err := UninstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !removed {
		t.Fatal("expected removed=true")
	}
	entries := readCodexSettingsTest(t, root)["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected 1 remaining entry; got %d", len(entries))
	}
	if matcher := entries[0].(map[string]any)["matcher"]; matcher != "startup" {
		t.Errorf("wrong entry remained: %v", matcher)
	}
}

func TestCodexStatus_ReportsCorrectly(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	ok, err := CodexCompactHandoffStatus(root)
	if err != nil {
		t.Fatalf("status before install: %v", err)
	}
	if ok {
		t.Fatal("expected status=false on fresh repo")
	}
	if _, err := InstallCodexCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	ok, err = CodexCompactHandoffStatus(root)
	if err != nil {
		t.Fatalf("status after install: %v", err)
	}
	if !ok {
		t.Fatal("expected status=true after install")
	}
}

func TestInstallCodex_MissingFeatureFlagWarns(t *testing.T) {
	home := isolateHome(t)
	// Write a config.toml without the flag.
	writeCodexFeatureFlag(t, home, "[features]\nsome_other_flag = true\n")
	if CodexFeatureFlagEnabled() {
		t.Error("expected CodexFeatureFlagEnabled=false when flag absent")
	}
	// And install should still succeed.
	root := t.TempDir()
	installed, err := InstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !installed {
		t.Fatal("install must succeed even when feature flag absent")
	}
}

func TestInstallCodex_FeatureFlagPresentNoWarn(t *testing.T) {
	home := isolateHome(t)
	writeCodexFeatureFlag(t, home, "[features]\ncodex_hooks = true\n")
	if !CodexFeatureFlagEnabled() {
		t.Error("expected CodexFeatureFlagEnabled=true when flag set")
	}
}

func TestCodexFeatureFlag_CommentedOutIsFalse(t *testing.T) {
	home := isolateHome(t)
	writeCodexFeatureFlag(t, home, "[features]\n# codex_hooks = true\n")
	if CodexFeatureFlagEnabled() {
		t.Error("expected CodexFeatureFlagEnabled=false when flag commented out")
	}
}

func TestCodexFeatureFlag_FalseValueIsFalse(t *testing.T) {
	home := isolateHome(t)
	writeCodexFeatureFlag(t, home, "[features]\ncodex_hooks = false\n")
	if CodexFeatureFlagEnabled() {
		t.Error("expected CodexFeatureFlagEnabled=false when flag explicitly false")
	}
}

func TestCodexFeatureFlag_MissingConfigIsFalse(t *testing.T) {
	isolateHome(t)
	// Don't write config.toml at all.
	if CodexFeatureFlagEnabled() {
		t.Error("expected CodexFeatureFlagEnabled=false when config.toml absent")
	}
}

func TestInstallCodex_InvalidJSONErrorsCleanlyNoMutate(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	original := "{ this is not valid json"
	writeCodexSettingsTest(t, root, original)

	installed, err := InstallCodexCompactHandoff(root)
	if err == nil {
		t.Fatal("expected install to error on invalid JSON")
	}
	if installed {
		t.Error("expected installed=false on parse error")
	}
	data, readErr := os.ReadFile(filepath.Join(root, ".codex", "hooks.json"))
	if readErr != nil {
		t.Fatalf("read hooks.json: %v", readErr)
	}
	if string(data) != original {
		t.Errorf("hooks.json was mutated despite parse error:\ngot:  %q\nwant: %q", string(data), original)
	}
}

func TestInstallCodex_SessionStartArrayExistsNoCompact_AddsCompactEntry(t *testing.T) {
	isolateHome(t)
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
	writeCodexSettingsTest(t, root, seed)
	installed, err := InstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !installed {
		t.Fatal("expected installed=true")
	}
	entries := readCodexSettingsTest(t, root)["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected 2 SessionStart entries (startup + compact); got %d", len(entries))
	}
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

func TestInstallCodex_CompactMatcherHasUserEntry_HeroEntryAddedAlongside(t *testing.T) {
	isolateHome(t)
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
	writeCodexSettingsTest(t, root, seed)
	installed, err := InstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !installed {
		t.Fatal("expected installed=true")
	}
	entries := readCodexSettingsTest(t, root)["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected single compact matcher entry; got %d", len(entries))
	}
	inner := entries[0].(map[string]any)["hooks"].([]any)
	if len(inner) != 2 {
		t.Fatalf("expected 2 inner hooks (user + hero) under shared matcher; got %d", len(inner))
	}
	foundUser, foundHero := false, false
	for _, h := range inner {
		hm := h.(map[string]any)
		cmd, _ := hm["command"].(string)
		if cmd == "my-custom-compact-tool" {
			foundUser = true
			if _, marked := hm[HeroMarkerField]; marked {
				t.Error("user inner hook must not carry hero marker")
			}
		}
		if cmd == "hero next compact-handoff --json" {
			foundHero = true
			if hm[HeroMarkerField] != true {
				t.Error("hero inner hook must carry added_by_hero=true")
			}
		}
	}
	if !foundUser {
		t.Error("user's custom-compact-tool hook was lost")
	}
	if !foundHero {
		t.Error("hero hook was not added alongside user's")
	}
}

func TestUninstallCodex_NoHeroEntries_IsNoOp(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
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
	writeCodexSettingsTest(t, root, seed)
	original, _ := os.ReadFile(filepath.Join(root, ".codex", "hooks.json"))

	removed, err := UninstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if removed {
		t.Error("expected removed=false when nothing to remove")
	}
	after, _ := os.ReadFile(filepath.Join(root, ".codex", "hooks.json"))
	if string(after) != string(original) {
		t.Errorf("file was mutated by a no-op uninstall:\nbefore: %s\nafter:  %s", original, after)
	}
}

func TestRemoveThenReinstallCodex_PreservesIdempotency(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()

	if _, err := InstallCodexCompactHandoff(root); err != nil {
		t.Fatalf("install#1: %v", err)
	}
	removed, err := UninstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !removed {
		t.Fatal("uninstall: expected removed=true")
	}
	// File should be gone since hooks block emptied out.
	if _, err := os.Stat(filepath.Join(root, ".codex", "hooks.json")); !os.IsNotExist(err) {
		t.Errorf("expected hooks.json removed after full cleanup; stat err=%v", err)
	}
	installed, err := InstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("install#2: %v", err)
	}
	if !installed {
		t.Fatal("second install should treat as fresh (installed=true after uninstall)")
	}
	again, err := InstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("install#3: %v", err)
	}
	if again {
		t.Error("third install should be a no-op")
	}
	entries := readCodexSettingsTest(t, root)["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Errorf("expected 1 SessionStart entry after cycle; got %d", len(entries))
	}
}

func TestCodexSettingsRemainValidJSONAfterInstall(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	writeCodexSettingsTest(t, root, preExistingCodexSettings)
	if _, err := InstallCodexCompactHandoff(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".codex", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("post-install hooks.json is not valid JSON: %v\n%s", err, data)
	}
}

// TestUninstallCodex_PreservesUserHookInSameMatcher — when a user has
// added their own command into the same compact matcher Hero installs
// into, uninstall must drop only Hero's command and keep the matcher
// alive with the user's command intact.
func TestUninstallCodex_PreservesUserHookInSameMatcher(t *testing.T) {
	isolateHome(t)
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
	writeCodexSettingsTest(t, root, seed)
	if _, err := InstallCodexCompactHandoff(root); err != nil {
		t.Fatalf("install: %v", err)
	}
	removed, err := UninstallCodexCompactHandoff(root)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !removed {
		t.Fatal("expected removed=true")
	}
	entries := readCodexSettingsTest(t, root)["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected matcher entry retained; got %d", len(entries))
	}
	inner := entries[0].(map[string]any)["hooks"].([]any)
	if len(inner) != 1 {
		t.Fatalf("expected 1 surviving inner hook; got %d", len(inner))
	}
	if cmd := inner[0].(map[string]any)["command"]; cmd != "my-custom-compact-tool" {
		t.Errorf("wrong inner hook survived: %v", cmd)
	}
}
