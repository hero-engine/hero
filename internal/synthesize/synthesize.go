// Package synthesize assembles the deterministic inputs for a feature
// "explainer" knowledge entry — the specs in a cluster, the git activity
// across their delivery window, and the decisions they reference — and
// renders them into a prompt (for LLM synthesis) or a scaffold (for an
// agent or human to fill). Prose generation itself lives with the caller:
// the CLI's LLM path or the in-session agent via the MCP tool.
//
// Part of the feature-knowledge-synthesis initiative (fks-on-demand-synthesizer).
package synthesize

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// Commit is a one-line git commit summary in the delivery window.
type Commit struct {
	Hash    string
	Subject string
	Date    string
}

// Packet is the assembled, deterministic input to a synthesis. It holds
// everything gathered from disk and git; prose generation consumes it.
type Packet struct {
	OutSlug       string       // slug for the explainer entry
	Title         string       // human title (from the dominant spec)
	Specs         []*spec.Spec // the resolved input specs
	Since         time.Time    // delivery window start
	Until         time.Time    // delivery window end
	Commits       []Commit     // commits in the window
	ChangedFiles  []string     // files touched across the window
	DecisionLinks []string     // decision-entry slugs referenced by the specs
}

// sectionSkeleton is the fixed "how it works" structure of an explainer.
// Mirrors core/skills/explainer-format/SKILL.md — keep the two aligned.
const sectionSkeleton = `## What it is

## Surfaces / entry points

## How it works

## Data & state

## Gotchas

## Related decisions

## Developer Notes
`

// Assemble resolves the slugs, computes the delivery window, gathers git
// activity, and collects referenced decisions. It fails loud on any
// unresolved slug and writes nothing — slug resolution is the caller's
// gate before generation.
func Assemble(heroDir, projectRoot string, slugs []string) (*Packet, error) {
	if len(slugs) == 0 {
		return nil, fmt.Errorf("synthesize: no spec slugs given")
	}
	all, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("synthesize: discovering specs: %w", err)
	}
	bySlug := make(map[string]*spec.Spec, len(all))
	for _, s := range all {
		bySlug[s.Slug] = s
	}

	p := &Packet{}
	for _, slug := range slugs {
		s := bySlug[slug]
		if s == nil {
			return nil, fmt.Errorf("synthesize: spec %q not found — nothing written", slug)
		}
		p.Specs = append(p.Specs, s)
	}

	// Dominant spec: an initiative if present (it names the feature),
	// otherwise the first input. Drives the out-slug and title.
	dominant := p.Specs[0]
	for _, s := range p.Specs {
		if s.Type == spec.TypeInitiative {
			dominant = s
			break
		}
	}
	p.OutSlug = dominant.Slug
	// Titles are often stored as quoted YAML scalars; strip the surrounding
	// quotes so the heading reads cleanly. Frontmatter re-quotes safely.
	p.Title = strings.Trim(dominant.Title, `"`)
	if p.Title == "" {
		p.Title = dominant.Slug
	}

	// Delivery window: earliest created → latest completed (or now if any
	// input is not yet completed).
	now := time.Now()
	var sawOpen bool
	for _, s := range p.Specs {
		if !s.CreatedAt.IsZero() && (p.Since.IsZero() || s.CreatedAt.Before(p.Since)) {
			p.Since = s.CreatedAt
		}
		if s.CompletedAt.IsZero() {
			sawOpen = true
		} else if s.CompletedAt.After(p.Until) {
			p.Until = s.CompletedAt
		}
	}
	if sawOpen || p.Until.IsZero() {
		p.Until = now
	}
	if p.Since.IsZero() {
		// No created dates — fall back to a wide window so git still helps.
		p.Since = p.Until.AddDate(0, -3, 0)
	}

	p.Commits = gitCommits(projectRoot, p.Since, p.Until)
	p.ChangedFiles = changedFiles(projectRoot, p.Commits)
	p.DecisionLinks = referencedDecisions(all, p.Specs)
	return p, nil
}

// referencedDecisions returns the slugs of decision entries that appear by
// slug in any input spec's body or relations. Bounded by the decision count.
func referencedDecisions(all, inputs []*spec.Spec) []string {
	var decisions []*spec.Spec
	for _, s := range all {
		if s.Type == spec.TypeDecision {
			decisions = append(decisions, s)
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, d := range decisions {
		for _, in := range inputs {
			if strings.Contains(in.RawContent, d.Slug) {
				if !seen[d.Slug] {
					seen[d.Slug] = true
					out = append(out, d.Slug)
				}
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// Material is the assembled context block shared by the LLM prompt and the
// MCP agent packet: the window, the specs, the git activity, and the
// decision links.
func (p *Packet) Material() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Feature: %s\n", p.Title)
	fmt.Fprintf(&b, "Delivery window: %s → %s\n\n",
		p.Since.Format("2006-01-02"), p.Until.Format("2006-01-02"))

	b.WriteString("## Source specs\n\n")
	for _, s := range p.Specs {
		fmt.Fprintf(&b, "### %s (%s)\n%s\n\n", s.Slug, s.Type, strings.TrimSpace(s.RawContent))
	}

	if len(p.Commits) > 0 {
		b.WriteString("## Commits in the window\n\n")
		for _, c := range p.Commits {
			fmt.Fprintf(&b, "- %s %s (%s)\n", c.Hash, c.Subject, c.Date)
		}
		b.WriteString("\n")
	}
	if len(p.ChangedFiles) > 0 {
		b.WriteString("## Files changed in the window\n\n")
		for _, f := range p.ChangedFiles {
			fmt.Fprintf(&b, "- %s\n", f)
		}
		b.WriteString("\n")
	}
	if len(p.DecisionLinks) > 0 {
		b.WriteString("## Decision entries referenced (link, don't restate)\n\n")
		for _, d := range p.DecisionLinks {
			fmt.Fprintf(&b, "- %s\n", d)
		}
		b.WriteString("\n")
	}
	return b.String()
}

const synthesisInstructions = `Write a knowledge "explainer" — a "how this feature works, as it exists now" ` +
	`document — from the material below. Use exactly these sections, in order: ` +
	`What it is, Surfaces / entry points, How it works, Data & state, Gotchas, ` +
	`Related decisions, Developer Notes. ` +
	`Describe how the shipped system actually works (use the commits and changed ` +
	`files as ground truth, not just the spec's intent). In "Related decisions", ` +
	`link the referenced decision slugs — do not restate their content. Leave the ` +
	`"Developer Notes" section empty: it is human-owned and must stay blank. ` +
	`Output only the markdown body, starting at "## What it is" — no frontmatter, ` +
	`no preamble.`

// Prompt returns the system + user prompts for LLM synthesis.
func (p *Packet) Prompt() (system, user string) {
	return synthesisInstructions, p.Material()
}

// Frontmatter renders the explainer's provenance frontmatter block.
func (p *Packet) Frontmatter(today string) string {
	var b strings.Builder
	b.WriteString("---\n")
	// Quote the title so em-dashes/colons stay valid YAML; escape any
	// embedded double-quotes.
	fmt.Fprintf(&b, "title: \"%s\"\n", strings.ReplaceAll(p.Title, `"`, `\"`))
	b.WriteString("type: explainer\n")
	b.WriteString("synthesized_from:\n")
	for _, s := range p.Specs {
		fmt.Fprintf(&b, "  - %s\n", s.Slug)
	}
	fmt.Fprintf(&b, "last_synthesized: %s\n", today)
	for _, s := range p.Specs {
		if s.Type == spec.TypeInitiative {
			fmt.Fprintf(&b, "source_initiative: %s\n", s.Slug)
			break
		}
	}
	b.WriteString("tags: []\n")
	b.WriteString("---\n")
	return b.String()
}

// Render wraps an LLM-generated body with the provenance frontmatter into
// the final explainer file content.
func (p *Packet) Render(body, today string) string {
	body = strings.TrimSpace(body)
	return p.Frontmatter(today) + "# " + p.Title + "\n\n" + body + "\n"
}

// Scaffold returns a ready-to-fill explainer: provenance frontmatter, the
// section skeleton, and the assembled material embedded as a guidance
// comment — for the no-LLM-key path, where an agent or human completes it.
func (p *Packet) Scaffold(today string) string {
	var b strings.Builder
	b.WriteString(p.Frontmatter(today))
	fmt.Fprintf(&b, "# %s\n\n", p.Title)
	b.WriteString(sectionSkeleton)
	b.WriteString("\n<!-- SYNTHESIS MATERIAL — fill the sections above from this, then delete this comment.\n\n")
	b.WriteString(p.Material())
	b.WriteString("-->\n")
	return b.String()
}

// AgentPacket is what the MCP tool hands back to an in-session agent: the
// instructions, the target path, and the assembled material, so the agent
// writes the explainer itself (the no-key default path).
func (p *Packet) AgentPacket(outPath, today string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", synthesisInstructions)
	fmt.Fprintf(&b, "Write the result to: %s\n", outPath)
	fmt.Fprintf(&b, "Prepend this frontmatter, then a `# %s` heading, then the body:\n\n", p.Title)
	b.WriteString(p.Frontmatter(today))
	b.WriteString("\n---\n\n")
	b.WriteString(p.Material())
	return b.String()
}

// gitCommits returns commit summaries in [since, until] at projectRoot. An
// empty repo (no commits yet) yields nil, not an error.
func gitCommits(projectRoot string, since, until time.Time) []Commit {
	cmd := exec.Command("git", "-C", projectRoot, "log",
		"--since="+since.Format("2006-01-02T15:04:05"),
		"--until="+until.Format("2006-01-02T15:04:05"),
		"--pretty=format:%h\t%s\t%aI",
		"--no-merges")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var commits []Commit
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		date := parts[2]
		if len(date) >= 10 {
			date = date[:10]
		}
		commits = append(commits, Commit{Hash: parts[0], Subject: parts[1], Date: date})
	}
	return commits
}

// changedFiles returns the de-duplicated set of files touched by the given
// commits, sorted for stable output.
func changedFiles(projectRoot string, commits []Commit) []string {
	seen := map[string]bool{}
	for _, c := range commits {
		cmd := exec.Command("git", "-C", projectRoot, "diff-tree",
			"--no-commit-id", "-r", "--name-only", c.Hash)
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				seen[line] = true
			}
		}
	}
	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}
