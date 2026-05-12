package integrity

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// PhasedPlanFinding describes a phased-plan table inside a spec
// where the ✅-shipped claims look inconsistent with reality. Today
// "reality" is two graph signals:
//
//   1. The spec's own frontmatter status. A phased plan with 100% ✅
//      rows but `status: planning` in frontmatter is structurally
//      contradictory.
//   2. The Criterion graph rollup. If 5/5 phases claim ✅ but 0 ACs
//      are passing, the table is asserting more progress than the
//      AC graph has evidence for.
type PhasedPlanFinding struct {
	Slug          string
	Path          string
	HeaderLine    int    // 1-indexed line of the table header in the spec
	Total         int    // rows after the separator
	Shipped       int    // ✅ rows
	Pending       int    // empty / `next` / `wip` etc.
	Other         int    // anything else (❌ ⚠️ etc.)
	FrontStatus   spec.Status
	PassingACs    int
	TotalACs      int
	Inconsistency string // human-readable explanation, empty if clean
}

// CheckPhasedPlans walks every spec, finds tables with a Status
// column containing checkmarks, and flags those whose claims look
// inconsistent with the rest of the graph. Returns one finding per
// suspicious table. Tables with no ✅ rows are skipped (nothing to
// verify against).
//
// finds is the per-spec Finding from CheckCompletedSpecs (for AC
// counts). Specs with no AC data get an empty Finding entry.
func CheckPhasedPlans(specs []*spec.Spec, finds map[string]Finding) []PhasedPlanFinding {
	out := []PhasedPlanFinding{}
	for _, s := range specs {
		if s == nil || s.RawContent == "" {
			continue
		}
		tables := findPhasedPlanTables(s.RawContent)
		for _, t := range tables {
			if t.Shipped == 0 {
				continue
			}
			f := PhasedPlanFinding{
				Slug:        s.Slug,
				Path:        s.Path,
				HeaderLine:  t.HeaderLine,
				Total:       t.Total,
				Shipped:     t.Shipped,
				Pending:     t.Pending,
				Other:       t.Other,
				FrontStatus: s.Status,
			}
			if af, ok := finds[s.Slug]; ok {
				f.PassingACs = af.Passing
				f.TotalACs = af.Total
			}
			f.Inconsistency = explainPhasedInconsistency(f)
			if f.Inconsistency != "" {
				out = append(out, f)
			}
		}
	}
	return out
}

// explainPhasedInconsistency returns a one-line reason when the
// phased plan disagrees with the rest of the graph. Empty string
// means "nothing surprising."
func explainPhasedInconsistency(f PhasedPlanFinding) string {
	allShipped := f.Shipped == f.Total && f.Total > 0
	pending := f.Pending + f.Other
	switch {
	case allShipped && f.FrontStatus == spec.StatusPlanning:
		return "phased plan claims 100% shipped but frontmatter says planning"
	case allShipped && f.FrontStatus == spec.StatusDelivering:
		return "phased plan claims 100% shipped but frontmatter says delivering"
	case allShipped && f.TotalACs > 0 && f.PassingACs == 0:
		return fmt.Sprintf("phased plan claims 100%% shipped but 0/%d ACs are passing", f.TotalACs)
	case allShipped && f.TotalACs > 0 && f.PassingACs < f.TotalACs/2:
		return fmt.Sprintf("phased plan claims 100%% shipped but only %d/%d ACs are passing",
			f.PassingACs, f.TotalACs)
	case f.FrontStatus == spec.StatusCompleted && pending > 0:
		return fmt.Sprintf("frontmatter says completed but phased plan has %d pending row(s) (%d/%d ✅)",
			pending, f.Shipped, f.Total)
	}
	return ""
}

// phasedTable is the parser-internal rollup of one table.
type phasedTable struct {
	HeaderLine int
	Total      int
	Shipped    int
	Pending    int
	Other      int
}

// findPhasedPlanTables scans the markdown body for tables whose
// header includes a "status"-ish column, and counts the rows. We
// don't model the table semantically — just the row distribution.
//
// A "phased-plan-ish" table is identified heuristically:
//
//   - A header row pipe-table with a column named status / state /
//     ship / shipped / done.
//   - Followed by a separator row of dashes.
//   - Then any number of body rows.
//
// Cell content is classified by checking the status column for
// known glyphs / words. Bold and emphasis markers are stripped first.
func findPhasedPlanTables(body string) []phasedTable {
	var tables []phasedTable

	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !looksLikeTableHeader(line) {
			continue
		}
		// Need a separator row immediately after.
		if i+1 >= len(lines) || !looksLikeTableSeparator(lines[i+1]) {
			continue
		}
		statusCol := findStatusColumn(line)
		if statusCol < 0 {
			i++ // skip past separator
			continue
		}
		t := phasedTable{HeaderLine: i + 1}
		// Walk body rows.
		j := i + 2
		for j < len(lines) {
			row := lines[j]
			if !strings.HasPrefix(strings.TrimSpace(row), "|") {
				break
			}
			cells := splitTableRow(row)
			if statusCol >= len(cells) {
				j++
				continue
			}
			t.Total++
			switch classifyStatusCell(cells[statusCol]) {
			case statusShipped:
				t.Shipped++
			case statusPending:
				t.Pending++
			default:
				t.Other++
			}
			j++
		}
		if t.Total > 0 {
			tables = append(tables, t)
		}
		i = j - 1
	}
	return tables
}

func looksLikeTableHeader(line string) bool {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "|") || !strings.HasSuffix(t, "|") {
		return false
	}
	return strings.Count(t, "|") >= 3
}

func looksLikeTableSeparator(line string) bool {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "|") {
		return false
	}
	// All cells must be dashes (with optional :) and pipes.
	for _, r := range t {
		if r != '|' && r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return strings.Contains(t, "-")
}

// findStatusColumn returns the (0-indexed) column number of a header
// cell whose text matches a known status header. -1 if none.
func findStatusColumn(headerLine string) int {
	cells := splitTableRow(headerLine)
	for i, raw := range cells {
		head := strings.ToLower(stripEmphasis(raw))
		head = strings.TrimSpace(head)
		switch head {
		case "status", "state", "ship", "shipped", "done", "phase status":
			return i
		}
	}
	return -1
}

func splitTableRow(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	parts := strings.Split(row, "|")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func stripEmphasis(s string) string {
	s = strings.TrimSpace(s)
	for _, m := range []string{"**", "*", "__", "_"} {
		s = strings.TrimPrefix(s, m)
		s = strings.TrimSuffix(s, m)
	}
	return strings.TrimSpace(s)
}

type cellStatus int

const (
	statusUnknown cellStatus = iota
	statusShipped
	statusPending
	statusOther
)

func classifyStatusCell(cell string) cellStatus {
	t := strings.ToLower(stripEmphasis(cell))
	t = strings.TrimSpace(t)
	if t == "" {
		return statusPending
	}
	// Glyph-based: ✅ → shipped; ❌, ⚠️, 🔻 → other; otherwise pending.
	if strings.Contains(t, "✅") {
		return statusShipped
	}
	if strings.ContainsAny(t, "❌⚠🔻") {
		return statusOther
	}
	switch {
	case strings.Contains(t, "shipped"), strings.Contains(t, "complete"), strings.Contains(t, "done"):
		return statusShipped
	case strings.Contains(t, "next"), strings.Contains(t, "wip"), strings.Contains(t, "in progress"),
		strings.Contains(t, "todo"), strings.Contains(t, "pending"), strings.Contains(t, "planning"):
		return statusPending
	}
	return statusOther
}
