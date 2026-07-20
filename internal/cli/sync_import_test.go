package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
)

func TestImportCmd_NoTracker(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("sync", "import")
	if err == nil {
		t.Fatal("expected error when no tracker configured")
	}
	if !strings.Contains(err.Error(), "no tracker configured") {
		t.Errorf("expected tracker error, got: %v", err)
	}
}

func TestImportCmd_NoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("sync", "import")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestImportCmd_InvalidType(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("sync", "import", "--type", "initiative")
	if err == nil {
		t.Fatal("expected error for invalid import type")
	}
	if !strings.Contains(err.Error(), "must be 'feature' or 'bug'") {
		t.Errorf("expected type error, got: %v", err)
	}
}

func TestInferSpecType(t *testing.T) {
	tests := []struct {
		issueType string
		want      string
	}{
		{"Bug", "bug"},
		{"bug", "bug"},
		{"Defect", "bug"},
		{"defect", "bug"},
		{"Story", "feature"},
		{"story", "feature"},
		{"Task", "feature"},
		{"task", "feature"},
		{"Feature Request", "feature"},
		{"feature request", "feature"},
		{"Feature", "feature"},
		{"Sub-task", "feature"},
		{"subtask", "feature"},
		{"Improvement", "feature"},
		{"Enhancement", "feature"},
		{"New Feature", "feature"},
		{"Epic", "initiative"},
		{"epic", "initiative"},
		{"", "feature"},        // empty defaults to feature
		{"Unknown", "feature"}, // unknown defaults to feature
	}

	for _, tt := range tests {
		issue := tracker.Issue{IssueType: tt.issueType}
		got := inferSpecType(issue)
		if got != tt.want {
			t.Errorf("inferSpecType(%q) = %q, want %q", tt.issueType, got, tt.want)
		}
	}
}

func TestResolveTypeFilter(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	tests := []struct {
		name     string
		args     []string
		flagType string
		cfgType  string
		want     string
	}{
		{"no args, no flag, no config", nil, "", "", ""},
		{"positional bugs", []string{"bugs"}, "", "", "bug"},
		{"positional bug", []string{"bug"}, "", "", "bug"},
		{"positional features", []string{"features"}, "", "", "feature"},
		{"positional feature", []string{"feature"}, "", "", "feature"},
		{"flag overrides config", nil, "bug", "feature", "bug"},
		{"config default_type", nil, "", "bug", "bug"},
		{"positional overrides flag", []string{"bugs"}, "feature", "", "bug"},
		{"positional overrides config", []string{"features"}, "", "bug", "feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncImportType = tt.flagType
			cfg := config.Config{}
			if tt.cfgType != "" {
				cfg.Import = &config.ImportConfig{DefaultType: tt.cfgType}
			}
			got := resolveTypeFilter(tt.args, cfg)
			if got != tt.want {
				t.Errorf("resolveTypeFilter(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestDominantType(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]int
		want   string
	}{
		{"empty", map[string]int{}, "all"},
		{"single bug", map[string]int{"bug": 5}, "bug"},
		{"single feature", map[string]int{"feature": 3}, "feature"},
		{"bug dominant", map[string]int{"bug": 10, "feature": 3}, "bug"},
		{"feature dominant", map[string]int{"bug": 2, "feature": 8}, "feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dominantType(tt.counts)
			if got != tt.want {
				t.Errorf("dominantType(%v) = %q, want %q", tt.counts, got, tt.want)
			}
		})
	}
}

func TestImportCmd_PositionalBugs(t *testing.T) {
	_ = newTestEnv(t)

	// Should accept positional arg "bugs" without error (will fail at tracker step)
	_, err := runCmd("sync", "import", "bugs")
	if err == nil {
		t.Fatal("expected error (no tracker), but should not fail at arg parsing")
	}
	if !strings.Contains(err.Error(), "no tracker configured") {
		t.Errorf("expected tracker error (arg parsing succeeded), got: %v", err)
	}
}

func TestImportCmd_PositionalFeatures(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("sync", "import", "features")
	if err == nil {
		t.Fatal("expected error (no tracker)")
	}
	if !strings.Contains(err.Error(), "no tracker configured") {
		t.Errorf("expected tracker error (arg parsing succeeded), got: %v", err)
	}
}

func TestImportCmd_TooManyArgs(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("sync", "import", "bugs", "extra")
	if err == nil {
		t.Fatal("expected error for too many args")
	}
	// cobra.MaximumNArgs(1) generates "accepts at most 1 arg(s)"
	if !strings.Contains(err.Error(), "at most 1") {
		t.Errorf("expected max args error, got: %v", err)
	}
}

func TestIssueToSlug(t *testing.T) {
	tests := []struct {
		title string
		id    string
		want  string
	}{
		{"Add user authentication", "PROJ-42", "proj-42-add-user-authentication"},
		{"[feature] CSV Export", "PROJ-123", "proj-123-csv-export"},
		{"[bug] Login fails on Safari", "BUG-7", "bug-7-login-fails-on-safari"},
		{"Fix API rate limiting & retry logic", "PROJ-99", "proj-99-fix-api-rate-limiting-retry-logic"},
		{"", "PROJ-55", "proj-55"},
		{"A very long title that goes on and on and should be truncated", "PROJ-1", "proj-1-a-very-long-title-that-goes-on-and-on-and-s"},
		// GitHub-style numeric IDs
		{"Add dark mode", "42", "42-add-dark-mode"},
	}

	for _, tt := range tests {
		issue := tracker.Issue{ID: tt.id, Title: tt.title}
		got := issueToSlug(issue)
		if got != tt.want {
			t.Errorf("issueToSlug(id=%q, title=%q) = %q, want %q", tt.id, tt.title, got, tt.want)
		}
	}
}

func TestGenerateImportedSpec_Feature(t *testing.T) {
	issue := tracker.Issue{
		ID:        "42",
		Title:     "Add dark mode",
		URL:       "https://github.com/test/repo/issues/42",
		Status:    "Open",
		Priority:  "High",
		CreatedAt: "2025-03-15T10:00:00Z",
	}

	content := generateImportedSpec(issue, "feature", "github", "add-dark-mode")

	checks := []string{
		`title: "Add dark mode"`,
		"slug: add-dark-mode",
		"type: feature",
		"status: planning",
		"tracker_id: 42",
		"created: 2025-03-15",
		"priority: high",
		"tags: [imported]",
		"# Github",
		"github_id: 42",
		"github_status: Open",
		"github_priority: High",
		"github_url: https://github.com/test/repo/issues/42",
		"## Goal",
		"## Design",
		"## Changes",
		"## Acceptance Criteria",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected %q in generated spec, got:\n%s", check, content)
		}
	}
}

func TestGenerateImportedSpec_Bug(t *testing.T) {
	issue := tracker.Issue{
		ID:        "7",
		Title:     "[Bug] Login broken",
		URL:       "https://github.com/test/repo/issues/7",
		Status:    "Open",
		Severity:  "Critical",
		Assignee:  "alice",
		IssueType: "Bug",
	}

	content := generateImportedSpec(issue, "bug", "jira", "login-broken")

	checks := []string{
		`title: "Login broken"`,
		"slug: login-broken",
		"type: bug",
		"severity: critical",
		"# Jira",
		"jira_id: 7",
		"jira_status: Open",
		"jira_severity: Critical",
		"jira_assignee: alice",
		"jira_type: Bug",
		"## Problem",
		"## Root Cause",
		"## Fix",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected %q in generated spec, got:\n%s", check, content)
		}
	}

	// Should not have feature sections
	if strings.Contains(content, "## Goal") {
		t.Error("bug spec should not have Goal section")
	}
}

func TestGenerateImportedSpec_TrackerActivityTimestamps(t *testing.T) {
	issue := tracker.Issue{
		ID:        "PROJ-42",
		Title:     "Preserve tracker activity",
		Status:    "Open",
		CreatedAt: "2026-07-19T23:30:40.123-0600",
		UpdatedAt: "2026-07-20T10:15:30.123456-0600",
	}

	content := generateImportedSpec(issue, "feature", "jira", "proj-42-activity")
	for _, want := range []string{
		"created: 2026-07-19",
		"tracker_updated_at: 2026-07-20T16:15:30.123456Z",
		"jira_updated_at: 2026-07-20T10:15:30.123456-0600",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("generated spec missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateImportedSpec_InvalidOrMissingTimestampsOmitFields(t *testing.T) {
	for _, tc := range []struct {
		name  string
		issue tracker.Issue
	}{
		{name: "missing"},
		{name: "malformed", issue: tracker.Issue{CreatedAt: "not-a-date", UpdatedAt: "also-not-a-date"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.issue.ID = "PROJ-42"
			tc.issue.Title = "Unknown activity"
			content := generateImportedSpec(tc.issue, "feature", "jira", "proj-42-unknown")
			for _, unwanted := range []string{"created:", "tracker_updated_at:", "jira_updated_at:"} {
				if strings.Contains(content, unwanted) {
					t.Errorf("generated spec contains fallback %q:\n%s", unwanted, content)
				}
			}
		})
	}
}

func TestSpecFieldsFromIssue_ProviderTimestampEvidence(t *testing.T) {
	const native = "2026-07-20T11:22:33.123456789Z"
	for _, provider := range []string{"jira", "github", "gitlab", "linear"} {
		t.Run(provider, func(t *testing.T) {
			fields := specFieldsFromIssue(tracker.Issue{UpdatedAt: native}, provider)
			if got := fields["tracker_updated_at"]; got != native {
				t.Errorf("tracker_updated_at = %q, want %q", got, native)
			}
			if got := fields[provider+"_updated_at"]; got != native {
				t.Errorf("%s_updated_at = %q, want exact native value", provider, got)
			}
		})
	}
}

type activityMockTracker struct {
	byTypeMockTracker
	issue tracker.Issue
}

func (m *activityMockTracker) Name() string { return "jira" }
func (m *activityMockTracker) GetIssue(string) (*tracker.Issue, error) {
	issue := m.issue
	return &issue, nil
}

func TestRefreshImportedSpecs_UpdatesPreservesAndIsIdempotent(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/activity/spec.md", `---
title: Activity
slug: activity
type: feature
status: planning
tracker_id: PROJ-42
created: 2026-07-19
tracker_updated_at: 2026-07-20T16:15:30.123Z
jira_id: PROJ-42
jira_updated_at: 2026-07-20T10:15:30.123-0600
---
# Activity
`)
	specPath := filepath.Join(env.heroDir, "planning", "features", "activity", "spec.md")
	mock := &activityMockTracker{issue: tracker.Issue{
		ID:        "PROJ-42",
		Status:    "Open",
		CreatedAt: "2026-07-19T23:30:40.123-0600",
		UpdatedAt: "2026-07-21T09:08:07.123456-0600",
	}}

	first := refreshImportedSpecs(config.Config{}, env.heroDir, mock, nil)
	if first.updated != 1 {
		t.Fatalf("first refresh updated = %d, want 1", first.updated)
	}
	afterFirst, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"tracker_updated_at: 2026-07-21T15:08:07.123456Z",
		"jira_updated_at: 2026-07-21T09:08:07.123456-0600",
	} {
		if !strings.Contains(string(afterFirst), want) {
			t.Errorf("refreshed spec missing %q:\n%s", want, afterFirst)
		}
	}

	second := refreshImportedSpecs(config.Config{}, env.heroDir, mock, nil)
	afterSecond, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if second.updated != 0 {
		t.Errorf("second refresh updated = %d, want 0", second.updated)
	}
	if string(afterSecond) != string(afterFirst) {
		t.Error("second refresh changed an already current spec")
	}

	mock.issue.CreatedAt = "malformed"
	mock.issue.UpdatedAt = ""
	third := refreshImportedSpecs(config.Config{}, env.heroDir, mock, nil)
	afterThird, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if third.updated != 0 {
		t.Errorf("malformed refresh updated = %d, want 0", third.updated)
	}
	if string(afterThird) != string(afterSecond) {
		t.Error("malformed/missing refresh cleared or replaced valid timestamps")
	}
}

func TestCurrentSpecFieldValue_MtimeIsNotTrackerCreatedEvidence(t *testing.T) {
	content := "---\ntitle: Missing created\nslug: missing-created\ntype: feature\nstatus: planning\ntracker_id: PROJ-42\n---\n"
	s, err := spec.Parse(content, filepath.Join(t.TempDir(), "spec.md"), time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if got := currentSpecFieldValue(s, "created"); got != "" {
		t.Errorf("current created = %q, want empty because value came from mtime", got)
	}
}

func TestRefreshImportedSpecs_StampsCreatedWhenOnlyMtimeMatches(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/missing-created/spec.md", `---
title: Missing created
slug: missing-created
type: feature
status: planning
tracker_id: PROJ-42
jira_id: PROJ-42
---
# Missing created
`)
	specPath := filepath.Join(env.heroDir, "planning", "features", "missing-created", "spec.md")
	mtime := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(specPath, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	mock := &activityMockTracker{issue: tracker.Issue{
		ID:        "PROJ-42",
		Status:    "Open",
		CreatedAt: "2026-07-19T23:30:40.123-0600",
	}}

	stats := refreshImportedSpecs(config.Config{}, env.heroDir, mock, nil)
	if stats.updated != 1 {
		t.Fatalf("refresh updated = %d, want 1", stats.updated)
	}
	content, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "created: 2026-07-19") {
		t.Errorf("refresh did not persist tracker creation date when mtime matched:\n%s", content)
	}
}

func TestTrackerTimestampFieldsAreNonPushable(t *testing.T) {
	for _, field := range []string{
		"tracker_updated_at",
		"jira_updated_at",
		"github_updated_at",
		"gitlab_updated_at",
		"linear_updated_at",
	} {
		got, ok := classifyField(field)
		if !ok || got.Class != classOrgState {
			t.Errorf("classifyField(%q) = %+v, %v; want org-state", field, got, ok)
		}
	}
}

func TestFindLinkedTrackerIDs(t *testing.T) {
	env := newTestEnv(t)

	// Add specs with and without tracker IDs
	env.addSpec("planning/features/linked/spec.md", `---
title: Linked Feature
type: feature
status: planning
tracker_id: 42
---
## Goal
Already linked.
`)
	env.addSpec("planning/features/unlinked/spec.md", `---
title: Unlinked Feature
type: feature
status: planning
---
## Goal
Not linked.
`)

	ids := findLinkedTrackerIDs(env.heroDir)
	if !ids["42"] {
		t.Error("expected tracker ID '42' to be found")
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 linked ID, got %d", len(ids))
	}
}

// resetImportFlags zeroes out all package-level import flags so tests are isolated.
func resetImportFlags() {
	importPreset = ""
	importJQL = ""
	importFilterID = ""
	importAssignee = ""
	importIssueType = ""
	importLabel = ""
	importStatus = ""
	importPriority = ""
	importOrderBy = ""
	importLimit = 0
}

func TestResolveImportQuery_NoConfig(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	cfg := config.Config{}
	query, err := resolveImportQuery(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With no config, base filter defaults should apply: Bug + unassigned + New
	if query.IssueType != "Bug" {
		t.Errorf("expected IssueType=Bug from base default, got %q", query.IssueType)
	}
	if query.Assignee != "unassigned" {
		t.Errorf("expected Assignee=unassigned from base default, got %q", query.Assignee)
	}
	if query.Status != "New" {
		t.Errorf("expected Status=New from base default, got %q", query.Status)
	}
}

func TestResolveImportQuery_DefaultFilter(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	cfg := config.Config{
		Import: &config.ImportConfig{
			Filter: &config.ImportFilter{
				Assignee: "me",
				Status:   "Open",
			},
		},
	}

	query, err := resolveImportQuery(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Filter should override base defaults
	if query.Assignee != "me" {
		t.Errorf("expected Assignee=me, got %q", query.Assignee)
	}
	if query.Status != "Open" {
		t.Errorf("expected Status=Open, got %q", query.Status)
	}
	// IssueType not in filter, so base default should apply
	if query.IssueType != "Bug" {
		t.Errorf("expected IssueType=Bug from base default, got %q", query.IssueType)
	}
}

func TestResolveImportQuery_PresetFound(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	importPreset = "bugs"

	cfg := config.Config{
		Import: &config.ImportConfig{
			Filter: &config.ImportFilter{
				Assignee: "default-user",
				Status:   "Open",
			},
			Presets: map[string]*config.ImportFilter{
				"bugs": {
					IssueType: "Bug",
					Priority:  "High",
					Status:    "New",
				},
				"stories": {
					IssueType: "Story",
				},
			},
		},
	}

	query, err := resolveImportQuery(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Preset should replace the filter layer but base defaults still apply underneath
	if query.IssueType != "Bug" {
		t.Errorf("expected IssueType=Bug from preset, got %q", query.IssueType)
	}
	if query.Priority != "High" {
		t.Errorf("expected Priority=High from preset, got %q", query.Priority)
	}
	if query.Status != "New" {
		t.Errorf("expected Status=New from preset, got %q", query.Status)
	}
	// Assignee not in preset, so base default (unassigned) should apply
	if query.Assignee != "unassigned" {
		t.Errorf("expected Assignee=unassigned from base default, got %q", query.Assignee)
	}
}

func TestResolveImportQuery_PresetWithCLIOverride(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	importPreset = "bugs"
	importAssignee = "alice" // CLI flag should override preset

	cfg := config.Config{
		Import: &config.ImportConfig{
			Presets: map[string]*config.ImportFilter{
				"bugs": {
					IssueType: "Bug",
					Assignee:  "unassigned",
				},
			},
		},
	}

	query, err := resolveImportQuery(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// IssueType from preset
	if query.IssueType != "Bug" {
		t.Errorf("expected IssueType=Bug from preset, got %q", query.IssueType)
	}
	// Assignee overridden by CLI flag
	if query.Assignee != "alice" {
		t.Errorf("expected Assignee=alice from CLI override, got %q", query.Assignee)
	}
}

func TestResolveImportQuery_PresetNotFound(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	importPreset = "nonexistent"

	cfg := config.Config{
		Import: &config.ImportConfig{
			Presets: map[string]*config.ImportFilter{
				"bugs":    {IssueType: "Bug"},
				"stories": {IssueType: "Story"},
			},
		},
	}

	_, err := resolveImportQuery(cfg)
	if err == nil {
		t.Fatal("expected error for unknown preset")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the unknown preset name, got: %v", err)
	}
	// Should list available presets
	if !strings.Contains(err.Error(), "bugs") || !strings.Contains(err.Error(), "stories") {
		t.Errorf("error should list available presets, got: %v", err)
	}
}

func TestResolveImportQuery_PresetNoPresetsConfigured(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	importPreset = "mypreset"

	cfg := config.Config{
		Import: &config.ImportConfig{},
	}

	_, err := resolveImportQuery(cfg)
	if err == nil {
		t.Fatal("expected error when no presets configured")
	}
	if !strings.Contains(err.Error(), "no import.presets configured") {
		t.Errorf("expected helpful error about missing presets config, got: %v", err)
	}
}

func TestResolveImportQuery_PresetNoImportConfig(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	importPreset = "mypreset"

	cfg := config.Config{}

	_, err := resolveImportQuery(cfg)
	if err == nil {
		t.Fatal("expected error when no import config at all")
	}
	if !strings.Contains(err.Error(), "no import.presets configured") {
		t.Errorf("expected helpful error about missing presets config, got: %v", err)
	}
}

func TestResolveImportQuery_PresetWithJQL(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	importPreset = "sprint"

	cfg := config.Config{
		Import: &config.ImportConfig{
			Presets: map[string]*config.ImportFilter{
				"sprint": {
					JQL: "project = HERO AND sprint in openSprints()",
				},
			},
		},
	}

	query, err := resolveImportQuery(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.RawQuery != "project = HERO AND sprint in openSprints()" {
		t.Errorf("expected JQL from preset, got %q", query.RawQuery)
	}
}

func TestResolveImportQuery_CustomBaseFilter(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	cfg := config.Config{
		Import: &config.ImportConfig{
			BaseFilter: &config.ImportFilter{
				IssueType: "Story",
				Status:    "Open",
				Assignee:  "me",
			},
		},
	}

	query, err := resolveImportQuery(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.IssueType != "Story" {
		t.Errorf("expected IssueType=Story from custom base, got %q", query.IssueType)
	}
	if query.Status != "Open" {
		t.Errorf("expected Status=Open from custom base, got %q", query.Status)
	}
	if query.Assignee != "me" {
		t.Errorf("expected Assignee=me from custom base, got %q", query.Assignee)
	}
}

func TestResolveImportQuery_BaseFilterPlusFilter(t *testing.T) {
	resetImportFlags()
	defer resetImportFlags()

	cfg := config.Config{
		Import: &config.ImportConfig{
			BaseFilter: &config.ImportFilter{
				IssueType: "Bug",
				Assignee:  "unassigned",
				Status:    "New",
			},
			Filter: &config.ImportFilter{
				Priority: "Critical",
				Labels:   []string{"production"},
			},
		},
	}

	query, err := resolveImportQuery(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Base filter values
	if query.IssueType != "Bug" {
		t.Errorf("expected IssueType=Bug from base, got %q", query.IssueType)
	}
	if query.Assignee != "unassigned" {
		t.Errorf("expected Assignee=unassigned from base, got %q", query.Assignee)
	}
	if query.Status != "New" {
		t.Errorf("expected Status=New from base, got %q", query.Status)
	}
	// Additional filter values layered on top
	if query.Priority != "Critical" {
		t.Errorf("expected Priority=Critical from filter, got %q", query.Priority)
	}
	if len(query.Labels) != 1 || query.Labels[0] != "production" {
		t.Errorf("expected Labels=[production] from filter, got %v", query.Labels)
	}
}
