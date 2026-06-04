---
title: "hero-next merge driver isn't portable — fresh clones get raw conflict markers in projected NEXT files"
slug: next-merge-driver-not-portable
type: bug
status: completed
severity: medium
priority: medium
size: small
domain: engineering
created: 2026-06-03
origin: session
root_cause_class: design
relates-to:
  - next-as-projection
  - next-projection-gate-punts-migration-to-user
  - next-project-file-conflict-not-regenerated
  - next-team-mode-per-user-handoff-unmaintained
completed_at: 2026-06-04T04:34:24Z
---

# hero-next merge driver isn't portable — fresh clones get raw conflict markers in projected NEXT files

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | medium — undercuts a headline contract ("conflicts never cause a problem"); hits every fresh clone, CI checkout, and not-yet-installed teammate, but an agent following `next-merge-recovery` self-heals and the files are disposable projections. |
| **Ease of Fix** | easy — change one driver name in a `.gitattributes` managed block (`hero-next` → `union`); union is a git built-in needing zero per-clone registration. |
| **Caused by our codebase?** | Yes — design choice to bind a *custom* merge driver (registered only in per-clone `.git/config`) to tracked files via `.gitattributes`. |
| **Needs more research?** | No — root cause confirmed against source and against live `git config` / `.gitattributes` state. |

### Background
`next-as-projection` shipped a custom git merge driver, `hero-next`, that regenerates the projected handoff files (`.hero/NEXT.md`, `.hero/next/*.md`, `.hero/QUEUE.md`, `.hero/SNAPSHOT.md`) from the local graph whenever git would otherwise produce a merge conflict. The stated contract: "for clones that have run install, conflicts in these files never produce markers." The problem is the *binding* — `merge.hero-next.driver` — lives only in `.git/config`, which git neither commits nor clones. `.gitattributes` (which **does** travel) declares `... merge=hero-next`, but on any clone that never ran `hero install` / `hero next install-hooks`, the named driver is undefined and git silently falls back to the **default text merge**, writing raw `<<<<<<<` / `=======` / `>>>>>>>` markers into the tracked projected files.

### Analysis
A git merge driver has two halves that must both be present at merge time:
1. The **attribute** `merge=<name>` on a path — lives in `.gitattributes`, **travels with the repo**.
2. The **driver definition** `merge.<name>.driver = <command>` — lives in `.git/config`, **does not travel** (git never commits or clones `.git/config`).

Hero ships (1) as a tracked file but installs (2) only via an imperative, per-clone step (`hero next install-hooks` / `hero install` → `registerMergeDriver`). When (1) is present and (2) is absent, git treats the named driver as undefined and falls back to its built-in text merge — the exact conflict-marker behavior projection was built to eliminate. The two declared backstops are also per-clone / best-effort: the `post-merge` hook lives in `.git/hooks/` (doesn't travel) and the `next-merge-recovery` LLM skill is agent-discretion (only fires if an agent is in the loop and notices markers before hand-resolving).

### Root Cause
`internal/cli/next_hooks.go:495-509` (`registerMergeDriver`) writes the driver binding to `.git/config` via `git config`, while `internal/cli/next_hooks.go:723-737` (`updateGitAttributes`) writes `merge=hero-next` into the tracked `.gitattributes`. The portability gap is structural: a tracked file references a binding that only ever exists in untracked per-clone config. This is a **design** root cause, not a logic error — every line of code does exactly what it intends; the contract is simply unachievable for clones that skip install.

### Source
- `internal/cli/next_hooks.go:495-509` — `registerMergeDriver` (writes `merge.hero-next.driver` to `.git/config`).
- `internal/cli/next_hooks.go:723-737` — `updateGitAttributes` (writes `merge=hero-next` into tracked `.gitattributes`).
- `internal/cli/next_hooks.go:80-114` — `runNextInstallHooks` wiring (the per-clone install step the whole mechanism depends on).
- `domains/engineering/skills/next-merge-recovery/SKILL.md:28-42` — documents the gap explicitly ("clones that haven't run install … git falls back … writes standard conflict markers").

### Fix Direction
Switch the `.gitattributes` managed block to git's **built-in** `merge=union` driver, which requires no `.git/config` registration and therefore travels with the repo via `.gitattributes` alone. Every clone — installed or not — then resolves these merges marker-free; the next `hero next checkpoint` (Stop-hook or pre-commit) regenerates a clean file from the graph via an idempotent total-overwrite. The custom `hero-next` driver can be kept as an optional optimization where installed, but it cannot co-bind the same path (`.gitattributes` names one driver per pattern), so `union` is the traveling guarantee and `hero-next` is dropped from the attribute block.

---

## Problem Statement

The `next-as-projection` feature promises that projected handoff files never surface git conflict markers — the `hero-next` merge driver regenerates them from the graph on conflict. This holds **only** on clones where `hero install` / `hero next install-hooks` ran. On any other clone, a concurrent edit to a projected file produces raw conflict markers in tracked content.

**Confirmed live state in this repo:**

```
$ git config --get-regexp 'merge\.'
merge.hero-next.name Hero — regenerate projected NEXT files from graph on conflict
merge.hero-next.driver hero next merge-resolve --output %A      # only in .git/config — does NOT travel

$ git ls-files .gitattributes
.gitattributes                                                   # tracked — DOES travel

$ cat .gitattributes
# >>> hero next merge driver (managed) >>>
.hero/next/*.md merge=hero-next
.hero/NEXT.md merge=hero-next
.hero/QUEUE.md merge=hero-next
# <<< hero next merge driver (managed) <<<
```

**Who hits it:**
- **Fresh clones** — a teammate clones the repo, never runs install, merges main into a branch that touched `.hero/NEXT.md` → raw markers in the tracked file.
- **CI checkouts** — runners clone and merge without ever running Hero install; any projected-file merge leaves markers, which can fail lint/format gates or get committed by an automated merge bot.
- **New teammates** — same as fresh clones, before onboarding.

**Reproduction (conceptual):**
1. Clone the repo into a directory where `hero install` was never run (so `.git/config` has no `merge.hero-next.driver`).
2. On branch A, edit `.hero/NEXT.md`; on branch B (from a common ancestor), edit the same region of `.hero/NEXT.md`.
3. `git merge` B into A. Because `.gitattributes` names `merge=hero-next` but the driver is undefined, git falls back to the default text merge and writes `<<<<<<<` / `=======` / `>>>>>>>` markers into the tracked `.hero/NEXT.md`.

**Secondary observation (not the reported bug):** the `.gitattributes` managed block covers `.hero/next/*.md`, `.hero/NEXT.md`, and `.hero/QUEUE.md`, but the merge-resolve handler in `runNextMergeResolve` (`internal/cli/next_hooks.go:153-158`) also handles `SNAPSHOT.md`, which is **not** listed in `.gitattributes`. So `.hero/SNAPSHOT.md` gets neither the custom driver nor (post-fix) the union fallback unless added to the attribute block. Worth folding the same `merge=union` line for `SNAPSHOT.md` into the fix.

## Environment Details
- Git's merge-driver resolution: `.gitattributes` supplies the `merge=<name>` attribute (committed/cloned); the driver command must be defined in config (`.git/config`, or `~/.gitconfig`, neither of which git commits or clones).
- `union` is a built-in git merge driver — no `[merge "union"]` config stanza is required; git resolves it internally regardless of clone state.

---

## Root Cause Analysis

**Confirmed (read in this session):**

1. `internal/cli/next_hooks.go:497-508` — `registerMergeDriver` shells out to `git -C <root> config merge.hero-next.driver "hero next merge-resolve --output %A"`. This writes to `.git/config`, which is per-clone and never committed.
2. `internal/cli/next_hooks.go:725-736` — `updateGitAttributes` writes `.hero/next/*.md merge=hero-next`, `.hero/NEXT.md merge=hero-next`, `.hero/QUEUE.md merge=hero-next` into `.gitattributes`, a tracked file.
3. Live `git config --get-regexp 'merge\.'` shows the binding exists only in `.git/config`; `git ls-files` confirms `.gitattributes` is tracked. The two halves are split across a traveling file and a non-traveling file.
4. `domains/engineering/skills/next-merge-recovery/SKILL.md:36-42` states the gap verbatim: clones that haven't run install fall back to default merge and get markers; the next checkpoint or an attentive agent self-heals.
5. `internal/cli/checkpoint.go:284-312` (`writeProjectedNextMD`) and `:369-378` (`writeProjectedFileIfSemanticChanged`) — the checkpoint render reads the graph and **fully overwrites** the projected file; it never trusts the existing file body. `normalizeUpdatedFrontmatter` (`:380-400`) only swaps the `updated:` line to a placeholder for the change-detection compare — it does **not** dedupe content. A union-merged file (duplicated body, doubled `---` fences) therefore differs from the clean projection and **is** overwritten on the next checkpoint. **This confirms the union fallback is fully recoverable** — the messy union output survives at most one turn until the next Stop-hook / pre-commit checkpoint fires.

**Hypothesis (not yet runtime-verified):** `merge=union` on a file with YAML frontmatter could, on a real conflict, concatenate both sides and produce two `---`-delimited frontmatter blocks. This is transiently ugly but harmless because (a) the file is never parsed for frontmatter in that window by anything load-bearing, and (b) the next checkpoint total-overwrites it. The Test Plan below asserts this end-to-end so the hypothesis is closed before delivery.

---

## Code Flow (End to End — how a fresh clone hits raw markers)

1. `hero next migrate-to-projection` (on the original repo) flips `next.projected = true` and `hero install` runs `runNextInstallHooks` (`internal/cli/next_hooks.go:80`), which calls `registerMergeDriver` (`:102`) and `updateGitAttributes` (`:108`).
2. `registerMergeDriver` (`:497`) writes `merge.hero-next.driver` into `.git/config` — **stays on the original machine**.
3. `updateGitAttributes` (`:725`) writes `merge=hero-next` into `.gitattributes` — **committed and tracked**.
4. A teammate runs `git clone`. `.gitattributes` arrives; `.git/config` is generated fresh by clone with **no** `merge.hero-next` stanza. Install was never run here.
5. Teammate's agent (or a CI job) does work, edits `.hero/NEXT.md`, then merges `main` (which also moved `.hero/NEXT.md`).
6. Git consults `.gitattributes`, sees `.hero/NEXT.md merge=hero-next`, looks up `merge.hero-next.driver` in config, finds nothing → treats the driver as undefined and falls back to the **built-in text merge**.
7. The text merge can't auto-resolve the overlapping region → writes `<<<<<<<` / `=======` / `>>>>>>>` markers into the tracked `.hero/NEXT.md`. The contract "conflicts never cause a problem" is broken.
8. Backstops are best-effort only: the `post-merge` hook (`.git/hooks/post-merge`, also didn't travel) would have run `hero next checkpoint` to regenerate — but it isn't installed either; the `next-merge-recovery` skill only helps if an agent is in the loop and notices markers before hand-resolving.

---

## Key Files

### Merge-driver install / attribute wiring
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next_hooks.go` | 495–509 | `registerMergeDriver` — writes the non-traveling `.git/config` binding. |
| `internal/cli/next_hooks.go` | 723–737 | `updateGitAttributes` — writes the traveling `.gitattributes` block referencing `merge=hero-next`. **Primary fix site.** |
| `internal/cli/next_hooks.go` | 80–114 | `runNextInstallHooks` — the per-clone install step the whole mechanism depends on. |
| `internal/cli/next_hooks.go` | 116–179 | `runNextMergeResolve` — the custom driver body (regenerates from graph). Handles NEXT/QUEUE/SNAPSHOT/user paths. |
| `.gitattributes` | 1–4 | The tracked managed block currently naming `merge=hero-next`. |

### Recovery / regeneration path (proves the union fallback is safe)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/checkpoint.go` | 284–312 | `writeProjectedNextMD` — total-overwrite from graph; never trusts existing body. |
| `internal/cli/checkpoint.go` | 369–400 | `writeProjectedFileIfSemanticChanged` / `normalizeUpdatedFrontmatter` — change-detection compares but does not dedupe; union output ≠ clean projection ⇒ overwritten. |
| `domains/engineering/skills/next-merge-recovery/SKILL.md` | 28–42 | LLM backstop skill — documents the gap; should be updated to reflect union as the primary, no-install-required guarantee. |

---

## Secondary Defects

1. **`SNAPSHOT.md` not in `.gitattributes`.** `runNextMergeResolve` (`internal/cli/next_hooks.go:153-158`) handles `.hero/SNAPSHOT.md`, but `updateGitAttributes` (`:729-733`) does not list it. So SNAPSHOT.md merges fall through to default text merge **even on installed clones** (the custom driver is never consulted for a path that lacks the `merge=` attribute). Fold `merge=union` for `.hero/SNAPSHOT.md` into the same managed block.
2. **`next-merge-recovery` skill describes install as the cure.** Once union is the portable default, the skill's "run install to prevent markers" framing (`SKILL.md:90-95`) is stale — markers shouldn't appear at all post-fix. Update the skill so it doesn't send users to a per-clone install that's no longer the guarantee.

---

## Notes
- Option evaluation (recommended → rejected):
  - **Option 1 — `merge=union` portable fallback (RECOMMENDED).** Built-in driver, travels via `.gitattributes` alone, zero per-clone setup. Trade-off: one turn of duplicated/concatenated content until the next checkpoint regenerates — acceptable because the files are disposable graph projections and the render total-overwrites (confirmed at `checkpoint.go:284-312` / `:369-378`).
  - **Option 2 — keep custom driver, harden install + `hero check` warning.** Adds a `hero check` warning when `.gitattributes` references `hero-next` but `.git/config` lacks the binding. Useful as a diagnostic but **does not fix fresh clones or CI**, where there's no opportunity to act on the warning before the merge. Rejected as the primary fix; viable as an additive safety net.
  - **Option 3 — hybrid (union default + optional custom driver).** Because `.gitattributes` binds exactly one driver per pattern, the hybrid collapses to "ship union in `.gitattributes`; treat `hero-next` as a config-only optimization that, where registered, would need a *different* binding mechanism." Since union already produces a recoverable result and the checkpoint regenerates regardless, the marginal benefit of also keeping the custom driver is near zero and adds complexity. **Collapses to Option 1.**
- Net recommendation: **Option 1.** Union is the traveling guarantee; the custom `hero-next` driver registration in `.git/config` becomes redundant for correctness (it only saves one turn of messy content where installed) and can be dropped from the attribute binding. Keeping `registerMergeDriver` is harmless but no longer load-bearing; the spec leaves it in place to avoid scope creep and recommends pointing `.gitattributes` at `union`.

---

## Recap
The `hero-next` merge driver's correctness depends on a binding (`merge.hero-next.driver`) that lives only in per-clone `.git/config` and never travels, while the `merge=hero-next` attribute that references it ships in tracked `.gitattributes` — so any clone that skipped `hero install` (fresh clones, CI, new teammates) falls back to git's default text merge and gets raw conflict markers in projected handoff files, breaking the "conflicts never cause a problem" contract. The fix is to point `.gitattributes` at git's built-in `merge=union` driver, which needs no config registration and travels with the repo; the next checkpoint total-overwrites the transiently-messy union output from the graph. Medium severity: real and recurring on uninstalled clones, but self-healing and confined to disposable projection files.

---

## Goal
Projected handoff files never surface raw git conflict markers on merge, on **any** clone — including ones that never ran `hero install` — without requiring per-clone setup. After a merge, the files may be transiently concatenated but the next `hero next checkpoint` regenerates them cleanly from the graph.

---

## Acceptance Criteria

- WHEN a clone that never ran `hero install` merges branches that both edited `.hero/NEXT.md` THE SYSTEM SHALL NOT leave raw `<<<<<<<` / `=======` / `>>>>>>>` conflict markers in the tracked projected files.
- THE SYSTEM SHALL bind the projected handoff paths (`.hero/next/*.md`, `.hero/NEXT.md`, `.hero/QUEUE.md`, `.hero/SNAPSHOT.md`) to a git merge driver that requires no `.git/config` registration to function (i.e. a built-in such as `union`).
- WHEN `merge=union` resolves a conflicting projected file THE SYSTEM SHALL produce content that the subsequent `hero next checkpoint` fully overwrites with a clean graph projection (no residual duplication, no leftover frontmatter fences).
- WHEN `hero next checkpoint` runs after a union merge THE SYSTEM SHALL produce a file byte-identical to a fresh projection from the same graph state (frontmatter intact, single `---` block, single body).
- WHERE the custom `hero-next` driver IS registered in `.git/config` THE SYSTEM SHALL continue to function (no regression for installed clones — they may resolve via either driver, both marker-free).

---

## Suggested Fix Approach

**Recommended: Option 1 — point `.gitattributes` at the built-in `merge=union` driver.**

### Change 1 — `updateGitAttributes` writes `merge=union` (and adds SNAPSHOT.md)

`internal/cli/next_hooks.go:723-737`

**Before:**
```go
// updateGitAttributes ensures .gitattributes contains the marker-
// bounded merge directive. Idempotent.
func updateGitAttributes(projectRoot string) error {
	path := filepath.Join(projectRoot, ".gitattributes")
	existing, _ := os.ReadFile(path)

	managed := fmt.Sprintf(`%s
.hero/next/*.md merge=%s
.hero/NEXT.md merge=%s
.hero/QUEUE.md merge=%s
%s`, gaMarkerStart, mergeDriverName, mergeDriverName, mergeDriverName, gaMarkerEnd)

	body := mergeMarkerBlock(string(existing), gaMarkerStart, gaMarkerEnd, managed)
	return os.WriteFile(path, []byte(body), 0o644)
}
```

**After:**
```go
// updateGitAttributes ensures .gitattributes contains the marker-
// bounded merge directive. Idempotent.
//
// Binds the projected handoff paths to git's BUILT-IN "union" merge
// driver. Unlike the custom "hero-next" driver — whose definition lives
// only in per-clone .git/config and never travels with the repo —
// "union" needs no config registration, so fresh clones, CI checkouts,
// and not-yet-installed teammates resolve these merges marker-free.
// Union concatenates both sides; the next `hero next checkpoint` total-
// overwrites the result from the graph (see checkpoint.go:writeProjectedNextMD).
func updateGitAttributes(projectRoot string) error {
	path := filepath.Join(projectRoot, ".gitattributes")
	existing, _ := os.ReadFile(path)

	managed := fmt.Sprintf(`%s
.hero/next/*.md merge=union
.hero/NEXT.md merge=union
.hero/QUEUE.md merge=union
.hero/SNAPSHOT.md merge=union
%s`, gaMarkerStart, gaMarkerEnd)

	body := mergeMarkerBlock(string(existing), gaMarkerStart, gaMarkerEnd, managed)
	return os.WriteFile(path, []byte(body), 0o644)
}
```

**Why:** `union` is resolved by git internally with no `.git/config` stanza, so the merge behavior now travels with the tracked `.gitattributes` alone. Adding `.hero/SNAPSHOT.md` closes the secondary defect where SNAPSHOT.md had no merge attribute at all.

### Change 2 — regenerate the tracked `.gitattributes` managed block to match

`.gitattributes` (tracked file, repo root)

**Before:**
```
# >>> hero next merge driver (managed) >>>
.hero/next/*.md merge=hero-next
.hero/NEXT.md merge=hero-next
.hero/QUEUE.md merge=hero-next
# <<< hero next merge driver (managed) <<<
```

**After:**
```
# >>> hero next merge driver (managed) >>>
.hero/next/*.md merge=union
.hero/NEXT.md merge=union
.hero/QUEUE.md merge=union
.hero/SNAPSHOT.md merge=union
# <<< hero next merge driver (managed) <<<
```

**Why:** The committed file is what fresh clones receive; it must already carry `merge=union` so the guarantee holds before anyone runs install. (Running `hero next install-hooks` after Change 1 regenerates this block, but the tracked file should be updated and committed directly so existing clones benefit without re-running install.)

### Change 3 (recommended, low-risk) — drop the now-redundant custom-driver registration from the contract, or keep it as an optimization

`registerMergeDriver` (`internal/cli/next_hooks.go:495-509`) and the `hero-next` driver body (`runNextMergeResolve`) are no longer load-bearing for correctness once `union` is the binding. Two acceptable resolutions — pick during delivery:
- **Keep them** (smaller diff): leave `registerMergeDriver` and `runNextMergeResolve` in place as dead-but-harmless install steps. They never get consulted because `.gitattributes` no longer names `hero-next`. Lowest risk; mild cruft.
- **Remove them** (cleaner): delete `registerMergeDriver`, its call site at `:102` / `:386`, the `merge-resolve` subcommand, and the `next-merge-recovery` skill's install guidance, since markers no longer appear. Larger blast radius — defer unless the delivery agent also owns the skill update.

The spec recommends **keep** for the bug fix (surgical), and flags removal as a follow-up cleanup so this fix stays minimal.

### Change 4 — update the `next-merge-recovery` skill framing

`domains/engineering/skills/next-merge-recovery/SKILL.md:28-42, 90-95`

Reframe so the skill no longer presents `hero install` as the cure for markers (post-fix, markers shouldn't appear because `union` resolves marker-free on every clone). Keep the `hero next checkpoint` self-heal guidance for the transient union-concatenation window. **Why:** the skill currently documents the exact bug as expected behavior; after the fix that framing is stale and would send agents to a no-longer-necessary install step.

---

## Test Plan

### Existing test review
| File | Relevance |
|------|-----------|
| `internal/cli/next_hooks_test.go` | Covers hook install / managed-block writing. Extend here for the `.gitattributes` content assertion. |
| `internal/cli/next_compact_handoff_test.go` | Exercises checkpoint regeneration; reuse its env scaffolding for the "checkpoint cleans union output" assertion. |

### Test changes needed
1. **`.gitattributes` content test** (unit, `next_hooks_test.go`): after `updateGitAttributes`, assert the managed block contains `merge=union` for all four paths (`.hero/next/*.md`, `.hero/NEXT.md`, `.hero/QUEUE.md`, `.hero/SNAPSHOT.md`) and does **not** contain `merge=hero-next`.
2. **No-driver merge produces no markers** (integration): in a temp git repo, write the `.gitattributes` union block but **do not** register any `merge.hero-next.driver` in `.git/config`. Create a base commit with a projected `.hero/NEXT.md`; branch A and branch B each edit overlapping regions; `git merge` B into A. Assert the merged `.hero/NEXT.md` contains **no** `<<<<<<<` / `=======` / `>>>>>>>` markers (union concatenates instead of conflicting).
3. **Frontmatter survives + checkpoint cleans** (integration): from the post-merge union output (which may have duplicated `---` frontmatter and a doubled body), run `hero next checkpoint`. Assert the resulting file is byte-identical to a fresh `projection.NextMD(...)` render of the same graph — single frontmatter block, single body, `updated:` line present, no markers, no duplication.
4. **Installed-clone regression guard**: with the custom `hero-next` driver still registered in `.git/config`, repeat test 2 and assert it remains marker-free (union wins via `.gitattributes`; the registered custom driver is simply not named, no conflict).

### Regression scope
- Repos that already ran `hero install` keep working — they get marker-free union merges; the custom-driver registration becomes inert, not broken.
- `hero next install-hooks` / `hero install` must regenerate the new union block idempotently (re-run safety preserved by `mergeMarkerBlock`).
- `hero hooks uninstall` (see sibling `hero-hooks-uninstall-misses-next-block`) must still strip the managed block cleanly — verify the marker-bounded removal isn't sensitive to the driver name change (it strips by markers, not content, so it should be fine, but assert it).
- The `nextMergeDriverRegistered` / `hero hooks status` reporting (`next_hooks.go:713-721`) still reflects `.git/config` state; if Change 3 keeps the registration, status output is unchanged. If a `hero check` warning is later added (Option 2 as a safety net), it should warn on the **absence** of `merge=union` in `.gitattributes`, not on the absence of the custom driver.

---

## Boundaries
- Does **not** remove the custom `hero-next` driver or the `merge-resolve` subcommand (flagged as optional follow-up cleanup, Change 3).
- Does **not** add a `hero check` warning (Option 2) — noted as an additive safety net, out of scope for the core fix.
- Does **not** change the projection render logic in `checkpoint.go` — that path is already correct (total-overwrite) and is what makes the union fallback safe.
- Does **not** address the sibling bugs (`next-project-file-conflict-not-regenerated`, `next-projection-gate-punts-migration-to-user`, `next-team-mode-per-user-handoff-unmaintained`) — related but separate.

## Risks
- **Union concatenation window**: between a merge and the next checkpoint, the projected file is transiently messy (duplicated content). Harmless because nothing load-bearing parses it in that window and the checkpoint overwrites it — but worth a one-line note in the skill so agents don't panic.
- **Stale `.git/config` registration**: leaving `merge.hero-next.driver` registered (Change 3 "keep") is inert but could confuse `hero hooks status` readers into thinking the custom driver is active. Acceptable; documented.
- **Tracked-file edit must be committed**: Change 2 edits a tracked file; it must travel with the fix commit (per the project's "handoff travels with commits" rule, and here doubly so since the whole bug is about a file that must travel).

---

## Delivery Note (2026-06-03)

Delivered on branch `fix/next-team-mode-per-user-handoff`. The delivery
lead directed the **remove** variant of Change 3 (delete the custom-driver
subsystem) rather than the spec's recommended **keep** — the bug fix and a
simplification landed together. Net diff is **−279 lines** (code + deleted
skill + `.gitattributes`).

Files changed:
- `.gitattributes` — managed block flipped to `merge=union` for all four projected files (added the missing `.hero/SNAPSHOT.md` line).
- `internal/cli/next_hooks.go` — `updateGitAttributes` now emits `merge=union` for all four paths; deleted `runNextMergeResolve`, `nextMergeResolveCmd`, `nextMergeResolveOutput`, `registerMergeDriver`, `isQueueOutputPath`, `isSnapshotOutputPath`, `runSnapshotMergeResolve`, `userFromOutputPath`, `snapshotProjectArgs`/`snapshotProject`. Kept `nextMergeDriverRegistered` + uninstall's `.git/config` cleanup, now scoped to clearing a **legacy** orphaned `merge.hero-next.*` stanza. Pre-commit / post-merge hooks kept.
- `internal/cli/next.go` — dropped `nextCmd.AddCommand(nextMergeResolveCmd)`.
- `internal/cli/checkpoint.go` — removed the dead `snapshotProject` indirection wiring (only consumer was the deleted merge driver).
- `internal/cli/next_migrate.go` — `ensureNextMDMergeDirective` now delegates to `updateGitAttributes` (single source of truth for the union block); doc/output strings updated.
- `internal/cli/hooks.go` — `hooks status` line reworded to surface only a *legacy* orphaned driver when present.
- `internal/cli/queue.go` — doc string updated (QUEUE.md is now merge=union, not hero-next).
- `domains/engineering/skills/next-merge-recovery/SKILL.md` — **deleted** (markers can no longer appear with `merge=union`; the skill's detect-and-heal premise is moot). `.claude/` mirror + `.hero/version.json` checksum entry removed.
- Tests: deleted `TestUserFromOutputPath` / `TestIsQueueOutputPath` (tested deleted code); added `TestUpdateGitAttributes_BindsAllFourPathsToUnion`, `TestUpdateGitAttributes_Idempotent`, `TestInstallNextHooks_DoesNotRegisterMergeDriver`; repurposed the merge-driver uninstall tests to seed + clear a legacy stanza.

Verified: `go build ./...`, `go vet ./internal/cli/...`, `gofmt -l` (touched files) clean; `go test ./...` green; real `git merge` on a clone with only `.gitattributes` (no driver) produces exit 0 and zero conflict markers.

## Kickoff

Cold-start prompt for a fresh delivery session:

> Deliver the bug fix spec at `.hero/planning/bugs/next-merge-driver-not-portable/spec.md`. The `hero-next` git merge driver isn't portable: its binding lives only in per-clone `.git/config` (written by `registerMergeDriver` at `internal/cli/next_hooks.go:497`), while `.gitattributes` (tracked, written by `updateGitAttributes` at `:725`) names `merge=hero-next` — so fresh clones / CI / not-yet-installed teammates fall back to git's default text merge and get raw conflict markers in `.hero/NEXT.md`, `.hero/next/*.md`, `.hero/QUEUE.md`.
>
> Fix: point the `.gitattributes` managed block at git's **built-in** `merge=union` driver (needs no `.git/config` registration, travels with the repo). Edit `updateGitAttributes` (`next_hooks.go:723-737`) to emit `merge=union` for `.hero/next/*.md`, `.hero/NEXT.md`, `.hero/QUEUE.md`, **and** add `.hero/SNAPSHOT.md` (currently missing an attribute entirely — secondary defect). Also update the tracked `.gitattributes` file directly so existing clones benefit without re-running install. The union output is transiently concatenated but the next `hero next checkpoint` total-overwrites it from the graph (`checkpoint.go:284-312` / `:369-378` confirm the render never trusts existing file content) — so it's fully recoverable.
>
> Keep `registerMergeDriver` / the custom driver in place (inert but harmless — surgical fix). Update `domains/engineering/skills/next-merge-recovery/SKILL.md` so it no longer frames `hero install` as the cure for markers. Add tests per the Test Plan: assert `.gitattributes` carries `merge=union` for all four paths and not `hero-next`; assert a no-driver-registered merge of two branches editing `.hero/NEXT.md` leaves zero conflict markers; assert `hero next checkpoint` after a union merge yields a file byte-identical to a fresh `projection.NextMD` render. Run `go test ./internal/cli/...` and confirm green before completing.
