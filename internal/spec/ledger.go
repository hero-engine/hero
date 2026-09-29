package spec

import (
	"fmt"
	"strings"
	"unicode"
)

// LedgerStatus represents the completion state of a ledger row.
type LedgerStatus string

const (
	LedgerDone    LedgerStatus = "DONE"
	LedgerPartial LedgerStatus = "PARTIAL"
	LedgerSkipped LedgerStatus = "SKIPPED"
	LedgerBlocked LedgerStatus = "BLOCKED"
	LedgerUnknown LedgerStatus = "UNKNOWN"
)

// LedgerRow is one row from the Completion Ledger tables.
type LedgerRow struct {
	Index     int
	Summary   string
	Status    LedgerStatus
	Note      string
	SignedOff bool // true if Note asserts a [signed-off] annotation (see parseSignOff)
	// SignOffRejected explains why a [signed-off] marker present in Note was
	// not honored; empty when there is no marker or it was honored.
	SignOffRejected string
}

// LedgerResult holds the parsed Completion Ledger from a spec.
type LedgerResult struct {
	Found             bool
	ACRows            []LedgerRow
	ChangesRows       []LedgerRow
	ExerciseChecked   bool
	ExerciseDetail    string // description text after the checkbox
	ExcellenceChecked bool
	ExcellenceNote    string
}

// ParseLedger extracts the Completion Ledger from a spec's sections.
// It looks for the "completion ledger" section (case-insensitive key)
// and parses the AC table, Changes table, and exercise/excellence checks.
func ParseLedger(s *Spec) LedgerResult {
	if s == nil {
		return LedgerResult{}
	}
	content, ok := s.Sections["completion ledger"]
	if !ok {
		return LedgerResult{}
	}
	return parseLedgerContent(content)
}

// parseLedgerContent parses the body of a Completion Ledger section.
func parseLedgerContent(content string) LedgerResult {
	result := LedgerResult{Found: true}

	// Split into sub-sections by ### headers
	type subSection struct {
		name string
		body string
	}
	var subs []subSection
	var currentName string
	var currentBody strings.Builder

	var fence fenceTracker
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		// A `### ` inside a fenced block is quoted, not a sub-section break.
		if fence.mark(line) || fence.inCode() {
			currentBody.WriteString(line)
			currentBody.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			if currentName != "" {
				subs = append(subs, subSection{currentName, currentBody.String()})
			}
			currentName = strings.ToLower(strings.TrimPrefix(trimmed, "### "))
			currentBody.Reset()
		} else {
			currentBody.WriteString(line)
			currentBody.WriteString("\n")
		}
	}
	if currentName != "" {
		subs = append(subs, subSection{currentName, currentBody.String()})
	}

	for _, sub := range subs {
		switch {
		case strings.Contains(sub.name, "acceptance criteria"):
			result.ACRows = parseTable(sub.body)
		case sub.name == "changes":
			result.ChangesRows = parseTable(sub.body)
		case strings.Contains(sub.name, "exercise"):
			result.ExerciseChecked, result.ExerciseDetail = parseCheckbox(sub.body)
		case strings.Contains(sub.name, "excellence"):
			result.ExcellenceChecked, result.ExcellenceNote = parseCheckbox(sub.body)
		}
	}

	return result
}

// parseTable extracts rows from a pipe-delimited markdown table.
// Tolerant of formatting variations: extra whitespace, bold text,
// missing leading/trailing pipes.
func parseTable(body string) []LedgerRow {
	lines := strings.Split(body, "\n")
	var rows []LedgerRow
	headerSeen := false
	separatorSeen := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Need at least one pipe to be a table line
		if !strings.Contains(trimmed, "|") {
			continue
		}

		cells := splitTableRow(trimmed)
		if len(cells) < 3 {
			continue
		}

		// Detect header row (contains "Status" or "#")
		if !headerSeen {
			for _, c := range cells {
				lower := strings.ToLower(c)
				if lower == "#" || lower == "status" || lower == "criterion" || lower == "changes item" {
					headerSeen = true
					break
				}
			}
			if headerSeen {
				continue
			}
		}

		// Detect separator row (all dashes/colons)
		if headerSeen && !separatorSeen {
			isSep := true
			for _, c := range cells {
				stripped := strings.Trim(c, "-: ")
				if stripped != "" {
					isSep = false
					break
				}
			}
			if isSep {
				separatorSeen = true
				continue
			}
		}

		// Parse data row
		row := parseDataRow(cells, len(rows)+1)
		rows = append(rows, row)
	}

	return rows
}

// splitTableRow splits a pipe-delimited markdown table row into cells,
// stripping leading/trailing pipes and whitespace.
func splitTableRow(line string) []string {
	// Strip leading/trailing pipe
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}

	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// parseDataRow converts table cells into a LedgerRow.
func parseDataRow(cells []string, fallbackIndex int) LedgerRow {
	row := LedgerRow{Index: fallbackIndex}

	// Try to parse index from first cell
	if len(cells) > 0 {
		idx := parseIndex(cells[0])
		if idx > 0 {
			row.Index = idx
		}
	}

	// Column mapping depends on cell count:
	// 4 cells: #, Summary, Status, Note
	// 3 cells: Summary, Status, Note (no index column)
	switch {
	case len(cells) >= 4:
		row.Summary = stripBold(cells[1])
		row.Status = parseStatus(cells[2])
		row.Note = stripBold(cells[3])
	case len(cells) >= 3:
		row.Summary = stripBold(cells[0])
		row.Status = parseStatus(cells[1])
		row.Note = stripBold(cells[2])
	}

	row.SignedOff, row.SignOffRejected = parseSignOff(row.Note)

	return row
}

// SignOffForm is the accepted shape of a ledger sign-off, shown to authors
// whenever a marker is rejected.
const SignOffForm = "`[signed-off] <who> — <why>` or `[signed-off: <who>] <why>` at the start of the Note"

var signOffMarkers = []string{"[signed-off", "[signed off"}

// signOffDenials may never appear in the signer attribution; they mean the
// note is requesting, deferring, or denying sign-off rather than asserting it.
var signOffDenials = map[string]bool{
	"not": true, "no": true, "never": true, "without": true, "missing": true,
	"need": true, "needs": true, "needed": true, "pending": true, "until": true,
	"before": true, "await": true, "awaits": true, "awaiting": true,
	"require": true, "requires": true, "required": true, "requested": true,
	"requesting": true, "withheld": true, "yet": true, "denied": true,
	"outstanding": true, "todo": true, "tbd": true,
}

// maxSignerWords bounds the attribution so free prose ("was never given") can
// never pass as a signer name.
const maxSignerWords = 3

// parseSignOff reports whether a ledger note asserts sign-off. Only the
// structured form counts: the note must open with the marker and name a
// signer, either inside the brackets (`[signed-off: bwheeler]`) or directly
// after them and terminated by a dash (`[signed-off] bwheeler — why`).
// Everything else fails closed and returns the reason for Gate 1 to report.
func parseSignOff(note string) (bool, string) {
	lower := strings.ToLower(strings.TrimSpace(note))
	var marker string
	for _, m := range signOffMarkers {
		if strings.Contains(lower, m+"]") || strings.Contains(lower, m+":") {
			marker = m
			break
		}
	}
	if marker == "" {
		return false, ""
	}
	if !strings.HasPrefix(lower, marker) {
		return false, "the marker must open the note"
	}
	rest := lower[len(marker):]

	var signer string
	switch {
	case strings.HasPrefix(rest, ":"):
		close := strings.Index(rest, "]")
		if close < 0 {
			return false, "the marker is missing its closing bracket"
		}
		signer = rest[1:close]
	case strings.HasPrefix(rest, "]"):
		rest = rest[1:]
		dash := -1
		for _, sep := range []string{" — ", " – ", " - "} {
			if i := strings.Index(rest, sep); i >= 0 && (dash < 0 || i < dash) {
				dash = i
			}
		}
		if dash < 0 {
			return false, "no signer before a dash after the marker"
		}
		signer = rest[:dash]
	default:
		return false, "the marker is malformed"
	}

	words := strings.FieldsFunc(signer, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' && r != '@'
	})
	if len(words) == 0 {
		return false, "no signer is named"
	}
	if len(words) > maxSignerWords {
		return false, fmt.Sprintf("the signer must be at most %d words", maxSignerWords)
	}
	for _, w := range words {
		if signOffDenials[w] {
			return false, fmt.Sprintf("the signer contains %q", w)
		}
	}
	return true, ""
}

// parseIndex extracts a numeric index from a cell like "1", "1.", "AC-1", etc.
func parseIndex(cell string) int {
	cell = strings.TrimSpace(cell)
	cell = stripBold(cell)

	// Try plain number
	var n int
	for _, ch := range cell {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		} else {
			break
		}
	}
	return n
}

// parseStatus matches a cell value to a LedgerStatus, case-insensitive.
func parseStatus(cell string) LedgerStatus {
	cell = strings.TrimSpace(cell)
	cell = stripBold(cell)
	cell = strings.Trim(cell, "`")

	switch strings.ToUpper(cell) {
	case "DONE":
		return LedgerDone
	case "PARTIAL":
		return LedgerPartial
	case "SKIPPED":
		return LedgerSkipped
	case "BLOCKED":
		return LedgerBlocked
	default:
		return LedgerUnknown
	}
}

// stripBold removes markdown bold markers (**) from text.
func stripBold(s string) string {
	return strings.ReplaceAll(s, "**", "")
}

// parseCheckbox extracts checked state and detail text from a checkbox section.
// Matches patterns like:
//   - [x] User-visible behavior was exercised: ran hero verify...
//   - [ ] OR: cannot be exercised because...
func parseCheckbox(body string) (checked bool, detail string) {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)

		// Match - [x] or - [ ] or * [x] etc.
		trimmed = strings.TrimLeft(trimmed, "-*+ ")
		trimmed = strings.TrimSpace(trimmed)

		if strings.HasPrefix(trimmed, "[x]") || strings.HasPrefix(trimmed, "[X]") {
			rest := strings.TrimSpace(trimmed[3:])
			// Strip common prefixes
			for _, prefix := range []string{
				"User-visible behavior was exercised end-to-end:",
				"User-visible behavior was exercised:",
				"Exercised end-to-end:",
				"Exercised:",
			} {
				if strings.HasPrefix(rest, prefix) {
					rest = strings.TrimSpace(rest[len(prefix):])
					break
				}
			}
			if rest != "" {
				return true, rest
			}
			return true, ""
		}

		if strings.HasPrefix(trimmed, "[ ]") {
			// Unchecked — but might have "OR: cannot be exercised" detail
			rest := strings.TrimSpace(trimmed[3:])
			if strings.HasPrefix(rest, "OR:") {
				rest = strings.TrimSpace(rest[3:])
			}
			return false, rest
		}
	}
	return false, ""
}
