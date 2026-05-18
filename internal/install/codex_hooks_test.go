package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWireCodexHooks_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}

	if err := wireCodexHooks(opts, &Result{}); err != nil {
		t.Fatalf("wireCodexHooks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
	if err != nil {
		t.Fatalf("hooks.json not created: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("hooks.json invalid JSON: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "hero next checkpoint") {
		t.Error("hooks.json missing hero checkpoint command")
	}
	if !strings.Contains(content, `"Stop"`) {
		t.Error("hooks.json missing Stop event")
	}
}

func TestWireCodexHooks_Idempotent(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}

	for i := 0; i < 3; i++ {
		if err := wireCodexHooks(opts, &Result{}); err != nil {
			t.Fatalf("wireCodexHooks pass %d: %v", i, err)
		}
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
	count := strings.Count(string(data), "hero next checkpoint")
	if count != 1 {
		t.Errorf("expected exactly one hero checkpoint entry, got %d", count)
	}
}

func TestWireCodexHooks_PreservesUserEntries(t *testing.T) {
	dir := t.TempDir()
	codexDir := filepath.Join(dir, ".codex")
	os.MkdirAll(codexDir, 0o755)

	existing := `{
  "hooks": {
    "Stop": [
      {"hooks": [{"type": "command", "command": "my-custom-tool --sync"}]}
    ]
  }
}`
	os.WriteFile(filepath.Join(codexDir, "hooks.json"), []byte(existing), 0o644)

	opts := Options{Mode: ModeProject, TargetDir: dir}
	if err := wireCodexHooks(opts, &Result{}); err != nil {
		t.Fatalf("wireCodexHooks: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(codexDir, "hooks.json"))
	content := string(data)

	if !strings.Contains(content, "hero next checkpoint") {
		t.Error("hero entry should be present")
	}
	if !strings.Contains(content, "my-custom-tool") {
		t.Error("user entry should be preserved")
	}
}

func TestUnwireCodexHooks_RemovesHeroOnly(t *testing.T) {
	dir := t.TempDir()
	codexDir := filepath.Join(dir, ".codex")
	os.MkdirAll(codexDir, 0o755)

	opts := Options{Mode: ModeProject, TargetDir: dir}

	// Wire first
	if err := wireCodexHooks(opts, &Result{}); err != nil {
		t.Fatalf("wireCodexHooks: %v", err)
	}

	// Then unwire
	if err := UnwireCodexHooks(opts); err != nil {
		t.Fatalf("UnwireCodexHooks: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(codexDir, "hooks.json"))
	if strings.Contains(string(data), "hero next checkpoint") {
		t.Error("hero entry should have been removed")
	}
}

func TestUnwireCodexHooks_NoFile(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}

	// Should be a no-op when hooks.json doesn't exist
	if err := UnwireCodexHooks(opts); err != nil {
		t.Fatalf("UnwireCodexHooks on missing file: %v", err)
	}
}

// TestWireCodexHooks_AddsSessionStartIngest pins the new
// SessionStart hook wiring for next-as-projection AC-7/AC-11: a
// fresh wire produces both the Stop entry (hero next checkpoint)
// and the SessionStart entry (hero next ingest --quiet).
func TestWireCodexHooks_AddsSessionStartIngest(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}

	if err := wireCodexHooks(opts, &Result{}); err != nil {
		t.Fatalf("wireCodexHooks: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
	var outer map[string]interface{}
	if err := json.Unmarshal(data, &outer); err != nil {
		t.Fatalf("hooks.json invalid JSON: %v", err)
	}
	hooks, _ := outer["hooks"].(map[string]interface{})
	sessionStart, ok := hooks["SessionStart"].([]interface{})
	if !ok || len(sessionStart) == 0 {
		t.Fatalf("SessionStart event missing or empty: %v", hooks)
	}
	entry := sessionStart[0].(map[string]interface{})
	inner := entry["hooks"].([]interface{})[0].(map[string]interface{})
	if cmd, _ := inner["command"].(string); !strings.HasPrefix(cmd, "hero next ingest") {
		t.Errorf("SessionStart command should be hero next ingest, got %q", cmd)
	}
}

func TestWireCodexHooks_NoMatcherField(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Mode: ModeProject, TargetDir: dir}

	if err := wireCodexHooks(opts, &Result{}); err != nil {
		t.Fatalf("wireCodexHooks: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
	// Codex format has no "matcher" field (unlike Claude)
	if strings.Contains(string(data), `"matcher"`) {
		t.Error("Codex hooks.json should not contain matcher field")
	}
}
