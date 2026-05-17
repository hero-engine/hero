package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/digest"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/traversal"
	"github.com/spf13/cobra"
)

// hero resume — the per-turn warm-up. Loads the focused, ranked
// subgraph relevant to the current developer + repo + working tree
// so the model starts every session in a hot state. Auto-fired by
// commands/resume.md natural-language triggers; users rarely type
// the command name directly.
//
// The previous `hero brief` is now collapsed into this — same
// digest, same flags, but named after the user intent ("resume my
// work"). `hero load` is registered as an alias for power users
// who want the action verb.

var (
	resumeBudget int
	resumeJSON   bool
	resumeEmail  string
	resumeFocus  []string
	resumeAuto   bool
)

var resumeCmd = &cobra.Command{
	Use:     "resume",
	Aliases: []string{"load"},
	Short:   "Warm up the session — load relevant context from the knowledge graph",
	Long: `Generates a focused, ranked, budget-bounded brief of the unified
knowledge graph state and prints it for the model to consume. Run
at the start of every session in a hero-aware repo so the model
lands with the most-relevant project state already in context —
who you are, what's in flight, what just changed, dead ends to
skip, blockers, files-nearby context.

The digester ranks candidates by recency × focus-match × signal
weight, then prunes to fit the token budget — the brief stays a
constant size as the corpus grows. As history accumulates, the
brief gets denser, not bigger.

Aliases: ` + "`hero load`" + ` for the action-verb form (same behavior).`,
	RunE: runResume,
}

func init() {
	resumeCmd.Flags().IntVar(&resumeBudget, "budget", 3000, "approximate token budget for the brief")
	resumeCmd.Flags().BoolVar(&resumeJSON, "json", false, "emit JSON instead of markdown")
	resumeCmd.Flags().StringVar(&resumeEmail, "email", "", "author email (defaults to git config user.email)")
	resumeCmd.Flags().StringSliceVar(&resumeFocus, "focus", nil, "files to bias scoring toward (default: changed files in working tree)")
	resumeCmd.Flags().BoolVar(&resumeAuto, "auto", false, "auto-detect focus from current branch + uncommitted changes")
}

func runResume(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	email := resumeEmail
	if email == "" {
		email = gitConfigEmail(projectRoot)
	}

	focus := resumeFocus
	if len(focus) == 0 && (resumeAuto || isInteractive()) {
		focus = autoFocus(projectRoot)
	}

	opts := digest.Options{
		RepoKey:     gitutil.RepoKey(projectRoot),
		Branch:      gitutil.CurrentBranch(projectRoot),
		AuthorEmail: email,
		FocusFiles:  focus,
		TokenBudget: resumeBudget,
		SessionID:   readSessionFromExistingNext(heroDir),
	}

	brief, err := digest.Generate(store, opts)
	if err != nil {
		return fmt.Errorf("generating brief: %w", err)
	}

	if resumeJSON {
		b, err := brief.JSON()
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	fmt.Print(brief.Markdown())

	// Phase 3 cross-repo-peering: passive surface peer-owned contract
	// imports in the changed-files set. Best-effort — never fail
	// `hero resume` on a peering miss. Always uses the working-tree
	// dirty set so the signal fires whether or not --focus / --auto
	// was supplied (those flags bias the brief; the signal is
	// independent).
	scanFiles := focus
	if len(scanFiles) == 0 {
		scanFiles = autoFocus(projectRoot)
	}
	if signal := contractImportSignalForFocus(projectRoot, scanFiles); signal != "" {
		fmt.Println()
		fmt.Print(signal)
	}
	return nil
}

// contractImportSignalForFocus runs the contract-import scanner over
// the given focus files and returns the rendered passive signal,
// or "" if nothing to surface. Errors are swallowed by design: the
// signal is supplementary context, never a blocker.
func contractImportSignalForFocus(projectRoot string, files []string) string {
	if len(files) == 0 {
		return ""
	}
	hits, err := peering.ScanContractImports(projectRoot, peering.ScanOptions{
		ChangedFiles: files,
	})
	if err != nil {
		return ""
	}
	return peering.RenderContractImportSignal(hits)
}

// --- graph search (powers the default `hero search`) ---------------------
//
// runGraphSearch is invoked from search.go when the user runs
// `hero search <query>` with no filters and no --specs flag. It walks
// the unified graph (Features, Decisions, Notes, Attempts, Commits,
// Symbols, etc.) — strictly more capable than the legacy FTS5
// spec-only search.

// graphSearchCandidates returns the count of graph nodes that match
// the search query, without rendering output. Used by runSearch to
// decide whether to route to graph search or fall back to FTS5.
func graphSearchCandidates(args []string) (int, error) {
	topic := strings.Join(args, " ")
	store, _, err := openRepoStore()
	if err != nil {
		return 0, err
	}
	defer store.Close()
	q := strings.ToLower(topic)
	var n int
	err = store.DB().QueryRow(
		`SELECT COUNT(*) FROM nodes
		  WHERE valid_to IS NULL
		    AND (lower(key) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.title'), '')) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.body'), '')) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.subject'), '')) LIKE '%' || ? || '%')`,
		q, q, q, q,
	).Scan(&n)
	return n, err
}

// nodeTypeWeight gives higher scores to high-signal node types so a
// fresh git import doesn't drown spec / knowledge / code results in a
// search. Tuned by hand from observing real corpora; revisit if a node
// type starts producing noisy hits.
func nodeTypeWeight(t string) float64 {
	switch t {
	case "Feature", "Bug", "Initiative", "Decision":
		return 10
	case "Convention", "Rule":
		return 9
	case "ContextDoc", "Note":
		return 8
	case "Symbol", "Package":
		return 6
	case "File":
		return 5
	case "Issue":
		return 3
	case "Commit", "Person":
		return 1
	default:
		return 4
	}
}

func runGraphSearch(args []string, budget int, asJSON bool) error {
	topic := strings.Join(args, " ")
	store, repoKey, err := openRepoStore()
	if err != nil {
		return err
	}
	defer store.Close()

	q := strings.ToLower(topic)
	type cand struct {
		nodeType, key, title, body, repo string
		score                            float64
	}
	rows, err := store.DB().Query(
		`SELECT type, key,
		        COALESCE(json_extract(props, '$.title'), '') AS title,
		        COALESCE(json_extract(props, '$.body'), '')  AS body,
		        COALESCE(json_extract(props, '$.subject'), '') AS subject,
		        COALESCE(repo, '') AS repo,
		        ingested_at
		   FROM nodes
		  WHERE valid_to IS NULL
		    AND (lower(key) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.title'), '')) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.body'), '')) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.subject'), '')) LIKE '%' || ? || '%')
		  ORDER BY ingested_at DESC
		  LIMIT 200`,
		q, q, q, q,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var cands []cand
	for rows.Next() {
		var c cand
		var subject, ingested string
		if err := rows.Scan(&c.nodeType, &c.key, &c.title, &c.body, &subject, &c.repo, &ingested); err != nil {
			return err
		}
		if c.body == "" && subject != "" {
			c.body = subject
		}
		c.score = nodeTypeWeight(c.nodeType)
		// Title/key match boosts score above body-only matches.
		lowKey := strings.ToLower(c.key)
		lowTitle := strings.ToLower(c.title)
		if strings.Contains(lowKey, q) || strings.Contains(lowTitle, q) {
			c.score += 2
		}
		cands = append(cands, c)
	}

	// High-signal types (specs, knowledge, code) beat commits/issues/people.
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].score > cands[j].score
	})
	if len(cands) > 30 {
		cands = cands[:30]
	}

	if asJSON {
		out, _ := json.MarshalIndent(cands, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if len(cands) == 0 {
		fmt.Printf("Nothing matching %q in graph for repo %s.\n", topic, repoKey)
		return nil
	}

	fmt.Printf("<!-- hero search: %q in %s + siblings, %d matches, budget=%d -->\n\n", topic, repoKey, len(cands), budget)
	used := 0
	dropped := 0
	for _, c := range cands {
		title := c.title
		if title == "" {
			title = c.key
		}
		line := fmt.Sprintf("- **%s** _(%s, `%s`)_", oneLineString(title), c.nodeType, c.key)
		// Show repo origin when it's a different repo than the local one
		// (federation pull or sibling-repo scan).
		if c.repo != "" && c.repo != repoKey {
			line += fmt.Sprintf(" _[%s]_", c.repo)
		}
		if c.body != "" {
			line += " — " + oneLineString(truncStr(c.body, 160))
		}
		tok := (len(line) + 3) / 4
		if used+tok > budget {
			dropped++
			continue
		}
		fmt.Println(line)
		used += tok
	}
	if dropped > 0 {
		fmt.Printf("\n_…+%d more — refine query or run `hero search` with `--specs` for FTS5 spec-only results_\n", dropped)
	}
	return nil
}

// --- code-graph impact section (called from hero impact) ----------------
//
// runCodeGraphImpact is invoked from impact.go after the spec-impact
// section, so `hero impact <thing>` returns both the spec hits ("which
// specs reference this") and the code-graph hits ("who imports /
// touched recently") in one report.

func runCodeGraphImpact(args []string) error {
	target := args[0]
	store, repoKey, err := openRepoStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Resolve target: try Symbol → File → Package by suffix match
	type hit struct {
		nodeType, key string
		id            int64
	}
	var hits []hit
	rows, err := store.DB().Query(
		`SELECT type, key, id FROM nodes
		  WHERE valid_to IS NULL AND repo = ?
		    AND type IN ('Symbol','File','Package')
		    AND (key = ? OR key LIKE '%' || ? OR key LIKE '%' || ? || '%')
		  LIMIT 10`,
		repoKey, target, ":"+target, target,
	)
	if err != nil {
		return err
	}
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.nodeType, &h.key, &h.id); err != nil {
			rows.Close()
			return err
		}
		hits = append(hits, h)
	}
	rows.Close()

	if len(hits) == 0 {
		fmt.Printf("No matches for %q in repo %s.\n", target, repoKey)
		return nil
	}

	for _, h := range hits {
		fmt.Printf("\n## %s `%s`\n\n", h.nodeType, h.key)

		switch h.nodeType {
		case "Package":
			// Importers
			ir, _ := store.DB().Query(
				`SELECT n.key FROM edges e
				   JOIN nodes n ON n.id = e.from_id AND n.valid_to IS NULL
				  WHERE e.to_id = ? AND e.type = 'imports' AND e.valid_to IS NULL
				  ORDER BY n.key LIMIT 30`, h.id)
			fmt.Println("**Importers:**")
			any := false
			for ir != nil && ir.Next() {
				var k string
				ir.Scan(&k)
				fmt.Printf("- `%s`\n", k)
				any = true
			}
			if ir != nil {
				ir.Close()
			}
			if !any {
				fmt.Println("_(none in this repo)_")
			}
		case "File":
			// Recent commits that touched it
			cr, _ := store.DB().Query(
				`SELECT json_extract(c.props, '$.sha'),
				        json_extract(c.props, '$.subject'),
				        json_extract(c.props, '$.date'),
				        json_extract(c.props, '$.author_name')
				   FROM nodes c
				   JOIN edges e ON e.from_id = c.id AND e.type = 'touches' AND e.valid_to IS NULL
				  WHERE e.to_id = ? AND c.valid_to IS NULL
				  ORDER BY json_extract(c.props, '$.date') DESC LIMIT 10`, h.id)
			fmt.Println("**Recent commits:**")
			for cr != nil && cr.Next() {
				var sha, subj, date, author string
				cr.Scan(&sha, &subj, &date, &author)
				fmt.Printf("- `%s` %s — %s _(%s)_\n", shortShaImpact(sha), oneLineString(subj), date, author)
			}
			if cr != nil {
				cr.Close()
			}
		case "Symbol":
			// File that defines it
			fr, _ := store.DB().Query(
				`SELECT n.key FROM edges e
				   JOIN nodes n ON n.id = e.from_id AND n.valid_to IS NULL
				  WHERE e.to_id = ? AND e.type = 'defines' AND e.valid_to IS NULL`, h.id)
			for fr != nil && fr.Next() {
				var k string
				fr.Scan(&k)
				fmt.Printf("Defined in: `%s`\n", k)
			}
			if fr != nil {
				fr.Close()
			}
		}
	}
	return nil
}

// --- why -----------------------------------------------------------------

var (
	whyDepth      int
	whyEdges      bool
	whySubproject string
)

var whyCmd = &cobra.Command{
	Use:   "why <slug-or-id>",
	Short: "Trace backwards: where did this come from?",
	Long: `Resolves the target to a graph node and walks origin edges in
reverse — multi-hop, depth-bounded, oldest-on-top — so you see the
chain of decisions, specs, and commits that led to the target's
existence.

Origin edge types walked: belongs_to, satisfied_by, attempted_in,
decided_in, supersedes, mentions, depends_on, derived_from,
originated_in, closes, fixes.

Default depth is 4 (matches federation depth-cap convention). Use
--depth to extend. Use --edges for the legacy single-hop view that
shows raw outgoing + incoming relationships at the target node.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runWhy,
}

func init() {
	whyCmd.Flags().IntVar(&whyDepth, "depth", traversal.DefaultDepth, "max recursion depth (origin chain length)")
	whyCmd.Flags().BoolVar(&whyEdges, "edges", false, "fall back to single-hop edge listing (no recursion)")
	whyCmd.Flags().StringVar(&whySubproject, "subproject", "", "highlight in-scope hops; 'all' disables. Default: active scope from cwd")
}

func runWhy(cmd *cobra.Command, args []string) error {
	target := strings.Join(args, " ")
	store, repoKey, err := openRepoStore()
	if err != nil {
		return err
	}
	defer store.Close()

	if whyEdges {
		return runWhyEdges(store, repoKey, target)
	}

	trace, err := traversal.Why(store, repoKey, target, whyDepth)
	if err != nil {
		return err
	}
	activeScope := resolveSubprojectFilter(whySubproject)
	if activeScope == "all" {
		activeScope = ""
	}
	fmt.Print(trace.MarkdownScoped(activeScope))
	return nil
}

// runWhyEdges is the legacy single-hop view, surfaced behind --edges
// for users who want a flat dump of every adjacent edge rather than
// the recursive origin trace.
func runWhyEdges(store *graph.Store, repoKey, target string) error {
	var (
		nodeType string
		nodeID   int64
		title    string
	)
	err := store.DB().QueryRow(
		`SELECT type, id, COALESCE(json_extract(props, '$.title'), key)
		   FROM nodes WHERE key = ? AND repo = ? AND valid_to IS NULL LIMIT 1`,
		target, repoKey,
	).Scan(&nodeType, &nodeID, &title)
	if err != nil {
		return fmt.Errorf("no node with key %q in repo %s", target, repoKey)
	}

	fmt.Printf("# %s `%s` (%s)\n\n", title, target, nodeType)

	// Outgoing edges — what this depends on / supersedes / mentions
	out, _ := store.DB().Query(
		`SELECT e.type, n.type, n.key, COALESCE(json_extract(n.props, '$.title'), n.key)
		   FROM edges e
		   JOIN nodes n ON n.id = e.to_id AND n.valid_to IS NULL
		  WHERE e.from_id = ? AND e.valid_to IS NULL
		  ORDER BY e.type, n.key`, nodeID)
	if out != nil {
		fmt.Println("## Outgoing relationships")
		any := false
		for out.Next() {
			var etype, ntype, nkey, ntitle string
			out.Scan(&etype, &ntype, &nkey, &ntitle)
			fmt.Printf("- _%s_ → %s `%s` (%s)\n", etype, ntitle, nkey, ntype)
			any = true
		}
		out.Close()
		if !any {
			fmt.Println("_(none)_")
		}
		fmt.Println()
	}

	// Incoming edges — what depends on this / mentions this
	in, _ := store.DB().Query(
		`SELECT e.type, n.type, n.key, COALESCE(json_extract(n.props, '$.title'), n.key)
		   FROM edges e
		   JOIN nodes n ON n.id = e.from_id AND n.valid_to IS NULL
		  WHERE e.to_id = ? AND e.valid_to IS NULL
		  ORDER BY e.type, n.key`, nodeID)
	if in != nil {
		fmt.Println("## Incoming relationships")
		any := false
		for in.Next() {
			var etype, ntype, nkey, ntitle string
			in.Scan(&etype, &ntype, &nkey, &ntitle)
			fmt.Printf("- %s `%s` (%s) → _%s_\n", ntitle, nkey, ntype, etype)
			any = true
		}
		in.Close()
		if !any {
			fmt.Println("_(none)_")
		}
	}
	return nil
}

// --- blocked -------------------------------------------------------------

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "Show every open Feature that's waiting on something incomplete",
	RunE:  runBlocked,
}

func runBlocked(cmd *cobra.Command, args []string) error {
	store, repoKey, err := openRepoStore()
	if err != nil {
		return err
	}
	defer store.Close()
	vocab := activeVocab(loadConfigSilent())

	rows, err := store.DB().Query(
		`SELECT f.key,
		        COALESCE(json_extract(f.props, '$.title'), f.key),
		        b.type, b.key,
		        COALESCE(json_extract(b.props, '$.status'), '')
		   FROM nodes f
		   JOIN edges e ON e.from_id = f.id AND e.type IN ('depends_on','blocks') AND e.valid_to IS NULL
		   JOIN nodes b ON e.to_id = b.id AND b.valid_to IS NULL
		  WHERE f.type = 'Feature' AND f.repo = ? AND f.valid_to IS NULL
		    AND COALESCE(json_extract(f.props, '$.status'), '') NOT IN ('completed','superseded')
		    AND COALESCE(json_extract(b.props, '$.status'), '') NOT IN ('completed','accepted')
		  ORDER BY f.key`, repoKey)
	if err != nil {
		return err
	}
	defer rows.Close()

	type chain struct{ ftitle, fkey, btype, bkey, bstatus string }
	byFeature := map[string][]chain{}
	for rows.Next() {
		var c chain
		if err := rows.Scan(&c.fkey, &c.ftitle, &c.btype, &c.bkey, &c.bstatus); err != nil {
			return err
		}
		byFeature[c.fkey] = append(byFeature[c.fkey], c)
	}
	failingByParent, err := acFailuresByParent(store, repoKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: AC join failed: %v\n", err)
	}

	if len(byFeature) == 0 && len(failingByParent) == 0 {
		fmt.Println("Nothing blocked.")
		return nil
	}

	keys := make([]string, 0, len(byFeature))
	for k := range byFeature {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cs := byFeature[k]
		fmt.Printf("- **%s** (`%s`)\n", cs[0].ftitle, k)
		for _, c := range cs {
			status := c.bstatus
			if status == "" {
				status = "?"
			}
			fmt.Printf("    waiting on %s `%s` (%s)\n", displayType(vocab, strings.ToLower(c.btype)), c.bkey, status)
		}
		// Add failing-AC chains for this feature, if any.
		for _, ac := range failingByParent[k] {
			fmt.Printf("    failing AC `%s` (%s)\n", ac.key, ac.status)
		}
		delete(failingByParent, k)
	}

	// Surface specs that aren't otherwise blocked but have failing
	// ACs — those are blocked by their own contract.
	if len(failingByParent) > 0 {
		extraKeys := make([]string, 0, len(failingByParent))
		for k := range failingByParent {
			extraKeys = append(extraKeys, k)
		}
		sort.Strings(extraKeys)
		for _, k := range extraKeys {
			fmt.Printf("- **%s** — has failing acceptance criteria\n", k)
			for _, ac := range failingByParent[k] {
				fmt.Printf("    failing AC `%s` (%s)\n", ac.key, ac.status)
			}
		}
	}
	return nil
}

type acFailure struct {
	key, status string
}

// acFailuresByParent returns failing/regressed Criterion nodes grouped
// by their parent spec slug for the given repo. Empty map when no
// failures exist (or no Criterion nodes at all).
func acFailuresByParent(store *graph.Store, repoKey string) (map[string][]acFailure, error) {
	rows, err := store.DB().Query(
		`SELECT key,
		        COALESCE(json_extract(props, '$.status'), '') AS status,
		        COALESCE(json_extract(props, '$.parent'), '') AS parent
		   FROM nodes
		  WHERE type = 'Criterion' AND repo = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.status'), '') IN ('failing','regressed')
		  ORDER BY parent, key`,
		repoKey,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]acFailure{}
	for rows.Next() {
		var key, status, parent string
		if err := rows.Scan(&key, &status, &parent); err != nil {
			return nil, err
		}
		if parent == "" {
			continue
		}
		out[parent] = append(out[parent], acFailure{key: key, status: status})
	}
	return out, nil
}

// --- shared helpers ------------------------------------------------------

func openRepoStore() (*graph.Store, string, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, "", fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}
	store, err := graph.Open(heroDir)
	if err != nil {
		return nil, "", err
	}
	return store, gitutil.RepoKey(projectRoot), nil
}

func gitConfigEmail(repoDir string) string {
	out, err := exec.Command("git", "-C", repoDir, "config", "user.email").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// autoFocus returns files that are dirty in the working tree — those
// are what the dev is currently looking at and should bias the brief.
func autoFocus(repoDir string) []string {
	out, err := exec.Command("git", "-C", repoDir, "status", "--porcelain").Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip the 2-char status prefix
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files
}

func isInteractive() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func oneLineString(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return s
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func shortShaImpact(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
