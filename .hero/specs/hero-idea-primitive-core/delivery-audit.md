# Delivery audit — hero-idea-primitive-core

**Audited:** `git diff 3aaad62^ -- core/spec-types/intake.md internal/spec/spec.go internal/spec/select.go internal/spec/intake_test.go internal/spec/graph_ingest.go internal/spec/graph_ingest_test.go internal/cli/intake.go internal/cli/intake_test.go internal/cli/status.go internal/cli/pipeline.go internal/cli/deliver.go internal/synthesize/detect.go internal/snapshot/rollup.go internal/snapshot/snapshot_test.go internal/cli/root.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Core discovers intake as `TypeIntake`; search/list can filter it — `internal/spec/spec.go` adds `TypeIntake` and `/intake/` path recognition; `TestParseHonorsIntakeFrontmatterType` and `TestTypeFromPath` assert modeling, while the generic type filters accept the modeled value.
- [✓] Committed-work rollups exclude intake through `IsWorkSpec()` — `internal/spec/select.go`, `internal/cli/status.go`, `internal/cli/pipeline.go`, `internal/cli/deliver.go`, `internal/synthesize/detect.go`, and `internal/snapshot/rollup.go` use the canonical allow-list; `TestIntakeNotReady` and `TestBuildAssignmentsUseCommittedWorkAllowList` cover exclusion.
- [✓] Planning intake appears only in the status pre-commitment listing — `internal/cli/status.go` segregates `IsPreCommitment()` before work buckets; `TestIntakeAbsentFromStatusWorkBuckets` asserts the CLI surface.
- [✓] `hero intake "<text>"` scaffolds a valid intake — `internal/cli/intake.go` writes `.hero/planning/intake/<slug>/spec.md` with `type: intake` and `status: planning`; `TestIntakeCaptureCreatesSpec` asserts the artifact, and the supplied end-to-end exercise captured an intake with a built binary.
- [✓] Promotion creates a roadmap spec, both provenance relations, and terminal state — `runIntakePromote` creates feature/bug specs, writes `derived_from` and `promotes_to`, and sets `status: promoted`; `TestIntakePromoteCreatesFeatureWithProvenance`, `TestIntakePromoteBugType`, and the supplied end-to-end exercise cover the behavior.
- [✓] Reject sets terminal `rejected` — `runIntakeReject` durably updates the intake; `TestIntakeReject` asserts the result.
- [✓] `hero why` can traverse promoted-spec provenance to intake — `internal/spec/graph_ingest.go` materializes `derived_from` and resolves same-slug targets away from the declaring feature; `TestSpecWriteGraphIntakeProvenance` proves the edge targets the Intake node rather than self-looping.
- [✓] `hero_queue` never returns intake — `IsReady` rejects everything except committed work and initiatives; `TestIntakeNotReady` asserts a planning intake is never ready.
- [✓] Existing work, initiative, and knowledge rollups remain unchanged — explicit initiative branches preserve container behavior; `TestIntakeNotReady`, `TestBuildAssignmentsUseCommittedWorkAllowList`, and snapshot regressions cover the allow-list boundary; supplied `go test ./...` and `go vet ./...` evidence is green.

## Changes
- [✓] Model intake type, statuses, path, and predicate — implemented in `internal/spec/spec.go` with predicate/model tests in `internal/spec/intake_test.go`.
- [✓] Route selection through the committed-work allow-list — `internal/spec/select.go` gates readiness with `IsWorkSpec()` while preserving initiatives; `TestIntakeNotReady` covers feature, initiative, note, and intake cases.
- [✓] Segregate status intake and gate work buckets canonically — `internal/cli/status.go` has a dedicated pre-commitment collection and section; CLI regression asserts placement.
- [✓] Convert pipeline, deliver, synthesize, and snapshot rollups — all named sites call `IsWorkSpec()`; snapshot regression directly excludes intake and knowledge while retaining committed work.
- [✓] Add intake capture/promote/reject/list CLI — new `internal/cli/intake.go` implements all four verbs and `internal/cli/root.go` registers the command; CLI tests cover each transition and listing.
- [✓] Preserve MCP search/why visibility and queue exclusion — discovery/type filtering remains generic, readiness excludes intake, and graph ingest adds correctly directed intake provenance edges.
- [✓] Reconcile intake lifecycle documentation — `core/spec-types/intake.md` now documents `planning → triaged → promoted | rejected | merged`, matching engine constants.
- [✓] Add modeling, no-leak, CLI, provenance, and regression tests — new/expanded tests exist in `internal/spec/intake_test.go`, `internal/cli/intake_test.go`, `internal/spec/graph_ingest_test.go`, and `internal/snapshot/snapshot_test.go`; supplied focused and full-suite evidence passes.

## Open items (if any)
- None.

## Audit notes
- No Completion Ledger rows were downgraded. The repaired graph resolver and its same-slug provenance assertions close the prior ambiguity around promoted feature → intake traversal.
