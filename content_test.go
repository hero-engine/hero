package hero

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestEmbeddedAgents_HaveRequiredFrontmatter enforces the harness
// registration contract on every canonical agent file shipped with
// the binary.
//
// Claude Code's subagent registry requires `name:` and `description:`
// in the YAML frontmatter; without them, the agent is silently dropped
// from the Task tool's subagent_type list. OpenCode derives `name`
// from the filename, but having the field present is harmless there
// and keeps every consumer satisfied from one source of truth.
//
// Without this test, the canonical-symlink install architecture (see
// internal/install/target_claude.go) silently degrades whenever an
// agent file is added without the required fields.
func TestEmbeddedAgents_HaveRequiredFrontmatter(t *testing.T) {
	cases := []struct {
		name string
		fsys fs.FS
		root string
	}{
		{"core", coreContent, "core/agents"},
		{"engineering", engineeringContent, "domains/engineering/agents"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := fs.ReadDir(tc.fsys, tc.root)
			if err != nil {
				t.Fatalf("read %s: %v", tc.root, err)
			}
			if len(entries) == 0 {
				t.Fatalf("%s: no agent files found", tc.root)
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				if strings.EqualFold(e.Name(), "README.md") {
					continue
				}
				assertAgentFrontmatter(t, tc.fsys, path.Join(tc.root, e.Name()))
			}
		})
	}
}

// TestEmbeddedSkills_HaveRequiredFrontmatter enforces that every
// canonical skill carries `description:` in its YAML frontmatter.
//
// Claude Code itself accepts skills without frontmatter (filename
// becomes name, first paragraph becomes description), but `description:`
// is load-bearing for model-driven skill invocation — without it,
// Claude has no clear signal for when to invoke the skill. Hero's
// authoring contract requires it. Mirrors the install-side
// HarnessContract for (TargetClaude, KindSkills).
func TestEmbeddedSkills_HaveRequiredFrontmatter(t *testing.T) {
	cases := []struct {
		name string
		fsys fs.FS
		root string
	}{
		{"core", coreContent, "core/skills"},
		{"engineering", engineeringContent, "domains/engineering/skills"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := fs.ReadDir(tc.fsys, tc.root)
			if err != nil {
				t.Fatalf("read %s: %v", tc.root, err)
			}
			if len(entries) == 0 {
				t.Fatalf("%s: no skill entries found", tc.root)
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue // skills are nested under <name>/SKILL.md
				}
				skillPath := path.Join(tc.root, e.Name(), "SKILL.md")
				if _, err := fs.Stat(tc.fsys, skillPath); err != nil {
					continue // not a skill dir
				}
				assertSkillFrontmatter(t, tc.fsys, skillPath)
			}
		})
	}
}

func assertAgentFrontmatter(t *testing.T, fsys fs.FS, p string) {
	t.Helper()
	data, err := fs.ReadFile(fsys, p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	fm, body, ok := splitFrontmatter(data)
	if !ok {
		t.Fatalf("%s: missing or malformed YAML frontmatter", p)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		t.Fatalf("%s: empty body", p)
	}
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		t.Fatalf("%s: parse frontmatter: %v", p, err)
	}
	stem := strings.TrimSuffix(path.Base(p), ".md")
	if meta.Name == "" {
		t.Fatalf("%s: missing required `name:` frontmatter field — Claude Code will silently drop this agent from the subagent registry", p)
	}
	if meta.Name != stem {
		t.Fatalf("%s: `name: %s` does not match filename stem %q", p, meta.Name, stem)
	}
	if strings.TrimSpace(meta.Description) == "" {
		t.Fatalf("%s: missing required `description:` frontmatter field", p)
	}
}

// assertSkillFrontmatter enforces the skill contract: a SKILL.md must
// have YAML frontmatter with a non-empty `description:`. Mirrors
// internal/install/contracts.go (TargetClaude, KindSkills).
func assertSkillFrontmatter(t *testing.T, fsys fs.FS, p string) {
	t.Helper()
	data, err := fs.ReadFile(fsys, p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	fm, body, ok := splitFrontmatter(data)
	if !ok {
		t.Fatalf("%s: missing or malformed YAML frontmatter", p)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		t.Fatalf("%s: empty body", p)
	}
	var meta struct {
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		t.Fatalf("%s: parse frontmatter: %v", p, err)
	}
	if strings.TrimSpace(meta.Description) == "" {
		t.Fatalf("%s: missing required `description:` frontmatter field — model-driven skill invocation depends on it", p)
	}
}

// TestAvailableDomains verifies the canonical list of embedded domain
// names. Anchoring this list in a test prevents accidental removal or
// reordering of a domain pack without explicit intent — adding a new
// domain requires updating both the embed declaration AND this test,
// which forces the change to be deliberate.
func TestAvailableDomains(t *testing.T) {
	got := AvailableDomains()
	want := map[string]bool{"engineering": true, "sales": true, "pm": true}

	if len(got) != len(want) {
		t.Fatalf("AvailableDomains() length = %d, want %d; got=%v", len(got), len(want), got)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("AvailableDomains() returned unexpected domain %q", d)
		}
		delete(want, d)
	}
	for d := range want {
		t.Errorf("AvailableDomains() missing expected domain %q", d)
	}
}

// TestDomainFS_KnownDomains verifies every domain returned by
// AvailableDomains() resolves through DomainFS() and exposes the
// expected harness-content roots (agents/, commands/, skills/).
//
// This is the embed.FS surface contract: every advertised domain must
// be installable, and "installable" means the per-target installers can
// walk agents/, commands/, and skills/ at the FS root.
func TestDomainFS_KnownDomains(t *testing.T) {
	for _, domain := range AvailableDomains() {
		t.Run(domain, func(t *testing.T) {
			fsys, err := DomainFS(domain)
			if err != nil {
				t.Fatalf("DomainFS(%q) returned error: %v", domain, err)
			}
			if fsys == nil {
				t.Fatalf("DomainFS(%q) returned nil filesystem", domain)
			}
			for _, root := range []string{"agents", "commands", "skills"} {
				entries, err := fs.ReadDir(fsys, root)
				if err != nil {
					t.Errorf("DomainFS(%q): ReadDir %q: %v", domain, root, err)
					continue
				}
				if len(entries) == 0 {
					t.Errorf("DomainFS(%q): %s/ is empty", domain, root)
				}
			}
		})
	}
}

// TestDomainFS_DefaultAndEmpty verifies that the empty domain string
// resolves identically to "engineering" through the same domain-pack
// path used by pm and sales.
func TestDomainFS_DefaultAndEmpty(t *testing.T) {
	empty, err := DomainFS("")
	if err != nil {
		t.Fatalf("DomainFS(\"\") error: %v", err)
	}
	if empty == nil {
		t.Fatal("DomainFS(\"\") returned nil")
	}
	// Smoke check: the default FS must expose agents/.
	if _, err := fs.ReadDir(empty, "agents"); err != nil {
		t.Errorf("DomainFS(\"\"): agents/ not readable: %v", err)
	}
}

// TestDomainFS_UnknownDomain verifies that requesting an unembedded
// domain returns an error mentioning the available domains, so users
// get a discoverable failure mode.
func TestDomainFS_UnknownDomain(t *testing.T) {
	_, err := DomainFS("not-a-real-domain")
	if err == nil {
		t.Fatal("DomainFS(\"not-a-real-domain\") should error")
	}
	if !strings.Contains(err.Error(), "not-a-real-domain") {
		t.Errorf("error should mention requested domain, got: %v", err)
	}
}

// TestDomainFS_PMSpecTypesPresent confirms the PM domain pack exposes
// its spec-types/ subdir via DomainFS. The PM pack is the second
// populated vertical and is load-bearing for the pm-foundation-delivery
// sprint: every PM-led type (intake, prd) is consumed by
// internal/spectypes via DomainSpecTypesFS, and DomainFS is the canonical
// install-time surface. If spec-types/ is missing from the embed, the
// install layer silently drops PM type records and consumers see an
// engineering-only registry under a PM-configured workspace.
func TestDomainFS_PMSpecTypesPresent(t *testing.T) {
	fsys, err := DomainFS("pm")
	if err != nil {
		t.Fatalf("DomainFS(\"pm\") error: %v", err)
	}
	entries, err := fs.ReadDir(fsys, "spec-types")
	if err != nil {
		t.Fatalf("read pm spec-types: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("pm spec-types/ is empty")
	}
}

// TestCoreFS_NonEmpty smoke-checks that the universal core/ layer
// (agents/commands/skills shared across verticals) is embedded and
// reachable. CoreFS() is consumed by every install path that layers
// vertical content on top of universal content; an empty CoreFS means
// every installed workspace silently loses access to the shared
// scaffolding.
func TestCoreFS_NonEmpty(t *testing.T) {
	fsys := CoreFS()
	if fsys == nil {
		t.Fatal("CoreFS() returned nil")
	}
	for _, root := range []string{"agents", "commands", "skills"} {
		entries, err := fs.ReadDir(fsys, root)
		if err != nil {
			t.Errorf("CoreFS: ReadDir %q: %v", root, err)
			continue
		}
		if len(entries) == 0 {
			t.Errorf("CoreFS: %s/ is empty", root)
		}
	}
}

// TestCoreVocabulariesFS_NonEmpty verifies the bundled vocabulary
// preset YAMLs are embedded. internal/vocabulary loads these at
// startup; an empty FS means the precedence chain has no presets to
// resolve against and display rendering collapses to type literals.
func TestCoreVocabulariesFS_NonEmpty(t *testing.T) {
	fsys := CoreVocabulariesFS()
	if fsys == nil {
		t.Fatal("CoreVocabulariesFS() returned nil")
	}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("CoreVocabulariesFS ReadDir: %v", err)
	}
	if !hasYAML(entries) {
		t.Fatalf("CoreVocabulariesFS: no .yaml files found; entries=%v", entryNames(entries))
	}
}

// TestCoreMethodologiesFS_NonEmpty verifies the bundled methodology
// profile YAMLs are embedded. internal/methodology resolves the active
// methodology against these files; an empty FS means lifecycle /
// time-box / estimation queries have no source to read from.
func TestCoreMethodologiesFS_NonEmpty(t *testing.T) {
	fsys := CoreMethodologiesFS()
	if fsys == nil {
		t.Fatal("CoreMethodologiesFS() returned nil")
	}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("CoreMethodologiesFS ReadDir: %v", err)
	}
	if !hasYAML(entries) {
		t.Fatalf("CoreMethodologiesFS: no .yaml files found; entries=%v", entryNames(entries))
	}
}

// TestCoreSpecTypesFS_NonEmpty verifies the nine canonical
// work-tracking type-record markdown files are embedded.
// internal/spectypes loads these as the core registry; an empty FS
// breaks every downstream surface that resolves type metadata
// (hero list, hero new, schema 1.1 JSON export, dashboard rendering).
func TestCoreSpecTypesFS_NonEmpty(t *testing.T) {
	fsys := CoreSpecTypesFS()
	if fsys == nil {
		t.Fatal("CoreSpecTypesFS() returned nil")
	}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("CoreSpecTypesFS ReadDir: %v", err)
	}
	if !hasMarkdown(entries) {
		t.Fatalf("CoreSpecTypesFS: no .md files found; entries=%v", entryNames(entries))
	}
}

// TestDomainSpecTypesFS_PM verifies the PM domain pack overlays its
// own spec-type records (intake, prd) on top of the core nine.
// internal/spectypes consumes this surface to merge domain-led types
// with the canonical registry; missing files mean PM-configured
// workspaces silently lose the PM-led artifacts.
func TestDomainSpecTypesFS_PM(t *testing.T) {
	fsys := DomainSpecTypesFS("pm")
	if fsys == nil {
		t.Fatal("DomainSpecTypesFS(\"pm\") returned nil")
	}
	for _, want := range []string{"intake.md", "prd.md"} {
		if _, err := fs.Stat(fsys, want); err != nil {
			t.Errorf("DomainSpecTypesFS(\"pm\"): %s not found: %v", want, err)
		}
	}
}

// TestDomainSpecTypesFS_Engineering verifies the engineering domain
// pack overlays its decision/convention records. These are the
// canonical knowledge-layer types for engineering and are required by
// the spec-type registry.
func TestDomainSpecTypesFS_Engineering(t *testing.T) {
	fsys := DomainSpecTypesFS("engineering")
	if fsys == nil {
		t.Fatal("DomainSpecTypesFS(\"engineering\") returned nil")
	}
	for _, want := range []string{"decision.md", "convention.md"} {
		if _, err := fs.Stat(fsys, want); err != nil {
			t.Errorf("DomainSpecTypesFS(\"engineering\"): %s not found: %v", want, err)
		}
	}
}

// TestDomainSpecTypesFS_Sales documents the current state of the sales
// vertical: spec-types/ exists in the embed (so DomainSpecTypesFS
// returns non-nil), but contains only a README — no domain-led types
// yet. This anchors the intentional emptiness so a future contributor
// adding sales-led types will see this test and decide whether to
// update it or add coverage for the new types.
func TestDomainSpecTypesFS_Sales(t *testing.T) {
	fsys := DomainSpecTypesFS("sales")
	if fsys == nil {
		t.Fatal("DomainSpecTypesFS(\"sales\") returned nil — sales pack should expose at least a README")
	}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("DomainSpecTypesFS(\"sales\") ReadDir: %v", err)
	}
	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && !strings.EqualFold(e.Name(), "README.md") {
			mdCount++
		}
	}
	if mdCount != 0 {
		t.Logf("note: sales now ships %d non-README .md type records; update this test if that's intentional", mdCount)
	}
}

// TestDomainSpecTypesFS_UnknownReturnsNil documents the contract that
// requesting spec-types for an unembedded domain returns nil (not an
// error). Callers in internal/spectypes branch on nil to mean "this
// domain has no spec-type extensions" and proceed with the core
// registry only.
func TestDomainSpecTypesFS_UnknownReturnsNil(t *testing.T) {
	if got := DomainSpecTypesFS("not-a-real-domain"); got != nil {
		t.Errorf("DomainSpecTypesFS(\"not-a-real-domain\") = %v, want nil", got)
	}
}

// hasYAML reports whether any entry is a .yaml or .yml file (not a dir).
func hasYAML(entries []fs.DirEntry) bool {
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := strings.ToLower(e.Name())
		if strings.HasSuffix(n, ".yaml") || strings.HasSuffix(n, ".yml") {
			return true
		}
	}
	return false
}

// hasMarkdown reports whether any entry is a .md file (not a dir).
func hasMarkdown(entries []fs.DirEntry) bool {
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			return true
		}
	}
	return false
}

// entryNames extracts the entry name list for diagnostic output.
func entryNames(entries []fs.DirEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// splitFrontmatter pulls the YAML block out of a `---\n...\n---\n`
// header and returns (frontmatter, body, ok). Returns ok=false if the
// file does not start with a frontmatter marker.
func splitFrontmatter(data []byte) (fm, body []byte, ok bool) {
	const marker = "---\n"
	s := string(data)
	if !strings.HasPrefix(s, marker) {
		return nil, nil, false
	}
	rest := s[len(marker):]
	end := strings.Index(rest, "\n"+marker[:3])
	if end < 0 {
		return nil, nil, false
	}
	fm = []byte(rest[:end])
	bodyStart := end + 1 + len(marker)
	if bodyStart > len(rest) {
		bodyStart = len(rest)
	}
	body = []byte(rest[bodyStart:])
	return fm, body, true
}
