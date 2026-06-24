package synthesize

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

const developerNotesHeading = "## Developer Notes"

// StaleReport names an explainer that has fallen behind its cluster, with the
// completed specs that joined since it was last synthesized.
type StaleReport struct {
	ExplainerSlug string
	Title         string
	Path          string
	SourceSlugs   []string // current synthesized_from
	NewSlugs      []string // completed cluster members not yet covered
}

// StaleExplainers reports explainers whose cluster has gained completed specs
// since `last_synthesized`. Cluster membership for a new spec means it relates
// to a `synthesized_from` member or shares a parent with one (which captures
// new children of the source initiative).
func StaleExplainers(heroDir string) ([]StaleReport, error) {
	all, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	var reports []StaleReport
	for _, e := range all {
		if e.Type != spec.TypeExplainer {
			continue
		}
		news := newClusterSpecs(all, e)
		if len(news) == 0 {
			continue
		}
		reports = append(reports, StaleReport{
			ExplainerSlug: e.Slug,
			Title:         strings.Trim(e.Title, `"`),
			Path:          e.Path,
			SourceSlugs:   append([]string(nil), e.SynthesizedFrom...),
			NewSlugs:      news,
		})
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].ExplainerSlug < reports[j].ExplainerSlug })
	return reports, nil
}

// newClusterSpecs returns completed specs that belong to the explainer's
// cluster, are not already covered, and are newer than last_synthesized.
func newClusterSpecs(all []*spec.Spec, e *spec.Spec) []string {
	bySlug := make(map[string]*spec.Spec, len(all))
	for _, s := range all {
		bySlug[s.Slug] = s
	}
	covered := map[string]bool{}
	for _, s := range e.SynthesizedFrom {
		covered[s] = true
	}
	// Parents shared by current members (e.g. the source initiative).
	parentTargets := map[string]bool{}
	for _, slug := range e.SynthesizedFrom {
		m := bySlug[slug]
		if m == nil {
			continue
		}
		for _, r := range m.Relations {
			if r.Kind == "parent" {
				parentTargets[r.Target] = true
			}
		}
	}
	last := parseSynthDate(e.LastSynthesized)

	var news []string
	for _, s := range all {
		if s.Status != spec.StatusCompleted || s.Type == spec.TypeExplainer {
			continue
		}
		if covered[s.Slug] {
			continue
		}
		if !related(s, covered, parentTargets, bySlug) {
			continue
		}
		if !last.IsZero() {
			t := s.CompletedAt
			if t.IsZero() {
				t = s.ModifiedAt
			}
			if !t.After(last) {
				continue
			}
		}
		news = append(news, s.Slug)
	}
	sort.Strings(news)
	return news
}

// related reports whether s belongs to the cluster: it edges to a covered
// member, or shares a parent with the members.
func related(s *spec.Spec, covered, parentTargets map[string]bool, bySlug map[string]*spec.Spec) bool {
	for _, r := range s.Relations {
		if covered[r.Target] {
			return true
		}
		if r.Kind == "parent" && parentTargets[r.Target] {
			return true
		}
	}
	// A covered member pointing at s also counts.
	for slug := range covered {
		m := bySlug[slug]
		if m == nil {
			continue
		}
		for _, r := range m.Relations {
			if r.Target == s.Slug {
				return true
			}
		}
	}
	return false
}

func parseSynthDate(s string) time.Time {
	if t, err := time.Parse("2006-01-02", strings.TrimSpace(s)); err == nil {
		return t
	}
	return time.Time{}
}

// SplitDeveloperNotes splits explainer content into the generated body and the
// human-owned Developer Notes section (heading included). Notes is "" when the
// section is absent. The body keeps everything above the heading.
func SplitDeveloperNotes(content string) (body, notes string) {
	if strings.HasPrefix(content, developerNotesHeading) {
		return "", content
	}
	marker := "\n" + developerNotesHeading
	idx := strings.Index(content, marker)
	if idx == -1 {
		return content, ""
	}
	return content[:idx+1], content[idx+1:]
}

// StripFrontmatter returns the content after a leading `---`…`---` block.
func StripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	rest := content[3:]
	if i := strings.Index(rest, "\n---"); i != -1 {
		after := rest[i+4:]
		return strings.TrimLeft(after, "\n")
	}
	return content
}

const amendInstructions = `You are AMENDING an existing knowledge "explainer", not rewriting it. ` +
	`Below is the current explainer body, followed by new material that has landed ` +
	`since it was last synthesized. Update the body: ADD new behavior, and ` +
	`strike/correct ONLY what the new material contradicts. Leave everything else ` +
	`exactly as it is — including any human edits. Keep the same section structure ` +
	`(What it is, Surfaces / entry points, How it works, Data & state, Gotchas, ` +
	`Related decisions). Output only the markdown body starting at "## What it is" ` +
	`— no frontmatter, no top-level "# " title, and do NOT include a Developer ` +
	`Notes section.`

// AmendPrompt builds the system+user prompts to amend currentBody (the
// generated body, Developer Notes already split off) with the new material.
func (p *Packet) AmendPrompt(currentBody string) (system, user string) {
	user = "Current explainer body:\n\n" + strings.TrimSpace(currentBody) +
		"\n\n---\n\nNew material that has landed since:\n\n" + p.Material()
	return amendInstructions, user
}

// RenderAmended assembles the amended file: refreshed frontmatter (expanded
// synthesized_from, bumped last_synthesized), the amended body, and the
// preserved Developer Notes verbatim.
func (p *Packet) RenderAmended(amendedBody, devNotes, today string) string {
	out := p.Frontmatter(today) + "# " + p.Title + "\n\n" + strings.TrimSpace(amendedBody) + "\n"
	if strings.TrimSpace(devNotes) != "" {
		out += "\n" + strings.TrimSpace(devNotes) + "\n"
	}
	return out
}

// AmendScaffold is the no-LLM-key amendment: keep the current body, append the
// new material as a guidance comment for an agent/human to fold in, and
// preserve Developer Notes. Frontmatter provenance is refreshed.
func (p *Packet) AmendScaffold(currentBody, devNotes, today string) string {
	var b strings.Builder
	b.WriteString(p.Frontmatter(today))
	b.WriteString(strings.TrimSpace(currentBody))
	b.WriteString("\n\n<!-- AMENDMENT MATERIAL — fold the new specs into the sections above")
	b.WriteString(" (add new behavior, correct what's contradicted), then delete this comment.\n\n")
	b.WriteString(p.Material())
	b.WriteString("-->\n")
	if strings.TrimSpace(devNotes) != "" {
		b.WriteString("\n" + strings.TrimSpace(devNotes) + "\n")
	}
	return b.String()
}

// AmendTargets returns the expanded cluster slugs (current ∪ new) for an
// explainer slug, plus its file path, or an error if not found / not stale.
func AmendTargets(heroDir, explainerSlug string) (slugs []string, path string, err error) {
	all, err := spec.Discover(heroDir)
	if err != nil {
		return nil, "", err
	}
	var e *spec.Spec
	for _, s := range all {
		if s.Type == spec.TypeExplainer && s.Slug == explainerSlug {
			e = s
			break
		}
	}
	if e == nil {
		return nil, "", fmt.Errorf("no explainer %q found", explainerSlug)
	}
	news := newClusterSpecs(all, e)
	set := map[string]bool{}
	var out []string
	for _, s := range append(append([]string(nil), e.SynthesizedFrom...), news...) {
		if !set[s] {
			set[s] = true
			out = append(out, s)
		}
	}
	return out, e.Path, nil
}
