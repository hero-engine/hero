package install

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// manifest.go — canonical content enumeration shared by the install
// pipeline and the docs-freshness checker.
//
// The install functions in content.go (installFlat, installSkillsNested,
// installSkillsFlat) and the ContentManifest returned by EnumerateContent
// both route through the same two selectors — selectFlatContent and
// selectSkillContent — so "what the checker counts" and "what install
// copies" cannot diverge by construction. Any change to the selection
// rules moves both in lockstep.

// ContentManifest is the canonical, deduped set of agent, command, and
// skill names that `hero install` materializes for a given domain. It is
// the single source of truth for "how much surface an install ships" —
// the docs-freshness checker counts these so its numbers can never drift
// from what install actually copies.
type ContentManifest struct {
	Agents   []string
	Commands []string
	Skills   []string
}

// EnumerateContent walks the merged content FS (universal core overlaid
// by the active domain) and returns the canonical deduped install set for
// domain. It applies the exact selection rules install uses, so the
// returned counts equal what install writes. An empty domain resolves to
// "engineering". Callers build contentFS the same way the install CLI
// does: hero.OverlayFS(hero.DomainFS(domain), hero.CoreFS()).
func EnumerateContent(contentFS fs.FS, domain string) (ContentManifest, error) {
	if domain == "" {
		domain = "engineering"
	}
	agents, err := selectFlatContent(contentFS, "agents", domain)
	if err != nil {
		return ContentManifest{}, fmt.Errorf("enumerating agents: %w", err)
	}
	commands, err := selectFlatContent(contentFS, "commands", domain)
	if err != nil {
		return ContentManifest{}, fmt.Errorf("enumerating commands: %w", err)
	}
	skills, err := selectSkillContent(contentFS)
	if err != nil {
		return ContentManifest{}, fmt.Errorf("enumerating skills: %w", err)
	}
	return ContentManifest{
		Agents:   trimMDNames(agents),
		Commands: trimMDNames(commands),
		Skills:   skillNames(skills),
	}, nil
}

// skillSource pairs a skill's canonical name with the path to its
// SKILL.md content within the source FS.
type skillSource struct {
	Name       string // skill name (source dir name or flat-file stem)
	SourcePath string // path within srcFS to the SKILL.md content bytes
}

// selectFlatContent returns the .md filenames under kind/ in srcFS that
// install would materialize for activeDomain, applying the same skip
// rules install uses: directories, non-.md files, and directory READMEs
// are excluded; agents additionally honor the `domains:` frontmatter
// filter. Returned names keep the .md extension and follow srcFS ReadDir
// order (OverlayFS already merges core+domain and sorts). A missing kind/
// directory yields an empty slice, not an error.
func selectFlatContent(srcFS fs.FS, kind, activeDomain string) ([]string, error) {
	entries, err := fs.ReadDir(srcFS, kind)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || isContentReadme(entry.Name()) {
			continue
		}
		// Agents may declare a `domains:` frontmatter field restricting
		// which active-domain workspaces they materialize into. Absent
		// field means "all domains". A read error falls through to
		// "include" — the same lenient behavior install applies.
		if kind == "agents" {
			data, readErr := fs.ReadFile(srcFS, kind+"/"+entry.Name())
			if readErr == nil && !agentMatchesActiveDomain(data, activeDomain) {
				continue
			}
		}
		out = append(out, entry.Name())
	}
	return out, nil
}

// selectSkillContent returns the skills under skills/ in srcFS that
// install would materialize. Two source shapes are recognized, matching
// installSkillsNested/installSkillsFlat: the canonical
// `skills/<name>/SKILL.md` directory layout, and the legacy flat
// `skills/<name>.md` file. Directory READMEs are excluded. A missing
// skills/ directory yields an empty slice, not an error.
func selectSkillContent(srcFS fs.FS) ([]skillSource, error) {
	entries, err := fs.ReadDir(srcFS, "skills")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []skillSource
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			srcSkill := "skills/" + name + "/SKILL.md"
			if _, statErr := fs.Stat(srcFS, srcSkill); statErr != nil {
				continue // directory without SKILL.md — not a skill
			}
			out = append(out, skillSource{Name: name, SourcePath: srcSkill})
			continue
		}
		if !strings.HasSuffix(name, ".md") || isContentReadme(name) {
			continue
		}
		out = append(out, skillSource{Name: strings.TrimSuffix(name, ".md"), SourcePath: "skills/" + name})
	}
	return out, nil
}

// trimMDNames strips the .md extension from flat content filenames,
// yielding bare names for the manifest.
func trimMDNames(files []string) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = strings.TrimSuffix(f, ".md")
	}
	return out
}

// skillNames projects skill sources to their bare names.
func skillNames(skills []skillSource) []string {
	out := make([]string, len(skills))
	for i, s := range skills {
		out[i] = s.Name
	}
	return out
}
