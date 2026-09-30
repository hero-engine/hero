package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/version"
)

func installDeepSeekFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := install.Run(install.Options{ContentFS: hero.ContentFS(), Target: install.TargetDeepSeek, Mode: install.ModeProject, TargetDir: root, Version: "test", Quiet: true}); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestDeepSeekUninstallOwnershipAndDryRun(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry-run", false: "remove"}[dryRun], func(t *testing.T) {
			root := installDeepSeekFixture(t)
			foreign := filepath.Join(root, ".dsh", "skills", "mine", "SKILL.md")
			os.MkdirAll(filepath.Dir(foreign), 0755)
			os.WriteFile(foreign, []byte("user skill"), 0644)
			modified := filepath.Join(root, ".dsh", "skills", "role-engineer", "SKILL.md")
			os.WriteFile(modified, []byte("edited role"), 0644)
			userConfig := filepath.Join(root, ".dsh", "cordis.patch.yml")
			os.WriteFile(userConfig, []byte("user configuration"), 0644)
			info, err := version.Read(filepath.Join(root, ".hero"))
			if err != nil {
				t.Fatal(err)
			}
			oldDry := uninstallDryRun
			t.Cleanup(func() { uninstallDryRun = oldDry })
			uninstallDryRun = dryRun
			if _, _, err := uninstallDeepSeek(root, info); err != nil {
				t.Fatal(err)
			}
			for path, want := range map[string]string{foreign: "user skill", modified: "edited role", userConfig: "user configuration"} {
				got, err := os.ReadFile(path)
				if err != nil || string(got) != want {
					t.Errorf("preserved %s = %q,%v", path, got, err)
				}
			}
			registered := install.InspectDeepSeekRegistration(root).Problem == ""
			if registered != dryRun {
				t.Errorf("home-patch entry registered=%v after uninstall (dry-run=%v)", registered, dryRun)
			}
			for _, rel := range []string{".dsh/skills/command-deliver/SKILL.md", "AGENTS.md"} {
				_, err := os.Stat(filepath.Join(root, rel))
				if dryRun && err != nil {
					t.Errorf("dry-run changed %s: %v", rel, err)
				}
				if !dryRun && !os.IsNotExist(err) {
					t.Errorf("owned file survived %s: %v", rel, err)
				}
			}
		})
	}
}

func TestDeepSeekUninstallKeepsSharedInstructionsAndClearsState(t *testing.T) {
	root := installDeepSeekFixture(t)
	if _, err := install.Run(install.Options{ContentFS: hero.ContentFS(), Target: install.TargetCodex, Mode: install.ModeProject, TargetDir: root, Version: "test", Quiet: true}); err != nil {
		t.Fatal(err)
	}
	oldWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	oldTarget, oldDry := uninstallTarget, uninstallDryRun
	t.Cleanup(func() { uninstallTarget, uninstallDryRun = oldTarget, oldDry })
	uninstallTarget, uninstallDryRun = "deepseek", false
	if err := runUninstall(uninstallCmd, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil || !strings.Contains(string(data), "Running Hero Workflows in Codex") {
		t.Fatalf("shared instructions lost: %v", err)
	}
	state, err := install.ReadInstallState(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Targets["deepseek"]; ok {
		t.Fatal("deepseek state remains")
	}
	if _, ok := state.Targets["codex"]; !ok {
		t.Fatal("codex state lost")
	}
}

func TestDeepSeekInstallJSONWorkspace(t *testing.T) {
	oldWorkspace, oldForceRoot := installWorkspace, installForceRoot
	t.Cleanup(func() {
		resetFlags()
		installWorkspace, installForceRoot = oldWorkspace, oldForceRoot
	})
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0755); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(root, "service with spaces"), 0755)
	output, err := runCmd("install", "project", root, "--target", "deepseek", "--workspace", "service with spaces", "--json", "--root")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, output)
	}
	var out install.InstallJSONOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("JSON: %v\n%s", err, output)
	}
	// The workspace registers the parent project's entry in the DeepSeek
	// home patch; no per-workspace overlay is written.
	reg := install.InspectDeepSeekRegistration(root)
	if reg.Problem != "" {
		t.Fatalf("workspace install did not register the project root: %+v", reg)
	}
	data, err := os.ReadFile(reg.Path)
	if err != nil || !strings.Contains(string(data), "--project-root") || !strings.Contains(string(data), reg.ServerName) {
		t.Fatalf("workspace binding: %s %v", data, err)
	}
	if !strings.Contains(output, reg.Path) {
		t.Fatalf("JSON missing MCP config path %s:\n%s", reg.Path, output)
	}
	if _, err := os.Stat(filepath.Join(root, "service with spaces", ".dsh", "hero.cordis.patch.yml")); !os.IsNotExist(err) {
		t.Fatalf("legacy workspace overlay written: %v", err)
	}
	info, err := version.Read(filepath.Join(root, ".hero"))
	if err != nil {
		t.Fatal(err)
	}
	oldDry := uninstallDryRun
	t.Cleanup(func() { uninstallDryRun = oldDry })
	uninstallDryRun = false
	if _, _, err := uninstallDeepSeek(root, info); err != nil {
		t.Fatal(err)
	}
	if install.InspectDeepSeekRegistration(root).Problem == "" {
		t.Fatal("home-patch entry survived uninstall")
	}
}

func TestDeepSeekUpgradeSelectionAndDoctor(t *testing.T) {
	root := installDeepSeekFixture(t)
	for _, requested := range [][]string{nil, {"deepseek"}} {
		targets, err := resolveUpgradeTargets(root, nil, requested)
		if err != nil || len(targets) != 1 || targets[0] != install.TargetDeepSeek {
			t.Fatalf("selection: %v,%v", targets, err)
		}
	}
	reg := install.InspectDeepSeekRegistration(root)
	report := buildInventorySection(doctorInfo{deepseekMCP: &reg, inventory: []install.TargetInventory{{Target: install.TargetDeepSeek, Agents: install.KindCount{NotApplicable: true}, Commands: install.KindCount{NotApplicable: true}, Skills: install.KindCount{Expected: 3, Actual: 3}, RootFile: "AGENTS.md"}}})
	for _, want := range []string{"role-*", "MCP: registered as " + reg.ServerName, reg.Path, "dsh --profile web"} {
		if !strings.Contains(report, want) {
			t.Errorf("doctor missing %q: %s", want, report)
		}
	}
}

func TestDeepSeekWorkspaceCollisionFailsBeforeRootInstall(t *testing.T) {
	oldWorkspace, oldForceRoot := installWorkspace, installForceRoot
	t.Cleanup(func() {
		resetFlags()
		installWorkspace, installForceRoot = oldWorkspace, oldForceRoot
	})
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "service"), 0755)
	// A home patch that is not a YAML list cannot be safely edited.
	homePatch, err := install.DeepSeekHomePatchPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(homePatch), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(homePatch, []byte("user: mapping\n"), 0644); err != nil {
		t.Fatal(err)
	}
	output, err := runCmd("install", "project", root, "--target", "deepseek", "--workspace", "service", "--json", "--root")
	if err == nil {
		t.Fatal("expected collision error")
	}
	var out install.InstallJSONOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil || out.Error == nil {
		t.Fatalf("expected JSON error: %s %v", output, err)
	}
	for _, rel := range []string{"AGENTS.md", ".dsh", ".hero", "service/.dsh"} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Errorf("root mutated before collision: %s %v", rel, err)
		}
	}
	data, err := os.ReadFile(homePatch)
	if err != nil || string(data) != "user: mapping\n" {
		t.Fatalf("foreign home patch changed: %q %v", data, err)
	}
}

func TestDeepSeekUninstallPreservesModifiedOverlay(t *testing.T) {
	root := installDeepSeekFixture(t)
	overlay := filepath.Join(root, ".dsh", "hero.cordis.patch.yml")
	if err := os.WriteFile(overlay, []byte("user modified overlay"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := version.Read(filepath.Join(root, ".hero"))
	if err != nil {
		t.Fatal(err)
	}
	oldDry := uninstallDryRun
	t.Cleanup(func() { uninstallDryRun = oldDry })
	uninstallDryRun = false
	if _, _, err := uninstallDeepSeek(root, info); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(overlay)
	if err != nil || string(data) != "user modified overlay" {
		t.Fatalf("modified overlay changed: %q %v", data, err)
	}
}

// Audit: an unsafe home patch fails uninstall before any project file goes.
func TestDeepSeekUninstallFailsBeforeRemovingFilesOnBadHomePatch(t *testing.T) {
	root := installDeepSeekFixture(t)
	homePatch, err := install.DeepSeekHomePatchPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(homePatch, []byte("not: a list\n"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := version.Read(filepath.Join(root, ".hero"))
	if err != nil {
		t.Fatal(err)
	}
	oldDry := uninstallDryRun
	t.Cleanup(func() { uninstallDryRun = oldDry })
	uninstallDryRun = false
	if _, _, err := uninstallDeepSeek(root, info); err == nil {
		t.Fatal("uninstall accepted an unsafe home patch")
	}
	if _, err := os.Stat(filepath.Join(root, ".dsh", "skills", "command-deliver", "SKILL.md")); err != nil {
		t.Fatalf("project files removed before the home-patch failure: %v", err)
	}
}
