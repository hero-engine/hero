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

// TestLoadPlate_GitDerivedUserMatchesClaim is the regression guard
// for the dashboard-user-identity-os-env-mismatch bug: a spec claimed
// via the writer-side identity (e.g. `chet-bellows` derived from
// `git config user.name`) MUST surface on the plate when the reader
// also uses that identity, even when $USER points elsewhere.
func TestLoadPlate_GitDerivedUserMatchesClaim(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	specDir := filepath.Join(heroDir, "planning", "features", "ledger-fix")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const body = `---
title: "Ledger Fix"
type: feature
status: delivering
claimed_by: chet-bellows
---

## Context

Writer side wrote claimed_by under the git identity.

## Acceptance Criteria

- THE SYSTEM SHALL surface this on the plate
`
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pass the git-derived identity (NOT $USER) — this is exactly
	// what serve/server.go passes after the fix.
	got := LoadPlate(PlateInputs{
		HeroDir:  heroDir,
		UserName: "chet-bellows",
	})
	if got.Primary == nil {
		t.Fatalf("expected primary card for git-identity claim, got nil (Total=%d)", got.Total)
	}
	if got.Primary.Slug != "ledger-fix" {
		t.Errorf("Primary.Slug = %q, want ledger-fix", got.Primary.Slug)
	}
}

// TestLoadPlate_OsUserDoesNotMatchGitClaim documents the pre-fix
// failure mode: when the reader passes the OS login but the writer
// claimed under the git identity, the plate must NOT match. Locking
// this in so a future "lenient match" doesn't accidentally cross
// namespaces.
func TestLoadPlate_OsUserDoesNotMatchGitClaim(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	specDir := filepath.Join(heroDir, "planning", "features", "ledger-fix")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const body = `---
title: "Ledger Fix"
type: feature
status: delivering
claimed_by: chet-bellows
---

## Context

Writer claimed under git identity.

## Acceptance Criteria

- THE SYSTEM SHALL not match the wrong identity
`
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadPlate(PlateInputs{
		HeroDir:  heroDir,
		UserName: "bwheeler", // OS login, not git identity
	})
	if got.Primary != nil {
		t.Errorf("expected no match for $USER vs git-identity claim; got %+v", got.Primary)
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
