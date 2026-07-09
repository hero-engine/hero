package install

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	hero "github.com/hero-engine/hero"
)

// TestEngineeringPackBodyMatchesGoFallback locks the canonical
// domains/engineering/AGENTS.md and the embedded Go fallback in
// lockstep. The on-disk pack file is the source of truth for engineering
// AGENTS.md content; the Go fallback exists as a "embedded binary lost
// its filesystem" floor and must stay byte-equal to the pack body
// (minus the H1 line and any trailing whitespace).
//
// Failure means the pack file or the Go fallback was edited without
// updating the other. Run the regenerator with:
//
//	HERO_REGEN_PACK_AGENTS=1 go test -run TestEngineeringPackBodyMatchesGoFallback ./internal/install/
//
// to rewrite the pack file from the Go fallback's current output.
func TestEngineeringPackBodyMatchesGoFallback(t *testing.T) {
	repoRoot, err := repoRootFromHere()
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	packPath := filepath.Join(repoRoot, "domains", "engineering", "AGENTS.md")

	goBody := generateEngineeringAgentsMdBody(resolveContentPathsForBody(Options{}))
	canonical := "# Hero — Spec-Driven AI Engineering\n\n" + goBody + "\n"

	if os.Getenv("HERO_REGEN_PACK_AGENTS") == "1" {
		if err := os.WriteFile(packPath, []byte(canonical), 0o644); err != nil {
			t.Fatalf("regenerate pack file: %v", err)
		}
		t.Logf("regenerated %s", packPath)
		return
	}

	got, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("read pack AGENTS.md: %v", err)
	}
	if string(got) != canonical {
		t.Fatalf("domains/engineering/AGENTS.md diverged from generateEngineeringAgentsMdBody output\n"+
			"re-run with HERO_REGEN_PACK_AGENTS=1 to regenerate.\n\n"+
			"--- want (first 400 bytes) ---\n%s\n--- got (first 400 bytes) ---\n%s",
			truncate(canonical, 400), truncate(string(got), 400))
	}
}

// TestEngineeringAgentsMdRosterComplete walks the shipped engineering
// (pack + core) commands, agents, and skills directories and asserts
// every one of them is named in domains/engineering/AGENTS.md — a
// command as a literal `/name` token, an agent or skill as its bare
// name. This is the backstop for domain-agents-md-skeleton's roster-
// completeness rule: an entry that ships but isn't named can't be
// routed to, and nothing else catches that regression. Run this test
// once with a name deliberately removed from any of the six
// directories to confirm it fails naming the missing entry.
func TestEngineeringAgentsMdRosterComplete(t *testing.T) {
	repoRoot, err := repoRootFromHere()
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}

	agentsMdPath := filepath.Join(repoRoot, "domains", "engineering", "AGENTS.md")
	raw, err := os.ReadFile(agentsMdPath)
	if err != nil {
		t.Fatalf("read %s: %v", agentsMdPath, err)
	}
	content := string(raw)

	var missing []string

	for _, dir := range []string{
		filepath.Join(repoRoot, "domains", "engineering", "commands"),
		filepath.Join(repoRoot, "core", "commands"),
	} {
		for _, name := range rosterMdFileNames(t, dir) {
			if !strings.Contains(content, "/"+name) {
				missing = append(missing, "command /"+name)
			}
		}
	}

	for _, dir := range []string{
		filepath.Join(repoRoot, "domains", "engineering", "agents"),
		filepath.Join(repoRoot, "core", "agents"),
	} {
		for _, name := range rosterMdFileNames(t, dir) {
			if !strings.Contains(content, name) {
				missing = append(missing, "agent "+name)
			}
		}
	}

	for _, dir := range []string{
		filepath.Join(repoRoot, "domains", "engineering", "skills"),
		filepath.Join(repoRoot, "core", "skills"),
	} {
		for _, name := range rosterSkillDirNames(t, dir) {
			if !strings.Contains(content, name) {
				missing = append(missing, "skill "+name)
			}
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("domains/engineering/AGENTS.md is missing roster entries for:\n  %s",
			strings.Join(missing, "\n  "))
	}
}

// rosterMdFileNames returns the base names (no ".md", README skipped)
// of every markdown file directly inside dir. Used for command and
// agent directories, which are flat files.
func rosterMdFileNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || e.Name() == "README.md" || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".md"))
	}
	return names
}

// rosterSkillDirNames returns the subdirectory names directly inside
// dir. Used for skills directories, where each skill is a subdir
// containing SKILL.md.
func rosterSkillDirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	return names
}

// TestSplitPackAgentsMd verifies the H1-stripping logic for pack
// AGENTS.md content.
func TestSplitPackAgentsMd(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantBody string
		wantTit  string
	}{
		{
			name:     "h1-with-body",
			in:       "# Hero PM — Spec-Driven AI Product Management\n\nbody content here\n",
			wantBody: "body content here",
			wantTit:  "Hero PM — Spec-Driven AI Product Management",
		},
		{
			name:     "h1-only",
			in:       "# Just A Title",
			wantBody: "",
			wantTit:  "Just A Title",
		},
		{
			name:     "no-h1",
			in:       "no leading H1\nlots of body\n",
			wantBody: "no leading H1\nlots of body",
			wantTit:  "",
		},
		{
			name:     "leading-blank-lines",
			in:       "\n\n# Title\n\nbody",
			wantBody: "body",
			wantTit:  "Title",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, title := splitPackAgentsMd(tc.in)
			if body != tc.wantBody {
				t.Errorf("body: got %q, want %q", body, tc.wantBody)
			}
			if title != tc.wantTit {
				t.Errorf("title: got %q, want %q", title, tc.wantTit)
			}
		})
	}
}

// TestLoadPackAgentsMdBody_OverrideShortCircuits ensures the override
// seam is honored even when a pack FS is provided.
func TestLoadPackAgentsMdBody_OverrideShortCircuits(t *testing.T) {
	override := []byte("# Custom Pack\n\ncustom body content")
	body, title, fell := loadPackAgentsMdBody(Options{AgentsMdBodyOverride: override})
	if fell {
		t.Fatal("override should not be treated as a fallback")
	}
	if title != "Custom Pack" {
		t.Errorf("title = %q, want %q", title, "Custom Pack")
	}
	if !strings.Contains(body, "custom body content") {
		t.Errorf("body missing custom content: %q", body)
	}
}

// TestLoadPackAgentsMdBody_PackMissingFallsBack ensures the Go fallback
// kicks in when the pack FS has no AGENTS.md.
func TestLoadPackAgentsMdBody_PackMissingFallsBack(t *testing.T) {
	body, title, fell := loadPackAgentsMdBody(Options{Domain: "engineering"})
	if !fell {
		t.Fatal("expected fallback when no source FS and no override")
	}
	if title != "" {
		t.Errorf("fallback title should be empty (caller emits legacy H2): got %q", title)
	}
	if !strings.Contains(body, "Spec-Driven AI Engineering") &&
		!strings.Contains(body, "Hero") {
		t.Errorf("fallback body missing canonical content")
	}
}

// TestPMInstallExcludesEngineeringAgents locks the agent-filtering
// contract: agents whose frontmatter restricts them to `domains:
// [engineering]` must not appear in a PM workspace's harness agent
// directory.
func TestPMInstallExcludesEngineeringAgents(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	pmFS, err := hero.DomainFS("pm")
	if err != nil {
		t.Fatalf("DomainFS(pm): %v", err)
	}
	if _, err := Run(Options{
		ContentFS: hero.OverlayFS(pmFS, hero.CoreFS()),
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: dir,
		Force:     true,
		Domain:    "pm",
	}); err != nil {
		t.Fatalf("install pm: %v", err)
	}

	engineeringOnly := []string{
		"feature-delivery-lead.md",
		"debug-investigator.md",
		"database-engineer.md",
		"devops-engineer.md",
		"release-engineer.md",
		"dependency-analyst.md",
		"migration-engineer.md",
		"architecture-reviewer.md",
	}
	for _, name := range engineeringOnly {
		path := filepath.Join(dir, ".claude", "agents", name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("engineering-only agent %s leaked into PM workspace at %s", name, path)
		}
	}
}

// TestCoreAgentDomainsFrontmatterFilters checks the same filter applied
// to a synthetic core-style agent: when the file has `domains:
// [engineering]`, a PM install drops it.
func TestCoreAgentDomainsFrontmatterFilters(t *testing.T) {
	const restrictedAgent = "---\nname: synthetic\ndomains: [engineering]\ndescription: test\n---\n# body\n"
	fakeDomain := fstest.MapFS{
		"agents/synthetic.md": &fstest.MapFile{Data: []byte(restrictedAgent)},
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(Options{
		ContentFS: fakeDomain,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: dir,
		Force:     true,
		Domain:    "pm",
	}); err != nil {
		t.Fatalf("install pm: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".claude", "agents", "synthetic.md")); err == nil {
		t.Error("agent with domains: [engineering] should not materialize in pm workspace")
	}
}

// TestAgentMatchesActiveDomain unit-tests the frontmatter parser
// directly against the shapes pack authors will write.
func TestAgentMatchesActiveDomain(t *testing.T) {
	cases := []struct {
		name    string
		content string
		domain  string
		want    bool
	}{
		{"no-frontmatter", "# Just markdown\n", "pm", true},
		{"no-domains-field", "---\nname: x\n---\n", "pm", true},
		{"inline-list-engineering", "---\nname: x\ndomains: [engineering]\n---\n", "pm", false},
		{"inline-list-engineering-eng", "---\nname: x\ndomains: [engineering]\n---\n", "engineering", true},
		{"inline-list-wildcard", "---\nname: x\ndomains: [\"*\"]\n---\n", "pm", true},
		{"inline-list-multi", "---\nname: x\ndomains: [engineering, pm]\n---\n", "pm", true},
		{"block-list", "---\nname: x\ndomains:\n  - engineering\n  - pm\n---\n", "sales", false},
		{"block-list-match", "---\nname: x\ndomains:\n  - pm\n---\n", "pm", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := agentMatchesActiveDomain([]byte(tc.content), tc.domain)
			if got != tc.want {
				t.Errorf("agentMatchesActiveDomain(domain=%s) = %v, want %v", tc.domain, got, tc.want)
			}
		})
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func repoRootFromHere() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
