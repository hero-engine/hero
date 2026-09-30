package install

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/version"
)

const deepseekManifestName = "hero-install-manifest.json"

type deepseekManifest struct {
	Version int               `json:"version"`
	Files   map[string]string `json:"files"`
}

func deepseekChecksums(opts Options, base string) (map[string]string, error) {
	out := map[string]string{}
	if opts.Mode == ModeGlobal {
		manifestPath := filepath.Join(base, deepseekManifestName)
		if info, err := os.Lstat(manifestPath); err == nil && !info.Mode().IsRegular() {
			return nil, fmt.Errorf("DeepSeek manifest is not a regular file: %s", manifestPath)
		}
		data, err := os.ReadFile(manifestPath)
		if os.IsNotExist(err) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var manifest deepseekManifest
		if err = json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("invalid DeepSeek manifest: %w", err)
		}
		if manifest.Version != 1 || manifest.Files == nil {
			return nil, fmt.Errorf("invalid DeepSeek manifest version or files")
		}
		for key, sum := range manifest.Files {
			decoded, err := hex.DecodeString(strings.TrimPrefix(sum, "sha256:"))
			if !validDeepSeekOwnedPath(key) || !strings.HasPrefix(sum, "sha256:") || err != nil || len(decoded) != 32 {
				return nil, fmt.Errorf("invalid DeepSeek manifest entry %q", key)
			}
		}
		return manifest.Files, nil
	}
	root := opts.TargetDir
	if opts.ProjectRoot != "" {
		root = opts.ProjectRoot
	}
	// base is absolute (deepseekBase); a relative root such as "." from
	// `hero install project .` must be made absolute before relating them.
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	prefix, err := filepath.Rel(root, base)
	if err != nil {
		return nil, err
	}
	prefix = filepath.ToSlash(prefix) + "/"
	info, err := version.Read(filepath.Join(root, ".hero"))
	if err != nil {
		return nil, err
	}
	if info != nil {
		for key, sum := range info.InstalledFiles {
			key = filepath.ToSlash(key)
			if strings.HasPrefix(key, prefix) {
				rel := strings.TrimPrefix(key, prefix)
				if validDeepSeekOwnedPath(rel) {
					out[rel] = sum
				}
			}
		}
	}
	for key, sum := range opts.TrustedChecksums {
		key = filepath.ToSlash(key)
		if strings.HasPrefix(key, prefix) {
			rel := strings.TrimPrefix(key, prefix)
			if validDeepSeekOwnedPath(rel) {
				out[rel] = sum
			}
		}
	}
	return out, nil
}
func validDeepSeekOwnedPath(key string) bool {
	if key == deepseekOverlayName {
		return true
	}
	parts := strings.Split(key, "/")
	return len(parts) == 3 && parts[0] == "skills" && parts[1] != "" && parts[1] != "." && parts[1] != ".." && !strings.Contains(parts[1], "\\") && parts[2] == "SKILL.md"
}
func deepseekUnchanged(path, sum string) bool {
	if sum == "" {
		return false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	got, err := version.FileChecksum(path)
	return err == nil && got == sum
}
func preflightDeepSeek(opts Options, base string, files map[string]deepseekFile, prior map[string]string) error {
	for key, file := range files {
		path := filepath.Join(base, filepath.FromSlash(key))
		// A symlinked ancestor would redirect writes outside the owned tree.
		for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
			if info, err := os.Lstat(dir); err == nil && info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("DeepSeek destination ancestor is a symlink: %s", dir)
			}
			if dir == base || dir == filepath.Dir(dir) {
				break
			}
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("DeepSeek destination is not a regular file: %s", path)
		}
		old, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Equal(old, file.data) || opts.Force || deepseekUnchanged(path, prior[key]) {
			continue
		}
		return fmt.Errorf("DeepSeek destination is unknown or modified: %s; use --force", path)
	}
	return nil
}
func pruneDeepSeek(opts Options, base string, files map[string]deepseekFile, prior map[string]string) error {
	for key, sum := range prior {
		if _, keep := files[key]; keep {
			continue
		}
		path := filepath.Join(base, filepath.FromSlash(key))
		if deepseekHasSymlinkAncestor(path, base) || !deepseekUnchanged(path, sum) {
			continue
		}
		if !opts.DryRun {
			if err := os.Remove(path); err != nil {
				return err
			}
			if strings.HasPrefix(key, "skills/") {
				_ = os.Remove(filepath.Dir(path))
			}
		}
	}
	return nil
}
func writeDeepSeekManifest(base string, files map[string]deepseekFile) error {
	manifest := deepseekManifest{Version: 1, Files: map[string]string{}}
	for key := range files {
		sum, err := version.FileChecksum(filepath.Join(base, filepath.FromSlash(key)))
		if err != nil {
			return err
		}
		manifest.Files[key] = sum
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(base, deepseekManifestName), append(data, '\n'), 0644)
}

func deepseekHasSymlinkAncestor(path, base string) bool {
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		if info, err := os.Lstat(dir); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return true
		}
		if dir == base || dir == filepath.Dir(dir) {
			return false
		}
	}
}
