package serve

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
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

// read-contract-conformance AC-1: the v1 contract is additive-only. Every
// path:type recorded in the golden must still be produced; new paths are
// allowed and are added with -update-read-contract.
func TestReadContractV1SchemaIsAdditiveOnly(t *testing.T) {
	srv, heroDir := readContractWorkspace(t)
	writeContractSpec(t, heroDir, "planning/bugs/b", "---\ntitle: B\nslug: b\ntype: bug\nstatus: planning\nseverity: high\ntracker_id: X-1\nrelations:\n  - target: feat\n    kind: related\n---\n# B\n")
	writeContractSpec(t, heroDir, "specs/shipped", "---\ntitle: Shipped\nslug: shipped\ntype: feature\nstatus: completed\ncompleted_at: "+readContractNow().UTC().Format("2006-01-02T15:04:05Z")+"\nrelations:\n  - target: b\n    kind: related\n---\n# S\n\n## Changes\n\n1. x\n\n## Acceptance Criteria\n\n- **AC-1:** THE SYSTEM SHALL x.\n")
	os.WriteFile(filepath.Join(heroDir, "NEXT.md"), []byte("---\nupdated: 2026-10-06T00:00:00Z\n---\n# Next\n"), 0o644)

	got := map[string]bool{}
	for tool, args := range map[string]map[string]interface{}{
		"hero_work":    {},
		"hero_spec":    {"slug": "feat"},
		"hero_handoff": {},
	} {
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
