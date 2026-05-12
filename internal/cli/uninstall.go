package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/version"
	"github.com/spf13/cobra"
)

var uninstallTarget string
var uninstallDryRun bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove hero files from the target tool",
	Long: `Removes hero-installed agents, commands, and skills from the specified
tool's configuration directory.

Only files that Hero originally installed are removed. User-created files
in these directories are preserved. Uses the manifest in .hero/version.json
to determine which files were installed by Hero.

For Claude Code, also removes the hero-managed section from CLAUDE.md.

Supported targets: opencode, cursor, claude, codex.`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().StringVar(&uninstallTarget, "target", "", "tool to uninstall from (opencode|cursor|claude|codex)")
	uninstallCmd.Flags().BoolVar(&uninstallDryRun, "dry-run", false, "show what would be removed without doing it")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()

	if uninstallTarget == "" {
		return fmt.Errorf("--target is required (opencode|cursor|claude|codex)")
	}

	// Load the install manifest to know which files Hero installed
	heroDir := filepath.Join(projectRoot, ".hero")
	versionInfo, _ := version.Read(heroDir)

	var removed, preserved int
	var err error

	switch uninstallTarget {
	case "opencode":
		removed, preserved, err = uninstallOpenCode(projectRoot, versionInfo)
	case "cursor":
		removed, preserved, err = uninstallCursor(projectRoot, versionInfo)
	case "claude":
		removed, preserved, err = uninstallClaude(projectRoot, versionInfo)
	case "codex":
		removed, preserved, err = uninstallCodex(projectRoot, versionInfo)
	default:
		return fmt.Errorf("unknown target %q; supported: opencode, cursor, claude, codex", uninstallTarget)
	}

	if err != nil {
		return err
	}

	if removed == 0 {
		fmt.Println("Nothing to remove.")
	} else {
		fmt.Printf("Removed %d hero files from %s.\n", removed, uninstallTarget)
	}
	if preserved > 0 {
		fmt.Printf("Preserved %d user-created files.\n", preserved)
	}

	// Clean up satellite folders for this target.
	if !uninstallDryRun {
		if satRemoved, satErr := uninstallSatellites(projectRoot, install.Target(uninstallTarget)); satErr != nil {
			fmt.Printf("  warning: could not clean satellites: %v\n", satErr)
		} else if satRemoved > 0 {
			fmt.Printf("Removed %s symlinks from %d satellite folder(s).\n", uninstallTarget, satRemoved)
		}
	}

	// Clean up the manifest entries for removed files
	if !uninstallDryRun && removed > 0 && versionInfo != nil && versionInfo.InstalledFiles != nil {
		if err := version.Write(heroDir, versionInfo); err != nil {
			fmt.Printf("  warning: could not update version.json: %v\n", err)
		}
	}

	return nil
}

func uninstallOpenCode(projectRoot string, versionInfo *version.Info) (int, int, error) {
	base := filepath.Join(projectRoot, ".opencode")
	dirs := []string{
		filepath.Join(base, "agents"),
		filepath.Join(base, "commands"),
		filepath.Join(base, "skills"),
	}

	removed, preserved := 0, 0
	for _, dir := range dirs {
		r, p, err := removeHeroFiles(projectRoot, dir, versionInfo)
		if err != nil {
			return removed, preserved, err
		}
		removed += r
		preserved += p
	}

	// Clean hero section from AGENTS.md
	agentsMdPath := filepath.Join(projectRoot, "AGENTS.md")
	if cleaned, err := removeHeroManagedSection(agentsMdPath); err == nil && cleaned {
		removed++
	}

	return removed, preserved, nil
}

func uninstallCursor(projectRoot string, versionInfo *version.Info) (int, int, error) {
	base := filepath.Join(projectRoot, ".cursor", "rules")
	dirs := []string{
		filepath.Join(base, "agents"),
		filepath.Join(base, "commands"),
		filepath.Join(base, "skills"),
	}

	removed, preserved := 0, 0
	for _, dir := range dirs {
		r, p, err := removeHeroFiles(projectRoot, dir, versionInfo)
		if err != nil {
			return removed, preserved, err
		}
		removed += r
		preserved += p
	}

	return removed, preserved, nil
}

func uninstallClaude(projectRoot string, versionInfo *version.Info) (int, int, error) {
	base := filepath.Join(projectRoot, ".claude")
	dirs := []string{
		filepath.Join(base, "agents"),
		filepath.Join(base, "commands"),
		filepath.Join(base, "skills"),
	}

	removed, preserved := 0, 0
	for _, dir := range dirs {
		r, p, err := removeHeroFiles(projectRoot, dir, versionInfo)
		if err != nil {
			return removed, preserved, err
		}
		removed += r
		preserved += p
	}

	// Remove hero section from CLAUDE.md
	claudeMdPath := filepath.Join(projectRoot, "CLAUDE.md")
	if cleaned, err := removeHeroManagedSection(claudeMdPath); err == nil && cleaned {
		removed++
	}

	// Strip hero-managed Stop / PreCompact entries from settings.json
	if err := install.UnwireClaudeHooks(install.Options{
		Mode:      install.ModeProject,
		TargetDir: projectRoot,
	}); err != nil {
		// Non-fatal: log nothing, return through removed count unchanged
		_ = err
	}

	return removed, preserved, nil
}

func uninstallCodex(projectRoot string, versionInfo *version.Info) (int, int, error) {
	base := filepath.Join(projectRoot, ".codex")
	dirs := []string{
		filepath.Join(base, "agents"),
		filepath.Join(base, "commands"),
		filepath.Join(base, "skills"),
	}

	removed, preserved := 0, 0
	for _, dir := range dirs {
		r, p, err := removeHeroFiles(projectRoot, dir, versionInfo)
		if err != nil {
			return removed, preserved, err
		}
		removed += r
		preserved += p
	}

	// Clean hero section from AGENTS.md
	agentsMdPath := filepath.Join(projectRoot, "AGENTS.md")
	if cleaned, err := removeHeroManagedSection(agentsMdPath); err == nil && cleaned {
		removed++
	}

	// Strip hero-managed Stop entry from .codex/hooks.json
	if err := install.UnwireCodexHooks(install.Options{
		Mode:      install.ModeProject,
		TargetDir: projectRoot,
	}); err != nil {
		_ = err
	}

	// Remove hero block from .codex/config.toml
	configPath := filepath.Join(base, "config.toml")
	if cleaned, err := removeHeroCodexConfigBlock(configPath); err == nil && cleaned {
		removed++
	}

	return removed, preserved, nil
}

// removeHeroCodexConfigBlock strips the # hero:managed ... # end:hero:managed block
// from .codex/config.toml. Returns true if a block was found and removed.
func removeHeroCodexConfigBlock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	content := string(data)
	const start = "# hero:managed"
	const end = "# end:hero:managed"

	si := strings.Index(content, start)
	if si < 0 {
		return false, nil
	}
	ei := strings.Index(content[si:], end)
	if ei < 0 {
		return false, nil
	}
	ei += si + len(end)

	newContent := strings.TrimRight(content[:si], "\n") + strings.TrimLeft(content[ei:], "\n")
	newContent = strings.TrimSpace(newContent)

	if newContent == "" {
		if !uninstallDryRun {
			os.Remove(path)
		}
		return true, nil
	}

	newContent += "\n"
	if !uninstallDryRun {
		if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
			return false, err
		}
	}
	return true, nil
}

// removeHeroFiles selectively removes only Hero-installed files from a directory.
// Files not tracked in the install manifest are preserved. Empty directories
// are cleaned up after removal.
func removeHeroFiles(projectRoot, dir string, versionInfo *version.Info) (int, int, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return 0, 0, nil
	}

	removed, preserved := 0, 0

	// Walk all files in the directory (including nested subdirs like skills/foo/SKILL.md)
	err = filepath.Walk(dir, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			return nil // process files only; dirs cleaned up after
		}

		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil
		}

		if isHeroInstalledFile(relPath, versionInfo) {
			if uninstallDryRun {
				fmt.Printf("  Would remove %s\n", path)
			} else {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("removing %s: %w", path, err)
				}
				fmt.Printf("  Removed %s\n", path)
			}
			// Remove from manifest so version.json stays accurate
			if versionInfo != nil && versionInfo.InstalledFiles != nil {
				delete(versionInfo.InstalledFiles, relPath)
			}
			removed++
		} else {
			if uninstallDryRun {
				fmt.Printf("  Preserving %s (user-created)\n", path)
			} else {
				fmt.Printf("  Preserving %s (user-created)\n", path)
			}
			preserved++
		}

		return nil
	})
	if err != nil {
		return removed, preserved, err
	}

	// Clean up empty directories (bottom-up)
	if !uninstallDryRun {
		cleanEmptyDirs(dir)
	}

	return removed, preserved, nil
}

// isHeroInstalledFile checks whether a file was installed by Hero.
// If we have a manifest, check if the file is tracked there.
// If no manifest exists (pre-version workspace), we fall back to checking
// whether the file matches known Hero file patterns.
func isHeroInstalledFile(relPath string, versionInfo *version.Info) bool {
	// If we have a manifest with tracked files, use it as the source of truth
	if versionInfo != nil && versionInfo.InstalledFiles != nil && len(versionInfo.InstalledFiles) > 0 {
		_, tracked := versionInfo.InstalledFiles[relPath]
		return tracked
	}

	// No manifest — fall back to known Hero file names.
	// These are the embedded file names from the hero binary (agents/, commands/, skills/).
	return isKnownHeroFile(relPath)
}

// knownHeroFiles is lazily built from the embedded ContentFS on first use.
// Keys are the embedded paths (e.g. "agents/feature-delivery-lead.md",
// "skills/spec-format.md").
var knownHeroFiles map[string]bool

// getKnownHeroFiles returns (lazily building) the set of file paths present in
// hero.ContentFS(). These are the canonical files that Hero ships.
func getKnownHeroFiles() map[string]bool {
	if knownHeroFiles != nil {
		return knownHeroFiles
	}
	knownHeroFiles = make(map[string]bool)
	contentFS := hero.ContentFS()
	fs.WalkDir(contentFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		knownHeroFiles[path] = true
		return nil
	})
	return knownHeroFiles
}

// isKnownHeroFile returns true if the relative path corresponds to a file that
// Hero ships in its embedded content FS. This is a fallback for workspaces
// without a version.json manifest.
//
// It handles the different install layouts:
//   - OpenCode: agents/*.md, commands/*.md, skills/<name>/SKILL.md (nested)
//   - Cursor:   .cursor/rules/agents/*.md, .cursor/rules/commands/*.md, .cursor/rules/skills/*.md
//   - Claude:   .claude/agents/*.md, .claude/commands/*.md, .claude/skills/*.md
func isKnownHeroFile(relPath string) bool {
	known := getKnownHeroFiles()

	// Normalize path separators to forward slash for comparison
	normalized := filepath.ToSlash(relPath)

	// Strip the tool-specific prefix to get to the content-relative path.
	// e.g. ".opencode/agents/foo.md" -> "agents/foo.md"
	//      ".cursor/rules/agents/foo.md" -> "agents/foo.md"
	//      ".claude/agents/foo.md" -> "agents/foo.md"
	prefixes := []string{
		".opencode/",
		".cursor/rules/",
		".claude/",
		".codex/",
	}
	contentRel := normalized
	for _, prefix := range prefixes {
		if strings.HasPrefix(normalized, prefix) {
			contentRel = normalized[len(prefix):]
			break
		}
	}

	// Direct match: agents/foo.md, commands/bar.md, skills/baz.md
	if known[contentRel] {
		return true
	}

	// Handle OpenCode nested skill layout: skills/<name>/SKILL.md
	// The embedded FS has skills/<name>.md but OpenCode installs as skills/<name>/SKILL.md
	if strings.HasPrefix(contentRel, "skills/") && strings.HasSuffix(contentRel, "/SKILL.md") {
		// Extract the skill name from skills/<name>/SKILL.md
		parts := strings.Split(contentRel, "/")
		if len(parts) == 3 {
			embeddedPath := "skills/" + parts[1] + ".md"
			if known[embeddedPath] {
				return true
			}
		}
	}

	return false
}

// cleanEmptyDirs removes empty directories from the bottom up.
func cleanEmptyDirs(root string) {
	// Walk bottom-up by collecting dirs first, then removing empty ones in reverse
	var dirs []string
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})

	// Remove in reverse order (deepest first)
	for i := len(dirs) - 1; i >= 0; i-- {
		entries, err := os.ReadDir(dirs[i])
		if err == nil && len(entries) == 0 {
			os.Remove(dirs[i])
		}
	}
}

// removeHeroManagedSection removes a <!-- hero:managed --> section from a markdown file.
// Returns true if a section was found and removed.
func removeHeroManagedSection(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	content := string(data)
	marker := "<!-- hero:managed -->"

	startIdx := strings.Index(content, marker)
	if startIdx < 0 {
		return false, nil
	}

	afterStart := startIdx + len(marker)
	endIdx := strings.Index(content[afterStart:], marker)
	if endIdx < 0 {
		return false, nil
	}
	endIdx += afterStart + len(marker)

	// Remove the section and any trailing newlines
	newContent := content[:startIdx] + strings.TrimLeft(content[endIdx:], "\n")
	newContent = strings.TrimRight(newContent, "\n")

	if newContent == "" {
		// File is now empty — remove it
		if !uninstallDryRun {
			os.Remove(path)
		}
		if uninstallDryRun {
			fmt.Printf("  Would remove %s (empty after cleanup)\n", path)
		} else {
			fmt.Printf("  Cleaned hero section from %s (removed empty file)\n", path)
		}
		return true, nil
	}

	newContent += "\n"

	if uninstallDryRun {
		fmt.Printf("  Would clean hero section from %s\n", path)
	} else {
		if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
			return false, err
		}
		fmt.Printf("  Cleaned hero section from %s\n", path)
	}

	return true, nil
}

// uninstallSatellites removes the symlink trees for the given target
// from every satellite folder recorded in satellites.local.json. Returns
// the number of satellite folders touched. Satellite manifest entries
// whose target list becomes empty are dropped.
func uninstallSatellites(projectRoot string, target install.Target) (int, error) {
	heroDir := filepath.Join(projectRoot, ".hero")
	local, err := install.LoadSatellitesLocal(heroDir)
	if err != nil {
		return 0, err
	}
	touched := 0
	keep := make([]install.SatelliteEntry, 0, len(local.Satellites))
	for _, e := range local.Satellites {
		hasTarget := false
		newTargets := make([]string, 0, len(e.Targets))
		for _, t := range e.Targets {
			if install.Target(t) == target {
				hasTarget = true
				continue
			}
			newTargets = append(newTargets, t)
		}
		if !hasTarget {
			keep = append(keep, e)
			continue
		}
		satAbs := filepath.Join(projectRoot, filepath.FromSlash(e.Path))
		if err := install.RemoveSatellite(satAbs, []install.Target{target}); err != nil {
			return touched, err
		}
		touched++
		if len(newTargets) > 0 {
			e.Targets = newTargets
			keep = append(keep, e)
		}
	}
	local.Satellites = keep
	return touched, install.SaveSatellitesLocal(heroDir, local)
}
