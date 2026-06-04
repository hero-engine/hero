package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/hooks"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage Hero git hook integration",
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Hero git hooks into .git/hooks/",
	RunE:  runHooksInstall,
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove all Hero git hooks (tracker hooks, hero-next projection hooks, merge driver) from .git/hooks/ and .gitattributes.",
	RunE:  runHooksUninstall,
}

var hooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Hero git hook installation status",
	RunE:  runHooksStatus,
}

func init() {
	hooksCmd.AddCommand(hooksInstallCmd, hooksUninstallCmd, hooksStatusCmd)
}

// findGitDir walks up from the project root looking for a .git/ directory.
// Returns the .git directory path or an error if not found.
func findGitDir() (string, error) {
	projectRoot := findProjectRoot()

	dir := projectRoot
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return gitDir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no .git directory found (not a git repository)")
}

func runHooksInstall(cmd *cobra.Command, args []string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	// Capture status before installation to know what changed
	before, err := hooks.Status(gitDir)
	if err != nil {
		return fmt.Errorf("checking hook status: %w", err)
	}

	beforeMap := make(map[string]bool)
	for _, h := range before {
		beforeMap[h.Name] = h.HasHero
	}

	if err := hooks.Install(gitDir); err != nil {
		return fmt.Errorf("installing hooks: %w", err)
	}

	// Also wire the hero-next staging pre-commit block so this generic
	// install path can never produce hooks-without-staging. Without
	// this, a repo set up via `hero hooks install` (not `hero init` /
	// `hero next install-hooks`) would project handoff files on every
	// commit but never stage them — stranding handoff state locally.
	// Idempotent + marker-bounded, so it coexists with the generic
	// `# Hero git hook` pre-commit block. Spec: next-unconditional-commit-staging.
	projectRoot := findProjectRoot()
	if err := installNextHooksQuiet(projectRoot); err != nil {
		return fmt.Errorf("installing handoff-staging hook: %w", err)
	}

	after, err := hooks.Status(gitDir)
	if err != nil {
		return fmt.Errorf("checking hook status after install: %w", err)
	}

	installed := 0
	skipped := 0
	for _, h := range after {
		if !h.HasHero {
			continue
		}
		if beforeMap[h.Name] {
			fmt.Printf("  skipped  %s (already installed)\n", h.Name)
			skipped++
		} else {
			fmt.Printf("  installed  %s\n", h.Name)
			installed++
		}
	}

	if installed == 0 && skipped > 0 {
		fmt.Println("Hero hooks already installed.")
	} else {
		fmt.Printf("Installed %d hook(s) into %s\n", installed, filepath.Join(gitDir, "hooks"))
	}

	return nil
}

func runHooksUninstall(cmd *cobra.Command, args []string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	before, err := hooks.Status(gitDir)
	if err != nil {
		return fmt.Errorf("checking hook status: %w", err)
	}

	if err := hooks.Uninstall(gitDir); err != nil {
		return fmt.Errorf("uninstalling hooks: %w", err)
	}

	removed := 0
	for _, h := range before {
		if h.HasHero {
			fmt.Printf("  removed  %s\n", h.Name)
			removed++
		}
	}

	// Also remove the hero-next projection-hook block, .gitattributes
	// entries, and merge-driver registration. Idempotent — a no-op
	// when hero next install-hooks was never run.
	projectRoot := findProjectRoot()
	nextRemoved, nerr := uninstallNextHooks(projectRoot)
	if nerr != nil {
		return fmt.Errorf("uninstalling next hooks: %w", nerr)
	}
	for _, p := range nextRemoved {
		fmt.Printf("  removed  %s (hero-next)\n", p)
	}

	total := removed + len(nextRemoved)
	if total == 0 {
		fmt.Println("No Hero hooks were installed.")
	} else {
		fmt.Printf("Removed Hero from %d hook(s).\n", total)
	}

	return nil
}

func runHooksStatus(cmd *cobra.Command, args []string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	statuses, err := hooks.Status(gitDir)
	if err != nil {
		return fmt.Errorf("checking hook status: %w", err)
	}

	fmt.Printf("%-22s  %-10s  %-14s\n", "Hook", "Installed", "Hero-managed")
	fmt.Printf("%-22s  %-10s  %-14s\n", "----", "---------", "------------")

	for _, h := range statuses {
		_, statErr := os.Stat(h.Path)
		installed := "no"
		if statErr == nil {
			installed = "yes"
		}
		heroManaged := "no"
		if h.HasHero {
			heroManaged = "yes"
		}
		fmt.Printf("%-22s  %-10s  %-14s\n", h.Name, installed, heroManaged)
	}

	// Report hero-next projection-hook state alongside the general
	// installer's per-hook table. Projected files now use git's built-in
	// merge=union (declared in .gitattributes, no per-clone driver), so
	// the only .git/config state worth surfacing is a legacy
	// merge.hero-next.* stanza lingering from an older install — and only
	// when one is actually present, since `hero hooks uninstall` clears it.
	projectRoot := findProjectRoot()
	preCommitState := "no"
	if preCommitHookInstalled(projectRoot) {
		preCommitState = "yes"
	}
	fmt.Printf("\n  hero next pre-commit block: %s\n", preCommitState)
	if nextMergeDriverRegistered(projectRoot) {
		fmt.Printf("  legacy hero-next merge driver in .git/config: present (run 'hero hooks uninstall' to clear)\n")
	}

	return nil
}
