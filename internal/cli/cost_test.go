package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestCostCmd_NoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("sprint", "estimate", "some-spec")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestCostCmd_SpecNotFound(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("sprint", "estimate", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing spec")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestCostCmd_SingleSpec(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/auth-login/spec.md", `---
title: Auth Login
type: feature
status: delivering
---

## Goal
Build login flow with OAuth2 integration for the main application.

## Design
We need OAuth2 with PKCE flow, session management, and token refresh.

## Changes
- internal/auth/login.go
- internal/auth/session.go
- internal/auth/oauth.go
- internal/auth/middleware.go
`)

	out, err := runCmd("sprint", "estimate", "auth-login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "auth-login") {
		t.Error("expected spec slug in output")
	}
	if !strings.Contains(out, "Auth Login") {
		t.Error("expected title in output")
	}
	if !strings.Contains(out, "Effort:") {
		t.Error("expected effort estimate in output")
	}
	if !strings.Contains(out, "Files in Changes:  4") {
		t.Error("expected 4 files count")
	}
	if !strings.Contains(out, "Signals:") {
		t.Error("expected signals section")
	}
}

func TestCostCmd_NoInFlight(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/done-thing/spec.md", `---
title: Done Thing
type: feature
status: completed
---

## Goal
Already done.
`)

	out, err := runCmd("sprint", "estimate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No in-flight specs") {
		t.Errorf("expected no in-flight message, got: %s", out)
	}
}

func TestCostCmd_MultipleInFlight(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/small-bug/spec.md", `---
title: Small Bug Fix
type: bug
status: delivering
---

## Goal
Fix the crash.

## Changes
- internal/foo.go
`)

	env.addSpec("specs/big-feature/spec.md", `---
title: Big Feature
type: feature
status: planning
---

## Goal
Build the entire new subsystem.

## Design
Lots of design here. We need multiple components and integrations. This will touch
many files and require careful coordination.

## Changes
- internal/sub/a.go
- internal/sub/b.go
- internal/sub/c.go
- internal/sub/d.go
- internal/sub/e.go
- internal/sub/f.go
- internal/sub/g.go
- internal/sub/h.go
`)

	out, err := runCmd("sprint", "estimate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "EFFORT") {
		t.Error("expected table header")
	}
	if !strings.Contains(out, "TOTAL") {
		t.Error("expected total row")
	}
	if !strings.Contains(out, "2 specs") {
		t.Error("expected 2 specs in total")
	}
}

func TestCostCmd_AllFlag(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/completed-one/spec.md", `---
title: Completed One
type: feature
status: completed
---

## Goal
Done.
`)
	env.addSpec("specs/active-one/spec.md", `---
title: Active One
type: feature
status: delivering
---

## Goal
In progress.
`)

	out, err := runCmd("sprint", "estimate", "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "2 specs") {
		t.Errorf("expected 2 specs with --all, got: %s", out)
	}
}

func TestCostCmd_WithCalibration(t *testing.T) {
	env := newTestEnv(t)

	// Add completed specs for calibration
	env.addSpec("specs/past-work-a/spec.md", `---
title: Past Work A
type: feature
status: completed
---

## Goal
Done previously.

## Changes
- internal/a1.go
- internal/a2.go
`)
	env.addSpec("specs/past-work-b/spec.md", `---
title: Past Work B
type: feature
status: completed
---

## Goal
Also done.

## Changes
- internal/b1.go
- internal/b2.go
- internal/b3.go
- internal/b4.go
`)

	// Now estimate a new spec
	env.addSpec("specs/new-work/spec.md", `---
title: New Work
type: feature
status: planning
---

## Goal
Something new.

## Changes
- internal/new1.go
- internal/new2.go
- internal/new3.go
`)

	out, err := runCmd("sprint", "estimate", "new-work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Calibration") {
		t.Error("expected calibration section in output")
	}
	if !strings.Contains(out, "2 completed specs") {
		t.Error("expected reference to 2 completed specs")
	}
}

func TestEstimateSpec_BugDiscount(t *testing.T) {
	bugSpec := &spec.Spec{
		Slug:         "test-bug",
		Title:        "Test Bug",
		Type:         spec.TypeBug,
		Status:       spec.StatusDelivering,
		FilesTouched: []string{"a.go", "b.go"},
		Sections:     map[string]string{"goal": "Fix it.", "changes": "- a.go\n- b.go"},
	}
	featureSpec := &spec.Spec{
		Slug:         "test-feature",
		Title:        "Test Feature",
		Type:         spec.TypeFeature,
		Status:       spec.StatusDelivering,
		FilesTouched: []string{"a.go", "b.go"},
		Sections:     map[string]string{"goal": "Build it.", "changes": "- a.go\n- b.go"},
	}

	cal := calibrationData{}
	bugEst := estimateSpec(bugSpec, cal)
	featureEst := estimateSpec(featureSpec, cal)

	if bugEst.Points >= featureEst.Points {
		t.Errorf("bug (%.1f) should have fewer points than feature (%.1f) with same files", bugEst.Points, featureEst.Points)
	}
}

func TestBucketFromPoints(t *testing.T) {
	tests := []struct {
		points float64
		bucket string
	}{
		{0.5, effortTrivial},
		{1.9, effortTrivial},
		{2.0, effortSmall},
		{4.9, effortSmall},
		{5.0, effortMedium},
		{9.9, effortMedium},
		{10.0, effortLarge},
		{19.9, effortLarge},
		{20.0, effortXLarge},
		{100.0, effortXLarge},
	}

	for _, tt := range tests {
		result := bucketFromPoints(tt.points)
		if result != tt.bucket {
			t.Errorf("bucketFromPoints(%.1f) = %q, want %q", tt.points, result, tt.bucket)
		}
	}
}

func TestCountWords(t *testing.T) {
	if n := countWords("hello world"); n != 2 {
		t.Errorf("countWords('hello world') = %d, want 2", n)
	}
	if n := countWords(""); n != 0 {
		t.Errorf("countWords('') = %d, want 0", n)
	}
	if n := countWords("one two three four five"); n != 5 {
		t.Errorf("countWords('one two three four five') = %d, want 5", n)
	}
}
