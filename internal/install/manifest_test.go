package install

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	hero "github.com/hero-engine/hero"
)

// TestEnumerateContent_DedupesByName verifies the canonical set counts a
// name defined in both the domain (top) and core (bottom) layers exactly
// once — the deduped install set, not a raw file total.
func TestEnumerateContent_DedupesByName(t *testing.T) {
	domain := fstest.MapFS{
		"agents/a.md":            {Data: []byte("---\n---\nA")},
		"agents/shared.md":       {Data: []byte("---\n---\ndomain wins")},
		"commands/x.md":          {Data: []byte("X")},
		"skills/one/SKILL.md":    {Data: []byte("one")},
		"skills/shared/SKILL.md": {Data: []byte("domain wins")},
	}
	core := fstest.MapFS{
		"agents/shared.md":       {Data: []byte("---\n---\ncore loses")},
		"agents/b.md":            {Data: []byte("---\n---\nB")},
		"commands/x.md":          {Data: []byte("core loses")},
		"commands/y.md":          {Data: []byte("Y")},
		"skills/shared/SKILL.md": {Data: []byte("core loses")},
		"skills/two/SKILL.md":    {Data: []byte("two")},
	}
	merged := hero.OverlayFS(domain, core)

	m, err := EnumerateContent(merged, "engineering")
	if err != nil {
		t.Fatalf("EnumerateContent: %v", err)
	}

	// agents: a, b, shared (shared counted once → 3, not raw 4)
	if got := len(m.Agents); got != 3 {
		t.Errorf("agents: got %d (%v), want 3 deduped", got, m.Agents)
	}
	// commands: x, y (x counted once → 2, not raw 3)
	if got := len(m.Commands); got != 2 {
		t.Errorf("commands: got %d (%v), want 2 deduped", got, m.Commands)
	}
	// skills: one, two, shared (shared counted once → 3, not raw 4)
	if got := len(m.Skills); got != 3 {
		t.Errorf("skills: got %d (%v), want 3 deduped", got, m.Skills)
	}
}

// TestEnumerateContent_RespectsDomainFrontmatter verifies an agent whose
// `domains:` frontmatter excludes the active domain is not counted — the
// same filter install applies.
func TestEnumerateContent_RespectsDomainFrontmatter(t *testing.T) {
	content := fstest.MapFS{
		"agents/universal.md": {Data: []byte("---\nname: universal\n---\nbody")},
		"agents/pm-only.md":   {Data: []byte("---\nname: pm-only\ndomains: [pm]\n---\nbody")},
		"agents/eng-only.md":  {Data: []byte("---\nname: eng-only\ndomains: [engineering]\n---\nbody")},
	}

	m, err := EnumerateContent(content, "engineering")
	if err != nil {
		t.Fatalf("EnumerateContent: %v", err)
	}
	got := map[string]bool{}
	for _, a := range m.Agents {
		got[a] = true
	}
	if !got["universal"] || !got["eng-only"] {
		t.Errorf("expected universal + eng-only agents, got %v", m.Agents)
	}
	if got["pm-only"] {
		t.Errorf("pm-only agent should be excluded for engineering domain, got %v", m.Agents)
	}
}

// TestEnumerateContent_SkipsReadmes verifies directory READMEs are not
// counted as content — matching install's isContentReadme filter.
func TestEnumerateContent_SkipsReadmes(t *testing.T) {
	content := fstest.MapFS{
		"agents/README.md":   {Data: []byte("docs")},
		"agents/real.md":     {Data: []byte("---\n---\nagent")},
		"commands/README.md": {Data: []byte("docs")},
		"commands/go.md":     {Data: []byte("cmd")},
		"skills/README.md":   {Data: []byte("docs")},
		"skills/s/SKILL.md":  {Data: []byte("skill")},
	}
	m, err := EnumerateContent(content, "engineering")
	if err != nil {
		t.Fatalf("EnumerateContent: %v", err)
	}
	if len(m.Agents) != 1 || m.Agents[0] != "real" {
		t.Errorf("agents: got %v, want [real]", m.Agents)
	}
	if len(m.Commands) != 1 || m.Commands[0] != "go" {
		t.Errorf("commands: got %v, want [go]", m.Commands)
	}
	if len(m.Skills) != 1 || m.Skills[0] != "s" {
		t.Errorf("skills: got %v, want [s]", m.Skills)
	}
}

// TestEnumerateContent_MatchesInstalledFiles is the core anti-divergence
// guarantee: the manifest count equals what install actually copies for
// the default domain, using the real embedded core + engineering content.
// If a future change makes install bypass the shared selectors, the counts
// diverge and this test fails.
func TestEnumerateContent_MatchesInstalledFiles(t *testing.T) {
	domainFS, err := hero.DomainFS("engineering")
	if err != nil {
		t.Fatalf("DomainFS: %v", err)
	}
	contentFS := hero.OverlayFS(domainFS, hero.CoreFS())

	m, err := EnumerateContent(contentFS, "engineering")
	if err != nil {
		t.Fatalf("EnumerateContent: %v", err)
	}
	if len(m.Agents) == 0 || len(m.Commands) == 0 || len(m.Skills) == 0 {
		t.Fatalf("canonical counts must be non-zero, got agents=%d commands=%d skills=%d",
			len(m.Agents), len(m.Commands), len(m.Skills))
	}

	opts := Options{ContentFS: contentFS, Domain: "engineering"}
	result := &Result{}

	agentsDir := t.TempDir()
	if err := installFlat(opts, result, "agents", agentsDir); err != nil {
		t.Fatalf("installFlat agents: %v", err)
	}
	commandsDir := t.TempDir()
	if err := installFlat(opts, result, "commands", commandsDir); err != nil {
		t.Fatalf("installFlat commands: %v", err)
	}
	skillsDir := t.TempDir()
	if err := installSkillsNested(opts, result, skillsDir); err != nil {
		t.Fatalf("installSkillsNested skills: %v", err)
	}

	if got := countInstalledMD(t, agentsDir); got != len(m.Agents) {
		t.Errorf("agents: manifest %d != installed %d", len(m.Agents), got)
	}
	if got := countInstalledMD(t, commandsDir); got != len(m.Commands) {
		t.Errorf("commands: manifest %d != installed %d", len(m.Commands), got)
	}
	if got := countInstalledSkillDirs(t, skillsDir); got != len(m.Skills) {
		t.Errorf("skills: manifest %d != installed %d", len(m.Skills), got)
	}
}

// countInstalledMD counts flat .md files written into dir by installFlat.
func countInstalledMD(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			n++
		}
	}
	return n
}

// countInstalledSkillDirs counts <name>/SKILL.md directories written by
// installSkillsNested.
func countInstalledSkillDirs(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); err == nil {
			n++
		}
	}
	return n
}
