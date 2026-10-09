package serve

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

var updateReadContract = flag.Bool("update-read-contract", false, "rewrite testdata/read_contract_v1.golden (additions only)")

// schemaPaths flattens a JSON value into "path:type" entries. Arrays use
// "[]" and record their element shape from every element seen.
func schemaPaths(prefix string, v interface{}, out map[string]bool) {
	switch x := v.(type) {
	case map[string]interface{}:
		out[prefix+":object"] = true
		for k, val := range x {
			schemaPaths(prefix+"."+k, val, out)
		}
	case []interface{}:
		out[prefix+":array"] = true
		for _, el := range x {
			schemaPaths(prefix+"[]", el, out)
		}
	case string:
		out[prefix+":string"] = true
	case float64:
		out[prefix+":number"] = true
	case bool:
		out[prefix+":boolean"] = true
	case nil:
		out[prefix+":null"] = true
	}
}

func callAndRecord(t *testing.T, srv *MCPServer, tool string, args map[string]interface{}, got map[string]bool) {
	t.Helper()
	result := callTool(t, srv, tool, args)
	if result.IsError {
		t.Fatalf("%s: %s", tool, result.Content[0].Text)
	}
	var v interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &v); err != nil {
		t.Fatal(err)
	}
	schemaPaths(tool, v, got)
}

// read-contract-conformance AC-1: the v1 contract is additive-only. Every
// path:type recorded in the golden must still be produced; new paths are
// allowed and are added with -update-read-contract.
func TestReadContractV1SchemaIsAdditiveOnly(t *testing.T) {
	srv, heroDir := readContractWorkspace(t)
	stamp := readContractNow().UTC().Format("2006-01-02T15:04:05Z")
	writeContractSpec(t, heroDir, "planning/bugs/b", "---\ntitle: B\nslug: b\ntype: bug\nstatus: planning\nseverity: high\ntracker_id: X-1\nrelations:\n  - target: feat\n    kind: related\n  - target: base\n    kind: blocks\n---\n# B\n")
	writeContractSpec(t, heroDir, "specs/shipped", "---\ntitle: Shipped\nslug: shipped\ntype: feature\nstatus: completed\ncompleted_at: "+stamp+"\nrelations:\n  - target: b\n    kind: related\n---\n# S\n\n## Changes\n\n1. x\n\n## Acceptance Criteria\n\n- **AC-1:** THE SYSTEM SHALL x.\n")
	// A verified, sized, audited item: audit and size as strings, next null.
	ledger := "\n## Completion Ledger\n\n### Acceptance Criteria\n\n| # | C | Status | Note |\n|---|---|---|---|\n| 1 | AC-1 | DONE | ok |\n\n### Changes\n\n| # | I | Status | Note |\n|---|---|---|---|\n| 1 | x | DONE | ok |\n"
	writeContractSpec(t, heroDir, "specs/verified", "---\ntitle: Verified\nslug: verified\ntype: feature\nstatus: completed\nsize: small\ncompleted_at: "+stamp+"\n---\n# V\n\n## Changes\n\n1. x\n\n## Acceptance Criteria\n\n- **AC-1:** THE SYSTEM SHALL x.\n"+ledger)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(heroDir, "specs", "verified", "delivery-audit.md"), []byte("# Delivery audit — verified\n\n**Verdict:** SHIP\n"), 0o644)
	// An initiative-child decision: verify null.
	writeContractSpec(t, heroDir, "planning/initiatives/init/choice", "---\ntitle: Choice\nslug: choice\ntype: decision\nstatus: proposed\nparent: init\n---\n# C\n\n## Decision\n\nYes.\n")

	type call struct {
		tool string
		args map[string]interface{}
	}
	calls := []call{
		{"hero_handoff", map[string]interface{}{}}, // before NEXT.md exists: updated_at null
		{"hero_work", map[string]interface{}{}},
		{"hero_spec", map[string]interface{}{"slug": "feat"}},
		{"hero_spec", map[string]interface{}{"slug": "init"}},     // children[] elements
		{"hero_spec", map[string]interface{}{"slug": "b"}},        // blocks[] elements
		{"hero_spec", map[string]interface{}{"slug": "verified"}}, // audit/size strings, next null
		{"hero_spec", map[string]interface{}{"slug": "choice"}},   // verify null
	}
	got := map[string]bool{}
	for i, c := range calls {
		if i == 1 {
			os.WriteFile(filepath.Join(heroDir, "NEXT.md"), []byte("---\nupdated: 2026-10-06T00:00:00Z\nrepo: r\n---\n# Next\n"), 0o644)
			callAndRecord(t, srv, "hero_handoff", map[string]interface{}{}, got)
		}
		callAndRecord(t, srv, c.tool, c.args, got)
	}

	// A thin backlog (the base workspace) also yields the Explore item, whose
	// slug is null.
	thin, _ := readContractWorkspace(t)
	callAndRecord(t, thin, "hero_work", map[string]interface{}{}, got)

	golden := filepath.Join("testdata", "read_contract_v1.golden")
	data, err := os.ReadFile(golden)
	if err != nil && !*updateReadContract {
		t.Fatalf("read %s: %v (create with -update-read-contract)", golden, err)
	}
	want := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			want[line] = true
		}
	}
	var missing []string
	for p := range want {
		if !got[p] {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("read contract v1 is additive-only; these paths disappeared or changed type:\n  %s", strings.Join(missing, "\n  "))
	}
	if *updateReadContract {
		for p := range got {
			want[p] = true
		}
		lines := make([]string, 0, len(want))
		for p := range want {
			lines = append(lines, p)
		}
		sort.Strings(lines)
		body := "# Read contract v1 schema (path:type). Additive-only: never delete a line.\n" + strings.Join(lines, "\n") + "\n"
		if err := os.WriteFile(golden, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
