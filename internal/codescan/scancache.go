package codescan

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// scanCacheVersion is bumped whenever ScanCache's on-disk shape changes.
// A version mismatch is treated as an unusable cache (full re-parse), not an error.
const scanCacheVersion = 2

// ScanCache carries per-file parse products forward across incremental scans so
// unchanged files need not be re-parsed while the merged Result stays complete.
// It is a sibling artifact to .checksums.json (not a change to Result's wire
// form) and is keyed by relative file path.
type ScanCache struct {
	Version       int                    `json:"version"`
	Generation    string                 `json:"generation"`
	ChecksumsHash string                 `json:"checksums_hash"`
	Parser        string                 `json:"parser"`
	Sources       map[string]bool        `json:"sources"`
	Files         map[string]FileInfo    `json:"files"`
	ConfigVars    map[string][]ConfigVar `json:"config_vars,omitempty"`
	Endpoints     map[string][]Endpoint  `json:"endpoints,omitempty"`
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
	if cache.Sources == nil {
		return nil, nil
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
	return atomicWriteFile(filepath.Join(codeDir, ".scan-cache.json"), data, 0o644)
}

// BuildScanCache constructs a fresh cache from the COMPLETE merged result, ready
// to carry unchanged files forward on the next incremental scan.
func BuildScanCache(result *Result) *ScanCache {
	return buildScanCache(result, "", "")
}

func buildScanCache(result *Result, parser, generation string) *ScanCache {
	cache := &ScanCache{
		Version:       scanCacheVersion,
		Generation:    generation,
		ChecksumsHash: checksumsHash(result.Checksums),
		Parser:        parser,
		Sources:       make(map[string]bool, len(result.Checksums)),
		Files:         make(map[string]FileInfo, len(result.Files)),
		ConfigVars:    make(map[string][]ConfigVar),
		Endpoints:     make(map[string][]Endpoint),
	}
	for _, fi := range result.Files {
		cache.Files[fi.Path] = fi
	}
	for path := range result.Checksums {
		cache.Sources[path] = true
	}
	for _, cv := range result.ConfigVars {
		cache.ConfigVars[cv.File] = append(cache.ConfigVars[cv.File], cv)
	}
	for _, ep := range result.Endpoints {
		cache.Endpoints[ep.File] = append(cache.Endpoints[ep.File], ep)
	}
	return cache
}

// LoadScanState returns a checksum/cache pair only when both artifacts describe
// the same completed source generation and parser backend.
func LoadScanState(codeDir, parser string) (map[string]string, *ScanCache, bool, error) {
	checksums, err := LoadChecksums(codeDir)
	if err != nil {
		return nil, nil, false, err
	}
	cache, err := LoadScanCache(codeDir)
	if err != nil {
		return nil, nil, false, err
	}
	if checksums == nil || cache == nil || cache.Generation == "" ||
		cache.ChecksumsHash != checksumsHash(checksums) || cache.Parser != parser {
		return nil, nil, false, nil
	}
	for path := range checksums {
		if !cache.Sources[path] {
			return nil, nil, false, nil
		}
		if !isEndpointOnlyExt(strings.ToLower(filepath.Ext(path))) {
			if _, ok := cache.Files[path]; !ok {
				return nil, nil, false, nil
			}
		}
	}
	return checksums, cache, true, nil
}

// CommitScanState advances the checksum/cache pair after all refresh phases
// have completed. Renamed temp files plus the cache's checksum manifest make a
// crash between renames detectable on the next load.
func CommitScanState(result *Result, codeDir, parser string) error {
	return CommitScanStateContext(context.Background(), result, codeDir, parser)
}

// CommitScanStateContext is CommitScanState with cancellation checks between
// each durable artifact step.
func CommitScanStateContext(ctx context.Context, result *Result, codeDir, parser string) error {
	if result == nil || !result.Complete {
		return fmt.Errorf("cannot persist incomplete scan state")
	}
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		return err
	}
	generation := fmt.Sprintf("%d", time.Now().UnixNano())
	checksumData, err := json.MarshalIndent(result.Checksums, "", "  ")
	if err != nil {
		return err
	}
	cacheData, err := json.MarshalIndent(buildScanCache(result, parser, generation), "", "  ")
	if err != nil {
		return err
	}
	checksumPath := filepath.Join(codeDir, ".checksums.json")
	cachePath := filepath.Join(codeDir, ".scan-cache.json")
	checksumTmp, err := writeTempFile(codeDir, ".checksums-", checksumData, 0o644)
	if err != nil {
		return err
	}
	defer os.Remove(checksumTmp)
	cacheTmp, err := writeTempFile(codeDir, ".scan-cache-", cacheData, 0o644)
	if err != nil {
		return err
	}
	defer os.Remove(cacheTmp)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(checksumTmp, checksumPath); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(cacheTmp, cachePath); err != nil {
		return err
	}
	return nil
}

func checksumsHash(checksums map[string]string) string {
	keys := make([]string, 0, len(checksums))
	for key := range checksums {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, key := range keys {
		h.Write([]byte(key))
		h.Write([]byte{0})
		h.Write([]byte(checksums[key]))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := writeTempFile(dir, "."+strings.TrimPrefix(filepath.Base(path), "."), data, mode)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	return os.Rename(tmp, path)
}

func writeTempFile(dir, pattern string, data []byte, mode os.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Chmod(mode); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}
