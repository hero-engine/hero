package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	satellitesYesAll        bool
	satellitesNoAll         bool
	satellitesRepair        bool
	satellitesMigrateNested bool
	satellitesApply         bool
	satellitesForce         bool
	satellitesForceResume   bool
)

var installSatellitesCmd = &cobra.Command{
	Use:   "satellites",
	Short: "Manage satellite installs in monorepo subprojects",
	Long: `Materialize, repair, and reconcile Hero satellite installs in
subproject folders of a monorepo.

Satellites are thin symlink trees in subproject folders that point at
the workspace root, so chats opened inside a subproject pick up the
same agents, commands, and skills as chats opened at the root. There
is still exactly one .hero/ workspace at the root.

Run with no flags to walk through detected candidate subprojects
one-by-one. Use --repair to reconcile existing satellites against
the current state of root and subprojects.json. Use --yes to accept
all detected candidates without prompting (intended for unattended
re-runs after a teammate has updated subprojects.json in git).`,
	RunE: runInstallSatellites,
}

func init() {
	installSatellitesCmd.Flags().BoolVar(&satellitesYesAll, "yes", false, "accept all candidates and reconciliation prompts without asking")
	installSatellitesCmd.Flags().BoolVar(&satellitesNoAll, "no", false, "decline all prompts (audit-only mode for unattended runs)")
	installSatellitesCmd.Flags().BoolVar(&satellitesRepair, "repair", false, "verify and repair existing satellite symlinks/markers without re-prompting candidates")
	installSatellitesCmd.Flags().BoolVar(&satellitesMigrateNested, "migrate-nested", false, "scan for legacy nested .hero/ workspaces under root and print a migration plan (read-only without --apply)")
	installSatellitesCmd.Flags().BoolVar(&satellitesApply, "apply", false, "with --migrate-nested: actually execute the migration (move files, append events, materialize satellite)")
	installSatellitesCmd.Flags().BoolVar(&satellitesForce, "force", false, "with --migrate-nested --apply: ignore dirty git state")
	installSatellitesCmd.Flags().BoolVar(&satellitesForceResume, "force-resume", false, "with --migrate-nested --apply: complete a partially-applied migration")

	installCmd.AddCommand(installSatellitesCmd)
}

func runInstallSatellites(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	ws, err := workspace.Locate(cwd)
	if err != nil {
		return fmt.Errorf("locate workspace: %w", err)
	}

	binaryVersion := rootCmd.Version
	if binaryVersion == "" {
		binaryVersion = "dev"
	}

	if satellitesRepair {
		return runSatelliteRepair(ws, binaryVersion, false)
	}

	if satellitesMigrateNested {
		nested := install.FindNestedHeroDirs(ws.Root)
		if len(nested) == 0 {
			fmt.Println("No nested .hero/ workspaces found under root.")
			return nil
		}

		// Plan-only mode (default).
		if !satellitesApply {
			fmt.Printf("Found %d nested .hero/ workspace(s):\n\n", len(nested))
			for _, n := range nested {
				plan, err := install.PlanMigration(ws.Root, n)
				if err != nil {
					fmt.Printf("  %s — error: %v\n", n, err)
					continue
				}
				fmt.Println(install.FormatMigrationPlan(plan))
			}
			fmt.Println("Re-run with --apply to execute the migration. Add --force to ignore dirty git state.")
			return nil
		}

		// Apply mode — print plans, prompt, then execute sequentially.
		fmt.Printf("Found %d nested .hero/ workspace(s) to migrate:\n\n", len(nested))
		for _, n := range nested {
			plan, err := install.PlanMigration(ws.Root, n)
			if err != nil {
				fmt.Printf("  %s — error: %v\n", n, err)
				continue
			}
			fmt.Println(install.FormatMigrationPlan(plan))
		}
		if !satellitesYesAll {
			// This migration moves files and deletes the nested .hero/, so it
			// must never run unconfirmed.
			//
			// The old guard was `!satellitesYesAll && isTerminal()`, i.e.
			// "confirm only when there is a terminal, otherwise just proceed".
			// That reads as an unattended-mode convenience, but combined with
			// the ModeCharDevice bug it produced two opposite outcomes for two
			// non-interactive invocations: `< /dev/null` was misclassified as a
			// terminal, so it prompted, read EOF, and aborted — while a pipe
			// was correctly classified and therefore silently performed the
			// migration with no confirmation at all.
			//
			// Fixing the predicate without restructuring the guard would have
			// made BOTH cases proceed, turning an accidental abort into an
			// unattended destructive migration. Requiring an explicit answer
			// instead keeps `< /dev/null` behaving exactly as before and makes
			// the piped case fail safe. --yes remains the unattended path,
			// which is what its help text already promises.
			proceed := false
			if prompt.IsInputTTY(cmd.InOrStdin()) {
				yes, err := prompt.Confirm(cmd.InOrStdin(), cmd.OutOrStdout(),
					"Apply these migrations? [y/N] ", false)
				if err != nil {
					return err
				}
				proceed = yes
			}
			if !proceed {
				fmt.Println("Aborted.")
				return nil
			}
		}
		for _, n := range nested {
			res, err := install.ApplyMigration(install.ApplyOptions{
				RootDir:     ws.Root,
				Version:     binaryVersion,
				Force:       satellitesForce,
				ForceResume: satellitesForceResume,
				DryRun:      false,
			}, n)
			fmt.Println(install.FormatApplyResult(res, false))
			if err != nil {
				fmt.Printf("Migration of %s halted: %v\n", n, err)
				return err
			}
		}
		// Re-index after the migration.
		fmt.Println("Re-indexing root workspace...")
		if _, err := index.Rebuild(ws.HeroDir); err != nil {
			fmt.Printf("Warning: index rebuild failed: %v\n", err)
		}
		fmt.Println("Migration complete. Commit the change as a single commit.")
		return nil
	}

	// Run repair first so we operate on a clean baseline.
	if err := runSatelliteRepair(ws, binaryVersion, true); err != nil {
		return err
	}

	subs, err := install.LoadSubprojects(ws.HeroDir)
	if err != nil {
		return err
	}

	// First: reconcile declared-but-not-materialized.
	if err := reconcileDeclared(ws, subs, binaryVersion, cmd.InOrStdin(), cmd.OutOrStdout()); err != nil {
		return err
	}

	// Then: walk new candidates that are neither declared nor excluded.
	candidates, err := install.DetectCandidates(ws.Root, subs, 4)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		fmt.Println("No new subproject candidates detected.")
		return nil
	}
	fmt.Printf("\nDetected %d candidate subproject(s):\n\n", len(candidates))
	return walkCandidates(ws, subs, candidates, binaryVersion, cmd.InOrStdin(), cmd.OutOrStdout())
}

func runSatelliteRepair(ws *workspace.Workspace, version string, quiet bool) error {
	res, err := install.Repair(install.RepairOptions{
		HeroDir: ws.HeroDir,
		RootDir: ws.Root,
		Version: version,
		DryRun:  false,
	})
	if err != nil {
		return err
	}
	if quiet && len(res.Repaired) == 0 {
		return nil
	}
	if len(res.Repaired) > 0 {
		fmt.Printf("Repaired %d satellite issue(s):\n", len(res.Repaired))
		for _, line := range res.Repaired {
			fmt.Printf("  - %s\n", line)
		}
	} else if !quiet {
		fmt.Println("No satellite drift detected.")
	}
	if !quiet && len(res.Findings) > 0 {
		fmt.Println()
		fmt.Println("Outstanding findings:")
		fmt.Print(res.FormatFindings())
	}
	return nil
}

func reconcileDeclared(ws *workspace.Workspace, subs *install.SubprojectsManifest, version string, in io.Reader, out io.Writer) error {
	local, err := install.LoadSatellitesLocal(ws.HeroDir)
	if err != nil {
		return err
	}

	missing := make([]install.Subproject, 0)
	for _, sp := range subs.Subprojects {
		if local.Find(sp.Path) == nil {
			missing = append(missing, sp)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if !satellitesYesAll && !satellitesNoAll && !prompt.IsInputTTY(in) {
		return fmt.Errorf("reconciling declared subprojects requires an attached terminal; pass --yes or --no")
	}
	fmt.Fprintf(out, "Found %d declared subproject(s) without local satellites:\n\n", len(missing))

	for _, sp := range missing {
		// Default Y for already-declared subprojects — the team already
		// decided by committing them to subprojects.json.
		materialize := !satellitesNoAll
		if !satellitesYesAll && !satellitesNoAll {
			yes, err := prompt.Confirm(in, out,
				fmt.Sprintf("  %-40s materialize satellite? [Y/n] ", sp.Path), true)
			if err != nil {
				return err
			}
			materialize = yes
		}
		if !materialize {
			fmt.Fprintf(out, "    skipped\n")
			continue
		}
		if err := materializeOne(ws, sp, version, out); err != nil {
			fmt.Fprintf(out, "    error: %v\n", err)
			continue
		}
	}
	return nil
}

func walkCandidates(ws *workspace.Workspace, subs *install.SubprojectsManifest, candidates []install.Candidate, version string, in io.Reader, out io.Writer) error {
	yesAll := satellitesYesAll
	skipAll := satellitesNoAll
	if !yesAll && !skipAll && !prompt.IsInputTTY(in) {
		return fmt.Errorf("walking subproject candidates requires an attached terminal; pass --yes or --no")
	}
	dirty := false

	defer func() {
		if dirty {
			if err := install.SaveSubprojects(ws.HeroDir, subs); err != nil {
				fmt.Fprintf(out, "warning: could not save subprojects.json: %v\n", err)
			} else {
				fmt.Fprintf(out, "\nUpdated %s — commit it to share with your team.\n", filepath.Join(workspace.HeroDir, install.SubprojectsFile))
			}
		}
	}()

	// skipUnder is the list of parent paths the user has chosen to
	// "exclude this whole subtree" via X. Inline filtering: when a
	// candidate falls under any prefix in this list, we skip it without
	// prompting (it was implicitly handled by the parent exclusion).
	var skipUnder []string

	for i := 0; i < len(candidates); i++ {
		c := candidates[i]
		if skipAll {
			break
		}
		if isUnderAny(c.Path, skipUnder) {
			continue
		}
		nestedHint := ""
		if c.HasNestedHero {
			nestedHint = " (legacy .hero/ present — likely standalone repo merged in)"
		}
		fmt.Fprintf(out, "  [%d/%d] %s\n        signals: %s%s\n",
			i+1, len(candidates), c.Path, c.ReasonStrings(), nestedHint)

		var raw string
		if yesAll {
			raw = "y"
		} else {
			// Not prompt.Confirm: this is a 7-way menu, not a yes/no, and
			// the answer is case-sensitive (X excludes a whole subtree, x
			// excludes only the leaf). Reading through prompt.Prompt removes
			// the bufio fork without flattening the menu into a boolean.
			raw, _ = prompt.Prompt(in, out, "        propose? [y/N/a/s/q/x/X/?] ")
		}

		// Capital X is distinct from lowercase x — handle case-sensitive
		// match before lowercasing for everything else.
		if raw == "X" || raw == "EXCLUDE" {
			parent := filepath.Dir(c.Path)
			if parent == "." || parent == "" || parent == "/" {
				// Top-level candidate has no meaningful parent — fall
				// back to leaf-exclude and tell the user why.
				subs.AddExcluded(c.Path)
				dirty = true
				fmt.Fprintf(out, "        top-level candidate — excluded leaf only (X = parent-exclude has no effect here)\n")
				continue
			}
			subs.AddExcluded(parent)
			dirty = true
			skipUnder = append(skipUnder, parent)
			// Count how many remaining queued candidates this just nuked.
			dropped := 0
			for j := i + 1; j < len(candidates); j++ {
				if isUnder(candidates[j].Path, parent) {
					dropped++
				}
			}
			fmt.Fprintf(out, "        excluded %s (skipped %d remaining candidate(s) under it)\n", parent, dropped)
			continue
		}

		decision := strings.ToLower(raw)
		switch decision {
		case "y", "yes":
			subs.AddSubproject(install.Subproject{Path: c.Path, Scope: c.Path})
			dirty = true
			sp := install.Subproject{Path: c.Path, Scope: c.Path}
			if err := materializeOne(ws, sp, version, out); err != nil {
				fmt.Fprintf(out, "        error: %v\n", err)
			}
		case "a", "all":
			yesAll = true
			subs.AddSubproject(install.Subproject{Path: c.Path, Scope: c.Path})
			dirty = true
			sp := install.Subproject{Path: c.Path, Scope: c.Path}
			if err := materializeOne(ws, sp, version, out); err != nil {
				fmt.Fprintf(out, "        error: %v\n", err)
			}
		case "x", "exclude":
			subs.AddExcluded(c.Path)
			dirty = true
			fmt.Fprintf(out, "        excluded permanently\n")
		case "s", "skip-all":
			skipAll = true
		case "q", "quit":
			return nil
		case "?", "help":
			fmt.Fprintln(out, helpText())
			// Re-prompt this same candidate.
			i--
			continue
		case "", "n", "no":
			fmt.Fprintf(out, "        skipped (will ask again next install)\n")
		default:
			fmt.Fprintf(out, "        unrecognized — skipping (use ? for help)\n")
		}
	}
	return nil
}

// isUnder reports whether childPath sits under (or equals) parentPath
// in forward-slash form. Both inputs should already be normalized.
func isUnder(childPath, parentPath string) bool {
	if childPath == parentPath {
		return true
	}
	return strings.HasPrefix(childPath, parentPath+"/")
}

// isUnderAny is the multi-parent variant.
func isUnderAny(childPath string, parents []string) bool {
	for _, p := range parents {
		if isUnder(childPath, p) {
			return true
		}
	}
	return false
}

func materializeOne(ws *workspace.Workspace, sp install.Subproject, version string, out io.Writer) error {
	satAbs := filepath.Join(ws.Root, filepath.FromSlash(sp.Path))
	if info, err := os.Stat(satAbs); err != nil || !info.IsDir() {
		return fmt.Errorf("not a directory: %s", sp.Path)
	}
	res, err := install.Materialize(install.SatelliteOptions{
		RootDir:      ws.Root,
		SatelliteDir: satAbs,
		Scope:        sp.Scope,
		Version:      version,
	})
	if err != nil {
		return err
	}
	suffix := ""
	if res.Degraded {
		suffix = " (degraded — symlinks unsupported)"
	}
	fmt.Fprintf(out, "        materialized: targets=%v%s\n", res.Targets, suffix)
	entry := install.SatelliteEntry{
		Path:     sp.Path,
		Targets:  targetsAsStrings(res.Targets),
		Degraded: res.Degraded,
	}
	return install.RecordSatellite(ws.HeroDir, entry)
}

func targetsAsStrings(ts []install.Target) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = string(t)
	}
	return out
}

func helpText() string {
	return `        options:
          y / yes        materialize satellite for this subproject (adds to subprojects.json)
          n / no         skip — ask again next time (default)
          a / all        yes to all remaining
          s / skip-all   skip everything else for this run
          x / exclude    permanently exclude this leaf (writes to excluded[] in subprojects.json)
          X / EXCLUDE    permanently exclude this folder's PARENT and skip all
                         remaining candidates under it (vendor/third-party tree shotgun)
          q / quit       stop prompting
          ? / help       show this help`
}
