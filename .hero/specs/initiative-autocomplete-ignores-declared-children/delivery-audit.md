# Delivery audit — initiative-autocomplete-ignores-declared-children

**Audited:** `git diff HEAD -- internal/` plus four untracked files (`internal/spec/declared_children.go`, `relation_keys.go`, and their tests). Work is uncommitted.
**Verdict:** SHIP
**Surface:** noteworthy

## Independent falsification

The auditor did not take the ledger's word. A detached worktree at `HEAD` was
built to `/tmp/hero-head` and run against a scratch workspace reproducing the
reported shape (`hero-ops-governance` declaring four children via `children:`,
only `blast-radius-tiers` on disk and completed):

| | pre-fix (`HEAD`) | post-fix (working tree) |
|---|---|---|
| `hero goal hero-ops-governance --check` | `"verdict": "done"`, `"remaining": null` | `"verdict": "continue"`, `"action": "design"`, `remaining` = the 3 unbuilt children |
| `hero check --reconcile` | auto-completed **and archived** the initiative — `planning ✓ completed  every declared child is completed` | no completion finding; initiative left at `status: planning` |

The reported defect reproduces on `HEAD` and does not reproduce on the change.

## Acceptance criteria

- [✓] **AC-1** `children:` (plural, inline and block) forms `child` edges — `internal/spec/spec.go:639-660` adds the key to the relation case and normalizes to `child`. Tests `TestParsePluralChildrenInline`, `TestParsePluralChildrenBlock` (the block test asserts relation-for-relation equality against the singular `child:` form, not just a count).
- [✓] **AC-2** No auto-complete / no message while a declared child is unbuilt — `internal/spec/initiative_complete.go:47-52` returns false for any `DeclaredChildren` slug absent from `statusBySlug` or not finished. `internal/cli/verify.go:196` prints only when `result.InitiativeCompleted != ""`, which is set only when the predicate passes (`verify.go:182,647`), so the message is structurally gated. Test `TestInitiativeBlockedByUnscaffoldedChild` covers 1-of-4, 2-of-4, and materialized-but-`delivering`. Reproduced live (table above).
- [✓] **AC-3** Completes exactly once on a full roster — `initiative_complete.go:22-28` returns false for an already-completed initiative, making the fire one-shot. Test `TestInitiativeCompletesWhenFullRosterDelivered` blocks on a table-only child, completes when it lands, and refuses to re-complete.
- [✓] **AC-4** One shared roster function — verified by grep, not by claim: `spec.DeclaredChildren` (`internal/spec/declared_children.go:32`) is the only definition; `childLinkRe` exists exactly once (`declared_children.go:13`) and is gone from `internal/drive`; `initiative_complete.go:47` and `drive/stage.go:71` are its only non-test callers, and `drive/check.go:228 buildIntended` reaches it through `declaredChildSlugs`. Test `TestCheckAndCompletionGateAgreeOnRoster` asserts the equivalence across 5 roster shapes rather than asserting by inspection.
- [✓] **AC-5** `hero goal --check` reports unscaffolded declared children as remaining — test `TestCheckHonorsFrontmatterDeclaredChildren` (asserts not-done, `Remaining == [financial-action-gate]`, `Action == ActionDesign`). Reproduced live by the auditor with a real binary.
- [✓] **AC-6** `hero check` warns on near-miss relation keys — classifier `spec.NearMissRelationKey` (`internal/spec/relation_keys.go:61`), backed by `Spec.UnknownKeys` (`internal/spec/spec.go:216-220,757`), surfaced at `internal/cli/check.go:534-555`. Verified live against this repo's 338-spec corpus: fires on exactly 4 specs (`superseded-by:` ×1, `related:` ×3), all genuine silent drops. See Audit notes for the untested CLI seam.
- [✓] **AC-7** Singular `child:` regression guard is real — `TestInitiativeSingularChildBehaviorUnchanged` asserts both directions (blocks with an unfinished child, completes when full). Independently, `internal/drive/stage_test.go` was **not** modified and still exercises `declaredChildSlugs` against the relocated regex, which is the relocation regression guard the spec's Risks section asked for.
- [✓] **AC-8** Reconcile and verify share the predicate — `internal/reconcile/reconcile.go:102` and `internal/cli/verify.go:647` are the only callers of `spec.InitiativeReadyToComplete`; neither call site changed. Test `TestReconcile_NegativeGuard_PluralChildrenRoster`. Reproduced live (`check --reconcile` left the initiative at `planning`).

## Changes

- [✓] `internal/spec/spec.go` — `children`/`child-of`/`child_of` accepted at the relation case; `Spec.UnknownKeys` added and populated in the `default` arm only when the key is not tracker-prefixed.
- [✓] `internal/spec/initiative_complete.go` — roster gate rewritten to iterate `DeclaredChildren`; new `childFinished` helper shared by both gates.
- [✓] `internal/spec/declared_children.go` (new) — `DeclaredChildren` + relocated `childLinkRe`; regex is byte-identical to the one removed from drive (diffed).
- [✓] `internal/drive/stage.go` — `declaredChildSlugs` is now a one-line delegation; `regexp` import removed. `internal/drive/check.go` needed no edit: `buildIntended:228` already routed through `declaredChildSlugs`, so it consumes the shared roster transitively. Ledger states this honestly rather than claiming an edit.
- [✓] `internal/cli/check.go` — near-miss report section (`:534-555`) + collector (`:824`); adds a `relation-key-near-miss` health row. Structure mirrors the adjacent `wikilink-edges` block exactly, including its `err == nil &&` / else-pass shape — matching house style, not a new defect.
- [~] Tests — every Validation-table row has a test **except** the `hero check` row, which is covered at the classifier level only (see Audit notes).

## Open items

None. No PARTIAL, SKIPPED, or BLOCKED rows in the ledger. No performative `DONE`
rows found: every cited file/line was checked and resolves to the claimed code
(two citations are off by a handful of lines — `UnknownKeys` is at `:216-220`,
not `:236-241`; the check section starts at `:534`, not `:531` — cosmetic, not
performative).

## Audit notes

1. **AC-6's CLI wiring has no automated test.** The classifier is well tested
   (`TestNearMissRelationKey`, with two negative sets), but the ~49 lines added to
   `internal/cli/check.go` are exercised only by the ledger's manual run. The
   precedent exists and was not followed: `internal/cli/check_test.go:29
   TestCheck_WikilinkEdgeWarning` is the analogous CLI-level test for the
   immediately-adjacent warning block. The ledger's Changes row 6 — "Every row in
   the Validation table is covered" — overstates on this one row, since the
   Validation row is written at the `hero check` level. Downgraded that row to `~`;
   AC-6 itself stays `✓` because the auditor reproduced the warning live.

2. **`declared_children.go:47` keeps an inverted `child-of` reading, and the
   unification now propagates it to drive.** `DeclaredChildren` treats
   `r.Kind == "child-of"` on the initiative as a *declared child*. That contradicts
   the engineer's own (correct, verified) deviation-1 argument: every consumer in
   the tree — `cli/verify.go:626`, `snapshot/rollup.go:315,758`,
   `snapshot/release.go:56`, `initiative_complete.go:60` — reads the `child-of`
   *kind* as "X is my parent." The arm is carried forward from the old gate, so
   the completion path has no regression (and it can only over-block, never
   re-open the safety hole). But `drive.declaredChildSlugs` did **not** read
   frontmatter relations before, and now does: a sub-initiative declaring
   `relations: - kind: child-of` would list its own parent as an intended child in
   `buildIntended`, so `hero goal --check` could never reach done. `kind: child-of`
   is present in this repo's real corpus (`.hero/specs/two-tier-mcp-responses`,
   `.hero/specs/peer-call-result-yaml-int-strict-parse`) — both leaves, so not
   reachable today. Untested and not named in the ledger's deviations or
   follow-ups. Low likelihood, safe direction; worth a follow-up, not a blocker.

3. **Classifier calibration — measured, not assumed.** The auditor enumerated
   every `UnknownKeys` value across the 589 specs `Discover` returns: 34 distinct
   keys, of which the classifier fires on exactly 2 spellings (`related`,
   `superseded-by`) — both genuine. Zero false positives on real data. Residual
   risks probed synthetically: `agent:` → `parent` (edit distance 2) is the one
   plausible future false positive in an agent-oriented tool; `child_id:` →
   `child-of` and `superseded:` → `supersedes` would also fire with unhelpful
   suggestions. Slight under-fire: `related_specs` (present once in this corpus,
   unambiguous relates-to intent) is missed while `child_specs` is in the alias
   table. All minor — the warning is signal, not noise.

4. **Deviation 2 is less of a deviation than declared.** The spec's own Suggested
   Fix §3 already says "present but not `completed`/`superseded` → return false,"
   and the Risks section names the superseded escape hatch. The genuine extension
   is applying `childFinished` to the *child-count* gate as well, which is
   necessary for the escape hatch to work and is covered by
   `TestInitiativeSupersededChildCountsAsFinished` (the fixture's leaves carry
   `parent` relations, so both gates are exercised). Justified.

5. **Deviation 3 (sorting `childTableSlugs` fallback keys) has no test.** It is a
   correct determinism fix — a completion gate must not depend on map iteration
   order — but nothing asserts the ordering with two `child*` sections present.
   Cosmetic gap.

6. **Deviation 1 is justified.** Normalizing `child-of:`/`child_of:` to `parent`
   was verified against all five consumers named above; normalizing to `child` as
   Suggested Fix §2 proposed would have inverted the edge and broken parent
   discovery in `autoCompleteParentIfReady`. `TestParseChildOfIsAParentPointer`
   pins the direction.

7. **No boundary violations.** Every touched file appears in the spec's `## Changes`
   list. Nothing outside the completion predicate, the roster source, the key
   aliases, the drive unification, and the near-miss warning was modified.

## Validation re-run by the auditor

- `go build ./...` — clean (exit 0).
- `go test -count=1 ./internal/spec/... ./internal/drive/... ./internal/reconcile/...` — all pass, uncached.
- `go test ./internal/cli/...` — pass (49s).
- `gofmt -l` — none of the touched files are listed (the 27 files it does list are pre-existing and untouched by this change).
- `go build -o /tmp/hero-audit ./cmd/hero` + live `hero check`, `hero goal --check`, `hero check --reconcile` — see the falsification table above.
