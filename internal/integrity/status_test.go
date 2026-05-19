package integrity

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
)

func TestVerify_LyingSpec(t *testing.T) {
	store := openStore(t)
	seedAC(t, store, "feat-x:AC-1", "First", "passing")
	seedAC(t, store, "feat-x:AC-2", "Second", "failing")
	s := completedSpec("feat-x")

	report, err := CheckCompletedSpecs([]*spec.Spec{s}, store)
	if err != nil {
		t.Fatalf("CheckCompletedSpecs: %v", err)
	}
	if report.Lying != 1 {
		t.Errorf("Lying = %d, want 1", report.Lying)
	}
	if !report.HasIssues() {
		t.Error("HasIssues = false, want true")
	}
	if got := report.Findings[0].Verdict; got != VerdictLying {
		t.Errorf("Verdict = %s, want lying", got)
	}
}

func TestVerify_PartialSpec(t *testing.T) {
	store := openStore(t)
	seedAC(t, store, "feat-x:AC-1", "First", "passing")
	seedAC(t, store, "feat-x:AC-2", "Second", "proposed")
	s := completedSpec("feat-x")

	report, err := CheckCompletedSpecs([]*spec.Spec{s}, store)
	if err != nil {
		t.Fatalf("CheckCompletedSpecs: %v", err)
	}
	if report.Partial != 1 {
		t.Errorf("Partial = %d, want 1", report.Partial)
	}
	if got := report.Findings[0].Verdict; got != VerdictPartial {
		t.Errorf("Verdict = %s, want partial", got)
	}
}

func TestVerify_VerifiedSpec(t *testing.T) {
	store := openStore(t)
	seedAC(t, store, "feat-x:AC-1", "First", "passing")
	seedAC(t, store, "feat-x:AC-2", "Second", "passing")
	s := completedSpec("feat-x")

	report, err := CheckCompletedSpecs([]*spec.Spec{s}, store)
	if err != nil {
		t.Fatalf("CheckCompletedSpecs: %v", err)
	}
	if report.Verified != 1 || report.HasIssues() {
		t.Errorf("Verified = %d, HasIssues = %v; want Verified=1 no issues",
			report.Verified, report.HasIssues())
	}
}

func TestVerify_UnverifiableNoACs(t *testing.T) {
	store := openStore(t)
	s := completedSpec("feat-x") // no Criterion seeded

	report, err := CheckCompletedSpecs([]*spec.Spec{s}, store)
	if err != nil {
		t.Fatalf("CheckCompletedSpecs: %v", err)
	}
	if report.Unverifiable != 1 || report.HasIssues() {
		t.Errorf("Unverifiable = %d, HasIssues = %v; want Unverifiable=1 no issues",
			report.Unverifiable, report.HasIssues())
	}
}

func TestVerify_NonCompletedSkipped(t *testing.T) {
	store := openStore(t)
	s := &spec.Spec{
		Slug:   "feat-y",
		Title:  "Y",
		Status: spec.StatusPlanning, // not completed → ignored
		Type:   spec.TypeFeature,
		Path:   "/x.md",
	}
	report, err := CheckCompletedSpecs([]*spec.Spec{s}, store)
	if err != nil {
		t.Fatalf("CheckCompletedSpecs: %v", err)
	}
	if report.Total() != 0 {
		t.Errorf("Total = %d, want 0", report.Total())
	}
}

func TestVerify_LyingSortsFirst(t *testing.T) {
	store := openStore(t)
	seedAC(t, store, "feat-a:AC-1", "x", "passing")
	seedAC(t, store, "feat-b:AC-1", "x", "failing")
	seedAC(t, store, "feat-c:AC-1", "x", "proposed")
	specs := []*spec.Spec{completedSpec("feat-a"), completedSpec("feat-b"), completedSpec("feat-c")}

	report, err := CheckCompletedSpecs(specs, store)
	if err != nil {
		t.Fatalf("CheckCompletedSpecs: %v", err)
	}
	if got := report.Findings[0].Slug; got != "feat-b" {
		t.Errorf("first finding slug = %q, want feat-b (lying first)", got)
	}
}

// --- helpers ---

func openStore(t *testing.T) *graph.Store {
	t.Helper()
	store, err := graph.Open(filepath.Join(t.TempDir()))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func completedSpec(slug string) *spec.Spec {
	return &spec.Spec{
		Slug:   slug,
		Title:  slug,
		Status: spec.StatusCompleted,
		Type:   spec.TypeFeature,
		Path:   "/" + slug + ".md",
	}
}

func seedAC(t *testing.T, store *graph.Store, key, statement, status string) {
	t.Helper()
	parent := key
	if i := indexOfColon(key); i > 0 {
		parent = key[:i]
	}
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Criterion",
		Domain:      "engineering",
		Key:  key,
		Props: map[string]any{
			"ac_id":     key,
			"statement": statement,
			"status":    status,
			"parent":    parent,
		},
		Repo:        "repo-x",
		ContentHash: key + "|" + statement + "|" + status,
		IngestedAt:  time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed AC: %v", err)
	}
}

func indexOfColon(s string) int {
	for i, r := range s {
		if r == ':' {
			return i
		}
	}
	return -1
}
