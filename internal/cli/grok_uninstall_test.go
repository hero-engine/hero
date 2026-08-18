package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/version"
)

func installGrokFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := install.Run(install.Options{
		ContentFS: hero.ContentFS(), Target: install.TargetGrok,
		Mode: install.ModeProject, TargetDir: root, Force: true, Version: "test",
	}); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestUninstallGrokPreservesUserContentAndRemovesState(t *testing.T) {
	root := installGrokFixture(t)
	userAgent := filepath.Join(root, ".grok", "agents", "mine.md")
	userSkill := filepath.Join(root, ".grok", "skills", "mine", "SKILL.md")
	for path, body := range map[string]string{userAgent: "user agent", userSkill: "user skill"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(root, ".grok", "config.toml")
	data, _ := os.ReadFile(config)
	userConfigPrefix := "model = \"grok\"\n\n"
	if err := os.WriteFile(config, append([]byte(userConfigPrefix), data...), 0o644); err != nil {
		t.Fatal(err)
	}
	agents := filepath.Join(root, "AGENTS.md")
	data, _ = os.ReadFile(agents)
	if err := os.WriteFile(agents, append([]byte("# User rules\n\n"), data...), 0o644); err != nil {
		t.Fatal(err)
	}

	oldTarget, oldDryRun := uninstallTarget, uninstallDryRun
	t.Cleanup(func() { uninstallTarget, uninstallDryRun = oldTarget, oldDryRun })
	uninstallTarget, uninstallDryRun = "grok", false
	oldWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := runUninstall(uninstallCmd, nil); err != nil {
		t.Fatal(err)
	}

	for path, want := range map[string]string{userAgent: "user agent", userSkill: "user skill"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("user file %s not preserved: %q, %v", path, got, err)
		}
	}
	configBytes, err := os.ReadFile(config)
	if err != nil || string(configBytes) != userConfigPrefix+"\n" {
		t.Errorf("user config bytes not preserved: %q, %v", configBytes, err)
	}
	agentsBytes, err := os.ReadFile(agents)
	if err != nil || strings.Contains(string(agentsBytes), "hero:managed-start") || !strings.Contains(string(agentsBytes), "# User rules") {
		t.Errorf("AGENTS.md cleanup incorrect: %q, %v", agentsBytes, err)
	}
	state, err := install.ReadInstallState(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Targets[string(install.TargetGrok)]; ok {
		t.Fatalf("grok state survived uninstall: %+v", state.Targets)
	}
}

func TestUninstallGrokWithoutStatePreservesSharedAgentsMd(t *testing.T) {
	root := installGrokFixture(t)
	if err := os.Remove(filepath.Join(root, ".hero", "install-state.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".opencode", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(filepath.Join(root, "AGENTS.md"))
	if info == nil {
		t.Fatal("fixture missing AGENTS.md")
	}
	versionInfo, _ := version.Read(filepath.Join(root, ".hero"))
	if _, _, err := uninstallGrok(root, versionInfo); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil || !strings.Contains(string(data), "hero:managed-start") {
		t.Fatalf("shared AGENTS.md was stripped without state: %v", err)
	}
}
