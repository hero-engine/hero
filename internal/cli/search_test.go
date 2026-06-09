package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/retrieval"
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

// TestSearchJSONFTS5Path asserts --json emits valid JSON on the FTS5 path
// (filter flags engage FTS5). Regression test for
// hero-search-json-flag-silently-ignored.
func TestSearchJSONFTS5Path(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/json-out/spec.md", `---
title: JSON Output Test
type: feature
status: planning
---
# JSON Output Test
`)

	env.indexAll()

	// --type forces the FTS5 routing path.
	output, err := runCmd("search", "JSON", "--type", "feature", "--json")
	if err != nil {
		t.Fatalf("search --json returned error: %v", err)
	}

	// Strip any non-JSON prefix lines (e.g. scope hints written to stderr
	// shouldn't appear, but the harness may capture stderr too).
	out := strings.TrimSpace(output)
	idx := strings.Index(out, "[")
	if idx < 0 {
		t.Fatalf("expected JSON array in output, got: %q", output)
	}
	out = out[idx:]

	var parsed []map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, output)
	}
	if len(parsed) == 0 {
		t.Fatalf("expected at least one result, got empty array: %q", output)
	}
	got := parsed[0]
	for _, k := range []string{"type", "key", "title", "status"} {
		if _, ok := got[k]; !ok {
			t.Errorf("result missing key %q: %+v", k, got)
		}
	}
	if got["key"] != "json-out" {
		t.Errorf("expected key=json-out, got %v", got["key"])
	}
}

// TestSearchJSONListMode asserts --list --json emits a JSON array on the
// runSearchFTS path.
func TestSearchJSONListMode(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/list-json-a/spec.md", `---
title: List JSON A
type: feature
status: planning
---
# List JSON A
`)
	env.addSpec("planning/bugs/list-json-b/spec.md", `---
title: List JSON B
type: bug
status: planning
---
# List JSON B
`)

	env.indexAll()

	output, err := runCmd("search", "--list", "--json")
	if err != nil {
		t.Fatalf("search --list --json returned error: %v", err)
	}

	out := strings.TrimSpace(output)
	idx := strings.Index(out, "[")
	if idx < 0 {
		t.Fatalf("expected JSON array, got: %q", output)
	}
	out = out[idx:]

	var parsed []map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, output)
	}
	if len(parsed) < 2 {
		t.Errorf("expected at least 2 results, got %d: %q", len(parsed), output)
	}
}

// TestSearchJSONFileMode asserts --file --json emits a JSON array on the
// runSearchFTS path.
func TestSearchJSONFileMode(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/file-json/spec.md", `---
title: File JSON Search
type: feature
status: delivering
---
# File JSON Search

## Changes
- Update `+"`src/foo/bar.go`"+`
`)

	env.indexAll()

	output, err := runCmd("search", "--file", "src/foo/bar.go", "--json")
	if err != nil {
		t.Fatalf("search --file --json returned error: %v", err)
	}

	out := strings.TrimSpace(output)
	idx := strings.Index(out, "[")
	if idx < 0 {
		t.Fatalf("expected JSON array, got: %q", output)
	}
	out = out[idx:]

	var parsed []map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, output)
	}
	if len(parsed) == 0 {
		t.Fatalf("expected at least 1 result, got empty array: %q", output)
	}
	if parsed[0]["key"] != "file-json" {
		t.Errorf("expected key=file-json, got %v", parsed[0]["key"])
	}
}

// TestSearchJSONNoResults asserts --json emits `[]` on the no-results path
// rather than the human "No results found." string.
func TestSearchJSONNoResults(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/unrelated2/spec.md", `---
title: Unrelated Two
type: feature
status: planning
---
# Unrelated Two
`)

	env.indexAll()

	output, err := runCmd("search", "zzzyyyxxx", "--json")
	if err != nil {
		t.Fatalf("search --json returned error: %v", err)
	}

	out := strings.TrimSpace(output)
	if out != "[]" {
		t.Errorf("expected `[]`, got: %q", output)
	}

	var parsed []any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Errorf("output should be valid JSON empty array: %v", err)
	}
}

// TestPrintGraphResults_RepoLabelAppearsForCrossRepoResult verifies
// SC-4 of unified-search: sibling-repo results get a [repo] label in
// printGraphResults output when the result's Repo differs from localRepo.
func TestPrintGraphResults_RepoLabelAppearsForCrossRepoResult(t *testing.T) {
	results := []retrieval.Result{
		{Type: "Feature", Key: "my-feature", Title: "My Feature", Score: 10, Source: "graph", Repo: "alice/sibling-repo"},
		{Type: "Bug", Key: "local-bug", Title: "Local Bug", Score: 9, Source: "graph", Repo: "owner/local-repo"},
		{Type: "Feature", Key: "no-repo", Title: "No Repo Feature", Score: 8, Source: "graph", Repo: ""},
	}

	out := captureStdout(func() {
		_ = printGraphResults(results, "test", 4000, false, "owner/local-repo")
	})

	// Cross-repo result should carry [alice/sibling-repo] label.
	if !strings.Contains(out, "[alice/sibling-repo]") {
		t.Errorf("expected cross-repo label in output; got:\n%s", out)
	}
	// Local result should NOT carry a label.
	if strings.Contains(out, "[owner/local-repo]") {
		t.Errorf("unexpected local repo label in output; got:\n%s", out)
	}
	// Result with empty Repo should also have no label.
	if strings.Contains(out, "[]") {
		t.Errorf("unexpected empty [] label in output; got:\n%s", out)
	}
}
