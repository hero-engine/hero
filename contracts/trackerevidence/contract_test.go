package trackerevidence

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type consumerFixtures struct {
	Version  string                     `json:"version"`
	Request  Request                    `json:"request"`
	Statuses map[string]json.RawMessage `json:"statuses"`
	Errors   map[string]Error           `json:"errors"`
}

func TestConsumerFixturesCoverV1StatesAndErrors(t *testing.T) {
	b, err := ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}

	var fixture consumerFixtures
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != Version {
		t.Fatalf("version = %q", fixture.Version)
	}
	if !fixture.Request.AttachmentsEnabled() {
		t.Fatal("fixture request unexpectedly disables attachments")
	}

	wantStates := map[string]State{
		"fetched":     StateFetched,
		"refreshed":   StateRefreshed,
		"current":     StateCurrent,
		"unsupported": StateUnsupported,
		"unavailable": StateUnavailable,
	}
	if len(fixture.Statuses) != len(wantStates) {
		t.Fatalf("statuses = %d, want %d", len(fixture.Statuses), len(wantStates))
	}
	for name, want := range wantStates {
		raw, ok := fixture.Statuses[name]
		if !ok {
			t.Fatalf("missing status fixture %q", name)
		}
		var status Status
		if err := json.Unmarshal(raw, &status); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if status.Version != Version || status.Status != want {
			t.Fatalf("%s = version %q status %q", name, status.Version, status.Status)
		}
	}

	wantErrors := map[string]ErrorCode{
		"spec_not_found":       ErrorSpecNotFound,
		"tracker_unlinked":     ErrorTrackerUnlinked,
		"ambiguous_connection": ErrorAmbiguousConnection,
		"unsupported_provider": ErrorUnsupportedProvider,
		"provider_unavailable": ErrorProviderUnavailable,
		"invalid_manifest":     ErrorInvalidManifest,
		"payload_missing":      ErrorPayloadMissing,
		"payload_corrupt":      ErrorPayloadCorrupt,
		"cancelled":            ErrorCancelled,
		"write_failed":         ErrorWriteFailed,
	}
	if len(fixture.Errors) != len(wantErrors) {
		t.Fatalf("errors = %d, want %d", len(fixture.Errors), len(wantErrors))
	}
	for name, want := range wantErrors {
		got, ok := fixture.Errors[name]
		if !ok {
			t.Fatalf("missing error fixture %q", name)
		}
		if got.Code != want {
			t.Fatalf("%s code = %q, want %q", name, got.Code, want)
		}
	}
}

func TestManifestFixtureUsesExactAllowlist(t *testing.T) {
	b, err := ManifestFixture()
	if err != nil {
		t.Fatal(err)
	}

	var manifest Manifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != Version {
		t.Fatalf("version = %q", manifest.Version)
	}

	want := []string{
		"attachment_count",
		"content_sha256",
		"evidence_path",
		"issue_id",
		"omission_count",
		"provider",
		"retrieved_at",
		"tracker_updated_at",
		"version",
	}
	assertJSONKeys(t, b, want)
}

func TestStatusAndManifestHaveClosedSafeShapes(t *testing.T) {
	status := Status{
		Version:          Version,
		Status:           StateCurrent,
		Provider:         "jira",
		ConnectionID:     "jira-main",
		SpecSlug:         "example",
		IssueID:          "MORPH-297",
		TrackerUpdatedAt: "2026-07-20T17:42:19.123-0600",
		ContentSHA256:    strings.Repeat("a", 64),
		ManifestPath:     ".hero/planning/bugs/example/tracker-evidence.json",
		EvidencePath:     ".hero/planning/bugs/example/.tracker-evidence/evidence.json",
		AttachmentCount:  2,
		OmissionCount:    1,
		CacheHit:         true,
		Error: &Error{
			Code:      ErrorProviderUnavailable,
			Message:   "tracker evidence is temporarily unavailable",
			Retryable: true,
		},
	}
	b, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONKeys(t, b, []string{
		"attachment_count",
		"cache_hit",
		"connection_id",
		"content_sha256",
		"error",
		"evidence_path",
		"issue_id",
		"manifest_path",
		"omission_count",
		"provider",
		"spec_slug",
		"status",
		"tracker_updated_at",
		"version",
	})

	manifestType := reflect.TypeFor[Manifest]()
	statusType := reflect.TypeFor[Status]()
	for _, typ := range []reflect.Type{manifestType, statusType} {
		for i := 0; i < typ.NumField(); i++ {
			name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
			for _, forbidden := range []string{
				"raw", "title", "description", "comment", "changelog", "url",
				"person", "assignee", "reporter", "filename", "credential",
				"token", "authorization", "cookie",
			} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("%s contains forbidden JSON field %q", typ.Name(), name)
				}
			}
		}
	}
}

func TestEncodingJSONIgnoresUnknownFields(t *testing.T) {
	var status Status
	if err := json.Unmarshal([]byte(`{
		"version":"tracker-evidence/v1",
		"status":"current",
		"future_status_field":{"nested":true},
		"error":{"code":"provider_unavailable","message":"safe","retryable":true,"future_error_field":7}
	}`), &status); err != nil {
		t.Fatal(err)
	}
	if status.Version != Version || status.Status != StateCurrent {
		t.Fatalf("status = %#v", status)
	}
	if status.Error == nil || status.Error.Code != ErrorProviderUnavailable {
		t.Fatalf("error = %#v", status.Error)
	}

	var manifest Manifest
	if err := json.Unmarshal([]byte(`{
		"version":"tracker-evidence/v1",
		"provider":"jira",
		"issue_id":"MORPH-297",
		"future_manifest_field":[1,2,3]
	}`), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Provider != "jira" || manifest.IssueID != "MORPH-297" {
		t.Fatalf("manifest = %#v", manifest)
	}
}

func TestRequestAttachmentDefault(t *testing.T) {
	if !(Request{}).AttachmentsEnabled() {
		t.Fatal("omitted include_attachments must default true")
	}
	enabled := true
	if !(Request{IncludeAttachments: &enabled}).AttachmentsEnabled() {
		t.Fatal("explicit true must enable attachments")
	}
	disabled := false
	if (Request{IncludeAttachments: &disabled}).AttachmentsEnabled() {
		t.Fatal("explicit false must disable attachments")
	}
}

func TestConsumerFixtureReturnsIsolatedCopy(t *testing.T) {
	first, err := ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	second, err := ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	first[0] = 'x'
	if second[0] == 'x' {
		t.Fatal("fixture calls share mutable storage")
	}
}

func assertJSONKeys(t *testing.T, b []byte, want []string) {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.Unmarshal(b, &value); err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(value))
	for key := range value {
		got = append(got, key)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON keys = %v, want %v", got, want)
	}
}
