// Package memory ingests per-project Claude memory files
// (~/.claude/projects/<project-key>/memory/) into the unified knowledge
// graph as Memory nodes scoped local — these never leave the machine.
package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// Summary reports counts written by WriteGraph.
type Summary struct {
	Files int
}

// DirForProject returns the per-project memory directory path used by
// the Claude Code harness: ~/.claude/projects/<encoded>/memory/, where
// the encoded segment is the absolute project root with each path
// separator replaced by a dash and a leading dash prepended.
//
// For example, /Users/foo/projects/bar becomes
// -Users-foo-projects-bar.
//
// Returns "" if the home dir cannot be resolved.
func DirForProject(projectRoot string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return ""
	}
	encoded := strings.ReplaceAll(abs, string(filepath.Separator), "-")
	return filepath.Join(home, ".claude", "projects", encoded, "memory")
}

// WriteGraph walks memoryDir and upserts a Memory node for each file
// found. Idempotent: unchanged bytes produce the same content hash.
//
// Memory nodes are stamped Scope=local — they never sync. repoKey
// stamps the partition column.
//
// Best-effort: missing dir is not an error (returns zero summary).
func WriteGraph(memoryDir, repoKey string, store *graph.Store) (*Summary, error) {
	if memoryDir == "" {
		return &Summary{}, nil
	}
	info, err := os.Stat(memoryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Summary{}, nil
		}
		return nil, fmt.Errorf("stat memory dir: %w", err)
	}
	if !info.IsDir() {
		return &Summary{}, nil
	}

	source := map[string]any{"kind": "memory"}
	summary := &Summary{}

	walkErr := filepath.WalkDir(memoryDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		hash := sha256Hex(bytes)
		fm := parseMemoryFrontmatter(bytes)

		key := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		props := map[string]any{
			"file":       d.Name(),
			"byte_count": len(bytes),
		}
		if fm.name != "" {
			props["name"] = fm.name
		}
		if fm.description != "" {
			props["description"] = fm.description
		}
		if fm.memType != "" {
			props["memory_type"] = fm.memType
		}

		if _, err := store.UpsertNode(&graph.Node{
			Type:        "Memory",
			Key:         key,
			Props:       props,
			Scope:       graph.ScopeLocal,
			Repo:        repoKey,
			ContentHash: hash,
			Source:      source,
		}); err != nil {
			return fmt.Errorf("upsert Memory %s: %w", path, err)
		}
		summary.Files++
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return summary, nil
}

type memoryFrontmatter struct {
	name, description, memType string
}

// parseMemoryFrontmatter extracts the leading YAML frontmatter fields
// the auto-memory system writes. Tolerant: malformed or missing
// frontmatter yields zero values rather than an error, since the file
// bytes remain authoritative regardless of metadata quality.
func parseMemoryFrontmatter(b []byte) memoryFrontmatter {
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		return memoryFrontmatter{}
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		return memoryFrontmatter{}
	}
	body := s[4 : 4+end]
	out := memoryFrontmatter{}
	for _, line := range strings.Split(body, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "name":
			out.name = v
		case "description":
			out.description = v
		case "type":
			out.memType = v
		}
	}
	return out
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
