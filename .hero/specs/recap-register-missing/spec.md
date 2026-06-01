---
title: "`hero recap register/unregister` referenced in slash commands but never implemented"
slug: recap-register-missing
type: bug
status: completed
severity: medium
root_cause_class: design
tags: [cli, recap, slash-commands, documentation, compaction, domain-pack]
created: 2026-05-15
completed_at: 2026-05-18T19:25:38Z
---

# `hero recap register/unregister` referenced in slash commands but never implemented

## Kickoff

`/deliver` and `/diagnose` told the agent to run `hero recap register <session-id> <slug> <cmd>` to survive compaction, but `hero recap` has no subcommands — only flags. The agent failed the very first instruction.

**Status:** completed (Direction a). The broken instruction was removed from all four affected files and replaced with a pointer to `hero next ask` / the `next-handoff-emit` skill.

**Files changed:** `commands/deliver.md`, `commands/diagnose.md`, `domains/engineering/commands/deliver.md`, `domains/engineering/commands/diagnose.md`.

**Validation passed:** `grep -rn "recap register\|recap unregister" commands/ domains/` returns no matches. `go build ./...` clean. `hero drift recap-register-missing` clean.

**Follow-up worth doing:**
- Smoke test that fenced `hero ...` invocations in embedded slash command docs are valid (catches this class of doc/code drift broadly).
- Reconcile the dual-tree embed (`commands/` legacy + `domains/engineering/commands/` domain pack) — `content.go:16-26` ships both; only some files have twins. A separate spec should consolidate.

## Issue

Reporter: internal — found while a delivery agent ran `/deliver` on the hero repo itself (`/Users/bwheeler/projects/hero-engine/repository/hero`, v0.9.1).

Symptom: `hero recap register <session-id> <slug> /deliver` exits with `unknown command "register" for "hero recap"`. The `/deliver` slash command instructs the agent to run this before doing any work, so the workflow stops at step one.

`hero recap --help` shows only flags: `--cross-repo`, `--format`, `--since`, `--subproject`. No subcommands.

## Investigation

### Where the broken instructions live

Four files reference `recap register`/`recap unregister`:

| File | Lines |
|------|-------|
| `commands/deliver.md` | 8–13 |
| `commands/diagnose.md` | 10–14 |
| `domains/engineering/commands/deliver.md` | 8–13 |
| `domains/engineering/commands/diagnose.md` | 10–14 |

Both trees ship to users. `content.go:16-26` embeds `domains/engineering/{agents,commands,skills}` (the domain pack) **and** the legacy root-level `{agents,commands,skills}` (`legacyContent`). `internal/cli/install.go:178-192` picks the domain pack when `cfg.Domain` is set in `.hero/hero.json`, otherwise it falls back to `legacyContent`. So both copies are real, both reach users, both need fixing.

### Source search

```
grep -rn "recap register\|RecapRegister\|recap_register" internal/
```

returns **nothing**. There is no `Register`/`Unregister` symbol anywhere under `internal/recap/` or `internal/cli/recap.go`. The full source of `internal/cli/recap.go` defines a single `recapCmd` with `RunE: runRecap` — no `AddCommand`, no subcommand registration.

`internal/recap/recap.go` exposes `ParseSince`, `Build`, `RenderText`, `RenderJSON`. Nothing for session registration.

### Git history

```
git log --all --oneline -S "recap register"
```

returns exactly one commit: `982742d hero v0.8.0 — spec-driven AI engineering workflow` (initial public release). That commit *introduced* the `recap register` reference in the slash-command docs. There is no later commit removing a `Register` symbol from `internal/recap/`. The feature was **never coded** — the docs were written ahead of a planned implementation that did not land.

```
git log --all --oneline -S "Register" -- internal/cli/recap.go internal/recap/
```

returns nothing — confirms `Register` never existed in those files at any point in repo history.

### What the workflow actually intends

The surrounding context in `commands/deliver.md:8-13` says:

> Before starting work, register the active spec **so context survives compaction**.

The intent is **session-survival across context compaction**, not "activity digest" (the actual purpose of `hero recap`). The naming overlap is coincidental.

The mechanism that *already exists* for compaction survival is the **NEXT.md projection** subsystem:

- `hero next ask "<text>"` — records a `UserAsk` graph node (singleton per user).
- `hero next suggest "<text>"` — records a `NextSuggestion` graph node (singleton per user).
- `hero next reflection "<text>"` — records a `SessionReflection` graph node (multiple).
- The Stop hook projects all three into `.hero/next/<user>.md`, which the next session reads via `hero next`.

This is governed by the `next-handoff-emit` skill (`skills/next-handoff-emit/SKILL.md`). It is the canonical way to preserve "what I'm doing on which spec" across compaction. `recap register` would have been a parallel — and weaker — mechanism: a flat session→slug map with no graph attribution, no per-user projection, no checkpoint integration.

### Why this slipped through

The reference dates from v0.8.0 — *before* the projection model existed. `next-handoff-emit` is newer. The slash command docs were never updated when the projection model superseded the planned `recap register` design.

### Canonical-vs-duplicate question

`commands/` and `domains/engineering/commands/` are **both canonical at runtime**, not duplicates with a clear primary. `content.go` embeds both. `install.go:179-186` resolves the active source from `hero.json`'s `domain` field, falling back to `legacyContent` (root `commands/`). Until the legacy fallback is removed, both copies must be kept in sync.

### Root cause

**DESIGN** — feature spec'd in docs and never built. The intent (compaction survival) was later satisfied by a different, better mechanism (`hero next` projection), but the dangling reference was never removed.

Sub-cause: doc drift between `commands/` and `domains/engineering/commands/` due to the dual-embed setup in `content.go`.

### Severity

**Medium**. Blocks the *first* instruction of every `/deliver` and `/diagnose` invocation. Any agent that follows the slash command literally errors out at step one. Disciplined agents will read the error and continue; less-disciplined ones may halt or loop. Workaround exists (ignore the failed command) but it is a credibility hit and a real obstacle for newly onboarded agents.

Caused by our codebase: **Yes**. Internal — this is the hero repo itself.

Needs more research: **No**. Root cause and fix direction are confirmed.

## Code Flow (End to End)

1. User invokes `/deliver <spec>` in their harness (Claude Code, Cursor, etc.).
2. Harness resolves `/deliver` to either `commands/deliver.md` (legacy embed) or `domains/engineering/commands/deliver.md` (domain pack), depending on whether `.hero/hero.json` sets a domain.
3. Agent reads the file. Line 8–13 instructs:
   ```
   hero recap register <session-id> <slug> /deliver
   ```
4. Agent shells out: `hero recap register $(date +%s) my-spec /deliver`.
5. `internal/cli/recap.go:12-17` registers `recapCmd` with `RunE: runRecap` — no subcommands.
6. Cobra parses the args. `register` is not a known subcommand and not a recognized arg pattern. Cobra emits `unknown command "register" for "hero recap"` and exits with code 1.
7. Agent observes the failure. Behavior varies: some report, some retry, some abandon the workflow.

## Key Files

### Slash command docs (need editing — both trees)
| File | Lines | Relevance |
|------|-------|-----------|
| `commands/deliver.md` | 8–13 | Tells agent to run `hero recap register` before work. Legacy embed. |
| `commands/diagnose.md` | 10–14 | Same instruction for bug diagnosis. Legacy embed. |
| `domains/engineering/commands/deliver.md` | 8–13 | Engineering-domain twin of `commands/deliver.md`. Active when `cfg.Domain` is set. |
| `domains/engineering/commands/diagnose.md` | 10–14 | Engineering-domain twin of `commands/diagnose.md`. |

### Source (no change needed under recommended fix)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/recap.go` | 1–101 | The actual `hero recap` command. No subcommands. Stays as-is. |
| `internal/recap/recap.go` | — | Recap library (build, render). No register/unregister code anywhere. |

### Embedding / domain resolution
| File | Lines | Relevance |
|------|-------|-----------|
| `content.go` | 16–26 | Embeds both `domains/engineering/{...}` and root `{agents,commands,skills}` — explains why both trees ship. |
| `internal/cli/install.go` | 178–192 | Resolves which content tree to install based on `cfg.Domain`. |

### Canonical compaction-survival mechanism (the right tool to point at)
| File | Lines | Relevance |
|------|-------|-----------|
| `skills/next-handoff-emit/SKILL.md` | 1–100 | Defines `hero next ask/suggest/reflection` — the projection-graph alternative to the never-built `recap register`. |
| `internal/cli/next_project.go` | 60–63 | Session-ID handling in the projection pipeline. |

## Suggested Fix Approach

Two plausible directions. **Tradeoff summary:**

| | (a) Remove docs | (b) Implement `recap register` |
|---|---|---|
| Effort | Trivial (4 file edits) | Multi-day (new CLI subcommand, persistence model, tests) |
| Mission fit | Same end state via existing tool | Adds a second mechanism that overlaps `hero next` |
| Doc drift risk | Low — one source of truth | High — two parallel handoff systems to keep aligned |
| User-visible change | One fewer broken command | New command surface to learn |
| Maintenance | One less line per slash command | Ongoing |

**Recommendation: (a) Remove docs.** The compaction-survival intent is already served by `hero next ask/suggest/reflection`. Adding `recap register` duplicates that surface without raising the corpus floor (the mission test).

### Recommended changes (Direction a)

#### 1. `commands/deliver.md` lines 8–13 — remove the broken instruction

Before:
```markdown
Be the `feature-delivery-lead` agent. Load the `context-injection` skill before starting.

**Before starting work**, register the active spec so context survives compaction:
```
hero recap register <session-id> <slug> /deliver
```
Use any unique session identifier (timestamp, hostname, etc.). When delivery
completes, unregister with `hero recap unregister <session-id>`.

## Delivery modes
```

After:
```markdown
Be the `feature-delivery-lead` agent. Load the `context-injection` skill before starting.

**Before starting work**, emit a `hero next ask` capturing what the user
asked for. This preserves session intent across compaction — see the
`next-handoff-emit` skill for the full pattern (ask / suggest / reflection).

## Delivery modes
```

Why: removes the broken command, points the agent at the real mechanism (`hero next ask`), and references the canonical skill so the agent loads the full pattern.

#### 2. `commands/diagnose.md` lines 10–14 — remove the broken instruction

Before:
```markdown
**Before starting work**, register the active spec so context survives compaction:
```
hero recap register <session-id> <slug> /diagnose
```
When diagnosis completes, unregister with `hero recap unregister <session-id>`.
```

After:
```markdown
**Before starting work**, emit `hero next ask` to capture the bug report
the user pasted in. This preserves session intent across compaction — see
the `next-handoff-emit` skill for the full pattern.
```

Why: same as above, for the diagnose path.

#### 3. `domains/engineering/commands/deliver.md` lines 8–13 — mirror change

Apply the identical edit as change 1. Both trees ship to users; both must match.

#### 4. `domains/engineering/commands/diagnose.md` lines 10–14 — mirror change

Apply the identical edit as change 2.

#### 5. (Optional, follow-up) Add a sync check

Once the four files match, consider adding a CI/check step (or extending `hero check`) that diffs `commands/<x>.md` against `domains/engineering/commands/<x>.md` and warns when they drift. Out of scope for this fix — flag as a separate convention/follow-up.

### Alternative direction (b) — implement `recap register/unregister`

If the team decides the projection model is insufficient and a dedicated session→spec map is wanted:

1. Add `recapRegisterCmd` and `recapUnregisterCmd` in `internal/cli/recap.go`, attached to `recapCmd` via `AddCommand`.
2. Persist `{sessionID → (slug, command, startedAt)}` to `.hero/recap/sessions.json` (new file).
3. Surface registrations in `hero recap` output as an "active sessions" header.
4. Wire teardown to delete entries on `unregister` and prune entries older than N hours.
5. Tests in `internal/recap/recap_test.go`.

Not recommended — see tradeoff table above.

## Test Plan

### Existing test review
- `internal/recap/recap_test.go` — exercises `Build`, `ParseSince`, render output. No coverage for slash-command instructions (those aren't tested).
- No existing tests verify that slash-command-prescribed CLI invocations are valid commands. This is a gap; see follow-up below.

### Test changes needed (Direction a)
Direction (a) is a doc-only edit. The functional change is "the agent no longer runs a broken command." Test coverage options:

1. **Regression test (recommended).** Add a smoke test in `internal/cli/commands_doc_test.go` (new file) that:
   - Reads each `commands/*.md` and `domains/engineering/commands/*.md` file.
   - Extracts ` ```` `-fenced `hero ...` invocations.
   - For each, splits the command and runs `hero <subcommand> --help` (or a dry-run mode if available); fails if any returns "unknown command".
   - This is a much broader guardrail than just this bug — it catches *any* future doc/code drift in the same class.

2. **Targeted regression.** Just grep for `hero recap register` across the embedded content trees in a unit test; assert no matches. Cheaper, less general.

### Regression scope
Direction (a) only touches markdown files in the embedded content trees. No Go source changes. Risks:

- Agents that have memorized the old instruction may still attempt `hero recap register` until they re-read the updated file. Acceptable — the error is informative and recoverable.
- Users with pinned older `hero` binaries (≤ 0.9.1) installed from source/legacy installs still have the broken instruction. Acceptable — they'll get the fix on next `hero install` / `hero upgrade`.
- The `next-handoff-emit` skill assumes `next.projected = true`. Projects that have not run `hero next migrate-to-projection` will need either to migrate or to fall back to writing NEXT.md directly (the `next-md` skill). Add a fallback line if the audience may include unmigrated workspaces.

## Boundaries

- Does NOT add the doc-validation smoke test in change 1 — that's a follow-up.
- Does NOT consolidate `commands/` and `domains/engineering/commands/` into one tree. That's a known architectural concern but separate work (would touch `content.go` and the install resolver).
- Does NOT touch `internal/cli/recap.go` or `internal/recap/` source.
- Does NOT change the `next-handoff-emit` skill itself.

## Risks

- If the team decides direction (b) after all, the doc removal must be reverted and the implementation added. Low risk — direction (b) is documented above.
- The replacement text references `next-handoff-emit`. If that skill's name or invocation pattern changes, this text drifts. Mitigation: the smoke test in the test plan would catch it.

## Validation

After applying changes 1–4:

1. Run `grep -rn "recap register\|recap unregister" commands/ domains/` — must return nothing.
2. Run `hero install --domain engineering` into a scratch workspace; open the installed `commands/deliver.md` — confirm the new text appears, no `recap register` line.
3. Run `hero install` (no domain flag) into another scratch workspace; same check on the legacy install path.
4. Manually invoke `/deliver <some-spec>` in a fresh harness session — agent should run `hero next ask "..."` (or equivalent) and proceed without `unknown command` errors.
5. Re-run `hero recap --help` — confirm unchanged (no spurious changes leaked into source).

## Notes

- This is a meta-bug: found in the hero repo itself while running hero's own workflow. The fix re-aligns the slash command docs with the actual shipped CLI surface.
- No `tracker_id` assigned — internal find. If one is added later, post diagnosis per the standard protocol.
- Worth a follow-up: the dual-tree embed in `content.go` is fragile. Six other slash commands exist in `domains/engineering/commands/` and only some have twins in root `commands/`. A separate spec should reconcile.

## Recap

`hero recap register/unregister` was documented in `/deliver` and `/diagnose` slash commands in v0.8.0 but was never implemented in source. The compaction-survival intent is already served by `hero next ask/suggest/reflection` (the projection model in `skills/next-handoff-emit`). Recommended fix: remove the broken instruction from all four affected files (both `commands/` and `domains/engineering/commands/` because both ship to users) and point at the existing mechanism. Severity medium — blocks the first step of every delivery and diagnosis workflow but has an obvious recovery path.
