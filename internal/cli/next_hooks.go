package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
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

// handoffFilePaths is the single source of truth for the projected
// handoff files that must (a) travel with every commit via the
// pre-commit staging `git add` and (b) be declared merge=union in
// .gitattributes. Both lists are derived from this slice so they can
// never drift apart again — the historical SNAPSHOT.md gap existed
// precisely because the two lists were maintained independently.
//
// Order is significant only for stable output (tests assert exact
// pathspec strings). The `.hero/next/*.md` glob matches the per-user
// handoff files (e.g. `alice.md`). It also expands to match
// `alice.local.md`, but those are NEVER staged: `.hero/next/*.local.md`
// is gitignored, and `git add` skips ignored paths (no `-f`). So
// gitignore — not the glob — is what keeps the local-only machine-state
// files out of commits. `.hero/QUEUE.md` may be absent (a sibling effort
// may drop it as a file); the per-path staging loop swallows missing
// paths via `2>/dev/null || true`, so its absence is a safe no-op.
var handoffFilePaths = []string{
	".hero/NEXT.md",
	".hero/next/*.md",
	".hero/SNAPSHOT.md",
	".hero/QUEUE.md",
}

var nextInstallHooksCmd = &cobra.Command{
	Use:   "install-hooks",
	Short: "Install git hooks + .gitattributes merge strategy for projection-based NEXT files",
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
  .gitattributes              — adds 'merge=union' for the projected
                                files (idempotent — adds a marker
                                block; preserves user content). 'union'
                                is a git built-in, so it needs no
                                per-clone .git/config registration and
                                travels with the repo: fresh clones and
                                CI resolve these merges marker-free.

All writes are marker-delimited and idempotent — re-run safely. To
remove, delete the marker blocks.`,
	RunE: runNextInstallHooks,
}

// runNextInstallHooks writes the git hooks and the .gitattributes
// merge directive.
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

	// 3. .gitattributes
	if err := updateGitAttributes(projectRoot); err != nil {
		return fmt.Errorf(".gitattributes: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "updated .gitattributes for merge=union")

	return nil
}

// preCommitHookInstalled reports whether the Hero managed block is
// present in .git/hooks/pre-commit. Returns false (no error) for
// non-git directories or repos with no hook file yet — callers
// treat "not installed" as the actionable case, not "error".
func preCommitHookInstalled(projectRoot string) bool {
	_, err := currentPreCommitManagedBlock(projectRoot)
	return err == nil
}

// preCommitHasHeroHookButNoStaging reports whether the installed
// .git/hooks/pre-commit contains SOME Hero-managed content (the generic
// installer's `# Hero git hook` marker) but lacks the hero-next staging
// managed block — i.e. a repo that fires `hero hook pre-commit` on every
// commit but never stages the projected handoff files. This is the
// distinct misconfiguration `hero check` flags: the projecting-but-not-
// staging gap. Returns false when there is no Hero pre-commit hook at all
// (that case is the plain "not installed" warning) or when the staging
// block is present (correctly wired).
func preCommitHasHeroHookButNoStaging(projectRoot string) bool {
	if preCommitHookInstalled(projectRoot) {
		// Staging block present — correctly wired, not a misconfig.
		return false
	}
	gitDir, err := resolveGitDir(projectRoot)
	if err != nil {
		return false
	}
	body, err := os.ReadFile(filepath.Join(gitDir, "hooks", "pre-commit"))
	if err != nil {
		return false
	}
	// genericHeroHookMarker mirrors internal/hooks.heroMarker ("# Hero
	// git hook"); duplicated as a literal here to avoid a CLI→hooks
	// dependency just for a substring check.
	return strings.Contains(string(body), "# Hero git hook")
}

// hookInstallOptedOut reports whether the user has explicitly opted
// out of automatic hook self-install. Used by `hero install`'s
// self-heal path to distinguish "never installed" (install away)
// from "user removed the markers and doesn't want them back".
//
// The opt-out signal is a `.no-hooks` sentinel inside the hero
// workspace (e.g. `.hero/.no-hooks`). `hero init`-style flows treat
// marker-absence as "fresh setup" and install; `hero install` is
// re-runnable, so it needs a durable opt-out that survives across
// invocations. Users opt out by either (a) passing `--no-hooks` to
// the install command or (b) `touch .hero/.no-hooks`.
func hookInstallOptedOut(projectRoot string) bool {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return false
	}
	sentinel := filepath.Join(cfg.HeroDir(projectRoot), ".no-hooks")
	_, err = os.Stat(sentinel)
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
		// Stage each projected handoff path independently. A single
		// combined `git add -- a b c` aborts the WHOLE add (and stages
		// nothing) the moment any pathspec matches no file — e.g. a
		// dropped QUEUE.md or an empty `.hero/next/*.md` glob in solo
		// mode. Looping per-path isolates that failure so the paths that
		// DO exist still travel. The 2>/dev/null || true swallows the
		// no-match case per path. Spec: next-unconditional-commit-staging.
		stage = fmt.Sprintf(`
  # Re-stage projected handoff files so they travel with the commit,
  # one path at a time so a missing path (dropped QUEUE.md, no per-user
  # file in solo mode) doesn't abort staging of the others.
  for p in %s; do
    git add -- "$p" 2>/dev/null || true
  done`, strings.Join(handoffFilePaths, " "))
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

// uninstallNextHooks strips the hero-next managed blocks from
// .git/hooks/pre-commit, .git/hooks/post-merge, and .gitattributes.
// It also clears any legacy merge.hero-next.* entry left in
// .git/config by older installs (Hero no longer registers a custom
// merge driver — projected files use the built-in merge=union — but
// existing clones may still carry the orphaned stanza).
// Idempotent — running it twice or against a never-installed repo is
// a no-op. Returns the list of paths actually modified (or that had
// the legacy driver registration cleared), suitable for caller print-out.
func uninstallNextHooks(projectRoot string) (removed []string, err error) {
	gitDir, gerr := resolveGitDir(projectRoot)
	if gerr != nil {
		// Not a git repo — nothing to uninstall.
		return nil, nil
	}

	// Hook files: pre-commit, post-merge.
	for _, name := range []string{"pre-commit", "post-merge"} {
		path := filepath.Join(gitDir, "hooks", name)
		changed, rerr := stripManagedBlockFromHook(path)
		if rerr != nil {
			return removed, fmt.Errorf("strip %s: %w", name, rerr)
		}
		if changed {
			removed = append(removed, path)
		}
	}

	// .gitattributes: strip block; remove file when empty after strip.
	gaPath := filepath.Join(projectRoot, ".gitattributes")
	changed, rerr := stripGitAttributesBlock(gaPath)
	if rerr != nil {
		return removed, fmt.Errorf("strip .gitattributes: %w", rerr)
	}
	if changed {
		removed = append(removed, gaPath)
	}

	// .git/config: clear any legacy merge.hero-next.* driver stanza
	// left by older installs. Best-effort — ignore "key not found"
	// errors so a never-installed (or already-cleaned) repo doesn't fail.
	driverCleared := false
	for _, key := range []string{
		"merge." + mergeDriverName + ".driver",
		"merge." + mergeDriverName + ".name",
	} {
		cmd := exec.Command("git", "-C", projectRoot, "config", "--unset-all", key)
		if out, cerr := cmd.CombinedOutput(); cerr != nil {
			// git config returns exit 5 when the section/key doesn't
			// exist. Treat that as a successful no-op; bubble anything
			// else with the output for debugging.
			if !isGitConfigKeyMissing(cerr, out) {
				return removed, fmt.Errorf("git config --unset-all %s: %w\n%s", key, cerr, out)
			}
			continue
		}
		driverCleared = true
	}
	if driverCleared {
		removed = append(removed, filepath.Join(gitDir, "config"))
	}
	return removed, nil
}

// isGitConfigKeyMissing identifies the "key not found" exit from
// `git config --unset-all`. git uses exit 5 for that condition; any
// other non-zero exit indicates a real error.
func isGitConfigKeyMissing(err error, _ []byte) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode() == 5
	}
	return false
}

// stripManagedBlockFromHook removes the hero-next managed block from
// the hook file at path. If the resulting body is just the
// "#!/usr/bin/env bash\nset -e" shebang we wrote at install time (or
// is otherwise empty), the file is removed. Returns (changed, error)
// where changed is true if anything was modified.
func stripManagedBlockFromHook(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	src := string(data)
	startIdx := strings.Index(src, hookMarkerStart)
	if startIdx < 0 {
		// No managed block here — nothing to do.
		return false, nil
	}
	endIdx := strings.Index(src, hookMarkerEnd)
	var stripped string
	if endIdx < 0 {
		// Truncated previous block — drop from start to EOF; better
		// than leaving the open marker dangling.
		stripped = strings.TrimRight(src[:startIdx], "\n")
	} else {
		endIdx += len(hookMarkerEnd)
		prefix := strings.TrimRight(src[:startIdx], "\n")
		suffix := strings.TrimLeft(src[endIdx:], "\n")
		switch {
		case prefix == "" && suffix == "":
			stripped = ""
		case prefix == "":
			stripped = suffix
		case suffix == "":
			stripped = prefix
		default:
			stripped = prefix + "\n\n" + suffix
		}
	}
	stripped = strings.TrimRight(stripped, "\n")

	// If the remaining body is just the install-time shebang / set -e
	// with no user content, remove the file entirely so we don't leave
	// behind a dead hook stub.
	if isHookStubOnly(stripped) {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return false, err
		}
		return true, nil
	}
	// Preserve a trailing newline for tidiness.
	if stripped != "" {
		stripped += "\n"
	}
	if err := os.WriteFile(path, []byte(stripped), 0o755); err != nil {
		return false, err
	}
	return true, nil
}

// isHookStubOnly reports whether the post-strip hook body is just the
// shebang and an optional `set -e` we installed at write time. Such a
// stub serves no purpose and is removed so uninstall leaves no trace.
func isHookStubOnly(body string) bool {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "#!") {
			continue
		}
		if l == "set -e" {
			continue
		}
		return false
	}
	return true
}

// stripGitAttributesBlock removes the hero-next merge-driver block from
// the .gitattributes file at path. If the resulting file is empty, the
// file is removed entirely (preserving user-content files intact).
// Returns (changed, error).
func stripGitAttributesBlock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	src := string(data)
	startIdx := strings.Index(src, gaMarkerStart)
	if startIdx < 0 {
		return false, nil
	}
	endIdx := strings.Index(src, gaMarkerEnd)
	var stripped string
	if endIdx < 0 {
		stripped = strings.TrimRight(src[:startIdx], "\n")
	} else {
		endIdx += len(gaMarkerEnd)
		prefix := strings.TrimRight(src[:startIdx], "\n")
		suffix := strings.TrimLeft(src[endIdx:], "\n")
		switch {
		case prefix == "" && suffix == "":
			stripped = ""
		case prefix == "":
			stripped = suffix
		case suffix == "":
			stripped = prefix
		default:
			stripped = prefix + "\n\n" + suffix
		}
	}
	stripped = strings.TrimRight(stripped, "\n")
	if stripped == "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return false, err
		}
		return true, nil
	}
	stripped += "\n"
	if err := os.WriteFile(path, []byte(stripped), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// nextMergeDriverRegistered reports whether a legacy hero-next merge
// driver stanza is still present in .git/config. Hero no longer
// registers a custom driver (projected files use the built-in
// merge=union), so this only ever returns true for clones carrying an
// orphaned entry from an older install. Used by `hero hooks status`.
func nextMergeDriverRegistered(projectRoot string) bool {
	cmd := exec.Command("git", "-C", projectRoot, "config", "--get", "merge."+mergeDriverName+".driver")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// updateGitAttributes ensures .gitattributes contains the marker-
// bounded merge directive. Idempotent.
//
// Binds the projected handoff paths to git's BUILT-IN "union" merge
// driver. Unlike a custom driver — whose definition would live only in
// per-clone .git/config and never travel with the repo — "union" needs
// no config registration, so fresh clones, CI checkouts, and not-yet-
// installed teammates resolve these merges marker-free. Union
// concatenates both sides; the next `hero next checkpoint` total-
// overwrites the result from the graph (see checkpoint.go:
// writeProjectedNextMD).
func updateGitAttributes(projectRoot string) error {
	path := filepath.Join(projectRoot, ".gitattributes")
	existing, _ := os.ReadFile(path)

	var lines strings.Builder
	for _, p := range handoffFilePaths {
		lines.WriteString(p)
		lines.WriteString(" merge=union\n")
	}
	managed := fmt.Sprintf("%s\n%s%s", gaMarkerStart, lines.String(), gaMarkerEnd)

	body := mergeMarkerBlock(string(existing), gaMarkerStart, gaMarkerEnd, managed)
	return os.WriteFile(path, []byte(body), 0o644)
}
