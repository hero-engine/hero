package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/version"
	"github.com/spf13/cobra"
)

var (
	upgradeDryRun    bool
	upgradeForce     bool
	upgradeNoHooks   bool
	upgradeTargets   []string
	upgradeContentFS fs.FS // overridable for tests; nil means use hero.ContentFS()
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the workspace to match the current hero binary",
	Long: `Updates agents, commands, and skills to match the installed hero binary version.

Files that you've customized are preserved — only unmodified files are updated.
Use --force to overwrite customized files.

When multiple AI tools are installed (e.g. both .opencode/ and .claude/),
upgrade walks every detected target by default. Use --target to narrow.

Examples:

  hero upgrade                       Upgrade every detected target
  hero upgrade --target claude       Upgrade only the claude target
  hero upgrade --target claude --target opencode    Upgrade two targets
  hero upgrade --dry-run             Show what would change without modifying anything
  hero upgrade --force               Overwrite even customized files`,
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "show what would change without modifying files")
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false, "overwrite customized files")
	upgradeCmd.Flags().BoolVar(&upgradeNoHooks, "no-hooks", false, "skip refreshing the installed pre-commit hook (mirrors `hero scan --no-hooks`)")
	upgradeCmd.Flags().StringSliceVar(&upgradeTargets, "target", nil, "narrow to one or more targets (claude, opencode, cursor); default: every detected target")
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := filepath.Join(projectRoot, cfg.Folder)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	binaryVersion := rootCmd.Version
	if binaryVersion == "" {
		binaryVersion = "dev"
	}

	// Read current version info
	info, _ := version.Read(heroDir)
	fromVersion := "unknown"
	if info != nil && info.HeroVersion != "" {
		fromVersion = info.HeroVersion
	}

	if fromVersion == binaryVersion && fromVersion != "dev" {
		fmt.Printf("Workspace is already at v%s — nothing to upgrade.\n", binaryVersion)
		return nil
	}

	// Reject downgrade attempts — binary is older than workspace.
	if binaryVersion != "dev" && fromVersion != "unknown" && fromVersion != "dev" {
		cmp := version.CompareVersions(binaryVersion, fromVersion)
		if cmp < 0 {
			return fmt.Errorf("cannot downgrade workspace from v%s to v%s — use a newer hero binary or run 'hero init' to create a fresh workspace", fromVersion, binaryVersion)
		}
	}

	fmt.Printf("Upgrading workspace from v%s to v%s\n\n", fromVersion, binaryVersion)

	// Use the embedded content filesystem (overridable for tests)
	contentFS := upgradeContentFS
	if contentFS == nil {
		contentFS = hero.ContentFS()
	}

	// Resolve which targets to upgrade. Filesystem probe finds every
	// installed tool; --target narrows the set. Default = upgrade all.
	targets, err := resolveUpgradeTargets(projectRoot, info, upgradeTargets)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Println("No installed target detected. Upgrading workspace version stamp only.")
		if !upgradeDryRun {
			if err := version.StampUpgrade(heroDir, fromVersion, binaryVersion, 0, 0); err != nil {
				return fmt.Errorf("writing version stamp: %w", err)
			}
		}
		fmt.Printf("\nWorkspace version updated to v%s\n", binaryVersion)
		if !upgradeNoHooks {
			if err := refreshHooksIfPresent(projectRoot, upgradeDryRun, cmd.OutOrStdout()); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: hook refresh failed: %v\n", err)
			}
		}
		return nil
	}

	if len(targets) == 1 {
		fmt.Printf("Detected target: %s\n\n", targets[0])
	} else {
		labels := make([]string, len(targets))
		for i, t := range targets {
			labels[i] = string(t)
		}
		fmt.Printf("Detected targets: %s\n\n", strings.Join(labels, ", "))
	}

	totalUpdated := 0
	totalSkipped := 0
	allChecksums := make(map[string]string)

	for i, target := range targets {
		if len(targets) > 1 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("--- %s ---\n", target)
		}
		updated, skipped, checksums := upgradeTarget(projectRoot, target, contentFS, info)
		totalUpdated += updated
		totalSkipped += skipped
		for k, v := range checksums {
			allChecksums[k] = v
		}
	}

	if !upgradeDryRun {
		// Update installed file checksums
		if info == nil {
			info = &version.Info{}
		}
		if info.InstalledFiles == nil {
			info.InstalledFiles = make(map[string]string)
		}
		for k, v := range allChecksums {
			info.InstalledFiles[k] = v
		}

		if err := version.StampUpgrade(heroDir, fromVersion, binaryVersion, totalUpdated, totalSkipped); err != nil {
			return fmt.Errorf("writing version stamp: %w", err)
		}
		// Also write the updated checksums
		info.HeroVersion = binaryVersion
		version.Write(heroDir, info)
	}

	fmt.Printf("\n%d updated, %d skipped", totalUpdated, totalSkipped)
	if upgradeDryRun {
		fmt.Print(" (dry run)")
	}
	if len(targets) > 1 {
		fmt.Printf(" across %d targets", len(targets))
	}
	fmt.Println()

	// Refresh installed git hooks so binary upgrades that change the
	// hook script (e.g. new staged files, additional invocations)
	// take effect for users who already have hooks installed. Skips
	// when not in a git repo or when no managed block is present —
	// `hero scan` is the install path; upgrade only refreshes what
	// the user already opted into.
	if !upgradeNoHooks {
		if err := refreshHooksIfPresent(projectRoot, upgradeDryRun, cmd.OutOrStdout()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: hook refresh failed: %v\n", err)
		}
	}

	return nil
}

// upgradeTarget runs the per-target upgrade pass: walk subdirs from
// the embedded FS, compare with checksums, write/update where allowed.
// Returns (updated, skipped, checksums-of-newly-written-files).
func upgradeTarget(projectRoot string, target install.Target, contentFS fs.FS, info *version.Info) (int, int, map[string]string) {
	updated := 0
	skipped := 0
	checksums := make(map[string]string)

	for _, subdir := range []string{"agents", "commands", "skills"} {
		entries, err := fs.ReadDir(contentFS, subdir)
		if err != nil {
			continue
		}

		destDir := resolveTargetDir(projectRoot, target, subdir)
		if destDir == "" {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			srcPath := subdir + "/" + entry.Name()
			destPath := filepath.Join(destDir, entry.Name())

			relPath, _ := filepath.Rel(projectRoot, destPath)
			if relPath == "" {
				relPath = filepath.Join(subdir, entry.Name())
			}

			// Check if destination exists and whether it's been modified
			if _, err := os.Stat(destPath); err == nil {
				if !upgradeForce && version.IsFileModified(info, relPath, destPath) {
					if upgradeDryRun {
						fmt.Printf("  skip    %s (customized)\n", relPath)
					} else {
						fmt.Printf("  skip    %s (customized — use --force to overwrite)\n", relPath)
					}
					skipped++
					continue
				}
			}

			if upgradeDryRun {
				fmt.Printf("  update  %s\n", relPath)
				updated++
				continue
			}

			data, err := fs.ReadFile(contentFS, srcPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  error   %s: %v\n", relPath, err)
				continue
			}

			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "  error   %s: %v\n", relPath, err)
				continue
			}

			if err := os.WriteFile(destPath, data, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "  error   %s: %v\n", relPath, err)
				continue
			}

			if cs, err := version.FileChecksum(destPath); err == nil {
				checksums[relPath] = cs
			}

			fmt.Printf("  update  %s\n", relPath)
			updated++
		}
	}

	if !upgradeDryRun {
		mcpOpts := install.Options{
			Target:    target,
			Mode:      install.ModeProject,
			TargetDir: projectRoot,
		}
		if mcpErr := install.RegisterMCP(target, mcpOpts); mcpErr != nil {
			fmt.Printf("  warning: could not update MCP server config: %v\n", mcpErr)
		} else {
			fmt.Println("  update  MCP server config")
		}
	}

	return updated, skipped, checksums
}

// resolveUpgradeTargets returns the list of install.Target values to
// upgrade. With no --target flags it returns every filesystem-detected
// target. With --target flags, it validates each against the supported
// set and returns the requested subset.
func resolveUpgradeTargets(projectRoot string, info *version.Info, requested []string) ([]install.Target, error) {
	detected := detectInstalledTargets(projectRoot, info)
	if len(requested) == 0 {
		return detected, nil
	}
	// Validate + dedupe requested.
	known := map[string]install.Target{
		"opencode": install.TargetOpenCode,
		"cursor":   install.TargetCursor,
		"claude":   install.TargetClaude,
	}
	seen := map[install.Target]bool{}
	out := make([]install.Target, 0, len(requested))
	for _, name := range requested {
		t, ok := known[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("unknown --target %q (valid: opencode, cursor, claude)", name)
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out, nil
}

// detectInstalledTargets returns every AI-tool target that has a
// directory installed in projectRoot. version.json's LastInstall is
// included so a freshly-installed-but-not-yet-touched target still
// shows up. Order is stable: opencode, cursor, claude — matches the
// supportedTargets list in install/.
//
// Returns an empty slice when no targets are detected (caller falls
// back to "stamp version only").
func detectInstalledTargets(projectRoot string, info *version.Info) []install.Target {
	seen := map[install.Target]bool{}
	var out []install.Target

	add := func(t install.Target) {
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, t)
	}

	// Filesystem probes — the authoritative signal. Order is stable.
	if _, err := os.Stat(filepath.Join(projectRoot, ".opencode")); err == nil {
		add(install.TargetOpenCode)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".cursor")); err == nil {
		add(install.TargetCursor)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".claude")); err == nil {
		add(install.TargetClaude)
	}

	// Fall back to LastInstall when no directory is detected — covers
	// edge cases like a targets-renamed repo where the dir was moved
	// but the install record is still authoritative.
	if len(out) == 0 && info != nil && info.LastInstall != nil && info.LastInstall.Target != "" {
		add(install.Target(info.LastInstall.Target))
	}

	return out
}

// resolveTargetDir returns the destination directory for a given subdir and target.
func resolveTargetDir(projectRoot string, target install.Target, subdir string) string {
	switch target {
	case install.TargetOpenCode:
		return filepath.Join(projectRoot, ".opencode", subdir)
	case install.TargetCursor:
		return filepath.Join(projectRoot, ".cursor", "rules", subdir)
	case install.TargetClaude:
		return filepath.Join(projectRoot, ".claude", subdir)
	default:
		return ""
	}
}
