package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlate_EmptyWorkspace(t *testing.T) {
	got := LoadPlate(PlateInputs{})
	if got.Primary != nil || got.Secondary != nil {
		t.Errorf("expected nil cards on empty input, got %+v / %+v", got.Primary, got.Secondary)
	}
	if got.Total != 0 {
		t.Errorf("Total = %d, want 0", got.Total)
	}
}

func TestLoadPlate_FindsClaimedSpec(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	specDir := filepath.Join(heroDir, "planning", "features", "demo-spec")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const body = `---
title: "Demo Spec"
type: feature
status: delivering
claimed_by: ben
---

## Context

This is a demo spec for the test.

## Acceptance Criteria

- THE SYSTEM SHALL demo this
`
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadPlate(PlateInputs{
		HeroDir:  heroDir,
		UserName: "ben",
	})
	if got.Primary == nil {
		t.Fatalf("expected primary card, got nil (Total=%d)", got.Total)
	}
	if got.Primary.Slug != "demo-spec" {
		t.Errorf("Primary.Slug = %q, want demo-spec", got.Primary.Slug)
	}
	if got.Primary.StatusLabel != "Delivering" {
		t.Errorf("StatusLabel = %q, want Delivering", got.Primary.StatusLabel)
	}
}
