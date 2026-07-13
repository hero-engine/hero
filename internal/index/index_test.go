package index

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

func setupTestDB(t *testing.T) (*DB, string) {
	t.Helper()
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	t.Cleanup(func() { idx.Close() })
	return idx, heroDir
}

func makeSpec(slug, title string, specType spec.Type, status spec.Status) *spec.Spec {
	return &spec.Spec{
		Slug:       slug,
		Title:      title,
		Type:       specType,
		Status:     status,
		Path:       "/project/.hero/specs/" + slug + "/spec.md",
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
		Sections:   make(map[string]string),
	}
}

func TestOpenAndClose(t *testing.T) {
	idx, _ := setupTestDB(t)
	if idx == nil {
		t.Fatal("DB should not be nil")
	}
}

func TestIndexAndSearchSpec(t *testing.T) {
	idx, _ := setupTestDB(t)

	s := makeSpec("add-csv-export", "Add CSV Export", spec.TypeFeature, spec.StatusCompleted)
	s.FilesTouched = []string{"src/api/users.ts", "src/export/csv.ts"}

	content := "# Add CSV Export\n\nExport user data to CSV format with streaming support."

	if err := idx.IndexSpec(s, content); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Full-text search
	results, err := idx.Search("CSV export")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search returned %d results, want 1", len(results))
	}
	if results[0].Slug != "add-csv-export" {
		t.Errorf("Slug = %q, want %q", results[0].Slug, "add-csv-export")
	}
	if results[0].Type != spec.TypeFeature {
		t.Errorf("Type = %q, want %q", results[0].Type, spec.TypeFeature)
	}

	// Search that shouldn't match
	results, err = idx.Search("authentication oauth")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search returned %d results, want 0", len(results))
	}
}

func TestSearchByFile(t *testing.T) {
	idx, _ := setupTestDB(t)

	s := makeSpec("add-csv-export", "Add CSV Export", spec.TypeFeature, spec.StatusCompleted)
	s.FilesTouched = []string{"src/api/users.ts", "src/export/csv.ts"}

	if err := idx.IndexSpec(s, "# CSV Export"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	results, err := idx.SearchByFile("src/api/users.ts")
	if err != nil {
		t.Fatalf("SearchByFile failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchByFile returned %d results, want 1", len(results))
	}

	results, err = idx.SearchByFile("src/unrelated/file.ts")
	if err != nil {
		t.Fatalf("SearchByFile failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchByFile returned %d results, want 0", len(results))
	}
}

func TestSearchFiltered(t *testing.T) {
	idx, _ := setupTestDB(t)

	s1 := makeSpec("feat-one", "Feature One", spec.TypeFeature, spec.StatusCompleted)
	s2 := makeSpec("bug-one", "Bug One", spec.TypeBug, spec.StatusCompleted)

	if err := idx.IndexSpec(s1, "# Feature One\nSome feature content"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}
	if err := idx.IndexSpec(s2, "# Bug One\nSome bug content"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Filter by type
	results, err := idx.SearchFiltered("content", "feature", "", "", "")
	if err != nil {
		t.Fatalf("SearchFiltered failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchFiltered returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Slug != "feat-one" {
		t.Errorf("Slug = %q, want %q", results[0].Slug, "feat-one")
	}

	// Filter by status
	results, err = idx.SearchFiltered("content", "", "completed", "", "")
	if err != nil {
		t.Fatalf("SearchFiltered failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("SearchFiltered returned %d results, want 2", len(results))
	}
}

func TestListFiltered(t *testing.T) {
	idx, _ := setupTestDB(t)

	s1 := makeSpec("feat-one", "Feature One", spec.TypeFeature, spec.StatusPlanning)
	s2 := makeSpec("feat-two", "Feature Two", spec.TypeFeature, spec.StatusCompleted)
	s3 := makeSpec("bug-one", "Bug One", spec.TypeBug, spec.StatusPlanning)

	for _, s := range []*spec.Spec{s1, s2, s3} {
		if err := idx.IndexSpec(s, "# "+s.Title); err != nil {
			t.Fatalf("IndexSpec failed: %v", err)
		}
	}

	// List all
	results, err := idx.ListFiltered("", "", "", "")
	if err != nil {
		t.Fatalf("ListFiltered failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("ListFiltered returned %d results, want 3", len(results))
	}

	// List features only
	results, err = idx.ListFiltered("feature", "", "", "")
	if err != nil {
		t.Fatalf("ListFiltered failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("ListFiltered(feature) returned %d results, want 2", len(results))
	}

	// List planning only
	results, err = idx.ListFiltered("", "planning", "", "")
	if err != nil {
		t.Fatalf("ListFiltered failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("ListFiltered(planning) returned %d results, want 2", len(results))
	}
}

func TestConventionScopes(t *testing.T) {
	idx, _ := setupTestDB(t)

	conv := makeSpec("api-format", "API Response Format", spec.TypeConvention, spec.StatusActive)
	conv.Scope = []string{"src/api/*.ts"}

	if err := idx.IndexSpec(conv, "# API Response Format"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// File that matches the scope
	results, err := idx.FindConventionsForFiles([]string{"src/api/users.ts"})
	if err != nil {
		t.Fatalf("FindConventionsForFiles failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("FindConventionsForFiles returned %d results, want 1", len(results))
	}

	// File that doesn't match
	results, err = idx.FindConventionsForFiles([]string{"src/db/queries.sql"})
	if err != nil {
		t.Fatalf("FindConventionsForFiles failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("FindConventionsForFiles returned %d results, want 0", len(results))
	}
}

func TestConventionCatchAllScope(t *testing.T) {
	idx, _ := setupTestDB(t)

	conv := makeSpec("logging-standard", "Logging Standard", spec.TypeConvention, spec.StatusActive)
	conv.Scope = []string{"*"}

	if err := idx.IndexSpec(conv, "# Logging Standard"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	results, err := idx.FindConventionsForFiles([]string{"anything/at/all.go"})
	if err != nil {
		t.Fatalf("FindConventionsForFiles failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("FindConventionsForFiles returned %d results, want 1", len(results))
	}
}

func TestFindConflicts(t *testing.T) {
	idx, _ := setupTestDB(t)

	s1 := makeSpec("feat-a", "Feature A", spec.TypeFeature, spec.StatusDelivering)
	s1.FilesTouched = []string{"src/api/users.ts", "src/db/users.sql"}

	s2 := makeSpec("feat-b", "Feature B", spec.TypeFeature, spec.StatusPlanning)
	s2.FilesTouched = []string{"src/api/users.ts", "src/api/auth.ts"}

	s3 := makeSpec("feat-c", "Feature C", spec.TypeFeature, spec.StatusCompleted)
	s3.FilesTouched = []string{"src/api/users.ts"} // completed, should not show as conflict

	for _, s := range []*spec.Spec{s1, s2, s3} {
		if err := idx.IndexSpec(s, "# "+s.Title); err != nil {
			t.Fatalf("IndexSpec failed: %v", err)
		}
	}

	conflicts, err := idx.FindConflicts("feat-a")
	if err != nil {
		t.Fatalf("FindConflicts failed: %v", err)
	}

	// feat-b overlaps on src/api/users.ts, feat-c is completed so not a conflict
	if len(conflicts) != 1 {
		t.Fatalf("FindConflicts returned %d conflicts, want 1", len(conflicts))
	}
	if conflicts[0].Slug != "feat-b" {
		t.Errorf("Conflict slug = %q, want %q", conflicts[0].Slug, "feat-b")
	}
	if len(conflicts[0].OverlappingFiles) != 1 || conflicts[0].OverlappingFiles[0] != "src/api/users.ts" {
		t.Errorf("OverlappingFiles = %v, want [src/api/users.ts]", conflicts[0].OverlappingFiles)
	}
}

// TestFindDeliveringConflicts: the delivering-scoped variant includes only
// specs whose status is delivering — planning and in-review overlaps are
// excluded (mirrors the /drive judge's IsLocallyDelivering scope).
func TestFindDeliveringConflicts(t *testing.T) {
	idx, _ := setupTestDB(t)

	// The candidate.
	s1 := makeSpec("feat-a", "Feature A", spec.TypeFeature, spec.StatusPlanning)
	s1.FilesTouched = []string{"src/api/users.ts", "src/db/users.sql"}

	// Delivering overlap → included.
	s2 := makeSpec("feat-b", "Feature B", spec.TypeFeature, spec.StatusDelivering)
	s2.FilesTouched = []string{"src/api/users.ts"}

	// Planning overlap → excluded by the delivering filter.
	s3 := makeSpec("feat-c", "Feature C", spec.TypeFeature, spec.StatusPlanning)
	s3.FilesTouched = []string{"src/api/users.ts"}

	// In-review overlap → excluded.
	s4 := makeSpec("feat-d", "Feature D", spec.TypeFeature, spec.StatusInReview)
	s4.FilesTouched = []string{"src/db/users.sql"}

	for _, s := range []*spec.Spec{s1, s2, s3, s4} {
		if err := idx.IndexSpec(s, "# "+s.Title); err != nil {
			t.Fatalf("IndexSpec failed: %v", err)
		}
	}

	// FindConflicts (all in-flight statuses) sees b, c, and d.
	all, err := idx.FindConflicts("feat-a")
	if err != nil {
		t.Fatalf("FindConflicts failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("FindConflicts returned %d, want 3 (feat-b, feat-c, feat-d)", len(all))
	}

	// FindDeliveringConflicts sees only the delivering feat-b.
	delivering, err := idx.FindDeliveringConflicts("feat-a")
	if err != nil {
		t.Fatalf("FindDeliveringConflicts failed: %v", err)
	}
	if len(delivering) != 1 {
		t.Fatalf("FindDeliveringConflicts returned %d, want 1", len(delivering))
	}
	if delivering[0].Slug != "feat-b" {
		t.Errorf("delivering conflict slug = %q, want feat-b", delivering[0].Slug)
	}
	if len(delivering[0].OverlappingFiles) != 1 || delivering[0].OverlappingFiles[0] != "src/api/users.ts" {
		t.Errorf("OverlappingFiles = %v, want [src/api/users.ts]", delivering[0].OverlappingFiles)
	}
}

func TestClaimAndUnclaim(t *testing.T) {
	idx, _ := setupTestDB(t)

	s := makeSpec("my-spec", "My Spec", spec.TypeFeature, spec.StatusPlanning)
	if err := idx.IndexSpec(s, "# My Spec"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Claim
	if err := idx.Claim("my-spec", "alice"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Verify claim shows in specs
	results, err := idx.ListFiltered("", "", "", "")
	if err != nil {
		t.Fatalf("ListFiltered failed: %v", err)
	}
	if len(results) != 1 || results[0].ClaimedBy != "alice" {
		t.Errorf("ClaimedBy = %q, want %q", results[0].ClaimedBy, "alice")
	}

	// Double claim by same person should succeed
	if err := idx.Claim("my-spec", "alice"); err != nil {
		t.Errorf("Re-claim by same person failed: %v", err)
	}

	// Claim by different person should fail
	if err := idx.Claim("my-spec", "bob"); err == nil {
		t.Error("Claim by different person should fail")
	}

	// Unclaim
	if err := idx.Unclaim("my-spec"); err != nil {
		t.Fatalf("Unclaim failed: %v", err)
	}

	results, err = idx.ListFiltered("", "", "", "")
	if err != nil {
		t.Fatalf("ListFiltered failed: %v", err)
	}
	if len(results) != 1 || results[0].ClaimedBy != "" {
		t.Errorf("ClaimedBy after unclaim = %q, want empty", results[0].ClaimedBy)
	}

	// Now bob can claim
	if err := idx.Claim("my-spec", "bob"); err != nil {
		t.Errorf("Claim by bob after unclaim failed: %v", err)
	}
}

func TestClaimNonexistent(t *testing.T) {
	idx, _ := setupTestDB(t)

	err := idx.Claim("nonexistent", "alice")
	if err == nil {
		t.Error("Claim on nonexistent spec should fail")
	}
}

func TestGetRelations(t *testing.T) {
	idx, _ := setupTestDB(t)

	parent := makeSpec("initiative-auth", "Auth Initiative", spec.TypeInitiative, spec.StatusPlanning)
	child := makeSpec("add-mfa", "Add MFA", spec.TypeFeature, spec.StatusPlanning)
	child.Relations = []spec.Relation{
		{Target: "initiative-auth", Kind: "parent"},
		{Target: "add-session", Kind: "depends-on"},
	}
	dep := makeSpec("add-session", "Add Session Management", spec.TypeFeature, spec.StatusCompleted)

	for _, s := range []*spec.Spec{parent, child, dep} {
		if err := idx.IndexSpec(s, "# "+s.Title); err != nil {
			t.Fatalf("IndexSpec failed: %v", err)
		}
	}

	rels, err := idx.GetRelations("add-mfa")
	if err != nil {
		t.Fatalf("GetRelations failed: %v", err)
	}

	if len(rels) != 2 {
		t.Fatalf("GetRelations returned %d relations, want 2", len(rels))
	}

	relMap := make(map[string]RelationResult)
	for _, r := range rels {
		relMap[r.Relation] = r
	}

	if r, ok := relMap["parent"]; !ok {
		t.Error("Missing parent relation")
	} else if r.Slug != "initiative-auth" {
		t.Errorf("Parent slug = %q, want %q", r.Slug, "initiative-auth")
	}

	if r, ok := relMap["depends-on"]; !ok {
		t.Error("Missing depends-on relation")
	} else if r.Slug != "add-session" {
		t.Errorf("Depends-on slug = %q, want %q", r.Slug, "add-session")
	}
}

func TestGetStats(t *testing.T) {
	idx, _ := setupTestDB(t)

	specs := []*spec.Spec{
		makeSpec("feat-1", "F1", spec.TypeFeature, spec.StatusPlanning),
		makeSpec("feat-2", "F2", spec.TypeFeature, spec.StatusCompleted),
		makeSpec("bug-1", "B1", spec.TypeBug, spec.StatusDelivering),
		makeSpec("conv-1", "C1", spec.TypeConvention, spec.StatusActive),
		makeSpec("dec-1", "D1", spec.TypeDecision, spec.StatusAccepted),
	}
	specs[0].FilesTouched = []string{"file1.go", "file2.go"}
	specs[2].FilesTouched = []string{"file3.go"}

	for _, s := range specs {
		if err := idx.IndexSpec(s, "# "+s.Title); err != nil {
			t.Fatalf("IndexSpec failed: %v", err)
		}
	}

	stats, err := idx.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalSpecs != 5 {
		t.Errorf("TotalSpecs = %d, want 5", stats.TotalSpecs)
	}
	if stats.Features != 2 {
		t.Errorf("Features = %d, want 2", stats.Features)
	}
	if stats.Bugs != 1 {
		t.Errorf("Bugs = %d, want 1", stats.Bugs)
	}
	if stats.Conventions != 1 {
		t.Errorf("Conventions = %d, want 1", stats.Conventions)
	}
	if stats.Decisions != 1 {
		t.Errorf("Decisions = %d, want 1", stats.Decisions)
	}
	if stats.Planning != 1 {
		t.Errorf("Planning = %d, want 1", stats.Planning)
	}
	if stats.Delivering != 1 {
		t.Errorf("Delivering = %d, want 1", stats.Delivering)
	}
	if stats.Completed != 1 {
		t.Errorf("Completed = %d, want 1", stats.Completed)
	}
	if stats.Active != 1 {
		t.Errorf("Active = %d, want 1", stats.Active)
	}
	if stats.Accepted != 1 {
		t.Errorf("Accepted = %d, want 1", stats.Accepted)
	}
	if stats.FilesTracked != 3 {
		t.Errorf("FilesTracked = %d, want 3", stats.FilesTracked)
	}
}

func TestRemoveSpec(t *testing.T) {
	idx, _ := setupTestDB(t)

	s := makeSpec("to-remove", "Remove Me", spec.TypeFeature, spec.StatusPlanning)
	s.FilesTouched = []string{"file.go"}
	if err := idx.IndexSpec(s, "# Remove Me"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Verify it exists
	results, err := idx.Search("Remove")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result before removal, got %d", len(results))
	}

	// Remove
	if err := idx.RemoveSpec("to-remove"); err != nil {
		t.Fatalf("RemoveSpec failed: %v", err)
	}

	// Verify it's gone
	results, err = idx.Search("Remove")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results after removal, got %d", len(results))
	}
}

func TestCheckStale(t *testing.T) {
	idx, _ := setupTestDB(t)

	// Create a spec with old modification time
	old := makeSpec("old-spec", "Old Spec", spec.TypeFeature, spec.StatusPlanning)
	old.ModifiedAt = time.Now().AddDate(0, 0, -30) // 30 days ago

	fresh := makeSpec("fresh-spec", "Fresh Spec", spec.TypeFeature, spec.StatusPlanning)

	for _, s := range []*spec.Spec{old, fresh} {
		if err := idx.IndexSpec(s, "# "+s.Title); err != nil {
			t.Fatalf("IndexSpec failed: %v", err)
		}
	}

	stale, err := idx.CheckStale(14)
	if err != nil {
		t.Fatalf("CheckStale failed: %v", err)
	}

	if len(stale) != 1 {
		t.Fatalf("CheckStale returned %d results, want 1", len(stale))
	}
	if stale[0].Slug != "old-spec" {
		t.Errorf("Stale slug = %q, want %q", stale[0].Slug, "old-spec")
	}
}

func TestCheckUnclaimed(t *testing.T) {
	idx, _ := setupTestDB(t)

	s1 := makeSpec("unclaimed", "Unclaimed Spec", spec.TypeFeature, spec.StatusPlanning)
	s2 := makeSpec("claimed", "Claimed Spec", spec.TypeFeature, spec.StatusPlanning)
	s3 := makeSpec("done", "Done Spec", spec.TypeFeature, spec.StatusCompleted)

	for _, s := range []*spec.Spec{s1, s2, s3} {
		if err := idx.IndexSpec(s, "# "+s.Title); err != nil {
			t.Fatalf("IndexSpec failed: %v", err)
		}
	}

	if err := idx.Claim("claimed", "alice"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	unclaimed, err := idx.CheckUnclaimed()
	if err != nil {
		t.Fatalf("CheckUnclaimed failed: %v", err)
	}

	if len(unclaimed) != 1 {
		t.Fatalf("CheckUnclaimed returned %d results, want 1", len(unclaimed))
	}
	if unclaimed[0].Slug != "unclaimed" {
		t.Errorf("Unclaimed slug = %q, want %q", unclaimed[0].Slug, "unclaimed")
	}
}

func TestBuildContext(t *testing.T) {
	idx, _ := setupTestDB(t)

	// Add a convention
	conv := makeSpec("api-format", "API Response Format", spec.TypeConvention, spec.StatusActive)
	conv.Scope = []string{"src/api/*.ts"}
	if err := idx.IndexSpec(conv, "# API Response Format"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Add a completed feature that touched the same files
	feat := makeSpec("add-users", "Add Users", spec.TypeFeature, spec.StatusCompleted)
	feat.FilesTouched = []string{"src/api/users.ts"}
	if err := idx.IndexSpec(feat, "# Add Users"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Add a decision
	dec := makeSpec("use-postgres", "Use PostgreSQL", spec.TypeDecision, spec.StatusAccepted)
	if err := idx.IndexSpec(dec, "# Use PostgreSQL"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Add a bug that touched the same files
	bug := makeSpec("user-crash", "User API Crash", spec.TypeBug, spec.StatusCompleted)
	bug.FilesTouched = []string{"src/api/users.ts"}
	if err := idx.IndexSpec(bug, "# User API Crash"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	ctx, err := idx.BuildContext([]string{"src/api/users.ts"})
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if ctx.IsEmpty() {
		t.Error("Context should not be empty")
	}

	if len(ctx.Conventions) != 1 {
		t.Errorf("Conventions = %d, want 1", len(ctx.Conventions))
	}
	if len(ctx.PastWork) < 1 {
		t.Errorf("PastWork = %d, want >= 1", len(ctx.PastWork))
	}
	if len(ctx.Decisions) != 1 {
		t.Errorf("Decisions = %d, want 1", len(ctx.Decisions))
	}
	if len(ctx.KnownRisks) != 1 {
		t.Errorf("KnownRisks = %d, want 1", len(ctx.KnownRisks))
	}
}

func TestBuildContextEmpty(t *testing.T) {
	idx, _ := setupTestDB(t)

	ctx, err := idx.BuildContext([]string{"src/unrelated/file.go"})
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if !ctx.IsEmpty() {
		t.Error("Context should be empty for unrelated files")
	}
}

func TestBuildNudge(t *testing.T) {
	idx, _ := setupTestDB(t)

	// Add a convention with catch-all scope
	conv := makeSpec("error-handling", "Error Handling", spec.TypeConvention, spec.StatusActive)
	conv.Scope = []string{"*"}
	if err := idx.IndexSpec(conv, "# Error Handling"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Add a completed spec
	feat := makeSpec("past-work", "Past Work", spec.TypeFeature, spec.StatusCompleted)
	feat.FilesTouched = []string{"src/main.go"}
	if err := idx.IndexSpec(feat, "# Past Work"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Add an in-flight spec
	inflight := makeSpec("active-work", "Active Work", spec.TypeFeature, spec.StatusDelivering)
	inflight.FilesTouched = []string{"src/main.go"}
	if err := idx.IndexSpec(inflight, "# Active Work"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	result, err := idx.BuildNudge([]string{"src/main.go"})
	if err != nil {
		t.Fatalf("BuildNudge failed: %v", err)
	}

	if result.IsEmpty() {
		t.Error("Nudge should not be empty")
	}
	if !result.HasConventions {
		t.Error("HasConventions should be true")
	}
	if !result.HasPastWork {
		t.Error("HasPastWork should be true")
	}
	if !result.HasPending {
		t.Error("HasPending should be true")
	}

	if len(result.Conventions) != 1 {
		t.Errorf("Conventions = %d, want 1", len(result.Conventions))
	}
	if len(result.RelatedSpecs) != 1 {
		t.Errorf("RelatedSpecs = %d, want 1", len(result.RelatedSpecs))
	}
	if len(result.PendingSpecs) != 1 {
		t.Errorf("PendingSpecs = %d, want 1", len(result.PendingSpecs))
	}
}

// TestBuildNudge_SurfacesSupersededWithMarker — superseded specs are
// kept in RelatedSpecs (not filtered out) and carry their replacement
// slug in SupersededBy so renderers can add the redirect annotation.
// Covers ACs:
//   - "WHEN BuildNudge is called for a file touched by a superseded
//     spec THE SYSTEM SHALL include that spec in RelatedSpecs with
//     SupersededBy populated."
func TestBuildNudge_SurfacesSupersededWithMarker(t *testing.T) {
	idx, _ := setupTestDB(t)

	old := makeSpec("surface-polish-v1", "V1", spec.TypeFeature, spec.StatusCompleted)
	old.FilesTouched = []string{"src/main.go"}
	old.SupersededBy = "surface-polish-v2"
	if err := idx.IndexSpec(old, "# V1"); err != nil {
		t.Fatalf("IndexSpec: %v", err)
	}

	result, err := idx.BuildNudge([]string{"src/main.go"})
	if err != nil {
		t.Fatalf("BuildNudge: %v", err)
	}
	if len(result.RelatedSpecs) != 1 {
		t.Fatalf("RelatedSpecs = %d, want 1 (superseded spec must surface, not be filtered)", len(result.RelatedSpecs))
	}
	got := result.RelatedSpecs[0]
	if got.Slug != "surface-polish-v1" {
		t.Errorf("got slug %q, want surface-polish-v1", got.Slug)
	}
	if got.SupersededBy != "surface-polish-v2" {
		t.Errorf("got SupersededBy = %q, want surface-polish-v2", got.SupersededBy)
	}
}

// TestUpsertSpec_PersistsSupersededBy — round-trip the field through
// the specs table. Covers the AC: "THE SYSTEM SHALL persist superseded_by
// in the spec index and include it in every SearchResult projection."
func TestUpsertSpec_PersistsSupersededBy(t *testing.T) {
	idx, _ := setupTestDB(t)

	s := makeSpec("old", "Old", spec.TypeFeature, spec.StatusCompleted)
	s.SupersededBy = "new"
	if err := idx.IndexSpec(s, "# Old"); err != nil {
		t.Fatalf("IndexSpec: %v", err)
	}

	all, err := idx.AllSpecs()
	if err != nil {
		t.Fatalf("AllSpecs: %v", err)
	}
	var found bool
	for _, r := range all {
		if r.Slug == "old" {
			found = true
			if r.SupersededBy != "new" {
				t.Errorf("SupersededBy = %q, want new", r.SupersededBy)
			}
		}
	}
	if !found {
		t.Fatal("inserted spec not found in AllSpecs")
	}
}

func TestBuildNudgeEmpty(t *testing.T) {
	idx, _ := setupTestDB(t)

	result, err := idx.BuildNudge([]string{"src/nothing/here.go"})
	if err != nil {
		t.Fatalf("BuildNudge failed: %v", err)
	}

	if !result.IsEmpty() {
		t.Error("Nudge should be empty when no context exists")
	}
}

func TestUpsertSpec(t *testing.T) {
	idx, _ := setupTestDB(t)

	s := makeSpec("evolving", "Version 1", spec.TypeFeature, spec.StatusPlanning)
	if err := idx.IndexSpec(s, "# Version 1"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Update the spec
	s.Title = "Version 2"
	s.Status = spec.StatusDelivering
	if err := idx.IndexSpec(s, "# Version 2"); err != nil {
		t.Fatalf("IndexSpec (upsert) failed: %v", err)
	}

	results, err := idx.ListFiltered("", "", "", "")
	if err != nil {
		t.Fatalf("ListFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result after upsert, got %d", len(results))
	}
	if results[0].Title != "Version 2" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Version 2")
	}
	if results[0].Status != spec.StatusDelivering {
		t.Errorf("Status = %q, want %q", results[0].Status, spec.StatusDelivering)
	}
}

func TestTagFiltering(t *testing.T) {
	idx, _ := setupTestDB(t)

	s1 := makeSpec("tagged", "Tagged Spec", spec.TypeFeature, spec.StatusPlanning)
	s1.Tags = []string{"auth", "security"}

	s2 := makeSpec("untagged", "Untagged Spec", spec.TypeFeature, spec.StatusPlanning)

	for _, s := range []*spec.Spec{s1, s2} {
		if err := idx.IndexSpec(s, "# "+s.Title); err != nil {
			t.Fatalf("IndexSpec failed: %v", err)
		}
	}

	results, err := idx.ListFiltered("", "", "auth", "")
	if err != nil {
		t.Fatalf("ListFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("ListFiltered(tag=auth) returned %d results, want 1", len(results))
	}
	if results[0].Slug != "tagged" {
		t.Errorf("Slug = %q, want %q", results[0].Slug, "tagged")
	}
}

func TestRebuild(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")

	// Create spec directories
	specDir := filepath.Join(heroDir, "specs", "my-feature")
	planDir := filepath.Join(heroDir, "planning", "features", "new-feature")
	for _, d := range []string{specDir, planDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	}

	// Write specs
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("# My Feature\n\n## Changes\n- Update `src/main.go`\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(planDir, "spec.md"), []byte("---\ntype: feature\nstatus: planning\n---\n# New Feature\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	stats, err := Rebuild(heroDir)
	if err != nil {
		t.Fatalf("Rebuild failed: %v", err)
	}

	if stats.TotalSpecs != 2 {
		t.Errorf("TotalSpecs = %d, want 2", stats.TotalSpecs)
	}
	if stats.Features != 2 {
		t.Errorf("Features = %d, want 2", stats.Features)
	}
	if stats.Planning != 1 {
		t.Errorf("Planning = %d, want 1", stats.Planning)
	}
	if stats.Completed != 1 {
		t.Errorf("Completed = %d, want 1", stats.Completed)
	}
}

func TestLooksLikeTrackerID(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{"MORPH-123", true},
		{"PROJ-42", true},
		{"key-1", true},
		{"#42", true},
		{"42", true},
		{"123", true},
		{"", false},
		{"some random text", false},
		{"CSV export", false},
		{"fix-login-timeout", false},
		{"MORPH", false},
	}
	for _, tt := range tests {
		got := looksLikeTrackerID(tt.query)
		if got != tt.want {
			t.Errorf("looksLikeTrackerID(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}

func TestSearchByTrackerID(t *testing.T) {
	idx, _ := setupTestDB(t)

	s := makeSpec("morph-123-fix-login", "Fix Login Timeout", spec.TypeBug, spec.StatusPlanning)
	s.TrackerID = "MORPH-123"
	if err := idx.IndexSpec(s, "# Fix Login Timeout"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	s2 := makeSpec("other-spec", "Other Spec", spec.TypeFeature, spec.StatusPlanning)
	if err := idx.IndexSpec(s2, "# Other Spec"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Search by exact tracker ID should find it
	results, err := idx.Search("MORPH-123")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search('MORPH-123') returned %d results, want 1", len(results))
	}
	if results[0].Slug != "morph-123-fix-login" {
		t.Errorf("Slug = %q, want %q", results[0].Slug, "morph-123-fix-login")
	}

	// Case-insensitive match
	results, err = idx.Search("morph-123")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search('morph-123') returned %d results, want 1", len(results))
	}

	// Non-matching tracker ID falls back to FTS
	results, err = idx.Search("MORPH-999")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search('MORPH-999') returned %d results, want 0", len(results))
	}
}

func TestTripwireTriggerIndex(t *testing.T) {
	idx, heroDir := setupTestDB(t)

	// Create a tripwire spec with triggers
	tw := makeSpec("no-pyo3", "Do Not Use PyO3", spec.TypeTripwire, spec.StatusActive)
	tw.Triggers = []string{"pyo3", "python bindings", "python wrapper"}
	tw.Scope = []string{"*.rs", "*.py"}
	tw.Severity = "critical"
	tw.Sections["constraint"] = "Do not propose or implement PyO3."
	tw.Sections["why"] = "This project exists to replace the Python wrapper."
	tw.Sections["instead"] = "Use MLX C++ kernels directly via Rust FFI."

	// Write a temp spec file so FindAllTripwires can read it
	specDir := filepath.Join(heroDir, "knowledge", "tripwires", "no-pyo3")
	os.MkdirAll(specDir, 0o755)
	specContent := `---
title: Do Not Use PyO3
type: tripwire
status: active
triggers: [pyo3, "python bindings", "python wrapper"]
scope: ["*.rs", "*.py"]
severity: critical
---
# Do Not Use PyO3

## Constraint

Do not propose or implement PyO3.

## Why

This project exists to replace the Python wrapper.

## Instead

Use MLX C++ kernels directly via Rust FFI.
`
	os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(specContent), 0o644)
	tw.Path = filepath.Join(specDir, "spec.md")

	if err := idx.IndexSpec(tw, specContent); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	// Test FindTripwiresByTrigger — exact match
	matched, err := idx.FindTripwiresByTrigger("should we use pyo3")
	if err != nil {
		t.Fatalf("FindTripwiresByTrigger failed: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("FindTripwiresByTrigger('pyo3') = %d results, want 1", len(matched))
	}
	if matched[0].Slug != "no-pyo3" {
		t.Errorf("matched slug = %q, want %q", matched[0].Slug, "no-pyo3")
	}
	if matched[0].Severity != "critical" {
		t.Errorf("severity = %q, want %q", matched[0].Severity, "critical")
	}

	// Test FindTripwiresByTrigger — multi-word trigger
	matched, err = idx.FindTripwiresByTrigger("what about python bindings")
	if err != nil {
		t.Fatalf("FindTripwiresByTrigger failed: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("FindTripwiresByTrigger('python bindings') = %d results, want 1", len(matched))
	}

	// Test FindTripwiresByTrigger — no match
	matched, err = idx.FindTripwiresByTrigger("should we use rust")
	if err != nil {
		t.Fatalf("FindTripwiresByTrigger failed: %v", err)
	}
	if len(matched) != 0 {
		t.Errorf("FindTripwiresByTrigger('rust') = %d results, want 0", len(matched))
	}

	// Test FindTripwiresForFiles — scope match
	scopeMatched, err := idx.FindTripwiresForFiles([]string{"main.rs"})
	if err != nil {
		t.Fatalf("FindTripwiresForFiles failed: %v", err)
	}
	if len(scopeMatched) != 1 {
		t.Fatalf("FindTripwiresForFiles('main.rs') = %d results, want 1", len(scopeMatched))
	}

	// Test FindTripwiresForFiles — no scope match
	scopeMatched, err = idx.FindTripwiresForFiles([]string{"main.go"})
	if err != nil {
		t.Fatalf("FindTripwiresForFiles failed: %v", err)
	}
	if len(scopeMatched) != 0 {
		t.Errorf("FindTripwiresForFiles('main.go') = %d results, want 0", len(scopeMatched))
	}

	// Test FindAllTripwires
	all, err := idx.FindAllTripwires(heroDir)
	if err != nil {
		t.Fatalf("FindAllTripwires failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("FindAllTripwires = %d results, want 1", len(all))
	}
	if all[0].Constraint != "Do not propose or implement PyO3." {
		t.Errorf("Constraint = %q, want %q", all[0].Constraint, "Do not propose or implement PyO3.")
	}
	if all[0].Instead != "Use MLX C++ kernels directly via Rust FFI." {
		t.Errorf("Instead = %q, want %q", all[0].Instead, "Use MLX C++ kernels directly via Rust FFI.")
	}
}

func TestTripwireInBuildContext(t *testing.T) {
	idx, _ := setupTestDB(t)

	// Create a tripwire with wildcard scope
	tw := makeSpec("no-pyo3", "Do Not Use PyO3", spec.TypeTripwire, spec.StatusActive)
	tw.Scope = []string{"*"}
	if err := idx.IndexSpec(tw, "tripwire content"); err != nil {
		t.Fatalf("IndexSpec failed: %v", err)
	}

	ctx, err := idx.BuildContext([]string{"src/main.rs"})
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(ctx.Tripwires) != 1 {
		t.Errorf("BuildContext Tripwires = %d, want 1", len(ctx.Tripwires))
	}
}

// TestConcurrentWrite_WaitsForBusyTimeout is the Trigger-B1 regression: a
// second connection holds the write lock, and a write on the primary index
// must WAIT for the lock to release (via the busy-timeout) and then succeed —
// not fail instantly with "database is locked". Before the busy_timeout + WAL
// fix (Open opened SQLite with no connection params) this returned
// SQLITE_BUSY immediately, which is the degraded tooling the second failing
// session observed under concurrent `hero next` hook processes.
func TestConcurrentWrite_WaitsForBusyTimeout(t *testing.T) {
	idx, heroDir := setupTestDB(t)
	dbPath := filepath.Join(heroDir, IndexFileName)

	// A second connection grabs and holds the write lock, simulating a
	// concurrent `hero next ingest`/`checkpoint` hook process or a second
	// daemon mid-write.
	locker, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open locker connection: %v", err)
	}
	defer locker.Close()

	ctx := context.Background()
	conn, err := locker.Conn(ctx)
	if err != nil {
		t.Fatalf("pin locker connection: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("locker acquire write lock: %v", err)
	}

	// Release the lock after a bounded hold. The primary write must block
	// until then and succeed, proving the busy-timeout engaged.
	const hold = 300 * time.Millisecond
	go func() {
		time.Sleep(hold)
		_, _ = conn.ExecContext(ctx, "COMMIT")
	}()

	s := makeSpec("concurrency-probe", "Concurrency Probe", spec.TypeFeature, spec.StatusPlanning)
	start := time.Now()
	err = idx.IndexSpec(s, "body content for the concurrency probe")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("IndexSpec under write contention failed — busy_timeout/WAL not applied: %v", err)
	}
	// It must have WAITED for the lock (proving contention was real and the
	// timeout absorbed it), not returned instantly.
	if elapsed < hold/2 {
		t.Fatalf("write returned in %v — did not wait for the held lock; the busy-timeout was not exercised", elapsed)
	}
	// And it must have succeeded well within the 5s busy-timeout budget.
	if elapsed > 5*time.Second {
		t.Fatalf("write took %v — exceeded the busy-timeout window", elapsed)
	}
}

// TestOpen_UsesWALJournalMode verifies the index opens in WAL mode (matching
// the graph's concurrency posture), so readers proceed while a writer holds
// the DB. This is the persistent half of the Trigger-B1 fix.
func TestOpen_UsesWALJournalMode(t *testing.T) {
	idx, _ := setupTestDB(t)

	var mode string
	if err := idx.RawDB().QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want \"wal\"", mode)
	}
}
