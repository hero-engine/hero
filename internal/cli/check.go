package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/reconcile"
	"github.com/hero-engine/hero/internal/snapshot"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// runSatelliteDryRun is the cross-file shim used by check.go and
// elsewhere. It is defined here next to its only consumers so the
// install package does not have to depend on cobra/CLI flags.
func runSatelliteDryRun(heroDir, rootDir string) (*install.RepairResult, error) {
	return install.Repair(install.RepairOptions{
		HeroDir: heroDir,
		RootDir: rootDir,
		DryRun:  true,
	})
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Health check the hero workspace",
	Long: `Reports on workspace health: stale specs, unclaimed work, convention count,
and general corpus statistics.`,
	RunE: runCheck,
}

var checkStaleDays int
var checkReconcile bool
var checkKnowledge bool
var checkJSON bool

func init() {
	checkCmd.Flags().IntVar(&checkStaleDays, "stale-days", 14, "number of days before a planning spec is considered stale")
	checkCmd.Flags().BoolVar(&checkReconcile, "reconcile", false, "auto-fix status drift (promotes planning → delivering when git evidence is clear)")
	checkCmd.Flags().BoolVar(&checkKnowledge, "knowledge", false, "lint knowledge base for stale references, orphans, and pending enrichment")
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "in addition to human output, write a categorized JSON summary to <heroDir>/cache/health.json (consumed by the serve dashboard health cache)")

	// Subverbs migrated from top-level commands. `hero check` alone
	// runs the default health check; subverbs target specific
	// dimensions of corpus health.
	checkCmd.AddCommand(validateCmd)  // hero check validate (was hero validate)
	checkCmd.AddCommand(triageCmd)    // hero check triage   (was hero triage)
	checkCmd.AddCommand(conflictsCmd) // hero check conflicts (was hero conflicts)
}

// healthJSONRow mirrors the on-disk schema consumed by
// internal/serve/projectpage/data/health.go and the in-process
// healthcache. Status is "pass" | "warn" | "fail" — see
// .hero/cache/health.json contract.
type healthJSONRow struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type healthJSONFile struct {
	CapturedAt time.Time       `json:"captured_at"`
	Rows       []healthJSONRow `json:"rows"`
}

// writeHealthJSON persists the categorized check result for downstream
// consumers (the serve dashboard's health cache). Failures are non-fatal
// — the CLI's human output stays the source of truth.
func writeHealthJSON(heroDir string, rows []healthJSONRow) error {
	path := filepath.Join(heroDir, "cache", "health.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir cache dir: %w", err)
	}
	payload := healthJSONFile{
		CapturedAt: time.Now().UTC(),
		Rows:       rows,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// orphanInstructionFile describes a root instruction file present on disk
// whose owning target is not recorded in install-state.json.
type orphanInstructionFile struct {
	file string // "CLAUDE.md" or "AGENTS.md"
	role string // "claude" or "non-claude"
}

// detectOrphanInstructionFiles returns the root instruction files present on
// disk whose owning target is neither recorded in install-state nor
// inferable from on-disk content. CLAUDE.md is orphaned when claude is not
// installed; AGENTS.md is orphaned when no non-claude target is. Informational
// only.
func detectOrphanInstructionFiles(projectRoot string) []orphanInstructionFile {
	// Resolve the installed set as the union of the persisted record and a
	// filesystem probe. PreviouslyInstalledTargets alone reads
	// install-state.json, which is gitignored (machine-local) — on a fresh
	// clone with a healthy install it returns nil, so both CLAUDE.md and
	// AGENTS.md would falsely read as orphaned. InferInstalledTargets
	// reconstructs the set from on-disk content dirs and works on a clone.
	// Mirrors resolveUpgradeTargets and the install-integrity check so
	// "installed" means the same thing everywhere.
	recorded := install.UnionTargets(
		install.PreviouslyInstalledTargets(projectRoot),
		install.InferInstalledTargets(projectRoot),
	)
	claudeRecorded := false
	nonClaudeRecorded := false
	for _, t := range recorded {
		if t == install.TargetClaude {
			claudeRecorded = true
		} else {
			nonClaudeRecorded = true
		}
	}
	var out []orphanInstructionFile
	if fileExists(filepath.Join(projectRoot, "CLAUDE.md")) && !claudeRecorded {
		out = append(out, orphanInstructionFile{file: "CLAUDE.md", role: "claude"})
	}
	if fileExists(filepath.Join(projectRoot, "AGENTS.md")) && !nonClaudeRecorded {
		out = append(out, orphanInstructionFile{file: "AGENTS.md", role: "non-claude"})
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// reportInstallIntegrity runs install.CheckIntegrity and surfaces the
// findings as the `install-integrity` row. Any damaged finding → fail
// (the "agents are running cold" signal); stale only → warn. Silent-pass
// row otherwise. Each finding line ends in the exact repair command.
func reportInstallIntegrity(projectRoot, domain string, addRow func(name, status, message string), issues *int) {
	resolvedDomain := domain
	if resolvedDomain == "" {
		resolvedDomain = "engineering"
	}
	domainFS, domainErr := hero.DomainFS(resolvedDomain)
	if domainErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: install integrity check skipped: resolving domain %q: %v\n", resolvedDomain, domainErr)
		addRow("install-integrity", "warn", fmt.Sprintf("check skipped: resolving domain %q: %v", resolvedDomain, domainErr))
		return
	}
	base := install.Options{
		ContentFS: hero.OverlayFS(domainFS, hero.CoreFS()),
		Domain:    domain,
	}

	findings, err := install.CheckIntegrity(projectRoot, base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: install integrity check failed: %v\n", err)
		addRow("install-integrity", "warn", fmt.Sprintf("check failed: %v", err))
		return
	}
	if len(findings) == 0 {
		addRow("install-integrity", "pass", "installed instruction files match what install would produce")
		return
	}

	damaged := false
	var parts []string
	fmt.Printf("Install integrity (%d):\n", len(findings))
	for _, f := range findings {
		var detail string
		switch f.Kind {
		case install.IntegrityDamaged:
			damaged = true
			if len(f.MissingSections) > 0 {
				detail = fmt.Sprintf("%s (%s): damaged — managed region missing section(s): %s",
					f.File, f.Target, strings.Join(f.MissingSections, ", "))
			} else {
				detail = fmt.Sprintf("%s (%s): damaged — Hero managed region missing", f.File, f.Target)
			}
		case install.IntegrityStale:
			detail = fmt.Sprintf("%s (%s): stale — managed content differs from what this binary would install", f.File, f.Target)
		}
		fmt.Printf("  %s\n", detail)
		fmt.Printf("    repair: run '%s'\n", f.RepairCmd)
		parts = append(parts, fmt.Sprintf("%s; run '%s'", detail, f.RepairCmd))
	}
	fmt.Println()
	*issues += len(findings)
	status := "warn"
	if damaged {
		status = "fail"
	}
	addRow("install-integrity", status, strings.Join(parts, " | "))
}

func runCheck(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// jsonRows accumulates one row per check category for --json output.
	// Status is "pass" / "warn" / "fail". Categories that find no issues
	// emit a "pass" row so the dashboard can show "all clear" per row.
	var jsonRows []healthJSONRow
	addRow := func(name, status, message string) {
		jsonRows = append(jsonRows, healthJSONRow{Name: name, Status: status, Message: message})
	}

	// Use stale-days from config if set
	staleDays := checkStaleDays
	if cfg.Team.StaleDays > 0 && !cmd.Flags().Changed("stale-days") {
		staleDays = cfg.Team.StaleDays
	}

	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	stats, err := idx.GetStats()
	if err != nil {
		return fmt.Errorf("getting stats: %w", err)
	}

	fmt.Println("Hero workspace health check")
	fmt.Println("===========================")
	fmt.Println()

	// Corpus stats
	fmt.Printf("Corpus: %d specs total\n", stats.TotalSpecs)
	fmt.Printf("  %d features, %d bugs, %d conventions, %d decisions\n",
		stats.Features, stats.Bugs, stats.Conventions, stats.Decisions)
	if stats.Initiatives > 0 {
		fmt.Printf("  %d initiatives\n", stats.Initiatives)
	}
	fmt.Printf("  %d files tracked, %d approach docs, %d root causes\n",
		stats.FilesTracked, stats.DecisionDocs, stats.RootCauses)
	fmt.Println()

	issues := 0

	// Check stale specs
	stale, err := idx.CheckStale(staleDays)
	if err == nil && len(stale) > 0 {
		issues += len(stale)
		fmt.Printf("Stale specs (>%d days in planning/in-review):\n", staleDays)
		for _, s := range stale {
			fmt.Printf("  %-30s  %-10s  %-10s  %s\n", s.Slug, s.Type, s.Status, s.Title)
		}
		fmt.Println()
		addRow("stale-specs", "warn", fmt.Sprintf("%d spec(s) older than %d days in planning/in-review", len(stale), staleDays))
	} else {
		addRow("stale-specs", "pass", "no stale specs")
	}

	// Check unclaimed specs
	unclaimed, err := idx.CheckUnclaimed()
	if err == nil && len(unclaimed) > 0 {
		issues += len(unclaimed)
		fmt.Printf("Unclaimed specs (planning/in-review with no claim):\n")
		for _, s := range unclaimed {
			fmt.Printf("  %-30s  %-10s  %-10s  %s\n", s.Slug, s.Type, s.Status, s.Title)
		}
		fmt.Println()
		addRow("unclaimed-specs", "warn", fmt.Sprintf("%d unclaimed spec(s) in planning/in-review", len(unclaimed)))
	} else {
		addRow("unclaimed-specs", "pass", "no unclaimed specs")
	}

	// In-flight summary
	inFlight := stats.Planning + stats.InReview + stats.Delivering
	if inFlight > 0 {
		fmt.Printf("In-flight: %d planning, %d in-review, %d delivering\n",
			stats.Planning, stats.InReview, stats.Delivering)
		if stats.Claims > 0 {
			fmt.Printf("  %d claimed\n", stats.Claims)
		}
		fmt.Println()
	}

	// Status drift (git-derived reconciliation)
	findings := reconcile.Reconcile(heroDir, projectRoot)
	if len(findings) == 0 {
		addRow("status-drift", "pass", "no git-derived status drift")
	} else {
		addRow("status-drift", "warn", fmt.Sprintf("%d spec(s) out of sync with git evidence", len(findings)))
	}
	if len(findings) > 0 {
		fixed := 0
		fmt.Printf("Status drift (%d spec(s) out of sync with git):\n", len(findings))
		for _, f := range findings {
			issues++
			action := "→"
			suffix := ""
			if checkReconcile && f.CanAutoFix() {
				switch f.Kind {
				case reconcile.FindingCompletedStuck:
					// Completed spec stuck in planning — move it to specs/
					destPath, moved, err := moveToSpecs(f.Spec.Path, heroDir)
					if err != nil {
						suffix = fmt.Sprintf("  (move failed: %v)", err)
					} else if moved {
						action = "✓"
						suffix = fmt.Sprintf("  (moved to %s)", destPath)
						fixed++
					}
				case reconcile.FindingInitiativeComplete:
					// All children done — complete + archive the initiative.
					// completeAndArchive flips status to completed (stamping
					// completed_at in the same write) and moves it to specs/.
					if _, err := completeAndArchive(f.Spec.Path, heroDir, true); err != nil {
						suffix = fmt.Sprintf("  (auto-complete failed: %v)", err)
					} else {
						action = "✓"
						suffix = fmt.Sprintf("  (completed + archived to specs/%s/)", f.Spec.Slug)
						fixed++
					}
				case reconcile.FindingOrphanCompletedAt:
					// completed_at set while status != completed — clear the
					// orphaned timestamp so the invariant holds.
					if err := clearCompletedAt(f.Spec.Path); err != nil {
						suffix = fmt.Sprintf("  (repair failed: %v)", err)
					} else {
						action = "✓"
						suffix = "  (cleared orphaned completed_at)"
						fixed++
					}
				default:
					if err := updateFrontmatterStatus(f.Spec.Path, string(f.SuggestedStatus)); err != nil {
						suffix = fmt.Sprintf("  (auto-fix failed: %v)", err)
					} else {
						action = "✓"
						suffix = "  (fixed)"
						fixed++
					}
				}
			}
			fmt.Printf("  %-30s  %s %s %s  %s%s\n",
				f.Spec.Slug,
				f.CurrentStatus,
				action,
				f.SuggestedStatus,
				f.Evidence,
				suffix,
			)
		}
		if fixed > 0 {
			fmt.Printf("\n  %d spec(s) auto-fixed. Run 'hero index' to update the search index.\n", fixed)
		} else if !checkReconcile {
			fmt.Printf("\n  Run 'hero check --reconcile' to auto-fix eligible items.\n")
		}
		fmt.Println()
	}

	// Missing created: (data-quality self-heal). A work spec whose CreatedAt is
	// an mtime guess rather than an authored `created:` gets stamped from its
	// first git commit under --reconcile; otherwise it's just reported. Kept
	// separate from status drift — it's data quality, not a git-status mismatch.
	missingCreated := workSpecsMissingCreated(heroDir)
	if len(missingCreated) == 0 {
		addRow("missing-created", "pass", "all work specs carry created:")
	} else {
		addRow("missing-created", "warn", fmt.Sprintf("%d spec(s) missing created:", len(missingCreated)))
		issues += len(missingCreated)
		stamped := 0
		fmt.Printf("Missing created: (%d spec(s)):\n", len(missingCreated))
		for _, s := range missingCreated {
			suffix := ""
			if checkReconcile {
				ts := createdDate(projectRoot, s.Path)
				if err := writeCreatedStamp(s.Path, ts); err != nil {
					suffix = fmt.Sprintf("  (stamp failed: %v)", err)
				} else {
					suffix = fmt.Sprintf("  ✓ stamped created: %s", ts.Format("2006-01-02"))
					stamped++
				}
			}
			fmt.Printf("  %-30s  no authored created:%s\n", s.Slug, suffix)
		}
		if stamped > 0 {
			fmt.Printf("\n  %d spec(s) stamped. Run 'hero index' to update the search index.\n", stamped)
		} else if !checkReconcile {
			fmt.Printf("\n  Run 'hero check --reconcile' to stamp them, or 'hero admin backfill-created'.\n")
		}
		fmt.Println()
	}

	// Knowledge lint
	if checkKnowledge {
		knowledgeIssues := runKnowledgeLint(heroDir, projectRoot)
		issues += knowledgeIssues
		if knowledgeIssues == 0 {
			addRow("knowledge", "pass", "knowledge base clean")
		} else {
			addRow("knowledge", "warn", fmt.Sprintf("%d knowledge-lint finding(s)", knowledgeIssues))
		}
	}

	// Pre-commit hook installation — without it, projected NEXT files
	// don't travel with commits and handoff state strands locally.
	// Only meaningful inside a git repo; non-git workspaces (test
	// fixtures, exported snapshots) skip the check entirely.
	// Spec: pre-commit-auto-stage-next.
	if _, err := resolveGitDir(projectRoot); err == nil {
		switch {
		case preCommitHasHeroHookButNoStaging(projectRoot):
			// A Hero pre-commit hook exists (generic `hero hook`
			// dispatch) but the handoff-file staging block is absent —
			// the projecting-but-not-staging gap. Distinct from "no hook
			// at all" so the user knows the precise invariant that's
			// broken. Spec: next-unconditional-commit-staging.
			issues++
			fmt.Println("Pre-commit hook present but handoff files are not staged:")
			fmt.Println("  A Hero pre-commit hook is installed, but it lacks the")
			fmt.Println("  handoff-file staging block — projected handoff files")
			fmt.Println("  (NEXT.md, SNAPSHOT.md, next/*.md) won't travel with commits.")
			fmt.Println("  Run 'hero next install-hooks' to wire staging.")
			fmt.Println()
			addRow("pre-commit-hook", "warn", "pre-commit hook present but handoff-file staging not wired; run 'hero next install-hooks'")
		case !preCommitHookInstalled(projectRoot):
			issues++
			fmt.Println("Pre-commit hook not installed:")
			fmt.Println("  Projected NEXT files won't travel with commits — handoff")
			fmt.Println("  state will strand on this machine. Run 'hero next install-hooks'")
			fmt.Println("  to fix.")
			fmt.Println()
			addRow("pre-commit-hook", "warn", "pre-commit hook not installed; run 'hero next install-hooks'")
		default:
			// Hook installed — check whether its managed block content
			// matches the current binary's hookScript() output. Drift
			// happens when the user upgrades hero but the installed
			// hook predates the new script. Spec: hero-upgrade-refreshes-hooks.
			if stale, err := preCommitHookStale(projectRoot); err == nil && stale {
				issues++
				fmt.Println("Pre-commit hook is stale:")
				fmt.Println("  Installed managed block doesn't match the current hero")
				fmt.Println("  binary's hook script. Run 'hero upgrade' (or")
				fmt.Println("  'hero next install-hooks') to refresh.")
				fmt.Println()
				addRow("pre-commit-hook", "warn", "pre-commit hook is stale; run 'hero next install-hooks'")
			} else {
				addRow("pre-commit-hook", "pass", "pre-commit hook installed and current")
			}
		}
	}

	// Kickoff section coverage — every non-completed work spec should
	// carry a `## Kickoff` section so it surfaces in `hero queue` and
	// gives the user a paste-ready cold-start prompt for that spec.
	// Spec: kickoff-prompts-queue.
	if missing, err := missingKickoffSpecs(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: kickoff coverage check failed: %v\n", err)
		addRow("kickoff-coverage", "warn", fmt.Sprintf("kickoff audit failed: %v", err))
	} else if len(missing) > 0 {
		issues += len(missing)
		fmt.Printf("Specs missing `## Kickoff` section (%d) — excluded from `hero queue`:\n", len(missing))
		const kickoffShowMax = 5
		for i, s := range missing {
			if i >= kickoffShowMax {
				fmt.Printf("  … and %d more.\n", len(missing)-kickoffShowMax)
				break
			}
			fmt.Printf("  %-30s  %-10s  %s\n", s.Slug, s.Status, s.Title)
		}
		fmt.Println("  Run /design or /deliver on each, or hand-edit per skills/kickoff-prompt.md.")
		fmt.Println()
		addRow("kickoff-coverage", "warn", fmt.Sprintf("%d spec(s) missing ## Kickoff section", len(missing)))
	} else {
		addRow("kickoff-coverage", "pass", "all work specs carry ## Kickoff sections")
	}

	// Initiative Goal-opener coverage — ADVISORY (does not bump issues). An
	// open initiative without a `## Goal` run-opener still surfaces in
	// `hero queue`, but can't be armed with `/drive` until it has one.
	// Spec: initiative-goal-section.
	if missing, err := missingGoalInitiatives(heroDir); err != nil {
		addRow("initiative-goal-coverage", "warn", fmt.Sprintf("initiative goal audit failed: %v", err))
	} else if len(missing) > 0 {
		fmt.Printf("Initiatives without a `## Goal` run-opener (%d) — can't `/drive` until added:\n", len(missing))
		for _, s := range missing {
			fmt.Printf("  %-30s  %-10s  %s\n", s.Slug, s.Status, s.Title)
		}
		fmt.Println()
		addRow("initiative-goal-coverage", "warn", fmt.Sprintf("%d initiative(s) missing ## Goal run-opener (advisory)", len(missing)))
	} else {
		addRow("initiative-goal-coverage", "pass", "all open initiatives carry a ## Goal run-opener")
	}

	// Wikilink relation intent — `[[slug]]` in a spec body reads like a
	// relationship but creates no graph edge (wikilinks are searchable
	// text only). Nudge toward the frontmatter that does form edges.
	if hits, err := specsWithWikilinks(heroDir); err == nil && len(hits) > 0 {
		issues += len(hits)
		fmt.Printf("Specs using `[[wikilinks]]` that create no graph edges (%d):\n", len(hits))
		const wikilinkShowMax = 5
		for i, h := range hits {
			if i >= wikilinkShowMax {
				fmt.Printf("  … and %d more.\n", len(hits)-wikilinkShowMax)
				break
			}
			fmt.Printf("  %-30s  %s\n", h.Slug, strings.Join(h.Links, ", "))
		}
		fmt.Println("  Wikilinks are searchable text only. For relationships use frontmatter:")
		fmt.Println("  `parent: <slug>`, `depends-on: [<slug>]`, or a relations: block.")
		fmt.Println()
		addRow("wikilink-edges", "warn", fmt.Sprintf("%d spec(s) use [[wikilinks]] that create no graph edges", len(hits)))
	} else {
		addRow("wikilink-edges", "pass", "no edge-intent wikilinks in spec bodies")
	}

	// Status truthfulness — one-line summary plus an issue count
	// bump for any spec lying about being completed. Run `hero check
	// status` for the full breakdown.
	if line, lyingPartial, err := statusTruthfulnessSummary(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: status truthfulness audit failed: %v\n", err)
		addRow("status-truthfulness", "warn", fmt.Sprintf("audit failed: %v", err))
	} else if line != "" {
		fmt.Println(line)
		if lyingPartial > 0 {
			fmt.Println("  Run 'hero check status' for the full breakdown.")
			issues += lyingPartial
			addRow("status-truthfulness", "fail", fmt.Sprintf("%d spec(s) lying or partial; run 'hero check status'", lyingPartial))
		} else {
			addRow("status-truthfulness", "pass", "completed specs match git evidence")
		}
		fmt.Println()
	}

	// Satellite drift — dry-run repair to surface findings.
	satIssues := reportSatelliteDrift(projectRoot, heroDir)
	if satIssues > 0 {
		issues += satIssues
		addRow("satellite-drift", "warn", fmt.Sprintf("%d satellite drift finding(s); run 'hero install --repair'", satIssues))
	} else {
		addRow("satellite-drift", "pass", "no satellite drift")
	}

	// Legacy install-layout drift — dangling harness-dir symlinks from
	// the P2→render-direct migration and stranded canonical mirror dirs.
	// A non-empty result means harness sessions in this project are
	// silently failing to load Hero skills/agents.
	driftFindings := install.DetectLegacyDrift(projectRoot)
	if len(driftFindings) > 0 {
		issues += len(driftFindings)
		fmt.Printf("Legacy install drift (%d):\n", len(driftFindings))
		for _, f := range driftFindings {
			switch f.Kind {
			case "broken_symlink":
				fmt.Printf("  [broken_symlink]       %s → %s (target missing)\n", f.Path, f.Target)
			case "legacy_canonical_dir":
				fmt.Printf("  [legacy_canonical_dir] %s\n", f.Path)
			}
		}
		fmt.Println("  Run 'hero upgrade' (or 'hero install <mode> .') to clean up.")
		fmt.Println()
		addRow("install-drift", "fail",
			fmt.Sprintf("%d legacy install artifact(s); run 'hero upgrade'", len(driftFindings)))
	} else {
		addRow("install-drift", "pass", "no legacy install drift")
	}

	// Orphaned root instruction files — INFORMATIONAL, not a failure. A
	// CLAUDE.md/AGENTS.md present for a target not recorded in
	// install-state.json `targets` (e.g. a Model-B phantom AGENTS.md from a
	// Claude-only install). Surface a heads-up with the opt-in prune hint;
	// do not count it as an issue.
	orphanFiles := detectOrphanInstructionFiles(projectRoot)
	if len(orphanFiles) > 0 {
		for _, o := range orphanFiles {
			fmt.Printf("Note: %s present but no %s target recorded in install-state; run 'hero install --prune-orphaned-instruction-files' to remove, or ignore to keep.\n", o.file, o.role)
		}
		fmt.Println()
		addRow("orphan-instruction-files", "warn",
			fmt.Sprintf("%d orphaned instruction file(s); informational — use 'hero install --prune-orphaned-instruction-files' to remove", len(orphanFiles)))
	} else {
		addRow("orphan-instruction-files", "pass", "no orphaned instruction files")
	}

	// Install integrity — re-render the managed body install would produce
	// right now and compare it to the body on disk in each installed
	// target's native instruction file (CLAUDE.md / AGENTS.md). Damaged
	// (region or sections gone) means agents in that harness run cold →
	// fail; stale (body drift) → warn. check only reports; `hero install`
	// / `hero upgrade` are the repair path. The Options here mirror the
	// install command's construction (domain from hero.json, domain pack
	// overlaid on core) so check resolves the same pack-body chain link
	// install did.
	reportInstallIntegrity(projectRoot, cfg.Domain, addRow, &issues)

	// Size drift — rate-limited. Per spec Implementation Notes, dump
	// at most two summary lines (one for leaf, one for container)
	// with counts and a hint pointing at `hero size --check` for the
	// per-spec breakdown. Avoids drowning the health summary in
	// dozens of per-spec rows.
	sizeLeaf, sizeContainer := reportSizeDriftSummary(heroDir)
	if sizeLeaf > 0 || sizeContainer > 0 {
		if sizeLeaf > 0 {
			fmt.Printf("Size drift (leaf): %d spec(s) with declared size out of sync with computed bucket. Run 'hero size --check' for detail.\n",
				sizeLeaf)
			addRow("size-drift-leaf", "warn",
				fmt.Sprintf("%d spec(s) with leaf size drift; run 'hero size --check'", sizeLeaf))
		} else {
			addRow("size-drift-leaf", "pass", "no leaf size drift")
		}
		if sizeContainer > 0 {
			fmt.Printf("Size drift (container): %d initiative(s) with declared size below child rollup. Run 'hero size --check' for detail.\n",
				sizeContainer)
			addRow("size-drift-container", "warn",
				fmt.Sprintf("%d container(s) with size drift; run 'hero size --check'", sizeContainer))
		} else {
			addRow("size-drift-container", "pass", "no container size drift")
		}
		issues += sizeLeaf + sizeContainer
		fmt.Println()
	} else {
		addRow("size-drift-leaf", "pass", "no leaf size drift")
		addRow("size-drift-container", "pass", "no container size drift")
	}

	// Snapshot containment + override health.
	snapIssues := reportSnapshotHealth(heroDir)
	if snapIssues > 0 {
		issues += snapIssues
		addRow("snapshot-health", "warn", fmt.Sprintf("%d snapshot containment issue(s)", snapIssues))
	} else {
		addRow("snapshot-health", "pass", "snapshot archives healthy")
	}

	// Severity-aware summary: surface failing vs advisory categories so a
	// scaffold-heavy but healthy workspace (all warnings) doesn't read as
	// broken. The flat item count is retained as detail.
	if issues == 0 {
		fmt.Println("No issues found.")
	} else {
		var fails, warns []string
		for _, r := range jsonRows {
			switch r.Status {
			case "fail":
				fails = append(fails, r.Name)
			case "warn":
				warns = append(warns, r.Name)
			}
		}
		if len(fails) == 0 {
			fmt.Printf("No failures — %d advisory check(s) with findings (%s), %d item(s). Advisories are non-blocking.\n",
				len(warns), strings.Join(warns, ", "), issues)
		} else {
			fmt.Printf("%d failing check(s): %s. Plus %d advisory check(s), %d item(s) total. Fix failures first.\n",
				len(fails), strings.Join(fails, ", "), len(warns), issues)
		}
	}

	if checkJSON {
		if err := writeHealthJSON(heroDir, jsonRows); err != nil {
			// Don't fail the command on JSON write errors — the human
			// output is already printed and is the source of truth.
			fmt.Fprintf(os.Stderr, "Warning: write health.json: %v\n", err)
		}
	}

	return nil
}

// reportSnapshotHealth validates the project-snapshot subsystem:
//   - .hero/surfaces.yaml parses (absence is not an error).
//   - .hero/snapshots/ entries carry historical/not_current
//     frontmatter and the historical banner.
//   - .hero/snapshots/ is excluded from the default search index
//     (out of scope here — the index config itself enforces it).
//
// Returns the number of issues found.
func reportSnapshotHealth(heroDir string) int {
	issues := 0

	// Surface override file: parse if present.
	if _, err := snapshot.LoadOverride(heroDir); err != nil {
		issues++
		fmt.Printf("Snapshot override (surfaces.yaml) parse error: %v\n", err)
		fmt.Println()
	}

	// Archive frontmatter + banner validation.
	archives, err := snapshot.List(heroDir)
	if err != nil {
		// Reading the dir failed; not a malformation by itself.
		return issues
	}
	if len(archives) == 0 {
		return issues
	}
	bad := 0
	for _, a := range archives {
		if !a.Historical || !a.NotCurrent {
			issues++
			bad++
			fmt.Printf("Snapshot archive %s missing isolation flags (historical/not_current)\n",
				a.Date)
		}
	}
	if bad > 0 {
		fmt.Println("  Run 'hero snapshot detect --explain' for inference state; archive files")
		fmt.Println("  missing flags can be regenerated by deleting and re-running")
		fmt.Println("  'hero snapshot archive'.")
		fmt.Println()
	}
	return issues
}

// reportSatelliteDrift runs the satellite reconciler in dry-run mode
// and prints any findings to stdout. Returns the count of findings,
// which is added to the overall issue count.
func reportSatelliteDrift(projectRoot, heroDir string) int {
	res, err := runSatelliteDryRun(heroDir, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: satellite check failed: %v\n", err)
		return 0
	}
	if res == nil || len(res.Findings) == 0 {
		return 0
	}
	fmt.Printf("Satellite drift (%d):\n", len(res.Findings))
	fmt.Print(res.FormatFindings())
	fmt.Println("  Run 'hero install --repair' to fix automatically-repairable issues.")
	fmt.Println()
	return len(res.Findings)
}

// missingKickoffSpecs returns work specs (feature/bug) in an open
// status that lack a `## Kickoff` section. Knowledge specs and
// closed work specs are skipped — kickoff is only meaningful for
// pickup-able work.
var wikilinkRe = regexp.MustCompile(`\[\[([^\]\n]+)\]\]`)

type wikilinkHit struct {
	Slug  string
	Links []string
}

// specsWithWikilinks returns work specs whose body contains `[[...]]`
// wikilinks. These read like relationships but create no graph edges,
// so hero check nudges the author toward relation frontmatter.
func specsWithWikilinks(heroDir string) ([]wikilinkHit, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	var out []wikilinkHit
	for _, s := range specs {
		if !s.IsWorkSpec() {
			continue
		}
		matches := wikilinkRe.FindAllStringSubmatch(s.RawContent, -1)
		if len(matches) == 0 {
			continue
		}
		seen := map[string]bool{}
		var links []string
		for _, m := range matches {
			t := strings.TrimSpace(m[1])
			if t != "" && !seen[t] {
				seen[t] = true
				links = append(links, t)
			}
		}
		out = append(out, wikilinkHit{Slug: s.Slug, Links: links})
	}
	return out, nil
}

func missingKickoffSpecs(heroDir string) ([]*spec.Spec, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	var out []*spec.Spec
	for _, s := range specs {
		if !s.IsWorkSpec() {
			continue
		}
		switch s.Status {
		case spec.StatusCompleted, spec.StatusSuperseded:
			continue
		}
		if strings.TrimSpace(s.Kickoff()) == "" {
			out = append(out, s)
		}
	}
	return out, nil
}

// missingGoalInitiatives returns initiatives in an open status whose
// `## Goal` run-opener is empty — they surface in `hero queue` without a
// paste-ready `/drive` opener. Advisory only: an initiative is still valid
// without one. Spec: initiative-goal-section.
func missingGoalInitiatives(heroDir string) ([]*spec.Spec, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	var out []*spec.Spec
	for _, s := range specs {
		if s.Type != spec.TypeInitiative {
			continue
		}
		switch s.Status {
		case spec.StatusCompleted, spec.StatusSuperseded:
			continue
		}
		if strings.TrimSpace(s.GoalSection()) == "" {
			out = append(out, s)
		}
	}
	return out, nil
}

// statusTruthfulnessSummary returns the one-line summary plus the
// count of lying+partial specs (used to bump the overall issue
// counter). Empty line and zero count when no completed specs exist.
func statusTruthfulnessSummary() (string, int, error) {
	report, err := buildStatusReport()
	if err != nil {
		return "", 0, err
	}
	return statusSummaryLine(report), report.Lying + report.Partial, nil
}

func runKnowledgeLint(heroDir, projectRoot string) int {
	issues := 0
	knowledgeDir := filepath.Join(heroDir, "knowledge")

	// Find all knowledge specs
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return 0
	}

	var knowledgeSpecs []*spec.Spec
	for _, s := range specs {
		if s.IsKnowledge() {
			knowledgeSpecs = append(knowledgeSpecs, s)
		}
	}

	if len(knowledgeSpecs) == 0 {
		return 0
	}

	fmt.Println("Knowledge lint:")

	// 1. Check for pending enrichment (ingested but not summarized)
	pendingCount := 0
	for _, s := range knowledgeSpecs {
		if strings.Contains(s.RawContent, "Pending enrichment") || strings.Contains(s.RawContent, "To be filled by agent") {
			pendingCount++
			if pendingCount <= 5 {
				fmt.Printf("  ⚠ %s — pending enrichment (ingested but not summarized)\n", s.Slug)
			}
		}
	}
	if pendingCount > 5 {
		fmt.Printf("  ... and %d more pending enrichment\n", pendingCount-5)
	}
	issues += pendingCount

	// 2. Check for stale knowledge (not modified in 90+ days)
	staleCount := 0
	cutoff := time.Now().AddDate(0, 0, -90)
	for _, s := range knowledgeSpecs {
		if s.ModifiedAt.Before(cutoff) {
			staleCount++
			if staleCount <= 5 {
				age := int(time.Since(s.ModifiedAt).Hours() / 24)
				fmt.Printf("  ⚠ %s — stale (%d days since last update)\n", s.Slug, age)
			}
		}
	}
	if staleCount > 5 {
		fmt.Printf("  ... and %d more stale entries\n", staleCount-5)
	}
	issues += staleCount

	// 3. Check for file references that no longer exist
	brokenRefs := 0
	for _, s := range knowledgeSpecs {
		for _, f := range s.FilesTouched {
			abs := filepath.Join(projectRoot, f)
			if _, err := os.Stat(abs); os.IsNotExist(err) {
				brokenRefs++
				if brokenRefs <= 5 {
					fmt.Printf("  ⚠ %s — references non-existent file %s\n", s.Slug, f)
				}
			}
		}
	}
	if brokenRefs > 5 {
		fmt.Printf("  ... and %d more broken references\n", brokenRefs-5)
	}
	issues += brokenRefs

	// 3b. Check explainer provenance (synthesized_from + last_synthesized).
	// An explainer claims to describe current reality, so it must name the
	// specs it was synthesized from and when — else readers can't judge
	// staleness. See feature-knowledge-synthesis.
	provCount := 0
	for _, s := range knowledgeSpecs {
		if s.Type != spec.TypeExplainer {
			continue
		}
		var missing []string
		if len(s.SynthesizedFrom) == 0 {
			missing = append(missing, "synthesized_from")
		}
		if s.LastSynthesized == "" {
			missing = append(missing, "last_synthesized")
		}
		if len(missing) > 0 {
			provCount++
			if provCount <= 5 {
				fmt.Printf("  ⚠ %s — explainer missing provenance: %s\n", s.Slug, strings.Join(missing, ", "))
			}
		}
	}
	if provCount > 5 {
		fmt.Printf("  ... and %d more explainers missing provenance\n", provCount-5)
	}
	issues += provCount

	// 4. Check for orphan raw files (raw/ entries with no corresponding knowledge entry)
	rawDir := filepath.Join(knowledgeDir, "raw")
	if entries, err := os.ReadDir(rawDir); err == nil {
		orphanCount := 0
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			slug := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			found := false
			for _, s := range knowledgeSpecs {
				if s.Slug == slug {
					found = true
					break
				}
			}
			if !found {
				orphanCount++
				if orphanCount <= 3 {
					fmt.Printf("  ⚠ raw/%s — orphan (no knowledge entry references it)\n", entry.Name())
				}
			}
		}
		if orphanCount > 3 {
			fmt.Printf("  ... and %d more orphan raw files\n", orphanCount-3)
		}
		issues += orphanCount
	}

	if issues == 0 {
		fmt.Println("  ✓ Knowledge base is clean")
	}
	fmt.Println()

	return issues
}
