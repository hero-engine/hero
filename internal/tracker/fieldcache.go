package tracker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// fieldCache represents the cached Jira custom field mappings stored at
// .hero/knowledge/tracker/config.json. This is "what Hero has learned about
// your tracker" — not user-editable config, but persistent discovery results.
type fieldCache struct {
	// DiscoveredAt is the timestamp of the last successful field discovery.
	DiscoveredAt string `json:"discovered_at"`

	// ProjectKey is the Jira project key this cache was built for.
	// If the project changes, the cache is invalidated and re-discovered.
	ProjectKey string `json:"project_key,omitempty"`

	// Fields maps lowercase field names to Jira custom field IDs.
	// Example: {"severity": "customfield_10100", "impact": "customfield_10234"}
	Fields map[string]string `json:"fields"`

	// AutoDiscoveryDone is true once we've scanned /rest/api/3/field for
	// common severity-like names. Prevents re-scanning when none are found.
	AutoDiscoveryDone bool `json:"auto_discovery_done"`
}

// fieldCacheFilename is the file within the tracker knowledge dir.
const fieldCacheFilename = "config.json"

// loadFieldCache reads the cached field mappings from disk.
// Returns nil (no error) if the file doesn't exist or can't be parsed.
func loadFieldCache(trackerKnowledgeDir string) *fieldCache {
	path := filepath.Join(trackerKnowledgeDir, fieldCacheFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var fc fieldCache
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil
	}
	return &fc
}

// saveFieldCache writes the discovered field mappings to disk.
// Creates the directory if needed. Errors are silently ignored (cache is best-effort).
func saveFieldCache(trackerKnowledgeDir string, fields map[string]string, projectKey string) {
	fc := fieldCache{
		DiscoveredAt:      time.Now().UTC().Format(time.RFC3339),
		ProjectKey:        projectKey,
		Fields:            fields,
		AutoDiscoveryDone: true,
	}
	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(trackerKnowledgeDir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(trackerKnowledgeDir, fieldCacheFilename), data, 0o644)
}
