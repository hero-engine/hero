package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// heroCommand is the command MCP configs point at. It is the bare name,
// not a resolved absolute path, on purpose: these configs live in
// project-root files that travel with the repo (they are meant to — a
// teammate who clones gets working MCP wiring without re-running install),
// and an absolute path is machine-specific and breaks the moment the file
// reaches another machine. Anyone able to use Hero's MCP server already
// has `hero` installed, so the harness resolves it from PATH at launch.
// The residual risk — a stale or wrong `hero` winning the PATH lookup —
// is what `hero doctor` exists to diagnose.
const heroCommand = "hero"

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
func RegisterMCP(target Target, opts Options) error {
	switch target {
	case TargetCursor:
		return registerMCPCursor(opts)
	case TargetClaude:
		return registerMCPClaude(opts)
	case TargetOpenCode:
		return registerMCPOpenCode(opts)
	case TargetCodex:
		return registerMCPCodex(opts)
	case TargetGrok:
		return registerMCPGrok(opts)
	default:
		return nil
	}
}

// registerMCPCursor writes to .cursor/mcp.json in the project.
// Format: { "mcpServers": { "hero": { "command": "hero", "args": ["mcp"] } } }
func registerMCPCursor(opts Options) error {
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

	return upsertMCPConfig(configPath, opts.DryRun, opts.ProjectRoot)
}

// registerMCPClaude writes to .mcp.json in the project root (project mode)
// or ~/.claude.json (global mode).
// Format: { "mcpServers": { "hero": { "command": "hero", "args": ["mcp"] } } }
func registerMCPClaude(opts Options) error {
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

	return upsertMCPConfig(configPath, opts.DryRun, opts.ProjectRoot)
}

// registerMCPOpenCode writes the hero MCP server to the OpenCode config.
// Project mode: <project>/opencode.json under the "mcp" key.
// Global mode: ~/.config/opencode/opencode.json under the "mcp" key.
// Format: { "mcp": { "hero": { "type": "local", "command": ["hero", "mcp"] } } }
func registerMCPOpenCode(opts Options) error {
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

	return upsertOpenCodeMCPConfig(configPath, opts.DryRun, opts.ProjectRoot)
}

// upsertMCPConfig reads an existing MCP config (or creates a new one) and ensures
// the hero server entry is present. Used for Cursor and Claude which use the
// { "mcpServers": { ... } } format.
func upsertMCPConfig(configPath string, dryRun bool, projectRoot string) error {
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
		Command: heroCommand,
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
func upsertOpenCodeMCPConfig(configPath string, dryRun bool, projectRoot string) error {
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

	// Build the command array: ["hero", "mcp"] or with --project-root for workspaces
	cmd := []string{heroCommand, "mcp"}
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

// registerMCPCodex writes the hero MCP server block to the project's
// .codex/config.toml (project mode) or ~/.codex/config.toml (global mode).
// The block lives in the project layer because an MCP server serves that
// project, and it points at the portable `hero` command (see heroCommand)
// so the file travels with the repo.
//
// .codex/config.toml is the user's own file — their model, approval, and
// other MCP settings live there too — so Hero writes only its marked span
// and never touches bytes outside it. Format:
//
//	# hero:managed
//	[mcp_servers.hero]
//	command = "hero"
//	args = ["mcp"]
//	# end:hero:managed
func registerMCPCodex(opts Options) error {
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

	return upsertManagedTOMLMCPConfig(configPath, opts.DryRun, opts.ProjectRoot)
}

// registerMCPGrok writes Hero's MCP server to Grok Build's native TOML
// configuration at the active project or user scope.
func registerMCPGrok(opts Options) error {
	var configPath string
	switch opts.Mode {
	case ModeProject:
		configPath = filepath.Join(opts.TargetDir, ".grok", "config.toml")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(home, ".grok", "config.toml")
	default:
		return fmt.Errorf("unknown mode: %s", opts.Mode)
	}
	return upsertManagedTOMLMCPConfig(configPath, opts.DryRun, opts.ProjectRoot)
}

const codexMCPMarker = "# hero:managed"
const codexMCPEndMarker = "# end:hero:managed"

// upsertCodexConfig reads an existing .codex/config.toml (or creates one) and
// ensures the hero MCP server block is present within hero:managed markers.
func upsertCodexConfig(configPath string, dryRun bool, projectRoot string) error {
	return upsertManagedTOMLMCPConfig(configPath, dryRun, projectRoot)
}

// upsertManagedTOMLMCPConfig owns only Hero's marked MCP table. It removes
// prior managed blocks and legacy unmanaged Hero tables, validates all bytes
// that remain as user-owned TOML, then appends one canonical managed block.
// This makes duplicate legacy Hero tables recoverable without permitting an
// unrelated malformed setting to be overwritten.
func upsertManagedTOMLMCPConfig(configPath string, dryRun bool, projectRoot string) error {
	if dryRun {
		fmt.Printf("  MCP server -> %s\n", configPath)
		return nil
	}

	args := `["mcp"]`
	if projectRoot != "" {
		args = fmt.Sprintf(`["mcp", "--project-root", %q]`, projectRoot)
	}

	heroBlock := fmt.Sprintf("%s\n[mcp_servers.hero]\ncommand = %q\nargs = %s\n%s",
		codexMCPMarker, heroCommand, args, codexMCPEndMarker)

	existing := ""
	if data, err := os.ReadFile(configPath); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read TOML config %s: %w", configPath, err)
	}

	withoutLegacy := stripUnmanagedCodexHeroTable(existing)
	withoutManaged, err := removeManagedTOMLBlocks(withoutLegacy)
	if err != nil {
		return fmt.Errorf("parse TOML config %s: %w", configPath, err)
	}
	var decoded map[string]any
	if _, err := toml.Decode(withoutManaged, &decoded); err != nil {
		return fmt.Errorf("parse TOML config %s: %w", configPath, err)
	}

	newContent, err := replaceManagedTOMLBlocks(withoutLegacy, heroBlock)
	if err != nil {
		return fmt.Errorf("parse TOML config %s: %w", configPath, err)
	}
	decoded = nil
	if _, err := toml.Decode(newContent, &decoded); err != nil {
		return fmt.Errorf("render valid TOML config %s: %w", configPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte(newContent), 0o644)
}

// replaceManagedTOMLBlocks replaces the first complete managed span in place
// and removes any additional stale spans. With no prior span it appends one,
// adding only the separator needed and leaving all existing bytes untouched.
func replaceManagedTOMLBlocks(s, block string) (string, error) {
	var out strings.Builder
	rest := s
	replaced := false
	for {
		start := strings.Index(rest, codexMCPMarker)
		end := strings.Index(rest, codexMCPEndMarker)
		if start < 0 && end < 0 {
			out.WriteString(rest)
			break
		}
		if start < 0 || end < 0 || end < start {
			return "", fmt.Errorf("unmatched Hero managed marker")
		}
		out.WriteString(rest[:start])
		if !replaced {
			out.WriteString(block)
			replaced = true
		}
		rest = rest[end+len(codexMCPEndMarker):]
	}
	if replaced {
		return out.String(), nil
	}
	if s == "" {
		return block + "\n", nil
	}
	separator := "\n\n"
	if strings.HasSuffix(s, "\n\n") {
		separator = ""
	} else if strings.HasSuffix(s, "\n") {
		separator = "\n"
	}
	return s + separator + block + "\n", nil
}

// removeManagedTOMLBlocks removes complete Hero marker spans while preserving
// every byte outside them. An unmatched marker is refused because its
// ownership boundary is ambiguous. Multiple complete spans are accepted and
// converge to one canonical block.
func removeManagedTOMLBlocks(s string) (string, error) {
	var out strings.Builder
	rest := s
	for {
		start := strings.Index(rest, codexMCPMarker)
		end := strings.Index(rest, codexMCPEndMarker)
		if start < 0 && end < 0 {
			out.WriteString(rest)
			return out.String(), nil
		}
		if start < 0 || end < 0 || end < start {
			return "", fmt.Errorf("unmatched Hero managed marker")
		}
		out.WriteString(rest[:start])
		rest = rest[end+len(codexMCPEndMarker):]
	}
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
			// Start dropping. Preceding bytes are user-owned and stay exact.
			skipping = true
			continue
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

// RemoveManagedTOMLMCPConfig removes Hero's marked MCP block and preserves
// every byte outside it. If no user-owned content remains, the file is removed.
func RemoveManagedTOMLMCPConfig(configPath string, dryRun bool) (bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	content := string(data)
	start := strings.Index(content, codexMCPMarker)
	end := strings.Index(content, codexMCPEndMarker)
	if start < 0 && end < 0 {
		return false, nil
	}
	if start < 0 || end < start {
		return false, fmt.Errorf("unmatched Hero managed marker in %s", configPath)
	}
	end += len(codexMCPEndMarker)
	remaining := content[:start] + content[end:]
	if dryRun {
		return true, nil
	}
	if strings.TrimSpace(remaining) == "" {
		return true, os.Remove(configPath)
	}
	return true, os.WriteFile(configPath, []byte(remaining), 0o644)
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
