package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	bstoml "github.com/BurntSushi/toml"
)

func TestGrokProjectInstallUsesNativeLayout(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.Run(TargetGrok, nil)

	for _, path := range []string{
		".grok/agents/engineer.md",
		".grok/skills/spec-format/SKILL.md",
		".grok/skills/command-design/SKILL.md",
		".grok/config.toml",
		"AGENTS.md",
	} {
		h.mustBeRegularFile(path)
	}
	h.mustNotExist(".grok/commands")
	h.mustNotExist(".claude")
	h.mustNotExist(".ai")
	h.mustContain(".grok/skills/command-design/SKILL.md", "name: command-design")
	h.mustContain(".grok/skills/command-design/SKILL.md", "Hero workflow for Grok Build")
	h.mustNotContain(".grok/skills/command-design/SKILL.md", "workflow for Codex")
	h.mustContain("AGENTS.md", ".grok/skills/command-deliver/SKILL.md")

	state, err := ReadInstallState(h.TargetDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Targets[string(TargetGrok)]; !ok {
		t.Fatalf("grok missing from install state: %+v", state.Targets)
	}
	row := findRow(t, harnessInventory(t, h), TargetGrok)
	if !row.Commands.NotApplicable || row.Skills.Expected != 4 || row.Skills.Actual != 4 {
		t.Fatalf("unexpected grok inventory: %+v", row)
	}
	h.mustSatisfyContract(TargetGrok, KindAgents)
	h.mustSatisfyContract(TargetGrok, KindSkills)
}

func TestGrokGlobalInstallUsesUserNativeLayout(t *testing.T) {
	h := newInstallHarness(t)
	home := os.Getenv("HOME")
	opts := Options{SourceDir: h.SourceDir, Target: TargetGrok, Mode: ModeGlobal, Force: true}
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		".grok/agents/engineer.md",
		".grok/skills/spec-format/SKILL.md",
		".grok/skills/command-design/SKILL.md",
		".grok/AGENTS.md",
		".grok/config.toml",
	} {
		if info, err := os.Stat(filepath.Join(home, rel)); err != nil || !info.Mode().IsRegular() {
			t.Errorf("expected global %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(h.TargetDir, ".grok")); !os.IsNotExist(err) {
		t.Fatal("global Grok install wrote project-local .grok")
	}
}

func TestGrokInstallIsIdempotent(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.runTwiceMustBeNoop(TargetGrok, func(o *Options) { o.Force = false })
}

func TestAutoSyncRefreshesGrokAndPreservesForeignContent(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.Run(TargetGrok, nil)
	foreign := filepath.Join(h.TargetDir, ".grok", "skills", "mine", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(foreign), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, []byte("user skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(h.SourceDir, "commands", "design.md")
	if err := os.WriteFile(command, []byte("---\ndescription: Updated.\n---\n# updated workflow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.Run(TargetClaude, func(o *Options) { o.AutoSyncTargets = true })
	h.mustContain(".grok/skills/command-design/SKILL.md", "# updated workflow")
	data, err := os.ReadFile(foreign)
	if err != nil || string(data) != "user skill" {
		t.Fatalf("auto-sync changed foreign Grok skill: %q, %v", data, err)
	}
}

func TestManagedTOMLMCPUpsertPreservesUserBytesAndConverges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	prefix := "# user prefix\nmodel = \"grok\"\n\n"
	suffix := "\n\n[ui]\ntheme = \"dark\"\n\n\n"
	original := prefix + "# hero:managed\n[mcp_servers.hero]\ncommand = \"old\"\nargs = [\"old\"]\n# end:hero:managed" + suffix
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := upsertManagedTOMLMCPConfig(path, false, "/workspace/root"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(first)
	if !strings.HasPrefix(got, prefix) || !strings.HasSuffix(got, suffix) {
		t.Fatalf("user bytes changed\nwant prefix %q suffix %q\ngot %q", prefix, suffix, got)
	}
	if strings.Count(got, "[mcp_servers.hero]") != 1 || !strings.Contains(got, `args = ["mcp", "--project-root", "/workspace/root"]`) {
		t.Fatalf("unexpected Hero MCP block:\n%s", got)
	}
	var parsed map[string]any
	if _, err := bstoml.Decode(got, &parsed); err != nil {
		t.Fatalf("rendered config is invalid TOML: %v", err)
	}
	if err := upsertManagedTOMLMCPConfig(path, false, "/workspace/root"); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(second) != got {
		t.Fatal("second MCP upsert was not byte-idempotent")
	}
}

func TestManagedTOMLMCPUpsertRemovesDuplicateLegacyHeroTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	input := "title = \"keep\"\n\n[mcp_servers.hero]\ncommand = \"one\"\n\n[mcp_servers.other]\ncommand = \"other\"\n\n[mcp_servers.hero]\ncommand = \"two\"\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := upsertManagedTOMLMCPConfig(path, false, ""); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	if strings.Count(got, "[mcp_servers.hero]") != 1 || !strings.Contains(got, "[mcp_servers.other]") || !strings.Contains(got, `title = "keep"`) {
		t.Fatalf("legacy dedupe damaged config:\n%s", got)
	}
	var parsed map[string]any
	if _, err := bstoml.Decode(got, &parsed); err != nil {
		t.Fatalf("deduped config invalid: %v", err)
	}
}

func TestGrokMalformedConfigIsRefusedWithoutMutation(t *testing.T) {
	h := newInstallHarness(t)
	path := filepath.Join(h.TargetDir, ".grok", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte("model = [unterminated\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Run(Options{SourceDir: h.SourceDir, Target: TargetGrok, Mode: ModeProject, TargetDir: h.TargetDir, Force: true})
	if err == nil || !strings.Contains(err.Error(), "parse TOML config") {
		t.Fatalf("expected useful malformed TOML error, got %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(original) {
		t.Fatalf("malformed config was replaced: %q", after)
	}
}

func TestGrokMCPDryRunDoesNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".grok", "config.toml")
	if err := upsertManagedTOMLMCPConfig(path, true, "/root"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote config: %v", err)
	}
}

func TestCommandAsSkillRendererPreservesCodexBytes(t *testing.T) {
	entry := canonicalEntry{Name: "design", Frontmatter: map[string]string{"description": "Produces a spec."}, Body: []byte("# /design command\n")}
	_, got, err := renderCommandAsCodexSkill(entry)
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nname: command-design\ndescription: Produces a spec.\nmetadata:\n  purpose: command-workflow\n---\n\n" +
		"> **This is a Hero workflow for Codex.** Read each step below and execute it in sequence.\n" +
		"> Do NOT summarize or treat these steps as documentation.\n" +
		"> Do NOT update spec frontmatter as a substitute for doing the actual work described.\n\n" +
		"# /design command\n"
	if string(got) != want {
		t.Fatalf("Codex command skill bytes changed\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestGrokSatelliteLinksOnlyDeclaredDirectories(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".hero", ".grok/agents", ".grok/skills"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sat := filepath.Join(root, "services", "api")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(SatelliteOptions{RootDir: root, SatelliteDir: sat, Scope: "services/api", Targets: []Target{TargetGrok}}); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"agents", "skills"} {
		if info, err := os.Lstat(filepath.Join(sat, ".grok", dir)); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected %s symlink: %v", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(sat, ".grok", "commands")); !os.IsNotExist(err) {
		t.Fatal("Grok satellite created a commands directory")
	}
	data, err := os.ReadFile(filepath.Join(sat, "AGENTS.md"))
	if err != nil || !strings.Contains(string(data), "grok") {
		t.Fatalf("Grok satellite marker missing: %v", err)
	}
}
