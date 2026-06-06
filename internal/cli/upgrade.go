package cli

import (
	"fmt"
	"io"
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
	upgradeCmd.Flags().StringSliceVar(&upgradeTargets, "target", nil, "narrow to one or more targets (claude, opencode, cursor, codex, copilot, generic); default: every detected target")
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

	// Proactively transition version-skewed workspaces to NEXT
	// projection. Workspaces created before projection existed default
	// to next.projected == false; rather than wait for the next
	// checkpoint to auto-migrate, do it at upgrade. Idempotent (no-op
	// when already projected). Content-preserving — surfaces only on
	// failure, leaving NEXT.md untouched in that case.
	if !cfg.NextProjected() {
		if err := migrateToProjection(projectRoot, cfg, io.Discard); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: NEXT.md projection migration failed: %v — run `hero next migrate-to-projection` to retry\n", err)
		} else if reloaded, lerr := config.Load(projectRoot); lerr == nil {
			cfg = reloaded
		}
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

	if fromVersion == binaryVersion && fromVersion != "dev" && !upgradeForce {
		fmt.Printf("Workspace is already at %s — nothing to upgrade.\n", displayVersion(binaryVersion))
		return nil
	}

	// Reject downgrade attempts — binary is older than workspace.
	if binaryVersion != "dev" && fromVersion != "unknown" && fromVersion != "dev" {
		cmp := version.CompareVersions(binaryVersion, fromVersion)
		if cmp < 0 {
			return fmt.Errorf("cannot downgrade workspace from %s to %s — use a newer hero binary or run 'hero init' to create a fresh workspace", displayVersion(fromVersion), displayVersion(binaryVersion))
		}
	}

	fmt.Printf("Upgrading workspace from %s to %s\n\n", displayVersion(fromVersion), displayVersion(binaryVersion))

	// Use the embedded content filesystem (overridable for tests).
	// When no override is provided, build the merged core + active-domain
	// FS so upgrade renders the same content shape that install does.
	// Existing workspaces installed before this change gain core files
	// on first upgrade; the trust map (built from result.Copied below)
	// records them so subsequent upgrades manage them rather than
	// treating them as user-authored.
	contentFS := upgradeContentFS
	if contentFS == nil {
		activeDomain := cfg.Domain
		if activeDomain == "" {
			activeDomain = "engineering"
		}
		domainFS, domainErr := hero.DomainFS(activeDomain)
		if domainErr != nil {
			return fmt.Errorf("resolving domain %q: %w", activeDomain, domainErr)
		}
		contentFS = hero.OverlayFS(domainFS, hero.CoreFS())
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
		fmt.Printf("\nWorkspace version updated to %s\n", displayVersion(binaryVersion))
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
	failedTargets := 0
	allChecksums := make(map[string]string)

	for i, target := range targets {
		if len(targets) > 1 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("--- %s ---\n", target)
		}
		updated, skipped, checksums, targetErr := upgradeTarget(projectRoot, target, contentFS, info)
		if targetErr != nil {
			failedTargets++
		}
		totalUpdated += updated
		totalSkipped += skipped
		for k, v := range checksums {
			allChecksums[k] = v
		}
	}

	// Only stamp the new binary version once every target installed
	// cleanly. Stamping after a partial failure lies to the next
	// upgrade — it'll short-circuit on "already at" and the user
	// has to hand-edit version.json to recover. Checksums of the
	// files that DID install still get recorded so future upgrades
	// treat them as Hero-managed rather than user-edited.
	stampVersion := failedTargets == 0

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

		// Record checksums for files that did install regardless of
		// failure — they ARE Hero-managed now, even in a partial run.
		// But only roll HeroVersion forward when every target finished
		// cleanly, so retry sees the real prior version.
		stampedVersion := fromVersion
		if stampVersion {
			stampedVersion = binaryVersion
		}
		if err := version.StampUpgrade(heroDir, fromVersion, stampedVersion, totalUpdated, totalSkipped); err != nil {
			return fmt.Errorf("writing version stamp: %w", err)
		}
		info.HeroVersion = stampedVersion
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

// displayVersion normalizes a version string for user-facing output so
// the visible form always carries exactly one leading "v". Versions
// coming from `git describe` already include the "v" prefix; values
// stamped via tests or older code paths may not. Without normalization
// the upgrade banner printed "vv0.14.3" or, worse, dropped the prefix
// inconsistently between code paths.
//
// Special markers ("unknown", "dev", empty) pass through unchanged so
// they read naturally in messages like "Upgrading workspace from
// unknown to v0.14.4".
func displayVersion(v string) string {
	switch v {
	case "", "unknown", "dev":
		return v
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// trustedChecksumsFromInfo returns the project-relative-path → SHA256
// map a prior Hero install recorded. Used by upgrade so the install
// pipeline can distinguish "Hero installed this at an older version"
// (safe to refresh) from "user edited this after Hero installed it"
// (must preserve).
func trustedChecksumsFromInfo(info *version.Info) map[string]string {
	if info == nil || len(info.InstalledFiles) == 0 {
		return nil
	}
	out := make(map[string]string, len(info.InstalledFiles))
	for k, v := range info.InstalledFiles {
		out[filepath.ToSlash(k)] = v
	}
	return out
}

// upgradeTarget runs the per-target upgrade by invoking the target's
// own installer (install.Run with Force=upgradeForce). This delegates
// to the per-target install class so upgrade automatically inherits
// every install-side behavior:
//   - Cleanup of dead bytes from prior install layouts
//     (e.g. .codex/agents/*.md once Codex switched to .toml).
//   - Format-rendering for harnesses that need a different shape
//     than canonical (Codex TOML, Copilot .prompt.md).
//   - Symlinks-where-supported under the canonical-source layout.
//   - Hook/permission/MCP wiring.
//
// User-customized files survive: the install pipeline's
// copyFileFromFS refuses to overwrite drifted destinations unless
// Force is set (multi-harness-install-collision contract).
func upgradeTarget(projectRoot string, target install.Target, contentFS fs.FS, info *version.Info) (int, int, map[string]string, error) {
	checksums := make(map[string]string)

	opts := install.Options{
		Target:           target,
		Mode:             install.ModeProject,
		TargetDir:        projectRoot,
		Force:            upgradeForce,
		DryRun:           upgradeDryRun,
		ContentFS:        contentFS,
		TrustedChecksums: trustedChecksumsFromInfo(info),
	}
	result, err := install.Run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  error   %s: %v\n", target, err)
		return 0, 0, checksums, err
	}

	updated := len(result.Copied)
	skipped := len(result.Skipped)

	for _, action := range result.Copied {
		relPath, _ := filepath.Rel(projectRoot, action.Dest)
		if relPath == "" {
			relPath = filepath.Base(action.Dest)
		}
		fmt.Printf("  update  %s\n", relPath)
	}
	for _, skipMsg := range result.Skipped {
		fmt.Printf("  skip    %s\n", skipMsg)
	}

	// Record per-file checksums at the harness-destination paths so the
	// version-info trust map survives upgrade cycles. Destinations may
	// be symlinks to canonical (filepath.Walk wouldn't recurse through
	// them) — resolve each destDir via EvalSymlinks before walking, then
	// rewrite paths back into harness-relative form.
	if !upgradeDryRun {
		for _, kind := range []string{"agents", "commands", "skills"} {
			destDir := resolveTargetDir(projectRoot, target, kind)
			if destDir == "" {
				continue
			}
			realDir, err := filepath.EvalSymlinks(destDir)
			if err != nil {
				continue
			}
			_ = filepath.Walk(realDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info == nil || info.IsDir() {
					return nil
				}
				cs, err := version.FileChecksum(path)
				if err != nil {
					return nil
				}
				// Rewrite the canonical-resolved path back into a
				// destDir-relative path for trust-map keying.
				inner, err := filepath.Rel(realDir, path)
				if err != nil {
					return nil
				}
				harnessPath := filepath.Join(destDir, inner)
				rel, err := filepath.Rel(projectRoot, harnessPath)
				if err != nil {
					return nil
				}
				checksums[rel] = cs
				return nil
			})
		}
	}

	return updated, skipped, checksums, nil
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
		"codex":    install.TargetCodex,
		"copilot":  install.TargetCopilot,
		"generic":  install.TargetGeneric,
	}
	seen := map[install.Target]bool{}
	out := make([]install.Target, 0, len(requested))
	for _, name := range requested {
		t, ok := known[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("unknown --target %q (valid: opencode, cursor, claude, codex, copilot, generic)", name)
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
// directory installed in projectRoot. Delegates to install.DetectInstalledTargets
// which walks the targetLayouts registry — so new targets added there
// are picked up automatically. version.json's LastInstall is included
// as a fallback so a freshly-installed-but-not-yet-touched target
// still shows up.
//
// Returns an empty slice when no targets are detected (caller falls
// back to "stamp version only").
func detectInstalledTargets(projectRoot string, info *version.Info) []install.Target {
	// Filesystem probes via the install package's registry — authoritative.
	out := install.DetectInstalledTargets(projectRoot)

	// Fall back to LastInstall when no directory is detected — covers
	// edge cases like a targets-renamed repo where the dir was moved
	// but the install record is still authoritative.
	if len(out) == 0 && info != nil && info.LastInstall != nil && info.LastInstall.Target != "" {
		out = append(out, install.Target(info.LastInstall.Target))
	}

	return out
}

// resolveTargetDir returns the destination directory for a given subdir and target.
// Uses the install package's TargetLayout registry so new targets are
// picked up automatically.
func resolveTargetDir(projectRoot string, target install.Target, subdir string) string {
	layout := install.LayoutFor(target)
	if layout == nil || layout.SubDir == "" {
		return ""
	}
	return filepath.Join(projectRoot, layout.SubDir, subdir)
}
