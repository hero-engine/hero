package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
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

func TestVerify_PlanningStatusGuarded(t *testing.T) {
	env := newTestEnv(t)
	// Fully complete spec (ledger + audit) — would pass every gate — but
	// in planning status. The lifecycle guard must block it anyway.
	planningSpec := strings.Replace(
		strings.Replace(specWithLedgerAndAudit, "test-feature", "planning-guard", -1),
		"status: delivering", "status: planning", 1)
	env.addSpec("planning/features/planning-guard/spec.md", planningSpec)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/planning-guard/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "planning-guard", -1))
	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "planning-guard")
	if err == nil {
		t.Fatal("expected verify to be blocked for a planning-status spec")
	}
	if !strings.Contains(err.Error(), "planning status") {
		t.Errorf("error = %q, want a lifecycle message mentioning 'planning status'", err.Error())
	}
	// A planning draft must never be archived.
	archivedPath := filepath.Join(env.heroDir, "specs", "planning-guard", "spec.md")
	if _, statErr := os.Stat(archivedPath); statErr == nil {
		t.Error("planning spec was archived despite the lifecycle guard")
	}
}

func TestVerify_PlanningStatusForceBypass(t *testing.T) {
	env := newTestEnv(t)
	planningSpec := strings.Replace(
		strings.Replace(specWithLedgerAndAudit, "test-feature", "planning-force", -1),
		"status: delivering", "status: planning", 1)
	env.addSpec("planning/features/planning-force/spec.md", planningSpec)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/planning-force/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "planning-force", -1))
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "--force", "planning-force")
	if err != nil {
		t.Fatalf("verify --force should bypass the lifecycle guard: %v\noutput: %s", err, output)
	}
	// With a complete ledger+audit, forced gates pass and the spec archives.
	archivedPath := filepath.Join(env.heroDir, "specs", "planning-force", "spec.md")
	if _, statErr := os.Stat(archivedPath); os.IsNotExist(statErr) {
		t.Error("forced verify did not archive the spec")
	}
}

func TestVerify_PlanningStatusJSON(t *testing.T) {
	env := newTestEnv(t)
	planningSpec := strings.Replace(
		strings.Replace(specWithLedgerAndAudit, "test-feature", "planning-json", -1),
		"status: delivering", "status: planning", 1)
	env.addSpec("planning/features/planning-json/spec.md", planningSpec)
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "--json", "planning-json")
	if err != nil {
		t.Fatalf("json verify on a planning spec should exit 0: %v\noutput: %s", err, output)
	}
	var result VerifyResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}
	if result.Result != "SKIPPED" {
		t.Errorf("Result = %q, want SKIPPED", result.Result)
	}
	if len(result.Gates) != 1 || result.Gates[0].Name != "lifecycle" {
		t.Errorf("expected a single 'lifecycle' gate, got %+v", result.Gates)
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

// seedGraphCriterion inserts a Criterion node into the graph at heroDir.
// seedGraphCriterion seeds a Criterion node in the partition production
// actually writes and reads. repo must be gitutil.RepoKey(projectRoot) —
// the key acceptance.Record and spec.WriteGraph both use. This previously
// hard-coded "test-repo", which only worked because the graph's Criterion
// lookup was unscoped; once node identity became (type, key, repo) the
// fixture was seeding a partition production never queries.
func seedGraphCriterion(t *testing.T, store *graph.Store, key, statement, status, repo string) {
	t.Helper()
	sum := sha256.Sum256([]byte(key + "|" + statement + "|" + status))
	hash := hex.EncodeToString(sum[:])
	_, err := store.UpsertNode(&graph.Node{
		Type:   "Criterion",
		Domain: "engineering",
		Key:    key,
		Props: map[string]any{
			"ac_id":     key,
			"statement": statement,
			"status":    status,
			"parent":    strings.SplitN(key, ":", 2)[0],
		},
		Repo:        repo,
		ContentHash: hash,
	})
	if err != nil {
		t.Fatalf("seed criterion %s: %v", key, err)
	}
}

func getGraphCriterionStatus(t *testing.T, store *graph.Store, key string) string {
	t.Helper()
	n, err := store.GetNode("Criterion", key, "")
	if err != nil {
		t.Fatalf("GetNode(%s): %v", key, err)
	}
	s, _ := n.Props["status"].(string)
	return s
}

func TestVerify_LedgerWritebackToGraph(t *testing.T) {
	env := newTestEnv(t)

	specContent := `---
title: Writeback Test
type: feature
status: delivering
slug: writeback-test
---
# Writeback Test

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL handle X
- AC-2: THE SYSTEM SHALL handle Y
- AC-3: THE SYSTEM SHALL handle Z

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | handle X | DONE | implemented |
| 2 | handle Y | DONE | implemented |
| 3 | handle Z | DONE | implemented |

### Exercise-the-feature check

- [x] Exercised: confirmed all three handlers work
`
	env.addSpec("planning/features/writeback-test/spec.md", specContent)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/writeback-test/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "writeback-test", -1))
	env.indexAll()

	// Seed Criterion nodes in the graph so Record can find and flip them.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	seedGraphCriterion(t, store, "writeback-test:AC-1", "THE SYSTEM SHALL handle X", "proposed", gitutil.RepoKey(env.dir))
	seedGraphCriterion(t, store, "writeback-test:AC-2", "THE SYSTEM SHALL handle Y", "proposed", gitutil.RepoKey(env.dir))
	seedGraphCriterion(t, store, "writeback-test:AC-3", "THE SYSTEM SHALL handle Z", "proposed", gitutil.RepoKey(env.dir))
	store.Close()

	output, err := runCmd("spec", "verify", "--skip-tests", "--json", "writeback-test")
	if err != nil {
		t.Fatalf("verify failed: %v\noutput: %s", err, output)
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	if result.ACStatusUpdates != 3 {
		t.Errorf("ACStatusUpdates = %d, want 3", result.ACStatusUpdates)
	}

	// Verify the graph nodes are now "passing".
	store2, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("graph.Open (re-read): %v", err)
	}
	defer store2.Close()

	for _, key := range []string{"writeback-test:AC-1", "writeback-test:AC-2", "writeback-test:AC-3"} {
		if s := getGraphCriterionStatus(t, store2, key); s != "passing" {
			t.Errorf("Criterion %s status = %q, want passing", key, s)
		}
	}
}

func TestVerify_ArchiveMovesSiblingArtifacts(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/sibling-test/spec.md",
		strings.Replace(specWithLedgerAndAudit, "test-feature", "sibling-test", -1))
	dir := filepath.Join(env.heroDir, "planning/features/sibling-test")
	writeVerifyFile(t, filepath.Join(dir, "delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "sibling-test", -1))
	writeVerifyFile(t, filepath.Join(dir, "plan.md"), "# Plan\n")
	env.indexAll()

	if _, err := runCmd("spec", "verify", "--skip-tests", "sibling-test"); err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	specDir := filepath.Join(env.heroDir, "specs", "sibling-test")
	for _, f := range []string{"spec.md", "delivery-audit.md", "plan.md"} {
		if _, err := os.Stat(filepath.Join(specDir, f)); os.IsNotExist(err) {
			t.Errorf("%s was not moved with the spec to specs/sibling-test/", f)
		}
	}
	// Source planning dir must not survive as an orphan.
	if _, err := os.Stat(dir); err == nil {
		t.Error("planning spec dir survived archive — siblings orphaned")
	}
}

func TestVerify_InitiativeNotCompletedWithUnbuiltChildren(t *testing.T) {
	env := newTestEnv(t)

	// Initiative declares two children (block-style list) but only one is
	// materialized. Completing it must NOT auto-complete the initiative.
	initiativeContent := `---
title: Multi Child Init
type: initiative
status: planning
slug: multi-child-init
child:
  - built-child
  - unbuilt-child
---
# Multi Child Init
`
	env.addSpec("planning/initiatives/multi-child-init/spec.md", initiativeContent)

	builtChild := `---
title: Built Child
type: feature
status: delivering
slug: built-child
parent: multi-child-init
---
# Built Child

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL do the built work

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | do the built work | DONE | implemented |

### Exercise-the-feature check

- [x] Exercised: confirmed built child works
`
	env.addSpec("planning/features/built-child/spec.md", builtChild)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/built-child/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "built-child", -1))
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "built-child")
	if err != nil {
		t.Fatalf("verify built-child failed: %v\noutput: %s", err, output)
	}
	if strings.Contains(output, "auto-completed") {
		t.Errorf("initiative wrongly auto-completed with an unbuilt declared child:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(env.heroDir, "specs", "multi-child-init", "spec.md")); err == nil {
		t.Error("initiative archived despite an unbuilt declared child")
	}
	if _, err := os.Stat(filepath.Join(env.heroDir, "planning/initiatives/multi-child-init/spec.md")); os.IsNotExist(err) {
		t.Error("initiative disappeared from planning")
	}
}

func TestVerify_InitiativeAutoComplete(t *testing.T) {
	env := newTestEnv(t)

	// Parent initiative.
	initiativeContent := `---
title: Parent Initiative
type: initiative
status: planning
slug: parent-init
---
# Parent Initiative
`
	env.addSpec("planning/initiatives/parent-init/spec.md", initiativeContent)

	// Child 1 — already completed and archived.
	child1Content := `---
title: Child One
type: feature
status: completed
slug: child-one
parent: parent-init
---
# Child One
`
	env.addSpec("specs/child-one/spec.md", child1Content)

	// Child 2 — delivering, about to be verified.
	child2Content := `---
title: Child Two
type: feature
status: delivering
slug: child-two
parent: parent-init
---
# Child Two

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL do child-two work

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | do child-two work | DONE | implemented |

### Exercise-the-feature check

- [x] Exercised: verified child two works
`
	env.addSpec("planning/features/child-two/spec.md", child2Content)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/child-two/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "child-two", -1))
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "--json", "child-two")
	if err != nil {
		t.Fatalf("verify failed: %v\noutput: %s", err, output)
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	if result.InitiativeCompleted != "parent-init" {
		t.Errorf("InitiativeCompleted = %q, want parent-init", result.InitiativeCompleted)
	}

	// Parent initiative should be archived.
	archivedPath := filepath.Join(env.heroDir, "specs", "parent-init", "spec.md")
	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Error("parent initiative was not auto-archived")
	}
}

// TestVerify_InitiativeAutoComplete_FlowStyleRelations is a regression test
// for a bug where content-remediation's 8th and final child
// (token-efficiency-pass) verified and archived cleanly but never
// auto-completed the parent initiative. Root cause: children declared their
// parent link with inline-flow YAML (`relations: [{target: x, kind:
// parent}]`, the shape /design templates emit) while the initiative declared
// its roster with a block-style `child:` list — both valid YAML, but at the
// time this ran, spec.go's relation parser only handled block-style
// `key: value` relation entries and silently dropped inline-flow ones. With
// zero parsed "parent" relations on the just-verified child,
// autoCompleteParentIfReady found nothing to check and returned immediately.
func TestVerify_InitiativeAutoComplete_FlowStyleRelations(t *testing.T) {
	env := newTestEnv(t)

	// Parent initiative with a block-style `child:` roster (not `relations:`).
	initiativeContent := `---
title: Content Remediation
type: initiative
status: planning
slug: content-remediation
child:
  - child-one
  - child-two
---
# Content Remediation
`
	env.addSpec("planning/initiatives/content-remediation/spec.md", initiativeContent)

	// Child 1 — already completed and archived, parent link in inline-flow style.
	child1Content := `---
title: Child One
type: bug
status: completed
slug: child-one
relations:
  - { target: content-remediation, kind: parent }
---
# Child One
`
	env.addSpec("specs/child-one/spec.md", child1Content)

	// Child 2 — delivering, about to be verified. Parent link also inline-flow.
	child2Content := `---
title: Child Two
type: enhancement
status: delivering
slug: child-two
relations:
  - { target: content-remediation, kind: parent }
---
# Child Two

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL do child-two work

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | do child-two work | DONE | implemented |

### Exercise-the-feature check

- [x] Exercised: verified child two works
`
	env.addSpec("planning/features/child-two/spec.md", child2Content)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/child-two/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "child-two", -1))
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "--json", "child-two")
	if err != nil {
		t.Fatalf("verify failed: %v\noutput: %s", err, output)
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	if result.InitiativeCompleted != "content-remediation" {
		t.Errorf("InitiativeCompleted = %q, want content-remediation", result.InitiativeCompleted)
	}

	archivedPath := filepath.Join(env.heroDir, "specs", "content-remediation", "spec.md")
	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Error("parent initiative was not auto-archived")
	}
}

func TestVerify_ExerciseDemotedToAdvisory(t *testing.T) {
	env := newTestEnv(t)

	// All ACs DONE but exercise-the-feature NOT checked.
	specContent := `---
title: Exercise Advisory
type: feature
status: delivering
slug: exercise-advisory
---
# Exercise Advisory

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL do X

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | do X | DONE | implemented |

### Exercise-the-feature check

- [ ] not exercised
`
	env.addSpec("planning/features/exercise-advisory/spec.md", specContent)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/exercise-advisory/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "exercise-advisory", -1))
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "--json", "exercise-advisory")
	if err != nil {
		t.Fatalf("verify should PASS with exercise unchecked (advisory only): %v\noutput: %s", err, output)
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	if result.Result != "PASS" {
		t.Errorf("Result = %q, want PASS (exercise is advisory)", result.Result)
	}

	// Gate 1 should PASS, and include the advisory note.
	for _, g := range result.Gates {
		if g.Name == "Completion Ledger" {
			if g.Result != "PASS" {
				t.Errorf("Completion Ledger gate Result = %q, want PASS", g.Result)
			}
			hasAdvisory := false
			for _, d := range g.Details {
				if strings.Contains(d, "ADVISORY") && strings.Contains(d, "exercise") {
					hasAdvisory = true
				}
			}
			if !hasAdvisory {
				t.Errorf("expected ADVISORY note about exercise in gate details, got: %v", g.Details)
			}
		}
	}
}

func TestVerify_ACKeyMismatchResilience(t *testing.T) {
	env := newTestEnv(t)

	// Ledger has 5 ACs, but graph only has 3 Criterion nodes.
	specContent := `---
title: Mismatch Test
type: feature
status: delivering
slug: mismatch-test
---
# Mismatch Test

## Acceptance Criteria

- AC-1: handle A
- AC-2: handle B
- AC-3: handle C
- AC-4: handle D
- AC-5: handle E

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | handle A | DONE | implemented |
| 2 | handle B | DONE | implemented |
| 3 | handle C | DONE | implemented |
| 4 | handle D | DONE | implemented |
| 5 | handle E | DONE | implemented |

### Exercise-the-feature check

- [x] Exercised: all five work
`
	env.addSpec("planning/features/mismatch-test/spec.md", specContent)
	writeVerifyFile(t, filepath.Join(env.heroDir, "planning/features/mismatch-test/delivery-audit.md"),
		strings.Replace(auditReportShip, "test-feature", "mismatch-test", -1))
	env.indexAll()

	// Only seed 3 of the 5 criteria in the graph.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	seedGraphCriterion(t, store, "mismatch-test:AC-1", "handle A", "proposed", gitutil.RepoKey(env.dir))
	seedGraphCriterion(t, store, "mismatch-test:AC-3", "handle C", "proposed", gitutil.RepoKey(env.dir))
	seedGraphCriterion(t, store, "mismatch-test:AC-5", "handle E", "proposed", gitutil.RepoKey(env.dir))
	store.Close()

	output, err := runCmd("spec", "verify", "--skip-tests", "--json", "mismatch-test")
	if err != nil {
		t.Fatalf("verify failed: %v\noutput: %s", err, output)
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	// Only the 3 existing criteria should be flipped; 2 unknown → no error.
	if result.ACStatusUpdates != 3 {
		t.Errorf("ACStatusUpdates = %d, want 3 (2 missing from graph should be silently skipped)", result.ACStatusUpdates)
	}

	if result.Result != "PASS" {
		t.Errorf("Result = %q, want PASS", result.Result)
	}

	// Verify the 3 existing ones are passing.
	store2, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	defer store2.Close()
	for _, key := range []string{"mismatch-test:AC-1", "mismatch-test:AC-3", "mismatch-test:AC-5"} {
		if s := getGraphCriterionStatus(t, store2, key); s != "passing" {
			t.Errorf("Criterion %s status = %q, want passing", key, s)
		}
	}
}

// verifyInitiativeWithChild is an initiative spec whose Children table lists an
// unmaterialized child (no spec.md on disk for it).
const verifyInitiativeWithChild = `---
title: Retrieval Quality
type: initiative
status: delivering
slug: retrieval-quality
---
# Retrieval Quality

## Children

| Slug | Title | Priority |
|---|---|---|
| configurable-reranking | Configurable reranking | P1 |
| query-expansion | Query expansion | P1 |
`

func TestVerify_UnmaterializedInitiativeChild(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/retrieval-quality/spec.md", verifyInitiativeWithChild)
	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "configurable-reranking")
	if err == nil {
		t.Fatal("expected verify to error on unmaterialized child slug")
	}
	msg := err.Error()
	if !strings.Contains(msg, "retrieval-quality") {
		t.Errorf("error %q should name the owning initiative", msg)
	}
	if !strings.Contains(msg, "/design") {
		t.Errorf("error %q should direct the user to /design", msg)
	}
}

// TestCheckReconcile_CompletesArchivedChildrenInitiative is the inverse of
// TestVerify_UnmaterializedInitiativeChild: an initiative whose block-style
// `child:` roster is fully completed AND already archived under specs/**, with
// no child verified in the current process, is completed and archived by the
// standalone `hero check --reconcile` re-check. Re-running the re-check is a
// clean no-op (idempotency).
func TestCheckReconcile_CompletesArchivedChildrenInitiative(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("planning/initiatives/content-remediation/spec.md", `---
title: Content Remediation
type: initiative
status: planning
slug: content-remediation
child:
  - child-one
  - child-two
---
# Content Remediation
`)
	for _, slug := range []string{"child-one", "child-two"} {
		env.addSpec("specs/"+slug+"/spec.md", `---
title: `+slug+`
type: bug
status: completed
slug: `+slug+`
relations:
  - { target: content-remediation, kind: parent }
---
# `+slug+`
`)
	}
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "stranded initiative")
	env.indexAll()

	// Dry run (no --reconcile) must only report, not mutate.
	out, err := runCmd("check")
	if err != nil {
		t.Fatalf("check (dry-run) errored: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(env.heroDir, "specs", "content-remediation", "spec.md")); !os.IsNotExist(statErr) {
		t.Fatal("dry-run check must not archive the initiative")
	}
	planningPath := filepath.Join(env.heroDir, "planning", "initiatives", "content-remediation", "spec.md")
	if _, statErr := os.Stat(planningPath); statErr != nil {
		t.Fatalf("dry-run check must leave the initiative in planning/: %v", statErr)
	}

	// Apply.
	out, err = runCmd("check", "--reconcile")
	if err != nil {
		t.Fatalf("check --reconcile errored: %v\n%s", err, out)
	}

	archivedPath := filepath.Join(env.heroDir, "specs", "content-remediation", "spec.md")
	data, statErr := os.ReadFile(archivedPath)
	if statErr != nil {
		t.Fatalf("initiative was not completed + archived to specs/: %v", statErr)
	}
	body := string(data)
	if !strings.Contains(body, "status: completed") {
		t.Errorf("archived initiative should be status: completed:\n%s", body)
	}
	if _, statErr := os.Stat(planningPath); !os.IsNotExist(statErr) {
		t.Error("initiative should no longer exist under planning/")
	}

	// Idempotency: re-running is a clean no-op (no error, no double-archive).
	env.indexAll()
	out, err = runCmd("check", "--reconcile")
	if err != nil {
		t.Fatalf("second check --reconcile errored: %v\n%s", err, out)
	}
	if strings.Contains(out, "completed + archived") {
		t.Errorf("second reconcile should not re-complete the initiative:\n%s", out)
	}
}

// TestCheckReconcile_ClearsOrphanCompletedAt proves the status ↔ completed_at
// invariant repair end-to-end: a genuinely reopened spec (completed_at set,
// status planning) has its orphaned timestamp cleared by `hero check
// --reconcile`.
func TestCheckReconcile_ClearsOrphanCompletedAt(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	specPath := "planning/features/half-restored/spec.md"
	env.addSpec(specPath, `---
title: Half Restored
type: feature
status: planning
slug: half-restored
completed_at: 2026-06-23T19:57:04Z
---
# Half Restored
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "orphaned completed_at")
	env.indexAll()

	out, err := runCmd("check", "--reconcile")
	if err != nil {
		t.Fatalf("check --reconcile errored: %v\n%s", err, out)
	}

	data, readErr := os.ReadFile(filepath.Join(env.heroDir, specPath))
	if readErr != nil {
		t.Fatal(readErr)
	}
	body := string(data)
	if strings.Contains(body, "completed_at") {
		t.Errorf("orphaned completed_at should be cleared:\n%s", body)
	}
	if !strings.Contains(body, "status: planning") {
		t.Errorf("status should remain planning:\n%s", body)
	}
}

func TestVerify_NoSignalBareMessage(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/retrieval-quality/spec.md", verifyInitiativeWithChild)
	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "totally-unrelated-xyz")
	if err == nil {
		t.Fatal("expected verify to error on unknown slug")
	}
	want := `spec "totally-unrelated-xyz" not found`
	if err.Error() != want {
		t.Errorf("error = %q, want exactly %q", err.Error(), want)
	}
}
