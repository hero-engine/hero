package serve

import (
	"github.com/hero-engine/hero/internal/config"
)

// ToolFilter determines which MCP tools are visible to a given client.
// It is constructed once from ServeConfig and reused across requests.
type ToolFilter struct {
	cfg *config.MCPToolFilter
}

// NewToolFilter creates a ToolFilter from the serve config.
// If cfg is nil, the filter allows all tools.
func NewToolFilter(cfg *config.MCPToolFilter) *ToolFilter {
	return &ToolFilter{cfg: cfg}
}

// FilterTools returns only the tools that should be exposed, given an optional
// profile name. If profile is empty, the base allow/deny lists apply.
func (f *ToolFilter) FilterTools(tools []ToolDefinition, profile string) []ToolDefinition {
	if f.cfg == nil {
		return tools
	}

	// Determine the effective allowlist
	allowList := f.cfg.Allow
	if profile != "" {
		if profileTools, ok := f.cfg.Profiles[profile]; ok {
			allowList = profileTools
		}
	}

	// Build denySet
	denySet := make(map[string]bool, len(f.cfg.Deny))
	for _, name := range f.cfg.Deny {
		denySet[name] = true
	}

	// Build allowSet (empty = allow all)
	allowSet := make(map[string]bool, len(allowList))
	for _, name := range allowList {
		allowSet[name] = true
	}

	var filtered []ToolDefinition
	for _, tool := range tools {
		// Deny takes precedence over allow
		if denySet[tool.Name] {
			continue
		}
		// If an allowSet is configured, only pass tools in it
		if len(allowSet) > 0 && !allowSet[tool.Name] {
			continue
		}
		filtered = append(filtered, tool)
	}
	return filtered
}
