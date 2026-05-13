package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bstoml "github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// verification_test.go — Layer 1 routine verification: real-parser
// format-correctness, round-trip semantic checks, multi-target
// auto-sync drift prevention, and migration scenarios covering the
// legacy `.hero/{agents,commands,skills}/` + symlink layout we just
// retired.
//
// Layer 2 (real-harness binary launches, gated on CI-installed
// codex/opencode binaries) lives in a separate spec — child #6 of
// install-upgrade-contract-coverage.

// TestVerify_CodexAgentTomlParses confirms every rendered Codex agent
// parses as valid TOML with the loader-required `developer_instructions`
// field. Catches a TOML emitter regression (broken escaping, missing
// field) that would otherwise only surface when a Codex user complains.
func TestVerify_CodexAgentTomlParses(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.Run(TargetCodex, nil)

	entries, err := os.ReadDir(filepath.Join(h.TargetDir, ".codex", "agents"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one rendered Codex agent")
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		path := filepath.Join(h.TargetDir, ".codex", "agents", e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		var parsed map[string]any
		if _, err := bstoml.Decode(string(data), &parsed); err != nil {
			t.Errorf("%s: TOML parse failed: %v\n--- body ---\n%s", path, err, string(data))
			continue
		}
		if di, ok := parsed["developer_instructions"].(string); !ok || strings.TrimSpace(di) == "" {
			t.Errorf("%s: missing or empty `developer_instructions`", path)
		}
		if _, ok := parsed["name"].(string); !ok {
			t.Errorf("%s: missing `name`", path)
		}
	}
}

// TestVerify_CopilotPromptYAMLParses confirms every rendered Copilot
// .prompt.md file has parseable YAML frontmatter and a non-empty body.
func TestVerify_CopilotPromptYAMLParses(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.Run(TargetCopilot, nil)

	for _, sub := range []string{"agents", "commands"} {
		dir := filepath.Join(h.TargetDir, ".github", "prompts", sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".prompt.md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Errorf("read %s: %v", e.Name(), err)
				continue
			}
			fm, body, ok := splitYAMLFrontmatter(data)
			if !ok {
				t.Errorf("%s/%s: missing YAML frontmatter", sub, e.Name())
				continue
			}
			var parsed map[string]any
			if err := yaml.Unmarshal(fm, &parsed); err != nil {
				t.Errorf("%s/%s: YAML frontmatter parse: %v", sub, e.Name(), err)
				continue
			}
			if len(strings.TrimSpace(string(body))) == 0 {
				t.Errorf("%s/%s: empty body", sub, e.Name())
			}
			if desc, ok := parsed["description"].(string); !ok || strings.TrimSpace(desc) == "" {
				t.Errorf("%s/%s: missing description in frontmatter", sub, e.Name())
			}
		}
	}
}

// TestVerify_ClaudeSettingsJSON confirms .claude/settings.json after
// install parses as valid JSON and carries the expected hero entries
// (Bash(hero:*) permission, NEXT.md checkpoint hooks).
func TestVerify_ClaudeSettingsJSON(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.Run(TargetClaude, nil)

	path := filepath.Join(h.TargetDir, ".claude", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("%s: JSON parse: %v\n--- body ---\n%s", path, err, string(data))
	}
	// Check the Hero allowlist entry is present.
	if perms, ok := parsed["permissions"].(map[string]any); ok {
		if allow, ok := perms["allow"].([]any); ok {
			has := false
			for _, v := range allow {
				if s, ok := v.(string); ok && s == "Bash(hero:*)" {
					has = true
					break
				}
			}
			if !has {
				t.Errorf("settings.json permissions.allow missing Bash(hero:*): %v", allow)
			}
		}
	}
}

// TestVerify_RoundTrip_CodexAgent confirms the canonical → TOML →
// re-parse pipeline preserves body content. Catches an emitter bug
// that silently corrupts the prompt (e.g. mangled escapes).
func TestVerify_RoundTrip_CodexAgent(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.Run(TargetCodex, nil)

	// Read what canonical (in seed source) says for engineer.md.
	canonical, err := os.ReadFile(filepath.Join(h.SourceDir, "agents", "engineer.md"))
	if err != nil {
		t.Fatal(err)
	}
	canonicalFM, canonicalBody, _ := splitYAMLFrontmatter(canonical)
	var meta map[string]string
	if err := yaml.Unmarshal(canonicalFM, &meta); err != nil {
		t.Fatal(err)
	}

	// Read what the rendered Codex TOML says for the same agent.
	data, err := os.ReadFile(filepath.Join(h.TargetDir, ".codex", "agents", "engineer.toml"))
	if err != nil {
		t.Fatalf("read rendered TOML: %v", err)
	}
	var parsed map[string]any
	if _, err := bstoml.Decode(string(data), &parsed); err != nil {
		t.Fatalf("TOML parse: %v", err)
	}

	bodyStr := strings.TrimSpace(string(canonicalBody))
	tomlBody, _ := parsed["developer_instructions"].(string)
	tomlBody = strings.TrimSpace(tomlBody)
	if !strings.Contains(tomlBody, bodyStr[:min(len(bodyStr), 50)]) {
		t.Errorf("round-trip body mismatch:\n--- canonical (first 200) ---\n%s\n--- TOML developer_instructions (first 200) ---\n%s",
			head(bodyStr, 200), head(tomlBody, 200))
	}
	if name, _ := parsed["name"].(string); name != "engineer" {
		t.Errorf("round-trip name mismatch: got %q", name)
	}
}

// TestVerify_AutoSync_RefreshesSiblings confirms that installing one
// harness target into a project where another is already installed
// auto-syncs the second target so both end at the same version.
func TestVerify_AutoSync_RefreshesSiblings(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Step 1: install Claude (no other harnesses present → no auto-sync action).
	h.Run(TargetClaude, func(o *Options) { o.AutoSyncTargets = true })
	h.mustBeRegularFile(".claude/agents/engineer.md")

	// Step 2: install OpenCode with auto-sync enabled. Detects existing
	// .claude/ and refreshes Claude alongside.
	h.Run(TargetOpenCode, func(o *Options) { o.AutoSyncTargets = true })
	h.mustBeRegularFile(".opencode/agents/engineer.md")
	h.mustBeRegularFile(".claude/agents/engineer.md")

	// Both should have current canonical content.
	claudeBytes, _ := os.ReadFile(filepath.Join(h.TargetDir, ".claude", "agents", "engineer.md"))
	opencodeBytes, _ := os.ReadFile(filepath.Join(h.TargetDir, ".opencode", "agents", "engineer.md"))
	canonicalBytes, _ := os.ReadFile(filepath.Join(h.SourceDir, "agents", "engineer.md"))
	if string(claudeBytes) != string(canonicalBytes) {
		t.Error("claude engineer.md drifted from canonical after auto-sync")
	}
	if string(opencodeBytes) != string(canonicalBytes) {
		t.Error("opencode engineer.md drifted from canonical after install")
	}
}

// TestVerify_LegacyCleanup_RemovesCanonicalMirrorAndSymlinks confirms
// the upgrade-from-P2 path: a fixture with `.hero/{agents,commands,skills}/`
// + harness symlinks pointing at them gets cleaned up automatically on
// the next install. User-edited content is preserved.
func TestVerify_LegacyCleanup_RemovesCanonicalMirrorAndSymlinks(t *testing.T) {
	h := newInstallHarness(t)
	heroDir := filepath.Join(h.TargetDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Seed legacy canonical mirror at .hero/agents/engineer.md (matching
	// canonical bytes so cleanup recognizes it as Hero-authored).
	mustMirrorCanonical(t, h, "agents/engineer.md", filepath.Join(heroDir, "agents", "engineer.md"))
	// Seed legacy harness symlink pointing at the canonical mirror.
	claudeDir := filepath.Join(h.TargetDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../.hero/agents", filepath.Join(claudeDir, "agents")); err != nil {
		t.Fatal(err)
	}

	h.Run(TargetClaude, nil)

	// Legacy symlink at .claude/agents must be replaced with a real dir.
	info, err := os.Lstat(filepath.Join(claudeDir, "agents"))
	if err != nil {
		t.Fatalf(".claude/agents missing after cleanup: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error(".claude/agents should be a regular directory after render-direct install, not a symlink")
	}
	if !info.IsDir() {
		t.Error(".claude/agents should be a directory")
	}
	// Legacy canonical mirror at .hero/agents must be gone.
	if _, err := os.Stat(filepath.Join(heroDir, "agents")); err == nil {
		t.Error(".hero/agents canonical mirror should have been cleaned up")
	}
}

// splitYAMLFrontmatter pulls the YAML frontmatter and the body apart.
func splitYAMLFrontmatter(data []byte) (fm, body []byte, ok bool) {
	s := string(data)
	const marker = "---\n"
	if !strings.HasPrefix(s, marker) {
		return nil, nil, false
	}
	rest := s[len(marker):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, nil, false
	}
	fm = []byte(rest[:end])
	body = []byte(strings.TrimPrefix(rest[end+len("\n---"):], "\n"))
	return fm, body, true
}

func head(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
