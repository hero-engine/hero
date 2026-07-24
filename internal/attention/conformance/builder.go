package conformance

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

	attentioncontract "github.com/hero-engine/hero/contracts/attention"
)

const (
	BundleVersion = 1
	BundlePath    = "contracts/attention/conformance/v1"
	Compatibility = "Unknown additive fields and identifiers must remain inert but decodable; never grant executable behavior from an unknown value."
)

type Artifact struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	Purpose   string `json:"purpose"`
	MediaType string `json:"media_type"`
	SHA256    string `json:"sha256"`
	Schema    string `json:"schema,omitempty"`
}

type Manifest struct {
	BundleVersion int        `json:"bundle_version"`
	SchemaVersion int        `json:"schema_version"`
	Compatibility string     `json:"compatibility"`
	Artifacts     []Artifact `json:"artifacts"`
}

type Bundle struct {
	Files          map[string][]byte
	ManifestSHA256 string
}

type fixtureManifest struct {
	Fixtures []struct {
		Path    string `json:"path"`
		Purpose string `json:"purpose"`
		Schema  string `json:"schema"`
	} `json:"fixtures"`
}

func Build(projectRoot string, toolDefinitionsJSON []byte) (Bundle, error) {
	schemaDir := filepath.Join(projectRoot, "contracts", "attention", "schema", "v1")
	fixtureDir := filepath.Join(projectRoot, "contracts", "attention", "testdata", "v1")
	files := make(map[string][]byte)
	var artifacts []Artifact

	schemaNames, err := jsonFiles(schemaDir, ".schema.json")
	if err != nil {
		return Bundle{}, err
	}
	for _, name := range schemaNames {
		data, err := os.ReadFile(filepath.Join(schemaDir, name))
		if err != nil {
			return Bundle{}, err
		}
		path := filepath.ToSlash(filepath.Join("schemas", name))
		files[path] = data
		artifacts = append(artifacts, artifact(path, "schema", "Published Attention v1 JSON Schema: "+name, "", data))
	}

	fixtureManifestBytes, err := os.ReadFile(filepath.Join(fixtureDir, "manifest.json"))
	if err != nil {
		return Bundle{}, err
	}
	var sourceManifest fixtureManifest
	if err := json.Unmarshal(fixtureManifestBytes, &sourceManifest); err != nil {
		return Bundle{}, fmt.Errorf("decode fixture manifest: %w", err)
	}
	fixtureMetadata := make(map[string]struct{ purpose, schema string })
	for _, fixture := range sourceManifest.Fixtures {
		fixtureMetadata[fixture.Path] = struct{ purpose, schema string }{fixture.Purpose, fixture.Schema}
	}

	fixtureNames, err := jsonFiles(fixtureDir, ".json")
	if err != nil {
		return Bundle{}, err
	}
	for _, name := range fixtureNames {
		data, err := os.ReadFile(filepath.Join(fixtureDir, name))
		if err != nil {
			return Bundle{}, err
		}
		path := filepath.ToSlash(filepath.Join("fixtures", name))
		kind, purpose, schema := "fixture", "Canonical Attention v1 fixture: "+name, ""
		if name == "manifest.json" {
			kind, purpose = "fixture-manifest", "Canonical fixture inventory retained for existing consumers"
		} else if metadata, ok := fixtureMetadata[name]; ok {
			purpose = metadata.purpose
			schema = filepath.ToSlash(filepath.Join("schemas", metadata.schema))
		} else {
			return Bundle{}, fmt.Errorf("fixture %s is missing from testdata/v1/manifest.json", name)
		}
		if name == "conversational-routes.json" {
			kind = "route-corpus"
		}
		files[path] = data
		artifacts = append(artifacts, artifact(path, kind, purpose, schema, data))
	}

	inventory, err := normalizeToolInventory(toolDefinitionsJSON)
	if err != nil {
		return Bundle{}, err
	}
	files["mcp-tools.json"] = inventory
	artifacts = append(artifacts, artifact(
		"mcp-tools.json",
		"mcp-tool-inventory",
		"Model-facing Attention MCP tools generated from Hero's runtime registry",
		"schemas/mcp-tool-inventory.schema.json",
		inventory,
	))
	if err := validateRouteInventoryBytes(
		inventory,
		files["fixtures/interaction-policy.json"],
		files["fixtures/conversational-routes.json"],
	); err != nil {
		return Bundle{}, err
	}

	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	manifestBytes, err := json.MarshalIndent(Manifest{
		BundleVersion: BundleVersion,
		SchemaVersion: 1,
		Compatibility: Compatibility,
		Artifacts:     artifacts,
	}, "", "  ")
	if err != nil {
		return Bundle{}, err
	}
	manifestBytes = append(manifestBytes, '\n')
	files["manifest.json"] = manifestBytes
	manifestSHA := sum(manifestBytes)

	handoffSource, err := os.ReadFile(filepath.Join(fixtureDir, "HERO-CODE-HANDOFF.md"))
	if err != nil {
		return Bundle{}, err
	}
	files["HERO-CODE-HANDOFF.md"] = append(append([]byte{}, handoffSource...), []byte(fmt.Sprintf(`

## Vendoring this conformance bundle

Vendor this entire directory as one unit. Validate every artifact hash in
`+"`manifest.json`"+` before decoding any fixture, then record the clean Hero
commit or release containing it in the consuming repository. Do not pin an
absolute checkout path or an uncommitted working tree.

- Bundle version: %d
- Attention schema version: 1
- Bundle manifest SHA-256: `+"`%s`"+`
- Runtime parity: HTTP and MCP contract discovery must advertise this exact
  bundle version and manifest hash.
- Forward compatibility: %s
`, BundleVersion, manifestSHA, Compatibility))...)

	return Bundle{Files: files, ManifestSHA256: manifestSHA}, nil
}

func Write(outputDir string, bundle Bundle) error {
	for path, data := range bundle.Files {
		target := filepath.Join(outputDir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func Check(outputDir string, bundle Bundle) error {
	if err := ValidateDirectory(outputDir); err != nil {
		return err
	}
	actual := make(map[string][]byte)
	err := filepath.WalkDir(outputDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(outputDir, path)
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
	return nil
}

func ValidateDirectory(dir string) error {
	manifestBytes, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.BundleVersion != BundleVersion || manifest.SchemaVersion != 1 {
		return fmt.Errorf("unsupported bundle or schema version")
	}

	expected := map[string]bool{"manifest.json": true, "HERO-CODE-HANDOFF.md": true}
	lastPath := ""
	for _, entry := range manifest.Artifacts {
		if entry.Path <= lastPath {
			return fmt.Errorf("artifacts are not strictly sorted at %s", entry.Path)
		}
		lastPath = entry.Path
		if expected[entry.Path] {
			return fmt.Errorf("duplicate or reserved artifact path %s", entry.Path)
		}
		expected[entry.Path] = true
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(entry.Path)))
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Path, err)
		}
		if sum(data) != entry.SHA256 {
			return fmt.Errorf("checksum mismatch for %s", entry.Path)
		}
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("decode %s: %w", entry.Path, err)
		}
		if entry.Schema != "" && !expectedOrPresent(dir, entry.Schema, manifest.Artifacts) {
			return fmt.Errorf("schema %s for %s is not bundled", entry.Schema, entry.Path)
		}
	}

	var actual []string
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
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
		actual = append(actual, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(actual)
	for _, path := range actual {
		if !expected[path] {
			return fmt.Errorf("extra unmanifested bundle artifact %s", path)
		}
		delete(expected, path)
	}
	if len(expected) != 0 {
		return fmt.Errorf("bundle is missing %d expected artifacts", len(expected))
	}
	return validateRouteInventory(dir)
}

func normalizeToolInventory(raw []byte) ([]byte, error) {
	var tools []map[string]any
	if err := json.Unmarshal(raw, &tools); err != nil {
		return nil, fmt.Errorf("decode tool definitions: %w", err)
	}
	for index, tool := range tools {
		name, _ := tool["name"].(string)
		description, _ := tool["description"].(string)
		if name == "" || description == "" || tool["inputSchema"] == nil || tool["annotations"] == nil || tool["_meta"] == nil {
			return nil, fmt.Errorf("tool definition %d is missing its name, description, complete input schema, annotations, or metadata", index)
		}
	}
	sort.Slice(tools, func(i, j int) bool {
		left, _ := tools[i]["name"].(string)
		right, _ := tools[j]["name"].(string)
		return left < right
	})
	encoded, err := json.MarshalIndent(map[string]any{"schema_version": 1, "tools": tools}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func jsonFiles(dir, suffix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func artifact(path, kind, purpose, schema string, data []byte) Artifact {
	return Artifact{Path: path, Kind: kind, Purpose: purpose, MediaType: "application/json", SHA256: sum(data), Schema: schema}
}

func sum(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}

func expectedOrPresent(dir, path string, artifacts []Artifact) bool {
	for _, artifact := range artifacts {
		if artifact.Path == path {
			_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(path)))
			return err == nil
		}
	}
	return false
}

func validateRouteInventory(dir string) error {
	inventory, err := os.ReadFile(filepath.Join(dir, "mcp-tools.json"))
	if err != nil {
		return err
	}
	policies, err := os.ReadFile(filepath.Join(dir, "fixtures", "interaction-policy.json"))
	if err != nil {
		return err
	}
	routes, err := os.ReadFile(filepath.Join(dir, "fixtures", "conversational-routes.json"))
	if err != nil {
		return err
	}
	return validateRouteInventoryBytes(inventory, policies, routes)
}

func validateRouteInventoryBytes(inventoryData, policyData, routeData []byte) error {
	var canonicalRoutes attentioncontract.ConversationalRouteFixture
	if err := json.Unmarshal(routeData, &canonicalRoutes); err != nil {
		return fmt.Errorf("decode conversational route fixture: %w", err)
	}
	if err := attentioncontract.ValidateConversationalRouteFixture(canonicalRoutes); err != nil {
		return fmt.Errorf("validate conversational route fixture: %w", err)
	}

	var inventory struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(inventoryData, &inventory); err != nil {
		return err
	}
	tools := make(map[string]bool)
	for _, tool := range inventory.Tools {
		tools[tool.Name] = true
	}

	var policies struct {
		Operations []struct {
			ID      string `json:"id"`
			Tool    string `json:"tool_name"`
			Action  string `json:"action_id"`
			Effect  string `json:"effect"`
			Consent string `json:"consent"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(policyData, &policies); err != nil {
		return err
	}
	operations := make(map[string]struct{ tool, action, effect, consent string })
	for _, operation := range policies.Operations {
		operations[operation.ID] = struct{ tool, action, effect, consent string }{
			operation.Tool, operation.Action, operation.Effect, operation.Consent,
		}
	}

	var routes struct {
		Cases []struct {
			ID          string `json:"id"`
			Category    string `json:"category"`
			Disposition string `json:"expected_disposition"`
			Surface     string `json:"expected_surface"`
			Operation   string `json:"expected_operation"`
			Tool        string `json:"expected_tool"`
			Action      string `json:"expected_action"`
			Command     string `json:"expected_command"`
			Effect      string `json:"expected_effect"`
			Consent     string `json:"expected_consent"`
			Mutations   int    `json:"expected_mutation_count"`
			Retry       string `json:"retry_expectation"`
			RetryCount  int    `json:"retry_expected_mutation_count"`
			ErrorCode   string `json:"expected_error_code"`
			Resolution  struct {
				ResolvedFacts []string `json:"resolved_facts"`
			} `json:"resolution"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(routeData, &routes); err != nil {
		return err
	}
	for _, route := range routes.Cases {
		if (route.Surface == "mcp_tool" || route.Surface == "advertised_action") && !tools[route.Tool] {
			return fmt.Errorf("%s references missing MCP tool %s", route.ID, route.Tool)
		}
		if route.Surface == "cli_workflow" {
			if !strings.HasPrefix(route.Command, "hero peer ") && route.Command != "hero handoff" {
				return fmt.Errorf("%s references invalid typed peering command", route.ID)
			}
			if route.Mutations != 1 || route.RetryCount != 0 {
				return fmt.Errorf("%s typed peering route has invalid mutation expectations", route.ID)
			}
			continue
		}
		if route.Operation == "" {
			if route.Mutations != 0 {
				return fmt.Errorf("%s non-dispatch route mutates %d times", route.ID, route.Mutations)
			}
			continue
		}
		policy, ok := operations[route.Operation]
		if !ok {
			return fmt.Errorf("%s references missing operation %s", route.ID, route.Operation)
		}
		if route.Tool != policy.tool || route.Action != policy.action || route.Effect != policy.effect || route.Consent != policy.consent {
			return fmt.Errorf("%s differs from published operation %s", route.ID, route.Operation)
		}
		replayedWrite := route.Category == "resilience" && route.Retry == "same_key_no_duplicate" &&
			stringSliceContains(route.Resolution.ResolvedFacts, "idempotency_key")
		switch {
		case route.ErrorCode != "" || route.Effect == "read" || replayedWrite:
			if route.Mutations != 0 {
				return fmt.Errorf("%s must mutate zero times", route.ID)
			}
		case route.Mutations != 1:
			return fmt.Errorf("%s successful write must mutate exactly once", route.ID)
		}
		if route.Retry == "refresh_then_retry" {
			if route.RetryCount != 1 {
				return fmt.Errorf("%s refreshed retry must mutate exactly once", route.ID)
			}
		} else if route.RetryCount != 0 {
			return fmt.Errorf("%s retry must not duplicate a mutation", route.ID)
		}
	}
	return nil
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
