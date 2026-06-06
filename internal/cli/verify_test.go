package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// specWithLedgerAndAudit is a complete spec ready to pass all gates.
const specWithLedgerAndAudit = `---
title: Test Feature
type: feature
status: delivering
slug: test-feature
---
# Test Feature

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL do X

## Changes

- ` + "`internal/foo.go`" + ` — add handler

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | THE SYSTEM SHALL do X | DONE | ` + "`internal/foo.go:42`" + ` — implemented |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | Edit ` + "`internal/foo.go`" + ` | DONE | handler added |

### Exercise-the-feature check

- [x] Exercised: ran hero verify test-feature, confirmed gates report correctly

### Excellence Bar self-check

- [x] yes — clean implementation
`

const auditReportShip = `# Delivery audit — test-feature

**Audited:** git diff main...HEAD
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] THE SYSTEM SHALL do X — internal/foo.go:42
`

const auditReportHold = `# Delivery audit — test-feature

**Audited:** git diff main...HEAD
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✗] THE SYSTEM SHALL do X — no evidence found
`

func TestVerify_AllGatesPass(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/test-feature/spec.md", specWithLedgerAndAudit)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/test-feature/delivery-audit.md"), auditReportShip)
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "test-feature")
	if err != nil {
		t.Fatalf("verify failed: %v\noutput: %s", err, output)
	}

	if !strings.Contains(output, "PASS") {
		t.Errorf("expected PASS in output, got:\n%s", output)
	}

	// Check spec was archived
	archivedPath := filepath.Join(env.heroDir, "specs", "test-feature", "spec.md")
	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Error("spec was not archived to specs/test-feature/")
	}
}

func TestVerify_MissingLedger(t *testing.T) {
	env := newTestEnv(t)
	specContent := `---
title: No Ledger
type: feature
status: delivering
slug: no-ledger
---
# No Ledger

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL do X
`
	env.addSpec("planning/features/no-ledger/spec.md", specContent)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/no-ledger/delivery-audit.md"), auditReportShip)
	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "no-ledger")
	if err == nil {
		t.Fatal("expected verify to fail for missing ledger")
	}
	if !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("error = %q, want 'verification failed'", err.Error())
	}
}

func TestVerify_PartialRows(t *testing.T) {
	env := newTestEnv(t)
	specContent := `---
title: Partial
type: feature
status: delivering
slug: partial-spec
---
# Partial

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL do X

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Do X | PARTIAL | handler exists but not wired |

### Exercise-the-feature check

- [ ] not exercised
`
	env.addSpec("planning/features/partial-spec/spec.md", specContent)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/partial-spec/delivery-audit.md"), auditReportShip)
	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "partial-spec")
	if err == nil {
		t.Fatal("expected verify to fail for PARTIAL rows")
	}

	// Spec should NOT be archived
	archivedPath := filepath.Join(env.heroDir, "specs", "partial-spec", "spec.md")
	if _, err := os.Stat(archivedPath); !os.IsNotExist(err) {
		t.Error("spec should NOT be archived when gates fail")
	}
}

func TestVerify_MissingAudit(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/no-audit/spec.md", strings.Replace(specWithLedgerAndAudit, "test-feature", "no-audit", -1))
	// No audit file written
	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "no-audit")
	if err == nil {
		t.Fatal("expected verify to fail for missing audit")
	}
}

func TestVerify_HoldAudit(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/hold-spec/spec.md", strings.Replace(specWithLedgerAndAudit, "test-feature", "hold-spec", -1))
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/hold-spec/delivery-audit.md"),
		strings.Replace(auditReportHold, "test-feature", "hold-spec", -1))
	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "hold-spec")
	if err == nil {
		t.Fatal("expected verify to fail for HOLD verdict")
	}
}

func TestVerify_Force(t *testing.T) {
	env := newTestEnv(t)
	specContent := `---
title: Force Test
type: feature
status: delivering
slug: force-test
---
# Force Test

## Acceptance Criteria

- AC-1: Do X
`
	// No ledger, no audit — would normally fail
	env.addSpec("planning/features/force-test/spec.md", specContent)
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "--force", "force-test")
	if err != nil {
		t.Fatalf("force verify should not error: %v\noutput: %s", err, output)
	}

	if !strings.Contains(output, "FORCED") {
		t.Errorf("expected FORCED in output, got:\n%s", output)
	}
}

func TestVerify_JSON(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/json-test/spec.md", strings.Replace(specWithLedgerAndAudit, "test-feature", "json-test", -1))
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/json-test/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "json-test", -1))
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "--json", "json-test")
	if err != nil {
		t.Fatalf("verify failed: %v\noutput: %s", err, output)
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if result.Slug != "json-test" {
		t.Errorf("Slug = %q, want json-test", result.Slug)
	}
	if result.Result != "PASS" {
		t.Errorf("Result = %q, want PASS", result.Result)
	}
	if len(result.Gates) < 4 {
		t.Errorf("Gates count = %d, want >= 4", len(result.Gates))
	}
}

func TestVerify_SignedOffPassesGate(t *testing.T) {
	env := newTestEnv(t)
	specContent := `---
title: Signed Off Pass
type: feature
status: delivering
slug: signed-pass
---
# Signed Off Pass

## Acceptance Criteria

- AC-1: Do X
- AC-2: Do Y

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Do X | DONE | implemented |
| 2 | Do Y | SKIPPED | out of scope [signed-off] |

### Exercise-the-feature check

- [x] Exercised: ran the command, works
`
	env.addSpec("planning/features/signed-pass/spec.md", specContent)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/signed-pass/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "signed-pass", -1))
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "signed-pass")
	if err != nil {
		t.Fatalf("verify should pass with signed-off SKIPPED: %v\noutput: %s", err, output)
	}
}

func TestVerify_AlreadyCompleted(t *testing.T) {
	env := newTestEnv(t)
	specContent := `---
title: Already Done
type: feature
status: completed
slug: already-done
---
# Already Done
`
	env.addSpec("specs/already-done/spec.md", specContent)
	env.indexAll()

	output, err := runCmd("spec", "verify", "already-done")
	if err != nil {
		t.Fatalf("verify on already-completed should not error: %v", err)
	}
	if !strings.Contains(output, "already completed") {
		t.Errorf("expected 'already completed' message, got:\n%s", output)
	}
}

func TestVerify_SkipTests(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/skip-test/spec.md", strings.Replace(specWithLedgerAndAudit, "test-feature", "skip-test", -1))
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/skip-test/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "skip-test", -1))
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "--json", "skip-test")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	// Find Gate 4
	found := false
	for _, g := range result.Gates {
		if g.Name == "Build & Tests" {
			found = true
			if g.Result != "SKIPPED" {
				t.Errorf("Gate 4 Result = %q, want SKIPPED", g.Result)
			}
		}
	}
	if !found {
		t.Error("Build & Tests gate not found in results")
	}
}

// writeVerifyFile is a test helper that creates a file with content.
func writeVerifyFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
