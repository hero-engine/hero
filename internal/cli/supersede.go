package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var supersedeCmd = &cobra.Command{
	Use:   "supersede <old-slug>",
	Short: "Mark a spec as superseded by another",
	Long: `Wires up the soft-archive genealogy between two specs.

	hero supersede <old-slug> --by <new-slug> [--reason "..."]
		Sets superseded_by: <new-slug> on the old spec and adds the inverse
		supersedes: <old-slug> relation to the new spec, then reindexes so
		retrieval de-weights the old spec and context-injection annotates it.

	hero supersede --scan
		Walks .hero/specs and .hero/planning looking for likely
		supersede pairs (slug-suffix matches, body mentions). Writes a
		candidate report to .hero/reports/supersede-candidates.md and
		never mutates a spec.

	hero supersede --list
		Lists current supersede chains.

	hero supersede --unset <old-slug>
		Clears superseded_by on a spec (rare — only when a chain was set in error).

Refuses to:
- supersede with a target that's itself superseded (would create a chain into an archive)
- create cycles (A supersedes B, B supersedes A)`,
	RunE: runSupersede,
}

var (
	supersedeBy     string
	supersedeReason string
	supersedeScan   bool
	supersedeList   bool
	supersedeUnset  bool
)

func init() {
	supersedeCmd.Flags().StringVar(&supersedeBy, "by", "", "slug of the spec that replaces the old one")
	supersedeCmd.Flags().StringVar(&supersedeReason, "reason", "", "optional rationale recorded in frontmatter as a comment")
	supersedeCmd.Flags().BoolVar(&supersedeScan, "scan", false, "detect candidate supersede pairs and write a report (no mutation)")
	supersedeCmd.Flags().BoolVar(&supersedeList, "list", false, "list current supersede chains")
	supersedeCmd.Flags().BoolVar(&supersedeUnset, "unset", false, "clear superseded_by on the given spec")
}

func runSupersede(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Mode dispatch — mutually exclusive flags.
	switch {
	case supersedeScan:
		return runSupersedeScan(heroDir)
	case supersedeList:
		return runSupersedeList(heroDir)
	case supersedeUnset:
		if len(args) != 1 {
			return fmt.Errorf("--unset requires exactly one spec slug")
		}
		return runSupersedeUnset(heroDir, args[0])
	}

	// The default mode has two independent local selectors. Existing supplied
	// values are never replaced; only an omitted old slug or empty --by is
	// eligible for a terminal picker.
	oldSupplied := len(args) == 1
	if len(args) > 1 {
		return errSupersedeMissingOld
	}
	oldSlug := ""
	if oldSupplied {
		oldSlug = args[0]
	}
	by := supersedeBy
	if !oldSupplied || by == "" {
		if !prompt.IsInputTTY(cmd.InOrStdin()) {
			if !oldSupplied {
				return errSupersedeMissingOld
			}
			return errSupersedeMissingBy
		}
		specs, err := spec.Discover(heroDir)
		if err != nil {
			return fmt.Errorf("discovering specs: %w", err)
		}
		if !oldSupplied {
			oldSlug, err = pickFromCorpus(cmd, "Spec to supersede", specSlugCandidates(specs), errSupersedeMissingOld)
			if err != nil {
				return err
			}
		}
		if by == "" {
			by, err = pickFromCorpus(cmd, "Replaced by", withoutSlug(specSlugCandidates(specs), oldSlug), errSupersedeMissingBy)
			if err != nil {
				return err
			}
		}
	}
	return runSupersedeSet(heroDir, oldSlug, by, supersedeReason)
}

var (
	errSupersedeMissingOld = errors.New("supply the old spec slug (the one being replaced)")
	errSupersedeMissingBy  = errors.New("--by <new-slug> is required (the replacement spec)")
)

// runSupersedeSet wires old.superseded_by = new and new.supersedes:
// <old-slug>, then reindexes. Refuses cycles and chain-into-archive.
func runSupersedeSet(heroDir, oldSlug, newSlug, reason string) error {
	if oldSlug == newSlug {
		return fmt.Errorf("a spec cannot supersede itself")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	bySlug := indexSpecsBySlug(specs)

	oldSpec, ok := bySlug[oldSlug]
	if !ok {
		return fmt.Errorf("spec %q not found", oldSlug)
	}
	newSpec, ok := bySlug[newSlug]
	if !ok {
		return fmt.Errorf("replacement spec %q not found", newSlug)
	}

	// Refuse if the replacement is itself superseded — would create a
	// chain pointing into an archive. Suggest the chain target so the
	// user can re-run with the right --by.
	if newSpec.SupersededBy != "" {
		return fmt.Errorf("replacement %q is itself superseded by %q; did you mean --by %s?",
			newSlug, newSpec.SupersededBy, newSpec.SupersededBy)
	}

	// Cycle check: walk superseded_by from `new` forward and abort if
	// `old` shows up in the chain. With the chain-into-archive guard
	// above, this can only fire when there's already a multi-hop chain;
	// keep it defensive anyway.
	if cycleSlug := findSupersedeCycle(bySlug, newSlug, oldSlug); cycleSlug != "" {
		return fmt.Errorf("setting %q.superseded_by = %q would create a cycle through %q",
			oldSlug, newSlug, cycleSlug)
	}

	// Update the old spec's frontmatter.
	oldContent, err := os.ReadFile(oldSpec.Path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", oldSpec.Path, err)
	}
	updated := spec.SetFrontmatterField(string(oldContent), "superseded_by", newSlug)
	if reason != "" {
		updated = spec.SetFrontmatterField(updated, "# superseded_reason", reason)
	}
	if err := os.WriteFile(oldSpec.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", oldSpec.Path, err)
	}

	// Append a `supersedes: <old-slug>` relation to the new spec.
	if err := appendSupersedesRelation(newSpec.Path, oldSlug); err != nil {
		return fmt.Errorf("updating %s: %w", newSpec.Path, err)
	}

	// Reindex so retrieval picks up the new field.
	stats, err := index.Rebuild(heroDir)
	if err != nil {
		return fmt.Errorf("rebuilding index: %w", err)
	}

	fmt.Printf("Marked %s superseded by %s.\n", oldSlug, newSlug)
	fmt.Printf("  - Updated %s\n", oldSpec.Path)
	fmt.Printf("  - Added supersedes relation to %s\n", newSpec.Path)
	fmt.Printf("  - Reindexed (%d specs)\n", stats.TotalSpecs)
	fmt.Println()
	fmt.Println("Search will now de-weight the old spec (0.3× score multiplier) and context-injection will annotate it with a redirect marker.")
	return nil
}

// runSupersedeUnset clears the superseded_by field on a spec.
func runSupersedeUnset(heroDir, slug string) error {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	bySlug := indexSpecsBySlug(specs)
	s, ok := bySlug[slug]
	if !ok {
		return fmt.Errorf("spec %q not found", slug)
	}
	if s.SupersededBy == "" {
		fmt.Printf("Spec %q does not carry superseded_by — nothing to clear.\n", slug)
		return nil
	}

	content, err := os.ReadFile(s.Path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", s.Path, err)
	}
	// Set to empty string — SetFrontmatterField updates in place, which
	// leaves the explicit `superseded_by: ` line so the reader sees the
	// field exists with an empty value (the index treats it as unset).
	updated := spec.SetFrontmatterField(string(content), "superseded_by", "")
	if err := os.WriteFile(s.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", s.Path, err)
	}

	if _, err := index.Rebuild(heroDir); err != nil {
		return fmt.Errorf("rebuilding index: %w", err)
	}
	fmt.Printf("Cleared superseded_by on %s.\n", slug)
	return nil
}

// runSupersedeList prints all current supersede chains.
func runSupersedeList(heroDir string) error {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	var pairs [][2]string
	for _, s := range specs {
		if s.SupersededBy != "" {
			pairs = append(pairs, [2]string{s.Slug, s.SupersededBy})
		}
	}
	if len(pairs) == 0 {
		fmt.Println("No specs are currently marked superseded.")
		return nil
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i][0] < pairs[j][0] })
	fmt.Printf("Supersede chains (%d):\n", len(pairs))
	for _, p := range pairs {
		fmt.Printf("  %s → %s\n", p[0], p[1])
	}
	return nil
}

// runSupersedeScan walks the spec corpus, detects likely supersede
// pairs, and writes a candidate report. Never mutates a spec.
func runSupersedeScan(heroDir string) error {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	candidates := detectSupersedeCandidates(specs)

	reportDir := filepath.Join(heroDir, "reports")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return fmt.Errorf("creating reports dir: %w", err)
	}
	reportPath := filepath.Join(reportDir, "supersede-candidates.md")

	var b strings.Builder
	b.WriteString("# Supersede Candidate Report\n\n")
	b.WriteString("Pairs detected by `hero supersede --scan`. The scan never\n")
	b.WriteString("mutates a spec — confirm each pair manually with:\n\n")
	b.WriteString("    hero supersede <old> --by <new>\n\n")
	if len(candidates) == 0 {
		b.WriteString("No candidate pairs detected.\n")
	} else {
		fmt.Fprintf(&b, "Detected %d candidate pair(s):\n\n", len(candidates))
		for _, c := range candidates {
			fmt.Fprintf(&b, "- old: %-40s | new: %-40s | heuristic: %-12s | confidence: %s\n",
				c.OldSlug, c.NewSlug, c.Heuristic, c.Confidence)
		}
	}

	if err := os.WriteFile(reportPath, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	fmt.Printf("Wrote %d candidate(s) to %s\n", len(candidates), reportPath)
	return nil
}

// SupersedeCandidate names a detected pair from --scan. Confidence is
// "high" / "medium" / "low" depending on how strong the signal is.
type SupersedeCandidate struct {
	OldSlug    string
	NewSlug    string
	Heuristic  string
	Confidence string
}

var versionSuffixRe = regexp.MustCompile(`^(.+?)-v(\d+)$`)
var bodyMentionRe = regexp.MustCompile(`(?i)(?:replaces|supersedes|deprecates)\s+` + "`?" + `([a-z0-9][a-z0-9-]+)` + "`?")

// detectSupersedeCandidates runs the heuristics described in the spec
// and returns candidate pairs. Pure function — easy to test.
func detectSupersedeCandidates(specs []*spec.Spec) []SupersedeCandidate {
	bySlug := indexSpecsBySlug(specs)
	seen := make(map[string]bool) // de-dupe "old→new" pairs
	var out []SupersedeCandidate

	add := func(old, newer, h, conf string) {
		// Don't propose pairs where one or both are already superseded.
		if oldSpec, ok := bySlug[old]; ok && oldSpec.SupersededBy != "" {
			return
		}
		key := old + "→" + newer
		if seen[key] || old == newer {
			return
		}
		seen[key] = true
		out = append(out, SupersedeCandidate{OldSlug: old, NewSlug: newer, Heuristic: h, Confidence: conf})
	}

	// Heuristic 1: slug-suffix pairs (foo-v1 and foo-v2; foo and foo-v2).
	versioned := map[string][]struct {
		Slug    string
		Version int
	}{}
	for _, s := range specs {
		if m := versionSuffixRe.FindStringSubmatch(s.Slug); m != nil {
			base := m[1]
			ver := 0
			fmt.Sscanf(m[2], "%d", &ver)
			versioned[base] = append(versioned[base], struct {
				Slug    string
				Version int
			}{s.Slug, ver})
		}
	}
	for base, entries := range versioned {
		sort.Slice(entries, func(i, j int) bool { return entries[i].Version < entries[j].Version })
		// Pair every older with the highest.
		newest := entries[len(entries)-1].Slug
		for _, e := range entries[:len(entries)-1] {
			add(e.Slug, newest, "slug-suffix", "high")
		}
		// Also pair the un-versioned base slug if it exists.
		if _, ok := bySlug[base]; ok {
			add(base, newest, "slug-suffix", "high")
		}
	}

	// Heuristic 2: body mentions of "supersedes/replaces/deprecates <slug>".
	for _, s := range specs {
		body := s.RawContent
		matches := bodyMentionRe.FindAllStringSubmatch(body, -1)
		for _, m := range matches {
			target := strings.ToLower(m[1])
			if _, ok := bySlug[target]; ok {
				add(target, s.Slug, "body-mention", "medium")
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].OldSlug != out[j].OldSlug {
			return out[i].OldSlug < out[j].OldSlug
		}
		return out[i].NewSlug < out[j].NewSlug
	})
	return out
}

// indexSpecsBySlug builds a slug→Spec map.
func indexSpecsBySlug(specs []*spec.Spec) map[string]*spec.Spec {
	m := make(map[string]*spec.Spec, len(specs))
	for _, s := range specs {
		m[s.Slug] = s
	}
	return m
}

// findSupersedeCycle walks superseded_by from `start` for a bounded
// number of hops. If `target` appears in the chain, returns it.
// Returns "" when no cycle is found within the bound.
func findSupersedeCycle(bySlug map[string]*spec.Spec, start, target string) string {
	const maxHops = 16 // defensive bound — chains in practice should be 1-2 hops
	visited := make(map[string]bool)
	cur := start
	for hop := 0; hop < maxHops; hop++ {
		if cur == target {
			return cur
		}
		if visited[cur] {
			return cur // pre-existing cycle in data; surface it
		}
		visited[cur] = true
		s, ok := bySlug[cur]
		if !ok || s.SupersededBy == "" {
			return ""
		}
		cur = s.SupersededBy
	}
	return cur // hit the depth bound — treat as cycle
}

// appendSupersedesRelation adds a `supersedes: <old-slug>` line to the
// new spec's frontmatter. If the spec already lists `supersedes:` with
// a value, append the old-slug to the list. The simpler scalar form
// `supersedes: foo` is replaced with a list when a second slug arrives.
func appendSupersedesRelation(specPath, oldSlug string) error {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return err
	}
	content := string(data)

	// Re-parse so we know if supersedes already lists this slug — avoid
	// duplicates if the user re-runs the command.
	parsed, err := spec.Parse(content, specPath, time.Now())
	if err != nil {
		return err
	}
	for _, rel := range parsed.Relations {
		if rel.Kind == "supersedes" && rel.Target == oldSlug {
			return nil // already recorded — idempotent
		}
	}

	// Collect existing supersedes targets, append the new one, write a
	// single bracket-list form. SetFrontmatterField updates in place
	// when the key exists.
	var targets []string
	for _, rel := range parsed.Relations {
		if rel.Kind == "supersedes" {
			targets = append(targets, rel.Target)
		}
	}
	targets = append(targets, oldSlug)

	listVal := "[" + strings.Join(targets, ", ") + "]"
	updated := spec.SetFrontmatterField(content, "supersedes", listVal)
	return os.WriteFile(specPath, []byte(updated), 0o644)
}
