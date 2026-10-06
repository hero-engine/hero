# Delivery audit — work-item-model

**Audited:** `git show 73d787be` (branch `feat/hero-read-contract`)
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: one item per work spec, knowledge excluded, path order. `IsWorkItem` and `Build` are in `internal/workmodel/model.go`. A decision counts only when its parent resolves to an initiative. The test is `TestBuildIncludesOnlyWorkSpecsInPathOrder`: 11 included, a convention and a parentless decision excluded, path order and repo-relative path asserted.
- [~] AC-2: one lane per item by first-match rules, honoring unmet `depends-on`/`blocks`. The lane order in `Lane` (model.go) matches `read-contract-v1`, and `TestLanes` and `TestRecentDaysWindow` assert all five lanes. **Gap:** at model.go:248, `UnmetDeps` only accepts `r.Kind == "depends-on"`. A `relations:` block entry with `kind: depends_on` keeps its raw kind (`spec.normalizeRelation` only defaults an empty kind), so it is silently ignored. The spec's Design says "Relation kinds `depends-on`/`depends_on` and `blocks` count as dependency edges." `internal/projection` (the rule the contract cites, via `graph_ingest.go:262`) treats both as edges. The real corpus has this form in `.hero/specs/contentfs-legacy-fallback-removal/spec.md`. No test covers `blocks`, `depends_on`, or a missing target.
- [✓] AC-3: verify state and audit from the validated report, null for decisions. `VerifyOf` uses `spec.FindAuditReport` (`Found` only), `ParseLedger`, and the status. `TestVerifyState` covers passed, partial (no audit), failed (HOLD and regressed), not_run, and a null decision. It does not cover a ledger with SKIPPED/BLOCKED rows.
- [~] AC-4: priority/severity normalization and initiative progress. Progress (`initiativeProgress`) is correct and tested (1/3). The normalization does not match the spec. The spec says P0|critical, P1|high, P2|medium|moderate, P3|low, "anything else null". `normalizeLevel` (model.go:370-376) also maps `blocker`, `highest`, `normal`, `p4`, `lowest`, `minor` and `trivial`, which the spec does not define. Severity `minor` becomes `low` while `major` becomes null, which is inconsistent. The ledger does not mention this deviation.
- [✓] AC-5: revision changes with file, related status or audit, and is otherwise stable. In `revision.go`, SHA-256[:16] covers the file bytes, the sorted `slug:status` of relations and declared children (missing targets become `missing`), and the audit verdict plus its mtime in ns. `TestRevisionChangesOnlyWithInputs` asserts this. An independent run over the real 513-item corpus gave identical revisions across two builds.

## Changes
- [✓] `internal/workmodel/model.go`: types, `Build`, normalization, lanes, verify, progress. Also `BuildOne`, `Corpus`/`Lookup`, `NextStep`/`Phase`/`Extra`, and `Options.Root`, all disclosed in the ledger.
- [✓] `internal/workmodel/revision.go`: item revision as specified.
- [✓] `internal/workmodel/model_test.go`: 6 tests over a real `spec.Discover` temp corpus. `go test ./internal/workmodel -count=1 -race` passes and `go vet` is clean (re-run by the auditor). `tests.log` matches. `vet.log` is empty, meaning clean.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **HOLD blockers (both small):**
  1. `UnmetDeps` must also accept `depends_on`, as the spec and projection parity require. Add a test for `blocks`, for `depends_on` in a `relations:` block, and for a missing target.
  2. Either restrict `normalizeLevel` to the spec's mapping, or amend the spec to name the extra synonyms and decide what `major` maps to.
- **Contract-level risk (implemented faithfully, but it needs a decision before `hero-work-tool`):** `FindAuditReport` rejects any report older than the spec file's mtime. On the real corpus, 164 finished specs have such a stale audit. Build yields verify `passed` 4, `partial` 404, `not_run` 102. Under the `next-step-engine` table, almost every completed spec would show "Delivered?" with a Verify action. File mtimes are also not stable across clones and checkouts. `read-contract-v1` owns this behaviour, so it is not a delivery defect here. It should go back to the decision.
- **Revision vs time-based lanes:** the lane depends on `Now`/`RecentDays` (recently_done → none), but the revision does not. An item can change lanes with an identical revision, so a client caching per item revision will show a stale lane. This matches the contract's literal revision definition. Flag it for the global-revision and `hero-work-tool` design.
- **Contract gap:** a spec that is not finished but has a SHIP audit gets `{state: not_run, audit: "ship"}`. The contract defines not_run as "with no audit", so this combination is undefined.
- **Contract gap:** "partial ... or some AC results are unknown" is not implemented. Verify never consults AC results. That is defensible, because the contract also says verify comes only from files on disk, but the contract contradicts itself here.
- **Edge cases checked:**
  - Missing `CompletedAt` falls back to `ModifiedAt` (model.go:279). Untested.
  - An initiative without children gets `progress {0,0}`, not designed, so lane `none` while in planning. Acceptable.
  - A decision whose parent is missing or not an initiative is excluded. Tested.
  - Cycles: no recursion anywhere (all lookups are one hop), so they are safe.
  - Superseded is `none`. Rejected and merged are unfinished non-planning statuses, so they fall to `none`.
- **Minor:**
  - Relation targets are not passed through `normalizeRelTarget` the way `DeclaredChildren` does, so a path-style `depends-on` target counts as permanently unmet and as `missing` in the revision.
  - `ParentSlug` does not handle a raw `child_of` kind from a `relations:` block.
  - `created_at`/`updated_at` emit `""` rather than null when zero (0 cases in the real corpus).
  - A nil `NextStep.Extras` serializes as `null`, not `[]`. That is the engine's concern.
- The working tree has untracked `internal/workmodel/next.go`, `next_test.go` and `core/commands/verify.md` from later work. They are not part of 73d787be. Package tests pass with them present.
