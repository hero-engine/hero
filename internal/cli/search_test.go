package cli

import (
	"strings"
	"testing"
)

func TestSearchByQuery(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export Feature
type: feature
status: planning
tags: [export, csv]
---
# CSV Export Feature

## Context
Users need to export their data in CSV format.

## Changes
- Add `+"`src/export/csv.go`"+`
`)

	env.indexAll()

	output, err := runCmd("search", "CSV export")
	if err != nil {
		t.Fatalf("search returned error: %v", err)
	}

	if !strings.Contains(output, "csv-export") {
		t.Errorf("search should find csv-export: %q", output)
	}

	if !strings.Contains(output, "1 result") {
		t.Errorf("search should show 1 result: %q", output)
	}
}

func TestSearchNoResults(t *testing.T) {
	env := newTestEnv(t)

	// Add a spec so FTS table is non-empty, then search for something not in it
	env.addSpec("planning/features/unrelated/spec.md", `---
title: Something Unrelated
type: feature
status: planning
---
# Something Unrelated
`)

	env.indexAll()

	output, err := runCmd("search", "zzzyyyxxx")
	if err != nil {
		t.Fatalf("search returned error: %v", err)
	}

	if !strings.Contains(output, "No results found") {
		t.Errorf("search should show no results: %q", output)
	}
}

func TestSearchByFile(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/auth-refactor/spec.md", `---
title: Auth Refactor
type: feature
status: delivering
---
# Auth Refactor

## Changes
- Update `+"`src/auth/session.go`"+`
- Update `+"`src/auth/token.go`"+`
`)

	env.indexAll()

	output, err := runCmd("search", "--file", "src/auth/session.go")
	if err != nil {
		t.Fatalf("search --file returned error: %v", err)
	}

	if !strings.Contains(output, "auth-refactor") {
		t.Errorf("search --file should find auth-refactor: %q", output)
	}
}

func TestSearchListOnly(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-a/spec.md", `---
title: Feature A
type: feature
status: planning
---
# Feature A
`)

	env.addSpec("planning/bugs/bug-a/spec.md", `---
title: Bug A
type: bug
status: planning
---
# Bug A
`)

	env.indexAll()

	// List all
	output, err := runCmd("search", "--list")
	if err != nil {
		t.Fatalf("search --list returned error: %v", err)
	}

	if !strings.Contains(output, "feat-a") {
		t.Errorf("search --list should show feat-a: %q", output)
	}
	if !strings.Contains(output, "bug-a") {
		t.Errorf("search --list should show bug-a: %q", output)
	}
}

func TestSearchListWithTypeFilter(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-b/spec.md", `---
title: Feature B
type: feature
status: planning
---
# Feature B
`)

	env.addSpec("planning/bugs/bug-b/spec.md", `---
title: Bug B
type: bug
status: planning
---
# Bug B
`)

	env.indexAll()

	// List only bugs
	output, err := runCmd("search", "--list", "--type", "bug")
	if err != nil {
		t.Fatalf("search --list --type bug returned error: %v", err)
	}

	if strings.Contains(output, "feat-b") {
		t.Errorf("search --list --type bug should not show feat-b: %q", output)
	}
	if !strings.Contains(output, "bug-b") {
		t.Errorf("search --list --type bug should show bug-b: %q", output)
	}
}

func TestSearchNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("search", "test")
	if err == nil {
		t.Fatal("search should error without workspace")
	}
}

func TestSearchRequiresQueryOrList(t *testing.T) {
	_ = newTestEnv(t)

	// No query and no --list should fail
	_, err := runCmd("search")
	if err == nil {
		t.Fatal("search without query or --list should fail")
	}
}
