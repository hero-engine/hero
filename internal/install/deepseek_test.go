package install

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// deepseek-harness-install-target AC-1 AC-2 AC-3 AC-5 AC-6.
func TestDeepSeekNativeLayoutAndInventory(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.Run(TargetDeepSeek, nil)
	for _, path := range []string{".dsh/skills/role-engineer/SKILL.md", ".dsh/skills/command-design/SKILL.md", ".dsh/skills/spec-format/SKILL.md", ".dsh/hero.cordis.patch.yml", "AGENTS.md"} {
		h.mustBeRegularFile(path)
	}
	for _, path := range []string{".dsh/agents", ".dsh/commands", ".claude", ".agents", ".mcp.json"} {
		h.mustNotExist(path)
	}
	row := findRow(t, harnessInventory(t, h), TargetDeepSeek)
	if !row.Agents.NotApplicable || !row.Commands.NotApplicable || row.Skills.Expected != 6 || row.Skills.Actual != 6 || row.Incomplete() {
		t.Fatalf("inventory: %+v", row)
	}
	h.mustContain("AGENTS.md", "stop at that named gate")
	h.mustContain(".dsh/skills/role-engineer/SKILL.md", "user-invocable: false")
	h.mustContain(".dsh/skills/role-engineer/SKILL.md", "# Engineer agent")
	h.runTwiceMustBeNoop(TargetDeepSeek, func(o *Options) { o.Force = false })
	if err := os.Remove(filepath.Join(h.TargetDir, ".dsh/skills/role-engineer/SKILL.md")); err != nil {
		t.Fatal(err)
	}
	row = findRow(t, harnessInventory(t, h), TargetDeepSeek)
	if !row.Incomplete() || !strings.Contains(strings.Join(row.Missing, ","), "role-engineer/SKILL.md") {
		t.Fatalf("missing role: %+v", row)
	}
}

// deepseek-harness-install-target AC-2: quoted scalars and role constraints.
func TestDeepSeekRenderingYAML(t *testing.T) {
	raw := []byte("---\nname: engineer\ndescription: 'Review: \"quoted\" text'\ntools: [Read, Write]\nmodel: large\n---\nIntact body.\n")
	fm, body := parseSimpleFrontmatter(raw)
	entry := canonicalEntry{Name: "engineer", Raw: raw, Body: body, Frontmatter: fm}
	for _, renderer := range []func(canonicalEntry) (string, []byte, error){renderDeepSeekRoleSkill, commandAsSkillRenderer("DeepSeek")} {
		_, data, err := renderer(entry)
		if err != nil {
			t.Fatal(err)
		}
		fields, err := deepseekFrontmatter(data)
		if err != nil {
			t.Fatal(err)
		}
		if fields["description"] != "Review: \"quoted\" text" || !strings.Contains(string(data), string(body)) {
			t.Fatalf("bad rendered bytes %s", data)
		}
		if fields["name"] == "role-engineer" {
			if fields["metadata"] == nil || fields["disable-model-invocation"] != nil {
				t.Fatalf("bad role metadata: %+v", fields)
			}
		}
	}
}

// deepseek-harness-install-target AC-3 AC-4 AC-5.
func TestDeepSeekGlobalOwnershipAndHome(t *testing.T) {
	h := newInstallHarness(t)
	base := filepath.Join(t.TempDir(), "custom home")
	t.Setenv("DSH_HOME", base)
	opts := Options{SourceDir: h.SourceDir, Target: TargetDeepSeek, Mode: ModeGlobal, Quiet: true}
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"AGENTS.md", "skills/role-engineer/SKILL.md", deepseekOverlayName, deepseekManifestName} {
		if _, err := os.Stat(filepath.Join(base, rel)); err != nil {
			t.Fatal(err)
		}
	}
	h.mustNotExist(".dsh")
	h.mustNotExist("AGENTS.md")
	data, _ := os.ReadFile(filepath.Join(base, deepseekOverlayName))
	if strings.Contains(string(data), h.TargetDir) || strings.Contains(string(data), "cwd:") {
		t.Fatal("global pins cwd")
	}
	// A malformed manifest must fail before writing any target files, even forced.
	path := filepath.Join(base, deepseekManifestName)
	if err := os.WriteFile(path, []byte(`{"version":1,"files":{"../outside":"bad"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	opts.Force = true
	if _, err := Run(opts); err == nil {
		t.Fatal("malformed manifest accepted")
	}
	got, _ := os.ReadFile(filepath.Join(base, deepseekOverlayName))
	if string(got) != string(data) {
		t.Fatal("overlay changed before validation")
	}
	for _, home := range []string{"", "   ", "~/custom", "relative-home", " spaced-home "} {
		t.Setenv("DSH_HOME", home)
		got, err := DeepSeekHome()
		if err != nil || !filepath.IsAbs(got) {
			t.Fatalf("home %q => %q %v", home, got, err)
		}
		if home == " spaced-home " && !strings.HasSuffix(got, home) {
			t.Fatal("significant whitespace trimmed")
		}
	}
}

// deepseek-harness-install-target AC-3 AC-5 AC-8.
func TestDeepSeekOverlayCollisionDryRunAndRemoval(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	base := filepath.Join(h.TargetDir, ".dsh")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, deepseekOverlayName)
	if err := os.WriteFile(path, []byte("user overlay"), 0644); err != nil {
		t.Fatal(err)
	}
	opts := Options{SourceDir: h.SourceDir, Target: TargetDeepSeek, Mode: ModeProject, TargetDir: h.TargetDir, Quiet: true}
	for _, dry := range []bool{false, true} {
		opts.DryRun = dry
		if _, err := Run(opts); err == nil {
			t.Fatal("overlay collision accepted")
		}
		h.mustNotExist("AGENTS.md")
		h.mustNotExist(".dsh/skills")
	}
	opts.Force = true
	opts.DryRun = true
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	h.mustNotExist("AGENTS.md")
	h.mustNotExist(".dsh/skills")
	opts.DryRun = false
	opts.ProjectRoot = "/workspace with spaces"
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var patch []struct {
		Insert []struct {
			Config struct {
				Command string   `yaml:"command"`
				Args    []string `yaml:"args"`
			} `yaml:"config"`
		} `yaml:"insert"`
	}
	if err := yaml.Unmarshal(data, &patch); err != nil {
		t.Fatal(err)
	}
	if len(patch) != 1 || len(patch[0].Insert) != 1 || patch[0].Insert[0].Config.Command != "hero" || strings.Join(patch[0].Insert[0].Config.Args, "|") != "mcp|--project-root|/workspace with spaces" {
		t.Fatalf("invalid overlay %s", data)
	}
	if ok, err := RemoveDeepSeekOverlay(h.TargetDir, true, false); err != nil || !ok {
		t.Fatalf("dry removal: %v %v", ok, err)
	}
	h.mustBeRegularFile(".dsh/hero.cordis.patch.yml")
	if err := os.WriteFile(path, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}
	if removed, err := RemoveDeepSeekOverlay(h.TargetDir, false, false); err != nil || removed {
		t.Fatal("removed modified overlay")
	}
	if ok, err := RemoveDeepSeekOverlay(h.TargetDir, false, true); err != nil || !ok {
		t.Fatalf("force removal: %v %v", ok, err)
	}
	// sept-review-cleanup AC-7: generated AGENTS.md guidance names the profile dsh requires.
	if section := renderDeepSeekWorkflowSection(); !strings.Contains(section, "with dsh --profile web --patch /absolute/") {
		t.Fatalf("DeepSeek AGENTS.md guidance lacks the interactive launch:\n%s", section)
	}
	// sept-review-cleanup AC-7: interactive launch, no headless profile or prompt.
	if got := DeepSeekLaunchCommand("/a'b c"); got != "dsh --profile web --patch '/a'\"'\"'b c'" {
		t.Fatalf("launch command = %q", got)
	}
	if !strings.Contains(DeepSeekLaunchCommand("/a'b c"), "'/a'\"'\"'b c'") {
		t.Fatal("unsafe patch quoting")
	}
}

// deepseek-harness-install-target AC-2 AC-5.
func TestDeepSeekNamespaceCollisionAndPruning(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.Run(TargetDeepSeek, nil)
	foreign := filepath.Join(h.TargetDir, ".dsh/skills/role-mine/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(foreign), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, []byte("mine"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, src := range []string{"agents/reviewer.md", "commands/design.md", "skills/test-strategy/SKILL.md"} {
		if err := os.Remove(filepath.Join(h.SourceDir, src)); err != nil {
			t.Fatal(err)
		}
	}
	h.Run(TargetDeepSeek, func(o *Options) { o.Force = false })
	for _, name := range []string{"role-reviewer", "command-design", "test-strategy"} {
		h.mustNotExist(".dsh/skills/" + name)
	}
	h.mustContain(".dsh/skills/role-mine/SKILL.md", "mine")
	collision := filepath.Join(h.SourceDir, "skills/role-engineer/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(collision), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(collision, []byte("duplicate"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{SourceDir: h.SourceDir, Target: TargetDeepSeek, Mode: ModeProject, TargetDir: h.TargetDir, Force: true, Quiet: true}); err == nil || !strings.Contains(err.Error(), "namespace collision") {
		t.Fatalf("collision: %v", err)
	}
}

// deepseek-harness-install-target AC-9.
func TestDeepSeekMixedRoutingAndSiblingSync(t *testing.T) {
	for _, other := range []Target{TargetCodex, TargetClaude, TargetGrok} {
		for _, reverse := range []bool{false, true} {
			t.Run(string(other)+fmtBool(reverse), func(t *testing.T) {
				h := newInstallHarness(t)
				mkHeroDir(t, h.TargetDir)
				first, second := TargetDeepSeek, other
				if reverse {
					first, second = second, first
				}
				h.Run(first, nil)
				h.Run(second, func(o *Options) { o.AutoSyncTargets = true })
				h.mustContain("AGENTS.md", "Running Hero Workflows in DeepSeek")
				if other == TargetCodex {
					h.mustContain("AGENTS.md", "Running Hero Workflows in Codex")
				}
				if other == TargetGrok {
					h.mustContain("AGENTS.md", "Running Hero Workflows in Grok")
				}
			})
		}
	}
}
func fmtBool(value bool) string {
	if value {
		return "-reverse"
	}
	return "-forward"
}

// deepseek-harness-install-target AC-7.
func TestDeepSeekSatelliteLayout(t *testing.T) {
	h := newInstallHarness(t)
	h.Run(TargetDeepSeek, nil)
	sat := filepath.Join(h.TargetDir, "child")
	if err := os.MkdirAll(sat, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(SatelliteOptions{RootDir: h.TargetDir, SatelliteDir: sat, Targets: []Target{TargetDeepSeek}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Readlink(filepath.Join(sat, ".dsh/skills")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"agents", "commands", "hero.cordis.patch.yml"} {
		if _, err := os.Lstat(filepath.Join(sat, ".dsh", name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected satellite %s", name)
		}
	}
	data, _ := os.ReadFile(filepath.Join(sat, "AGENTS.md"))
	if !strings.Contains(string(data), DeepSeekLaunchCommand(filepath.Join(h.TargetDir, ".dsh", deepseekOverlayName))) {
		t.Fatal("missing parent activation")
	}
	if err := RemoveSatellite(sat, []Target{TargetDeepSeek}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(sat, ".dsh/skills")); !os.IsNotExist(err) {
		t.Fatal("link survived")
	}
}

// deepseek-harness-install-target AC-4 AC-5: global manifest preserves user edits.
func TestDeepSeekGlobalRefreshPruneAndDryRun(t *testing.T) {
	h := newInstallHarness(t)
	base := filepath.Join(t.TempDir(), "global")
	t.Setenv("DSH_HOME", base)
	opts := Options{SourceDir: h.SourceDir, Target: TargetDeepSeek, Mode: ModeGlobal, Quiet: true, DryRun: true}
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatal("global dry run wrote files")
	}
	opts.DryRun = false
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	modified := filepath.Join(base, "skills/role-reviewer/SKILL.md")
	if err := os.WriteFile(modified, []byte("user edited role"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"agents/reviewer.md", "commands/design.md"} {
		if err := os.Remove(filepath.Join(h.SourceDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(modified); err != nil || string(data) != "user edited role" {
		t.Fatal("modified dropped role lost")
	}
	if _, err := os.Stat(filepath.Join(base, "skills/command-design")); !os.IsNotExist(err) {
		t.Fatal("unchanged dropped workflow survived")
	}
	owned := filepath.Join(base, "skills/role-engineer/SKILL.md")
	if err := os.WriteFile(owned, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(opts); err == nil {
		t.Fatal("modified global role overwritten")
	}
	opts.Force = true
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(owned)
	if !strings.Contains(string(data), "# Engineer agent") {
		t.Fatal("force did not restore")
	}
}

// deepseek-harness-install-target AC-3 AC-5: workspace overlay ownership follows its root.
func TestDeepSeekWorkspaceOverlayRefresh(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	ws := filepath.Join(h.TargetDir, "workspace space")
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatal(err)
	}
	opts := Options{Target: TargetDeepSeek, Mode: ModeProject, TargetDir: ws, ProjectRoot: h.TargetDir, Quiet: true}
	if err := RegisterMCP(TargetDeepSeek, opts); err != nil {
		t.Fatal(err)
	}
	base, _ := deepseekBase(opts)
	sums, err := deepseekChecksums(opts, base)
	if err != nil || sums[deepseekOverlayName] == "" {
		t.Fatalf("workspace ownership missing: %v %v", sums, err)
	}
	if err := RegisterMCP(TargetDeepSeek, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".hero")); !os.IsNotExist(err) {
		t.Fatal("workspace overlay made shadow .hero")
	}
}

// countingFS counts opens of one path so tests can assert render passes.
type countingFS struct {
	fs.FS
	path  string
	opens int
}

func (c *countingFS) Open(name string) (fs.File, error) {
	if name == c.path {
		c.opens++
	}
	return c.FS.Open(name)
}

// sept-review-cleanup AC-8: one install renders the DeepSeek file set in a
// single pass (selection plus render reads), not once per preflight.
func TestDeepSeekInstallRendersFileSetOnce(t *testing.T) {
	h := newInstallHarness(t)
	onePass := &countingFS{FS: os.DirFS(h.SourceDir), path: "agents/engineer.md"}
	if _, _, err := deepseekFiles(Options{ContentFS: onePass, Target: TargetDeepSeek, Mode: ModeProject, TargetDir: h.TargetDir}); err != nil {
		t.Fatal(err)
	}
	install := &countingFS{FS: os.DirFS(h.SourceDir), path: "agents/engineer.md"}
	h.Run(TargetDeepSeek, func(o *Options) { o.ContentFS = install })
	if onePass.opens == 0 || install.opens != onePass.opens {
		t.Fatalf("install opened agents/engineer.md %d times; one render pass opens it %d times", install.opens, onePass.opens)
	}
}

// `hero install project . --target deepseek` passes a relative TargetDir;
// ownership lookup must not fail relating it to the absolute .dsh base.
func TestDeepSeekInstallAcceptsRelativeTargetDir(t *testing.T) {
	h := newInstallHarness(t)
	t.Chdir(h.TargetDir)
	h.Run(TargetDeepSeek, func(o *Options) { o.TargetDir = "."; o.ProjectRoot = "." })
	h.mustExist(".dsh/hero.cordis.patch.yml")
	h.mustExist(".dsh/skills/command-design/SKILL.md")
	// Repeat install exercises the prior-checksum path with the relative root.
	h.Run(TargetDeepSeek, func(o *Options) { o.TargetDir = "."; o.ProjectRoot = "." })
}
