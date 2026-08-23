package mailread

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
)

func TestGoldenFixturesSchemasDTOsAndChecksums(t *testing.T) {
	manifestData, err := os.ReadFile("testdata/v1/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest FixtureManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != SchemaVersion || len(manifest.Fixtures) != 19 {
		t.Fatalf("manifest version/count = %d/%d", manifest.SchemaVersion, len(manifest.Fixtures))
	}
	for _, fixture := range manifest.Fixtures {
		fixture := fixture
		t.Run(fixture.Path, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata/v1", fixture.Path))
			if err != nil {
				t.Fatal(err)
			}
			if got := SumSHA256(data); got != fixture.SHA256 {
				t.Fatalf("sha256 = %s; manifest = %s", got, fixture.SHA256)
			}
			schemaData, err := os.ReadFile(filepath.Join("schema/v1", fixture.Schema))
			if err != nil {
				t.Fatal(err)
			}
			var schema, value any
			if err := json.Unmarshal(schemaData, &schema); err != nil {
				t.Fatalf("schema: %v", err)
			}
			if err := json.Unmarshal(data, &value); err != nil {
				t.Fatalf("fixture: %v", err)
			}
			if err := validateFixtureSchema(schema, schema, value, "$"); err != nil {
				t.Fatal(err)
			}
			if err := decodeAndValidateFixture(fixture.Path, data); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConformanceBundleIsDeterministicCurrentAndIndependentlyHashed(t *testing.T) {
	first, err := BuildBundle(".")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildBundle(".")
	if err != nil {
		t.Fatal(err)
	}
	if first.ManifestSHA256 != second.ManifestSHA256 || first.ManifestSHA256 != ConformanceManifestSHA256 {
		t.Fatalf("bundle hashes = %s / %s; compiled = %s", first.ManifestSHA256, second.ManifestSHA256, ConformanceManifestSHA256)
	}
	if !reflect.DeepEqual(first.Files, second.Files) {
		t.Fatal("repeated bundle builds differ")
	}
	if err := CheckBundle("conformance/v1", first); err != nil {
		t.Fatalf("checked-in bundle: %v", err)
	}

	for path, want := range map[string]string{
		"../testdata/v1/manifest.json":    "059b5418cc2005d506b0ca718df1ed25109ed022e385c7b16bca3e3c4a0d8e07",
		"../conformance/v1/manifest.json": "2c29a1c6e04c3504969736494d0759f566a982d01c59ab7d8552c751a64b31fa",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := SumSHA256(data); got != want {
			t.Fatalf("released Attention artifact %s changed: %s", path, got)
		}
	}
}

func TestConformanceBundleConsumerValidationRejectsDrift(t *testing.T) {
	bundle, err := BuildBundle(".")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid copy", func(t *testing.T) {
		dir := t.TempDir()
		if err := WriteBundle(dir, bundle); err != nil {
			t.Fatal(err)
		}
		if err := ValidateBundle(dir); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		dir := t.TempDir()
		if err := WriteBundle(dir, bundle); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "fixtures", "detail.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ValidateBundle(dir); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("extra", func(t *testing.T) {
		dir := t.TempDir()
		if err := WriteBundle(dir, bundle); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "extra.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ValidateBundle(dir); err == nil || !strings.Contains(err.Error(), "extra unmanifested") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestFixtureSemanticsPinCompositeIdentityOrderingActionsAndMaximumBody(t *testing.T) {
	var cross ListResponse
	readFixture(t, "cross-project.json", &cross)
	if len(cross.Items) != 2 || cross.Items[0].MessageID != cross.Items[1].MessageID || cross.Items[0].ActivityAt != cross.Items[1].ActivityAt {
		t.Fatalf("cross-project duplicate/equal-time fixture lost its invariant: %#v", cross.Items)
	}
	if cross.Items[0].Project.PeerID != "peer_alpha" || cross.Items[1].Project.PeerID != "peer_beta" {
		t.Fatalf("equal-time stable peer ordering changed: %#v", cross.Items)
	}

	var actions ListResponse
	readFixture(t, "canonical-actions.json", &actions)
	gotIDs := make([]string, 0, len(actions.Items[0].Actions))
	for _, descriptor := range actions.Items[0].Actions {
		gotIDs = append(gotIDs, descriptor.ID)
		if descriptor.OperationID == "" || descriptor.Effect == "" || descriptor.Consent == "" || len(descriptor.InputSchema) == 0 || !descriptor.RequiresIdempotency {
			t.Fatalf("incomplete descriptor: %#v", descriptor)
		}
	}
	wantIDs := []string{ActionMarkRead, ActionAcknowledge, ActionDismiss, ActionPromote, ActionAddToToday, ActionReply}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("action IDs = %v; want %v", gotIDs, wantIDs)
	}

	var detail DetailResponse
	readFixture(t, "max-body.json", &detail)
	if detail.Envelope == nil || len(detail.Envelope.Body) != attention.MaxBodyBytes || !strings.HasSuffix(detail.Envelope.Body, "CANARY!") || detail.Envelope.Body[len(detail.Envelope.Body)-1] != '!' {
		t.Fatalf("maximum body/canary invariant failed: envelope=%v", detail.Envelope != nil)
	}
	encoded, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip DetailResponse
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.Envelope.Body != detail.Envelope.Body {
		t.Fatal("maximum valid body did not round-trip byte-for-byte")
	}
}

func TestContractResponsePinsPublishedBundle(t *testing.T) {
	response := ContractResponse{SchemaVersion: SchemaVersion, BundleVersion: BundleVersion, BundleManifestSHA256: ConformanceManifestSHA256, Compatibility: Compatibility}
	if err := ValidateContractResponse(response); err != nil {
		t.Fatalf("published contract: %#v", err)
	}
}

func decodeAndValidateFixture(name string, data []byte) error {
	var target any
	var validate func() *attention.ContractError
	switch name {
	case "list-request.json":
		value := &ListRequest{}
		target, validate = value, func() *attention.ContractError { return ValidateListRequest(*value) }
	case "list-page.json", "threaded-page.json", "canonical-actions.json", "error-validation.json", "error-unavailable.json", "unknown-additive.json", "cross-project.json":
		value := &ListResponse{}
		target, validate = value, func() *attention.ContractError { return ValidateListResponse(*value) }
	case "detail.json", "receipt-state.json", "max-body.json", "error-missing.json":
		value := &DetailResponse{}
		target, validate = value, func() *attention.ContractError { return ValidateDetailResponse(*value) }
	case "action-request.json":
		value := &ActionRequest{}
		target, validate = value, func() *attention.ContractError { return ValidateActionRequest(*value) }
	case "action-success.json", "action-failure.json", "error-incompatible-version.json":
		value := &ActionResponse{}
		target, validate = value, func() *attention.ContractError { return ValidateActionResponse(*value) }
	case "reply-request.json":
		value := &ReplyRequest{}
		target, validate = value, func() *attention.ContractError { return ValidateReplyRequest(*value) }
	case "reply-success.json", "reply-failure.json":
		value := &ReplyResponse{}
		target, validate = value, func() *attention.ContractError { return ValidateReplyResponse(*value) }
	default:
		return fmt.Errorf("fixture %q has no DTO mapping", name)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	if err := validate(); err != nil {
		return fmt.Errorf("DTO validation: %w (%s)", err, err.Field)
	}
	return nil
}

func readFixture(t *testing.T, name string, target any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata/v1", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func validateFixtureSchema(root, raw, value any, path string) error {
	schema, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: schema is not an object", path)
	}
	if ref, ok := schema["$ref"].(string); ok {
		const prefix = "#/$defs/"
		if !strings.HasPrefix(ref, prefix) {
			return fmt.Errorf("%s: unsupported ref %q", path, ref)
		}
		defs := root.(map[string]any)["$defs"].(map[string]any)
		return validateFixtureSchema(root, defs[strings.TrimPrefix(ref, prefix)], value, path)
	}
	if variants, ok := schema["anyOf"].([]any); ok {
		matched := false
		for _, variant := range variants {
			if validateFixtureSchema(root, variant, value, path) == nil {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s: matched no anyOf branch", path)
		}
	}
	if expected, ok := schema["type"].(string); ok && !matchesFixtureType(expected, value) {
		return fmt.Errorf("%s: expected %s, got %T", path, expected, value)
	}
	if constant, ok := schema["const"]; ok && !reflect.DeepEqual(constant, value) {
		return fmt.Errorf("%s: expected constant %v", path, constant)
	}
	if enum, ok := schema["enum"].([]any); ok {
		found := false
		for _, candidate := range enum {
			found = found || reflect.DeepEqual(candidate, value)
		}
		if !found {
			return fmt.Errorf("%s: value %v is outside enum", path, value)
		}
	}
	if object, ok := value.(map[string]any); ok {
		if required, ok := schema["required"].([]any); ok {
			for _, key := range required {
				if _, exists := object[key.(string)]; !exists {
					return fmt.Errorf("%s: missing required %s", path, key)
				}
			}
		}
		if properties, ok := schema["properties"].(map[string]any); ok {
			for key, property := range properties {
				if child, exists := object[key]; exists {
					if err := validateFixtureSchema(root, property, child, path+"."+key); err != nil {
						return err
					}
				}
			}
		}
	}
	if array, ok := value.([]any); ok {
		if max, ok := schema["maxItems"].(float64); ok && len(array) > int(max) {
			return fmt.Errorf("%s: exceeds maxItems", path)
		}
		if items, ok := schema["items"]; ok {
			for i, child := range array {
				if err := validateFixtureSchema(root, items, child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	}
	if text, ok := value.(string); ok {
		if min, ok := schema["minLength"].(float64); ok && len([]rune(text)) < int(min) {
			return fmt.Errorf("%s: below minLength", path)
		}
		if max, ok := schema["maxLength"].(float64); ok && len([]rune(text)) > int(max) {
			return fmt.Errorf("%s: exceeds maxLength", path)
		}
		if max, ok := schema["x-maxBytes"].(float64); ok && len([]byte(text)) > int(max) {
			return fmt.Errorf("%s: exceeds x-maxBytes", path)
		}
		if pattern, ok := schema["pattern"].(string); ok && !regexp.MustCompile(pattern).MatchString(text) {
			return fmt.Errorf("%s: does not match pattern", path)
		}
		if schema["format"] == "date-time" {
			if _, err := time.Parse(time.RFC3339Nano, text); err != nil {
				return fmt.Errorf("%s: invalid date-time", path)
			}
		}
	}
	if number, ok := value.(float64); ok {
		if min, exists := schema["minimum"].(float64); exists && number < min {
			return fmt.Errorf("%s: below minimum", path)
		}
		if max, exists := schema["maximum"].(float64); exists && number > max {
			return fmt.Errorf("%s: above maximum", path)
		}
	}
	return nil
}

func matchesFixtureType(expected string, value any) bool {
	switch expected {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == float64(int64(number))
	default:
		return false
	}
}
