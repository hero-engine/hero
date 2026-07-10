package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/hooks"
	"github.com/hero-engine/hero/internal/integrity"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook <event>",
	Short: "Internal: handle a git hook event (called by hook scripts)",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runHook,
}

func runHook(cmd *cobra.Command, args []string) error {
	// Hook scripts must never cause a non-zero exit — recover from all
	// panics. The one event allowed to BLOCK git is pre-commit, and
	// only when hooks.status_truth is enabled in hero.json. Every
	// other event falls through to the always-zero exit at the bottom.
	defer func() {
		if r := recover(); r != nil {
			// Silently swallow panics
		}
	}()

	eventName := args[0]
	hookArgs := args[1:]

	projectRoot := findProjectRoot()

	cfg, err := config.Load(projectRoot)
	if err != nil {
		// Non-fatal: proceed with defaults
		cfg = config.DefaultConfig()
	}

	heroDir := cfg.HeroDir(projectRoot)

	// Discover all spec slugs
	allSpecs, err := spec.Discover(heroDir)
	if err != nil {
		// Non-fatal: no specs to match
		allSpecs = nil
	}

	slugs := make([]string, 0, len(allSpecs))
	for _, s := range allSpecs {
		slugs = append(slugs, s.Slug)
	}

	// Determine branch patterns from config
	patterns := hooks.DefaultBranchPatterns
	if cfg.Hooks != nil && len(cfg.Hooks.BranchPatterns) > 0 {
		patterns = cfg.Hooks.BranchPatterns
	}

	eventsLogPath := filepath.Join(heroDir, "events.log")

	switch eventName {
	case "post-checkout":
		slug, oldStatus, newStatus := hooks.HandlePostCheckout(hookArgs, slugs, patterns)
		if slug != "" {
			updateSpecStatus(allSpecs, slug, oldStatus, newStatus)
			_ = hooks.LogEvent(eventsLogPath, map[string]string{
				"event":      "post-checkout",
				"slug":       slug,
				"old_status": oldStatus,
				"new_status": newStatus,
			})
		}

	case "post-merge":
		slug := hooks.HandlePostMerge(hookArgs, slugs, patterns)
		if slug != "" {
			updateSpecStatus(allSpecs, slug, "delivering", "completed")
			_ = hooks.LogEvent(eventsLogPath, map[string]string{
				"event":      "post-merge",
				"slug":       slug,
				"old_status": "delivering",
				"new_status": "completed",
			})
		}

	case "post-commit":
		// Refresh NEXT.md machine block so cross-session handoff stays
		// fresh on every commit. This is the universal fallback for
		// host tools without a native end-of-turn hook (cursor, copilot,
		// generic). Errors are swallowed — hooks must never fail.
		_, _ = writeCheckpoint()

	case "prepare-commit-msg":
		if cfg.Hooks != nil && cfg.Hooks.InjectCommitPrefix {
			injectCommitPrefix(hookArgs, slugs, patterns)
		}

	case "pre-commit":
		// spec-status-integrity AC-5: opt-in pre-commit gate. Runs
		// hero check status against the AC graph. If any spec is
		// lying or partial, return a non-zero exit so git blocks the
		// commit. Off by default — only the user setting
		// hooks.status_truth = true asks to be gated.
		if cfg.Hooks == nil || !cfg.Hooks.StatusTruth {
			return nil
		}
		if err := runStatusTruthGate(heroDir, allSpecs); err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "  Run `hero check status` for details, or")
			fmt.Fprintln(os.Stderr, "  `hero check status --auto-fix` to downgrade lying frontmatter.")
			fmt.Fprintln(os.Stderr, "  Bypass with `git commit --no-verify` if you really must.")
			os.Exit(1) // BLOCK the commit
		}
	}

	// Always exit 0
	return nil
}

// runStatusTruthGate is the pre-commit gate. Opens the graph, runs
// the integrity check, and returns an error when any spec is lying
// or partial. Caller decides whether to propagate the error to a
// non-zero exit (only pre-commit does, per the spec).
func runStatusTruthGate(heroDir string, specs []*spec.Spec) error {
	store, err := graph.Open(heroDir)
	if err != nil {
		return nil // best-effort — graph unavailable means we can't gate
	}
	defer store.Close()
	report, err := integrity.CheckCompletedSpecs(specs, store)
	if err != nil {
		return nil // best-effort
	}
	if report.HasIssues() {
		return fmt.Errorf("status truthfulness: %d lying, %d partial",
			report.Lying, report.Partial)
	}
	return nil
}

// updateSpecStatus finds the spec by slug and updates its frontmatter status field.
func updateSpecStatus(specs []*spec.Spec, slug, oldStatus, newStatus string) {
	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == slug {
			target = s
			break
		}
	}
	if target == nil {
		return
	}

	data, err := os.ReadFile(target.Path)
	if err != nil {
		return
	}

	content := string(data)
	updated := replaceStatusInFrontmatter(content, oldStatus, newStatus)
	if updated == content {
		return
	}

	_ = os.WriteFile(target.Path, []byte(updated), 0o644)
}

// replaceStatusInFrontmatter replaces `status: <old>` with `status: <new>` in YAML frontmatter.
func replaceStatusInFrontmatter(content, oldStatus, newStatus string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	closedFrontmatter := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if i == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter && !closedFrontmatter {
			if trimmed == "---" {
				closedFrontmatter = true
				continue
			}
			if trimmed == "status: "+oldStatus {
				// Preserve indentation
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[i] = indent + "status: " + newStatus
				break
			}
		}
	}

	return strings.Join(lines, "\n")
}

// injectCommitPrefix prepends the spec slug to the commit message file.
// hookArgs[0] is the commit message file path.
func injectCommitPrefix(hookArgs []string, slugs []string, patterns []string) {
	if len(hookArgs) == 0 {
		return
	}
	msgFile := hookArgs[0]

	// Get current branch
	out, err := exec.Command("git", "symbolic-ref", "--short", "HEAD").Output()
	if err != nil {
		return
	}
	branch := strings.TrimSpace(string(out))

	slug, ok := hooks.MatchBranch(branch, patterns)
	if !ok {
		return
	}

	// Confirm the slug exists in the corpus
	found, ok := hooks.FindMatchingSpec(slug, slugs)
	if !ok {
		return
	}

	data, err := os.ReadFile(msgFile)
	if err != nil {
		return
	}

	msg := string(data)
	prefix := fmt.Sprintf("[%s] ", found)
	if strings.HasPrefix(msg, prefix) {
		return // already prefixed
	}

	_ = os.WriteFile(msgFile, []byte(prefix+msg), 0o644)
}
