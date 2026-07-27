package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/codescan"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/extract"
	"github.com/hero-engine/hero/internal/filelock"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/knowledge"
	"github.com/hero-engine/hero/internal/memory"
	"github.com/hero-engine/hero/internal/mission"
	"github.com/hero-engine/hero/internal/nextdoc"
	"github.com/hero-engine/hero/internal/scan"
	// Anchor engineering scanner registration so `hero scan` finds the
	// active pack via Dispatch when running under test harnesses
	// (cmd/hero/main.go's blank import only covers the production
	// binary). Future packs may follow the same pattern from their
	// own cli entry points.
	_ "github.com/hero-engine/hero/domains/engineering/scan"
	"github.com/hero-engine/hero/internal/sessions"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

const defaultIncrementalScanDeadline = 10 * time.Second

var (
	scanDryRun      bool
	scanForce       bool
	scanCodeOnly    bool
	scanNoHooks     bool
	scanIncremental bool
	scanDeadline    time.Duration
	scanQuiet       bool
)

// activeDomain returns the workspace's active domain pack, defaulting
// to engineering when unset. Used at graph-write call sites that need
// to stamp DSKG domain tags through the wire-through path described in
// scan-pluggability spec §5.
func activeDomain(cfg config.Config) string {
	if cfg.Domain == "" {
		return "engineering"
	}
	return cfg.Domain
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Analyze codebase and generate initial knowledge base entries",
	Long: `Scans the current project to detect the technology stack, project structure,
build tools, CI/CD configuration, linters, test frameworks, and common patterns.

By default, generates knowledge base entries (context, conventions, rules) based
on what is detected, plus code intelligence (symbols, packages, dependencies).

Use --dry-run to preview without writing anything.
Use --code to run only the code intelligence scan.

Generated entries are stubs — review and enrich them with project-specific details,
or use /scan in your agent to let the AI fill them in.

Code scanning depth is controlled by code_scan.depth in hero.json:
  "normal"   — structure extraction only (fast, default)
  "deep"     — structure + LLM-generated descriptions
  "disabled" — skip code scanning entirely

Examples:
  hero scan              # full scan (stack + code)
  hero scan --code       # code intelligence only
  hero scan --code --incremental --deadline 10s -q
  hero scan --dry-run    # show what would be generated
  hero scan --force      # overwrite existing entries`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().BoolVar(&scanDryRun, "dry-run", false, "show what would be generated without writing")
	scanCmd.Flags().BoolVar(&scanForce, "force", false, "overwrite existing knowledge entries")
	scanCmd.Flags().BoolVar(&scanCodeOnly, "code", false, "run code intelligence scan only")
	scanCmd.Flags().BoolVar(&scanIncremental, "incremental", false, "refresh changed code intelligence only")
	scanCmd.Flags().DurationVar(&scanDeadline, "deadline", defaultIncrementalScanDeadline, "maximum incremental refresh duration")
	scanCmd.Flags().BoolVarP(&scanQuiet, "quiet", "q", false, "suppress output and normalize incremental failures for hooks")
	scanCmd.Flags().BoolVar(&scanNoHooks, "no-hooks", false, "deprecated no-op — hook install moved to 'hero init'; kept for backwards compatibility")

	RegisterSmoke(scanCmd, func(cmd *cobra.Command) error {
		scanDryRun = true
		return runScan(cmd, nil)
	})
}

func runScan(cmd *cobra.Command, args []string) (retErr error) {
	if scanIncremental && !scanCodeOnly {
		return fmt.Errorf("--incremental requires --code")
	}
	if scanQuiet && !scanIncremental {
		return fmt.Errorf("--quiet is only supported with --code --incremental")
	}
	if scanIncremental && scanQuiet {
		defer func() {
			if retErr != nil {
				retErr = nil
			}
		}()
	}
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Pre-commit hook install moved to `hero init` (spec:
	// scan-output-cleanup). `--no-hooks` is preserved as a no-op flag
	// for backwards compatibility with existing scripts.
	_ = scanNoHooks

	// Domain-aware dispatch (scan-pluggability spec §2):
	//   - If the active pack ships no scanner manifest, print the
	//     friendly skip and exit 0.
	//   - --code is an engineering-only flag — reject it for any
	//     non-engineering active pack.
	//   - Otherwise let the active pack route. The engineering
	//     scanner's Scan() is a no-op stub at this milestone (see
	//     domains/engineering/scan/scanner.go); the legacy direct
	//     call path below continues to drive the actual engineering
	//     scan until scan-pluggability §8 PR 2 (relocation) lands.
	domain := activeDomain(cfg)
	if scanCodeOnly && domain != "engineering" {
		return fmt.Errorf("hero scan --code is engineering-only; current active domain is %q", domain)
	}
	scanOpts := scan.ScanOpts{
		ProjectRoot: projectRoot,
		HeroDir:     heroDir,
		Config:      cfg,
		Flags:       map[string]any{"code": scanCodeOnly, "dry-run": scanDryRun, "force": scanForce},
		DryRun:      scanDryRun,
		Force:       scanForce,
		Reporter:    scan.StdoutReporter(scanOutput()),
	}
	if _, err := scan.Dispatch("scan", scanOpts); err != nil {
		if errors.Is(err, scan.ErrScannerNotFound) {
			fmt.Printf("%s pack does not ship a scanner; nothing to do.\n", domain)
			return nil
		}
		if errors.Is(err, scan.ErrSubcommandUnsupported) {
			return fmt.Errorf("%s pack does not implement 'scan': %w", domain, err)
		}
		return fmt.Errorf("dispatch scan: %w", err)
	}

	// Code-only mode
	if scanCodeOnly {
		err := runCodeScanContext(cmd.Context(), cfg, projectRoot, heroDir)
		if scanIncremental && scanQuiet {
			return nil
		}
		return err
	}

	// Run stack analysis
	result, err := scan.Analyze(projectRoot)
	if err != nil {
		return fmt.Errorf("scanning project: %w", err)
	}

	// Print summary
	fmt.Print(result.Summary())

	// Show matched skills
	skills := result.StackSkills()
	if len(skills) > 0 {
		fmt.Printf("\nMatched Hero skills: %v\n", skills)
	}

	// Show import sources
	if len(result.ImportSources) > 0 {
		fmt.Printf("\nFound %d existing knowledge file(s) to import.\n", len(result.ImportSources))
	}

	// Generate entries
	entries := scan.Generate(result, heroDir)
	if len(entries) == 0 {
		fmt.Println("\nNo knowledge entries to generate.")
	} else {
		fmt.Printf("\nKnowledge entries to generate: %d\n", len(entries))
		for _, e := range entries {
			fmt.Printf("  [%s] %s\n", e.Type, e.Slug)
		}

		if !scanDryRun {
			// Plan merge — decide what to create, update, or skip
			decisions := scan.PlanMerge(entries, scanForce)

			// Show merge plan
			for _, d := range decisions {
				switch d.Strategy {
				case scan.MergeCreate:
					fmt.Printf("  + [%s] %s — %s\n", d.Entry.Type, d.Entry.Slug, d.Reason)
				case scan.MergeUpdate:
					fmt.Printf("  ~ [%s] %s — %s\n", d.Entry.Type, d.Entry.Slug, d.Reason)
				case scan.MergeSkipCustomized:
					fmt.Printf("  = [%s] %s — %s\n", d.Entry.Type, d.Entry.Slug, d.Reason)
				case scan.MergeForce:
					fmt.Printf("  ! [%s] %s — %s\n", d.Entry.Type, d.Entry.Slug, d.Reason)
				}
			}

			// Execute merge
			mergeResult, err := scan.ExecuteMerge(decisions)
			if err != nil {
				return fmt.Errorf("writing entries: %w", err)
			}

			fmt.Printf("\nCreated: %d, Updated: %d, Skipped (customized): %d",
				mergeResult.Created, mergeResult.Updated, mergeResult.Skipped)
			if mergeResult.Forced > 0 {
				fmt.Printf(", Forced: %d", mergeResult.Forced)
			}
			fmt.Println()

			if mergeResult.Skipped > 0 && !scanForce {
				fmt.Println("Use --force to overwrite customized entries.")
			}
			if mergeResult.Updated > 0 && mergeResult.Skipped == 0 && mergeResult.Created == 0 {
				fmt.Println("Updated entries hadn't been customized — they regenerate cleanly. Hand-edits are preserved on future scans.")
			}

			if mergeResult.Created+mergeResult.Updated > 0 {
				fmt.Println("\nGenerated entries are stubs — review and enrich them with project-specific details.")
			}
		}
	}

	if scanDryRun {
		fmt.Println("\nDry run — no files written.")
		return nil
	}

	// Re-index the spec corpus so newly-written stubs are searchable
	// immediately. Best-effort: a failure here doesn't break scan, the
	// user can always re-run `hero index`.
	if stats, err := index.Rebuild(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: spec index rebuild failed: %v\n", err)
	} else {
		fmt.Printf("Indexed %d specs for hero ask / search.\n", stats.TotalSpecs)
	}

	// Run code scan (unless disabled)
	if !cfg.CodeScan.IsDisabled() {
		fmt.Println()
		return runCodeScan(cfg, projectRoot, heroDir)
	}

	return nil
}

func runCodeScan(cfg config.Config, projectRoot, heroDir string) error {
	return runCodeScanContext(context.Background(), cfg, projectRoot, heroDir)
}

func runCodeScanContext(parent context.Context, cfg config.Config, projectRoot, heroDir string) error {
	if cfg.CodeScan.IsDisabled() {
		if !scanQuiet {
			fmt.Fprintln(scanOutput(), "Code scanning is disabled (code_scan.depth: disabled)")
		}
		return nil
	}

	ctx := parent
	cancel := func() {}
	if scanIncremental {
		if scanDeadline <= 0 {
			return fmt.Errorf("incremental deadline must be greater than zero")
		}
		ctx, cancel = context.WithTimeout(parent, scanDeadline)
	}
	defer cancel()

	parser := resolveParserQuiet(cfg.CodeScan.Parser, scanQuiet)
	if !scanQuiet {
		fmt.Fprintf(scanOutput(), "Scanning code structure (depth: %s, parser: %s)...\n", cfg.CodeScan.Depth, parser)
	}
	stats, err := refreshCodeIndex(ctx, cfg, projectRoot, heroDir, codeRefreshOptions{
		Incremental: scanIncremental,
		Quiet:       scanQuiet,
		DryRun:      scanDryRun,
		Parser:      parser,
	})
	if err != nil {
		return err
	}
	if stats.Skipped {
		if !scanQuiet {
			fmt.Fprintf(scanOutput(), "Code refresh skipped: %s\n", stats.SkipReason)
		}
		return nil
	}
	result := stats.Result
	if result == nil {
		return fmt.Errorf("code refresh returned no result")
	}

	// Summary
	totalFiles := 0
	totalSymbols := 0
	for _, pkg := range result.Packages {
		totalFiles += pkg.FileCount
		for _, s := range pkg.Symbols {
			if s.Exported {
				totalSymbols++
			}
		}
	}

	if !scanQuiet {
		fmt.Fprintf(scanOutput(), "Found %d packages, %d files, %d exported symbols (%s)\n",
			len(result.Packages), totalFiles, totalSymbols, stats.Elapsed.Round(time.Millisecond))

		if len(result.DepGraph) > 0 {
			fmt.Fprintf(scanOutput(), "Dependency edges: %d\n", len(result.DepGraph))
		}
		if len(result.HotFiles) > 0 {
			fmt.Fprintf(scanOutput(), "Hot files identified: %d\n", len(result.HotFiles))
		}
	}

	if scanDryRun {
		fmt.Fprintln(scanOutput(), "\nDry run — no code knowledge files written.")
		// Print package list
		for _, pkg := range result.Packages {
			fmt.Fprintf(scanOutput(), "  [%s] %s (%d files, %d symbols)\n",
				pkg.Language, pkg.Path, pkg.FileCount, len(pkg.Symbols))
		}
		return nil
	}

	if !scanQuiet {
		if stats.Wrote {
			fmt.Fprintf(scanOutput(), "Code knowledge written to %s\n", cfg.CodeDir(projectRoot))
		} else {
			fmt.Fprintln(scanOutput(), "Code knowledge is current; no writes needed.")
		}
	}

	// Tour: tell the user what they have now and where to look first.
	// New users land on a workspace full of generated content with no
	// obvious entry point — these three pointers are the highest-value
	// next reads.
	if !scanQuiet {
		fmt.Fprintln(scanOutput())
		fmt.Fprintln(scanOutput(), "What's next:")
		fmt.Fprintln(scanOutput(), "  hero status                     — overview of specs and knowledge")
		fmt.Fprintln(scanOutput(), "  hero suggest                    — high-churn files without spec coverage")
		fmt.Fprintln(scanOutput(), "  hero ask 'how does X work'      — answer questions from the indexed corpus")
		fmt.Fprintln(scanOutput(), "  hero design <slug>              — start writing a feature spec")
		fmt.Fprintln(scanOutput(), "  Review the stubs in .hero/knowledge/ and AGENTS.md — they're starting points, enrich them.")
	}
	return nil
}

type codeRefreshOptions struct {
	Incremental bool
	Quiet       bool
	DryRun      bool
	Parser      string
}

type codeRefreshStats struct {
	Result             *codescan.Result
	Graph              codescan.GraphWriteSummary
	Embeddings         embeddingPhaseOutcome
	Projected          int
	Changed            bool
	Wrote              bool
	Complete           bool
	Skipped            bool
	SkipReason         string
	Phase              string
	PostStructureReady bool
	Elapsed            time.Duration
}

func refreshCodeIndex(ctx context.Context, cfg config.Config, projectRoot, heroDir string, opts codeRefreshOptions) (codeRefreshStats, error) {
	start := time.Now()
	stats := codeRefreshStats{Phase: "lock"}
	lock, busy, err := acquireCodeRefreshLock(heroDir)
	if err != nil {
		return stats, fmt.Errorf("acquire code refresh lock: %w", err)
	}
	if busy {
		stats.Skipped = true
		stats.SkipReason = "another code refresh owns the workspace lock"
		stats.Elapsed = time.Since(start)
		return stats, nil
	}
	defer lock.Close()

	codeDir := cfg.CodeDir(projectRoot)
	stats.Phase = "load-state"
	prevChecksums, prevCache, cacheUsable, err := codescan.LoadScanState(codeDir, opts.Parser)
	if err != nil {
		if opts.Incremental {
			stats.Skipped = true
			stats.SkipReason = "scan cache is unavailable; run hero scan --code to rebuild it"
			stats.Elapsed = time.Since(start)
			return stats, nil
		}
		prevChecksums, prevCache = nil, nil
	}
	if opts.Incremental && !cacheUsable {
		stats.Skipped = true
		stats.SkipReason = "scan cache is unavailable; run hero scan --code to rebuild it"
		stats.Elapsed = time.Since(start)
		return stats, nil
	}
	if !cacheUsable {
		prevChecksums, prevCache = nil, nil
	}

	stats.Phase = "scan"
	scanner := codescan.NewScannerWithMode(cfg.CodeScan, projectRoot, opts.Parser)
	result, err := scanner.ScanContext(ctx, prevChecksums, prevCache)
	if err != nil {
		return stats, fmt.Errorf("code scan failed: %w", err)
	}
	stats.Result = result
	stats.Complete = result.Complete
	stats.Changed = result.Stats.Added+result.Stats.Changed+result.Stats.Deleted > 0 ||
		(cacheUsable && result.Stats.Reparsed > 0)
	if !result.Complete {
		return stats, fmt.Errorf("code scan incomplete: %d source error(s)", len(result.Diagnostics))
	}
	if opts.Incremental && cacheUsable && !stats.Changed {
		stats.Phase = "post-structure"
		stats.PostStructureReady = true
		if !opts.DryRun {
			stats.Phase = "embeddings"
			stats.Embeddings, err = runWorkspaceEmbeddingPhase(ctx, cfg, heroDir)
			if err != nil {
				return stats, fmt.Errorf("refreshing embeddings: %w", err)
			}
			stats.Phase = "post-structure"
		}
		stats.Elapsed = time.Since(start)
		return stats, nil
	}

	result.HotFiles = codescan.DetectHotFiles(result, projectRoot)
	if opts.DryRun {
		stats.Phase = "complete"
		stats.Elapsed = time.Since(start)
		return stats, nil
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}

	stats.Phase = "graph"
	store, err := graph.Open(heroDir)
	if err != nil {
		return stats, fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()
	graphSummary, err := codescan.WriteGraphContext(ctx, result, store, activeDomain(cfg))
	if err != nil {
		return stats, fmt.Errorf("reconciling code graph: %w", err)
	}
	stats.Graph = *graphSummary

	report := &ingestReport{}
	report.add(stepResult{
		name: "code", ok: true,
		detail: fmt.Sprintf("%d packages, %d files, %d symbols, %d edges",
			graphSummary.Packages, graphSummary.Files, graphSummary.Symbols, graphSummary.Edges),
	})
	if !opts.Incremental {
		if err := writeWorkSubgraph(cfg, projectRoot, heroDir, store, report); err != nil {
			report.add(stepResult{name: "work", failed: true, err: err})
		}
		if err := writeSiblingSubgraphs(cfg, projectRoot, store, report); err != nil {
			report.add(stepResult{name: "sibling-repos", failed: true, err: err})
		}
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}

	stats.Phase = "projection"
	idx, err := index.Open(heroDir)
	if err != nil {
		return stats, fmt.Errorf("opening search index: %w", err)
	}
	projected, err := idx.ProjectGraphNodesContext(ctx, store.DB())
	if err != nil {
		idx.Close()
		return stats, fmt.Errorf("projecting graph nodes: %w", err)
	}
	stats.Projected = projected
	report.add(stepResult{name: "fts-project", ok: true,
		detail: fmt.Sprintf("%d nodes projected into search index", projected)})

	stats.Phase = "knowledge"
	if err := codescan.GenerateKnowledgeContext(ctx, result, codeDir); err != nil {
		idx.Close()
		return stats, fmt.Errorf("generating code knowledge: %w", err)
	}

	stats.Phase = "post-structure"
	stats.PostStructureReady = true
	stats.Phase = "embeddings"
	stats.Embeddings, err = runEmbeddingPhase(ctx, cfg, heroDir, idx.RawDB(), store.DB())
	if err != nil {
		idx.Close()
		return stats, fmt.Errorf("refreshing embeddings: %w", err)
	}
	if stats.Embeddings.Skipped {
		report.add(stepResult{name: "embeddings", skipped: true, reason: stats.Embeddings.Reason})
	} else {
		report.add(stepResult{name: "embeddings", ok: true, detail: fmt.Sprintf("embeddings %s", stats.Embeddings.Stats)})
	}
	if err := idx.Close(); err != nil {
		return stats, fmt.Errorf("closing search index: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}

	stats.Phase = "state"
	if err := codescan.CommitScanStateContext(ctx, result, codeDir, opts.Parser); err != nil {
		return stats, fmt.Errorf("committing scan state: %w", err)
	}
	stats.Wrote = true
	stats.Phase = "complete"
	stats.Elapsed = time.Since(start)
	if !opts.Quiet {
		report.print()
	}
	return stats, nil
}

type codeRefreshLock struct {
	lock *filelock.Lock
}

func acquireCodeRefreshLock(heroDir string) (*codeRefreshLock, bool, error) {
	cacheDir := filepath.Join(heroDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, false, err
	}
	lock, busy, err := filelock.TryAcquire(filepath.Join(cacheDir, "code-refresh.lock"), 0o644)
	if err != nil {
		return nil, false, fmt.Errorf("acquire code refresh lock: %w", err)
	}
	if busy {
		return nil, true, nil
	}
	return &codeRefreshLock{lock: lock}, false, nil
}

func (l *codeRefreshLock) Close() error {
	if l == nil || l.lock == nil {
		return nil
	}
	if err := l.lock.Close(); err != nil {
		return fmt.Errorf("close code refresh lock: %w", err)
	}
	return nil
}

func scanOutput() io.Writer {
	if scanQuiet {
		return io.Discard
	}
	return os.Stdout
}

// stepResult is the outcome of one ingest step. Exactly one of ok /
// skipped / failed is true; detail/reason/err carry the human-facing
// message.
type stepResult struct {
	name    string
	ok      bool
	skipped bool
	failed  bool
	detail  string // populated when ok
	reason  string // populated when skipped
	err     error  // populated when failed
}

// ingestReport collects stepResults and renders a single block.
type ingestReport struct {
	steps []stepResult
}

func (r *ingestReport) add(s stepResult) { r.steps = append(r.steps, s) }

// print emits the "Graph ingest summary" block per master-ingest-
// restore AC-6. Glyphs: ✅ ran, ⊘ skipped (preconditions unmet),
// ❌ failed (best-effort caught the error).
func (r *ingestReport) print() {
	if len(r.steps) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("Graph ingest summary:")
	for _, s := range r.steps {
		switch {
		case s.ok:
			fmt.Printf("  ✅ %-12s %s\n", s.name+":", s.detail)
		case s.skipped:
			fmt.Printf("  ⊘  %-12s %s\n", s.name+":", s.reason)
		case s.failed:
			fmt.Printf("  ❌ %-12s %v\n", s.name+":", s.err)
		}
	}
	for _, s := range r.steps {
		if s.name == "enrichment" && s.skipped {
			fmt.Println("  note: enrichment is optional LLM analysis; the structural graph (specs, relations) is built without it.")
			break
		}
	}
}

// writeSiblingSubgraphs iterates hero.json's `repos` block and ingests
// each accessible sibling repo's specs into the local graph.db, tagged
// with the sibling's git remote-origin key (gitutil.RepoKey).
//
// Limitation: the local graph.db has a UNIQUE (type, key) constraint so
// a sibling spec sharing a slug with a local spec will overwrite (or
// be overwritten by) the local one. In practice slugs are repo-scoped
// in users' workflows; lifting this constraint to (type, repo, key) is
// future work tracked under unified-search v2.
func writeSiblingSubgraphs(cfg config.Config, projectRoot string, store *graph.Store, report *ingestReport) error {
	if len(cfg.Repos) == 0 {
		return nil
	}
	statuses := cfg.ResolveAllRepos(projectRoot)
	localKey := gitutil.RepoKey(projectRoot)
	for alias, status := range statuses {
		if !status.Accessible {
			continue
		}
		siblingKey := gitutil.RepoKey(status.Path)
		if siblingKey == localKey {
			// Same repo as the local one — already scanned above.
			continue
		}
		siblingHero := filepath.Join(status.Path, cfg.Folder)
		specs, err := spec.Discover(siblingHero)
		if err != nil {
			report.add(stepResult{name: "sibling " + alias, failed: true, err: err})
			continue
		}
		if len(specs) == 0 {
			continue
		}
		summary, err := spec.WriteGraph(specs, siblingKey, graph.DomainFor(cfg, graph.IntrinsicActive), store)
		if err != nil {
			report.add(stepResult{name: "sibling " + alias, failed: true, err: err})
			continue
		}
		report.add(stepResult{
			name:   "sibling " + alias,
			ok:     true,
			detail: fmt.Sprintf("%s, %d specs, %d edges", siblingKey, summary.Specs, summary.Edges),
		})
	}
	return nil
}

func writeWorkSubgraph(cfg config.Config, projectRoot, heroDir string, store *graph.Store, report *ingestReport) error {
	repoKey := gitutil.RepoKey(projectRoot)

	specs, err := spec.Discover(heroDir)
	if err != nil {
		report.add(stepResult{name: "planning", failed: true, err: err})
	} else if specSummary, err := spec.WriteGraph(specs, repoKey, graph.DomainFor(cfg, graph.IntrinsicActive), store); err != nil {
		report.add(stepResult{name: "planning", failed: true, err: err})
	} else {
		report.add(stepResult{
			name: "planning",
			ok:   true,
			detail: fmt.Sprintf("%d specs, %d criteria, %d spec-edges",
				specSummary.Specs, specSummary.Criteria, specSummary.Edges),
		})
	}

	// Mission charter — best-effort. Missing file means no charter
	// authored yet (fresh repos); skips silently.
	if m, err := mission.LoadFile(heroDir); err != nil {
		report.add(stepResult{name: "mission", failed: true, err: err})
	} else if m == nil {
		report.add(stepResult{name: "mission", skipped: true, reason: "no .hero/mission.md"})
	} else if err := m.WriteGraph(repoKey, store); err != nil {
		report.add(stepResult{name: "mission", failed: true, err: err})
	} else {
		report.add(stepResult{
			name: "mission",
			ok:   true,
			detail: fmt.Sprintf("%s v%s, %d principles, %d anti-patterns",
				m.Scope, m.Version, len(m.Principles), len(m.AntiPatterns)),
		})
	}

	if sessList, err := sessions.List(heroDir); err != nil {
		report.add(stepResult{name: "sessions", failed: true, err: err})
	} else if sessSummary, err := sessions.WriteGraph(sessList, repoKey, store); err != nil {
		report.add(stepResult{name: "sessions", failed: true, err: err})
	} else if sessSummary.Sessions > 0 {
		report.add(stepResult{
			name:   "sessions",
			ok:     true,
			detail: fmt.Sprintf("%d sessions, %d edges", sessSummary.Sessions, sessSummary.Edges),
		})
	}

	if gitSummary, err := gitutil.WriteGitLogGraph(projectRoot, repoKey, 0, store); err != nil {
		report.add(stepResult{name: "git", failed: true, err: err})
	} else if gitSummary.Commits > 0 {
		report.add(stepResult{
			name: "git",
			ok:   true,
			detail: fmt.Sprintf("%d commits, %d persons, %d issues, %d edges",
				gitSummary.Commits, gitSummary.Persons, gitSummary.Issues, gitSummary.Edges),
		})
	}

	if rawSummary, err := knowledge.WriteRawGraph(heroDir, repoKey, store); err != nil {
		report.add(stepResult{name: "raw", failed: true, err: err})
	} else if rawSummary.Documents > 0 {
		report.add(stepResult{
			name:   "raw",
			ok:     true,
			detail: fmt.Sprintf("%d documents", rawSummary.Documents),
		})
	}

	if nextSummary, err := nextdoc.WriteGraph(heroDir, repoKey, store); err != nil {
		report.add(stepResult{name: "next", failed: true, err: err})
	} else if nextSummary.Sessions > 0 || nextSummary.Attempts > 0 {
		report.add(stepResult{
			name: "next",
			ok:   true,
			detail: fmt.Sprintf("%d sessions, %d attempts, %d edges",
				nextSummary.Sessions, nextSummary.Attempts, nextSummary.Edges),
		})
	}

	// Round-trip ingest of per-user handoff files. In solo-no-Cloud
	// mode these files are the cross-machine federation medium —
	// home-laptop projection lands as office-desktop graph state via
	// this path. Best-effort: a parse error on one file doesn't kill
	// the whole scan.
	handoffStep := stepResult{name: "handoff"}
	if entries, _ := os.ReadDir(filepath.Join(heroDir, "next")); len(entries) > 0 {
		// singleTravelFile gates the cross-machine alias mirror to the
		// unambiguous single-identity case, so a brand-new teammate's
		// empty graph never has another user's handoff mirrored onto it
		// (see handoff.IngestUserFile).
		singleTravelFile := len(nextFileUserSlugs(heroDir)) == 1
		ingested := 0
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".local.md") {
				continue
			}
			path := filepath.Join(heroDir, "next", name)
			if err := handoff.IngestUserFile(store, repoKey, graph.DomainFor(cfg, graph.IntrinsicActive), path, nextUserSlug(cfg), singleTravelFile); err != nil {
				handoffStep.failed = true
				handoffStep.err = err
				break
			}
			ingested++
		}
		if !handoffStep.failed && ingested > 0 {
			handoffStep.ok = true
			handoffStep.detail = fmt.Sprintf("%d user file(s)", ingested)
		}
	}
	if handoffStep.ok || handoffStep.failed {
		report.add(handoffStep)
	}

	// Compute File→Criterion participation edges from the join
	// (Criterion --satisfied_by--> Commit --touches--> File). Phase 4
	// of acceptance-criteria-graph. Idempotent — only emits new edges
	// when satisfied_by/touches data has appeared since the last run.
	if part, err := acceptance.ComputeParticipation(store, repoKey); err != nil {
		report.add(stepResult{name: "ac-participation", failed: true, err: err})
	} else if part.Edges > 0 || part.Skipped > 0 {
		report.add(stepResult{
			name: "ac-participation",
			ok:   true,
			detail: fmt.Sprintf("%d new, %d already-current edges across %d files",
				part.Edges, part.Skipped, part.Touched),
		})
	}

	// Tier-2 extraction — best-effort.
	if tierTwo, err := extract.RunAuto(context.Background(), store, heroDir, repoKey); err != nil {
		report.add(stepResult{name: "enrichment", failed: true, err: err})
	} else if tierTwo.Skipped {
		report.add(stepResult{name: "enrichment", skipped: true, reason: tierTwo.Reason})
	} else if tierTwo.Sources > 0 {
		report.add(stepResult{
			name: "enrichment",
			ok:   true,
			detail: fmt.Sprintf("%d decisions, %d concepts, %d edges across %d sources (%d cached)",
				tierTwo.Decisions, tierTwo.Concepts, tierTwo.Edges, tierTwo.Sources, tierTwo.Cached),
		})
	} else {
		report.add(stepResult{name: "enrichment", skipped: true, reason: "no sources to extract from"})
	}

	// Claude Code memory ingest. Omit the step entirely when the dir
	// doesn't exist (the common case for projects without much Claude
	// Code activity yet); emit a friendly skip if the dir exists but
	// is empty so users can see the step ran.
	memDir := memory.DirForProject(projectRoot)
	if memDir != "" {
		if _, statErr := os.Stat(memDir); statErr == nil {
			if memSummary, err := memory.WriteGraph(memDir, repoKey, store); err != nil {
				report.add(stepResult{name: "claude-memory", failed: true, err: err})
			} else if memSummary.Files > 0 {
				report.add(stepResult{
					name:   "claude-memory",
					ok:     true,
					detail: fmt.Sprintf("%d files (scope: local)", memSummary.Files),
				})
			} else {
				report.add(stepResult{
					name:    "claude-memory",
					skipped: true,
					reason:  "Claude Code memory store for this project is empty — Hero will pull from it automatically as you accumulate memories.",
				})
			}
		}
	}

	// Tracker pull.
	pullResult, err := tracker.PullAndWriteGraph(
		cfg.Tracker, cfg.Jira,
		cfg.TrackerKnowledgeDir(projectRoot),
		repoKey, store,
	)
	if err != nil {
		report.add(stepResult{name: "tracker", failed: true, err: err})
	} else if pullResult.Skipped {
		report.add(stepResult{name: "tracker", skipped: true, reason: pullResult.Reason})
	} else if pullResult.Issues > 0 {
		report.add(stepResult{
			name: "tracker",
			ok:   true,
			detail: fmt.Sprintf("%d issues, %d persons, %d edges",
				pullResult.Issues, pullResult.Persons, pullResult.Edges),
		})
	} else {
		report.add(stepResult{name: "tracker", skipped: true, reason: "no open issues found"})
	}

	// Opportunistic team-server sync.
	teamSync, err := runOpportunisticTeamSync(cfg, repoKey, store)
	if err != nil {
		report.add(stepResult{name: "team-server", failed: true, err: err})
	} else if teamSync.Skipped {
		report.add(stepResult{name: "team-server", skipped: true, reason: teamSync.Reason})
	} else {
		detail := fmt.Sprintf("%d pushed, %d nodes / %d edges pulled",
			teamSync.Pushed, teamSync.NodesPulled, teamSync.EdgesPulled)
		if teamSync.EdgesDeferred > 0 {
			detail += fmt.Sprintf(" (%d edges deferred)", teamSync.EdgesDeferred)
		}
		report.add(stepResult{name: "team-server", ok: true, detail: detail})
	}
	return nil
}

// resolveParser returns the actual parser backend that will be used.
func resolveParser(parser string) string {
	return resolveParserQuiet(parser, false)
}

func resolveParserQuiet(parser string, quiet bool) string {
	if parser == "auto" || parser == "treesitter" {
		if _, err := exec.LookPath("tree-sitter"); err == nil {
			return "treesitter"
		}
		if parser == "treesitter" && !quiet {
			fmt.Fprintf(os.Stderr, "Warning: tree-sitter CLI not found on PATH, falling back to heuristic parser\n")
		}
		return "heuristic"
	}
	return parser
}
