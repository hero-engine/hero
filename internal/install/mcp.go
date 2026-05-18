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

// findHeroBinary locates the hero binary. Checks:
// 1. The running binary itself
// 2. PATH lookup
func findHeroBinary() (string, error) {
	// Try to find in PATH
	path, err := exec.LookPath("hero")
	if err == nil {
		return path, nil
	}

	// If we can't find it, return a reasonable default
	return "hero", nil
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

// registerMCPCodex writes the hero MCP server to .codex/config.toml.
// Project mode: <project>/.codex/config.toml. Global mode: ~/.codex/config.toml.
// Format uses a hero:managed marker block so we can update without clobbering user content:
//
//	# hero:managed
//	[mcp_servers.hero]
//	command = "/path/to/hero"
//	args = ["mcp"]
//	# end:hero:managed
func registerMCPCodex(heroPath string, opts Options) error {
	var configPath string
	switch opts.Mode {
	case ModeProject:
		configPath = filepath.Join(opts.TargetDir, ".codex", "config.toml")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(home, ".codex", "config.toml")
	}

	return upsertCodexConfig(configPath, heroPath, opts.DryRun, opts.ProjectRoot)
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

	var newContent string
	startIdx := strings.Index(existing, codexMCPMarker)
	endIdx := strings.Index(existing, codexMCPEndMarker)
	if startIdx >= 0 && endIdx > startIdx {
		// Replace existing hero block
		newContent = existing[:startIdx] + heroBlock + existing[endIdx+len(codexMCPEndMarker):]
	} else if existing == "" {
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
