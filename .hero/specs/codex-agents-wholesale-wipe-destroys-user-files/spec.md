---
title: "Codex install wholesale-wipes .codex/agents every run — destroys user-authored agent files"
slug: codex-agents-wholesale-wipe-destroys-user-files
type: bug
status: completed
severity: high
priority: P1
domain: engineering
created: 2026-07-16
origin: session
root_cause_class: code
tags: [install, codex, data-loss, cleanup, user-content]
relations:
  - target: manifest-driven-prune
    kind: related
  - target: agents-md-erased-by-snapshot-pointer-writer
    kind: related
completed_at: 2026-07-17T00:39:00Z
---

# Codex install wholesale-wipes `.codex/agents` every run — destroys user-authored agent files

## Kickoff

Every `hero install`/`hero upgrade` on the Codex target runs `os.RemoveAll` over the **entire**
`.codex/agents/` tree before re-rendering — the live directory Codex loads `<name>.toml`
agents from. Any file a user put there (a hand-authored `.toml` agent, notes, a subdir) is
gone. The dead-bytes cleanup was only ever meant to clear pre-`.toml` legacy `.md` files;
the code does far more than the intent, and the divergence is written in plain sight — the
function's own header comment says it cleans "`.codex/agents/*.md`" while the code wipes
everything.

**Status:** diagnosed. Root cause CONFIRMED against source. Fix designed, not yet implemented.

**Pick up at:** implement the recommended fix (option **c** — a `*.md`-only predicate on the
`.codex/agents` cleanup call), add the survival test, run the suite. This is now a *clean*
fix: `manifest-driven-prune` (shipped today) already owns the `.toml` removal lifecycle
provenance-safely, so the `.codex/agents` cleanup no longer needs to touch `.toml` at all —
it only needs to clear the legacy `.md` dead-bytes.

→ `.hero/planning/bugs/codex-agents-wholesale-wipe-destroys-user-files/spec.md`

**Files:** `internal/install/target_codex.go:60`, `internal/install/cleanup.go:29-103`,
`internal/install/prune_test.go:265-285` (the `fileTgts` table that documents the hole),
`internal/install/harness_smoke_test.go:102-104` (dead-`.md` assertions the fix must keep green).

**Skip:** do NOT change the `.codex/commands` cleanup (target_codex.go:63) — that dir has no
Codex loader and wholesale removal is correct. Do NOT change copilot's cleanup
(target_copilot.go:62) — it targets the disjoint legacy `.github/copilot/` tree, never the
live `.github/prompts/` render dest. Do NOT try to make the `.toml` lifecycle part of this
cleanup — render + manifest-prune already own it.

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — silent, unrecoverable data loss of user content on a routine, idempotent-looking command (`hero install`/`hero upgrade`), every run. |
| **Ease of Fix** | easy — scope one cleanup call to `*.md`-only; the `.toml` and user-file safety is already handled by the just-shipped `manifest-driven-prune`. |
| **Caused by our codebase?** | Yes — `runCodex` calls `removeLegacyDir` (unconditional `os.RemoveAll`) against a live loader directory. |
| **Root cause class** | code |
| **Needs more research?** | No — root cause confirmed against source; the cleanup, render, and prune paths are all read and understood, and the existing test suite already documents the wholesale-wipe as a load-bearing assumption. |

### Background
Codex loads project agents from `.codex/agents/<name>.toml` (source-verified against
openai/codex, cited in `target_codex.go:9-14`). Hero renders each canonical agent into that
directory as TOML via `renderCodexAgentToml`. Before Codex switched to `.toml`, Hero rendered
agents there as markdown (`.codex/agents/*.md`); those pre-`.toml` files are genuinely dead —
Codex's loader ignores `.md` in that directory. A cleanup was added to clear that dead-byte
legacy.

## Problem

On **every** Codex install and upgrade, `runCodex` deletes the entire contents of
`.codex/agents/` — every file and subdirectory, regardless of who wrote it — and only then
re-renders Hero's current agents as `.toml`. A user who authored their own Codex agent at,
say, `.codex/agents/my-reviewer.toml` loses it the next time any `hero install` or
`hero upgrade` touches the Codex target (including the auto-sync that fires when a sibling
harness is installed). There is no warning, no backup, no recovery — the bytes are gone.

## Root Cause

**CONFIRMED. Code defect.**

`runCodex` (`internal/install/target_codex.go:60`) calls:

```go
if err := removeLegacyDir(opts, filepath.Join(destBase, "agents")); err != nil {
	return nil, fmt.Errorf("cleanup .codex/agents: %w", err)
}
```

`removeLegacyDir` → `removeLegacyEntry` (`internal/install/cleanup.go`) iterates every entry
in the directory and, for anything that is not a foreign symlink, does:

```go
// Directory or regular file → remove unconditionally.
...
if err := os.RemoveAll(full); err != nil {
	return err
}
```

The only entries spared are symlinks pointing *outside* a Hero-managed tree. Every regular
file and every subdirectory — `engineer.toml`, `my-reviewer.toml`, a user's `notes.md`, a
nested folder — is removed unconditionally. `.codex/agents` is the **live directory Codex
reads agents from**, so this is wholesale destruction of a directory that legitimately holds
user content.

The cleanup's documented purpose is narrower than what the code does. `cleanup.go`'s header
frames it as removing "dead bytes from prior install layouts" where "every known legacy
location has no current consumer." That premise is **false for `.codex/agents`**: it is not
a dead legacy location, it is the current consumer. The premise holds for the genuinely-dead
`.codex/commands` (no loader) and for copilot's `.github/copilot/` tree — but not here.

## Evidence

**The function's own contract already says `*.md`, not everything.** `target_codex.go:42-43`
documents the cleanup as:

> Cleans up dead bytes from prior installs at `.codex/agents/*.md` and `.codex/commands/*`.

The intent recorded in the comment is `.codex/agents/`**`*.md`** — markdown only. The code
(`removeLegacyDir` → unconditional `os.RemoveAll`) removes **everything**. Comment and code
diverge; the comment is the correct spec of intent, the code overreaches. This is the
strongest single piece of evidence that the wholesale behavior is a defect, not a decision.

**The just-shipped `manifest-driven-prune` test suite documents the hole as a known
mechanism.** `internal/install/prune_test.go:265-285` defines the per-target `fileTgts`
table with a `pruneActor` flag, and codex is the one target set to `false`:

```go
// pruneActor is false for codex, whose .codex/agents dir is wiped
// wholesale every run by removeLegacyDir (dead-bytes cleanup) — a dropped
// codex agent is removed by that pre-existing mechanism, so pruneStaleFiles
// correctly no-ops on it (AC-10) and prints nothing.
```

And `TestPruneStaleFiles_NeverRemovesUserFile` (`prune_test.go:417-421`) **explicitly excludes
codex** from the "a user file survives a prune" guarantee:

```go
func TestPruneStaleFiles_NeverRemovesUserFile(t *testing.T) {
	for _, tc := range fileTgts {
		if !tc.pruneActor {   // skips codex
			continue
		}
```

with the comment *"its .codex/agents dir is owned wholesale by removeLegacyDir (a separate
dead-bytes mechanism), not by this prune."* The provenance-safe prune was built to protect
user files on five targets, and codex was carved out precisely because the wholesale wipe
would delete the planted user file. The carve-out **is** the bug, made visible.

**Ordering confirms the wipe wins.** The target runner (`runCodex`, which calls
`removeLegacyDir`) executes inside `Run` before `pruneStaleFiles` (`install.go:206-210`). So
`.codex/agents` is emptied before the provenance-safe prune ever inspects it — which is why
`pruneStaleFiles` finds a dropped codex agent already gone and no-ops (`prune.go:260-263`,
AC-10). The safe mechanism never gets a chance to be the one in charge for codex.

**Render records `.toml` in the manifest.** `renderCodexAgentToml` returns `"<name>.toml"`
(no path separator), so `renderToFile` records it in `result.rendered` →
`TargetState.Files` (`render.go:75-83`). `pruneStaleFiles` therefore *already* covers the
`.toml` removal lifecycle for codex; the only reason it does not act is that `removeLegacyDir`
beats it to the directory. Remove the `.toml` from the cleanup's blast radius and the
manifest prune seamlessly takes over — no `.toml` orphans, no user-file loss.

## All-Targets Audit (harness-changes-cover-all-targets)

`removeLegacyDir` has exactly three call sites in target code, plus one in the legacy
canonical-mirror cleanup:

| Call site | Target dir | Live loader dir? | Verdict |
|---|---|---|---|
| `target_codex.go:60` | `.codex/agents` | **YES** — Codex reads `<name>.toml` here | **THE BUG.** Wholesale wipe = user data loss. |
| `target_codex.go:63` | `.codex/commands` | No — Codex has no command loader (commands render as skills under `.agents/skills/command-*`) | Correct. Wholesale removal is right. Leave unchanged. |
| `target_copilot.go:62` | `.github/copilot/{agents,commands,skills}` | No — verified: Copilot Chat never ingests `.github/copilot/` (`target_copilot.go:28-30`); the live render dests are `.github/prompts/{agents,commands}` and `.github/skills`, which this call does NOT touch | Correct. Disjoint legacy tree. Leave unchanged. |
| `cleanup.go:179` | `.hero/{agents,commands,skills}` | No — legacy P2-era canonical mirror dirs, no consumer under render-direct | Correct. Not user content. Leave unchanged. |

The other four render targets — claude, opencode, cursor, generic — do **not** call
`removeLegacyDir` on their live agent dests at all. They copy `.md` directly and rely on the
provenance-safe `pruneStaleFiles` for removal, which is exactly why their user files are
protected (and tested). **Codex is the only target that points the unconditional cleanup at a
live loader directory.** No other target has the same hole.

## Acceptance Criteria

- **AC-1:** WHEN `hero install`/`hero upgrade` runs the Codex target AND `.codex/agents/`
  contains a user-authored `.toml` agent NOT written by Hero (absent from the install
  manifest), THE SYSTEM SHALL leave that file present and byte-identical after the run.
  *(Load-bearing.)*
- **AC-2:** WHEN `.codex/agents/` contains legacy Hero-rendered `.md` dead-bytes (e.g.
  `engineer.md` from the pre-`.toml` era), THE SYSTEM SHALL remove them during install/upgrade
  — the smoke-test invariant `mustNotExist(".codex/agents/engineer.md")` SHALL keep passing.
- **AC-3:** WHEN a Hero-rendered `.codex/agents/<name>.toml` agent was recorded in the prior
  install manifest but is no longer in the product's canonical set, THE SYSTEM SHALL remove
  it via `pruneStaleFiles` (manifest provenance), NOT via the dead-bytes cleanup.
- **AC-4:** THE SYSTEM SHALL continue rendering the current canonical agents to
  `.codex/agents/<name>.toml` on every run — `mustBeRegularFile(".codex/agents/engineer.toml")`
  SHALL keep passing (`TestHarness_SmokeCodex`, `TestRunCodexProject`).
- **AC-5:** THE `.codex/commands` cleanup (`target_codex.go:63`) SHALL retain wholesale
  removal behavior byte-for-byte unchanged (no loader exists there; nothing repopulates it).
- **AC-6:** THE copilot cleanup of `.github/copilot/{agents,commands,skills}`
  (`target_copilot.go:62`) SHALL remain unchanged, AND no install target SHALL run an
  unconditional `os.RemoveAll` over a live loader directory — verified by the all-targets
  audit above.
- **AC-7:** WHERE `.codex/agents/` does not exist (fresh install), THE SYSTEM SHALL no-op the
  cleanup and exit without error.
- **AC-8:** WHEN the scoped cleanup leaves non-`.md` entries behind (user `.toml`, subdirs),
  THE SYSTEM SHALL NOT remove the `.codex/agents/` directory itself (it is non-empty and live).

## Suggested Fix Approach

Three directions were evaluated (all satisfy the invariants; they differ in surface area and
clarity):

- **(a)** Scope the `.codex/agents` cleanup to remove only `*.md`, inline in `runCodex`.
- **(b)** Drop the `removeLegacyDir(.codex/agents)` call entirely and add a bespoke
  legacy-`.md`-only helper. Rejected: reimplements the walk/empty-dir/symlink logic
  `removeLegacyDir` already has, for no benefit.
- **(c) — RECOMMENDED.** Give `removeLegacyDir` an optional match predicate. The
  `.codex/agents` call passes a `*.md`-only predicate; `.codex/commands`, copilot, and the
  `.hero/` mirror calls pass `nil` (allow-all) and keep byte-identical wholesale behavior.

**Why (c):** smallest behavioral delta, keeps the proven walk/empty-dir/symlink handling in
one place, makes "wholesale" vs. "scoped" an explicit, readable property at each call site,
and leaves every other call site's semantics provably unchanged (a `nil` predicate is the
current code path). It also directly satisfies the `harness-changes-cover-all-targets`
tripwire by making the per-target scope legible at the call.

### `internal/install/cleanup.go` — add an optional predicate

**Before** (`removeLegacyDir`, cleanup.go:29-49):
```go
func removeLegacyDir(opts Options, legacyDir string) error {
	info, err := os.Lstat(legacyDir)
	...
	for _, e := range entries {
		full := filepath.Join(legacyDir, e.Name())
		if err := removeLegacyEntry(opts, full, e); err != nil {
			return err
		}
	}
	...
}
```

**After** — keep the existing exported name as a wholesale wrapper, add a matching variant:
```go
// removeLegacyDir removes everything inside legacyDir (wholesale). Kept for
// dead legacy locations with no current consumer (.codex/commands,
// .github/copilot/*, .hero/{agents,commands,skills}).
func removeLegacyDir(opts Options, legacyDir string) error {
	return removeLegacyDirMatching(opts, legacyDir, nil)
}

// removeLegacyDirMatching removes entries in legacyDir for which shouldRemove
// returns true; a nil predicate matches every entry (wholesale). Use a
// predicate when legacyDir is ALSO a live loader directory that legitimately
// holds user content — e.g. .codex/agents, where only pre-.toml *.md
// dead-bytes are Hero's to remove and any other file (a user .toml agent) must
// survive. The empty-dir cleanup only fires when nothing is left, so a live
// dir with user files or current .toml is preserved.
func removeLegacyDirMatching(opts Options, legacyDir string, shouldRemove func(name string) bool) error {
	info, err := os.Lstat(legacyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if shouldRemove != nil && !shouldRemove(e.Name()) {
			continue // not ours to remove — leave user files / current .toml in place
		}
		full := filepath.Join(legacyDir, e.Name())
		if err := removeLegacyEntry(opts, full, e); err != nil {
			return err
		}
	}
	// (unchanged) remove the dir only if it is now empty
	...
}
```

### `internal/install/target_codex.go` — scope the agents cleanup, leave commands wholesale

**Before** (target_codex.go:56-65):
```go
// .codex/agents/*.md (Codex requires .toml — markdown is dead;
// the dir is repopulated by renderToFile below with .toml).
// .codex/commands/* (no loader at any scope; nothing repopulates).
if err := removeLegacyDir(opts, filepath.Join(destBase, "agents")); err != nil {
	return nil, fmt.Errorf("cleanup .codex/agents: %w", err)
}
if err := removeLegacyDir(opts, filepath.Join(destBase, "commands")); err != nil {
	return nil, fmt.Errorf("cleanup .codex/commands: %w", err)
}
```

**After**:
```go
// .codex/agents is the LIVE dir Codex loads <name>.toml agents from, so it
// legitimately holds user files. Remove ONLY pre-.toml *.md dead-bytes here;
// dropped Hero .toml agents are pruned provenance-safely by pruneStaleFiles
// (see prune.go / manifest-driven-prune), and user files are left untouched.
if err := removeLegacyDirMatching(opts, filepath.Join(destBase, "agents"), isLegacyCodexAgentMarkdown); err != nil {
	return nil, fmt.Errorf("cleanup .codex/agents: %w", err)
}
// .codex/commands has no loader at any scope; nothing repopulates it, so
// wholesale removal is correct.
if err := removeLegacyDir(opts, filepath.Join(destBase, "commands")); err != nil {
	return nil, fmt.Errorf("cleanup .codex/commands: %w", err)
}
```

Add the predicate (next to the loader-path doc block in `target_codex.go`):
```go
// isLegacyCodexAgentMarkdown reports whether name is a pre-.toml Hero agent
// dead-byte in .codex/agents. Codex's loader ignores .md there, so any .md is
// non-functional legacy; .toml (current render + manifest-pruned) and every
// other user file are preserved.
func isLegacyCodexAgentMarkdown(name string) bool {
	return strings.HasSuffix(name, ".md")
}
```
(Also update the `target_codex.go:42-43` header comment to state the cleanup is `.md`-only by
construction now, closing the comment/code divergence.)

### Residual (state explicitly, do not silently expand scope)
A user file that is itself a `.md` inside `.codex/agents` is still removed. This is
acceptable and intentional: Codex's loader does not read `.md` from that directory, so such a
file is non-functional there, and it is indistinguishable by name from the pre-`.toml`
dead-bytes the cleanup exists to remove. The load-bearing data-loss vector — a user's live
`.toml` agent, or any subdir/other file — is fully protected. A stricter "only remove `.md`
whose stem matches a current canonical agent" variant was considered and rejected as
over-engineering for an inert-file edge case; it would couple the cleanup to the canonical
set for no functional gain.

## Test Plan

### Existing test review
- `internal/install/harness_smoke_test.go:102-104` — `TestHarness_SmokeCodex` asserts
  `engineer.toml` / `reviewer.toml` present and `engineer.md` absent. Must stay green (AC-2,
  AC-4). The `.md`-scoped cleanup still removes `engineer.md`; render still writes `.toml`.
- `internal/install/install_test.go:1434` — `TestRunCodexProject` asserts `.toml` render.
  Must stay green (AC-4).
- `internal/install/prune_test.go:265-285` — the `fileTgts` table and its `pruneActor:false`
  carve-out for codex are the *documentation of this bug*. After the fix, codex user files
  survive, so this table's rationale changes (see below).

### Test changes needed
1. **NEW — user-file survival (the load-bearing guard).** Add
   `TestRunCodex_PreservesUserAuthoredTomlAgent` (or a codex case in a shared cleanup test):
   - install codex once,
   - plant `.codex/agents/my-reviewer.toml` with distinctive bytes (a file Hero never wrote),
   - re-run install (and/or upgrade),
   - assert `my-reviewer.toml` still exists and is byte-identical (AC-1),
   - assert a planted legacy `.codex/agents/engineer.md` dead-byte is removed (AC-2),
   - assert `.codex/agents/engineer.toml` (current render) is present (AC-4).
   Verify this test goes **RED against the current wholesale wipe** and green after the fix —
   the regression must not be able to land silently again.
2. **Extend `TestPruneStaleFiles_NeverRemovesUserFile`** to include codex by planting a
   `.toml` (not `.md`) user file, OR keep the `.md`-based case excluded and rely on the new
   dedicated test above for codex. Update the `pruneActor:false` comment (prune_test.go:266-270)
   to reflect that codex user files now survive because the cleanup is `.md`-scoped, not
   because "the dir is wiped wholesale."
3. **Dropped-`.toml` via manifest (AC-3).** Add/confirm a codex case: install with agent
   `foo` (renders `foo.toml`, recorded in manifest), drop `foo` from source, re-install,
   assert `foo.toml` removed — by `pruneStaleFiles`, now that the cleanup no longer wipes it.
   (`TestPruneStaleFiles_RemovesDroppedAgent` already covers codex removal; with the fix the
   remover becomes `pruneStaleFiles` rather than `removeLegacyDir` — update its
   `pruneActor:false` expectation for codex so it asserts the prune report fires.)
4. **`.codex/commands` unchanged (AC-5).** Add/confirm a test that a file planted under
   `.codex/commands` is still wholesale-removed on install.

### Regression scope
- `runCodex` end-to-end (install + upgrade + auto-sync), the two Codex cleanup call sites,
  and the shared `removeLegacyDir` used by copilot and the `.hero/` mirror cleanup — the `nil`
  predicate keeps all of those byte-identical.
- The manifest-prune path for codex `.toml` agents now becomes *active* (previously masked by
  the wipe); confirm no double-remove error (prune tolerates already-gone files, prune.go:260).
- Empty-dir cleanup: confirm `.codex/agents` is NOT removed when user `.toml`/files remain
  (AC-8), and IS removed only if truly empty after `.md` removal.

**Needs more research?** → No. Root cause confirmed against source; fix designed and bounded;
all-targets audit complete (codex is the sole target with the hole; copilot verified safe).

## Changes (as delivered)

- `internal/install/cleanup.go` — split `removeLegacyDir` into a thin nil-predicate
  wrapper over a new `removeLegacyDirMatching(opts, dir, shouldRemove func(name string) bool)`.
  Per-entry removal is gated on the predicate (nil = allow-all = byte-identical wholesale
  behavior); the empty-dir cleanup only fires when nothing is left after the scoped removal.
- `internal/install/target_codex.go` — the `.codex/agents` cleanup now calls
  `removeLegacyDirMatching(..., isLegacyCodexAgentMarkdown)` (new predicate:
  `strings.HasSuffix(name, ".md")`); `.codex/commands` stays wholesale (`removeLegacyDir`).
  Added `strings` import, the predicate func, and updated the loader-path header comment to
  state the cleanup is `.md`-scoped by construction.
- `internal/install/prune_test.go` — flipped codex `pruneActor` to `true`, added a per-target
  `userFile` field (codex plants a `.toml`, others `.md`), and updated the carve-out comments
  so they describe the `.md`-scoped cleanup instead of a wipe that no longer happens.
- `internal/install/cleanup_test.go` — added the load-bearing survival test
  (`TestRunCodex_PreservesUserAuthoredTomlAgent`), `.codex/commands` wholesale test (AC-5), and
  four focused `removeLegacyDirMatching` unit tests (AC-2/5/6/7/8).

## Completion Ledger

### Understanding & validation

Fix implemented per the spec's option (c): `removeLegacyDir` gains an optional match predicate;
codex's `.codex/agents` call scopes to `*.md` dead-bytes only, every other call site passes the
nil-predicate wrapper (byte-identical wholesale). Stack: Go; loaded stack-detection, go-stack,
implementation-principles, testing-and-validation, completion-ledger. Validation: `go build ./...`
clean; `go test -race -count=1 ./internal/install/` green; `go test -count=1 ./...` green with zero
failures (the noted opsrunner subprocess-timeout flakes did not fire). The load-bearing survival
test was verified RED against a temporarily-restored wholesale wipe, then GREEN after the fix.
Manual repro run end-to-end with the built binary (transcript below).

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | User-authored `.toml` agent survives byte-identical after run *(load-bearing)* | DONE | `internal/install/target_codex.go:67` + `cleanup.go:37`; `TestRunCodex_PreservesUserAuthoredTomlAgent`, `TestPruneStaleFiles_NeverRemovesUserFile[codex]`; verified RED-then-GREEN; manual repro sha-match |
| 2 | Legacy `.md` dead-bytes removed; `mustNotExist(.codex/agents/engineer.md)` stays green | DONE | `isLegacyCodexAgentMarkdown` (`target_codex.go:138`); `TestHarness_SmokeCodex`, `TestRunCodexProject`, survival test, `TestRemoveLegacyDirMatching_MdScopedPreservesOthers` all green |
| 3 | Dropped Hero `.toml` agent removed via `pruneStaleFiles`, not dead-bytes cleanup | DONE | `TestPruneStaleFiles_RemovesDroppedAgent[codex]` now asserts the prune report fires (`pruneActor:true`) |
| 4 | Current canonical agents still render to `.codex/agents/<name>.toml` | DONE | `TestHarness_SmokeCodex`, `TestRunCodexProject`, survival test assert `engineer.toml`/`reviewer.toml` present; manual repro |
| 5 | `.codex/commands` cleanup retains wholesale removal byte-for-byte | DONE | `target_codex.go:72` unchanged (`removeLegacyDir`); `TestRunCodex_CommandsDirWholesaleRemoved` (planted `.toml` also removed), `TestRemoveLegacyDirMatching_NilPredicateWholesale` |
| 6 | Copilot cleanup unchanged; no target runs `os.RemoveAll` over a live loader dir | DONE | `target_copilot.go:62` untouched (nil-predicate wrapper); nil path proven byte-identical by `TestRemoveLegacyDirMatching_NilPredicateWholesale`; all-targets audit in spec |
| 7 | Fresh install (`.codex/agents` absent) no-ops the cleanup without error | DONE | `removeLegacyDirMatching` returns nil on `os.IsNotExist` (`cleanup.go`); `TestRemoveLegacyDirMatching_MissingDirNoOp` |
| 8 | Scoped cleanup leaving non-`.md` entries does NOT remove `.codex/agents` itself | DONE | Empty-dir cleanup gated on post-scan emptiness; survival test asserts `mustBeDirectory(.codex/agents)`; `TestRemoveLegacyDirMatching_MdScopedPreservesOthers` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `cleanup.go` — add `removeLegacyDirMatching` predicate variant; `removeLegacyDir` becomes nil-predicate wrapper | DONE | Walk/Lstat/ReadDir/symlink/empty-dir logic intact; per-entry removal gated on predicate |
| 2 | `target_codex.go` — scope `.codex/agents` to `*.md` via `isLegacyCodexAgentMarkdown`; keep `.codex/commands` wholesale; add `strings` import; fix header comment | DONE | Predicate `strings.HasSuffix(name, ".md")`; commands call unchanged |
| 3 | `prune_test.go` — flip codex `pruneActor` carve-out; per-target `userFile`; refresh stale comments | DONE | Codex now exercises the manifest prune for `.toml`; comments describe `.md`-scoped cleanup |
| 4 | Load-bearing survival test + AC coverage tests | DONE | `cleanup_test.go`: survival, commands-wholesale, 4× `removeLegacyDirMatching` unit tests |

### Exercise-the-feature check

- [x] User-visible behavior exercised end-to-end with the built `hero` binary (`go build -o hero ./cmd/hero`).
  In a scratch dir: `hero init` + `hero install project . --target codex`; planted a user
  `.codex/agents/my-custom.toml` (sha `b169a0a2…`) and a legacy `.codex/agents/engineer.md`; re-ran
  `hero install project . --target codex --force`. Observed: install output printed only
  `cleanup .codex/agents/engineer.md (removed dead bytes)` — not a dir wipe; `my-custom.toml`
  SURVIVES byte-identical (sha unchanged `b169a0a2…`); `engineer.md` REMOVED; `engineer.toml` PRESENT.

### Excellence Bar self-check

Yes — a senior engineer who cares would ship this. The behavioral delta is minimal and legible
(scope vs. wholesale is explicit at each call site), every other call site is provably unchanged
(nil predicate is the pre-existing code path), the stale `pruneActor:false` carve-out was flipped
to actively exercise the now-correct manifest prune rather than left describing a wipe that no
longer happens, all 8 ACs have targeted tests, and the load-bearing guard was verified to fail
against the old behavior so the regression cannot silently reland.
