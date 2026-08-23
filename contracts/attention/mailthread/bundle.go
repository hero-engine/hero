package mailthread

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const BundlePath = "contracts/attention/mailthread/conformance/v1"

type BundleArtifact struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	Purpose   string `json:"purpose"`
	MediaType string `json:"media_type"`
	SHA256    string `json:"sha256"`
	Schema    string `json:"schema,omitempty"`
}

type BundleManifest struct {
	BundleVersion int              `json:"bundle_version"`
	SchemaVersion int              `json:"schema_version"`
	Compatibility string           `json:"compatibility"`
	Artifacts     []BundleArtifact `json:"artifacts"`
}

type Bundle struct {
	Files          map[string][]byte
	ManifestSHA256 string
}

var fixtureSchemas = map[string]string{
	"state-open.json":        "thread-view.schema.json",
	"state-resolved.json":    "thread-view.schema.json",
	"state-archived.json":    "thread-view.schema.json",
	"canonical-actions.json": "capabilities.schema.json",
	"action-request.json":    "action-request.schema.json",
	"action-success.json":    "action-response.schema.json",
	"errors.json":            "error-matrix.schema.json",
	"migration.json":         "migration-matrix.schema.json",
	"unknown-additive.json":  "thread-view.schema.json",
}

func BuildBundle(packageDir string) (Bundle, error) {
	files := map[string][]byte{}
	var artifacts []BundleArtifact
	for _, source := range []struct{ dir, target, kind string }{{"schema/v1", "schemas", "schema"}, {"testdata/v1", "fixtures", "fixture"}} {
		entries, err := os.ReadDir(filepath.Join(packageDir, filepath.FromSlash(source.dir)))
		if err != nil {
			return Bundle{}, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(packageDir, filepath.FromSlash(source.dir), entry.Name()))
			if err != nil {
				return Bundle{}, err
			}
			path := filepath.ToSlash(filepath.Join(source.target, entry.Name()))
			schema := ""
			if source.kind == "fixture" {
				name, ok := fixtureSchemas[entry.Name()]
				if !ok {
					return Bundle{}, fmt.Errorf("fixture %s has no schema mapping", entry.Name())
				}
				schema = "schemas/" + name
			}
			files[path] = data
			artifacts = append(artifacts, BundleArtifact{Path: path, Kind: source.kind, Purpose: "Mail thread lifecycle v1 " + source.kind + ": " + entry.Name(), MediaType: "application/json", SHA256: sum(data), Schema: schema})
		}
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	manifest, err := json.MarshalIndent(BundleManifest{BundleVersion: BundleVersion, SchemaVersion: SchemaVersion, Compatibility: Compatibility, Artifacts: artifacts}, "", "  ")
	if err != nil {
		return Bundle{}, err
	}
	manifest = append(manifest, '\n')
	files["manifest.json"] = manifest
	handoff, err := os.ReadFile(filepath.Join(packageDir, "testdata", "v1", "HERO-CODE-HANDOFF.md"))
	if err != nil {
		return Bundle{}, err
	}
	files["HERO-CODE-HANDOFF.md"] = handoff
	return Bundle{Files: files, ManifestSHA256: sum(manifest)}, nil
}

func WriteBundle(dir string, bundle Bundle) error {
	for path, data := range bundle.Files {
		target := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func CheckBundle(dir string, bundle Bundle) error {
	actual := map[string][]byte{}
	if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		actual[filepath.ToSlash(rel)] = data
		return nil
	}); err != nil {
		return err
	}
	if len(actual) != len(bundle.Files) {
		return fmt.Errorf("checked-in bundle has %d files; generated bundle has %d", len(actual), len(bundle.Files))
	}
	for path, expected := range bundle.Files {
		if !bytes.Equal(actual[path], expected) {
			return fmt.Errorf("checked-in bundle is stale at %s", path)
		}
	}
	return validateBundle(dir)
}

func validateBundle(dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest BundleManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if manifest.BundleVersion != BundleVersion || manifest.SchemaVersion != SchemaVersion || manifest.Compatibility != Compatibility {
		return errors.New("incompatible Mail thread bundle metadata")
	}
	for _, artifact := range manifest.Artifacts {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(artifact.Path)))
		if err != nil {
			return err
		}
		if sum(content) != artifact.SHA256 {
			return fmt.Errorf("checksum mismatch for %s", artifact.Path)
		}
	}
	return nil
}

func sum(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
