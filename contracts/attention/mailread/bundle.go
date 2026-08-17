package mailread

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const BundlePath = "contracts/attention/mailread/conformance/v1"

type FixtureArtifact struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
	Schema  string `json:"schema"`
	SHA256  string `json:"sha256"`
}

type FixtureManifest struct {
	SchemaVersion int               `json:"schema_version"`
	Fixtures      []FixtureArtifact `json:"fixtures"`
}

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

// BuildBundle creates the deterministic vendorable bundle from a mailread
// package directory containing schema/v1 and testdata/v1.
func BuildBundle(packageDir string) (Bundle, error) {
	schemaDir := filepath.Join(packageDir, "schema", "v1")
	fixtureDir := filepath.Join(packageDir, "testdata", "v1")
	files := make(map[string][]byte)
	var artifacts []BundleArtifact

	schemas, err := jsonFiles(schemaDir)
	if err != nil {
		return Bundle{}, err
	}
	for _, name := range schemas {
		data, err := os.ReadFile(filepath.Join(schemaDir, name))
		if err != nil {
			return Bundle{}, err
		}
		path := filepath.ToSlash(filepath.Join("schemas", name))
		files[path] = data
		artifacts = append(artifacts, bundleArtifact(path, "schema", "Published Mail-read v1 JSON Schema: "+name, "", data))
	}

	fixtureManifestBytes, err := os.ReadFile(filepath.Join(fixtureDir, "manifest.json"))
	if err != nil {
		return Bundle{}, err
	}
	var fixtureManifest FixtureManifest
	if err := json.Unmarshal(fixtureManifestBytes, &fixtureManifest); err != nil {
		return Bundle{}, fmt.Errorf("decode fixture manifest: %w", err)
	}
	if fixtureManifest.SchemaVersion != SchemaVersion {
		return Bundle{}, fmt.Errorf("fixture manifest schema version is %d", fixtureManifest.SchemaVersion)
	}
	metadata := make(map[string]FixtureArtifact, len(fixtureManifest.Fixtures))
	for _, fixture := range fixtureManifest.Fixtures {
		if _, duplicate := metadata[fixture.Path]; duplicate {
			return Bundle{}, fmt.Errorf("duplicate fixture manifest path %s", fixture.Path)
		}
		metadata[fixture.Path] = fixture
	}

	fixtures, err := jsonFiles(fixtureDir)
	if err != nil {
		return Bundle{}, err
	}
	for _, name := range fixtures {
		data, err := os.ReadFile(filepath.Join(fixtureDir, name))
		if err != nil {
			return Bundle{}, err
		}
		path := filepath.ToSlash(filepath.Join("fixtures", name))
		kind, purpose, schema := "fixture", "Canonical Mail-read v1 fixture: "+name, ""
		if name == "manifest.json" {
			kind, purpose = "fixture-manifest", "Canonical Mail-read fixture inventory"
		} else {
			fixture, ok := metadata[name]
			if !ok {
				return Bundle{}, fmt.Errorf("fixture %s is missing from testdata/v1/manifest.json", name)
			}
			if fixture.SHA256 != SumSHA256(data) {
				return Bundle{}, fmt.Errorf("fixture %s checksum differs from testdata manifest", name)
			}
			purpose = fixture.Purpose
			schema = filepath.ToSlash(filepath.Join("schemas", fixture.Schema))
		}
		files[path] = data
		artifacts = append(artifacts, bundleArtifact(path, kind, purpose, schema, data))
	}
	if len(metadata) != len(fixtures)-1 {
		return Bundle{}, fmt.Errorf("fixture manifest has %d entries for %d fixture files", len(metadata), len(fixtures)-1)
	}

	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	manifestBytes, err := json.MarshalIndent(BundleManifest{
		BundleVersion: BundleVersion,
		SchemaVersion: SchemaVersion,
		Compatibility: Compatibility,
		Artifacts:     artifacts,
	}, "", "  ")
	if err != nil {
		return Bundle{}, err
	}
	manifestBytes = append(manifestBytes, '\n')
	files["manifest.json"] = manifestBytes

	handoff, err := os.ReadFile(filepath.Join(fixtureDir, "HERO-CODE-HANDOFF.md"))
	if err != nil {
		return Bundle{}, err
	}
	files["HERO-CODE-HANDOFF.md"] = handoff
	return Bundle{Files: files, ManifestSHA256: SumSHA256(manifestBytes)}, nil
}

func CheckBundle(dir string, bundle Bundle) error {
	actual := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
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
	})
	if err != nil {
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
	return ValidateBundle(dir)
}

func ValidateBundle(dir string) error {
	manifestBytes, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest BundleManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.BundleVersion != BundleVersion || manifest.SchemaVersion != SchemaVersion || manifest.Compatibility != Compatibility {
		return fmt.Errorf("incompatible Mail-read bundle metadata")
	}
	expected := map[string]bool{"manifest.json": true, "HERO-CODE-HANDOFF.md": true}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if artifact.Path <= lastPath {
			return fmt.Errorf("artifacts are not strictly sorted at %s", artifact.Path)
		}
		lastPath = artifact.Path
		if expected[artifact.Path] {
			return fmt.Errorf("duplicate or reserved artifact path %s", artifact.Path)
		}
		expected[artifact.Path] = true
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(artifact.Path)))
		if err != nil {
			return fmt.Errorf("read %s: %w", artifact.Path, err)
		}
		if SumSHA256(data) != artifact.SHA256 {
			return fmt.Errorf("checksum mismatch for %s", artifact.Path)
		}
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("decode %s: %w", artifact.Path, err)
		}
		if artifact.Schema != "" && !hasArtifact(manifest.Artifacts, artifact.Schema) {
			return fmt.Errorf("schema %s for %s is not bundled", artifact.Schema, artifact.Path)
		}
	}

	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !expected[rel] {
			return fmt.Errorf("extra unmanifested bundle artifact %s", rel)
		}
		delete(expected, rel)
		return nil
	})
	if err != nil {
		return err
	}
	if len(expected) != 0 {
		return fmt.Errorf("bundle is missing %d expected artifacts", len(expected))
	}
	return nil
}

func SumSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func jsonFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func bundleArtifact(path, kind, purpose, schema string, data []byte) BundleArtifact {
	return BundleArtifact{Path: path, Kind: kind, Purpose: purpose, MediaType: "application/json", SHA256: SumSHA256(data), Schema: schema}
}

func hasArtifact(artifacts []BundleArtifact, path string) bool {
	for _, artifact := range artifacts {
		if artifact.Path == path {
			return true
		}
	}
	return false
}
