---
type: delivery-audit
spec: master-ingest-restore
audited: 2026-06-09
verdict: SHIP
surface: noteworthy
confidence: high
auditor: claude-sonnet-4-6 (cold audit)
---

# Delivery Audit — master-ingest-restore

## Summary

All 8 ACs audited. Code, tests, and claims corroborate. Verdict: **SHIP**.
Two noteworthy observations (neither a blocker): `TestSmokeScan` is thin and
AC-4's happy-path (network + auth) has no automated coverage — both acknowledged
in the spec itself.

---

## AC Verdicts

### AC-1 — Note nodes from `knowledge/notes/*/spec.md`

**VERIFIED.** `internal/spec/graph_ingest.go:203` maps `TypeNote` → `"Note"`,
confirming `spec.WriteGraph` creates Note-typed graph nodes for any spec file
with `type: note` frontmatter. This was pre-existing, not new delivery work.
The spec correctly notes it was always passing.

### AC-2 — Memory nodes from `~/.claude/.../memory/`

**VERIFIED.** `memory.WriteGraph` exists at `internal/memory/graph_ingest.go:51`.
Wired at `scan.go:660–678` with a correct best-effort pattern: skips silently
when dir is absent (os.Stat check), emits a friendly skip reason when dir is
empty. Two unit tests confirm node-creation (`TestWriteGraph_UpsertsMemoryNodes`)
and idempotency (`TestWriteGraph_Idempotent`). Two integration-style scan tests
confirm the CLI behavior: `TestScanOmitsClaudeMemoryStepWhenAbsent` (no step
emitted), `TestScanEmitsFriendlyClaudeMemoryWhenEmpty` (step emitted with the
exact "Claude Code memory store for this project is empty" message). All four
tests pass.

### AC-3 — Tracker pull → Issue + Person nodes

**VERIFIED.** `tracker.PullAndWriteGraph` exists at `internal/tracker/pull.go:33`.
Wired at `scan.go:681–699`. Returns `Skipped+Reason` when tracker not configured
or token missing. `TestWriteIssuesGraph_UpsertsIssuesAndPersons` and
`TestWriteIssuesGraph_Idempotent` live in `internal/tracker/graph_ingest_test.go:183`
and `209` respectively; both found and confirmed. No end-to-end test of the skip
path in a scan test (coverage comes from unit tests + manual verification noted
in spec).

### AC-4 — Opportunistic team-server sync

**VERIFIED (with caveat).** `runOpportunisticTeamSync` exists in
`internal/cli/scan_team_sync.go:39`. Wired at `scan.go:702–714`. The skip
logic (no creds, no org_id) is correct. The rate-limiting logic (`shouldPull`,
`pullStaleAfter = 5m`) is unit-tested in `internal/cli/scan_team_sync_test.go`
with 4 cases (empty cursor, garbage cursor, recent, stale). The full
`runOpportunisticTeamSync` function — the push+pull happy path — has **no
automated test coverage**; the spec acknowledges "Two-machine convergence test
deferred until live cloud account is available." This is a **known gap, not a
surprise**, and the skip path (most runs) is implicitly exercised by every scan
test that runs without cloud creds. Acceptable for a best-effort best-effort enrichment
step, but leaves a coverage hole on the authenticated push/pull paths.

### AC-5 — Tier-2 LLM extraction auto-runs on scan

**VERIFIED.** `extract.RunAuto` exists at `internal/extract/auto.go:25`. Wired
at `scan.go:640–653`. Skips with a clear reason ("no ANTHROPIC_API_KEY") when key
is absent; returns `Skipped+Reason` rather than erroring. Idempotency is structural
(content-hash cache in `ExtractFromSource`). Tests in `internal/extract/` pass.
The API-key-present happy path requires a live model call and is appropriately
not unit-tested.

### AC-6 — "Graph ingest summary" block at end of scan

**VERIFIED.** `stepResult` and `ingestReport` are defined at `scan.go:423–462`
(within stated range). `ingestReport.print()` emits the block with three glyphs
(✅/⊘/❌) as described. `TestIngestReport_RendersAllOutcomes` exercises all
three outcomes in one report — verifying isolation — and checks for specific
strings including the error message surfaced inline. `TestIngestReport_EmptyPrintsNothing`
confirms no output on an empty report. Both tests pass and are in
`internal/cli/scan_report_test.go`.

### AC-7 — Idempotent on consecutive runs

**VERIFIED (structurally).** All ingest paths use content-hashed upserts:
`memory.WriteGraph` computes sha256 at `graph_ingest.go:80+`, `tracker.WriteIssuesGraph`
and `spec.WriteGraph` follow the same upsert pattern. `TestWriteGraph_Idempotent`
and `TestWriteIssuesGraph_Idempotent` confirm node IDs are stable on re-ingest.
Spec documents live-system verification (2186 nodes / 3752 edges → identical on
re-run). No automated full-scan idempotency test, but the per-path unit tests
cover the mechanism.

### AC-8 — Per-step failure isolation

**VERIFIED.** Each ingest step adds a `stepResult` to the shared `ingestReport`
independently; errors are not propagated (each step is wrapped in its own
`if err != nil { report.add(...failed...) } else { ... }` block). 
`TestIngestReport_RendersAllOutcomes` places ok + skipped + failed outcomes in
the same report and asserts all three appear in the rendered output — confirming
one failure cannot mask others. This is the correct test for the isolation
property.

---

## Noteworthy Observations

### 1. `TestSmokeScan` is thin

The ledger cites `TestSmokeScan` as scan test evidence, but the test only checks
that `hero scan --smoke` exits without error and prints "Scans the current
project" in the help text. It does not assert that any ingest step ran or that
the summary block was emitted. The meaningful AC-2 coverage comes from
`TestScanOmitsClaudeMemoryStepWhenAbsent` and `TestScanEmitsFriendlyClaudeMemoryWhenEmpty`
instead. The ledger's citation is slightly misleading but not incorrect —
`TestSmokeScan` does pass, just doesn't exercise the ingest behavior directly.

### 2. AC-4 happy-path (authenticated push/pull) has no automated test

The `runOpportunisticTeamSync` integration path is only reachable with real
cloud credentials. `TestShouldPull` covers the rate-limiting logic only.
This is explicitly flagged in the spec ("deferred until live cloud account
is available") and is consistent with the best-effort nature of the step.
A future improvement would be an interface-based mock for `graph.SyncClient`
that allows the push/pull paths to be unit-tested without network access.

---

## Test Runs (auditor-executed)

```
go test ./internal/cli/... -run ".*[Ss]can.*" -v
```
All scan-related tests pass: TestScanBasic, TestScanOmitsClaudeMemoryStepWhenAbsent,
TestScanEmitsFriendlyClaudeMemoryWhenEmpty, TestSmokeScan, TestIngestReport_RendersAllOutcomes,
TestIngestReport_EmptyPrintsNothing, TestScanNoHooksFlagAccepted, and others.
Result: PASS (1.457s)

```
go test ./internal/memory/... ./internal/tracker/... ./internal/extract/...
```
All pass (some cached).

---

## Files Audited

- `internal/cli/scan.go` — lines 423–462 (stepResult/ingestReport), 620–715 (ingest orchestration)
- `internal/cli/scan_report_test.go` — TestIngestReport_RendersAllOutcomes, TestIngestReport_EmptyPrintsNothing
- `internal/cli/scan_test.go` — TestScanOmitsClaudeMemoryStepWhenAbsent, TestScanEmitsFriendlyClaudeMemoryWhenEmpty
- `internal/cli/smoke_test.go` — TestSmokeScan
- `internal/cli/scan_team_sync.go` — runOpportunisticTeamSync, shouldPull
- `internal/cli/scan_team_sync_test.go` — TestShouldPull
- `internal/memory/graph_ingest.go` — WriteGraph
- `internal/memory/graph_ingest_test.go` — TestWriteGraph_UpsertsMemoryNodes, TestWriteGraph_Idempotent
- `internal/tracker/pull.go` — PullAndWriteGraph
- `internal/tracker/graph_ingest_test.go` — TestWriteIssuesGraph_UpsertsIssuesAndPersons, TestWriteIssuesGraph_Idempotent
- `internal/extract/auto.go` — RunAuto
- `internal/spec/graph_ingest.go` — TypeNote → "Note" mapping
