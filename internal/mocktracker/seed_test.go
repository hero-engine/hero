package mocktracker

import (
	"context"
	"testing"
	"testing/fstest"

	sprout "github.com/bdwheeler/sprout/go"
)

// AC-10: clean apply of the embedded default seed.
func TestSeed_CleanApply(t *testing.T) {
	ctx := context.Background()
	st, err := NewStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if _, err := st.Seed(ctx, "", false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	issues, err := st.ListIssues(ctx, IssueFilter{State: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 6 {
		t.Fatalf("got %d issues, want 6", len(issues))
	}
	epics, _ := st.ListEpics(ctx)
	if len(epics) != 2 {
		t.Errorf("got %d epics, want 2", len(epics))
	}
	miles, _ := st.ListMilestones(ctx)
	if len(miles) != 1 {
		t.Errorf("got %d milestones, want 1", len(miles))
	}
	iters, _ := st.ListIterations(ctx)
	if len(iters) != 1 {
		t.Errorf("got %d iterations, want 1", len(iters))
	}
	// labels + iid hydrated
	one, err := st.GetIssueByGlobalID(ctx, "ACME-101")
	if err != nil || one == nil {
		t.Fatalf("ACME-101 lookup: %v", err)
	}
	if one.IID != "101" {
		t.Errorf("IID = %q, want 101", one.IID)
	}
	if len(one.Labels) != 2 {
		t.Errorf("ACME-101 labels = %v, want 2", one.Labels)
	}
	if !one.Weight.Valid || one.Weight.Int64 != 3 {
		t.Errorf("ACME-101 weight = %v, want 3", one.Weight)
	}
}

// AC-10: re-applying is idempotent (sprout checksum-skip — every file
// Skipped on the second pass).
func TestSeed_IdempotentReapply(t *testing.T) {
	ctx := context.Background()
	st, _ := NewStore(ctx)
	defer st.Close()

	if _, err := st.Seed(ctx, "", false); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	res, err := st.Seed(ctx, "", false)
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	for _, f := range res.Files {
		if !f.Skipped {
			t.Errorf("file %q was not checksum-skipped on re-apply", f.Name)
		}
	}
	// row count unchanged
	issues, _ := st.ListIssues(ctx, IssueFilter{State: "all"})
	if len(issues) != 6 {
		t.Errorf("after re-apply got %d issues, want 6", len(issues))
	}
}

// AC-10: a malformed seed surfaces sprout's validation/parse error.
func TestSeed_InvalidSeed(t *testing.T) {
	ctx := context.Background()
	st, _ := NewStore(ctx)
	defer st.Close()

	bad := fstest.MapFS{
		"seed/seeds.list": {Data: []byte("V01_bad.yaml\n")},
		// `seed` must be a mapping; a scalar is a hard parse error.
		"seed/V01_bad.yaml": {Data: []byte("seed: not-a-mapping\n")},
	}
	_, err := st.seedFromFS(ctx, bad, "seed", false)
	if err == nil {
		t.Fatal("expected an error from a malformed seed")
	}
}

// guard that sprout's force re-apply works (mid-run reset path).
func TestSeed_ForceReapply(t *testing.T) {
	ctx := context.Background()
	st, _ := NewStore(ctx)
	defer st.Close()
	if _, err := st.Seed(ctx, "", false); err != nil {
		t.Fatal(err)
	}
	res, err := st.Seed(ctx, "", true) // force
	if err != nil {
		t.Fatalf("force seed: %v", err)
	}
	skippedAny := false
	for _, f := range res.Files {
		if f.Skipped {
			skippedAny = true
		}
	}
	if skippedAny {
		t.Error("force should bypass checksum-skip, but some files were skipped")
	}
	var _ sprout.Results = res
}
