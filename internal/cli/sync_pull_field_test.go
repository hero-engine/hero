package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
)

// mockPullTracker is a no-network tracker.Tracker used to exercise the
// field-level pull path. Only GetFields / Name carry behavior; the rest
// satisfy the interface and are never called by the pull path.
type mockPullTracker struct {
	fields    map[string]tracker.Value
	fieldsErr error
}

func (m *mockPullTracker) GetFields(issueID string) (map[string]tracker.Value, error) {
	if m.fieldsErr != nil {
		return nil, m.fieldsErr
	}
	return m.fields, nil
}

func (m *mockPullTracker) Name() string { return "mock" }

// --- unused interface methods (never hit by the pull path) ---

func (m *mockPullTracker) CreateIssue(s *spec.Spec) (string, error)              { return "", nil }
func (m *mockPullTracker) UpdateStatus(issueID string, status spec.Status) error { return nil }
func (m *mockPullTracker) UpdateSize(issueID, localTier string) error            { return nil }
func (m *mockPullTracker) GetIssue(issueID string) (*tracker.Issue, error)       { return nil, nil }
func (m *mockPullTracker) UpdateFields(issueID string, f map[string]tracker.Value) error {
	return nil
}
func (m *mockPullTracker) ListIssues(label string, limit int) ([]tracker.Issue, error) {
	return nil, nil
}
func (m *mockPullTracker) Search(q tracker.SearchQuery) ([]tracker.Issue, error) { return nil, nil }
func (m *mockPullTracker) AddComment(issueID, body string) error                 { return nil }
func (m *mockPullTracker) AttachFile(issueID, filePath, fileName string) error   { return nil }
func (m *mockPullTracker) SupportsHierarchy() bool                               { return false }
func (m *mockPullTracker) MapSize(localTier string) (string, error)              { return "", nil }
func (m *mockPullTracker) ReverseMapSize(trackerValue string) (string, error)    { return "", nil }

// withMockTracker swaps the pull tracker factory for a mock and restores
// it on cleanup. Returns the mock so the test can program its fields.
func withMockTracker(t *testing.T, m *mockPullTracker) {
	t.Helper()
	orig := newTrackerForPull
	newTrackerForPull = func(cfg config.Config, projectRoot string) (tracker.Tracker, error) {
		return m, nil
	}
	t.Cleanup(func() { newTrackerForPull = orig })
}

// addTrackerBackedSpec writes a spec with a tracker_id under a github
// tracker config so the field-pull path resolves a real tracker.
func addTrackerBackedSpec(env *testEnv, slug, trackerID string) {
	env.t.Helper()
	writeTrackerConfig(env, "github", "acme/widgets")
	env.t.Setenv("HERO_TEST_TOKEN", "fake-token")
	body := "---\ntitle: " + slug + "\ntype: feature\nstatus: planning\n"
	if trackerID != "" {
		body += "tracker_id: " + trackerID + "\n"
	}
	body += "---\n# " + slug + "\n"
	env.addSpec("planning/features/"+slug+"/spec.md", body)
}

// decodePullEnvelope parses the single-line JSON envelope from stdout.
func decodePullEnvelope(t *testing.T, out string) pullEnvelope {
	t.Helper()
	line := strings.TrimSpace(out)
	if i := strings.Index(line, "{"); i >= 0 {
		line = line[i:]
	}
	var env pullEnvelope
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		t.Fatalf("decode envelope from %q: %v", out, err)
	}
	return env
}

// TestSyncPull_Field_ReturnsTrackerValue: --field <name> returns the
// mocked current tracker value in a {slug,field,value} envelope.
func TestSyncPull_Field_ReturnsTrackerValue(t *testing.T) {
	env := newTestEnv(t)
	addTrackerBackedSpec(env, "csv-export", "42")
	withMockTracker(t, &mockPullTracker{
		fields: map[string]tracker.Value{
			"priority": tracker.StringValue("P0"),
		},
	})

	out, err := runCmd("sync", "pull", "csv-export", "--field", "priority", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := decodePullEnvelope(t, out)
	if got.Slug != "csv-export" || got.Field != "priority" {
		t.Fatalf("slug/field = %q/%q, want csv-export/priority", got.Slug, got.Field)
	}
	if got.Value == nil || *got.Value != "P0" {
		t.Fatalf("value = %v, want \"P0\"", got.Value)
	}
}

func TestSyncPull_Field_ReturnsFullTrackerDescription(t *testing.T) {
	env := newTestEnv(t)
	addTrackerBackedSpec(env, "morph-14171", "MORPH-14171")
	description := "## Environment\n\n" + strings.Repeat("Complete Jira evidence. ", 80)
	withMockTracker(t, &mockPullTracker{
		fields: map[string]tracker.Value{
			"description": tracker.StringValue(description),
		},
	})

	out, err := runCmd("sync", "pull", "morph-14171", "--field", "description", "--json")
	if err != nil {
		t.Fatal(err)
	}
	got := decodePullEnvelope(t, out)
	if got.Value == nil || *got.Value != description {
		t.Fatal("description pull did not return the complete tracker body")
	}
}

// TestSyncPull_Field_NullWhenTrackerHasNoValue: tracker has no value for
// the field → value is null (Swift treats null as skip).
func TestSyncPull_Field_NullWhenTrackerHasNoValue(t *testing.T) {
	env := newTestEnv(t)
	addTrackerBackedSpec(env, "csv-export", "42")
	withMockTracker(t, &mockPullTracker{
		fields: map[string]tracker.Value{
			"title": tracker.StringValue("Some Title"),
		},
	})

	out, err := runCmd("sync", "pull", "csv-export", "--field", "priority", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := decodePullEnvelope(t, out)
	if got.Field != "priority" {
		t.Fatalf("field = %q, want priority", got.Field)
	}
	if got.Value != nil {
		t.Fatalf("value = %v, want null", *got.Value)
	}
	// The raw JSON must carry an explicit null, not omit the key.
	if !strings.Contains(out, `"value":null`) {
		t.Errorf("envelope %q missing explicit \"value\":null", strings.TrimSpace(out))
	}
}

// TestSyncPull_Field_NoTrackerID: spec has no tracker_id → graceful
// null-value envelope, no crash, no error.
func TestSyncPull_Field_NoTrackerID(t *testing.T) {
	env := newTestEnv(t)
	addTrackerBackedSpec(env, "csv-export", "") // no tracker_id
	// GetFields must never be reached; program a tracker that would fail
	// if it were.
	withMockTracker(t, &mockPullTracker{fieldsErr: errFromAuth()})

	out, err := runCmd("sync", "pull", "csv-export", "--field", "priority", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := decodePullEnvelope(t, out)
	if got.Value != nil {
		t.Fatalf("value = %v, want null for no-tracker_id spec", *got.Value)
	}
	if got.Status != "no-tracker" {
		t.Errorf("status = %q, want no-tracker", got.Status)
	}
}

// TestSyncPull_Field_AuthErrorExitsTwo: GetFields returns a 401 (auth)
// FieldError → exit code 2 path is taken and a failure envelope emitted.
func TestSyncPull_Field_AuthErrorExitsTwo(t *testing.T) {
	env := newTestEnv(t)
	addTrackerBackedSpec(env, "csv-export", "42")
	withMockTracker(t, &mockPullTracker{fieldsErr: errFromAuth()})

	var exitCode int
	origExit := osExitPull
	osExitPull = func(code int) { exitCode = code }
	t.Cleanup(func() { osExitPull = origExit })

	out, _ := runCmd("sync", "pull", "csv-export", "--field", "priority", "--json")
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	got := decodePullEnvelope(t, out)
	if got.Status != "failed" {
		t.Errorf("status = %q, want failed", got.Status)
	}
	if got.Value != nil {
		t.Errorf("value = %v, want null on failure", *got.Value)
	}
}

// errFromAuth builds a 401 auth FieldError equivalent to what an adapter
// surfaces on a credential failure.
func errFromAuth() error {
	return &tracker.FieldError{
		Kind:    tracker.FieldErrorAuth,
		Status:  401,
		Message: "github API returned 401 — check tracker credentials",
	}
}

// TestSyncPull_StatusOnly_Unregressed: without --field, the existing
// status-only pull still validates workspace/tracker/tracker_id and is
// unchanged (no envelope path). A spec without tracker_id errors as before.
func TestSyncPull_StatusOnly_Unregressed(t *testing.T) {
	env := newTestEnv(t)
	writeTrackerConfig(env, "github", "acme/widgets")
	t.Setenv("HERO_TEST_TOKEN", "fake-token")
	env.addSpec("planning/features/no-id/spec.md", `---
title: No ID
type: feature
status: planning
---
# No ID
`)
	specPath := env.heroDir + "/planning/features/no-id/spec.md"
	_, err := runCmd("sync", "pull", specPath)
	if err == nil {
		t.Fatal("expected error for no tracker_id (status-only path)")
	}
	if !strings.Contains(err.Error(), "no tracker_id") {
		t.Errorf("error = %q, want 'no tracker_id'", err.Error())
	}
}
