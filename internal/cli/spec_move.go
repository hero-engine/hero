package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var (
	moveToScope  string
	moveRelocate bool
	moveDryRun   bool
)

var specMoveCmd = &cobra.Command{
	Use:   "move <slug>",
	Short: "Move a spec to a different subproject scope",
	Long: `Update the spec's subproject: frontmatter to a new scope, optionally
relocating the file under a scope-prefixed path, then re-index and
emit a subproject_changed event.

Examples:
  hero spec move some-feature --to-scope engines/cuda
  hero spec move some-feature --to-scope ""           # re-root (remove scope)
  hero spec move some-feature --to-scope engines/cuda --relocate
  hero spec move some-feature --to-scope engines/cuda --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runSpecMove,
}

func init() {
	specMoveCmd.Flags().StringVar(&moveToScope, "to-scope", "", "target subproject scope (must be declared in subprojects.json or empty for root)")
	specMoveCmd.Flags().BoolVar(&moveRelocate, "relocate", false, "move the spec file to a scope-prefixed path under .hero/planning/<bucket>/<scope>/<slug>/")
	specMoveCmd.Flags().BoolVar(&moveDryRun, "dry-run", false, "report changes without writing")
	specCmd.AddCommand(specMoveCmd)
}

func runSpecMove(cmd *cobra.Command, args []string) error {
	if !cmd.Flags().Changed("to-scope") {
		return fmt.Errorf("--to-scope is required (use --to-scope \"\" to re-root)")
	}
	slug := strings.TrimSpace(args[0])
	if slug == "" {
		return fmt.Errorf("slug is required")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Resolve target scope: empty / "root" → root scope; otherwise must
	// be declared in subprojects.json.
	subs, err := install.LoadSubprojects(heroDir)
	if err != nil {
		return fmt.Errorf("loading subprojects: %w", err)
	}
	targetScope := strings.TrimSpace(moveToScope)
	if targetScope == "root" {
		targetScope = ""
	}
	if targetScope != "" && !subs.IsDeclared(targetScope) {
		declared := subs.DeclaredPaths()
		if len(declared) == 0 {
			return fmt.Errorf("target scope %q is not declared and no subprojects exist; declare it in .hero/subprojects.json first", targetScope)
		}
		return fmt.Errorf("target scope %q is not declared; available scopes: %s",
			targetScope, strings.Join(declared, ", "))
	}

	// Find the spec.
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == slug {
			target = s
			break
		}
	}
	if target == nil {
		return fmt.Errorf("no spec with slug %q", slug)
	}

	oldScope := target.Subproject
	if oldScope == targetScope {
		fmt.Printf("Spec %q is already in scope %q. No change.\n", slug, formatScope(targetScope))
		return nil
	}

	// Plan the path move (if --relocate).
	var newPath string
	if moveRelocate {
		newPath = relocatedPath(target.Path, projectRoot, targetScope)
	}

	if moveDryRun {
		fmt.Printf("Would update %s\n", target.Path)
		fmt.Printf("  subproject: %s -> %s\n", formatScope(oldScope), formatScope(targetScope))
		if moveRelocate && newPath != target.Path {
			fmt.Printf("  relocate:   %s -> %s\n", target.Path, newPath)
		}
		fmt.Println("(dry-run — no changes written)")
		return nil
	}

	// Apply: rewrite frontmatter, then optionally relocate.
	if err := writeSubprojectFrontmatter(target.Path, targetScope, false); err != nil {
		return fmt.Errorf("rewrite frontmatter: %w", err)
	}
	if moveRelocate && newPath != target.Path {
		if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
			return fmt.Errorf("create destination dir: %w", err)
		}
		if err := moveSpecFile(projectRoot, target.Path, newPath); err != nil {
			return fmt.Errorf("relocate: %w", err)
		}
		target.Path = newPath
	}

	// Re-index and emit event.
	if _, err := index.RefreshIfStale(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: index refresh failed: %v\n", err)
	}
	emitSubprojectChanged(cfg, projectRoot, slug, oldScope, targetScope)

	fmt.Printf("Moved %s to scope %s\n", slug, formatScope(targetScope))
	if moveRelocate && newPath != "" {
		fmt.Printf("  file: %s\n", newPath)
	}
	return nil
}

// formatScope renders a scope id for human display, surfacing "(root)"
// when the scope is empty.
func formatScope(s string) string {
	if s == "" {
		return "(root)"
	}
	return s
}

// relocatedPath computes the scope-prefixed destination for a spec
// file. Scheme: .hero/planning/<bucket>/<scope-segments...>/<slug>/spec.md.
// When targetScope is empty (root), the path returns to the
// non-scope-prefixed shape.
func relocatedPath(currentPath, projectRoot, targetScope string) string {
	rel, err := filepath.Rel(projectRoot, currentPath)
	if err != nil {
		return currentPath
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	// Expected shape: .hero/planning/<bucket>/<scope?>/.../<slug>/spec.md
	// Find ".hero" + "planning" + bucket; everything between bucket and the last
	// (slug, spec.md) pair is treated as old scope segments to replace.
	if len(parts) < 5 || parts[0] != ".hero" || parts[1] != "planning" || parts[len(parts)-1] != "spec.md" {
		return currentPath
	}
	bucket := parts[2]
	slugDir := parts[len(parts)-2]
	newParts := []string{".hero", "planning", bucket}
	if targetScope != "" {
		newParts = append(newParts, strings.Split(targetScope, "/")...)
	}
	newParts = append(newParts, slugDir, "spec.md")
	return filepath.Join(projectRoot, filepath.FromSlash(strings.Join(newParts, "/")))
}

// moveSpecFile uses git mv when the source is tracked, falling back to
// os.Rename. Same pattern as the migration apply path.
func moveSpecFile(projectRoot, src, dst string) error {
	if isGitTracked(projectRoot, src) {
		cmd := exec.Command("git", "-C", projectRoot, "mv", src, dst)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git mv: %v: %s", err, string(out))
		}
		return nil
	}
	return os.Rename(src, dst)
}

func isGitTracked(projectRoot, path string) bool {
	cmd := exec.Command("git", "-C", projectRoot, "ls-files", "--error-unmatch", path)
	return cmd.Run() == nil
}

// emitSubprojectChanged appends a feed event recording the scope change.
func emitSubprojectChanged(cfg config.Config, projectRoot, slug, oldScope, newScope string) {
	logPath := filepath.Join(cfg.HeroDir(projectRoot), "events.log")
	agent := os.Getenv("HERO_AGENT")
	if agent == "" {
		agent = "human/" + gitUserName()
	}
	evt := feed.FeedEvent{
		Type:       "subproject_changed",
		Agent:      agent,
		Slug:       slug,
		Subproject: newScope,
		Message:    fmt.Sprintf("scope %s -> %s", formatScope(oldScope), formatScope(newScope)),
	}
	_ = feed.AppendEvent(logPath, evt)
}
