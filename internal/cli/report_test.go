package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

func TestReportCmd_Empty(t *testing.T) {
	_ = newTestEnv(t)

	out, err := runCmd("sprint", "report")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Report generated") {
		t.Errorf("expected report generated message, got: %s", out)
	}
}

func TestReportCmd_GeneratesHTML(t *testing.T) {
	env := newTestEnv(t)

	// Add some specs
	env.addSpec("planning/features/auth/spec.md", `---
title: Auth System
type: feature
status: planning
tags: [auth]
---

## Goal
Build auth.

## Changes
- internal/auth/login.go
`)
	env.addSpec("specs/api/spec.md", `---
title: API Layer
type: feature
status: completed
tags: [api]
---

## Goal
Build API.
`)
	env.addSpec("knowledge/conventions/naming/spec.md", `---
title: Naming Convention
type: convention
status: active
---

## Convention
Use camelCase.
`)

	out, err := runCmd("sprint", "report")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Report generated") {
		t.Errorf("expected report generated message, got: %s", out)
	}

	// Check the HTML was created
	reportPath := filepath.Join(env.heroDir, "reports", "report.html")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}
	html := string(data)

	if !strings.Contains(html, "Hero Report") {
		t.Error("report should contain 'Hero Report'")
	}
	if !strings.Contains(html, "Total Specs") {
		t.Error("report should contain 'Total Specs'")
	}
	if !strings.Contains(html, "auth") {
		t.Error("report should mention auth spec")
	}
}

func TestReportCmd_CustomOutput(t *testing.T) {
	env := newTestEnv(t)

	outPath := filepath.Join(env.dir, "my-report.html")
	out, err := runCmd("sprint", "report", "--output", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, outPath) {
		t.Errorf("expected output path in message, got: %s", out)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Error("report file was not created at custom path")
	}
}

func TestReportCmd_NoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("sprint", "report")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestReportCmd_CoverageData(t *testing.T) {
	env := newTestEnv(t)

	// Add specs with varying coverage attributes
	env.addSpec("planning/features/claimed-spec/spec.md", `---
title: Claimed Spec
type: feature
status: delivering
claimed_by: alice
tracker_id: GH-123
---

## Changes
- src/foo.go
`)
	env.addSpec("planning/features/bare-spec/spec.md", `---
title: Bare Spec
type: feature
status: planning
---

## Goal
Minimal spec.
`)

	out, err := runCmd("sprint", "report")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Report generated") {
		t.Errorf("expected report generated, got: %s", out)
	}

	reportPath := filepath.Join(env.heroDir, "reports", "report.html")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}
	html := string(data)

	// Should contain coverage section
	if !strings.Contains(html, "Coverage") {
		t.Error("report should contain Coverage section")
	}
	if !strings.Contains(html, "Tracker linked") {
		t.Error("report should contain tracker linked coverage")
	}
}

func TestBuildReportData(t *testing.T) {
	// Unit test for the data builder
	specs := []*spec.Spec{
		{Slug: "feat-1", Title: "Feature 1", Type: spec.TypeFeature, Status: spec.StatusPlanning},
		{Slug: "feat-2", Title: "Feature 2", Type: spec.TypeFeature, Status: spec.StatusCompleted, ClaimedBy: "bob"},
		{Slug: "conv-1", Title: "Convention 1", Type: spec.TypeConvention, Status: spec.StatusActive},
	}

	cfg := config.DefaultConfig()
	data := buildReportData(specs, cfg, "/tmp/test-project")

	if data.TotalSpecs != 3 {
		t.Errorf("expected 3 total specs, got %d", data.TotalSpecs)
	}
	if data.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", data.Completed)
	}
	if data.InFlight != 1 {
		t.Errorf("expected 1 in-flight, got %d", data.InFlight)
	}
	if data.ProjectName != "test-project" {
		t.Errorf("expected project name 'test-project', got %s", data.ProjectName)
	}
	if len(data.ClaimedSpecs) != 1 {
		t.Errorf("expected 1 claimed spec, got %d", len(data.ClaimedSpecs))
	}
}
