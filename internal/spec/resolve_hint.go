package spec

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveOrHint finds the spec whose slug matches `slug`. On an exact or
// case-insensitive match it returns (spec, ""). On no match it returns
// (nil, hint) where hint is a human-readable next-action string, or
// (nil, "") when no signal applies (caller falls back to its bare message).
//
// Resolution order:
//  1. Exact slug match.
//  2. Case-insensitive slug match (resolves silently).
//  3. Initiative-child detection — the slug is a first-column entry of some
//     initiative's children table but has no spec of its own on disk.
//  4. Fuzzy "did you mean" — discoverable slugs within Levenshtein distance 2.
//  5. No signal — empty hint, caller emits its bare not-found message.
//
// Initiative-child (step 3) precedes fuzzy (step 4) so the more actionable
// message wins on a tie.
func ResolveOrHint(slug string, specs []*Spec) (*Spec, string) {
	// 1. Exact match — preserves today's fast path.
	for _, s := range specs {
		if s.Slug == slug {
			return s, ""
		}
	}

	// 2. Case-insensitive match — resolve silently.
	for _, s := range specs {
		if strings.EqualFold(s.Slug, slug) {
			return s, ""
		}
	}

	// 3. Initiative-child detection.
	for _, s := range specs {
		if s.Type != TypeInitiative {
			continue
		}
		body, ok := childrenSectionBody(s)
		if !ok {
			continue
		}
		if childTableHasSlug(body, slug) {
			return nil, fmt.Sprintf(
				"`%s` is listed as a child of initiative `%s` but hasn't been designed into its own spec yet — run /design to materialize it before verifying.",
				slug, s.Slug,
			)
		}
	}

	// 4. Fuzzy "did you mean" — closest discoverable slugs within distance 2.
	if suggestions := fuzzySuggestions(slug, specs); len(suggestions) > 0 {
		quoted := make([]string, len(suggestions))
		for i, sug := range suggestions {
			quoted[i] = fmt.Sprintf("`%s`", sug)
		}
		return nil, fmt.Sprintf("no spec `%s` found — did you mean %s?", slug, strings.Join(quoted, ", "))
	}

	// 5. No signal — caller emits the unchanged bare message.
	return nil, ""
}

// childrenSectionBody returns the body of the spec's children section, looking
// up by prefix so heading variants like "## Children — six features" (keyed as
// "children — six features") still match. The canonical key is "children".
func childrenSectionBody(s *Spec) (string, bool) {
	for key, body := range s.Sections {
		if key == "children" || strings.HasPrefix(key, "children") {
			return body, true
		}
	}
	return "", false
}

// childTableHasSlug reports whether any table row in body has a first column
// equal (case-insensitively) to slug. Markdown-link cells "[text](path)" are
// normalized to their text. Separator rows ("|---|") are skipped. Pipe
// splitting is delegated to splitTableRow so leading/trailing pipes are
// tolerated.
func childTableHasSlug(body, slug string) bool {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.Contains(trimmed, "|") {
			continue
		}
		cells := splitTableRow(trimmed)
		if len(cells) == 0 {
			continue
		}
		first := normalizeCell(cells[0])
		if first == "" {
			continue
		}
		// Skip separator rows like "---".
		if strings.Trim(first, "-:") == "" {
			continue
		}
		if strings.EqualFold(first, slug) {
			return true
		}
	}
	return false
}

// normalizeCell reduces a table cell to its comparable slug text: it strips
// markdown-link syntax "[text](path)" down to "text" and trims backticks and
// surrounding whitespace. Falls back to the raw (trimmed) cell.
func normalizeCell(cell string) string {
	cell = strings.TrimSpace(cell)
	if strings.HasPrefix(cell, "[") {
		if close := strings.Index(cell, "]"); close > 0 {
			cell = cell[1:close]
		}
	}
	return strings.TrimSpace(strings.Trim(cell, "`"))
}

// fuzzySuggestions returns up to 3 discoverable slugs within Levenshtein
// distance 2 of slug, sorted ascending by distance. For very short slugs
// (≤3 chars) the closest match must be strictly closer than the slug length
// to avoid over-matching noise.
func fuzzySuggestions(slug string, specs []*Spec) []string {
	type cand struct {
		slug string
		dist int
	}
	var cands []cand
	for _, s := range specs {
		if s.Slug == "" {
			continue
		}
		d := levenshtein(strings.ToLower(slug), strings.ToLower(s.Slug))
		if d <= 2 {
			cands = append(cands, cand{slug: s.Slug, dist: d})
		}
	}
	if len(cands) == 0 {
		return nil
	}

	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].dist < cands[j].dist
	})

	// Guard short slugs against over-matching: require the closest match to be
	// strictly nearer than the slug length.
	if len([]rune(slug)) <= 3 && cands[0].dist >= len([]rune(slug)) {
		return nil
	}

	var out []string
	for _, c := range cands {
		out = append(out, c.slug)
		if len(out) == 3 {
			break
		}
	}
	return out
}

// levenshtein computes the edit distance between a and b using an O(n*m)
// two-row dynamic program.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = del
			if ins < curr[j] {
				curr[j] = ins
			}
			if sub < curr[j] {
				curr[j] = sub
			}
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
