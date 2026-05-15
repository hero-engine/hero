package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/workspace"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Print runtime context blocks for prompt injection",
	Long: `Subcommands here emit small, machine-friendly context blocks
intended to be prepended to slash-command prompts so the model knows
the runtime state of the workspace (active subproject scope, etc).

Each subcommand exits 0 with empty output when there is nothing
relevant to inject — so a slash-command template can call them
unconditionally without needing if/else logic.`,
}

var contextScopeCmd = &cobra.Command{
	Use:   "scope",
	Short: "Print the active subproject scope as a prompt preamble",
	Long: `Emit a multi-line block describing the active workspace and
subproject scope. Used by slash-command templates that create
artifacts so the model can stamp `+"`subproject:`"+` frontmatter
without the user having to remember.

Quiet (no output, exit 0) when running at the workspace root with no
active scope.`,
	RunE: runContextScope,
}

// contextImportsCmd surfaces peer-owned contract imports across
// changed files. Phase 3 of cross-repo-peering: pure passive signal,
// no blocking, no prompts.
//
// Wired into the slash-command resume / context flow alongside
// `hero context scope`. Quiet (no output, exit 0) when no changed
// file imports a peer-owned contract symbol — so the slash-command
// template can call it unconditionally.
var (
	contextImportsFiles []string
)

var contextImportsCmd = &cobra.Command{
	Use:   "imports",
	Short: "Surface peer-owned contract symbols imported by changed files",
	Long: `Scans changed files in the working tree for Go imports of contract
symbols listed in any configured peer's manifest. When a match is
found, prints a one-line signal naming the symbol, the owning peer,
the governing convention, and the last-changed commit on the peer
side.

This is a passive context signal — it never blocks, prompts, or
auto-triggers a peer call. Use it as a hint that the change you're
making crosses a repo boundary.

By default scans the dirty files reported by ` + "`git status`" + `. Pass
--files to override (comma-separated paths, repeatable).`,
	RunE: runContextImports,
}

func init() {
	contextCmd.AddCommand(contextScopeCmd)
	contextCmd.AddCommand(contextImportsCmd)
	contextImportsCmd.Flags().StringSliceVar(&contextImportsFiles, "files", nil, "files to scan (defaults to git-status dirty set)")
	rootCmd.AddCommand(contextCmd)
}

func runContextImports(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	files := contextImportsFiles
	if len(files) == 0 {
		files = append(files, args...)
	}
	if len(files) == 0 {
		files = autoFocus(projectRoot)
	}
	if len(files) == 0 {
		return nil
	}
	hits, err := peering.ScanContractImports(projectRoot, peering.ScanOptions{
		ChangedFiles: files,
	})
	if err != nil {
		// Stay quiet on scan errors — passive signal, not a guardrail.
		return nil
	}
	signal := peering.RenderContractImportSignal(hits)
	if signal == "" {
		return nil
	}
	fmt.Fprint(cmd.OutOrStdout(), signal)
	return nil
}

func runContextScope(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	ws, err := workspace.Locate(cwd)
	if err != nil {
		return nil
	}
	heroDir := filepath.Join(ws.Root, ".hero")
	subs, _ := install.LoadSubprojects(heroDir)
	var declared []string
	if subs != nil {
		declared = subs.DeclaredPaths()
	}
	scope := ws.Scope(declared)

	if ws.CWD == ws.Root && scope == workspace.RootScope {
		return nil
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "<!-- hero:context-scope -->")
	fmt.Fprintf(out, "WORKSPACE: %s\n", ws.Root)
	if scope != workspace.RootScope {
		fmt.Fprintf(out, "SCOPE:     %s\n", scope)
	}
	if ws.IsSatellite {
		fmt.Fprintf(out, "SATELLITE: yes (%s)\n", relPath(ws.Root, ws.SatellitePath))
	} else {
		fmt.Fprintf(out, "SATELLITE: no (cwd: %s)\n", relPath(ws.Root, ws.CWD))
	}
	fmt.Fprintln(out)
	if scope != workspace.RootScope {
		fmt.Fprintf(out, "Any spec, knowledge, or note you create in this session must include\n")
		fmt.Fprintf(out, "`subproject: %s` in its YAML frontmatter, and must be written under\n", scope)
		fmt.Fprintf(out, "`%s/.hero/...` regardless of the cwd this session is running in.\n", ws.Root)
	} else {
		fmt.Fprintf(out, "No subproject scope is active. Artifacts created in this session\n")
		fmt.Fprintf(out, "go to the workspace root (no `subproject:` field needed).\n")
	}
	return nil
}

// relPath returns the path of target relative to base, falling back to
// the absolute path if a relative form can't be computed.
func relPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}
