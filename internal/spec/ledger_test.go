package spec

import (
	"testing"
	"time"
)

func TestParseLedger_AllDone(t *testing.T) {
	content := `---
title: Test Feature
type: feature
status: delivering
---
# Test Feature

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL do X
- AC-2: WHEN Y THE SYSTEM SHALL do Z

## Changes

- ` + "`internal/foo.go`" + ` — add handler
- ` + "`internal/bar.go`" + ` — update router

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | THE SYSTEM SHALL do X | DONE | ` + "`internal/foo.go:42`" + ` — implemented |
| 2 | WHEN Y THE SYSTEM SHALL do Z | DONE | ` + "`internal/bar.go:10`" + ` — wired up |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Edit ` + "`internal/foo.go`" + ` | DONE | handler added |
| 2 | Edit ` + "`internal/bar.go`" + ` | DONE | route registered |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: ran hero verify test-slug, confirmed output

### Excellence Bar self-check

- [x] yes — clean implementation, well tested
`

	s, err := Parse(content, "/project/.hero/planning/features/test/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	ledger := ParseLedger(s)
	if !ledger.Found {
		t.Fatal("expected ledger to be found")
	}
	if len(ledger.ACRows) != 2 {
		t.Fatalf("ACRows = %d, want 2", len(ledger.ACRows))
	}
	for i, row := range ledger.ACRows {
		if row.Status != LedgerDone {
			t.Errorf("ACRow[%d].Status = %q, want DONE", i, row.Status)
		}
	}
	if len(ledger.ChangesRows) != 2 {
		t.Fatalf("ChangesRows = %d, want 2", len(ledger.ChangesRows))
	}
	for i, row := range ledger.ChangesRows {
		if row.Status != LedgerDone {
			t.Errorf("ChangesRow[%d].Status = %q, want DONE", i, row.Status)
		}
	}
	if !ledger.ExerciseChecked {
		t.Error("ExerciseChecked = false, want true")
	}
	if ledger.ExerciseDetail == "" {
		t.Error("ExerciseDetail is empty, want detail text")
	}
	if !ledger.ExcellenceChecked {
		t.Error("ExcellenceChecked = false, want true")
	}
}

func TestParseLedger_MixedStatuses(t *testing.T) {
	content := `---
title: Mixed
type: feature
status: delivering
---
# Mixed

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Do X | DONE | implemented |
| 2 | Do Y | PARTIAL | handler exists but not wired to router |
| 3 | Do Z | SKIPPED | out of scope for v1 |
| 4 | Do W | BLOCKED | depends on upstream API |

### Exercise-the-feature check

- [ ] OR: cannot be exercised because feature is incomplete
`

	s, err := Parse(content, "/project/.hero/planning/features/mixed/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	ledger := ParseLedger(s)
	if !ledger.Found {
		t.Fatal("expected ledger to be found")
	}

	want := []LedgerStatus{LedgerDone, LedgerPartial, LedgerSkipped, LedgerBlocked}
	if len(ledger.ACRows) != 4 {
		t.Fatalf("ACRows = %d, want 4", len(ledger.ACRows))
	}
	for i, row := range ledger.ACRows {
		if row.Status != want[i] {
			t.Errorf("ACRow[%d].Status = %q, want %q", i, row.Status, want[i])
		}
	}

	if ledger.ExerciseChecked {
		t.Error("ExerciseChecked = true, want false")
	}
	if ledger.ExerciseDetail == "" {
		t.Error("ExerciseDetail should have detail about why not exercised")
	}
}

func TestParseLedger_SignedOff(t *testing.T) {
	content := `---
title: Signed Off
type: feature
status: delivering
---
# Signed Off

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Do X | DONE | implemented |
| 2 | Do Y | SKIPPED | out of scope [signed-off] |
| 3 | Do Z | BLOCKED | upstream dep [signed off] |

### Exercise-the-feature check

- [x] Exercised: ran the CLI command and confirmed output
`

	s, err := Parse(content, "/project/.hero/planning/features/signed/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	ledger := ParseLedger(s)
	if !ledger.Found {
		t.Fatal("expected ledger to be found")
	}

	if len(ledger.ACRows) != 3 {
		t.Fatalf("ACRows = %d, want 3", len(ledger.ACRows))
	}

	// Row 2 should be signed-off
	if !ledger.ACRows[1].SignedOff {
		t.Error("ACRow[1].SignedOff = false, want true (has [signed-off])")
	}
	if ledger.ACRows[1].Status != LedgerSkipped {
		t.Errorf("ACRow[1].Status = %q, want SKIPPED", ledger.ACRows[1].Status)
	}

	// Row 3 should be signed-off (with space variant)
	if !ledger.ACRows[2].SignedOff {
		t.Error("ACRow[2].SignedOff = false, want true (has [signed off])")
	}
}

func TestParseLedger_Missing(t *testing.T) {
	content := `---
title: No Ledger
type: feature
status: delivering
---
# No Ledger

## Changes

- Edit some file
`

	s, err := Parse(content, "/project/.hero/planning/features/noledger/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	ledger := ParseLedger(s)
	if ledger.Found {
		t.Error("expected ledger to not be found")
	}
}

func TestParseLedger_CaseInsensitiveStatus(t *testing.T) {
	content := `---
title: Case Test
type: feature
status: delivering
---
# Case Test

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Do X | done | lowercase |
| 2 | Do Y | Done | titlecase |
| 3 | Do Z | DONE | uppercase |
| 4 | Do W | partial | lowercase partial |

### Exercise-the-feature check

- [X] Exercised end-to-end: confirmed working
`

	s, err := Parse(content, "/project/.hero/planning/features/case/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	ledger := ParseLedger(s)
	if len(ledger.ACRows) != 4 {
		t.Fatalf("ACRows = %d, want 4", len(ledger.ACRows))
	}
	for i := 0; i < 3; i++ {
		if ledger.ACRows[i].Status != LedgerDone {
			t.Errorf("ACRow[%d].Status = %q, want DONE", i, ledger.ACRows[i].Status)
		}
	}
	if ledger.ACRows[3].Status != LedgerPartial {
		t.Errorf("ACRow[3].Status = %q, want PARTIAL", ledger.ACRows[3].Status)
	}
}

func TestParseLedger_ExerciseNoDetail(t *testing.T) {
	content := `---
title: No Detail
type: feature
status: delivering
---
# No Detail

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Do X | DONE | done |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end:
`

	s, err := Parse(content, "/project/.hero/planning/features/nodetail/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	ledger := ParseLedger(s)
	if !ledger.ExerciseChecked {
		t.Error("ExerciseChecked = false, want true")
	}
	// Checkbox checked but colon with nothing after it — detail should be empty
	if ledger.ExerciseDetail != "" {
		t.Errorf("ExerciseDetail = %q, want empty", ledger.ExerciseDetail)
	}
}

func TestParseLedger_BoldStatus(t *testing.T) {
	content := `---
title: Bold
type: feature
status: delivering
---
# Bold

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | **Do X** | **DONE** | implemented |

### Exercise-the-feature check

- [x] Exercised: ran it
`

	s, err := Parse(content, "/project/.hero/planning/features/bold/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	ledger := ParseLedger(s)
	if len(ledger.ACRows) != 1 {
		t.Fatalf("ACRows = %d, want 1", len(ledger.ACRows))
	}
	if ledger.ACRows[0].Status != LedgerDone {
		t.Errorf("ACRow[0].Status = %q, want DONE (bold markers should be stripped)", ledger.ACRows[0].Status)
	}
}

func TestParseLedger_NilSpec(t *testing.T) {
	ledger := ParseLedger(nil)
	if ledger.Found {
		t.Error("expected ledger to not be found for nil spec")
	}
}
