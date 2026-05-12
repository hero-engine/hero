package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	stampScopeAll       bool
	stampScopeDryRun    bool
	stampScopeFromCwd   bool
	stampScopeOverwrite bool
)

var specStampScopeCmd = &cobra.Command{
	Use:   "stamp-scope [<slug-or-path>]",
	Short: "Stamp the active subproject scope into a spec's frontmatter",
	Long: `Add or update the subproject: frontmatter field on a spec.

Without arguments, stamps the spec at cwd's active scope onto every
spec that doesn't currently declare one. With a slug or path, stamps
just that spec.

The active scope is computed from cwd against .hero/subprojects.json,
the same way 'hero note' and other artifact-creating commands stamp
new artifacts.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSpecStampScope,
}

func init() {
	specStampScopeCmd.Flags().BoolVar(&stampScopeAll, "all", false, "process every spec in the workspace")
	specStampScopeCmd.Flags().BoolVar(&stampScopeDryRun, "dry-run", false, "report what would change without writing")
	specStampScopeCmd.Flags().BoolVar(&stampScopeFromCwd, "from-cwd", false, "use the active cwd scope rather than per-spec auto-detection")
	specStampScopeCmd.Flags().BoolVar(&stampScopeOverwrite, "overwrite", false, "replace existing subproject: values rather than skipping them")
	specCmd.AddCommand(specStampScopeCmd)
}

func runSpecStampScope(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	subs, _ := install.LoadSubprojects(heroDir)
	declared := []string{}
	if subs != nil {
		declared = subs.DeclaredPaths()
	}

	cwdScope := ""
	if stampScopeFromCwd {
		cwdScope = resolveActiveScope(projectRoot, heroDir)
		if cwdScope == "" {
			return fmt.Errorf("--from-cwd: cwd is at root or under no declared subproject; nothing to stamp")
		}
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	// Filter to a single spec if a slug or path was given.
	if len(args) == 1 && !stampScopeAll {
		match := args[0]
		filtered := make([]*spec.Spec, 0, 1)
		for _, s := range specs {
			if s.Slug == match || s.Path == match || strings.HasSuffix(s.Path, match) {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("no spec matched %q", match)
		}
		specs = filtered
	} else if !stampScopeAll && len(args) == 0 {
		return fmt.Errorf("specify a slug, a path, or --all")
	}

	stamped := 0
	skipped := 0
	for _, s := range specs {
		want := cwdScope
		if !stampScopeFromCwd {
			// Auto-detect from spec path: where does it live, what
			// declared subproject prefix matches it?
			//
			// Specs almost always live under .hero/planning/, so this
			// is mainly useful when a spec was moved/migrated and its
			// path encodes the originating subproject.
			want = inferScopeFromSpecPath(s.Path, projectRoot, declared)
		}
		if want == "" {
			skipped++
			continue
		}
		if s.Subproject == want {
			skipped++
			continue
		}
		if s.Subproject != "" && !stampScopeOverwrite {
			skipped++
			continue
		}
		if err := writeSubprojectFrontmatter(s.Path, want, stampScopeDryRun); err != nil {
			fmt.Fprintf(os.Stderr, "  error: %s: %v\n", s.Slug, err)
			continue
		}
		action := "stamped"
		if stampScopeDryRun {
			action = "would stamp"
		}
		fmt.Printf("  %s %s -> subproject: %s\n", action, s.Slug, want)
		stamped++
	}

	if stampScopeDryRun {
		fmt.Printf("\ndry run — %d would change, %d skipped\n", stamped, skipped)
	} else {
		fmt.Printf("\n%d stamped, %d skipped\n", stamped, skipped)
	}
	return nil
}

// inferScopeFromSpecPath returns the longest declared subproject prefix
// of the spec's directory relative to the project root. Used when
// auto-stamping and the spec path encodes the originating subproject
// (e.g. legacy migrated specs).
func inferScopeFromSpecPath(specPath, projectRoot string, declared []string) string {
	rel, err := filepath.Rel(projectRoot, filepath.Dir(specPath))
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	return workspace.MatchScope(projectRoot, filepath.Join(projectRoot, filepath.FromSlash(rel)), declared)
}

// writeSubprojectFrontmatter rewrites a spec.md file to add or replace
// its `subproject:` frontmatter line. Frontmatter must already exist
// (delimited by ---). On dryRun, the file is not modified.
func writeSubprojectFrontmatter(path, scope string, dryRun bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("no frontmatter")
	}

	// Locate frontmatter close.
	rest := content[4:]
	closeIdx := strings.Index(rest, "\n---")
	if closeIdx < 0 {
		return fmt.Errorf("frontmatter not closed")
	}
	frontmatter := rest[:closeIdx]
	body := rest[closeIdx:]

	// Split into lines and replace or append subproject: line.
	scanner := bufio.NewScanner(strings.NewReader(frontmatter))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []string
	replaced := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "subproject:") {
			out = append(out, fmt.Sprintf("subproject: %s", scope))
			replaced = true
			continue
		}
		out = append(out, line)
	}
	if !replaced {
		out = append(out, fmt.Sprintf("subproject: %s", scope))
	}

	newFrontmatter := strings.Join(out, "\n")
	newContent := "---\n" + newFrontmatter + body

	if dryRun {
		return nil
	}
	return os.WriteFile(path, []byte(newContent), 0o644)
}
