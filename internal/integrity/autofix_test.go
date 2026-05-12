package integrity

import (
	"strings"
	"testing"
)

const sampleSpec = `---
title: Sample
type: feature
status: completed
priority: P0
tags: [test]
---

## Goal

Ship the thing.
`

func TestRewriteFrontmatterStatus_DowngradesAndAnnotates(t *testing.T) {
	out, changed, err := rewriteFrontmatterStatus([]byte(sampleSpec), "planning", "3 of 5 ACs failing")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	body := string(out)
	if !strings.Contains(body, "status: planning") {
		t.Errorf("status not rewritten:\n%s", body)
	}
	if strings.Contains(body, "status: completed") {
		t.Errorf("old status still present:\n%s", body)
	}
	if !strings.Contains(body, "auto_downgraded:") {
		t.Errorf("annotation missing:\n%s", body)
	}
	if !strings.Contains(body, "3 of 5 ACs failing") {
		t.Errorf("reason missing from annotation:\n%s", body)
	}
}

func TestRewriteFrontmatterStatus_IdempotentOnReRun(t *testing.T) {
	first, _, err := rewriteFrontmatterStatus([]byte(sampleSpec), "planning", "reason")
	if err != nil {
		t.Fatal(err)
	}
	second, changed, err := rewriteFrontmatterStatus(first, "planning", "another reason")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Errorf("second run reported changed=true; want idempotent no-op")
	}
	if !bytesEqual(first, second) {
		t.Errorf("re-run modified bytes:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestRewriteFrontmatterStatus_ReplacesPriorAnnotation(t *testing.T) {
	withAnno := `---
title: Sample
status: planning
auto_downgraded: "2026-04-28 by hero check status: 2 of 7 failing"
priority: P0
---

## Goal

x
`
	out, changed, err := rewriteFrontmatterStatus([]byte(withAnno), "delivering", "5 of 7 passing now")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("changed = false")
	}
	body := string(out)
	count := strings.Count(body, "auto_downgraded:")
	if count != 1 {
		t.Errorf("auto_downgraded appears %d times, want 1:\n%s", count, body)
	}
	if !strings.Contains(body, "5 of 7 passing now") {
		t.Errorf("new reason missing:\n%s", body)
	}
	if strings.Contains(body, "2 of 7 failing") {
		t.Errorf("old reason survived:\n%s", body)
	}
}

func TestRewriteFrontmatterStatus_BailsWhenNoFrontmatter(t *testing.T) {
	plain := []byte("# No frontmatter here\n\nsome content\n")
	out, changed, err := rewriteFrontmatterStatus(plain, "planning", "x")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Errorf("changed = true on plain markdown, want false")
	}
	if !bytesEqual(out, plain) {
		t.Errorf("plain markdown was rewritten unexpectedly:\n%s", out)
	}
}

func TestRewriteFrontmatterStatus_ErrorsWhenNoStatusLine(t *testing.T) {
	noStatus := `---
title: Sample
type: feature
---

x
`
	_, _, err := rewriteFrontmatterStatus([]byte(noStatus), "planning", "x")
	if err == nil {
		t.Errorf("expected error for missing status line, got nil")
	}
}

func TestPlanFixes_LyingAndPartialBecomeProposals(t *testing.T) {
	report := &Report{
		Findings: []Finding{
			{Slug: "feat-a", Path: "/tmp/feat-a.md", Verdict: VerdictLying, Total: 5, Failing: 3, Passing: 2, FailingKeys: []string{"feat-a:AC-1"}},
			{Slug: "feat-b", Path: "/tmp/feat-b.md", Verdict: VerdictPartial, Total: 5, Passing: 3, ProposedOrOpen: 2},
			{Slug: "feat-c", Path: "/tmp/feat-c.md", Verdict: VerdictVerified, Total: 5, Passing: 5},
			{Slug: "feat-d", Path: "/tmp/feat-d.md", Verdict: VerdictUnverifiable},
		},
	}
	plan := PlanFixes(report)
	if len(plan) != 4 {
		t.Fatalf("plan len = %d, want 4", len(plan))
	}

	// Lying → planning.
	if plan[0].Skipped || plan[0].NewStatus != "planning" {
		t.Errorf("feat-a action = %+v", plan[0])
	}
	// Partial → delivering.
	if plan[1].Skipped || plan[1].NewStatus != "delivering" {
		t.Errorf("feat-b action = %+v", plan[1])
	}
	// Verified, unverifiable → skipped.
	if !plan[2].Skipped || !plan[3].Skipped {
		t.Errorf("verified/unverifiable should be skipped: %+v %+v", plan[2], plan[3])
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
