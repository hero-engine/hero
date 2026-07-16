package install

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MCPServerConfig represents an MCP server entry for Cursor/Claude configs.
// Format: { "command": "hero", "args": ["mcp"] }
type MCPServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// OpenCodeMCPConfig represents an MCP server entry in OpenCode's config.
// Format: { "type": "local", "command": ["hero", "mcp"] }
type OpenCodeMCPConfig struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
}

// RegisterMCP adds the Hero MCP server to the agent's MCP configuration.
// It finds the hero binary path and writes the appropriate MCP config.
func RegisterMCP(target Target, opts Options) error {
	heroPath, err := findHeroBinary()
	if err != nil {
		return fmt.Errorf("finding hero binary: %w", err)
	}

	switch target {
	case TargetCursor:
		return registerMCPCursor(heroPath, opts)
	case TargetClaude:
		return registerMCPClaude(heroPath, opts)
	case TargetOpenCode:
		return registerMCPOpenCode(heroPath, opts)
	case TargetCodex:
		return registerMCPCodex(heroPath, opts)
	default:
		return nil
	}
}

// Test seams for findHeroBinary — overridden in tests to simulate
// resolution failure without disturbing the real environment.
var (
	osExecutable = os.Executable
	execLookPath = exec.LookPath
)

// findHeroBinary locates the hero binary to wire into MCP configs. Checks:
//  1. The running binary itself (os.Executable, symlinks resolved) — the
//     hero performing the install is the hero the config must point at,
//     not whichever hero happens to be first on the ambient PATH.
//  2. PATH lookup, only as a fallback when os.Executable fails.
//
// Returns an error when neither resolves: writing a config that points at
// a binary we could not locate would only defer the failure to the
// harness's MCP startup, silently.
func findHeroBinary() (string, error) {
	exe, exeErr := osExecutable()
	if exeErr == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			return resolved, nil
		}
		return exe, nil
	}

	path, lookErr := execLookPath("hero")
	if lookErr == nil {
		return path, nil
	}

	return "", fmt.Errorf("locating hero binary: os.Executable: %v; PATH lookup: %v", exeErr, lookErr)
}

// registerMCPCursor writes to .cursor/mcp.json in the project.
// Format: { "mcpServers": { "hero": { "command": "hero", "args": ["mcp"] } } }
func registerMCPCursor(heroPath string, opts Options) error {
	var configPath string
	switch opts.Mode {
	case ModeProject:
		configPath = filepath.Join(opts.TargetDir, ".cursor", "mcp.json")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(home, ".cursor", "mcp.json")
	}

	return upsertMCPConfig(configPath, heroPath, opts.DryRun, opts.ProjectRoot)
}

// registerMCPClaude writes to .mcp.json in the project root (project mode)
// or ~/.claude.json (global mode).
// Format: { "mcpServers": { "hero": { "command": "hero", "args": ["mcp"] } } }
func registerMCPClaude(heroPath string, opts Options) error {
	var configPath string
	switch opts.Mode {
	case ModeProject:
		configPath = filepath.Join(opts.TargetDir, ".mcp.json")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(home, ".claude.json")
	}

	return upsertMCPConfig(configPath, heroPath, opts.DryRun, opts.ProjectRoot)
}

// registerMCPOpenCode writes the hero MCP server to the OpenCode config.
// Project mode: <project>/opencode.json under the "mcp" key.
// Global mode: ~/.config/opencode/opencode.json under the "mcp" key.
// Format: { "mcp": { "hero": { "type": "local", "command": ["hero", "mcp"] } } }
func registerMCPOpenCode(heroPath string, opts Options) error {
	var configPath string
	switch opts.Mode {
	case ModeProject:
		configPath = filepath.Join(opts.TargetDir, "opencode.json")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(home, ".config", "opencode", "opencode.json")
	}

	return upsertOpenCodeMCPConfig(configPath, heroPath, opts.DryRun, opts.ProjectRoot)
}

// upsertMCPConfig reads an existing MCP config (or creates a new one) and ensures
// the hero server entry is present. Used for Cursor and Claude which use the
// { "mcpServers": { ... } } format.
func upsertMCPConfig(configPath, heroPath string, dryRun bool, projectRoot string) error {
	if dryRun {
		fmt.Printf("  MCP server -> %s\n", configPath)
		return nil
	}

	// Read existing config or start fresh
	config := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			// If the file is malformed, start fresh
			config = make(map[string]interface{})
		}
	}

	// Ensure mcpServers key exists
	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}

	// Add/update hero entry
	args := []string{"mcp"}
	if projectRoot != "" {
		args = append(args, "--project-root", projectRoot)
	}
	servers["hero"] = MCPServerConfig{
		Command: heroPath,
		Args:    args,
	}
	config["mcpServers"] = servers

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(configPath, data, 0o644)
}

// upsertOpenCodeMCPConfig reads an existing OpenCode config (or creates one) and
// ensures the hero MCP server entry is present under the "mcp" key.
// OpenCode format: { "mcp": { "hero": { "type": "local", "command": ["hero", "mcp"] } } }
func upsertOpenCodeMCPConfig(configPath, heroPath string, dryRun bool, projectRoot string) error {
	if dryRun {
		fmt.Printf("  MCP server -> %s\n", configPath)
		return nil
	}

	// Read existing config or start fresh
	config := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			// If the file is malformed, start fresh
			config = make(map[string]interface{})
		}
	}

	// Ensure mcp key exists
	mcp, ok := config["mcp"].(map[string]interface{})
	if !ok {
		mcp = make(map[string]interface{})
	}

	// Build the command array: ["/path/to/hero", "mcp"] or with --project-root for workspaces
	cmd := []string{heroPath, "mcp"}
	if projectRoot != "" {
		cmd = append(cmd, "--project-root", projectRoot)
	}

	// Add/update hero entry
	mcp["hero"] = OpenCodeMCPConfig{
		Type:    "local",
		Command: cmd,
	}
	config["mcp"] = mcp

	// Clean up legacy mcpServers key if present (from older hero installs)
	delete(config, "mcpServers")

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(configPath, data, 0o644)
}

// registerMCPCodex writes the hero MCP server block to Codex's machine-local
// User layer (~/.codex/config.toml) in BOTH project and global modes. Codex
// deep-merges config layers — User (~/.codex/config.toml, precedence 20) <
// Project (<repo>/.codex/, precedence 25), recursive per-key table merge
// (openai/codex: config_layer_source.rs, merge.rs) — so one User-layer block
// serves every Hero project on the machine (`hero mcp` resolves the workspace
// from the session's cwd), and the machine-specific absolute binary path
// never lands in a git-tracked project file.
//
// Exception: workspace mode (opts.ProjectRoot != "") pins the MCP server to a
// specific project root via `--project-root`. That value is project-specific,
// not machine-generic, so it stays in the workspace's own .codex/config.toml
// — Codex's Project layer, which overrides the User layer per-key.
//
// Format uses a hero:managed marker block so we can update without clobbering user content:
//
//	# hero:managed
//	[mcp_servers.hero]
//	command = "/path/to/hero"
//	args = ["mcp"]
//	# end:hero:managed
//
// Project mode also migrates older installs: if the project's
// .codex/config.toml still carries the managed block, exactly that span is
// removed. Bytes outside it are untouched, and the file is left in place —
// it is the user's file (their model/approval/other-MCP settings live
// there), and deleting a possibly git-tracked file from an installer would
// be a surprise.
func registerMCPCodex(heroPath string, opts Options) error {
	if opts.Mode == ModeProject && opts.ProjectRoot != "" {
		// Workspace mode: project-specific wiring belongs in the project layer.
		configPath := filepath.Join(opts.TargetDir, ".codex", "config.toml")
		return upsertCodexConfig(configPath, heroPath, opts.DryRun, opts.ProjectRoot)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".codex", "config.toml")

	if opts.Mode == ModeProject && !opts.DryRun {
		projectConfig := filepath.Join(opts.TargetDir, ".codex", "config.toml")
		migrated, err := removeCodexManagedBlock(projectConfig)
		if err != nil {
			return fmt.Errorf("removing hero block from %s: %w", projectConfig, err)
		}
		if migrated {
			fmt.Printf("  moved hero MCP block: %s -> %s (Codex User layer)\n", projectConfig, configPath)
		}
	}

	return upsertCodexConfig(configPath, heroPath, opts.DryRun, opts.ProjectRoot)
}

// removeCodexManagedBlock deletes Hero's `# hero:managed` … `# end:hero:managed`
// span (plus the single trailing newline Hero wrote after it) from the file
// at path. Every byte outside the span is left untouched, and the file is
// left in place even when only whitespace remains. Returns true when a
// block was found and removed.
func removeCodexManagedBlock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	s := string(data)
	start := strings.Index(s, codexMCPMarker)
	if start < 0 {
		return false, nil
	}
	end := strings.Index(s[start:], codexMCPEndMarker)
	if end < 0 {
		return false, nil
	}
	end += start + len(codexMCPEndMarker)
	if end < len(s) && s[end] == '\n' {
		end++
	}

	return true, os.WriteFile(path, []byte(s[:start]+s[end:]), 0o644)
}

const codexMCPMarker = "# hero:managed"
const codexMCPEndMarker = "# end:hero:managed"

// upsertCodexConfig reads an existing .codex/config.toml (or creates one) and
// ensures the hero MCP server block is present within hero:managed markers.
func upsertCodexConfig(configPath, heroPath string, dryRun bool, projectRoot string) error {
	if dryRun {
		fmt.Printf("  MCP server -> %s\n", configPath)
		return nil
	}

	args := `["mcp"]`
	if projectRoot != "" {
		args = fmt.Sprintf(`["mcp", "--project-root", %q]`, projectRoot)
	}

	heroBlock := fmt.Sprintf("%s\n[mcp_servers.hero]\ncommand = %q\nargs = %s\n%s",
		codexMCPMarker, heroPath, args, codexMCPEndMarker)

	existing := ""
	if data, err := os.ReadFile(configPath); err == nil {
		existing = string(data)
	}

	// First, strip any pre-existing unmanaged [mcp_servers.hero] table that
	// would collide with the managed block. TOML treats two `[mcp_servers.hero]`
	// tables in the same file as a duplicate-key error, so leaving an old
	// hand-written entry next to our managed block silently breaks codex.
	// Only strips tables OUTSIDE the managed markers — the managed block
	// itself is handled by the marker replacement below.
	existing = stripUnmanagedCodexHeroTable(existing)

	var newContent string
	startIdx := strings.Index(existing, codexMCPMarker)
	endIdx := strings.Index(existing, codexMCPEndMarker)
	if startIdx >= 0 && endIdx > startIdx {
		// Replace existing hero block
		newContent = existing[:startIdx] + heroBlock + existing[endIdx+len(codexMCPEndMarker):]
	} else if strings.TrimSpace(existing) == "" {
		newContent = heroBlock + "\n"
	} else {
		// Append hero block
		newContent = strings.TrimRight(existing, "\n") + "\n\n" + heroBlock + "\n"
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte(newContent), 0o644)
}

// stripUnmanagedCodexHeroTable removes any `[mcp_servers.hero]` TOML table
// that sits OUTSIDE the hero:managed marker block, including its
// immediately-following key/value lines (up to the next table header,
// the managed marker, or EOF).
//
// Pre-v0.14.2 installs (and any hand-written codex configs) could leave
// an unmanaged `[mcp_servers.hero]` table in place; the subsequent install
// would add the managed block alongside it, producing duplicate-key TOML
// that codex rejects. This is the dedup step that makes the managed block
// authoritative on every write.
func stripUnmanagedCodexHeroTable(s string) string {
	const tableHeader = "[mcp_servers.hero]"
	lines := strings.Split(s, "\n")
	var out []string

	inManaged := false
	skipping := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track whether we're inside the managed block — the managed
		// block's own [mcp_servers.hero] is fine and must be preserved.
		if !inManaged && trimmed == codexMCPMarker {
			inManaged = true
			out = append(out, line)
			continue
		}
		if inManaged && trimmed == codexMCPEndMarker {
			inManaged = false
			out = append(out, line)
			continue
		}
		if inManaged {
			out = append(out, line)
			continue
		}

		if skipping {
			// Stop skipping at the next TOML table header or at the
			// managed marker. The skipped region included everything
			// from `[mcp_servers.hero]` up to (but not including) this
			// terminator.
			if strings.HasPrefix(trimmed, "[") || trimmed == codexMCPMarker {
				skipping = false
				out = append(out, line)
			}
			// Else: still inside the unmanaged hero table — drop it.
			continue
		}

		if trimmed == tableHeader {
			// Start dropping. Do not emit this line; remove any trailing
			// blank line that immediately preceded it for cleanliness.
			for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
				out = out[:len(out)-1]
			}
			skipping = true
			continue
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

// RegisterProject registers a project directory in the global daemon registry
// (~/.hero/projects.json). This is called during hero install so the daemon
// knows about all projects on this machine.
func RegisterProject(projectDir string, dryRun bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	registryPath := filepath.Join(home, ".hero", "projects.json")

	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Verify .hero directory exists
	heroDir := filepath.Join(absPath, ".hero")
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		// Project might use a custom folder name, skip registration
		return nil
	}

	if dryRun {
		fmt.Printf("  project -> %s\n", registryPath)
		return nil
	}

	// Read existing registry
	var registry map[string]interface{}
	data, err := os.ReadFile(registryPath)
	if err == nil {
		json.Unmarshal(data, &registry)
	}
	if registry == nil {
		registry = map[string]interface{}{}
	}

	projects, _ := registry["projects"].(map[string]interface{})
	if projects == nil {
		projects = map[string]interface{}{}
	}

	slug := filepath.Base(absPath)

	if existing, ok := projects[slug]; ok {
		if entry, ok := existing.(map[string]interface{}); ok {
			if entry["path"] == absPath {
				return nil // already registered, idempotent
			}
		}
	}

	projects[slug] = map[string]interface{}{
		"path":       absPath,
		"registered": "auto",
	}
	registry["projects"] = projects

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(registryPath), 0o755); err != nil {
		return err
	}

	out, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	return os.WriteFile(registryPath, out, 0o644)
}
