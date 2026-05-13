package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/codescan"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/extract"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/knowledge"
	"github.com/hero-engine/hero/internal/memory"
	"github.com/hero-engine/hero/internal/mission"
	"github.com/hero-engine/hero/internal/nextdoc"
	"github.com/hero-engine/hero/internal/scan"
	"github.com/hero-engine/hero/internal/sessions"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var (
	scanDryRun   bool
	scanForce    bool
	scanCodeOnly bool
	scanNoHooks  bool
)

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
  hero scan --dry-run    # show what would be generated
  hero scan --force      # overwrite existing entries`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().BoolVar(&scanDryRun, "dry-run", false, "show what would be generated without writing")
	scanCmd.Flags().BoolVar(&scanForce, "force", false, "overwrite existing knowledge entries")
	scanCmd.Flags().BoolVar(&scanCodeOnly, "code", false, "run code intelligence scan only")
	scanCmd.Flags().BoolVar(&scanNoHooks, "no-hooks", false, "deprecated no-op — hook install moved to 'hero init'; kept for backwards compatibility")

	RegisterSmoke(scanCmd, func(cmd *cobra.Command) error {
		scanDryRun = true
		return runScan(cmd, nil)
	})
}

func runScan(cmd *cobra.Command, args []string) error {
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

	// Code-only mode
	if scanCodeOnly {
		return runCodeScan(cfg, projectRoot, heroDir)
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
	if cfg.CodeScan.IsDisabled() {
		fmt.Println("Code scanning is disabled (code_scan.depth: disabled)")
		return nil
	}

	codeDir := cfg.CodeDir(projectRoot)

	// Load previous checksums for incremental scanning
	prevChecksums, err := codescan.LoadChecksums(codeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load previous checksums: %v\n", err)
	}

	fmt.Printf("Scanning code structure (depth: %s, parser: %s)...\n", cfg.CodeScan.Depth, resolveParser(cfg.CodeScan.Parser))
	start := time.Now()

	scanner := codescan.NewScannerWithMode(cfg.CodeScan, projectRoot, resolveParser(cfg.CodeScan.Parser))
	result, err := scanner.Scan(prevChecksums)
	if err != nil {
		return fmt.Errorf("code scan failed: %w", err)
	}

	// Detect hot files
	result.HotFiles = codescan.DetectHotFiles(result, projectRoot)

	elapsed := time.Since(start)

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

	fmt.Printf("Found %d packages, %d files, %d exported symbols (%s)\n",
		len(result.Packages), totalFiles, totalSymbols, elapsed.Round(time.Millisecond))

	if len(result.DepGraph) > 0 {
		fmt.Printf("Dependency edges: %d\n", len(result.DepGraph))
	}
	if len(result.HotFiles) > 0 {
		fmt.Printf("Hot files identified: %d\n", len(result.HotFiles))
	}

	if scanDryRun {
		fmt.Println("\nDry run — no code knowledge files written.")
		// Print package list
		for _, pkg := range result.Packages {
			fmt.Printf("  [%s] %s (%d files, %d symbols)\n",
				pkg.Language, pkg.Path, pkg.FileCount, len(pkg.Symbols))
		}
		return nil
	}

	// Generate knowledge files
	if err := codescan.GenerateKnowledge(result, codeDir); err != nil {
		return fmt.Errorf("generating code knowledge: %w", err)
	}

	fmt.Printf("Code knowledge written to %s\n", codeDir)

	// Additively populate the unified knowledge graph. Failure here must
	// not break scan — the markdown output above is still authoritative.
	if err := writeCodeSubgraph(cfg, result, projectRoot, heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: graph ingest failed: %v\n", err)
	}

	// Tour: tell the user what they have now and where to look first.
	// New users land on a workspace full of generated content with no
	// obvious entry point — these three pointers are the highest-value
	// next reads.
	fmt.Println()
	fmt.Println("What's next:")
	fmt.Println("  hero status                     — overview of specs and knowledge")
	fmt.Println("  hero suggest                    — high-churn files without spec coverage")
	fmt.Println("  hero ask 'how does X work'      — answer questions from the indexed corpus")
	fmt.Println("  hero design <slug>              — start writing a feature spec")
	fmt.Println("  Review the stubs in .hero/knowledge/ and AGENTS.md — they're starting points, enrich them.")
	return nil
}

// writeCodeSubgraph opens (or creates) the graph DB and writes the
// code subgraph plus the work subgraph (specs, sessions, git log).
// Errors are returned so the caller can log a warning without failing
// the whole scan.
//
// Per master-ingest-restore AC-6, the per-step "Graph X: …" prints
// are replaced by a single structured "Graph ingest summary" block
// emitted at the end. Each step contributes a stepResult to the
// shared report; failures and skips are first-class outcomes (per
// AC-8 — no step blocks any other).
func writeCodeSubgraph(cfg config.Config, result *codescan.Result, projectRoot, heroDir string) error {
	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	report := &ingestReport{}

	codeSummary, err := codescan.WriteGraph(result, store)
	if err != nil {
		report.add(stepResult{name: "code", failed: true, err: err})
	} else {
		report.add(stepResult{
			name: "code",
			ok:   true,
			detail: fmt.Sprintf("%d packages, %d files, %d symbols, %d edges",
				codeSummary.Packages, codeSummary.Files, codeSummary.Symbols, codeSummary.Edges),
		})
	}

	// Work subgraph — best-effort, never fail the scan. Each substep
	// adds its own row to the shared report; only the outermost wrap
	// prints a Warning if the whole subgraph blew up.
	if err := writeWorkSubgraph(cfg, projectRoot, heroDir, store, report); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: work-subgraph ingest failed: %v\n", err)
	}

	// Sibling repos configured in hero.json — ingest their specs into our
	// local graph.db so unified search surfaces them without a flag.
	// Best-effort: skip inaccessible repos, log per-repo failures.
	if err := writeSiblingSubgraphs(cfg, projectRoot, store, report); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sibling-repo ingest failed: %v\n", err)
	}

	// Project graph nodes into the unified FTS5 search index.
	idx, idxErr := index.Open(heroDir)
	if idxErr == nil {
		n, projErr := idx.ProjectGraphNodes(store.DB())
		if projErr != nil {
			report.add(stepResult{name: "fts-project", failed: true, err: projErr})
		} else {
			report.add(stepResult{name: "fts-project", ok: true,
				detail: fmt.Sprintf("%d nodes projected into search index", n)})
		}
		idx.Close()
	}

	report.print()
	return nil
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
		summary, err := spec.WriteGraph(specs, siblingKey, store)
		if err != nil {
			report.add(stepResult{name: "sibling " + alias, failed: true, err: err})
			continue
		}
		report.add(stepResult{
			name: "sibling " + alias,
			ok:   true,
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
	} else if specSummary, err := spec.WriteGraph(specs, repoKey, store); err != nil {
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
		ingested := 0
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".local.md") {
				continue
			}
			path := filepath.Join(heroDir, "next", name)
			if err := handoff.IngestUserFile(store, repoKey, path); err != nil {
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
		report.add(stepResult{name: "tier-2", failed: true, err: err})
	} else if tierTwo.Skipped {
		report.add(stepResult{name: "tier-2", skipped: true, reason: tierTwo.Reason})
	} else if tierTwo.Sources > 0 {
		report.add(stepResult{
			name: "tier-2",
			ok:   true,
			detail: fmt.Sprintf("%d decisions, %d concepts, %d edges across %d sources (%d cached)",
				tierTwo.Decisions, tierTwo.Concepts, tierTwo.Edges, tierTwo.Sources, tierTwo.Cached),
		})
	} else {
		report.add(stepResult{name: "tier-2", skipped: true, reason: "no sources to extract from"})
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
	if parser == "auto" || parser == "treesitter" {
		// Check if tree-sitter CLI is available
		if _, err := exec.LookPath("tree-sitter"); err == nil {
			return "treesitter"
		}
		if parser == "treesitter" {
			fmt.Fprintf(os.Stderr, "Warning: tree-sitter CLI not found on PATH, falling back to heuristic parser\n")
		}
		return "heuristic"
	}
	return parser
}
