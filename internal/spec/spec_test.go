package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseFeatureSpec(t *testing.T) {
	content := `---
title: Add CSV Export
type: feature
status: planning
tags: [export, users]
created: 2025-01-15
---
# Add CSV Export

## Context
Users need to export data.

## Changes
- Update ` + "`src/api/users.ts`" + ` to add export endpoint
- Add ` + "`src/export/csv.ts`" + ` for CSV generation

## Risks
Performance with large datasets.
`
	s, err := Parse(content, "/project/.hero/planning/features/add-csv-export/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if s.Title != "Add CSV Export" {
		t.Errorf("Title = %q, want %q", s.Title, "Add CSV Export")
	}
	if s.Type != TypeFeature {
		t.Errorf("Type = %q, want %q", s.Type, TypeFeature)
	}
	if s.Status != StatusPlanning {
		t.Errorf("Status = %q, want %q", s.Status, StatusPlanning)
	}
	if s.Slug != "add-csv-export" {
		t.Errorf("Slug = %q, want %q", s.Slug, "add-csv-export")
	}
	if len(s.Tags) != 2 || s.Tags[0] != "export" || s.Tags[1] != "users" {
		t.Errorf("Tags = %v, want [export, users]", s.Tags)
	}
	if s.CreatedAt.Year() != 2025 || s.CreatedAt.Month() != 1 || s.CreatedAt.Day() != 15 {
		t.Errorf("CreatedAt = %v, want 2025-01-15", s.CreatedAt)
	}
	if len(s.FilesTouched) != 2 {
		t.Errorf("FilesTouched = %v, want 2 files", s.FilesTouched)
	}
	if _, ok := s.Sections["context"]; !ok {
		t.Error("Sections missing 'context'")
	}
	if _, ok := s.Sections["risks"]; !ok {
		t.Error("Sections missing 'risks'")
	}
}

func TestParseBugSpec(t *testing.T) {
	content := `---
title: Login Timeout Race
type: bug
status: delivering
claimed_by: alice
---
# Login Timeout Race

## Investigation
Race condition in session handling.

## Changes
- Fix ` + "`src/auth/session.go`" + `

## Risks
Regression in concurrent sessions.
`
	s, err := Parse(content, "/project/.hero/planning/bugs/login-timeout/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if s.Type != TypeBug {
		t.Errorf("Type = %q, want %q", s.Type, TypeBug)
	}
	if s.Status != StatusDelivering {
		t.Errorf("Status = %q, want %q", s.Status, StatusDelivering)
	}
	if s.ClaimedBy != "alice" {
		t.Errorf("ClaimedBy = %q, want %q", s.ClaimedBy, "alice")
	}
	if s.Slug != "login-timeout" {
		t.Errorf("Slug = %q, want %q", s.Slug, "login-timeout")
	}
}

func TestParseConventionSpec(t *testing.T) {
	content := `---
title: API Response Format
type: convention
status: active
scope: [src/api/**/*.ts, src/api/**/*.js]
tags: api
---
# API Response Format

## Rule
All API error responses use the ApiError class.
`
	s, err := Parse(content, "/project/.hero/conventions/api-response-format/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if s.Type != TypeConvention {
		t.Errorf("Type = %q, want %q", s.Type, TypeConvention)
	}
	if s.Status != StatusActive {
		t.Errorf("Status = %q, want %q", s.Status, StatusActive)
	}
	if len(s.Scope) != 2 {
		t.Errorf("Scope = %v, want 2 items", s.Scope)
	}
	if s.Scope[0] != "src/api/**/*.ts" {
		t.Errorf("Scope[0] = %q, want %q", s.Scope[0], "src/api/**/*.ts")
	}
}

func TestParseDecisionSpec(t *testing.T) {
	content := `---
title: Use PostgreSQL FTS
type: decision
status: accepted
tags: [search, database]
---
# Use PostgreSQL FTS

## Decision
Use PostgreSQL full-text search rather than Elasticsearch.
`
	s, err := Parse(content, "/project/.hero/decisions/use-postgres-fts/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if s.Type != TypeDecision {
		t.Errorf("Type = %q, want %q", s.Type, TypeDecision)
	}
	if s.Status != StatusAccepted {
		t.Errorf("Status = %q, want %q", s.Status, StatusAccepted)
	}
}

func TestParseInitiativeSpec(t *testing.T) {
	content := `---
title: Q3 Auth Overhaul
type: initiative
status: planning
tags: auth, q3
---
# Q3 Auth Overhaul

## Goal
Overhaul the authentication system for Q3.
`
	s, err := Parse(content, "/project/.hero/planning/initiatives/q3-auth-overhaul/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if s.Type != TypeInitiative {
		t.Errorf("Type = %q, want %q", s.Type, TypeInitiative)
	}
}

func TestParseRelations(t *testing.T) {
	content := `---
title: Add MFA
type: feature
status: planning
depends-on: add-session-management
parent: q3-auth-overhaul
relates-to: use-postgres-fts
---
# Add MFA
`
	s, err := Parse(content, "/project/.hero/planning/features/add-mfa/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(s.Relations) != 3 {
		t.Fatalf("Relations = %d, want 3", len(s.Relations))
	}

	relMap := make(map[string]string)
	for _, r := range s.Relations {
		relMap[r.Kind] = r.Target
	}

	if relMap["depends-on"] != "add-session-management" {
		t.Errorf("depends-on = %q, want %q", relMap["depends-on"], "add-session-management")
	}
	if relMap["parent"] != "q3-auth-overhaul" {
		t.Errorf("parent = %q, want %q", relMap["parent"], "q3-auth-overhaul")
	}
	if relMap["relates-to"] != "use-postgres-fts" {
		t.Errorf("relates-to = %q, want %q", relMap["relates-to"], "use-postgres-fts")
	}
}

func TestParseRelations_ShorthandAliasesAndBlockList(t *testing.T) {
	// The shorthands first-use sessions reach for must form relations,
	// not silently drop: `initiative:` (→ parent), `depends_on:`
	// (underscore → depends-on), and a block-style `child:` list.
	content := `---
title: Config Loader
type: feature
status: planning
initiative: i1-config-plane
depends_on: [f2-config-store, f3-watcher]
child:
  - sub-a
  - sub-b
---
# Config Loader
`
	s, err := Parse(content, "/project/.hero/planning/features/config-loader/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var parents, dependsOn, children []string
	for _, r := range s.Relations {
		switch r.Kind {
		case "parent":
			parents = append(parents, r.Target)
		case "depends-on":
			dependsOn = append(dependsOn, r.Target)
		case "child":
			children = append(children, r.Target)
		}
	}

	if len(parents) != 1 || parents[0] != "i1-config-plane" {
		t.Errorf("initiative: should map to a parent relation, got parents=%v", parents)
	}
	if len(dependsOn) != 2 || dependsOn[0] != "f2-config-store" || dependsOn[1] != "f3-watcher" {
		t.Errorf("depends_on: should map to depends-on relations, got %v", dependsOn)
	}
	if len(children) != 2 || children[0] != "sub-a" || children[1] != "sub-b" {
		t.Errorf("block-style child: list should parse, got %v", children)
	}
}

func TestTypeFromPath(t *testing.T) {
	tests := []struct {
		path string
		want Type
	}{
		{"/project/.hero/planning/features/foo/spec.md", TypeFeature},
		{"/project/.hero/planning/bugs/bar/spec.md", TypeBug},
		{"/project/.hero/conventions/baz/spec.md", TypeConvention},
		{"/project/.hero/decisions/qux/spec.md", TypeDecision},
		{"/project/.hero/planning/initiatives/quux/spec.md", TypeInitiative},
		{"/project/.hero/specs/foo/spec.md", TypeFeature},
		{"/project/.hero/knowledge/explainers/agent-outposts/spec.md", TypeExplainer},
	}

	for _, tt := range tests {
		got := typeFromPath(tt.path)
		if got != tt.want {
			t.Errorf("typeFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestStatusFromPath(t *testing.T) {
	tests := []struct {
		path string
		want Status
	}{
		{"/project/.hero/planning/features/foo/spec.md", StatusPlanning},
		{"/project/.hero/specs/foo/spec.md", StatusCompleted},
		{"/project/.hero/conventions/baz/spec.md", StatusActive},
		{"/project/.hero/decisions/qux/spec.md", StatusAccepted},
	}

	for _, tt := range tests {
		got := statusFromPath(tt.path)
		if got != tt.want {
			t.Errorf("statusFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestSlugFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/project/.hero/planning/features/add-csv-export/spec.md", "add-csv-export"},
		{"/project/.hero/specs/dark-mode/spec.md", "dark-mode"},
		{"/project/.hero/conventions/api-format/spec.md", "api-format"},
		// Three-file requirements.md → directory name (not "requirements").
		{"/project/.hero/planning/features/dark-mode/requirements.md", "dark-mode"},
		// Flat <slug>.md → filename stem, not the parent dir (initiative
		// children stored as siblings). Regression: flat-named-spec-discovery.
		{"/project/.hero/planning/initiatives/make-it-fast/f-15-buffer-pool.md", "f-15-buffer-pool"},
		{"/project/.hero/planning/loose-feature.md", "loose-feature"},
	}

	for _, tt := range tests {
		got := slugFromPath(tt.path)
		if got != tt.want {
			t.Errorf("slugFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestIsWorkSpec(t *testing.T) {
	feature := &Spec{Type: TypeFeature}
	bug := &Spec{Type: TypeBug}
	convention := &Spec{Type: TypeConvention}
	decision := &Spec{Type: TypeDecision}

	if !feature.IsWorkSpec() {
		t.Error("feature should be work spec")
	}
	if !bug.IsWorkSpec() {
		t.Error("bug should be work spec")
	}
	if convention.IsWorkSpec() {
		t.Error("convention should not be work spec")
	}
	if decision.IsWorkSpec() {
		t.Error("decision should not be work spec")
	}
}

func TestIsInFlight(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusPlanning, true},
		{StatusInReview, true},
		{StatusDelivering, true},
		{StatusCompleted, false},
		{StatusActive, false},
		{StatusAccepted, false},
		{StatusSuperseded, false},
	}

	for _, tt := range tests {
		s := &Spec{Status: tt.status}
		if got := s.IsInFlight(); got != tt.want {
			t.Errorf("IsInFlight(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	content := `# My Feature

## Changes
- Update ` + "`src/main.go`" + `
`
	s, err := Parse(content, "/project/.hero/planning/features/my-feature/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should infer from path
	if s.Type != TypeFeature {
		t.Errorf("Type = %q, want %q", s.Type, TypeFeature)
	}
	if s.Status != StatusPlanning {
		t.Errorf("Status = %q, want %q", s.Status, StatusPlanning)
	}
	if s.Title != "My Feature" {
		t.Errorf("Title = %q, want %q", s.Title, "My Feature")
	}
}

func TestParseFrontmatterOverridesPath(t *testing.T) {
	content := `---
type: bug
status: delivering
---
# A Bug Report
`
	// Path says feature+planning, but frontmatter overrides
	s, err := Parse(content, "/project/.hero/planning/features/oops/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if s.Type != TypeBug {
		t.Errorf("Type = %q, want %q (frontmatter should override path)", s.Type, TypeBug)
	}
	if s.Status != StatusDelivering {
		t.Errorf("Status = %q, want %q (frontmatter should override path)", s.Status, StatusDelivering)
	}
}

func TestExtractFilePaths(t *testing.T) {
	changes := `
- Update ` + "`src/api/users.ts`" + ` to add export
- Add ` + "`src/export/csv.ts`" + ` for CSV generation
- Modify ` + "`src/db/queries/users.sql`" + `
- Some text without file paths
- A bare word with slash but no extension: src/api
`
	paths := extractFilePaths(changes)

	expected := []string{"src/api/users.ts", "src/export/csv.ts", "src/db/queries/users.sql"}
	if len(paths) != len(expected) {
		t.Fatalf("extractFilePaths got %d paths, want %d: %v", len(paths), len(expected), paths)
	}
	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("paths[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestParseList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a, b, c", []string{"a", "b", "c"}},
		{"[a, b, c]", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"", nil},
		{`["quoted", "values"]`, []string{"quoted", "values"}},
	}

	for _, tt := range tests {
		got := parseList(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseList(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseList(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestDiscover(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")

	dirs := []string{
		filepath.Join(heroDir, "planning", "features", "feat-one"),
		filepath.Join(heroDir, "planning", "bugs", "bug-one"),
		filepath.Join(heroDir, "specs", "feat-done"),
		filepath.Join(heroDir, "conventions", "conv-one"),
		filepath.Join(heroDir, "decisions", "dec-one"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	}

	// Write spec.md files
	specs := map[string]string{
		filepath.Join(dirs[0], "spec.md"): "# Feature One\n\n## Changes\n- Update `src/main.go`\n",
		filepath.Join(dirs[1], "spec.md"): "---\ntype: bug\n---\n# Bug One\n",
		filepath.Join(dirs[2], "spec.md"): "# Feature Done\n",
		filepath.Join(dirs[3], "spec.md"): "---\ntype: convention\nstatus: active\nscope: [*.go]\n---\n# Convention One\n",
		filepath.Join(dirs[4], "spec.md"): "---\ntype: decision\nstatus: accepted\n---\n# Decision One\n",
	}

	for path, content := range specs {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	// Also write a non-spec file (should be ignored)
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	discovered, err := Discover(heroDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(discovered) != 5 {
		t.Errorf("Discover found %d specs, want 5", len(discovered))
		for _, s := range discovered {
			t.Logf("  %s (%s/%s)", s.Slug, s.Type, s.Status)
		}
	}

	// Verify types are correct
	typeCount := make(map[Type]int)
	for _, s := range discovered {
		typeCount[s.Type]++
	}

	if typeCount[TypeFeature] != 2 {
		t.Errorf("Features = %d, want 2", typeCount[TypeFeature])
	}
	if typeCount[TypeBug] != 1 {
		t.Errorf("Bugs = %d, want 1", typeCount[TypeBug])
	}
	if typeCount[TypeConvention] != 1 {
		t.Errorf("Conventions = %d, want 1", typeCount[TypeConvention])
	}
	if typeCount[TypeDecision] != 1 {
		t.Errorf("Decisions = %d, want 1", typeCount[TypeDecision])
	}
}

// TestDiscoverFlatNamedSpec covers initiative children stored as flat
// `<slug>.md` files sibling to the initiative's spec.md — they must be
// discovered by their declared work type. Regression for
// flat-named-spec-discovery.
func TestDiscoverFlatNamedSpec(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), ".hero")
	initDir := filepath.Join(heroDir, "planning", "initiatives", "make-it-fast")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	files := map[string]string{
		// The initiative itself (canonical spec.md).
		filepath.Join(initDir, "spec.md"): "---\ntype: initiative\nslug: make-it-fast\nstatus: delivering\n---\n# Make It Fast\n",
		// Flat children sibling to the initiative — must be discovered.
		filepath.Join(initDir, "f-01-buffer-pool.md"): "---\ntitle: Buffer Pool\nslug: f-01-buffer-pool\ntype: feature\nstatus: delivering\nparent: make-it-fast\n---\n# Buffer Pool\n",
		filepath.Join(initDir, "f-02-wal-commit.md"):  "---\nslug: f-02-wal-commit\ntype: bug\nstatus: delivering\nparent: make-it-fast\n---\n# WAL Commit\n",
		// Flat feature WITHOUT an explicit slug (relies on filename) — still
		// discovered because type is explicit; slug derives from filename.
		filepath.Join(heroDir, "planning", "loose-feature.md"): "---\ntype: feature\nstatus: planning\n---\n# Loose Feature\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	discovered, err := Discover(heroDir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	bySlug := make(map[string]*Spec)
	for _, s := range discovered {
		bySlug[s.Slug] = s
	}

	for _, want := range []string{"make-it-fast", "f-01-buffer-pool", "f-02-wal-commit", "loose-feature"} {
		if _, ok := bySlug[want]; !ok {
			t.Errorf("Discover missed flat spec %q; got %d specs: %v", want, len(discovered), slugs(discovered))
		}
	}
	if got := bySlug["f-01-buffer-pool"]; got != nil && got.Type != TypeFeature {
		t.Errorf("f-01 type = %q, want feature", got.Type)
	}
	if got := bySlug["f-02-wal-commit"]; got != nil && got.Type != TypeBug {
		t.Errorf("f-02 type = %q, want bug", got.Type)
	}
}

// TestDiscoverIgnoresNonSpecFlatFiles guards against the explicit-type gate
// regressing into a vacuum cleaner: untyped artifacts and knowledge entries
// stored as flat `.md` files must stay out of work-spec discovery.
func TestDiscoverIgnoresNonSpecFlatFiles(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), ".hero")
	initDir := filepath.Join(heroDir, "planning", "initiatives", "make-it-fast")
	auditDir := filepath.Join(initDir, "audits")
	nextDir := filepath.Join(heroDir, "next")
	knowDir := filepath.Join(heroDir, "knowledge", "decisions")
	for _, d := range []string{auditDir, nextDir, knowDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}

	files := map[string]string{
		// One real spec so the directory isn't empty.
		filepath.Join(initDir, "spec.md"): "---\ntype: initiative\nslug: make-it-fast\nstatus: delivering\n---\n# Make It Fast\n",
		// Untyped artifacts under an initiative dir — would default to
		// initiative/feature via typeFromPath if the explicit gate broke.
		filepath.Join(auditDir, "f-01-followup-2026-06-24.md"): "# Audit\n\nNo frontmatter, just prose.\n",
		filepath.Join(nextDir, "alice.md"):                     "---\nuser: alice\n---\n# Handoff\n",
		// Knowledge entries with explicit non-work types — excluded.
		filepath.Join(knowDir, "some-decision.md"):             "---\ntype: decision\nstatus: accepted\n---\n# A Decision\n",
		filepath.Join(heroDir, "knowledge", "a-convention.md"): "---\ntype: convention\nslug: a-convention\nstatus: active\n---\n# A Convention\n",
		filepath.Join(heroDir, "mission.md"):                   "---\ntype: mission\n---\n# Mission\n",
		filepath.Join(initDir, "f-01-followup.retro.md"):       "---\ntype: retro\n---\n# Retro\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	discovered, err := Discover(heroDir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if len(discovered) != 1 || discovered[0].Slug != "make-it-fast" {
		t.Errorf("Discover should find only the initiative; got %d: %v", len(discovered), slugs(discovered))
	}
}

func TestLooksLikeFilePath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"src/main.go", true},
		{"src/api/users.ts", true},
		{"main.go", false},      // no slash
		{"src/api", false},      // no extension
		{"src/api/", false},     // trailing slash, no extension
		{"src/main.xyz", false}, // unknown extension
		{"src/db/query.sql", true},
		{"path/to/file.py", true},
	}

	for _, tt := range tests {
		got := looksLikeFilePath(tt.input)
		if got != tt.want {
			t.Errorf("looksLikeFilePath(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestClassifyCriterion(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantKind     CriterionKind
		wantTrigger  string
		wantBehavior string
	}{
		{
			name:         "event",
			raw:          "WHEN a user submits invalid form data THE SYSTEM SHALL display field-level validation errors",
			wantKind:     CriterionEvent,
			wantTrigger:  "a user submits invalid form data",
			wantBehavior: "display field-level validation errors",
		},
		{
			name:         "event with trailing period",
			raw:          "WHEN export completes THE SYSTEM SHALL return HTTP 200.",
			wantKind:     CriterionEvent,
			wantTrigger:  "export completes",
			wantBehavior: "return HTTP 200",
		},
		{
			name:         "event lowercase keywords",
			raw:          "when a user logs out the system shall invalidate the session",
			wantKind:     CriterionEvent,
			wantTrigger:  "a user logs out",
			wantBehavior: "invalidate the session",
		},
		{
			name:         "state",
			raw:          "WHILE a sync is in flight THE SYSTEM SHALL block concurrent sync attempts",
			wantKind:     CriterionState,
			wantTrigger:  "a sync is in flight",
			wantBehavior: "block concurrent sync attempts",
		},
		{
			name:         "optional",
			raw:          "WHERE auto_capture IS ENABLED THE SYSTEM SHALL persist learnings after /deliver",
			wantKind:     CriterionOptional,
			wantTrigger:  "auto_capture IS ENABLED",
			wantBehavior: "persist learnings after /deliver",
		},
		{
			name:         "unwanted",
			raw:          "IF the tracker token is missing THEN THE SYSTEM SHALL print a setup hint and exit non-zero",
			wantKind:     CriterionUnwanted,
			wantTrigger:  "the tracker token is missing",
			wantBehavior: "print a setup hint and exit non-zero",
		},
		{
			name:         "ubiquitous",
			raw:          "THE SYSTEM SHALL log every failed login attempt",
			wantKind:     CriterionUbiquitous,
			wantTrigger:  "",
			wantBehavior: "log every failed login attempt",
		},
		{
			name:         "freeform falls through",
			raw:          "the system should be fast enough",
			wantKind:     CriterionFreeform,
			wantTrigger:  "",
			wantBehavior: "",
		},
		{
			name:         "looks-like but missing SHALL",
			raw:          "WHEN a user logs in they should be redirected",
			wantKind:     CriterionFreeform,
			wantTrigger:  "",
			wantBehavior: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCriterion(tt.raw)
			if got.Kind != tt.wantKind {
				t.Errorf("Kind = %v, want %v", got.Kind, tt.wantKind)
			}
			if got.Trigger != tt.wantTrigger {
				t.Errorf("Trigger = %q, want %q", got.Trigger, tt.wantTrigger)
			}
			if got.Behavior != tt.wantBehavior {
				t.Errorf("Behavior = %q, want %q", got.Behavior, tt.wantBehavior)
			}
			if got.Raw != tt.raw {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.raw)
			}
		})
	}
}

func TestAcceptanceCriteria(t *testing.T) {
	content := `---
title: EARS sample
type: feature
---
# EARS sample

## Acceptance Criteria

- WHEN a user submits a form THE SYSTEM SHALL display validation errors
- THE SYSTEM SHALL log every failed login attempt
- the system should be fast enough
* WHILE a sync is in flight THE SYSTEM SHALL block concurrent attempts
`
	s, err := Parse(content, "/project/.hero/planning/features/ears-sample/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	criteria := s.AcceptanceCriteria()
	if len(criteria) != 4 {
		t.Fatalf("got %d criteria, want 4", len(criteria))
	}

	wantKinds := []CriterionKind{CriterionEvent, CriterionUbiquitous, CriterionFreeform, CriterionState}
	for i, want := range wantKinds {
		if criteria[i].Kind != want {
			t.Errorf("criteria[%d].Kind = %v, want %v", i, criteria[i].Kind, want)
		}
	}
}

func TestAcceptanceCriteriaNoSection(t *testing.T) {
	content := `# No criteria here

## Goal
Do something.
`
	s, err := Parse(content, "/project/.hero/planning/features/no-criteria/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if got := s.AcceptanceCriteria(); got != nil {
		t.Errorf("AcceptanceCriteria() = %v, want nil", got)
	}
}

func TestCriterionKindString(t *testing.T) {
	tests := []struct {
		k    CriterionKind
		want string
	}{
		{CriterionFreeform, "freeform"},
		{CriterionUbiquitous, "ubiquitous"},
		{CriterionEvent, "event"},
		{CriterionState, "state"},
		{CriterionOptional, "optional"},
		{CriterionUnwanted, "unwanted"},
	}
	for _, tt := range tests {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("String() for %d = %q, want %q", tt.k, got, tt.want)
		}
	}
	if CriterionFreeform.IsEARS() {
		t.Error("Freeform should not be EARS")
	}
	if !CriterionEvent.IsEARS() {
		t.Error("Event should be EARS")
	}
}

func TestParsePrioritySeverity(t *testing.T) {
	content := `---
title: Crash on export
type: bug
status: planning
priority: high
severity: critical
tracker_id: PROJ-99
---
# Crash on export
`
	s, err := Parse(content, "/project/.hero/planning/bugs/crash-on-export/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Priority != "high" {
		t.Errorf("Priority = %q, want %q", s.Priority, "high")
	}
	if s.Severity != "critical" {
		t.Errorf("Severity = %q, want %q", s.Severity, "critical")
	}
}

func TestParseSize(t *testing.T) {
	content := `---
title: Some Feature
type: feature
status: planning
size: medium
---
# Some Feature
`
	s, err := Parse(content, "/project/.hero/planning/features/some-feature/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Size != "medium" {
		t.Errorf("Size = %q, want %q", s.Size, "medium")
	}
}

func TestParseSize_AbsentLeavesUnset(t *testing.T) {
	content := `---
title: Some Feature
type: feature
status: planning
---
# Some Feature
`
	s, err := Parse(content, "/project/.hero/planning/features/some-feature/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Size != "" {
		t.Errorf("Size = %q, want empty (unset has no default)", s.Size)
	}
}

func TestParseSize_AllValidValuesRoundTrip(t *testing.T) {
	tiers := []string{"trivial", "small", "medium", "large", "x-large", "giant"}
	for _, tier := range tiers {
		t.Run(tier, func(t *testing.T) {
			content := `---
title: Some Feature
type: feature
status: planning
size: ` + tier + `
---
# Some Feature
`
			s, err := Parse(content, "/project/.hero/planning/features/some-feature/spec.md", time.Now())
			if err != nil {
				t.Fatalf("Parse failed for size %q: %v", tier, err)
			}
			if s.Size != tier {
				t.Errorf("Size = %q, want %q", s.Size, tier)
			}
		})
	}
}

func TestParseSize_UnknownValuePassesThrough(t *testing.T) {
	content := `---
title: Some Feature
type: feature
status: planning
size: bogus
---
# Some Feature
`
	s, err := Parse(content, "/project/.hero/planning/features/some-feature/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse should not fail on unknown size: %v", err)
	}
	if s.Size != "bogus" {
		t.Errorf("Size = %q, want %q (unknown values pass through)", s.Size, "bogus")
	}
}

func TestParseSize_AbbreviationsNormalize(t *testing.T) {
	cases := map[string]string{
		"XS": "trivial", "S": "small", "M": "medium",
		"L": "large", "XL": "x-large", "G": "giant",
		"l": "large", " L ": "large",
	}
	for abbrev, want := range cases {
		t.Run(abbrev, func(t *testing.T) {
			content := "---\ntitle: F\ntype: feature\nstatus: planning\nsize: " + abbrev + "\n---\n# F\n"
			s, err := Parse(content, "/project/.hero/planning/features/f/spec.md", time.Now())
			if err != nil {
				t.Fatalf("Parse failed for size %q: %v", abbrev, err)
			}
			if s.Size != want {
				t.Errorf("Size = %q, want %q", s.Size, want)
			}
		})
	}
}

func TestParseSizeAck(t *testing.T) {
	content := `---
title: Big Initiative
type: initiative
status: planning
size: giant
size_ack: giant
---
# Big Initiative
`
	s, err := Parse(content, "/project/.hero/planning/initiatives/big-initiative/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.SizeAck != "giant" {
		t.Errorf("SizeAck = %q, want %q", s.SizeAck, "giant")
	}
}

func TestParseSmoke_Deferred(t *testing.T) {
	content := `---
title: Some Feature
type: feature
status: planning
smoke: deferred
---
# Some Feature
`
	s, err := Parse(content, "/project/.hero/planning/features/some-feature/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Smoke == nil {
		t.Fatal("Smoke should not be nil when smoke: deferred is set")
	}
	if !s.Smoke.Deferred {
		t.Error("Smoke.Deferred should be true for smoke: deferred")
	}
	if s.Smoke.Script != "" {
		t.Errorf("Smoke.Script should be empty for deferred, got %q", s.Smoke.Script)
	}
}

func TestParseSmoke_Block(t *testing.T) {
	content := `---
title: Some Feature
type: feature
status: planning
smoke:
  script: scripts/smoke/some-feature.sh
  expects: [some-feature:AC-1, some-feature:AC-2]
  runs_on: [commit-touches:internal/cli/*.go, nightly]
---
# Some Feature
`
	s, err := Parse(content, "/project/.hero/planning/features/some-feature/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Smoke == nil {
		t.Fatal("Smoke should not be nil when smoke: block is present")
	}
	if s.Smoke.Deferred {
		t.Error("Smoke.Deferred should be false for a full block")
	}
	if s.Smoke.Script != "scripts/smoke/some-feature.sh" {
		t.Errorf("Smoke.Script = %q, want %q", s.Smoke.Script, "scripts/smoke/some-feature.sh")
	}
	if len(s.Smoke.Expects) != 2 || s.Smoke.Expects[0] != "some-feature:AC-1" {
		t.Errorf("Smoke.Expects = %v, want [some-feature:AC-1, some-feature:AC-2]", s.Smoke.Expects)
	}
	if len(s.Smoke.RunsOn) != 2 || s.Smoke.RunsOn[1] != "nightly" {
		t.Errorf("Smoke.RunsOn = %v, want [commit-touches:..., nightly]", s.Smoke.RunsOn)
	}
}

func TestParseSmoke_Absent(t *testing.T) {
	content := `---
title: Some Feature
type: feature
status: planning
---
# Some Feature
`
	s, err := Parse(content, "/project/.hero/planning/features/some-feature/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Smoke != nil {
		t.Error("Smoke should be nil when smoke: field is absent")
	}
}

func TestParseTripwireSpec(t *testing.T) {
	content := `---
title: Do Not Use PyO3
type: tripwire
status: active
triggers: [pyo3, "python bindings", "python wrapper"]
scope: ["*.rs", "*.py"]
severity: critical
---
# Do Not Use PyO3

## Constraint

Do not propose or implement PyO3 as a solution.

## Why

This project exists to replace the Python wrapper.

## Instead

Use the MLX C++ kernels directly via Rust FFI.
`
	s, err := Parse(content, "/project/.hero/knowledge/tripwires/no-pyo3/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if s.Type != TypeTripwire {
		t.Errorf("Type = %q, want %q", s.Type, TypeTripwire)
	}
	if s.Status != StatusActive {
		t.Errorf("Status = %q, want %q", s.Status, StatusActive)
	}
	if s.Severity != "critical" {
		t.Errorf("Severity = %q, want %q", s.Severity, "critical")
	}
	if len(s.Triggers) != 3 {
		t.Fatalf("Triggers = %v, want 3 items", s.Triggers)
	}
	if s.Triggers[0] != "pyo3" {
		t.Errorf("Triggers[0] = %q, want %q", s.Triggers[0], "pyo3")
	}
	if s.Triggers[1] != "python bindings" {
		t.Errorf("Triggers[1] = %q, want %q", s.Triggers[1], "python bindings")
	}
	if len(s.Scope) != 2 {
		t.Errorf("Scope = %v, want 2 items", s.Scope)
	}
	if s.Sections["constraint"] == "" {
		t.Error("Constraint section should not be empty")
	}
	if s.Sections["why"] == "" {
		t.Error("Why section should not be empty")
	}
	if s.Sections["instead"] == "" {
		t.Error("Instead section should not be empty")
	}
	if !s.IsKnowledge() {
		t.Error("Tripwire should be classified as knowledge")
	}
}

func TestParseKickoffSection(t *testing.T) {
	content := `---
title: Has Kickoff
type: feature
status: planning
pinned: true
---
# Has Kickoff

## Kickoff

Short paste-ready opener.

**Status:** planning — design just landed.
**Pick up at:** wire the selector.

→ ` + "`hero queue`" + `

## Goal

The kickoff above is what hero queue surfaces.
`
	s, err := Parse(content, "/project/.hero/specs/has-kickoff/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !s.Pinned {
		t.Error("Pinned = false, want true")
	}

	got := s.Kickoff()
	if got == "" {
		t.Fatal("Kickoff() returned empty string")
	}
	if !strings.Contains(got, "Short paste-ready opener.") {
		t.Errorf("Kickoff() missing opener line: %q", got)
	}
	if !strings.Contains(got, "Pick up at:") {
		t.Errorf("Kickoff() missing pick-up line: %q", got)
	}
	if strings.Contains(got, "## Goal") {
		t.Errorf("Kickoff() leaked into next section: %q", got)
	}
}

func TestParseKickoffMissing(t *testing.T) {
	content := `---
title: No Kickoff
type: feature
status: planning
---
## Goal
Just a goal, no kickoff section.
`
	s, err := Parse(content, "/project/.hero/specs/no-kickoff/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Kickoff() != "" {
		t.Errorf("Kickoff() = %q, want empty for spec without ## Kickoff section", s.Kickoff())
	}
	if s.Pinned {
		t.Error("Pinned should default to false when frontmatter omits it")
	}
}

func TestParsePinnedVariants(t *testing.T) {
	cases := map[string]bool{
		"true":  true,
		"True":  true,
		"yes":   true,
		"on":    true,
		"1":     true,
		"false": false,
		"no":    false,
		"":      false,
	}
	for raw, want := range cases {
		content := `---
title: T
type: feature
status: planning
pinned: ` + raw + `
---
## Goal
x
`
		s, err := Parse(content, "/p/.hero/specs/t/spec.md", time.Now())
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", raw, err)
		}
		if s.Pinned != want {
			t.Errorf("pinned: %q -> %v, want %v", raw, s.Pinned, want)
		}
	}
}

// TestParseSupersededBy covers the new authoritative genealogy field
// added by spec superseded-specs-soft-archive.
func TestParseSupersededBy(t *testing.T) {
	content := `---
title: V1 surface polish
type: feature
status: completed
superseded_by: hero-surface-polish-v2
---
## Goal
old
`
	s, err := Parse(content, "/p/.hero/specs/hero-surface-polish-v1/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.SupersededBy != "hero-surface-polish-v2" {
		t.Errorf("SupersededBy = %q, want hero-surface-polish-v2", s.SupersededBy)
	}
	// Lifecycle stays orthogonal: status remains completed.
	if s.Status != StatusCompleted {
		t.Errorf("Status = %q, want completed (genealogy is orthogonal to lifecycle)", s.Status)
	}
	if !s.IsSuperseded() {
		t.Error("IsSuperseded() = false, want true (superseded_by is set)")
	}
}

// TestIsSuperseded_LegacyStatus exercises the back-compat path where a
// spec carries the legacy `status: superseded` enum value without the
// new authoritative field.
func TestIsSuperseded_LegacyStatus(t *testing.T) {
	content := `---
title: Old
type: feature
status: superseded
---
## Goal
x
`
	s, err := Parse(content, "/p/.hero/specs/old/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.SupersededBy != "" {
		t.Errorf("SupersededBy = %q, want empty", s.SupersededBy)
	}
	if !s.IsSuperseded() {
		t.Error("IsSuperseded() = false, want true (status enum still counts)")
	}
}

// TestRenderSpecBody_SupersededBanner verifies the render-time banner
// is prepended for superseded specs without mutating the on-disk body.
func TestRenderSpecBody_SupersededBanner(t *testing.T) {
	body := "# Title\n\nbody text\n"
	s := &Spec{SupersededBy: "new-spec"}
	got := RenderSpecBody(s, body)
	if !strings.HasPrefix(got, "> **SUPERSEDED by new-spec**") {
		t.Errorf("rendered body missing banner; got:\n%s", got)
	}
	if !strings.Contains(got, "# Title") {
		t.Error("rendered body lost the original content")
	}

	// Non-superseded spec is returned unchanged.
	plain := &Spec{}
	if got := RenderSpecBody(plain, body); got != body {
		t.Error("non-superseded spec should be returned unchanged")
	}

	// status: superseded without a replacement slug gets an
	// "unknown" banner so the ambiguity is visible.
	legacy := &Spec{Status: StatusSuperseded}
	gotLegacy := RenderSpecBody(legacy, body)
	if !strings.Contains(gotLegacy, "replacement unknown") {
		t.Errorf("legacy banner missing 'replacement unknown'; got:\n%s", gotLegacy)
	}
}

func TestTripwireTypeFromPath(t *testing.T) {
	tp := typeFromPath("/project/.hero/knowledge/tripwires/no-pyo3/spec.md")
	if tp != TypeTripwire {
		t.Errorf("typeFromPath = %q, want %q", tp, TypeTripwire)
	}

	st := statusFromPath("/project/.hero/knowledge/tripwires/no-pyo3/spec.md")
	if st != StatusActive {
		t.Errorf("statusFromPath = %q, want %q", st, StatusActive)
	}
}

func TestExplainerParse(t *testing.T) {
	content := `---
title: How Agent Outposts Work
type: explainer
synthesized_from:
  - agent-outposts
  - outpost-credentials
last_synthesized: 2026-06-23
---
# How Agent Outposts Work

## What it is

Operable external systems with scoped credentials.

## Developer Notes

Watch the token refresh path.
`
	s, err := Parse(content, "/project/.hero/knowledge/explainers/agent-outposts/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Type != TypeExplainer {
		t.Errorf("Type = %q, want %q", s.Type, TypeExplainer)
	}
	if !s.IsKnowledge() {
		t.Error("explainer should be classified as knowledge")
	}
	if s.IsWorkSpec() {
		t.Error("explainer must not be classified as a work spec")
	}
	if len(s.SynthesizedFrom) != 2 || s.SynthesizedFrom[0] != "agent-outposts" {
		t.Errorf("SynthesizedFrom = %v, want [agent-outposts outpost-credentials]", s.SynthesizedFrom)
	}
	if s.LastSynthesized != "2026-06-23" {
		t.Errorf("LastSynthesized = %q, want %q", s.LastSynthesized, "2026-06-23")
	}
}

// withNowFn swaps the package-level nowFn for the duration of the test
// and restores the original on cleanup. Tests use this to make the
// `completed_at:` stamp deterministic.
func withNowFn(t *testing.T, fixed time.Time) {
	t.Helper()
	prev := nowFn
	nowFn = func() time.Time { return fixed }
	t.Cleanup(func() { nowFn = prev })
}

func TestStampCompletedAt_Idempotent(t *testing.T) {
	fixed := time.Date(2026, 5, 31, 19, 42, 8, 0, time.UTC)
	withNowFn(t, fixed)

	in := "---\ntitle: x\nstatus: completed\n---\nbody\n"
	first := StampCompletedAt(in)
	if !strings.Contains(first, "completed_at: 2026-05-31T19:42:08Z") {
		t.Fatalf("first stamp did not add expected completed_at: %q", first)
	}

	// Move the clock forward; second call must be a no-op.
	nowFn = func() time.Time { return fixed.Add(time.Hour) }
	second := StampCompletedAt(first)
	if second != first {
		t.Fatalf("second stamp mutated content:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestStampCompletedAt_RespectsCamelCase(t *testing.T) {
	fixed := time.Date(2026, 5, 31, 19, 42, 8, 0, time.UTC)
	withNowFn(t, fixed)

	in := "---\ntitle: x\nstatus: completed\ncompletedAt: 2025-01-02T03:04:05Z\n---\nbody\n"
	out := StampCompletedAt(in)
	if out != in {
		t.Fatalf("camelCase completedAt should be left alone, got:\n%s", out)
	}
	if strings.Contains(out, "completed_at:") {
		t.Fatalf("camelCase content gained snake_case completed_at:\n%s", out)
	}
}

func TestParseFrontmatter_CompletedAt_RFC3339(t *testing.T) {
	content := `---
title: t
type: feature
status: completed
completed_at: 2026-05-31T19:42:08Z
---
body
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 31, 19, 42, 8, 0, time.UTC)
	if !s.CompletedAt.Equal(want) {
		t.Errorf("CompletedAt = %v, want %v", s.CompletedAt, want)
	}
}

func TestParseFrontmatter_CompletedAt_DateOnly(t *testing.T) {
	content := `---
title: t
type: feature
status: completed
completed_at: 2026-05-31
---
body
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	if !s.CompletedAt.Equal(want) {
		t.Errorf("CompletedAt = %v, want %v", s.CompletedAt, want)
	}
}

func TestParseFrontmatter_CompletedAt_CamelCase(t *testing.T) {
	content := `---
title: t
type: feature
status: completed
completedAt: 2026-05-31T19:42:08Z
---
body
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 31, 19, 42, 8, 0, time.UTC)
	if !s.CompletedAt.Equal(want) {
		t.Errorf("CompletedAt = %v, want %v", s.CompletedAt, want)
	}
}
