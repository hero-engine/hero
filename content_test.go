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
		{"legacy/root", legacyContent, "agents"},
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
		{"legacy/root", legacyContent, "skills"},
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
