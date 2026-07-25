package spec

import (
	"regexp"
	"sort"
	"strings"
)

// childLinkRe matches the `[slug](slug/spec.md)` links an initiative writes in
// its `## Child Specs & Sequence` table. Kept byte-identical to the pattern it
// replaced in internal/drive so folding the two rosters together does not shift
// what the drive path considers a declared child.
var childLinkRe = regexp.MustCompile(`\[([a-z0-9][a-z0-9-]*)\]\((?:\./)?([a-z0-9-]+)/spec\.md\)`)

// DeclaredChildren returns every child slug an initiative declares: the
// de-duplicated union of two rosters that used to be read independently.
//
//   - frontmatter `child` relations — what the completion gate
//     (InitiativeReadyToComplete) read, and
//   - `## Child Specs & Sequence` body-table links — what drive's child-set
//     builder read.
//
// Reading both from one function is the point: when the two paths derived
// "declared children" from different sources they could disagree by
// construction, which is how `hero spec verify` auto-completed an initiative
// at 1-of-4 children while `hero goal --check` agreed the run was done. A
// child declared in either place now counts in both.
//
// Frontmatter relations come first in declaration order, then table-only
// slugs. Targets are slug-normalized (a path-form target collapses to its
// folder slug) and each slug appears once. A self-reference is dropped.
func DeclaredChildren(init *Spec) []string {
	if init == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(slug string) {
		slug = strings.TrimSpace(slug)
		if slug == "" || slug == init.Slug || seen[slug] {
			return
		}
		seen[slug] = true
		out = append(out, slug)
	}
	// Only `child`. A `child-of` relation on this spec means "X is my parent"
	// — that is how every other consumer reads the kind — so counting it here
	// would list an initiative's own parent among its children. Harmless while
	// this roster only over-blocked a completion gate; now that drive's
	// intended-child set reads the same roster, it would make `done`
	// unreachable for a sub-initiative. The `child-of:`/`child_of:`
	// frontmatter keys normalize to `parent` at parse time, so no authored
	// form is lost.
	for _, r := range init.Relations {
		if r.Kind == "child" {
			add(normalizeRelTarget(r.Target))
		}
	}
	for _, slug := range childTableSlugs(init) {
		add(slug)
	}
	return out
}

// childTableSlugs parses the slugs named in an initiative's `## Child Specs &
// Sequence` section. A child listed in the table but not yet materialized on
// disk is declared, not absent — that is the completeness signal keeping a run
// (and now a completion) from short-circuiting past unbuilt children.
//
// Only a link whose text matches its folder segment counts, so prose links to
// other documents are ignored.
func childTableSlugs(init *Spec) []string {
	body := init.Sections["child specs & sequence"]
	if body == "" {
		// Fall back to any section whose header starts with "child". Sort the
		// candidates so a spec with more than one such section always yields
		// the same roster — map order must not decide a completion gate.
		keys := make([]string, 0, len(init.Sections))
		for k := range init.Sections {
			if strings.HasPrefix(k, "child") {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			body = init.Sections[keys[0]]
		}
	}
	if body == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range childLinkRe.FindAllStringSubmatch(body, -1) {
		slug := m[1]
		if slug == m[2] && !seen[slug] { // link text matches the folder slug
			seen[slug] = true
			out = append(out, slug)
		}
	}
	return out
}
