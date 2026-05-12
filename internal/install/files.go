package install

import (
	"encoding/json"
	"fmt"
	"io"
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

	if _, err := os.Stat(dst); err == nil && !opts.Force {
		result.Skipped = append(result.Skipped, dst)
		return fmt.Errorf("refusing to overwrite %s (use --force to replace)", dst)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	srcFile, err := srcFS.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	result.Copied = append(result.Copied, CopyAction{Source: srcPath, Dest: dst})
	return nil
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
