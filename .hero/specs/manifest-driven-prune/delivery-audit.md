# Delivery audit — manifest-driven-prune

**Audited:** `git diff HEAD -- internal/install/` (prune_test.go intent-to-added first)
**Verdict:** SHIP
**Surface:** noteworthy

Cold audit by a fresh auditor. Priority per the invocation: the data-loss safety
invariant — a false prune of a user's file is the worst possible outcome.

## The data-loss invariant (highest priority) — VERIFIED

`pruneStaleFiles` (`internal/install/prune.go:221-278`) can never delete a file
that isn't in the recorded prior manifest. Read line by line:

- The only iteration that can produce a deletion candidate is `for _, rel := range
  prior.Files` (`prune.go:251`). It iterates the **prior manifest**, never a
  directory. There is **no `os.ReadDir` / directory scan anywhere in
  `pruneStaleFiles`** — confirmed by reading the whole function.
- A candidate is added to `stale` only when `!currentSet[rel]` (in prior, absent
  from this run's render). So the delete set is exactly `prior ∖ current`.
- Each stale path is `os.Stat`-guarded (`prune.go:260`): already-gone → skip.
  Then `os.Remove` (single file, not `RemoveAll`) at `prune.go:269`.
- Guard cascade before any of that: nil/empty prior → return nil (`:235`); empty
  current render → return nil (`:242`). Non-project mode → return nil (`:222`).

A file not recorded in `prior.Files` is structurally invisible to the prune. The
membership check *is* the provenance proof. Invariant holds.

`TestPruneStaleFiles_NeverRemovesUserFile` (`prune_test.go:417-448`) genuinely
plants `my-custom-agent.md` in the dest dir (never in source → never in manifest),
drops `foo`, re-runs, and asserts the user file survives **byte-for-byte**
(`bytes.Equal`). Runs across the 5 flat targets (codex excluded — see below). PASS.

## The manifest-source trap (point 2) — VERIFIED + PROVEN

The manifest is sourced from `result.rendered`, **not** `result.Copied`.
`copyFileFromFS` only appends to `Copied` on an actual write and skips it on a
`!Force` byte-match no-op (`files.go:29-36`). `result.rendered` is populated
**unconditionally**, before/independent of the copy:

- `installFlat` — `content.go:53` appends `dst` before calling `copyFileFromFS`.
- `installSkillsFlat` — `content.go:195`, same.
- `renderToFile` — `render.go:81-83` appends before the `DryRun` branch, so both
  the dry-run and write paths record.

**Proven, not just reasoned.** I wrote a throwaway test (run, passed, deleted, not
committed): two consecutive installs, the second a non-force all-no-op. Observed
`result.Copied len=0` (copy layer recorded nothing) while `result.rendered len=4`
(fully populated), and the persisted `TargetState.Files` stayed **full and
identical** to the first run — nothing pruned. Had the manifest been sourced from
`Copied`, the no-op run would have recorded an empty manifest and the next run
would have pruned everything. It does not.

## No-op safety triple (point 3) — VERIFIED, non-vacuous

- **nil/empty prior** → `prune.go:235` `if !ok || len(prior.Files) == 0 { return nil }`.
  `TestPruneStaleFiles_NoPriorManifestIsNoOp` sets `Files = nil` on a real prior
  state, drops `foo`, re-runs, asserts `foo` lingers (AC-3/AC-6). PASS.
- **empty current render** → `prune.go:242` `if len(current) == 0 { return nil }`.
  `TestPruneStaleFiles_EmptyRenderIsNoOp` empties the source `agents/commands/skills`
  dirs and asserts `foo` and `engineer` both survive (AC-12). PASS.
- **missing install-state.json** → `ReadInstallState` returns a zero-state with
  empty `Targets`, then `st.Targets[target]` not-ok → return nil.
  `TestPruneStaleFiles_FreshCloneNoState` removes the file, drops `foo`, asserts
  `foo` survives + fresh state re-recorded (AC-13). PASS.

## First-run migration AC-6 (point 4) — VERIFIED

`TestPruneStaleFiles_NoPriorManifestIsNoOp` (`prune_test.go:452-484`) installs
once (recording SkillDirs), then reads state, sets `ts.Files = nil` while leaving
SkillDirs intact, and writes it back — exactly a pre-`Files` TargetState (SkillDirs
present, Files nil). Drops `foo`, re-runs. Asserts (a) `foo` is NOT pruned and
(b) a fresh non-empty `Files` manifest reflecting the foo-less render is recorded.
`PersistInferredTargets` carries `prior.Files` forward (`state.go:319`) for the
same reason it carries SkillDirs. PASS.

## Disjointness from prune.go, AC-10 (point 5) — VERIFIED

`renderToFile` records to `result.rendered` only when the rendered `destName` has
no path separator: `if !strings.ContainsAny(destName, "/\\")` (`render.go:81`). So
Codex's `command-<n>/SKILL.md` (nested, owned by `prune.go`/`SkillDirs`) is
excluded and never enters the file manifest. `TestPruneStaleFiles_LeavesNestedSkillDirs`
drops a flat agent AND a nested skill in one run, asserts each is removed by its own
mechanism, `Run` does not error, and the file manifest contains no
`/skills/.../SKILL.md` path. Double-remove is tolerated by the `os.Stat`-skip. PASS.

## Codex removeLegacyDir nuance (point 6) — VERIFIED; pre-existing data-loss hole confirmed

The engineer's claim is accurate. `runCodex` calls `removeLegacyDir(opts,
.codex/agents)` at the **top** of the target (`target_codex.go:60`), before
`renderToFile` and before `pruneStaleFiles`. `removeLegacyDir` removes "Regular
files and directories ... unconditionally" (`cleanup.go:21-26`) — it wipes the
entire `.codex/agents` dir every run and `renderToFile` repopulates it with
`.toml`. Consequences, all correct:

- A dropped Codex agent's `.toml` is gone before `pruneStaleFiles` runs; the file
  prune `os.Stat`-skips it (AC-10) and still records the `.toml` manifest (AC-5).
  The test encodes this as `pruneActor: false` for codex and asserts no prune
  report line for it — honest, not papered over.

**The pre-existing hole (flagged per the audit's data-loss priority):** a
user-authored file in `.codex/agents/` **is destroyed on every codex install** by
`removeLegacyDir` — it is removed unconditionally and not repopulated. This is
**NOT introduced by this spec** — `removeLegacyDir` is the pre-existing dead-bytes
cleanup, and `pruneStaleFiles` correctly no-ops there. This spec's manifest prune
does not touch it. But it is a genuine data-loss surface in the codex target that
predates and is out of scope for this delivery, and the "never delete a user file"
guarantee this feature advertises does **not** hold for `.codex/agents/`. Worth a
follow-up (`hero check` advisory or a scoped removeLegacyDir fix), not a HOLD on
this spec.

## Acceptance criteria

- [✓] AC-1 remove dropped manifest file + print cleanup line — `pruneStaleFiles` `prune.go:258-276`; `TestPruneStaleFiles_RemovesDroppedAgent` asserts the `removed — dropped from product` line for the 5 pruneActor targets
- [✓] AC-2 never remove a file absent from manifest — invariant proven above; `TestPruneStaleFiles_NeverRemovesUserFile` (byte-for-byte, 5 targets)
- [✓] AC-3 nil/empty prior → no-op — `prune.go:235`; `TestPruneStaleFiles_NoPriorManifestIsNoOp`
- [✓] AC-4 per-target manifest, all six — `TestPruneStaleFiles_RemovesDroppedAgent` table over all six; each isolated workspace + `st.Targets[target].Files`
- [✓] AC-5 record actual rendered dest (.toml/.prompt.md) — `render.go:74-83`; test asserts `.toml` (codex) / `.prompt.md` (copilot) manifest membership
- [✓] AC-6 pre-`Files` TargetState → no prune + fresh manifest — `prune.go:235`, `state.go:319`; `TestPruneStaleFiles_NoPriorManifestIsNoOp`
- [✓] AC-7 --dry-run reports, deletes nothing, writes no manifest — `prune.go:265`, `RecordTargetInstall` DryRun early-return `state.go:158`; `TestPruneStaleFiles_DryRunDeletesNothing`
- [✓] AC-8 record full rel set, replacing prior — `renderedFileManifest` `prune.go:189-212` + `RecordTargetInstall` `state.go:190-201`; `TestRecordTargetInstall_FilesReplaced`
- [✓] AC-9 never manifest/prune instruction files — structural (bypass the three primitives); `TestPruneStaleFiles_NeverManifestsInstructionFiles` (claude/codex/copilot)
- [✓] AC-10 leave nested skill dirs; no error on already-removed — `render.go:81` separator rule + `os.Stat`-skip `prune.go:260`; `TestPruneStaleFiles_LeavesNestedSkillDirs`
- [✓] AC-11 prune dropped Cursor flat skill — `content.go:195`; `TestPruneStaleFiles_CursorFlatSkill`
- [✓] AC-12 empty current render → no-op — `prune.go:242`; `TestPruneStaleFiles_EmptyRenderIsNoOp`
- [✓] AC-13 install-state.json absent → no-op — `ReadInstallState` zero-state + not-ok return; `TestPruneStaleFiles_FreshCloneNoState`

## Changes

- [✓] `install.go` — `Result.rendered` field (`:128-136`) + `pruneStaleFiles` hook in `Run` (`:206-210`), placed after the installer, outside `!DryRun`, inside ModeProject, **before** `RecordTargetInstall` (`:214`). Ordering is correct: prune reads prior, then record writes new.
- [✓] `content.go` — `installFlat` (`:53`) + `installSkillsFlat` (`:195`) instrumented unconditionally; `installSkillsNested` left uninstrumented.
- [✓] `render.go` — `renderToFile` records flat names only, in both DryRun and write branches (`:81-83`).
- [✓] `prune.go` — doctrine header extension (`:41-53`), `renderedFileManifest` (`:189-212`), `pruneStaleFiles` (`:221-278`).
- [✓] `state.go` — `TargetState.Files` (`:83-90`), replace-not-merge in `RecordTargetInstall` (`:190-201`), carry-forward in `PersistInferredTargets` (`:319`).
- [✓] `prune_test.go` — 10 file-prune tests, table-driven over the six targets.
- [✓] Item 7 `docs/` — SKIPPED with concrete reason: `rg -l "prune|SkillDirs|dead bytes" docs/` empty; spec item 7 explicitly instructs skip-rather-than-invent. Legitimate, spec-sanctioned skip.

## Tests

`go test -race -count=1 ./internal/install/` → `ok` (3.4s). Spot-checked
`NeverRemovesUserFile` (5 subtests), `EmptyRenderIsNoOp`, `NoPriorManifestIsNoOp`,
`LeavesNestedSkillDirs` under `-v` — all PASS and each asserts what its AC claims,
not merely that code runs. Independent throwaway test proved the Copied-vs-rendered
trap (deleted, uncommitted). Working tree contains only the six delivery files.

## Audit notes

- No performative rows. Every DONE row maps to real diff + a test that asserts the
  claimed behavior. No downgrades.
- **Pre-existing data-loss hole in the codex target** (`removeLegacyDir` wiping
  `.codex/agents/` including user files) is real, correctly out of scope, but
  worth surfacing to the user given the feature's "never delete a user file"
  framing does not extend to that directory. Recommend a follow-up.
- Surface is `noteworthy` solely because of that pre-existing codex hole — the
  delivery itself is clean.
