// Package hooks provides natural-language file-event hooks — parse, match,
// render, and install into host tool configurations.
package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Hook represents a parsed .hero/hooks/*.md file.
type Hook struct {
	Name    string `json:"name"`
	Event   string `json:"event"`    // file.save, file.create, file.delete
	Match   string `json:"match"`    // glob pattern
	Agent   string `json:"agent"`    // optional agent slug
	Mode    string `json:"mode"`     // silent, confirm, foreground
	Body    string `json:"body"`     // prompt body (markdown after frontmatter)
	Path    string `json:"path"`     // absolute path to hook file
}

var validEvents = map[string]bool{
	"file.save":   true,
	"file.create": true,
	"file.delete": true,
}

// Discover finds all hook files in .hero/hooks/.
func Discover(heroDir string) ([]*Hook, error) {
	hooksDir := filepath.Join(heroDir, "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var hooks []*Hook
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		h, err := ParseHookFile(filepath.Join(hooksDir, e.Name()))
		if err != nil {
			continue // skip malformed
		}
		hooks = append(hooks, h)
	}
	return hooks, nil
}

// ParseHookFile parses a single hook file.
func ParseHookFile(path string) (*Hook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("hook file missing frontmatter: %s", path)
	}

	// Parse frontmatter
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, fmt.Errorf("hook file missing closing frontmatter: %s", path)
	}

	fm := rest[:idx]
	body := strings.TrimSpace(rest[idx+4:])

	h := &Hook{
		Path: path,
		Mode: "confirm", // default
		Body: body,
	}

	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"'")

		switch key {
		case "name":
			h.Name = val
		case "event":
			h.Event = val
		case "match":
			h.Match = val
		case "agent":
			h.Agent = val
		case "mode":
			h.Mode = val
		}
	}

	if h.Name == "" {
		h.Name = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	if h.Event == "" {
		return nil, fmt.Errorf("hook %q missing event field", h.Name)
	}
	if !validEvents[h.Event] {
		return nil, fmt.Errorf("hook %q has invalid event %q", h.Name, h.Event)
	}
	if h.Match == "" {
		h.Match = "**/*"
	}

	return h, nil
}

// Matches checks if a file path matches the hook's glob pattern.
func (h *Hook) Matches(filePath string) bool {
	// Simple glob matching — support ** as "any path segment"
	pattern := h.Match
	if strings.Contains(pattern, "**") {
		// Convert ** patterns to work with filepath.Match
		// Split on ** and check if any segment matches
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			suffix := strings.TrimPrefix(parts[1], "/")
			if suffix == "" || suffix == "*" {
				// Matches everything
				return true
			}
			// Check if the file matches the suffix pattern
			matched, _ := filepath.Match(suffix, filepath.Base(filePath))
			if matched {
				// Check prefix
				prefix := strings.TrimSuffix(parts[0], "/")
				if prefix == "" || strings.HasPrefix(filePath, prefix) {
					return true
				}
			}
		}
		return false
	}

	matched, _ := filepath.Match(pattern, filePath)
	return matched
}

// Render substitutes template variables in the hook body.
func (h *Hook) Render(filePath string) string {
	body := h.Body
	body = strings.ReplaceAll(body, "{{file}}", filePath)
	body = strings.ReplaceAll(body, "{{relative_path}}", filePath)
	body = strings.ReplaceAll(body, "{{ext}}", filepath.Ext(filePath))
	body = strings.ReplaceAll(body, "{{basename}}", filepath.Base(filePath))
	body = strings.ReplaceAll(body, "{{dir}}", filepath.Dir(filePath))
	return body
}
