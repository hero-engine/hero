package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ToolExecutor runs tools in-process.
type ToolExecutor struct {
	projectRoot string
	heroDir     string
}

// NewToolExecutor creates a tool executor for the project.
func NewToolExecutor(projectRoot, heroDir string) *ToolExecutor {
	return &ToolExecutor{projectRoot: projectRoot, heroDir: heroDir}
}

// Execute runs a tool call and returns the result text.
func (t *ToolExecutor) Execute(call ToolCall) string {
	switch call.Name {
	case "read_file":
		return t.readFile(call.Input)
	case "write_file":
		return t.writeFile(call.Input)
	case "list_directory":
		return t.listDirectory(call.Input)
	case "run_command":
		return t.runCommand(call.Input)
	case "search_files":
		return t.searchFiles(call.Input)
	default:
		// Try hero tools via CLI
		return t.heroTool(call.Name, call.Input)
	}
}

// ToolDefinitions returns the tool definitions for the agent.
func (t *ToolExecutor) ToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read a file from the project",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string", "description": "File path relative to project root"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Write content to a file in the project",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":    map[string]interface{}{"type": "string", "description": "File path relative to project root"},
					"content": map[string]interface{}{"type": "string", "description": "File content"},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "list_directory",
			Description: "List files in a directory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string", "description": "Directory path relative to project root"},
				},
			},
		},
		{
			Name:        "run_command",
			Description: "Run a shell command in the project directory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{"type": "string", "description": "Shell command to execute"},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "search_files",
			Description: "Search for a pattern in project files using grep",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{"type": "string", "description": "Search pattern (regex)"},
					"path":    map[string]interface{}{"type": "string", "description": "Directory to search in (default: project root)"},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func (t *ToolExecutor) readFile(input map[string]interface{}) string {
	path, _ := input["path"].(string)
	if path == "" {
		return "error: path is required"
	}
	abs := t.resolve(path)
	data, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(data)
}

func (t *ToolExecutor) writeFile(input map[string]interface{}) string {
	path, _ := input["path"].(string)
	content, _ := input["content"].(string)
	if path == "" {
		return "error: path is required"
	}
	abs := t.resolve(path)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Sprintf("error creating directory: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path)
}

func (t *ToolExecutor) listDirectory(input map[string]interface{}) string {
	path, _ := input["path"].(string)
	if path == "" {
		path = "."
	}
	abs := t.resolve(path)
	entries, err := os.ReadDir(abs)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	var lines []string
	for _, e := range entries {
		suffix := ""
		if e.IsDir() {
			suffix = "/"
		}
		lines = append(lines, e.Name()+suffix)
	}
	return strings.Join(lines, "\n")
}

func (t *ToolExecutor) runCommand(input map[string]interface{}) string {
	command, _ := input["command"].(string)
	if command == "" {
		return "error: command is required"
	}
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = t.projectRoot
	out, err := cmd.CombinedOutput()
	result := string(out)
	if err != nil {
		result += fmt.Sprintf("\nexit code: %v", err)
	}
	if len(result) > 10000 {
		result = result[:10000] + "\n... (truncated)"
	}
	return result
}

func (t *ToolExecutor) searchFiles(input map[string]interface{}) string {
	pattern, _ := input["pattern"].(string)
	searchPath, _ := input["path"].(string)
	if pattern == "" {
		return "error: pattern is required"
	}
	if searchPath == "" {
		searchPath = "."
	}
	abs := t.resolve(searchPath)
	cmd := exec.Command("grep", "-rn", "--include=*.go", "--include=*.ts", "--include=*.js", "--include=*.py", "--include=*.md", pattern, abs)
	out, _ := cmd.CombinedOutput()
	result := string(out)
	if len(result) > 10000 {
		result = result[:10000] + "\n... (truncated)"
	}
	if result == "" {
		return "no matches found"
	}
	return result
}

func (t *ToolExecutor) heroTool(name string, input map[string]interface{}) string {
	// Map hero_* MCP tool names to CLI commands
	cmdName := strings.TrimPrefix(name, "hero_")
	args := []string{cmdName}

	// Pass input as flags
	for k, v := range input {
		switch val := v.(type) {
		case string:
			if val != "" {
				args = append(args, "--"+strings.ReplaceAll(k, "_", "-"), val)
			}
		case bool:
			if val {
				args = append(args, "--"+strings.ReplaceAll(k, "_", "-"))
			}
		}
	}

	cmd := exec.Command("hero", args...)
	cmd.Dir = t.projectRoot
	out, err := cmd.CombinedOutput()
	result := string(out)
	if err != nil {
		result += fmt.Sprintf("\nerror: %v", err)
	}
	return result
}

func (t *ToolExecutor) resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.projectRoot, path)
}
