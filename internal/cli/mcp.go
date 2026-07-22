package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:    "mcp",
	Short:  "Run as MCP stdio server (used by AI tools)",
	Hidden: true,
	Long: `Runs a JSON-RPC 2.0 server over stdin/stdout, intended to be launched
as a child process by AI coding tools (Claude Code, OpenCode, Cursor).

This command is not meant to be run manually. It is configured automatically
by 'hero install' and invoked by AI tools at session start.

The server exposes Hero tools (context, search, status, nudge, etc.) via the
Model Context Protocol (MCP). Auto-prime injects project context into the
session at connection time.`,
	RunE: runMCP,
}

var mcpProjectRoot string

func init() {
	mcpCmd.Flags().StringVar(&mcpProjectRoot, "project-root", "", "override project root (used by workspace installs)")
}

func runMCP(cmd *cobra.Command, args []string) error {
	projectRoot := mcpProjectRoot
	if projectRoot == "" {
		projectRoot = findProjectRoot()
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := filepath.Join(projectRoot, cfg.Folder)

	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("hero workspace not found at %s (run 'hero init' first)", heroDir)
	}

	version := rootCmd.Version
	if version == "" {
		version = "dev"
	}

	var filter *serve.ToolFilter
	if cfg.Serve != nil && cfg.Serve.ToolFilter != nil {
		filter = serve.NewToolFilter(cfg.Serve.ToolFilter)
	}

	mcpSrv := serve.NewMCPServerWithFilter(heroDir, projectRoot, version, filter)
	mcpSrv.SetContext(cmd.Context())
	return mcpSrv.Run()
}
