# Delivery audit — initiative-autocomplete-misses-completed-children

**Audited:** working tree vs HEAD — `git diff -- internal/cli/{check,complete,verify,verify_test}.go internal/reconcile/{reconcile,reconcile_test}.go internal/spec/spec.go` + new file `internal/spec/initiative_complete.go`
**Verdict:** SHIP
**Surface:** noteworthy
**Scope note:** `.hero/**` spec-move churn was explicitly excluded from this audit (separate reconcile-data concern). Code fix only.

## Acceptance criteria

- [✓] Standalone re-check completes an initiative whose block-style `child:` roster is all completed+archived, no child verified in-process — `internal/reconcile/reconcile.go:95-108` (FindingInitiativeComplete) + `internal/cli/check.go:215-224` (apply via `completeAndArchive`). Test: `TestCheckReconcile_CompletesArchivedChildrenInitiative` (cli) builds exactly this end-state and asserts archive to `specs/content-remediation/spec.md` with `status: completed`. Would fail if the fix were reverted.
- [✓] Idempotent command; re-run is a safe no-op — `InitiativeReadyToComplete` returns false for a completed initiative (`initiative_complete.go:26-28`). Tests: `TestReconcile_InitiativeComplete_Idempotent` + the idempotency tail of `TestCheckReconcile_CompletesArchivedChildrenInitiative` (asserts second reconcile prints no "completed + archived").
- [✓] Premature-completion guard preserved — roster gate `declaredCount>0 && !declaredComplete → false` (`initiative_complete.go:50-52`). Tests: `TestVerify_UnmaterializedInitiativeChild` and `TestVerify_InitiativeAutoComplete_FlowStyleRelations` still exist (verify_test.go:956, :707) and pass; NEW `TestReconcile_NegativeGuard_UnmaterializedChild` proves the reconcile path also declines.
- [✓] No spec left with `completed_at` set while `status != completed`; reconcile detects + repairs — `reconcile.go:111-124` (FindingOrphanCompletedAt) + `check.go:225-233` clears via `clearCompletedAt`. Tests: `TestReconcile_OrphanCompletedAt`, `TestCheckReconcile_ClearsOrphanCompletedAt` (asserts `completed_at` gone, `status: planning` retained).
- [✓] Inverse-case regression test alongside the premature guard — `TestReconcile_InitiativeCompleteFromArchivedChildren` (reconcile) + `TestCheckReconcile_CompletesArchivedChildrenInitiative` (cli). Genuine: fully-completed-archived roster, no in-process child, asserts completion.

## Changes (fix items 1–3)

- [✓] Fix 1 — extract shared predicate + wire into reconcile — `internal/spec/initiative_complete.go` (`InitiativeReadyToComplete`); called by BOTH `autoCompleteParentIfReady` (verify.go:647) and reconcile (reconcile.go:102). The two paths cannot diverge.
- [✓] Fix 2 — `status` ↔ `completed_at` invariant — detection (reconcile.go:111-124) + repair (`clearCompletedAt` complete.go:314-330, `spec.ClearFrontmatterField` spec.go:1791-1824). Precedence correct: initiative-complete branch `continue`s before the orphan branch, so a completable initiative completes (stamping both) rather than merely clearing.
- [~] Fix 3 — reconcile the two stranded initiatives on disk — OUT OF AUDIT SCOPE per invocation (`.hero/**` data moves handled separately). Not assessed.

## Audit notes

- **Report-only vs apply split is clean.** `Reconcile` never touches disk (doc comment reconcile.go:68 "The function never modifies any files"; it only appends findings). All mutation is gated behind `checkReconcile && f.CanAutoFix()` in check.go:203. The dry-run leg of `TestCheckReconcile_CompletesArchivedChildrenInitiative` asserts no archive + spec still under planning/. Confirmed report-only cannot complete/archive/clear.
- **Leaf safety confirmed.** Reconcile completion fires only under `s.Type == spec.TypeInitiative` (reconcile.go:100) AND `InitiativeReadyToComplete` re-checks `parent.Type != TypeInitiative → false` (initiative_complete.go:23). A leaf feature/bug can never be auto-completed from git/roster evidence.
- **In-process behavior preserved.** The inline gate (old verify.go:643-691) was lifted verbatim into the predicate; `parentSlug` → `parent.Slug` is equivalent (parent was found by `s.Slug == parentSlug`). One nuance: the predicate uses `normalizeRelTarget` where the old inline used `normalizeVerifyParentTarget`. They agree for the two real relation-target forms (bare slug; `.../slug/spec.md` path); they differ only for a slash-bearing target that does NOT end in `.md` (cross-repo qualifier) — not a form these relations take. `TestVerify_InitiativeAutoComplete_FlowStyleRelations` passes, confirming no behavior change on real inputs. Minor, not a defect.
- **`clearCompletedAt` is frontmatter-safe.** `ClearFrontmatterField` (spec.go) requires an opening/closing `---`, removes only a single top-level (`leadingSpaceCount==0`) scalar line matching `key:`, and returns content unchanged if absent. It cannot corrupt or drop block fields. Clears both `completed_at` and camelCase `completedAt`. `CompletedAt` is parsed from both spellings (spec.go:532), so `.IsZero()` detection is sound.
- **Dead code removed cleanly.** `Finding.NeedsMove` is deleted; `grep -rn NeedsMove` across the tree returns zero references. check.go now dispatches on the typed `f.Kind` switch instead.
- **No Completion Ledger in the spec.** The invocation said the engineer's `## Completion Ledger` is at the end of the spec — it is not present (spec ends at the Root Cause Analysis evidence index, line 320). There are no DONE rows to grade; ACs were audited directly against code + test assertions instead (all genuine). The missing delivery record is a process gap worth the user's note, not a code defect.

## Build / test evidence (run fresh from repo root)

- `go build ./...` — exit 0
- `go vet ./internal/cli/... ./internal/reconcile/... ./internal/spec/...` — exit 0
- `go test ./internal/cli/... ./internal/reconcile/... ./internal/spec/...` — all `ok`
- Named tests verified individually: 6 new + 2 preserved guards all PASS.
