package mailthread

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGoldenFixturesDecodeAndValidate(t *testing.T) {
	read := func(name string, target any) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("testdata", "v1", name))
		if err != nil || json.Unmarshal(data, target) != nil {
			t.Fatalf("read %s: %v", name, err)
		}
	}
	for _, name := range []string{"state-open.json", "state-resolved.json", "state-archived.json", "unknown-additive.json"} {
		var value ThreadView
		read(name, &value)
		if err := ValidateThreadView(value); err != nil {
			t.Fatalf("%s: %v (%s)", name, err, err.Field)
		}
	}
	var capabilities CapabilitySet
	read("canonical-actions.json", &capabilities)
	if err := ValidateCapabilitySet(capabilities); err != nil {
		t.Fatal(err)
	}
	var request ActionRequest
	read("action-request.json", &request)
	if err := ValidateActionRequest(request); err != nil {
		t.Fatal(err)
	}
	var success ActionResponse
	read("action-success.json", &success)
	if err := ValidateActionResponse(success); err != nil {
		t.Fatal(err)
	}
	var failures []ActionResponse
	read("errors.json", &failures)
	for _, failure := range failures {
		if err := ValidateActionResponse(failure); err != nil {
			t.Fatal(err)
		}
	}
	var migrations []MigrationResult
	read("migration.json", &migrations)
	for _, migration := range migrations {
		if err := ValidateMigrationResult(migration); err != nil {
			t.Fatal(err)
		}
	}
	var list ThreadListResponse
	read("thread-list.json", &list)
	if err := ValidateThreadListResponse(list); err != nil {
		t.Fatal(err)
	}
	var detail ThreadDetailResponse
	read("thread-detail.json", &detail)
	if err := ValidateThreadDetailResponse(detail); err != nil {
		t.Fatal(err)
	}
}

func TestConformanceBundleIsDeterministicAndPinned(t *testing.T) {
	first, err := BuildBundle(".")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildBundle(".")
	if err != nil || first.ManifestSHA256 != second.ManifestSHA256 {
		t.Fatalf("non-deterministic bundle: %s / %s / %v", first.ManifestSHA256, second.ManifestSHA256, err)
	}
	if first.ManifestSHA256 != ConformanceManifestSHA256 {
		t.Fatalf("bundle hash = %s; compiled = %s", first.ManifestSHA256, ConformanceManifestSHA256)
	}
	if err := CheckBundle("conformance/v1", first); err != nil {
		t.Fatal(err)
	}
	response := ContractResponse{SchemaVersion: 1, BundleVersion: 1, BundleManifestSHA256: first.ManifestSHA256, Compatibility: Compatibility}
	if err := ValidateContractResponse(response); err != nil {
		t.Fatal(err)
	}
}
