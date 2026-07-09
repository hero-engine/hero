package install

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// content.go — shared primitives for installing agent/command/skill content
// from the source filesystem into a target directory.
//
// installFlat   — writes each .md file directly into destDir (used by harnesses
//                 that read flat instruction files: Cursor, Copilot, Generic).
// installSkillsNested — writes each skill as destDir/<name>/SKILL.md per the
//                 Anthropic Agent Skills format. Used by harnesses whose Skill
//                 loader requires this directory layout (Claude Code, opencode,
//                 Codex). Includes legacy-flat-file cleanup so re-running
//                 install against a buggy prior install self-migrates.
// installSkillsFlat — flattens each skill (canonical `skills/<name>/SKILL.md`
//                 source layout) to a single destDir/<name>.md file. Used by
//                 harnesses that only read flat instruction files (Cursor
//                 rules) and therefore can't consume the nested layout.

func installFlat(opts Options, result *Result, kind, destDir string) error {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		return fmt.Errorf("no content source available")
	}

	entries, err := fs.ReadDir(srcFS, kind)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	activeDomain := opts.Domain
	if activeDomain == "" {
		activeDomain = "engineering"
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || isContentReadme(entry.Name()) {
			continue
		}
		srcPath := kind + "/" + entry.Name()

		// Agents may declare a `domains:` frontmatter field that
		// restricts which active-domain workspaces they're materialized
		// into. Absent field means "all domains" (today's default). The
		// universal core + pack overlay merges happen at the FS level
		// upstream of this loop (OverlayFS in internal/cli/install.go),
		// so the filter is applied uniformly to whatever the merged FS
		// surfaces — pack files shadowing core files inherit the pack's
		// frontmatter, not the core file's.
		if kind == "agents" {
			data, readErr := fs.ReadFile(srcFS, srcPath)
			if readErr == nil && !agentMatchesActiveDomain(data, activeDomain) {
				continue
			}
		}

		dst := filepath.Join(destDir, entry.Name())

		if err := copyFileFromFS(opts, result, srcFS, srcPath, dst); err != nil {
			return err
		}
	}

	return nil
}

// agentMatchesActiveDomain returns true if the agent file's frontmatter
// has no `domains:` field, or the field includes the active domain or
// the `*` wildcard. The check is a deliberately small parser that reads
// only the leading `---` YAML block — no full YAML dependency for one
// list field.
func agentMatchesActiveDomain(content []byte, activeDomain string) bool {
	domains, ok := readAgentDomainsFrontmatter(content)
	if !ok {
		return true
	}
	for _, d := range domains {
		if d == "*" || d == activeDomain {
			return true
		}
	}
	return false
}

// readAgentDomainsFrontmatter parses the leading `---` YAML frontmatter
// block (if any) and returns the parsed `domains:` list. ok == false
// means the field is absent — caller should treat as "all domains".
//
// Supports two shapes:
//
//	domains: [engineering, pm]
//	domains:
//	  - engineering
//	  - pm
func readAgentDomainsFrontmatter(content []byte) (domains []string, ok bool) {
	s := bufio.NewScanner(strings.NewReader(string(content)))
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	if !s.Scan() {
		return nil, false
	}
	if strings.TrimSpace(s.Text()) != "---" {
		return nil, false
	}

	var inList bool
	for s.Scan() {
		line := s.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			return domains, ok
		}
		if inList {
			if strings.HasPrefix(strings.TrimLeft(line, " \t"), "- ") {
				item := strings.TrimSpace(strings.TrimPrefix(strings.TrimLeft(line, " \t"), "-"))
				item = strings.Trim(item, `"'`)
				domains = append(domains, item)
				continue
			}
			inList = false
		}
		if !strings.HasPrefix(line, "domains:") && !strings.HasPrefix(line, "domains :") {
			continue
		}
		ok = true
		value := strings.TrimSpace(strings.TrimPrefix(line, "domains:"))
		if value == "" {
			inList = true
			continue
		}
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			inner := strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
			for _, item := range strings.Split(inner, ",") {
				item = strings.TrimSpace(item)
				item = strings.Trim(item, `"'`)
				if item != "" {
					domains = append(domains, item)
				}
			}
		}
	}
	return domains, ok
}

func installSkillsNested(opts Options, result *Result, destDir string) error {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		return fmt.Errorf("no content source available")
	}

	entries, err := fs.ReadDir(srcFS, "skills")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Clean up any legacy flat-file skills at destDir/<name>.md from prior
	// installs that wrote skills as flat files. Anthropic's SKILL.md format
	// requires <name>/SKILL.md directory layout — flat files are silently
	// invisible to the Skill loader even when content is otherwise correct.
	if err := cleanupFlatSkills(opts, destDir); err != nil {
		return fmt.Errorf("cleaning legacy flat skills: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()

		// Canonical source layout: `skills/<name>/SKILL.md`. Read directly
		// from there, write to `<destDir>/<name>/SKILL.md`.
		if entry.IsDir() {
			srcSkill := "skills/" + name + "/SKILL.md"
			if _, err := fs.Stat(srcFS, srcSkill); err != nil {
				continue // directory without SKILL.md — not a skill
			}
			dst := filepath.Join(destDir, name, "SKILL.md")
			if err := copyFileFromFS(opts, result, srcFS, srcSkill, dst); err != nil {
				return err
			}
			continue
		}

		// Legacy flat layout: `skills/<name>.md`. Still supported for
		// backward compat; rendered into the nested layout at dest.
		if !strings.HasSuffix(name, ".md") || isContentReadme(name) {
			continue
		}
		base := strings.TrimSuffix(name, ".md")
		srcPath := "skills/" + name
		dst := filepath.Join(destDir, base, "SKILL.md")
		if err := copyFileFromFS(opts, result, srcFS, srcPath, dst); err != nil {
			return err
		}
	}

	return nil
}

// installSkillsFlat writes each skill as a single destDir/<name>.md file,
// flattening the canonical `skills/<name>/SKILL.md` source layout. Legacy
// flat source files `skills/<name>.md` pass through unchanged.
func installSkillsFlat(opts Options, result *Result, destDir string) error {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		return fmt.Errorf("no content source available")
	}

	entries, err := fs.ReadDir(srcFS, "skills")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			srcSkill := "skills/" + name + "/SKILL.md"
			if _, err := fs.Stat(srcFS, srcSkill); err != nil {
				continue // directory without SKILL.md — not a skill
			}
			dst := filepath.Join(destDir, name+".md")
			if err := copyFileFromFS(opts, result, srcFS, srcSkill, dst); err != nil {
				return err
			}
			continue
		}

		if !strings.HasSuffix(name, ".md") || isContentReadme(name) {
			continue
		}
		dst := filepath.Join(destDir, name)
		if err := copyFileFromFS(opts, result, srcFS, "skills/"+name, dst); err != nil {
			return err
		}
	}

	return nil
}

// isContentReadme reports whether a source entry is a directory README —
// documentation for humans browsing the content tree, not installable
// content. The pm and sales domains ship README.md alongside their agents,
// commands, and skills; installing those would materialize pseudo-agents.
func isContentReadme(name string) bool {
	return strings.EqualFold(name, "README.md")
}

// cleanupFlatSkills removes flat *.md files at destDir written by prior
// (buggy) installs that used installFlat for skills. The Anthropic SKILL.md
// directory layout supersedes them; leaving the flat copies behind clutters
// the harness directory and risks confusion. Subdirectories (the correct
// layout) are left untouched.
func cleanupFlatSkills(opts Options, destDir string) error {
	if opts.DryRun {
		return nil
	}
	entries, err := os.ReadDir(destDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		full := filepath.Join(destDir, name)
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing legacy flat skill %s: %w", full, err)
		}
	}
	return nil
}
