package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWireClaudeHooks_CreatesSettingsFromScratch verifies that wiring
// against a project with no existing settings.json creates one with
// our hero-managed Stop, PreCompact, and SessionStart entries.
func TestWireClaudeHooks_CreatesSettingsFromScratch(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}
	result := &Result{}

	if err := wireClaudeHooks(opts, result); err != nil {
		t.Fatalf("wireClaudeHooks: %v", err)
	}

	settings := readSettings(t, dir)
	hooks := settings["hooks"].(map[string]interface{})

	for _, ev := range []string{"Stop", "PreCompact"} {
		entries, ok := hooks[ev].([]interface{})
		if !ok {
			t.Fatalf("missing %s hook entries", ev)
		}
		if len(entries) != 1 {
			t.Errorf("%s: want 1 entry, got %d", ev, len(entries))
		}
		entry := entries[0].(map[string]interface{})
		inner := entry["hooks"].([]interface{})[0].(map[string]interface{})
		if !strings.HasPrefix(inner["command"].(string), "hero next checkpoint") {
			t.Errorf("%s: expected hero next checkpoint, got %v", ev, inner["command"])
		}
	}

	// SessionStart fires `hero next ingest` for cross-machine
	// round-trip continuity (next-as-projection AC-7/AC-11).
	sessionStart, ok := hooks["SessionStart"].([]interface{})
	if !ok || len(sessionStart) != 1 {
		t.Fatalf("missing or wrong-count SessionStart entries: %v", hooks["SessionStart"])
	}
	ssInner := sessionStart[0].(map[string]interface{})["hooks"].([]interface{})[0].(map[string]interface{})
	if !strings.HasPrefix(ssInner["command"].(string), "hero next ingest") {
		t.Errorf("SessionStart: expected hero next ingest, got %v", ssInner["command"])
	}
}

// TestWireClaudeHooks_Idempotent verifies that wiring twice produces
// exactly one hero entry per event — no duplicates.
func TestWireClaudeHooks_Idempotent(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}

	for i := 0; i < 3; i++ {
		if err := wireClaudeHooks(opts, &Result{}); err != nil {
			t.Fatalf("wireClaudeHooks pass %d: %v", i, err)
		}
	}

	settings := readSettings(t, dir)
	hooks := settings["hooks"].(map[string]interface{})

	for _, ev := range []string{"Stop", "PreCompact", "SessionStart"} {
		entries := hooks[ev].([]interface{})
		heroCount := 0
		for _, e := range entries {
			if entryIsHero(e.(map[string]interface{})) {
				heroCount++
			}
		}
		if heroCount != 1 {
			t.Errorf("%s: want 1 hero entry after 3 wires, got %d", ev, heroCount)
		}
	}
}

// TestWireClaudeHooks_PreservesUserContent verifies that wiring leaves
// the user's existing settings (permissions, other unrelated hook
// entries, top-level keys) untouched.
func TestWireClaudeHooks_PreservesUserContent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}

	original := `{
  "permissions": {"allow": ["Bash(*)"]},
  "model": "sonnet",
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [{"type": "command", "command": "echo user-stop"}]}
    ],
    "PostToolUse": [
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "echo user-edit"}]}
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := wireClaudeHooks(Options{Mode: ModeProject, TargetDir: dir}, &Result{}); err != nil {
		t.Fatalf("wireClaudeHooks: %v", err)
	}

	settings := readSettings(t, dir)

	if settings["model"] != "sonnet" {
		t.Errorf("top-level model key lost: %v", settings["model"])
	}
	if _, ok := settings["permissions"]; !ok {
		t.Errorf("permissions key lost")
	}

	hooks := settings["hooks"].(map[string]interface{})

	stopEntries := hooks["Stop"].([]interface{})
	if len(stopEntries) != 2 {
		t.Errorf("Stop: want 2 entries (1 hero + 1 user), got %d", len(stopEntries))
	}
	foundUserStop := false
	for _, e := range stopEntries {
		em := e.(map[string]interface{})
		inner := em["hooks"].([]interface{})[0].(map[string]interface{})
		if inner["command"] == "echo user-stop" {
			foundUserStop = true
		}
	}
	if !foundUserStop {
		t.Errorf("user's Stop hook was clobbered")
	}

	postToolUse := hooks["PostToolUse"].([]interface{})
	if len(postToolUse) != 1 {
		t.Errorf("PostToolUse: user entry lost, got %d entries", len(postToolUse))
	}
}

// TestUnwireClaudeHooks_RemovesHeroOnly verifies that uninstall pulls
// our entries out cleanly while leaving user entries alone.
func TestUnwireClaudeHooks_RemovesHeroOnly(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-seed with a user-owned Stop hook plus other settings.
	pre := `{
  "permissions": {"allow": ["Bash(*)"]},
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [{"type": "command", "command": "echo user"}]}
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(pre), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := wireClaudeHooks(opts, &Result{}); err != nil {
		t.Fatalf("wire: %v", err)
	}
	if err := UnwireClaudeHooks(opts); err != nil {
		t.Fatalf("unwire: %v", err)
	}

	settings := readSettings(t, dir)

	if _, ok := settings["permissions"]; !ok {
		t.Errorf("permissions lost during unwire")
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatalf("hooks key lost during unwire")
	}

	if _, ok := hooks["PreCompact"]; ok {
		t.Errorf("PreCompact event should be removed (no user entries)")
	}

	stop, ok := hooks["Stop"].([]interface{})
	if !ok || len(stop) != 1 {
		t.Fatalf("Stop should retain 1 user entry, got %v", stop)
	}
	inner := stop[0].(map[string]interface{})["hooks"].([]interface{})[0].(map[string]interface{})
	if inner["command"] != "echo user" {
		t.Errorf("user Stop entry lost: %v", inner)
	}
}

// TestUnwireClaudeHooks_NoSettingsFile verifies the no-op path: if
// settings.json doesn't exist, unwire should silently succeed.
func TestUnwireClaudeHooks_NoSettingsFile(t *testing.T) {
	dir := t.TempDir()
	if err := UnwireClaudeHooks(Options{Mode: ModeProject, TargetDir: dir}); err != nil {
		t.Errorf("unwire on missing settings: %v", err)
	}
}

// TestWireClaudePermissions_FromEmpty verifies that wiring against a
// directory with no settings.json yet creates one containing only the
// permissions allowlist (mirrors wireClaudeHooks behavior).
func TestWireClaudePermissions_FromEmpty(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}

	added, err := wireClaudePermissions(opts, &Result{})
	if err != nil {
		t.Fatalf("wireClaudePermissions: %v", err)
	}
	if !added {
		t.Fatal("expected added=true on first wire")
	}

	allow := allowlistFromSettings(t, dir)
	if len(allow) != 1 || allow[0] != "Bash(hero:*)" {
		t.Errorf("permissions.allow = %v, want [Bash(hero:*)]", allow)
	}
}

// TestEnsureClaudeHeroAllowlistIdempotent: calling the exported
// helper repeatedly must not duplicate the hero entry and must
// preserve unrelated allowlist entries.
func TestEnsureClaudeHeroAllowlistIdempotent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"permissions": {"allow": ["Bash(*)"]}}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		added, gotPath, err := EnsureClaudeHeroAllowlist(ModeProject, dir)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if gotPath != settingsPath {
			t.Errorf("call %d: path = %q, want %q", i, gotPath, settingsPath)
		}
		if i == 0 && !added {
			t.Errorf("call 0: expected added=true (entry missing)")
		}
		if i > 0 && added {
			t.Errorf("call %d: expected added=false on subsequent calls", i)
		}
	}

	allow := allowlistFromSettings(t, dir)
	wantOrdered := []string{"Bash(*)", "Bash(hero:*)"}
	if len(allow) != len(wantOrdered) {
		t.Fatalf("permissions.allow = %v, want %v", allow, wantOrdered)
	}
	for i, want := range wantOrdered {
		if allow[i] != want {
			t.Errorf("permissions.allow[%d] = %q, want %q (full=%v)", i, allow[i], want, allow)
		}
	}
}

// TestEnsureClaudeHeroAllowlist_GlobalScope verifies that ModeGlobal
// writes the allowlist into $HOME/.claude/settings.json and does not
// create any project-local .claude directory.
func TestEnsureClaudeHeroAllowlist_GlobalScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Sanity: a separate "project" dir that must NOT be touched.
	projectDir := t.TempDir()

	added, gotPath, err := EnsureClaudeHeroAllowlist(ModeGlobal, "")
	if err != nil {
		t.Fatalf("EnsureClaudeHeroAllowlist: %v", err)
	}
	if !added {
		t.Errorf("expected added=true on first call")
	}
	wantPath := filepath.Join(home, ".claude", "settings.json")
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	allow := allowlistFromSettings(t, home)
	if len(allow) != 1 || allow[0] != "Bash(hero:*)" {
		t.Errorf("permissions.allow = %v, want [Bash(hero:*)]", allow)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".claude")); !os.IsNotExist(err) {
		t.Errorf("project .claude must not exist after global trust: stat err = %v", err)
	}
}

// TestEnsureClaudeHeroAllowlist_GlobalIdempotent verifies that repeat
// global calls don't duplicate the hero entry and preserve unrelated
// user entries already in $HOME/.claude/settings.json.
func TestEnsureClaudeHeroAllowlist_GlobalIdempotent(t *testing.T) {
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

	for i := 0; i < 3; i++ {
		added, gotPath, err := EnsureClaudeHeroAllowlist(ModeGlobal, "")
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if gotPath != settingsPath {
			t.Errorf("call %d: path = %q, want %q", i, gotPath, settingsPath)
		}
		if added {
			t.Errorf("call %d: expected added=false (entry seeded)", i)
		}
	}

	allow := allowlistFromSettings(t, home)
	heroCount := 0
	for _, e := range allow {
		if e == "Bash(hero:*)" {
			heroCount++
		}
	}
	if heroCount != 1 {
		t.Errorf("Bash(hero:*) must appear exactly once, got %d (allow=%v)", heroCount, allow)
	}
	if !func() bool {
		for _, e := range allow {
			if e == "Bash(ls)" {
				return true
			}
		}
		return false
	}() {
		t.Errorf("user entry Bash(ls) was lost: %v", allow)
	}
}

// TestEnsureClaudeHeroAllowlist_GlobalPreservesOtherKeys verifies that
// global wiring touches only permissions.allow — unrelated top-level
// keys (model, hooks, permissions.deny) survive byte-equivalent.
func TestEnsureClaudeHeroAllowlist_GlobalPreservesOtherKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{
  "model": "sonnet",
  "permissions": {"allow": ["Bash(ls)"], "deny": ["Bash(rm:*)"]},
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [{"type": "command", "command": "echo user-stop"}]}
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	added, _, err := EnsureClaudeHeroAllowlist(ModeGlobal, "")
	if err != nil {
		t.Fatalf("EnsureClaudeHeroAllowlist: %v", err)
	}
	if !added {
		t.Fatal("expected added=true")
	}

	settings := readSettings(t, home)
	if settings["model"] != "sonnet" {
		t.Errorf("model key lost: %v", settings["model"])
	}
	hooks, _ := settings["hooks"].(map[string]interface{})
	stop, _ := hooks["Stop"].([]interface{})
	if len(stop) != 1 {
		t.Errorf("user Stop hook lost: %v", stop)
	}
	perms, _ := settings["permissions"].(map[string]interface{})
	if deny, _ := perms["deny"].([]interface{}); len(deny) != 1 || deny[0] != "Bash(rm:*)" {
		t.Errorf("permissions.deny clobbered: %v", deny)
	}

	allow := allowlistFromSettings(t, home)
	want := []string{"Bash(ls)", "Bash(hero:*)"}
	if len(allow) != len(want) {
		t.Fatalf("permissions.allow = %v, want %v", allow, want)
	}
	for i, w := range want {
		if allow[i] != w {
			t.Errorf("permissions.allow[%d] = %q, want %q", i, allow[i], w)
		}
	}
}

// TestEnsureClaudeHeroAllowlist_ProjectAndGlobalIndependent verifies
// project and global writes target distinct files and don't bleed into
// each other.
func TestEnsureClaudeHeroAllowlist_ProjectAndGlobalIndependent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := t.TempDir()

	addedG1, _, err := EnsureClaudeHeroAllowlist(ModeGlobal, "")
	if err != nil {
		t.Fatalf("global pass 1: %v", err)
	}
	if !addedG1 {
		t.Errorf("global pass 1: expected added=true")
	}

	addedP, _, err := EnsureClaudeHeroAllowlist(ModeProject, projectDir)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if !addedP {
		t.Errorf("project: expected added=true")
	}

	addedG2, _, err := EnsureClaudeHeroAllowlist(ModeGlobal, "")
	if err != nil {
		t.Fatalf("global pass 2: %v", err)
	}
	if addedG2 {
		t.Errorf("global pass 2: expected added=false (already present)")
	}

	if !containsHeroAllow(allowlistFromSettings(t, home)) {
		t.Errorf("global file missing Bash(hero:*)")
	}
	if !containsHeroAllow(allowlistFromSettings(t, projectDir)) {
		t.Errorf("project file missing Bash(hero:*)")
	}
}

// TestEnsureClaudeHeroAllowlist_ProjectModeRequiresDir verifies the
// guard against an empty projectDir in project mode.
func TestEnsureClaudeHeroAllowlist_ProjectModeRequiresDir(t *testing.T) {
	_, _, err := EnsureClaudeHeroAllowlist(ModeProject, "")
	if err == nil {
		t.Fatal("expected error for empty projectDir in project mode")
	}
}

func containsHeroAllow(allow []string) bool {
	for _, e := range allow {
		if e == "Bash(hero:*)" {
			return true
		}
	}
	return false
}

// TestWireClaudePermissions_PreservesOtherKeys: seed settings.json
// with unrelated top-level keys plus the user's own hook entries, then
// wire permissions. Everything except the new allowlist entry must
// survive byte-equivalent.
func TestWireClaudePermissions_PreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{
  "model": "sonnet",
  "permissions": {"allow": ["Bash(ls)"], "deny": ["Bash(rm:*)"]},
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [{"type": "command", "command": "echo user-stop"}]}
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	added, err := wireClaudePermissions(Options{Mode: ModeProject, TargetDir: dir}, &Result{})
	if err != nil {
		t.Fatalf("wireClaudePermissions: %v", err)
	}
	if !added {
		t.Fatal("expected added=true")
	}

	settings := readSettings(t, dir)
	if settings["model"] != "sonnet" {
		t.Errorf("model key lost: %v", settings["model"])
	}
	hooks, _ := settings["hooks"].(map[string]interface{})
	stop, _ := hooks["Stop"].([]interface{})
	if len(stop) != 1 {
		t.Errorf("user Stop hook lost: %v", stop)
	}
	perms, _ := settings["permissions"].(map[string]interface{})
	if deny, _ := perms["deny"].([]interface{}); len(deny) != 1 || deny[0] != "Bash(rm:*)" {
		t.Errorf("permissions.deny clobbered: %v", deny)
	}

	allow := allowlistFromSettings(t, dir)
	want := []string{"Bash(ls)", "Bash(hero:*)"}
	if len(allow) != len(want) {
		t.Fatalf("permissions.allow = %v, want %v", allow, want)
	}
	for i, w := range want {
		if allow[i] != w {
			t.Errorf("permissions.allow[%d] = %q, want %q", i, allow[i], w)
		}
	}
}

// TestInstallClaudeWritesHeroAllowlist: the full claude install path
// must end up with Bash(hero:*) in permissions.allow without the user
// having to run `hero trust claude` separately.
func TestInstallClaudeWritesHeroAllowlist(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	if _, err := Run(Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}); err != nil {
		t.Fatalf("claude install failed: %v", err)
	}

	allow := allowlistFromSettings(t, targetDir)
	found := false
	for _, e := range allow {
		if e == "Bash(hero:*)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("permissions.allow missing Bash(hero:*): %v", allow)
	}
}

func allowlistFromSettings(t *testing.T, dir string) []string {
	t.Helper()
	settings := readSettings(t, dir)
	perms, _ := settings["permissions"].(map[string]interface{})
	raw, _ := perms["allow"].([]interface{})
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func readSettings(t *testing.T, dir string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
