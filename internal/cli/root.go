package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/spectypes"
	"github.com/hero-engine/hero/internal/version"
	"github.com/hero-engine/hero/internal/workspace"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hero",
	Short: "Spec-driven AI engineering workflow",
	Long:  "Hero adds a spec-driven 2-phase workflow to AI coding tools: design before you build, diagnose before you fix.",
}

// SetVersion sets the version string displayed by --version.
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&globalSmoke, "smoke", false, "run smoke verification for this command and exit")

	// Set PersistentPreRun here (not in the var declaration) to avoid an
	// initialization cycle — the closure captures rootCmd.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Skip version check for init, install, upgrade, mcp, and version itself
		name := cmd.Name()
		if name == "init" || name == "install" || name == "trust" || name == "upgrade" || name == "mcp" || name == "version" || name == "help" {
			return
		}

		projectRoot := findProjectRoot()
		heroDir := filepath.Join(projectRoot, ".hero")
		if _, err := os.Stat(heroDir); os.IsNotExist(err) {
			return // no workspace, nothing to check
		}

		// Migrate workspaces predating peer_id: mint one on first
		// invocation when missing. Best-effort — errors here don't
		// block the user's command.
		if _, minted, err := peering.EnsurePeerID(projectRoot, "migration"); err == nil && minted {
			fmt.Fprintf(os.Stderr, "hero: minted peer_id for cross-repo peering (recorded in events.log)\n")
		}

		binaryVersion := rootCmd.Version
		if msg := version.Mismatch(heroDir, binaryVersion); msg != "" {
			fmt.Fprintf(os.Stderr, "hero: %s\n", msg)
		}

		// Refresh .hero/cache/spec-types.json so hero-code and other
		// cross-language consumers always see the current registry.
		// Best-effort: failures here don't block the user's command.
		exportSpecTypesCache(projectRoot)
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(trustCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(whyCmd)
	rootCmd.AddCommand(blockedCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(queueCmd)
	rootCmd.AddCommand(relevantCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(noteCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(doCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(sprintCmd) // sprint umbrella: load, estimate, status, retro, report, velocity
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(hooksCmd)
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(skillCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(docsCmd)
	rootCmd.AddCommand(pipelineCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(jobRunCmd)
	rootCmd.AddCommand(specCmd)
	rootCmd.AddCommand(domainCmd)
	// Top-level aliases for the core-loop verbs (canonical paths
	// live under `hero spec`, but these stay one keystroke away).
	rootCmd.AddCommand(deliverAliasCmd)
	rootCmd.AddCommand(diagnoseAliasCmd)
	rootCmd.AddCommand(designAliasCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(driftCmd)
	rootCmd.AddCommand(recapCmd)
	rootCmd.AddCommand(feedCmd)
	rootCmd.AddCommand(impactCmd)
	rootCmd.AddCommand(coverageCmd)
	rootCmd.AddCommand(templatesCmd)
	rootCmd.AddCommand(ciCmd)
	rootCmd.AddCommand(suggestCmd)
	rootCmd.AddCommand(smokeCmd)

	// Subsystem umbrellas (each one collapses several former
	// top-level commands into a single grouped surface):
	rootCmd.AddCommand(publishCmd) // publish wiki / publish pages
	rootCmd.AddCommand(agentCmd)   // agent run / jobs / automate / approve / events
	rootCmd.AddCommand(adminCmd)   // admin team / users / domain / repos
	rootCmd.AddCommand(tripwireCmd)
	rootCmd.AddCommand(anchorCmd)
	rootCmd.AddCommand(handoffCmd) // cross-repo async handoff
	rootCmd.AddCommand(peerCmd)    // cross-repo peer manifest / list / show / call
	rootCmd.AddCommand(snapshotCmd)    // project-shape rollup + archive trail
	rootCmd.AddCommand(embeddingsCmd)  // embeddings status / rebuild

	// Wrap every direct subcommand that has a RunE with the smoke interceptor.
	// Must come after all AddCommand calls so the full command set is present.
	// Commands register real smoke logic via RegisterSmoke in their own init().
	for _, cmd := range rootCmd.Commands() {
		if cmd.RunE != nil {
			cmd.RunE = smokeInterceptor(cmd.RunE)
		}
	}
}

// exportSpecTypesCache regenerates .hero/cache/spec-types.json from the
// embedded core + active domain spec-type records. The cache is the
// cross-language contract that hero-code (Rust dashboard) and any other
// downstream tooling consume. Skips silently on any error — the cache is
// a derived artifact, not a precondition for hero CLI to function.
func exportSpecTypesCache(projectRoot string) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return
	}
	reg, err := spectypes.Load(cfg.Domain)
	if err != nil {
		return
	}
	_ = spectypes.ExportTo(reg, projectRoot)
}

// findGitRoot finds the git root for the current project without walking into
// unrelated parent repos. Used by `init` to avoid matching a .hero/ or .git/
// from a different project higher up the tree.
//
// Strategy: walk up looking for .git, but stop if we'd leave the current
// project (heuristic: don't cross more than 3 levels without finding .git,
// and never go above $HOME). If nothing found, use cwd.
func findGitRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	home, _ := os.UserHomeDir()
	dir := cwd
	for i := 0; i < 5; i++ {
		// Don't use home directory as a project root
		if home != "" && dir == home {
			break
		}
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return cwd
}

// findProjectRoot walks up from the current directory looking for a Hero
// workspace. Resolution order matches internal/workspace.Locate: a
// .hero-satellite marker (resolves to the linked root), then .hero/, then
// .git/ as a last-resort fallback. Returns the cwd if nothing is found.
//
// Most new code should call workspace.LocateFromCWD directly to also get
// scope and satellite metadata; this thin wrapper is preserved for the
// existing callers that just need a string.
func findProjectRoot() string {
	if ws, err := workspace.LocateFromCWD(); err == nil {
		return ws.Root
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	cwd, _ := os.Getwd()
	return cwd
}
