package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// auto_sync_test.go — sibling-detection tests.
//
// The rule under test: auto-sync must only refresh harnesses Hero has
// actually installed content for. A bare harness config directory is
// created by the host tool itself (Claude Code writes
// `.claude/settings.local.json`) and by `hero init` (which writes
// `.claude/settings.json` for the SessionStart{compact} hook regardless
// of --target), so treating it as an install materialized a full
// harness the user never requested.

// mkdirAllT is a fatal-on-error MkdirAll for test setup.
func mkdirAllT(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// writeFileT is a fatal-on-error file write that creates parent dirs.
func writeFileT(t *testing.T, path, body string) {
	t.Helper()
	mkdirAllT(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// containsTarget reports whether ts includes want.
func containsTarget(ts []Target, want Target) bool {
	for _, t := range ts {
		if t == want {
			return true
		}
	}
	return false
}

// TestDetectInstalledTargetDirs_SettingsOnlyClaudeIsNotAnInstall is the
// regression test for the reported bug: `hero init` writes
// `.claude/settings.json` unconditionally, and that alone must not make
// auto-sync conclude Claude is installed.
func TestDetectInstalledTargetDirs_SettingsOnlyClaudeIsNotAnInstall(t *testing.T) {
	dir := t.TempDir()
	writeFileT(t, filepath.Join(dir, ".claude", "settings.json"), `{"hooks":{}}`)

	got, err := detectInstalledTargetDirs(dir, TargetCopilot)
	if err != nil {
		t.Fatalf("detectInstalledTargetDirs: %v", err)
	}
	if containsTarget(got, TargetClaude) {
		t.Errorf("a .claude/ holding only settings.json must not count as a claude install; got %v", got)
	}
}

// TestDetectInstalledTargetDirs_EmptyHarnessDirIsNotAnInstall covers the
// other host-tool-created shape: a bare, empty harness directory.
func TestDetectInstalledTargetDirs_EmptyHarnessDirIsNotAnInstall(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{".claude", ".opencode", ".codex", ".ai", ".grok"} {
		mkdirAllT(t, filepath.Join(dir, sub))
	}
	mkdirAllT(t, filepath.Join(dir, ".cursor", "rules"))

	got, err := detectInstalledTargetDirs(dir, "")
	if err != nil {
		t.Fatalf("detectInstalledTargetDirs: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty harness dirs must not count as installs; got %v", got)
	}
}

// TestDetectInstalledTargetDirs_ContentDirCountsAsInstall confirms the
// tightening did not break real detection: any one installed content
// directory is sufficient evidence.
func TestDetectInstalledTargetDirs_ContentDirCountsAsInstall(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    Target
	}{
		{"claude agents", filepath.Join(".claude", "agents"), TargetClaude},
		{"claude commands", filepath.Join(".claude", "commands"), TargetClaude},
		{"claude skills", filepath.Join(".claude", "skills"), TargetClaude},
		{"opencode agents", filepath.Join(".opencode", "agents"), TargetOpenCode},
		{"cursor agents", filepath.Join(".cursor", "rules", "agents"), TargetCursor},
		{"codex agents", filepath.Join(".codex", "agents"), TargetCodex},
		{"generic agents", filepath.Join(".ai", "agents"), TargetGeneric},
		{"grok skills", filepath.Join(".grok", "skills"), TargetGrok},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mkdirAllT(t, filepath.Join(dir, tc.content))

			got, err := detectInstalledTargetDirs(dir, "")
			if err != nil {
				t.Fatalf("detectInstalledTargetDirs: %v", err)
			}
			if !containsTarget(got, tc.want) {
				t.Errorf("expected %s detected from %s; got %v", tc.want, tc.content, got)
			}
		})
	}
}

// TestDetectInstalledTargetDirs_CopilotMarkerFile confirms copilot's
// evidence — a Hero-written instructions file rather than a host-tool
// directory — still detects.
func TestDetectInstalledTargetDirs_CopilotMarkerFile(t *testing.T) {
	dir := t.TempDir()
	writeFileT(t, filepath.Join(dir, ".github", "copilot-instructions.md"), "# hero\n")

	got, err := detectInstalledTargetDirs(dir, "")
	if err != nil {
		t.Fatalf("detectInstalledTargetDirs: %v", err)
	}
	if !containsTarget(got, TargetCopilot) {
		t.Errorf("expected copilot detected from copilot-instructions.md; got %v", got)
	}
}

// TestDetectInstalledTargetDirs_ExcludesRequestedTarget confirms the
// just-installed target is never returned as its own sibling.
func TestDetectInstalledTargetDirs_ExcludesRequestedTarget(t *testing.T) {
	dir := t.TempDir()
	mkdirAllT(t, filepath.Join(dir, ".claude", "agents"))
	mkdirAllT(t, filepath.Join(dir, ".opencode", "agents"))

	got, err := detectInstalledTargetDirs(dir, TargetClaude)
	if err != nil {
		t.Fatalf("detectInstalledTargetDirs: %v", err)
	}
	if containsTarget(got, TargetClaude) {
		t.Errorf("excluded target must not be returned; got %v", got)
	}
	if !containsTarget(got, TargetOpenCode) {
		t.Errorf("expected opencode still detected; got %v", got)
	}
}

// TestAutoSync_SettingsOnlyClaudeDoesNotInstallClaude is the end-to-end
// proof: installing copilot into a project whose only Claude artifact is
// the init-written settings.json must not produce a Claude install.
func TestAutoSync_SettingsOnlyClaudeDoesNotInstallClaude(t *testing.T) {
	h := newInstallHarness(t)
	mkdirAllT(t, filepath.Join(h.TargetDir, ".hero"))
	// Exactly what `hero init` leaves behind.
	writeFileT(t, filepath.Join(h.TargetDir, ".claude", "settings.json"),
		`{"hooks":{"SessionStart":[]}}`)

	h.Run(TargetCopilot, func(o *Options) { o.AutoSyncTargets = true })

	h.mustExist(".github/copilot-instructions.md")
	h.mustNotExist("CLAUDE.md")
	h.mustNotExist(".claude/agents")
	h.mustNotExist(".claude/commands")
	h.mustNotExist(".claude/skills")

	// The init-written settings file itself must survive untouched.
	h.mustExist(".claude/settings.json")

	st, err := ReadInstallState(h.TargetDir)
	if err != nil {
		t.Fatalf("ReadInstallState: %v", err)
	}
	if _, ok := st.Targets[string(TargetClaude)]; ok {
		keys := make([]string, 0, len(st.Targets))
		for k := range st.Targets {
			keys = append(keys, k)
		}
		t.Errorf("claude must not be recorded in install-state; targets=%v", keys)
	}
	if _, ok := st.Targets[string(TargetCopilot)]; !ok {
		t.Error("copilot should be recorded in install-state")
	}
}

// TestAutoSync_RealClaudeInstallStillSyncs is the counterweight: a
// genuinely installed Claude harness must still be refreshed, so the
// drift-prevention purpose of auto-sync survives the tightening.
func TestAutoSync_RealClaudeInstallStillSyncs(t *testing.T) {
	h := newInstallHarness(t)
	mkdirAllT(t, filepath.Join(h.TargetDir, ".hero"))

	h.Run(TargetClaude, func(o *Options) { o.AutoSyncTargets = true })
	h.mustBeRegularFile(".claude/agents/engineer.md")

	// Simulate drift: overwrite the installed agent with stale bytes.
	agentPath := filepath.Join(h.TargetDir, ".claude", "agents", "engineer.md")
	if err := os.WriteFile(agentPath, []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write stale agent: %v", err)
	}

	h.Run(TargetCopilot, func(o *Options) {
		o.AutoSyncTargets = true
		o.Force = true
	})

	got, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("read agent: %v", err)
	}
	if string(got) == "stale\n" {
		t.Error("auto-sync should have refreshed the genuinely installed claude target")
	}

	st, err := ReadInstallState(h.TargetDir)
	if err != nil {
		t.Fatalf("ReadInstallState: %v", err)
	}
	for _, want := range []Target{TargetClaude, TargetCopilot} {
		if _, ok := st.Targets[string(want)]; !ok {
			raw, _ := json.Marshal(st.Targets)
			t.Errorf("expected %s in install-state targets; got %s", want, raw)
		}
	}
}
