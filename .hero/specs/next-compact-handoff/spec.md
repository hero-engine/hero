---
title: "Compact Handoff — Session-Scoped Resume Context at Compaction Time"
slug: next-compact-handoff
type: feature
status: completed
priority: medium
horizon: next
tags: [next, harness-integration, compaction, hooks]
relations:
  - target: session-compaction-context
    kind: builds-on
  - target: agent-cold-start
    kind: builds-on
  - target: next-md
    kind: related
  - target: next-handoff-emit
    kind: related
  - target: compact-handoff-summarizer
    kind: followed-by
---

# Compact Handoff — Session-Scoped Resume Context at Compaction Time

## Problem

Hero's projected NEXT files (`.hero/next/<user>.md`) work well for single-session flows: a user works on one thing, the projection accumulates UserAsks, NextSuggestions, and SessionReflections, and `hero next checkpoint` keeps the file fresh on every Stop and PreCompact event.

The model breaks when multiple sessions run against the same project at the same time — common in real use: a morning Claude Code session investigating bug A, an afternoon Codex session designing feature B, a parallel agent doing peer-callouts on initiative C. All three contribute events to the same per-user NEXT file. At compaction time the harness restores from its conversation summary and never re-reads NEXT.md, so the actively-injected context that *would* help the rehydrating session never lands. Meanwhile any file Hero writes at PreCompact time is invisible to the restored context.

Three concrete failures observed:

1. **Cross-session bleed when NEXT is read.** When `/resume` or a fresh session reads the per-user file, the model sees a wall of cross-session work it doesn't recognize. The conversation summary can't reconcile against it — that work happened outside this session.
2. **NEXT files stranded in commits.** The per-user file changes every turn. Models often stage it as part of their current work, then aren't sure what to write in the commit message — or worse, leave it dirty and untracked, polluting `git status`.
3. **PreCompact hook produces no model-visible output.** Today's `.claude/settings.json` wires `hero next checkpoint --quiet` to `PreCompact`. The hook runs, the file refreshes — and Claude Code never injects it. PreCompact does not support `additionalContext`. The hook is doing useful side-effect work (keeping the file current) but contributes nothing to the post-compaction model context.

The root cause: NEXT is a passive projection serving two audiences with conflicting needs. **Cross-session continuity** (browsing, `/resume`, "what was I doing yesterday") needs the full picture. **Compaction restoration** (model rehydrating mid-task) needs only this session's live thread — *delivered through the harness's inline-context mechanism, not via a file the harness won't re-read.*

## Goal

Introduce a second handoff channel — **the compact handoff** — that fires *only* at compaction time, returns content tailored to the active session, and is delivered to the model via the harness's documented context-injection mechanism (`SessionStart` with `source: "compact"` + `additionalContext`).

The existing per-user NEXT.md projection stays exactly as it is. It continues to serve cross-session continuity. The compact handoff is additive — a new, ephemeral, session-scoped artifact produced on demand.

This spec is the **deterministic skeleton** MVP. An LLM-curated middle section that synthesizes the live transcript is covered in the follow-up spec [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md) and is explicitly out of scope here. The skeleton alone is a meaningful improvement over today's "nothing is injected at all."

**Concrete success criteria:**

- After auto- or manual compaction in Claude Code, the model receives a ≤1500-token deterministic handoff that names the active spec (full content), session metadata, branch/commits, original kickoff, files touched this session, and recent decisions logged to the graph.
- The handoff contains zero content from other concurrent sessions, except where another session's events are anchored to the same active spec (intentional cross-session collaboration on shared work).
- No new files are added to the working tree at compaction time. Nothing for the model to stage or commit.
- First-compaction-of-a-fresh-session degrades gracefully via branch + recent-file heuristics rather than over-claiming.
- Codex support follows the same shape; Claude Code first.

## What already exists (build on, don't duplicate)

Substantial infrastructure is already shipped. The delivery here is a thin layer on top, not a parallel system:

- **`internal/active/`** — session-id → spec mapping at `.hero/.active-sessions.json`. `Register`, `Unregister`, `Prune`, `ActiveSpecs` already implemented. `hero active` CLI and `hero_active` MCP tool already exist. The compact handoff queries this directly; no new registry.
- **`internal/cli/checkpoint.go`** — `hero next checkpoint` already projects per-user NEXT from graph events. The graph queries are reusable for the compact handoff's skeleton content.
- **`internal/projection/`** — `UserHandoffMD` and `NextMD` already query UserAsk/NextSuggestion/SessionReflection events from the graph. Same query path; just narrowed by session_id.
- **`internal/cli/next_hooks.go`** — git-hook install pattern (`hero next install-hooks`) for `pre-commit` / `post-merge` / merge driver. The **host-tool hook** install for `.claude/settings.json` is *not* covered there and is what this spec adds.
- **`.claude/settings.json`** — already wires `hero next checkpoint --quiet` to `PreCompact` and `Stop`. The compact handoff adds a `SessionStart{source: "compact"}` entry alongside; the PreCompact entry stays (its file-refresh side effect remains useful for the post-session-resumes-fresh path).
- **`session-compaction-context` (status: completed)** — `[ACTIVE SPEC]` content is already injected into `hero context` / `hero_context` output. The compact handoff reuses the same active-spec resolution.

## Design

### Mechanism — SessionStart{source: "compact"}

Both Claude Code and Codex CLI fire a `SessionStart` event after compaction completes (auto or manual), with `source: "compact"` as the matcher. Both support an `additionalContext` field in the hook's JSON response that gets injected directly into the model's restored context — no file required.

The hook entry in `.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "compact",
        "hooks": [{
          "type": "command",
          "command": "hero next compact-handoff --json"
        }]
      }
    ]
  }
}
```

Claude Code passes the session payload to the hook on stdin as JSON (`session_id`, `transcript_path`, `cwd`, `source`, `hook_event_name`). The CLI reads stdin, extracts `session_id`, and resolves everything else from local state. (We don't take `--session` as a flag — reading stdin matches how the existing checkpoint hook works and avoids a shell-quoting failure mode.)

**Codex parity (implemented).** Codex's `SessionStartSource::Compact` is shipped (`codex-rs/hooks/src/events/session_start.rs`) and the JSON shape of `<projectRoot>/.codex/hooks.json` mirrors Claude Code's hooks block plus an optional per-command `timeout` field. The Hero installer writes the same `SessionStart{matcher:"compact"}` entry with `command: "hero next compact-handoff --json"`, `timeout: 30`, and the `added_by_hero: true` marker. Two operator-side prerequisites are surfaced (we do not mutate either):

1. Codex hooks are off by default — user must add `codex_hooks = true` under `[features]` in `~/.codex/config.toml`. The installer reads (line-scan) the global config and prints a warning when the flag is absent; the install itself still succeeds.
2. Codex prompts to "trust" project-local config on first run. Installer prints an informational note.

**Other harnesses** (Cursor, Continue, Cline, Roo, Zed) have no equivalent post-compaction context-injection mechanism. They are out of scope. Hero behaves as today in those environments.

### CLI — `hero next compact-handoff`

A new subcommand under the existing `nextCmd` (added in `init()` of `internal/cli/next.go` next to `nextCheckpointCmd`).

**Invocation forms:**

```
hero next compact-handoff --json              # reads stdin, emits JSON envelope
hero next compact-handoff --session <id>      # debug: skip stdin, force session id
hero next compact-handoff                     # debug: print human-readable text to stdout
```

**Output envelope (with `--json`):**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "<assembled handoff markdown>"
  }
}
```

Exit code 0 even when no session is resolvable — return a minimal envelope rather than failing the hook. Compaction must never block waiting on Hero.

### Content shape — the "good compact middle"

The handoff is one focused markdown blob. Sections, in order:

```markdown
## Hero session handoff (post-compact)

> Resume context for **this session only**. Other concurrent sessions' work
> is intentionally excluded. The full cross-session rollup lives at
> .hero/next/<user>.md if you need it.

**Session:** <id> · started <RFC3339> · <elapsed>
**Branch:** <branch> · <clean | dirty(N files)>
**Active spec:** [<slug>](<path>) — <type>, <status>

### What you were doing
<one-paragraph summary derived from the active spec's `## Goal` section,
truncated to ~300 chars. If no active spec, falls back to the original
kickoff prompt — see below — or "Exploratory session, no active spec.">

### Active spec — full content
<inline body of the active spec's spec.md, frontmatter stripped. Capped at
~6000 chars; if the spec is longer, truncate the body and append
"… (truncated — read full at <path>)">

### Original kickoff (this session)
> <first UserAsk event for this session_id, or first user prompt from
>  transcript_path if no UserAsk recorded yet. Truncated at 400 chars.>

### Files touched this session
- <path> — <event count, e.g. "3 edits">
- ...
(Capped at 10. Sourced from graph events tagged with this session_id.)

### Recent decisions (this session)
- <Decision event content, most recent first>
(Capped at 5. From graph Decision events with this session_id.)

### Next concrete action
<most recent NextSuggestion event for this session_id, OR next
unchecked acceptance criterion on the active spec, OR
"None recorded — pick up where the conversation left off.">

### Working tree
<git status --short, capped at 15 lines>
```

**Why this content.** The conversation summary Claude Code generates at compaction already captures the prose of what was discussed. The handoff adds what the summary *can't* reliably preserve:

- **Hard pointers** — spec slug, file paths, branch — exact strings the summary may paraphrase.
- **Active spec full body** — frontier of "what we're actually building" needs to be re-grounded post-compact; spec text is the source of truth.
- **Session-scoped event facts** — files touched, decisions, next suggestion — short-term memory the conversation summary deprioritizes.
- **Original kickoff** — the framing question that started the session, which compaction summaries often blur.

### Session filtering

The graph query for "this session's events":

```
events WHERE session_id = :current
  AND timestamp >= :session_start
UNION
events WHERE spec_slug = :active_spec
  AND timestamp >= :session_start
  AND event_type IN ('Decision', 'AcceptanceCriterionFlip', 'CommitOnSpec')
```

The second clause is what lets cross-session work *on the same spec* carry forward intentionally. If two sessions are both working on `auth-refactor`, both see each other's spec-anchored Decisions and AC flips — that's correct, they're collaborating. Cross-session UserAsks / NextSuggestions / SessionReflections from elsewhere stay excluded.

**Session start time** is read from `internal/active`'s `Session.Started` field; the registry already records it.

**Active spec resolution** uses `active.ActiveSpecs(heroDir)` filtered by current session_id. If multiple specs are registered for the same session (rare; happens when the user pivots mid-session), the most-recently-registered wins. If none, `Active spec: none`.

### Hook installer — `hero hooks install --host=claude` / `--host=codex`

The existing `hero hooks install` (in `internal/cli/hooks.go`) handles **git hooks**. This spec extends it with a `--host` flag for **host-tool hooks** — wiring the SessionStart entry into `.claude/settings.json` and Codex's hooks config.

Behavior:

- `hero hooks install` (no flag) — installs git hooks as today (no change).
- `hero hooks install --host=claude` — writes the SessionStart{compact} entry into `.claude/settings.json` under an `added_by: hero` marker. Idempotent. Preserves other entries.
- `hero hooks install --host=codex` — same, into Codex's hooks config path.
- `hero hooks install --host=all` — both, plus git.
- `hero hooks uninstall --host=claude` — removes only the Hero-marked entries; leaves user-authored hooks intact.
- `hero hooks status` — reports git hook state plus host-tool hook state per harness.

The marker convention matches the existing git-hook installer's pattern (`# >>> hero ... >>>` for shell hooks; for JSON, use an `added_by_hero: true` field on each Hero-installed hook entry so JSON parsers can identify them without comment-parsing).

`hero init` runs `hero hooks install --host=all` by default unless `--no-hooks` is passed.

### Token discipline

- Hard cap: 1500 tokens for the full `additionalContext`.
- Default target: 600–900 tokens.
- Truncation order if over budget: Working tree → Files touched tail → Recent decisions tail → Active spec body (truncate further) → Original kickoff (truncate further). Always preserve: Session metadata header, Active spec slug+title, Next concrete action.

### Out of scope

- **LLM-curated middle section** (synthesizing the conversation transcript). Covered in [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md).
- **Per-session NEXT files on disk.** Explicitly rejected — see Decisions.
- **Cleaning up or restructuring the existing per-user NEXT projection.** That continues to serve cross-session continuity unchanged.
- **Harnesses without a post-compact context-injection hook** (Cursor, Continue, Cline, Roo, Zed). They keep today's behavior.
- **Cross-session conflict resolution.** Two sessions registering as active on the same spec is fine; both get a handoff. No coordination protocol.

## Decisions

**Considered: per-session NEXT files on disk** (`.hero/next/by-session/<id>.md`). Rejected because (a) sessions don't have a clean end signal in most harnesses, so cleanup is unbounded, (b) any new file in the working tree gets stranded in commits or pollutes `git status`, (c) the on-disk projection serves no purpose the inline `additionalContext` return doesn't already serve better.

**Considered: PreCompact hook for injection.** Rejected because PreCompact explicitly does not support `additionalContext` — it can only block compaction. SessionStart with `source: "compact"` is the documented path. The existing PreCompact entry stays for its file-refresh side effect (keeps the per-user NEXT projection current so `/resume` and human browsing reflect the latest state), not for context injection.

**Considered: PostCompact hook for injection.** Rejected — PostCompact is side-effects only, no `additionalContext` support.

**Considered: bundling the LLM summarization in this MVP.** Deferred. Skeleton-only ships meaningful value today (today's behavior is *no injection at all*); the LLM step adds an API-key or hero-cloud dependency that deserves its own architectural treatment. See follow-up spec.

## Risks

- **Codex `source: "compact"` is undocumented.** The matcher is in Codex's shipped source code but not on the public hooks docs page as of 2026-05. If OpenAI changes the value before documenting it, the hook silently stops matching in Codex. Mitigation (deferred — see follow-ups below): `hero check` to probe for "no SessionStart{compact} firing recorded recently in a Codex environment." Re-verify the matcher value on each Codex release until documented.
- **Codex hooks bootstrapping is fragile.** Two operator gates (global `codex_hooks = true` feature flag + per-project trust prompt) must be satisfied for the installed hook to actually fire. The installer surfaces both as warnings/notes but cannot fix them. Open Codex regressions around hooks tracked upstream as issues #17532 and #19199 — if either lands, the installed hook may produce no observable injection until the upstream regression is resolved.
- **Active-spec resolution miss.** Sessions that never called `hero active register` (most do not, today — the registration is opt-in) will land in the "no active spec" path. Mitigation: `/deliver` and `/diagnose` slash commands should call `hero active register` at start; this is a small ergonomic upgrade with an outsized payoff for the handoff quality.
- **Session-id absent from stdin.** If a harness invokes the hook without `session_id` in the JSON payload (older Claude Code versions, edge cases), the handoff falls back to "any session in the registry whose `Started` falls within the last hour, or none." Skeleton degrades gracefully.
- **Token cap truncates the active spec.** Long specs (4000+ chars) will get body-truncated. Mitigation: the spec slug + path is always preserved in the header, so the model can `Read` it explicitly when needed.
- **First-compaction edge case.** Sessions that compact within a few turns of starting will have few session-scoped events to draw on. Skeleton falls back to: original kickoff, working tree, branch — still useful, just thinner.
- **Hook output failure must not block compaction.** If the CLI crashes mid-handoff, the hook returns nothing and the harness continues without injection. Defer panics + always-exit-0 contract are mandatory.

## Acceptance criteria

- [ ] `hero next compact-handoff --json` (reading stdin) returns a valid `additionalContext` JSON envelope.
- [ ] `hero next compact-handoff --session <id>` (no stdin) returns the same envelope for debugging.
- [ ] The handoff includes: session header, active spec full body, original kickoff, files touched this session, recent decisions, next concrete action, working tree summary.
- [ ] No new files are written under `.hero/next/` or anywhere else in the working tree as a result of running the command.
- [ ] Session filtering: events with matching `session_id` are included; events from other sessions are excluded *unless* their `spec_slug` matches the active spec (and event type is Decision/ACFlip/CommitOnSpec).
- [ ] When the session has fewer than 3 graph events tagged to it, fallback path activates and uses original kickoff + working tree + branch.
- [ ] When no active spec is registered, the handoff renders `Active spec: none (exploratory session)` without inventing one.
- [ ] Token cap enforced; truncation order matches the spec when over budget. Header + Active spec slug/title + Next concrete action always preserved.
- [ ] `hero hooks install --host=claude` writes the SessionStart{compact} entry into `.claude/settings.json` with an `added_by_hero: true` marker. Idempotent. Preserves other entries.
- [x] `hero hooks install --host=codex` writes the equivalent entry into Codex's hooks config (`<projectRoot>/.codex/hooks.json`) with `added_by_hero: true` marker and `timeout: 30`. Idempotent. Surfaces warning when `codex_hooks` feature flag is not enabled and an informational note about Codex's trust prompt on first run.
- [ ] `hero hooks uninstall --host=claude` removes only Hero-marked entries; user-authored hooks intact.
- [ ] `hero hooks status` reports host-tool hook installation state per harness alongside git hook state.
- [ ] Defer-panic + always-exit-0 contract for `compact-handoff --json`. Crash on bad stdin → return minimal valid envelope.
- [ ] Integration test: simulate a Claude Code SessionStart{compact} JSON payload on stdin and assert the envelope shape.

## Changes

- `internal/cli/next_compact_handoff.go` (new) — the subcommand and content assembly with token cap + truncation order.
- `internal/cli/next.go` — register `nextCompactHandoffCmd` in `init()`.
- `internal/cli/host_hooks.go` (new) — `--host=claude|codex|all` flag plumbing for `hero hooks install/uninstall/status`.
- `internal/cli/init.go` — call `hooks.InstallClaudeCompactHandoff` by default unless `--no-hooks`.
- `internal/hooks/claude_settings.go` (new) — read/write `.claude/settings.json` with `added_by_hero` marker preservation.
- `internal/hooks/codex_settings.go` — Codex installer implementation (`<projectRoot>/.codex/hooks.json`). Identical marker / preservation contract as Claude installer; adds `timeout: 30` per command and a `CodexFeatureFlagEnabled` line-scan of `~/.codex/config.toml`.
- `internal/cli/host_hooks.go` — Codex install/uninstall/status now exercise the real installer; warning + trust-note printed on install when applicable.
- `internal/projection/compact_handoff.go` (new) — graph queries filtered to session_id + active-spec carryover, plus path-token extraction for "files touched" tally.
- Tests:
  - `internal/cli/next_compact_handoff_test.go` (new) — stdin parsing, JSON envelope shape, token cap truncation order, kickoff/next-action edge cases.
  - `internal/hooks/claude_settings_test.go` (new) — fresh install, idempotency, preservation of pre-existing PreCompact/Stop/permissions entries, uninstall surgical removal, status, post-install JSON validity.
  - `internal/hooks/codex_settings_test.go` (new) — same matrix as Claude, plus feature-flag detection (present / absent / commented / explicit false / missing config), user-entry-in-shared-matcher preservation through uninstall, and full-cleanup file removal after uninstall.
  - `internal/cli/host_hooks_test.go` — repurposed `TestHostHooksInstall_CodexPrintsUnsupportedAndExitsZero` → `TestHostHooksInstall_CodexInstallsToProjectFile`; extended `TestHostHooksInstall_AllInstallsGitAndClaude` and `TestHostHooksUninstall_AllRemovesGitAndClaude` to also assert Codex install/uninstall side-effects.
  - `internal/projection/compact_handoff_test.go` (new) — session-tagged decisions, other-session exclusion, spec-anchored carryover, files-touched tally from Attempt/Reflection bodies, empty-session safety.

## Kickoff

Build the deterministic compact-handoff MVP: a new `hero next compact-handoff --json` subcommand that reads a SessionStart{compact} hook payload from stdin, queries the existing `internal/active` registry + graph projection scoped to that session_id, and returns the `additionalContext` JSON envelope. Plus a `--host=claude|codex|all` extension to the existing `hero hooks install` that writes the SessionStart hook entry into `.claude/settings.json` (and Codex's equivalent).

Start by reading:
- This spec and [compact-handoff-summarizer/spec.md](../compact-handoff-summarizer/spec.md) for context on what's intentionally deferred.
- [internal/cli/checkpoint.go](../../../../internal/cli/checkpoint.go) — existing graph projection for per-user handoff (the queries to narrow).
- [internal/active/active.go](../../../../internal/active/active.go) — session-id registry already in place.
- [internal/cli/next_hooks.go](../../../../internal/cli/next_hooks.go) — marker-block install pattern to mirror for JSON settings files.
- [internal/cli/hooks.go](../../../../internal/cli/hooks.go) — existing `hero hooks` parent command to extend.
- [.claude/settings.json](../../../../.claude/settings.json) — current hook entries to coexist with.

Then implement in order:

1. `hero next compact-handoff --json` reading stdin, with the deterministic content assembly. Test against a hand-crafted SessionStart payload.
2. Session-filtered graph projection helper (`internal/projection/compact_handoff.go`).
3. `internal/hooks/claude_settings.go` — JSON read/write with idempotent marker entries; `--host=claude` flag wiring.
4. `internal/hooks/codex_settings.go` — same for Codex.
5. `hero hooks status` extension to report host-tool hook state.
6. `hero init` calls `hooks install --host=all` by default.

The MVP is reviewable in ~400–600 LOC and ships meaningful behavior the day it lands: today's compaction produces *no* injected handoff; after this lands, every Claude Code compaction injects a session-scoped, deterministic resume packet with the active spec, recent decisions, and next action. The LLM-curated middle section comes later via [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md).
