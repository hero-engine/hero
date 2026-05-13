package install

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// files.go — low-level file primitives shared by all targets.

func copyFileFromFS(opts Options, result *Result, srcFS fs.FS, srcPath, dst string) error {
	if opts.DryRun {
		progressf(opts, "  %s -> %s\n", srcPath, dst)
		result.Copied = append(result.Copied, CopyAction{Source: srcPath, Dest: dst})
		return nil
	}

	// Read source once; reused for both the equality check and the write.
	srcData, err := fs.ReadFile(srcFS, srcPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(dst); err == nil && !opts.Force {
		// Existing destination: idempotency contract — if the bytes match
		// what we'd write, treat as a silent no-op (canonical content is
		// re-materialized on every install across harnesses). Only refuse
		// when the user has actually edited the file.
		dstData, readErr := os.ReadFile(dst)
		if readErr == nil && bytes.Equal(srcData, dstData) {
			return nil
		}
		// Trusted-checksum upgrade path: if the destination file matches
		// a checksum recorded as Hero-installed at a prior version,
		// it's safe to overwrite even though bytes differ from current
		// canonical (the prior install just had different bytes).
		if isTrustedHeroInstalledFile(opts, dst, dstData) {
			// fall through to write
		} else {
			result.Skipped = append(result.Skipped, dst)
			return fmt.Errorf("refusing to overwrite %s (use --force to replace)", dst)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(dst, srcData, 0o644); err != nil {
		return err
	}

	result.Copied = append(result.Copied, CopyAction{Source: srcPath, Dest: dst})
	return nil
}

// isTrustedHeroInstalledFile checks whether dst's current bytes match
// a checksum recorded in opts.TrustedChecksums for the equivalent
// project-relative path. Used to authorize overwriting a file whose
// bytes drift from current canonical because Hero installed it at a
// prior version (i.e. it's "Hero-authored, just outdated").
func isTrustedHeroInstalledFile(opts Options, dst string, dstData []byte) bool {
	if len(opts.TrustedChecksums) == 0 || opts.TargetDir == "" {
		return false
	}
	rel, err := filepath.Rel(opts.TargetDir, dst)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	want, ok := opts.TrustedChecksums[rel]
	if !ok {
		return false
	}
	got := sha256.Sum256(dstData)
	// Match version.FileChecksum format: "sha256:" + hex.
	gotChecksum := "sha256:" + hex.EncodeToString(got[:])
	return gotChecksum == want
}

// mergeJSONFromData does a shallow merge of source JSON data into an existing
// JSON file. If force is true, source values win on conflict; otherwise target
// values win.
func mergeJSONFromData(srcData []byte, dstPath string, force bool) error {
	dstData, err := os.ReadFile(dstPath)
	if err != nil {
		return err
	}

	var srcMap, dstMap map[string]interface{}
	if err := json.Unmarshal(srcData, &srcMap); err != nil {
		return fmt.Errorf("parsing source config: %w", err)
	}
	if err := json.Unmarshal(dstData, &dstMap); err != nil {
		return fmt.Errorf("parsing target config: %w", err)
	}

	var merged map[string]interface{}
	if force {
		merged = deepMerge(dstMap, srcMap)
	} else {
		merged = deepMerge(srcMap, dstMap)
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}

	out = append(out, '\n')
	return os.WriteFile(dstPath, out, 0o644)
}

// deepMerge merges base and override maps. Override values win on conflict.
func deepMerge(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range base {
		result[k] = v
	}

	for k, v := range override {
		if baseVal, ok := result[k]; ok {
			baseMap, baseIsMap := baseVal.(map[string]interface{})
			overMap, overIsMap := v.(map[string]interface{})
			if baseIsMap && overIsMap {
				result[k] = deepMerge(baseMap, overMap)
				continue
			}
		}
		result[k] = v
	}

	return result
}
