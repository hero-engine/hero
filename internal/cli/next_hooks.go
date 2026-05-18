package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/projection"
	"github.com/spf13/cobra"
)

// Marker block delimiters used in .git/hooks/* and .gitattributes so
// hero install / install-hooks can be re-run idempotently — anything
// outside the markers belongs to the user and is preserved.
const (
	hookMarkerStart = "# >>> hero next hooks (managed) >>>"
	hookMarkerEnd   = "# <<< hero next hooks (managed) <<<"
	gaMarkerStart   = "# >>> hero next merge driver (managed) >>>"
	gaMarkerEnd     = "# <<< hero next merge driver (managed) <<<"

	mergeDriverName = "hero-next"
)

var nextInstallHooksCmd = &cobra.Command{
	Use:   "install-hooks",
	Short: "Install git hooks + merge driver for projection-based NEXT files",
	Long: `Writes:

  .git/hooks/pre-commit       — runs 'hero next checkpoint -q' before
                                each commit so the projected files
                                reflect graph state at commit time,
                                then 'git add' the projected files so
                                they travel with the commit (prevents
                                agents from stranding handoff state
                                on the local machine).
  .git/hooks/post-merge       — runs 'hero next checkpoint -q' after
                                a merge so the merged result is regen
                                from graph.
  .git/config (merge driver)  — registers 'hero-next' driver pointing
                                at 'hero next merge-resolve'. Catches
                                conflicts on .hero/next/*.md and
                                regenerates instead of marking up.
  .gitattributes              — adds 'merge=hero-next' for the
                                projected files (idempotent — adds a
                                marker block; preserves user content).

All writes are marker-delimited and idempotent — re-run safely. To
remove, delete the marker blocks.`,
	RunE: runNextInstallHooks,
}

var nextMergeResolveOutput string

var nextMergeResolveCmd = &cobra.Command{
	Use:   "merge-resolve",
	Short: "Internal: git merge driver that regenerates a projected file from graph",
	Long: `Used by the 'hero-next' git merge driver to resolve conflicts on
projected NEXT files. Ignores both sides of the merge and writes a
fresh projection from the local graph to --output.

Not intended for direct user invocation — git calls this when merging
a file marked 'merge=hero-next' in .gitattributes. To trigger a
regen manually, use 'hero next checkpoint' instead.`,
	Hidden: true,
	RunE:   runNextMergeResolve,
}

func init() {
	nextMergeResolveCmd.Flags().StringVar(&nextMergeResolveOutput, "output", "", "path the regenerated content should be written to (git's %A)")
}

// runNextInstallHooks writes the git hooks, the .gitattributes
// directive, and registers the merge driver in .git/config.
func runNextInstallHooks(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	gitDir, err := resolveGitDir(projectRoot)
	if err != nil {
		return err
	}

	// 1. pre-commit hook
	preCommit := filepath.Join(gitDir, "hooks", "pre-commit")
	if err := writeHookFile(preCommit, hookScript("pre-commit")); err != nil {
		return fmt.Errorf("pre-commit hook: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "wrote", preCommit)

	// 2. post-merge hook
	postMerge := filepath.Join(gitDir, "hooks", "post-merge")
	if err := writeHookFile(postMerge, hookScript("post-merge")); err != nil {
		return fmt.Errorf("post-merge hook: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "wrote", postMerge)

	// 3. merge driver in .git/config
	if err := registerMergeDriver(projectRoot); err != nil {
		return fmt.Errorf("merge driver: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "registered merge driver:", mergeDriverName)

	// 4. .gitattributes
	if err := updateGitAttributes(projectRoot); err != nil {
		return fmt.Errorf(".gitattributes: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "updated .gitattributes for merge=hero-next")

	return nil
}

// runNextMergeResolve is invoked by git as the merge driver. The
// merge driver protocol passes %A (current/ours), %O (base), %B
// (theirs) — git expects the merged result written to %A.
//
// Strategy: ignore %O / %B, regenerate from the local graph, write
// to %A. Result: no conflict markers ever land in projected files
// for users who have the driver registered.
func runNextMergeResolve(cmd *cobra.Command, args []string) error {
	if nextMergeResolveOutput == "" {
		return fmt.Errorf("--output is required (git supplies %%A)")
	}
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(projectRoot)

	// QUEUE.md doesn't depend on the graph store — it's regenerated
	// from the on-disk spec corpus. Detect the path and route there
	// before attempting NEXT-file projection.
	if isQueueOutputPath(nextMergeResolveOutput) {
		content, err := RenderQueueSnapshot(heroDir)
		if err != nil {
			return fmt.Errorf("queue snapshot: %w", err)
		}
		return os.WriteFile(nextMergeResolveOutput, []byte(content), 0o644)
	}

	// SNAPSHOT.md is regenerated from the graph + repo shape by the
	// project-snapshot projector. Same merge-driver pattern as
	// NEXT.md: ignore both sides, write a fresh projection.
	if isSnapshotOutputPath(nextMergeResolveOutput) {
		return runSnapshotMergeResolve(projectRoot, heroDir, cfg)
	}

	user := userFromOutputPath(nextMergeResolveOutput)
	if user == "" {
		// Unknown projected path — leave file as-is (current side
		// wins, no conflict markers). Still exit 0 so git treats
		// the merge as successful.
		return nil
	}

	content, err := projection.UserHandoffMD(store, projection.UserHandoffOptions{
		User:    user,
		RepoKey: repoKey,
	})
	if err != nil {
		return fmt.Errorf("projection: %w", err)
	}
	if err := os.WriteFile(nextMergeResolveOutput, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// isQueueOutputPath reports whether the merge driver's --output path
// names the QUEUE.md snapshot.
func isQueueOutputPath(path string) bool {
	return filepath.Base(path) == QueueFileName
}

// isSnapshotOutputPath reports whether the merge driver's --output
// path names the project SNAPSHOT.md.
func isSnapshotOutputPath(path string) bool {
	return filepath.Base(path) == "SNAPSHOT.md"
}

// runSnapshotMergeResolve regenerates .hero/SNAPSHOT.md from the
// local graph and writes the result to --output. Used by the
// hero-next merge driver when git asks it to resolve a conflict on
// the snapshot file. Mirrors the strategy used for NEXT.md and
// QUEUE.md: ignore both sides, project fresh.
func runSnapshotMergeResolve(projectRoot, heroDir string, cfg config.Config) error {
	// Snapshot projection is implemented in internal/snapshot. We
	// invoke it directly here via the same options the checkpoint
	// command uses; the only difference is that we redirect the
	// rendered bytes to --output rather than letting the projector
	// write to .hero/SNAPSHOT.md.
	missionPath := filepath.Join(heroDir, "mission.md")
	mission := readMissionOneLiner(missionPath)
	projectName := filepath.Base(projectRoot)

	// Run the projector at HeroDir; capture rendered bytes after the
	// write by reading the written file.
	archive := cfg.SnapshotArchive()
	_, err := snapshotProject(snapshotProjectArgs{
		ProjectRoot:  projectRoot,
		HeroDir:      heroDir,
		ProjectName:  projectName,
		Mission:      mission,
		ArchiveCfg:   archive,
		Milestones:   cfg.SnapshotMilestonesEnabled(),
	})
	if err != nil {
		return fmt.Errorf("snapshot projection: %w", err)
	}
	rendered, err := os.ReadFile(filepath.Join(heroDir, "SNAPSHOT.md"))
	if err != nil {
		return fmt.Errorf("read regenerated snapshot: %w", err)
	}
	return os.WriteFile(nextMergeResolveOutput, rendered, 0o644)
}

// snapshotProjectArgs is the merge-driver glue type so we don't
// pull internal/snapshot into the import-graph of every CLI file.
type snapshotProjectArgs struct {
	ProjectRoot string
	HeroDir     string
	ProjectName string
	Mission     string
	ArchiveCfg  config.SnapshotArchiveConfig
	Milestones  bool
}

// snapshotProject is filled in by checkpoint.go (which already
// imports internal/snapshot) to keep this file free of the
// dependency. The indirection lets the merge driver reuse the
// projection without bloating package boundaries.
var snapshotProject = func(args snapshotProjectArgs) (any, error) {
	return nil, fmt.Errorf("snapshot projection unavailable in this build")
}

// userFromOutputPath extracts <user> from .../next/<user>.md. Returns
// empty string for paths that don't fit the user-handoff shape.
func userFromOutputPath(path string) string {
	base := filepath.Base(path)
	if !strings.HasSuffix(base, ".md") || strings.HasSuffix(base, ".local.md") {
		return ""
	}
	parent := filepath.Base(filepath.Dir(path))
	if parent != "next" {
		return ""
	}
	return strings.TrimSuffix(base, ".md")
}

// preCommitHookInstalled reports whether the Hero managed block is
// present in .git/hooks/pre-commit. Returns false (no error) for
// non-git directories or repos with no hook file yet — callers
// treat "not installed" as the actionable case, not "error".
func preCommitHookInstalled(projectRoot string) bool {
	_, err := currentPreCommitManagedBlock(projectRoot)
	return err == nil
}

// currentPreCommitManagedBlock returns the contents of the
// hero-managed block (start marker through end marker) inside the
// installed `.git/hooks/pre-commit` file. Returns an error when the
// repo is non-git, the hook file is missing, or the markers are
// absent / truncated. Used by the upgrade refresh path and the
// `hero check` stale-hook detector.
func currentPreCommitManagedBlock(projectRoot string) (string, error) {
	gitDir, err := resolveGitDir(projectRoot)
	if err != nil {
		return "", err
	}
	body, err := os.ReadFile(filepath.Join(gitDir, "hooks", "pre-commit"))
	if err != nil {
		return "", err
	}
	src := string(body)
	startIdx := strings.Index(src, hookMarkerStart)
	if startIdx < 0 {
		return "", fmt.Errorf("pre-commit hook has no hero managed block")
	}
	endIdx := strings.Index(src, hookMarkerEnd)
	if endIdx < 0 {
		return "", fmt.Errorf("pre-commit hook managed block is truncated (missing end marker)")
	}
	return src[startIdx : endIdx+len(hookMarkerEnd)], nil
}

// preCommitHookStale reports whether the installed pre-commit
// managed block diverges from `hookScript("pre-commit")`. Returns
// (false, nil) when the hook isn't installed (callers handle missing
// separately via preCommitHookInstalled). Whitespace-trimmed compare
// so trailing newlines don't trigger false positives.
func preCommitHookStale(projectRoot string) (bool, error) {
	installed, err := currentPreCommitManagedBlock(projectRoot)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(installed) != strings.TrimSpace(hookScript("pre-commit")), nil
}

// refreshHooksIfPresent re-runs the hook install when a hero managed
// block is already present in `.git/hooks/pre-commit`. Used by
// `hero upgrade` so binary upgrades that change hookScript() take
// effect for users who already have hooks installed. Skips silently
// for non-git repos and for repos with no managed block (respecting
// explicit user removal — `hero scan` is the install path).
//
// In dryRun mode, reports what would happen without modifying files.
func refreshHooksIfPresent(projectRoot string, dryRun bool, w io.Writer) error {
	if _, err := resolveGitDir(projectRoot); err != nil {
		// Not a git repo — silently skip.
		return nil
	}
	if !preCommitHookInstalled(projectRoot) {
		// User explicitly has no managed block — respect that.
		return nil
	}
	stale, err := preCommitHookStale(projectRoot)
	if err != nil {
		return err
	}
	if !stale {
		if dryRun {
			fmt.Fprintln(w, "pre-commit hook is current — no refresh needed")
		}
		return nil
	}
	if dryRun {
		fmt.Fprintln(w, "would refresh pre-commit hook (managed block content has changed)")
		return nil
	}
	if err := installNextHooksQuiet(projectRoot); err != nil {
		return err
	}
	fmt.Fprintln(w, "refreshed pre-commit hook")
	return nil
}

// installNextHooksQuiet runs the same install as `hero next install-hooks`
// but discards stdout — used by callers that want to ensure the hook
// is in place without mirroring the install command's verbose output.
func installNextHooksQuiet(projectRoot string) error {
	gitDir, err := resolveGitDir(projectRoot)
	if err != nil {
		return err
	}
	preCommit := filepath.Join(gitDir, "hooks", "pre-commit")
	if err := writeHookFile(preCommit, hookScript("pre-commit")); err != nil {
		return fmt.Errorf("pre-commit hook: %w", err)
	}
	postMerge := filepath.Join(gitDir, "hooks", "post-merge")
	if err := writeHookFile(postMerge, hookScript("post-merge")); err != nil {
		return fmt.Errorf("post-merge hook: %w", err)
	}
	if err := registerMergeDriver(projectRoot); err != nil {
		return fmt.Errorf("merge driver: %w", err)
	}
	if err := updateGitAttributes(projectRoot); err != nil {
		return fmt.Errorf(".gitattributes: %w", err)
	}
	return nil
}

// resolveGitDir returns the absolute path to the .git directory by
// asking git itself — handles worktrees and submodules where .git
// may not be a directory.
func resolveGitDir(projectRoot string) (string, error) {
	cmd := exec.Command("git", "-C", projectRoot, "rev-parse", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repo: %w", err)
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(projectRoot, gitDir)
	}
	return gitDir, nil
}

// writeHookFile writes (or merges into) a hook file. Preserves any
// pre-existing content outside the marker block; replaces content
// between markers; creates the file with shebang + marker block when
// none exists.
func writeHookFile(path, managedBlock string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	existing, _ := os.ReadFile(path)
	body := mergeMarkerBlock(string(existing), hookMarkerStart, hookMarkerEnd, managedBlock)
	if !strings.HasPrefix(body, "#!") {
		body = "#!/usr/bin/env bash\nset -e\n\n" + strings.TrimLeft(body, "\n")
	}
	return os.WriteFile(path, []byte(body), 0o755)
}

// hookScript returns the body of a single hook's managed block.
//
// pre-commit additionally re-stages the projected NEXT files so they
// travel with the commit being made. Without that, agents that do
// `git add <their files> && git commit` leave the regenerated handoff
// state stranded as unstaged drift on the local machine — defeating
// the whole "next session starts smart" promise. See spec
// pre-commit-auto-stage-next.
func hookScript(kind string) string {
	queueRefresh := ""
	indexRefresh := ""
	stage := ""
	if kind == "pre-commit" {
		// Index sync first so the queue write reads from a current
		// index. All three calls are best-effort.
		indexRefresh = `
  hero index --if-stale -q || true`
		queueRefresh = `
  hero queue write -q || true`
		stage = `
  # Re-stage projected files so they travel with the commit. The
  # 2>/dev/null || true swallows the case where a path doesn't exist
  # yet (fresh repo, solo mode, no per-user file).
  git add -- .hero/NEXT.md .hero/next/*.md .hero/QUEUE.md 2>/dev/null || true`
	}
	return fmt.Sprintf(`%s
# Refresh projected NEXT files, sync the search index against any
# spec edits in this commit, and regenerate the QUEUE.md snapshot
# so the commit / merge reflects current state. Best-effort: a hero
# failure shouldn't block git operations.
if command -v hero >/dev/null 2>&1; then
  hero next checkpoint -q || true%s%s%s
fi
%s`, hookMarkerStart, indexRefresh, queueRefresh, stage, hookMarkerEnd)
}

// mergeMarkerBlock either replaces the existing managed block in src
// or appends a new one if no markers are found. Outside-marker
// content is preserved verbatim.
func mergeMarkerBlock(src, start, end, block string) string {
	startIdx := strings.Index(src, start)
	if startIdx < 0 {
		// No managed block yet — append.
		if strings.TrimSpace(src) == "" {
			return block + "\n"
		}
		return strings.TrimRight(src, "\n") + "\n\n" + block + "\n"
	}
	endIdx := strings.Index(src, end)
	if endIdx < 0 {
		// Truncated previous block — replace from start to EOF.
		return strings.TrimRight(src[:startIdx], "\n") + "\n\n" + block + "\n"
	}
	endIdx += len(end)
	prefix := strings.TrimRight(src[:startIdx], "\n")
	suffix := strings.TrimLeft(src[endIdx:], "\n")
	if prefix == "" && suffix == "" {
		return block + "\n"
	}
	if prefix == "" {
		return block + "\n\n" + suffix
	}
	if suffix == "" {
		return prefix + "\n\n" + block + "\n"
	}
	return prefix + "\n\n" + block + "\n\n" + suffix
}

// registerMergeDriver writes [merge "hero-next"] driver in .git/config.
// Idempotent — git config set always overwrites the prior value.
func registerMergeDriver(projectRoot string) error {
	driver := "hero next merge-resolve --output %A"
	for _, args := range [][]string{
		{"merge." + mergeDriverName + ".name", "Hero — regenerate projected NEXT files from graph on conflict"},
		{"merge." + mergeDriverName + ".driver", driver},
	} {
		cmd := exec.Command("git", append([]string{"-C", projectRoot, "config"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config %v: %w\n%s", args, err, out)
		}
	}
	return nil
}

// updateGitAttributes ensures .gitattributes contains the marker-
// bounded merge directive. Idempotent.
func updateGitAttributes(projectRoot string) error {
	path := filepath.Join(projectRoot, ".gitattributes")
	existing, _ := os.ReadFile(path)

	managed := fmt.Sprintf(`%s
.hero/next/*.md merge=%s
.hero/NEXT.md merge=%s
.hero/QUEUE.md merge=%s
%s`, gaMarkerStart, mergeDriverName, mergeDriverName, mergeDriverName, gaMarkerEnd)

	body := mergeMarkerBlock(string(existing), gaMarkerStart, gaMarkerEnd, managed)
	return os.WriteFile(path, []byte(body), 0o644)
}
