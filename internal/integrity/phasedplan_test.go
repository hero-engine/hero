package integrity

import (
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestFindPhasedPlanTables_ParsesGraphMemoryShape(t *testing.T) {
	body := `## Approach

| Phase | Scope | Status |
|---|---|---|
| 1 | Schema + code subgraph | ✅ |
| 2 | Work subgraph | ✅ |
| 3a | Raw docs | ✅ |
| **3b** | Federation contracts | next |
| 4 | NEXT.md projections | |
| 5 | hero why / blocked / impact | |

Other prose follows.
`
	tables := findPhasedPlanTables(body)
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}
	tbl := tables[0]
	if tbl.Total != 6 {
		t.Errorf("Total = %d, want 6", tbl.Total)
	}
	if tbl.Shipped != 3 {
		t.Errorf("Shipped = %d, want 3", tbl.Shipped)
	}
	if tbl.Pending != 3 {
		t.Errorf("Pending = %d (next + 2 empty), want 3", tbl.Pending)
	}
}

func TestFindPhasedPlanTables_IgnoresTablesWithoutStatusColumn(t *testing.T) {
	body := `| Name | Description |
|---|---|
| ABC | first |
| XYZ | second |
`
	tables := findPhasedPlanTables(body)
	if len(tables) != 0 {
		t.Errorf("got %d tables, want 0 (no Status column)", len(tables))
	}
}

func TestFindPhasedPlanTables_HandlesShippedKeyword(t *testing.T) {
	body := `| Phase | Status |
|---|---|
| 1 | shipped |
| 2 | done |
| 3 | wip |
`
	tables := findPhasedPlanTables(body)
	if len(tables) != 1 {
		t.Fatalf("len = %d", len(tables))
	}
	if tables[0].Shipped != 2 || tables[0].Pending != 1 {
		t.Errorf("got Shipped=%d Pending=%d, want 2/1", tables[0].Shipped, tables[0].Pending)
	}
}

func TestExplainPhasedInconsistency_FlagsLyingTable(t *testing.T) {
	cases := []struct {
		name string
		f    PhasedPlanFinding
		want bool
	}{
		{
			"all shipped + planning frontmatter",
			PhasedPlanFinding{Total: 5, Shipped: 5, FrontStatus: spec.StatusPlanning},
			true,
		},
		{
			"all shipped + delivering frontmatter",
			PhasedPlanFinding{Total: 5, Shipped: 5, FrontStatus: spec.StatusDelivering},
			true,
		},
		{
			"all shipped + 0 passing ACs",
			PhasedPlanFinding{Total: 5, Shipped: 5, FrontStatus: spec.StatusCompleted, TotalACs: 5, PassingACs: 0},
			true,
		},
		{
			"all shipped + minority passing ACs",
			PhasedPlanFinding{Total: 5, Shipped: 5, FrontStatus: spec.StatusCompleted, TotalACs: 10, PassingACs: 2},
			true,
		},
		{
			"all shipped + all ACs passing — clean",
			PhasedPlanFinding{Total: 5, Shipped: 5, FrontStatus: spec.StatusCompleted, TotalACs: 5, PassingACs: 5},
			false,
		},
		{
			"partially shipped — no judgment for planning",
			PhasedPlanFinding{Total: 5, Shipped: 2, FrontStatus: spec.StatusPlanning},
			false,
		},
		{
			"completed frontmatter + pending phases — fraud",
			PhasedPlanFinding{Total: 10, Shipped: 3, Pending: 7, FrontStatus: spec.StatusCompleted},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := explainPhasedInconsistency(tc.f)
			if (got != "") != tc.want {
				t.Errorf("explanation=%q, want non-empty=%v", got, tc.want)
			}
		})
	}
}

func TestCheckPhasedPlans_CombinesMultipleSpecs(t *testing.T) {
	specs := []*spec.Spec{
		{
			Slug:   "feat-a",
			Path:   "/tmp/a.md",
			Status: spec.StatusPlanning,
			RawContent: `## Plan
| Phase | Status |
|---|---|
| 1 | ✅ |
| 2 | ✅ |
`,
		},
		{
			Slug:       "feat-b",
			Path:       "/tmp/b.md",
			Status:     spec.StatusDelivering,
			RawContent: "## Plan\n\nNo table here.\n",
		},
	}
	out := CheckPhasedPlans(specs, nil)
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if out[0].Slug != "feat-a" {
		t.Errorf("got finding for %q, want feat-a", out[0].Slug)
	}
}
