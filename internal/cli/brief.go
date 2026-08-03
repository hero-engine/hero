package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/digest"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/spec"
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

	// Cold-graph nudge (resume-brief-missing-project-context, Change 2b):
	// SessionStart ingest auto-rebuilds project context on a fresh clone
	// (Change 2a), but if that rebuild was skipped or no-op'd — e.g.
	// resume is the first command run, before any ingest — the brief's
	// project-context sections would render empty with no explanation.
	// Detect the cold graph here and name the one command that restores
	// context, so the user never lands on a silent empty map. Best-effort
	// and never fatal; emitted to stderr so it doesn't pollute the brief
	// the model consumes on stdout.
	if projectGraphCold(store, gitutil.RepoKey(projectRoot)) {
		fmt.Fprintln(os.Stderr,
			"hint: project context is empty for this clone — run `hero scan` to populate it (commits, specs, blockers)")
	}

	email := resumeEmail
	if email == "" {
		email = gitConfigEmail(projectRoot)
	}

	focus := resumeFocus
	if len(focus) == 0 && (resumeAuto || isInteractive(cmd.OutOrStdout())) {
		focus = autoFocus(projectRoot)
	}

	opts := digest.Options{
		RepoKey:     gitutil.RepoKey(projectRoot),
		Branch:      gitutil.CurrentBranch(projectRoot),
		AuthorEmail: email,
		User:        nextUserSlug(cfg),
		Domain:      graph.DomainFor(cfg, graph.IntrinsicActive),
		FocusFiles:  focus,
		TokenBudget: resumeBudget,
		SessionID:   readSessionFromExistingNext(heroDir),
	}

	// Cross-machine identity reconciliation: the local slug
	// (nextUserSlug) is derived from volatile git/$USER config and can
	// differ from the slug baked into a traveled .hero/next/<user>.md.
	// When the handoff query misses under the local slug, resolve the
	// identity from the files on disk; when it can't be resolved but
	// handoff files DO exist under other slugs, emit a diagnostic rather
	// than a silent empty section. Never fails resume.
	resolveHandoffIdentity(store, heroDir, &opts, os.Stderr)

	brief, err := digest.Generate(store, opts)
	if err != nil {
		return fmt.Errorf("generating brief: %w", err)
	}

	if resumeJSON {
		b, err := brief.JSON()
		if err != nil {
			return err
		}
		var payload map[string]any
		if err := json.Unmarshal(b, &payload); err != nil {
			return err
		}
		if summary, summaryErr := projectMailSummary(); summaryErr == nil {
			payload["mail"] = summary
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}
	fmt.Print(brief.Markdown())
	printProjectMailSummary()

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

// resolveHandoffIdentity reconciles the locally-derived handoff slug
// (opts.User) against the slugs actually present in the traveled
// .hero/next/<user>.md files. It mutates opts.User in place when a
// single fallback identity resolves, and writes a diagnostic to warn
// when handoff content exists under slugs that don't match the local
// identity and can't be auto-resolved.
//
// Best-effort by contract: any error path leaves opts.User untouched
// and emits nothing. Resume must never fail on a handoff miss.
//
// Resolution rules:
//   - If the local slug already has handoff content → no-op (the common
//     same-machine case; fresh repos with no files also fall through
//     silently).
//   - Else scan .hero/next/*.md (excluding *.local.md) for distinct
//     frontmatter `user:` values that carry handoff content in the graph.
//   - Exactly one such slug (≠ local) → adopt it as opts.User.
//   - More than one, or one that still doesn't resolve → emit a
//     diagnostic naming the queried slug vs. the available slugs.
func resolveHandoffIdentity(store *graph.Store, heroDir string, opts *digest.Options, warn io.Writer) {
	if opts == nil || store == nil {
		return
	}
	// If the local slug already resolves, nothing to do.
	if handoffHasContent(store, opts.User, opts.RepoKey, opts.Domain) {
		return
	}

	fileUsers := nextFileUserSlugs(heroDir)
	if len(fileUsers) == 0 {
		return // no traveled files → genuinely-fresh repo, stay silent
	}

	// Of the slugs recorded in the files, which actually carry handoff
	// content in the graph (i.e. ingest has run for them)?
	var resolved []string
	for _, u := range fileUsers {
		if u == opts.User {
			continue
		}
		if handoffHasContent(store, u, opts.RepoKey, opts.Domain) {
			resolved = append(resolved, u)
		}
	}

	switch len(resolved) {
	case 1:
		// Single unambiguous fallback identity — adopt it.
		opts.User = resolved[0]
	default:
		// Either zero slugs resolve (files present but not ingested, or
		// keyed differently) or several do (ambiguous). In both cases the
		// local read found nothing under its own slug while handoff files
		// exist on disk — surface that observably instead of an empty
		// section. The hint names the queried slug vs. what's available.
		if warn != nil && len(fileUsers) > 0 {
			fmt.Fprintf(warn,
				"warning: handoff present for %v but local identity is %q — set tracking.defaultAgent or git user.name to match, or run `hero next ingest`\n",
				fileUsers, opts.User)
		}
	}
}

// handoffHasContent reports whether any handoff singleton (ask /
// suggestion / reflection) exists for the (user, repo, domain) triple.
// Errors are treated as "no content" — this is a best-effort probe.
func handoffHasContent(store *graph.Store, user, repoKey, domain string) bool {
	if user == "" {
		return false
	}
	if ask, _ := handoff.LatestAsk(store, user, repoKey, domain); ask != nil && ask.Text != "" {
		return true
	}
	if sug, _ := handoff.LatestSuggestion(store, user, repoKey, domain); sug != nil && sug.Text != "" {
		return true
	}
	if refs, _ := handoff.RecentReflections(store, user, repoKey, domain, 1); len(refs) > 0 {
		return true
	}
	return false
}

// nextFileUserSlugs returns the distinct frontmatter `user:` values
// across the travel-eligible .hero/next/*.md files (excluding
// *.local.md), sorted for deterministic diagnostics. Files that don't
// parse or carry no `user:` are skipped.
func nextFileUserSlugs(heroDir string) []string {
	entries, err := os.ReadDir(filepath.Join(heroDir, nextDirName))
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".local.md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(heroDir, nextDirName, name))
		if err != nil {
			continue
		}
		parsed, err := handoff.ParseUserHandoff(data)
		if err != nil || parsed.User == "" {
			continue
		}
		if _, ok := seen[parsed.User]; ok {
			continue
		}
		seen[parsed.User] = struct{}{}
		out = append(out, parsed.User)
	}
	sort.Strings(out)
	return out
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

	// Reconcile the spec subgraph from disk before resolving, so a spec
	// created since the last graph ingest (e.g. a peer-received spec-out
	// spec) resolves without a prior manual `hero scan`. This is the
	// graph-side analogue of the index's RefreshIfStale and mirrors what
	// `hero blocked` already does. Covers both the recursive trace below
	// and the --edges path (runWhyEdges is only reached from here).
	reconcileSpecGraph(store, findProjectRoot(), repoKey, loadConfigSilent())

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

var (
	blockedAllDomains bool
	blockedDomain     string
)

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "Show every open Feature that's waiting on something incomplete",
	Long: `Filters by the active domain by default — a PM workspace
sees PM blockers, an engineering workspace sees engineering ones.
Pass --all-domains to surface cross-domain blockers (e.g. an
engineering Feature waiting on a PM PRD) tagged with their domain.`,
	RunE: runBlocked,
}

func init() {
	blockedCmd.Flags().BoolVar(&blockedAllDomains, "all-domains", false, "include blockers from any domain; cross-domain entries are tagged inline")
	blockedCmd.Flags().StringVar(&blockedDomain, "domain", "", "override the active domain filter (\"*\" = all)")
}

func runBlocked(cmd *cobra.Command, args []string) error {
	store, repoKey, err := openRepoStore()
	if err != nil {
		return err
	}
	defer store.Close()
	cfg := loadConfigSilent()
	vocab := activeVocab(cfg)
	override := blockedDomain
	if blockedAllDomains {
		override = "*"
	}
	var cfgVal config.Config
	if cfg != nil {
		cfgVal = *cfg
	}
	scope := graph.ResolveDomain(cfgVal, override)

	// Reconcile the spec subgraph from frontmatter (the durable source of
	// truth) before querying, so `hero blocked` reflects current relations
	// even on a fresh clone where graph.db — a regenerable cache — hasn't
	// been reingested yet. This keeps `hero blocked` in agreement with
	// `hero queue`, which reads frontmatter directly. Shared with `hero why`
	// so the two read paths stay in lockstep. Best-effort: on any error we
	// just query whatever the graph already holds.
	reconcileSpecGraph(store, findProjectRoot(), repoKey, cfg)

	// f.domain = scope is the active filter on the Feature row; we
	// always JOIN both endpoints so cross-domain rows can be rendered
	// with a [domain: …] tag in --all-domains mode.
	q := `SELECT f.key,
	             COALESCE(json_extract(f.props, '$.title'), f.key),
	             b.type, b.key,
	             COALESCE(json_extract(b.props, '$.status'), ''),
	             f.domain, b.domain
	        FROM nodes f
	        JOIN edges e ON e.from_id = f.id AND e.type IN ('depends_on','blocks') AND e.valid_to IS NULL
	        JOIN nodes b ON e.to_id = b.id AND b.valid_to IS NULL
	       WHERE f.type = 'Feature' AND f.repo = ? AND f.valid_to IS NULL
	         AND COALESCE(json_extract(f.props, '$.status'), '') NOT IN ('completed','superseded')
	         AND COALESCE(json_extract(b.props, '$.status'), '') NOT IN ('completed','accepted')`
	args2 := []any{repoKey}
	if frag, fragArgs := scope.Where("f"); frag != "" {
		// Default scope: feature AND blocker both in the active
		// domain. Cross-domain blockers only surface with --all-domains.
		q += ` AND ` + frag
		args2 = append(args2, fragArgs...)
		bfrag, bfragArgs := scope.Where("b")
		q += ` AND ` + bfrag
		args2 = append(args2, bfragArgs...)
	}
	q += ` ORDER BY f.key`
	rows, err := store.DB().Query(q, args2...)
	if err != nil {
		return err
	}
	defer rows.Close()

	type chain struct {
		ftitle, fkey, btype, bkey, bstatus, fdomain, bdomain string
	}
	byFeature := map[string][]chain{}
	for rows.Next() {
		var c chain
		if err := rows.Scan(&c.fkey, &c.ftitle, &c.btype, &c.bkey, &c.bstatus, &c.fdomain, &c.bdomain); err != nil {
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
			domainTag := ""
			if c.fdomain != c.bdomain && c.bdomain != "" {
				// Cross-domain blocker; tag the blocker's domain so
				// the reader sees the boundary explicitly.
				domainTag = fmt.Sprintf(" [domain: %s]", c.bdomain)
			}
			fmt.Printf("    waiting on %s `%s` (%s)%s\n", displayType(vocab, strings.ToLower(c.btype)), c.bkey, status, domainTag)
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
	return store, graphRepoKey(projectRoot), nil
}

// graphRepoKey is the single derivation of the graph partition key for
// every graph writer and reader in this package. It MUST match the key
// `hero scan` writes under and the key traversal.Why / hero blocked /
// hero impact filter on, so nodes written by any command are reachable.
//
// Do NOT derive a graph repoKey via filepath.Base(projectRoot): that
// yields the bare directory name ("hero") and diverges from
// gitutil.RepoKey ("hero-engine/hero") whenever an origin remote is set,
// silently partitioning writes into a subgraph the reader never queries.
// (See graph-why-resolution-and-peer-spec-indexing/spec.md.)
// Non-CLI graph writers (the peer-receive ingest path) call
// gitutil.RepoKey directly — same derivation, since gitutil cannot be
// wrapped in package graph without an import cycle.
func graphRepoKey(projectRoot string) string {
	return gitutil.RepoKey(projectRoot)
}

// reconcileSpecGraph re-writes the spec subgraph from frontmatter (the
// durable source of truth) into the graph substrate before a read-side
// traversal queries it. It is the graph-side analogue of the index's
// RefreshIfStale: `hero why` / `hero blocked` read graph.db — a
// regenerable cache with no read-side self-heal — so a spec created since
// the last ingest (notably a peer-received spec-out spec that never hit an
// ingest path) is otherwise invisible to them while `hero graph` /
// `hero search` find it in the self-healing index.
//
// It reconciles ONLY the spec subgraph (not code/sessions/git), so it is
// far cheaper than a full `hero graph reingest` — the same cost
// `hero blocked` already pays. spec.WriteGraph is idempotent, so on a warm
// graph this produces no new history for unchanged specs. It runs
// unconditionally (not gated on projectGraphCold): the bug it fixes is a
// warm graph missing a handful of newer specs, which a cold-graph guard
// would skip entirely.
//
// repoKey MUST be graphRepoKey(projectRoot) so the reconciled nodes land
// in the partition the reader filters on — this is also what keeps a local
// spec's node live in the local partition even when a federated peer copy
// exists under a sibling repoKey (see the team-oauth federation case).
//
// Best-effort by contract: any error leaves the graph as-is and the caller
// queries whatever it already holds. A read command must never fail
// because the reconcile couldn't run.
func reconcileSpecGraph(store *graph.Store, projectRoot, repoKey string, cfg *config.Config) {
	if store == nil || cfg == nil {
		return
	}
	specs, err := spec.Discover(cfg.HeroDir(projectRoot))
	if err != nil {
		return
	}
	_, _ = spec.WriteGraph(specs, repoKey, graph.DomainFor(*cfg, graph.IntrinsicActive), store)
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

func isInteractive(out io.Writer) bool {
	return prompt.IsOutputTTY(out)
}

func oneLineString(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return s
}

func shortShaImpact(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
