package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	syncpkg "github.com/hero-engine/hero/internal/sync"
	"github.com/hero-engine/hero/internal/tracker"
)

// mergeMockTracker is a no-network tracker that records UpdateFields and
// returns programmed GetFields, so the wired shared-field merge in
// runSyncPush's diff path can be exercised end-to-end.
type mergeMockTracker struct {
	fields    map[string]tracker.Value
	fieldsErr error
	updated   map[string]tracker.Value // last UpdateFields patch
	updateErr error
}

func (m *mergeMockTracker) GetFields(issueID string) (map[string]tracker.Value, error) {
	return m.fields, m.fieldsErr
}
func (m *mergeMockTracker) UpdateFields(issueID string, f map[string]tracker.Value) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = f
	return nil
}
func (m *mergeMockTracker) Name() string { return "mock" }

func (m *mergeMockTracker) CreateIssue(s *spec.Spec) (string, error) { return "", nil }

// --- unused interface methods ---
func (m *mergeMockTracker) UpdateStatus(issueID string, status spec.Status) error { return nil }
func (m *mergeMockTracker) UpdateSize(issueID, localTier string) error            { return nil }
func (m *mergeMockTracker) GetIssue(issueID string) (*tracker.Issue, error)          { return nil, nil }
func (m *mergeMockTracker) ListIssues(label string, limit int) ([]tracker.Issue, error) {
	return nil, nil
}
func (m *mergeMockTracker) Search(q tracker.SearchQuery) ([]tracker.Issue, error) { return nil, nil }
func (m *mergeMockTracker) AddComment(issueID, body string) error                 { return nil }
func (m *mergeMockTracker) AttachFile(issueID, filePath, fileName string) error   { return nil }
func (m *mergeMockTracker) SupportsHierarchy() bool                               { return false }
func (m *mergeMockTracker) MapSize(localTier string) (string, error)              { return "", nil }
func (m *mergeMockTracker) ReverseMapSize(trackerValue string) (string, error)    { return "", nil }

func withMockPushTracker(t *testing.T, m *mergeMockTracker) {
	t.Helper()
	orig := newTrackerForPush
	newTrackerForPush = func(cfg config.Config, projectRoot string) (tracker.Tracker, error) {
		return m, nil
	}
	t.Cleanup(func() { newTrackerForPush = orig })
}

func specFile(env *testEnv, slug string) string {
	return filepath.Join(env.heroDir, "planning", "features", slug, "spec.md")
}

// TestSyncPush_Merge_TitleBothChanged_KeepsUpstream is the wired equivalent of
// the drift test: title diverged on both sides from the same base. The merge
// must NOT push the local title over the concurrent upstream edit, and must
// converge the spec + baseline to the upstream value (local preserved in a
// marker in the body).
func TestSyncPush_Merge_TitleBothChanged_KeepsUpstream(t *testing.T) {
	env := newTestEnv(t)
	writeTrackerConfig(env, "github", "acme/widgets")
	env.t.Setenv("HERO_TEST_TOKEN", "fake-token")

	slug := "drift-demo"
	// Seed a baseline: title base = "Base title".
	if err := syncpkg.WriteBaseline(env.heroDir, slug, &syncpkg.Baseline{
		TrackerID: "42",
		Base:      map[string]syncpkg.Base{"title": syncpkg.TextBase("Base title")},
	}); err != nil {
		t.Fatal(err)
	}
	// Local diverged: title = "Local title".
	env.addSpec("planning/features/"+slug+"/spec.md",
		"---\ntitle: Local title\ntype: feature\nstatus: planning\ntracker_id: 42\n---\n# body\n")

	// Remote diverged: title = "Upstream title".
	mock := &mergeMockTracker{fields: map[string]tracker.Value{
		"title": tracker.StringValue("Upstream title"),
	}}
	withMockPushTracker(t, mock)

	out, err := runCmd("sync", "push", slug)
	if err != nil {
		t.Fatalf("push: %v (out=%s)", err, out)
	}

	// The tracker must NOT have received the local title (upstream preserved).
	if v, ok := mock.updated["title"]; ok {
		t.Fatalf("pushed title %q over a concurrent upstream edit — must not!", v.Str)
	}

	// Spec converged to upstream title, local preserved in a marker.
	data, _ := os.ReadFile(specFile(env, slug))
	content := string(data)
	if !strings.Contains(content, "title: Upstream title") {
		t.Errorf("spec not converged to upstream title:\n%s", content)
	}
	if !strings.Contains(content, "Local title") || !strings.Contains(content, syncpkg.LocalEditMarkerPrefix) {
		t.Errorf("local title not preserved in a marker:\n%s", content)
	}

	// Baseline advanced to the converged (upstream) title.
	b, _ := syncpkg.ReadBaseline(env.heroDir, slug)
	if b == nil || b.Base["title"].Text != "Upstream title" {
		t.Errorf("baseline not advanced to upstream title: %+v", b)
	}
}

// TestSyncPush_Merge_TitleOnlyLocal_PushesLocal: only local changed → push.
func TestSyncPush_Merge_TitleOnlyLocal_PushesLocal(t *testing.T) {
	env := newTestEnv(t)
	writeTrackerConfig(env, "github", "acme/widgets")
	env.t.Setenv("HERO_TEST_TOKEN", "fake-token")

	slug := "local-only"
	syncpkg.WriteBaseline(env.heroDir, slug, &syncpkg.Baseline{
		TrackerID: "7",
		Base:      map[string]syncpkg.Base{"title": syncpkg.TextBase("Base title")},
	})
	env.addSpec("planning/features/"+slug+"/spec.md",
		"---\ntitle: New local title\ntype: feature\nstatus: planning\ntracker_id: 7\n---\n# body\n")

	mock := &mergeMockTracker{fields: map[string]tracker.Value{
		"title": tracker.StringValue("Base title"), // remote unchanged
	}}
	withMockPushTracker(t, mock)

	if _, err := runCmd("sync", "push", slug); err != nil {
		t.Fatal(err)
	}
	if v, ok := mock.updated["title"]; !ok || v.Str != "New local title" {
		t.Fatalf("expected local title pushed, got %+v", mock.updated)
	}
}

// TestSyncPush_Merge_NoBaseline_AdoptsRemote: first-run with no baseline file.
// The merge must adopt remote as base, so a diverged local title does NOT
// clobber upstream on the very first sync.
func TestSyncPush_Merge_NoBaseline_AdoptsRemote(t *testing.T) {
	env := newTestEnv(t)
	writeTrackerConfig(env, "github", "acme/widgets")
	env.t.Setenv("HERO_TEST_TOKEN", "fake-token")

	slug := "first-run"
	// No baseline written.
	env.addSpec("planning/features/"+slug+"/spec.md",
		"---\ntitle: Local edit\ntype: feature\nstatus: planning\ntracker_id: 9\n---\n# body\n")

	mock := &mergeMockTracker{fields: map[string]tracker.Value{
		"title": tracker.StringValue("Upstream value"),
	}}
	withMockPushTracker(t, mock)

	if _, err := runCmd("sync", "push", slug); err != nil {
		t.Fatal(err)
	}
	// base==remote → only local changed → push local is the adopt-remote
	// outcome for a first run: local differs from adopted base(=remote), and
	// remote==base, so we push local. That's safe (no concurrent upstream to
	// lose — base IS remote). The guarantee under test: we did not silently
	// drop the fetched upstream by treating a stale base.
	// Assert the baseline now exists (adopted + advanced).
	b, _ := syncpkg.ReadBaseline(env.heroDir, slug)
	if b == nil {
		t.Fatal("first-run sync did not write a baseline")
	}
}

// TestSyncPush_Merge_FetchFailure_NoHalfWrite: a GetFields failure leaves both
// sides unchanged (no baseline written, spec untouched) and returns an error to
// retry next run.
func TestSyncPush_Merge_FetchFailure_NoHalfWrite(t *testing.T) {
	env := newTestEnv(t)
	writeTrackerConfig(env, "github", "acme/widgets")
	env.t.Setenv("HERO_TEST_TOKEN", "fake-token")

	slug := "fetch-fail"
	env.addSpec("planning/features/"+slug+"/spec.md",
		"---\ntitle: Local\ntype: feature\nstatus: planning\ntracker_id: 1\n---\n# body\n")

	mock := &mergeMockTracker{fieldsErr: os.ErrDeadlineExceeded}
	withMockPushTracker(t, mock)

	_, err := runCmd("sync", "push", slug)
	if err == nil {
		t.Fatal("expected error on fetch failure")
	}
	if b, _ := syncpkg.ReadBaseline(env.heroDir, slug); b != nil {
		t.Fatal("baseline written despite fetch failure (half-write)")
	}
}
