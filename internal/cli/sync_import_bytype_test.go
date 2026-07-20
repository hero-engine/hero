package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
)

// byTypeMockTracker records each Search query and returns a canned
// result keyed by the query's IssueType (the type-label filter the
// per-type effective filter carries). Only Search / Name carry behavior.
type byTypeMockTracker struct {
	queries []tracker.SearchQuery
	byType  map[string][]tracker.Issue
}

func TestFetchByTypeUnion_ReportsEffectiveRawJQLAndCounts(t *testing.T) {
	jql := "project = MORPH AND issuetype = Bug AND status NOT IN (Done, Rejected, Cancelled)"
	cfg := config.Config{Import: &config.ImportConfig{ByType: map[string]*config.ImportFilter{
		"bug": {JQL: jql},
	}}}
	mock := &byTypeMockTracker{byType: map[string][]tracker.Issue{
		"Bug": {{ID: "MORPH-1"}, {ID: "MORPH-2"}},
	}}

	out := captureStdout(func() {
		_, _ = fetchByTypeUnion(cfg, mock, 5000)
	})
	if !strings.Contains(out, "JQL: "+jql) || !strings.Contains(out, "Matched: 2, added after dedup: 2") {
		t.Fatalf("per-type output did not expose effective JQL and count:\n%s", out)
	}
	if strings.Contains(out, "Assignee: unassigned") || strings.Contains(out, "Status: New") {
		t.Fatalf("raw JQL output misleadingly displayed ignored synthesized filters:\n%s", out)
	}
}

func (m *byTypeMockTracker) Search(q tracker.SearchQuery) ([]tracker.Issue, error) {
	m.queries = append(m.queries, q)
	return m.byType[q.IssueType], nil
}
func (m *byTypeMockTracker) Name() string { return "mock" }

// --- unused interface methods ---
func (m *byTypeMockTracker) CreateIssue(s *spec.Spec) (string, error)              { return "", nil }
func (m *byTypeMockTracker) UpdateStatus(issueID string, status spec.Status) error { return nil }
func (m *byTypeMockTracker) UpdateSize(issueID, localTier string) error            { return nil }
func (m *byTypeMockTracker) GetIssue(issueID string) (*tracker.Issue, error)       { return nil, nil }
func (m *byTypeMockTracker) UpdateFields(id string, f map[string]tracker.Value) error {
	return nil
}
func (m *byTypeMockTracker) GetFields(id string) (map[string]tracker.Value, error) { return nil, nil }
func (m *byTypeMockTracker) ListIssues(label string, limit int) ([]tracker.Issue, error) {
	return nil, nil
}
func (m *byTypeMockTracker) AddComment(issueID, body string) error              { return nil }
func (m *byTypeMockTracker) AttachFile(id, filePath, fileName string) error     { return nil }
func (m *byTypeMockTracker) SupportsHierarchy() bool                            { return false }
func (m *byTypeMockTracker) MapSize(localTier string) (string, error)           { return "", nil }
func (m *byTypeMockTracker) ReverseMapSize(trackerValue string) (string, error) { return "", nil }

// TestFetchByTypeUnion_UnionsAndDedups runs each configured type's
// effective filter, unions the results, and dedups by external id.
func TestFetchByTypeUnion_UnionsAndDedups(t *testing.T) {
	cfg := config.Config{Import: &config.ImportConfig{
		BaseFilter: &config.ImportFilter{Status: "Open"},
		ByType: map[string]*config.ImportFilter{
			"bug":  {IssueType: "bug", Priority: "High"},
			"epic": {IssueType: "epic"},
		},
	}}
	mock := &byTypeMockTracker{byType: map[string][]tracker.Issue{
		"bug":  {{ID: "1"}, {ID: "2"}},
		"epic": {{ID: "2"}, {ID: "3"}}, // ID 2 overlaps → dedup
	}}

	issues, err := fetchByTypeUnion(cfg, mock, 100)
	if err != nil {
		t.Fatalf("fetchByTypeUnion: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("union size = %d, want 3 (dedup by id); got %+v", len(issues), issues)
	}
	seen := map[string]bool{}
	for _, iss := range issues {
		if seen[iss.ID] {
			t.Errorf("duplicate id %q in union", iss.ID)
		}
		seen[iss.ID] = true
	}
	for _, want := range []string{"1", "2", "3"} {
		if !seen[want] {
			t.Errorf("union missing id %q", want)
		}
	}

	// Each type's effective filter should carry the base status plus the
	// per-type override.
	if len(mock.queries) != 2 {
		t.Fatalf("expected 2 Search calls (one per type), got %d", len(mock.queries))
	}
	for _, q := range mock.queries {
		if q.Status != "Open" {
			t.Errorf("query %+v missing base status Open", q)
		}
	}
}

// TestFetchByTypeUnion_AppliesLimitPerType ensures one large type cannot consume
// a global union budget and prevent later configured type passes from running.
func TestFetchByTypeUnion_AppliesLimitPerType(t *testing.T) {
	cfg := config.Config{Import: &config.ImportConfig{
		ByType: map[string]*config.ImportFilter{
			"bug":  {IssueType: "bug"},
			"epic": {IssueType: "epic"},
		},
	}}
	mock := &byTypeMockTracker{byType: map[string][]tracker.Issue{
		"bug":  {{ID: "1"}, {ID: "2"}, {ID: "3"}},
		"epic": {{ID: "4"}, {ID: "5"}},
	}}
	issues, err := fetchByTypeUnion(cfg, mock, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 5 {
		t.Errorf("union size = %d, want 5 (limit applies independently to each type query)", len(issues))
	}
	if len(mock.queries) != 2 || mock.queries[0].Limit != 2 || mock.queries[1].Limit != 2 {
		t.Fatalf("per-type query limits = %+v, want two independent limit=2 queries", mock.queries)
	}
}

// TestHasExplicitQueryOverride toggles a CLI flag var and confirms the
// override short-circuit is detected. Restores state on cleanup.
func TestHasExplicitQueryOverride(t *testing.T) {
	if hasExplicitQueryOverride() {
		t.Fatal("no flags set should report no override")
	}
	orig := importPriority
	importPriority = "High"
	t.Cleanup(func() { importPriority = orig })
	if !hasExplicitQueryOverride() {
		t.Error("--priority set should report an override")
	}
}

// TestHasConfiguredImportFilter distinguishes a real user/config filter
// from the synthesized base_filter defaults. The regression: a plain
// import with only `tracker` set (no import block) must report FALSE so
// it broad-fetches via ListIssues instead of Search-with-defaults.
func TestHasConfiguredImportFilter(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.Config
		want bool
	}{
		{"no import block", config.Config{}, false},
		{"empty import block", config.Config{Import: &config.ImportConfig{}}, false},
		{"empty base_filter", config.Config{Import: &config.ImportConfig{BaseFilter: &config.ImportFilter{}}}, false},
		{"configured base_filter", config.Config{Import: &config.ImportConfig{BaseFilter: &config.ImportFilter{Status: "Open"}}}, true},
		{"configured filter", config.Config{Import: &config.ImportConfig{Filter: &config.ImportFilter{Labels: []string{"ready"}}}}, true},
		{"by_type", config.Config{Import: &config.ImportConfig{ByType: map[string]*config.ImportFilter{"bug": {Priority: "High"}}}}, true},
	}
	for _, c := range cases {
		if got := hasConfiguredImportFilter(c.cfg); got != c.want {
			t.Errorf("%s: hasConfiguredImportFilter = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestIsEmptyFilter covers the per-field emptiness check.
func TestIsEmptyFilter(t *testing.T) {
	if !isEmptyFilter(nil) {
		t.Error("nil filter should be empty")
	}
	if !isEmptyFilter(&config.ImportFilter{}) {
		t.Error("zero filter should be empty")
	}
	if isEmptyFilter(&config.ImportFilter{OrderBy: "created DESC"}) {
		t.Error("filter with OrderBy should not be empty")
	}
}
