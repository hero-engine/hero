package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hookScriptTemplate is the shell script template for a git hook.
// It appends a call to `hero hook <event>` that passes all args.
//
// Most hooks intentionally swallow exit codes ("|| true") so a hero
// failure never breaks git. pre-commit is the exception: when
// hooks.status_truth is enabled in hero.json, the pre-commit hook
// must propagate a non-zero exit so the commit is blocked. The
// template for pre-commit drops the "|| true" suffix.
const hookScriptTemplate = `#!/bin/sh
# Hero git hook — %s
# Installed by: hero hooks install
# Remove with: hero hooks uninstall
hero hook %s "$@" 2>/dev/null || true
`

const hookScriptTemplateBlocking = `#!/bin/sh
# Hero git hook — %s
# Installed by: hero hooks install
# Remove with: hero hooks uninstall
# This hook can BLOCK the git operation when hero hook returns non-zero.
hero hook %s "$@"
`

// templateForHook returns the right script template for a hook name.
// pre-commit uses the blocking template so exit codes propagate.
func templateForHook(name string) string {
	if name == "pre-commit" {
		return hookScriptTemplateBlocking
	}
	return hookScriptTemplate
}

// heroMarker is used to detect Hero-managed sections in hook scripts.
const heroMarker = "# Hero git hook"

// supportedHooks are the git hooks Hero manages.
var supportedHooks = []string{
	"post-checkout",
	"post-merge",
	"post-commit",
	"prepare-commit-msg",
	"pre-commit",
}

// InstalledHook describes a currently installed hook.
type InstalledHook struct {
	Name    string
	Path    string
	HasHero bool   // true if hook script contains Hero's marker
	Version string // "hero hooks install" if Hero-managed, "" otherwise
}

// Install installs Hero git hooks into .git/hooks/.
// If a hook file already exists and doesn't contain Hero's marker,
// it appends Hero's call rather than overwriting.
// Supported hooks: post-checkout, post-merge, post-commit, prepare-commit-msg
func Install(gitDir string) error {
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	for _, name := range supportedHooks {
		hookPath := filepath.Join(hooksDir, name)

		data, err := os.ReadFile(hookPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reading hook %s: %w", name, err)
		}

		if os.IsNotExist(err) {
			// Write full template
			content := fmt.Sprintf(templateForHook(name), name, name)
			if err := os.WriteFile(hookPath, []byte(content), 0o755); err != nil {
				return fmt.Errorf("writing hook %s: %w", name, err)
			}
			continue
		}

		existing := string(data)
		if strings.Contains(existing, heroMarker) {
			// Already installed — skip
			continue
		}

		// Append Hero's call to existing hook. pre-commit drops the
		// "|| true" so the commit can be blocked; other hooks keep it
		// so a hero failure never breaks git.
		var appendLine string
		if name == "pre-commit" {
			appendLine = fmt.Sprintf("\nhero hook %s \"$@\"\n", name)
		} else {
			appendLine = fmt.Sprintf("\nhero hook %s \"$@\" 2>/dev/null || true\n", name)
		}
		updated := existing
		if !strings.HasSuffix(updated, "\n") {
			updated += "\n"
		}
		updated += appendLine

		if err := os.WriteFile(hookPath, []byte(updated), 0o755); err != nil {
			return fmt.Errorf("updating hook %s: %w", name, err)
		}
	}

	return nil
}

// Uninstall removes Hero's additions from .git/hooks/ files.
// If Hero's lines are the only content, removes the file.
// Otherwise, strips only the Hero block.
func Uninstall(gitDir string) error {
	hooksDir := filepath.Join(gitDir, "hooks")

	for _, name := range supportedHooks {
		hookPath := filepath.Join(hooksDir, name)

		data, err := os.ReadFile(hookPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("reading hook %s: %w", name, err)
		}

		content := string(data)
		if !strings.Contains(content, heroMarker) {
			continue
		}

		stripped := stripHeroBlock(content, name)

		// If remaining content is just a shebang or empty, remove the file
		trimmed := strings.TrimSpace(stripped)
		if trimmed == "" || trimmed == "#!/bin/sh" || trimmed == "#!/bin/bash" {
			if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing hook %s: %w", name, err)
			}
			continue
		}

		if err := os.WriteFile(hookPath, []byte(stripped), 0o755); err != nil {
			return fmt.Errorf("updating hook %s: %w", name, err)
		}
	}

	return nil
}

// stripHeroBlock removes the Hero-managed block from hook content.
// It removes lines from "# Hero git hook — <name>" through the heroMarker block,
// as well as any bare "hero hook <name> ..." lines appended to existing hooks.
func stripHeroBlock(content, name string) string {
	lines := strings.Split(content, "\n")
	var result []string
	skip := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Detect the start of the Hero block (full template header)
		if trimmed == fmt.Sprintf("# Hero git hook — %s", name) {
			skip = true
			continue
		}

		if skip {
			// Skip lines until we've consumed the hero hook invocation line
			if strings.HasPrefix(trimmed, "hero hook "+name) {
				skip = false
			}
			continue
		}

		// Also strip bare appended lines (when Hero was appended to an
		// existing hook). Both swallow-form (most hooks) and blocking-
		// form (pre-commit) need to match for uninstall to be clean.
		if trimmed == fmt.Sprintf("hero hook %s \"$@\" 2>/dev/null || true", name) {
			continue
		}
		if trimmed == fmt.Sprintf("hero hook %s \"$@\"", name) {
			continue
		}

		result = append(result, line)
	}

	// Clean up extra blank lines at end
	joined := strings.Join(result, "\n")
	// Normalize trailing newline
	joined = strings.TrimRight(joined, "\n")
	if joined != "" {
		joined += "\n"
	}
	return joined
}

// Status returns the status of Hero hooks in .git/hooks/.
func Status(gitDir string) ([]InstalledHook, error) {
	hooksDir := filepath.Join(gitDir, "hooks")
	var result []InstalledHook

	for _, name := range supportedHooks {
		hookPath := filepath.Join(hooksDir, name)
		hook := InstalledHook{
			Name: name,
			Path: hookPath,
		}

		data, err := os.ReadFile(hookPath)
		if err != nil {
			if os.IsNotExist(err) {
				result = append(result, hook)
				continue
			}
			return nil, fmt.Errorf("reading hook %s: %w", name, err)
		}

		content := string(data)
		if strings.Contains(content, heroMarker) {
			hook.HasHero = true
			hook.Version = "hero hooks install"
		}

		result = append(result, hook)
	}

	return result, nil
}
