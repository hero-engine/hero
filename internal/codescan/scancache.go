package codescan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// scanCacheVersion is bumped whenever ScanCache's on-disk shape changes.
// A version mismatch is treated as an unusable cache (full re-parse), not an error.
const scanCacheVersion = 1

// ScanCache carries per-file parse products forward across incremental scans so
// unchanged files need not be re-parsed while the merged Result stays complete.
// It is a sibling artifact to .checksums.json (not a change to Result's wire
// form) and is keyed by relative file path.
type ScanCache struct {
	Version    int                    `json:"version"`
	Files      map[string]FileInfo    `json:"files"`
	ConfigVars map[string][]ConfigVar `json:"config_vars,omitempty"`
	Endpoints  map[string][]Endpoint  `json:"endpoints,omitempty"`
}

// LoadScanCache reads the scan cache from .hero/knowledge/code/.scan-cache.json.
//
// A nil cache is the "unusable → full-parse" signal the merge guard keys on.
// It is returned (with a nil error) for a missing file or a version mismatch
// (legacy/unusable). An unmarshal error is returned so the caller can warn and
// proceed with nil.
func LoadScanCache(codeDir string) (*ScanCache, error) {
	path := filepath.Join(codeDir, ".scan-cache.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no cache; caller will full-parse
		}
		return nil, fmt.Errorf("reading scan cache: %w", err)
	}
	var cache ScanCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing scan cache: %w", err)
	}
	if cache.Version != scanCacheVersion {
		return nil, nil // legacy/unusable cache → full-parse; not an error
	}
	if cache.Files == nil {
		cache.Files = make(map[string]FileInfo)
	}
	if cache.ConfigVars == nil {
		cache.ConfigVars = make(map[string][]ConfigVar)
	}
	if cache.Endpoints == nil {
		cache.Endpoints = make(map[string][]Endpoint)
	}
	return &cache, nil
}

// Save writes the scan cache to .hero/knowledge/code/.scan-cache.json.
func (c *ScanCache) Save(codeDir string) error {
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(codeDir, ".scan-cache.json"), data, 0o644)
}

// BuildScanCache constructs a fresh cache from the COMPLETE merged result, ready
// to carry unchanged files forward on the next incremental scan.
func BuildScanCache(result *Result) *ScanCache {
	cache := &ScanCache{
		Version:    scanCacheVersion,
		Files:      make(map[string]FileInfo, len(result.Files)),
		ConfigVars: make(map[string][]ConfigVar),
		Endpoints:  make(map[string][]Endpoint),
	}
	for _, fi := range result.Files {
		cache.Files[fi.Path] = fi
	}
	for _, cv := range result.ConfigVars {
		cache.ConfigVars[cv.File] = append(cache.ConfigVars[cv.File], cv)
	}
	for _, ep := range result.Endpoints {
		cache.Endpoints[ep.File] = append(cache.Endpoints[ep.File], ep)
	}
	return cache
}
