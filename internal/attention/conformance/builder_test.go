package conformance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/serve"
)

func TestBuildIsDeterministicAndCheckedInBundleIsCurrent(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	toolJSON, err := json.Marshal(serve.AttentionToolDefinitions())
	if err != nil {
		t.Fatal(err)
	}
	first, err := Build(root, toolJSON)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(root, toolJSON)
	if err != nil {
		t.Fatal(err)
	}
	if first.ManifestSHA256 != second.ManifestSHA256 || len(first.Files) != len(second.Files) {
		t.Fatal("repeated builds differ")
	}
	for path, content := range first.Files {
		if !bytes.Equal(content, second.Files[path]) {
			t.Fatalf("repeated builds differ at %s", path)
		}
	}
	if err := Check(filepath.Join(root, filepath.FromSlash(BundlePath)), first); err != nil {
		t.Fatalf("checked-in bundle is stale: %v", err)
	}
}

func TestConsumerValidationRejectsMissingExtraStaleAndMalformedArtifacts(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	toolJSON, err := json.Marshal(serve.AttentionToolDefinitions())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Build(root, toolJSON)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid copied bundle", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		if err := ValidateDirectory(dir); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("missing", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(dir, "fixtures", "attention-snapshot.json")); err != nil {
			t.Fatal(err)
		}
		if err := ValidateDirectory(dir); err == nil {
			t.Fatal("expected missing artifact rejection")
		}
	})

	t.Run("extra", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "fixtures", "extra.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ValidateDirectory(dir); err == nil || !strings.Contains(err.Error(), "extra unmanifested") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "fixtures", "attention-snapshot.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ValidateDirectory(dir); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("malformed manifest", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ValidateDirectory(dir); err == nil || !strings.Contains(err.Error(), "decode manifest") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("reordered without regeneration", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		var manifest Manifest
		if err := json.Unmarshal(bundle.Files["manifest.json"], &manifest); err != nil {
			t.Fatal(err)
		}
		manifest.Artifacts[0], manifest.Artifacts[1] = manifest.Artifacts[1], manifest.Artifacts[0]
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(data, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ValidateDirectory(dir); err == nil || !strings.Contains(err.Error(), "not strictly sorted") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("checksum-valid invalid route disposition", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		rewriteArtifactJSON(t, dir, "fixtures/conversational-routes.json", func(value map[string]any) {
			cases := value["cases"].([]any)
			cases[0].(map[string]any)["expected_disposition"] = "future_dispatch"
		})
		if err := ValidateDirectory(dir); err == nil || !strings.Contains(err.Error(), "stable v1 disposition") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("checksum-valid route inventory inconsistency", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		rewriteArtifactJSON(t, dir, "fixtures/conversational-routes.json", func(value map[string]any) {
			cases := value["cases"].([]any)
			cases[0].(map[string]any)["expected_tool"] = "hero_mail_show"
		})
		if err := ValidateDirectory(dir); err == nil || !strings.Contains(err.Error(), "canonical operation policy") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("unknown inventory identifiers remain decodable and inert", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(dir, bundle); err != nil {
			t.Fatal(err)
		}
		rewriteArtifactJSON(t, dir, "mcp-tools.json", func(value map[string]any) {
			tools := value["tools"].([]any)
			data, err := json.Marshal(tools[0])
			if err != nil {
				t.Fatal(err)
			}
			var futureTool map[string]any
			if err := json.Unmarshal(data, &futureTool); err != nil {
				t.Fatal(err)
			}
			futureTool["name"] = "hero_future_quantum_write"
			futureTool["_meta"].(map[string]any)["hero.dev/operation_id"] = "future.operation"
			value["tools"] = append(tools, futureTool)
		})
		if err := ValidateDirectory(dir); err != nil {
			t.Fatalf("unknown additive inventory entry should remain decodable: %v", err)
		}

		rewriteArtifactJSON(t, dir, "fixtures/conversational-routes.json", func(value map[string]any) {
			cases := value["cases"].([]any)
			route := cases[0].(map[string]any)
			route["expected_operation"] = "future.operation"
			route["expected_tool"] = "hero_future_quantum_write"
		})
		if err := ValidateDirectory(dir); err == nil || !strings.Contains(err.Error(), "canonical Attention or peering operation") {
			t.Fatalf("unknown identifier became executable; error = %v", err)
		}
	})
}

func rewriteArtifactJSON(t *testing.T, dir, path string, mutate func(map[string]any)) {
	t.Helper()
	target := filepath.Join(dir, filepath.FromSlash(path))
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	mutate(value)
	data, err = json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	found := false
	for index := range manifest.Artifacts {
		if manifest.Artifacts[index].Path == path {
			manifest.Artifacts[index].SHA256 = sum(data)
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("manifest does not contain %s", path)
	}
	manifestData, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(manifestData, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
