package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// auto_sync.go — keeps multi-harness installs at the same binary
// version. When `hero install --target X` runs against a project
// where other harnesses are also installed, the auto-sync step
// refreshes those siblings through the same per-target installer.
// Drift between harness copies (the original motivation for the
// canonical-symlink architecture) is prevented behaviorally instead
// of structurally.

// autoSyncSiblings detects other installed harnesses in
// opts.TargetDir and invokes Run for each one with AutoSyncTargets
// disabled (so we don't recurse) and TrustedChecksums populated from
// the project's version info (so prior-version content can refresh
// without --force).
func autoSyncSiblings(opts Options, result *Result) error {
	siblings, err := detectInstalledTargetDirs(opts.TargetDir, opts.Target)
	if err != nil {
		return err
	}
	if len(siblings) == 0 {
		return nil
	}

	progressf(opts, "  auto-sync siblings: %v\n", siblings)
	for _, t := range siblings {
		sub := opts
		sub.Target = t
		sub.AutoSyncTargets = false  // prevent recursion
		sub.SkipCanonicalRender = true // legacy cleanup already ran once this turn
		if _, subErr := Run(sub); subErr != nil {
			fmt.Fprintf(os.Stderr, "  warning: auto-sync %s failed: %v\n", t, subErr)
			// Continue; primary install already succeeded.
			continue
		}
		result.Merged = append(result.Merged, fmt.Sprintf("auto-sync:%s", t))
	}
	return nil
}

// DetectFirstInstalledTarget returns the first installed-harness target
// detected in projectDir, or an empty Target if none. Used by the
// install command's --migrate fallback to pick a primary target when
// the user didn't pass --target.
func DetectFirstInstalledTarget(projectDir string) (Target, error) {
	candidates, err := detectInstalledTargetDirs(projectDir, "")
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", nil
	}
	return candidates[0], nil
}

// detectInstalledTargetDirs scans projectDir for harness destBase
// directories. Returns the set of installed targets EXCLUDING
// excludeTarget (the one we just installed).
func detectInstalledTargetDirs(projectDir string, excludeTarget Target) ([]Target, error) {
	candidates := []struct {
		target Target
		path   string
	}{
		{TargetClaude, ".claude"},
		{TargetOpenCode, ".opencode"},
		{TargetCursor, filepath.Join(".cursor", "rules")},
		{TargetCodex, ".codex"},
		{TargetCopilot, filepath.Join(".github", "copilot-instructions.md")},
		{TargetGeneric, ".ai"},
	}
	var found []Target
	for _, c := range candidates {
		if c.target == excludeTarget {
			continue
		}
		full := filepath.Join(projectDir, c.path)
		// Either a dir (most targets) or a regular file (copilot's marker
		// instructions file) confirms the install.
		if _, err := os.Lstat(full); err != nil {
			continue
		}
		found = append(found, c.target)
	}
	return found, nil
}
