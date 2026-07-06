---
title: "Harness Agnosticism Sweep — De-Claude and De-Dogfood the Shipped Content"
slug: harness-agnosticism-sweep
type: enhancement
status: planning
priority: P1
size: medium
domain: engineering
created: 2026-07-06
tags: [content, harness-agnosticism, install-targets, frontmatter, dogfood-leakage, agents-md]
relations:
  - target: content-remediation
    kind: parent
  - target: hero-content-audit
    kind: related
  - target: content-dedup-resync
    kind: builds-on
  - target: agent-safety-conventions
    kind: related
---

# Harness Agnosticism Sweep — De-Claude and De-Dogfood the Shipped Content

## Context

Hero installs its content packs to six targets — `opencode|cursor|claude|copilot|codex|generic` (`internal/cli/install.go:86`) — under the `harness-changes-cover-all-targets` tripwire (`.hero/specs/hero-content-audit/spec.md:109-111`): no Claude-only assumptions, and any harness-specific text must be explicitly scoped. The content audit (`.hero/specs/hero-content-audit/findings-{routing,commands,skills,agents}.md`) found systematic violations in three families:

1. **Claude-Code machinery presented as the mechanism.** The engineering AGENTS.md "Internal Lookups — Tool Routing" section (`domains/engineering/AGENTS.md:110-127`) teaches `mcp__hero__*` double-underscore naming, the `Explore` agent, and `ToolSearch` deferred-tool loading — all Claude-Code-only, unscoped, shipped identically to all six targets (routing S2). Three engineering commands use Claude tool vocabulary — diagnose.md "Task agents", deliver.md "general-purpose subagent", mock.md "the Agent tool" (F23). The core `next-md`/`next-handoff-emit` skills present the Stop-hook checkpoint as universal, but hooks install only for claude and codex (`internal/install/claude_hooks.go`, `codex_hooks.go`; opencode explicitly deferred at `internal/cli/install.go:365`) — on cursor/copilot/generic the "machine half" never refreshes and auto-emit never fires (skills S2). `core/agents/project-context-builder.md` (core-only since the dedup) is written for OpenCode by name (lines 13, 25) yet installs on all six targets; `roadmap-reviewer.md:62-64` uses `mcp__hero__`-prefixed names where sibling agents use bare `hero_*`.

2. **hero-engine-repo context leaking into installed content.** Skills reference artifacts that exist only in *this* repo: `cross-repo-peering` points at `CROSS-REPO-PEERING.md` and `.hero/knowledge/{conventions,decisions}/...` (lines 14, 151, 161-163); `roadmap-review` quotes `internal/sizing/ambient.go` and the sibling spec `roadmap-review-ambient-surfacing` (lines 156, 204); `spec-composition` cites `internal/snapshot/rollup.go` and `multi-spec-design-routing` (lines 22-23, 83); the `drive` skill's arming step points at `scripts/drive/stop-hook.sh` (line 29) — unexecutable in any user workspace; pm skills cite `internal/vocabulary/` (pm-preset-detection:26,41), `internal/spec/` (handoff-protocol:127), `core/spec-types/` (story-writing-invest:118). `/release`'s mandatory `hero docs check` pre-flight (release.md:8-13) validates the hero repo's own README/GETTING-STARTED layout and errors or reports noise in every user project (F14).

3. **Inert or wrong frontmatter shipped everywhere.** `compatibility:` has no engine consumer (`internal/install/content.go` reads only agents' `domains:`), is stamped `opencode` on files installed into five other harnesses, and appears in three value formats. `role:` is read by nothing. Pack-agent `domains:` can never filter (install merges core + exactly one domain). Verified current counts post-dedup (commit `177e8a1` deleted 34 core↔engineering duplicates; `core/` is single master, `content_parity_test.go` gates shadows): **67** skill files with `compatibility:` (13 core + 35 engineering + 19 pm; 64 `opencode`, 2 comma-string, 1 YAML list in `domains/engineering/skills/code-scrub`), **16** agent files with `role:` (15 engineering + pm-reviewer), **13** pack agents with inert `domains:` (8 engineering + 5 sales; 0 on core agents, the only place it would be load-bearing).

Additionally, the slash/CLI parity table exists **only** in this repo's installed CLAUDE.md managed region (`CLAUDE.md:45-52`) — no pack source or Go fallback generates it, so the next `hero install` silently drops it; and its content has drifted (F22). Broken relative skill cross-links (`skills/next-md.md`-style) survived the dedup in `core/skills/next-handoff-emit/SKILL.md:20`, `core/skills/next-md/SKILL.md:3,16`, and `core/commands/handoff.md:30-31` (S3).

**The dual-edit constraint governs everything touching the engineering AGENTS.md body:** `domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody` in `internal/install/agents_md.go` (body renders at ~lines 379-483; Internal Lookups at 471-483) are kept content-identical by `TestEngineeringPackBodyMatchesGoFallback` — every change lands in both or CI fails.

## Goal

Every piece of shipped pack content works as written on all six install targets: harness-specific tools and mechanisms are named only inside explicitly scoped examples ("e.g. ... on Claude Code"); hook-dependent behavior carries a hookless fallback; no installed file references a path or artifact that exists only in the hero-engine repo; the inert `compatibility:`/`role:`/pack-`domains:` frontmatter is gone pack-wide; and the corrected slash/CLI parity table is generated from pack source so it survives reinstall. Verified by grep-zero checks, the Go-fallback parity test, `content_parity_test.go`, and a clean `hero install` smoke on two non-Claude targets.

## Kickoff

Sweeps Claude-Code-only assumptions and hero-repo dogfood leakage out of the shipped content packs so all six install targets get instructions that actually work for them, plus a scripted strip of three inert frontmatter fields (~96 files).

**Status:** planning — spec authored from audit findings; all paths verified against post-dedup tree (`177e8a1`).

**Pick up at:** Change 1 — rewrite the Internal Lookups section harness-neutrally in `domains/engineering/AGENTS.md` AND `internal/install/agents_md.go` together (test-enforced identical), then the parity table (Change 2).

→ `.hero/planning/initiatives/content-remediation/harness-agnosticism-sweep/spec.md`

**Files:** `domains/engineering/AGENTS.md:110-127`, `internal/install/agents_md.go:471-483`, `core/skills/next-md/SKILL.md`, `domains/engineering/skills/drive/SKILL.md:28-34`
**Skip:** adding engine semantics for `compatibility:`/`role:` — audit verdict is strip, not wire.

## Approach

Three passes, ordered so the dual-edit files are touched once.

**Pass 1 — routing surface (dual-edit pair).** Rewrite the harness-specific sections of the engineering AGENTS.md body and add the corrected parity table, editing `domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody` in lockstep. The pattern for scoping already exists in the same file (line 82 names `.claude/commands/` explicitly as a per-harness *example*): state the capability neutrally, then scope the harness-specific instance. MCP tool names use the bare `hero_*` form everywhere; the `mcp__hero__` prefix appears only inside a scoped "on Claude Code these surface as `mcp__hero__<name>`" aside.

**Pass 2 — targeted content edits.** Command phrasing (F23, F14), hook scoping paragraphs (next-md, next-handoff-emit, drive), dogfood-leakage rewrites (describe engine *behavior*, never engine *paths*), agent fixes (project-context-builder, roadmap-reviewer), broken cross-links (use skill names, not file links — installed layout is `<dest>/<name>/SKILL.md`).

**Pass 3 — mechanical frontmatter strip.** One scripted change over ~96 files. Frontmatter-aware (delete only keys inside the leading `---` block, including the multi-line YAML-list form in code-scrub), followed by grep-zero verification and the full test suite. Kept as the last commit so review of the semantic edits isn't drowned by the mechanical diff.

Content-only where possible: the two Go files touched are `agents_md.go` (string content of the fallback body — required by the dual-edit) and nothing else. The drive skill's stop-hook problem is fixed by *scoping the text* to what ships today (hooks exist on claude/codex; manual `hero goal <init> --check` loop elsewhere), not by building new hook-install machinery.

## Changes

1. **Rewrite "Internal Lookups — Tool Routing" harness-neutrally — `domains/engineering/AGENTS.md:110-127` AND `internal/install/agents_md.go` (`generateEngineeringAgentsMdBody`, ~lines 471-483), in lockstep.**
   - Table rows name bare `hero_*` MCP tools (`hero_search`, `hero_read_spec`, `hero_list`, `hero_queue`, `hero_blocked`, `hero_why`); one scoped sentence notes the Claude Code surface form (`mcp__hero__<name>`).
   - Replace the `Explore` agent row with capability phrasing: "a context-protective read-only search subagent, where your harness provides one (e.g. Claude Code's `Explore` agent); otherwise `rg` + targeted reads."
   - Replace the `ToolSearch`/deferred-tool-friction paragraph with a scoped one-liner ("some harnesses defer MCP schemas behind a one-time lookup — e.g. Claude Code's `ToolSearch`; the load is one round-trip and worth it") or cut it.
   - Run `go test ./internal/install -run TestEngineeringPackBodyMatchesGoFallback` after every edit to this pair.

2. **Move the corrected slash/CLI parity table into pack source (F22) — same dual-edit pair.**
   - Add the table (currently only in this repo's `CLAUDE.md:45-52` managed region) to `domains/engineering/AGENTS.md` + the Go fallback so reinstall regenerates it.
   - Fix omissions: `/resume`, `/blocked`, `/peer` move to **Both** (real CLI twins exist); `/roadmap-review` added to **slash-only**.
   - Annotate the two different-semantics rows: `/import` (slash = tracker import, CLI twin is `hero sync import`; root `hero import` is knowledge-base ingestion) and `/handoff` (slash = NEXT.md refresh; `hero handoff` = cross-repo drop) — keep them in Both but with the semantic split stated inline.
   - Fix the two stale skill paths in `core/commands/handoff.md:30-31` (`skills/next-md.md`, `skills/kickoff-prompt.md` → skill names; folded here per F22).

3. **Harness-neutral subagent phrasing in three engineering commands (F23).**
   - `domains/engineering/commands/diagnose.md:39` — "launch multiple Task agents" → "launch parallel subagents via your harness's delegation mechanism (e.g. Task agents on Claude Code)".
   - `domains/engineering/commands/deliver.md:188` — "Invoke a general-purpose subagent" → "Invoke a fresh subagent with no delivery context".
   - `domains/engineering/commands/mock.md:85` — "When the Agent tool completes" → "When the subagent completes"; keep the orchestrator-must-relay-links rule intact.

4. **Scope the Stop-hook machinery in the core handoff skills (skills S2).**
   - `core/skills/next-md/SKILL.md` — add one scoping paragraph after the two-halves table (~line 32): the checkpoint hook is installed for claude and codex only; on opencode/cursor/copilot/generic the agent runs `hero next checkpoint` itself at end of turn, and the "no agent discipline required" claim applies only where the hook exists.
   - `core/skills/next-handoff-emit/SKILL.md` — same paragraph adapted (~lines 43-46 auto-emit note): on hookless harnesses treat auto-emit as absent and use `hero next ask` explicitly; the existing `transcript_path` aside stays as the model of correct scoping.
   - Fix the broken relative links while in these files (S3): `next-handoff-emit/SKILL.md:20` `[skills/next-md.md](next-md.md)` and `next-md/SKILL.md:3,16` `(next-handoff-emit.md)` → refer by skill name ("the `next-handoff-emit` skill").

5. **De-dogfood the drive skill — `domains/engineering/skills/drive/SKILL.md:23-34` (+ frontmatter description line 3).**
   - Remove `scripts/drive/stop-hook.sh` (repo-only, zero engine references) from arming step 4; describe the shipped mechanism: on harnesses with Stop hooks (claude, codex — the same pair that gets `hero next checkpoint` wiring) the hook runs `hero goal <init> --check` each turn with `$HERO_DRIVE_INITIATIVE` set; on hookless harnesses the supervisor runs the check loop manually between turns.
   - Define or scope "the harness `/goal`": no pack ships a `/goal` command — rephrase as "your harness's loop/continuation mechanism, where one exists," and adjust the description frontmatter to match. If delivery finds the claude/codex drive-hook wiring doesn't actually exist in `internal/install/`, scope the text to manual-loop-everywhere and file the hook wiring as a follow-on (see Boundaries).

6. **De-dogfood cross-repo-peering — `domains/engineering/skills/cross-repo-peering/SKILL.md:14,151,161-163`.**
   - Inline the protocol depth the skill needs (it already calls itself "the operational distillation"); drop the pointers to `CROSS-REPO-PEERING.md` (workspace root), `.hero/knowledge/conventions/peering-protocol.md`, and `.hero/knowledge/decisions/cross-repo-peering-local-first.md`, or rephrase as "if your workspace carries a peering convention under `.hero/knowledge/`, read it" (conditional, not asserted-present).
   - Line 151's setup pointer becomes the command itself (`hero admin repos add <alias> <path>`), no doc reference.

7. **De-dogfood roadmap-review and spec-composition skills.**
   - `domains/engineering/skills/roadmap-review/SKILL.md:156,204` — state the ambient-hint and record-consumption behavior without `internal/sizing/ambient.go` or the `roadmap-review-ambient-surfacing` sibling-spec slug.
   - `domains/engineering/skills/spec-composition/SKILL.md:22-23,83` — describe the multi-spec design routing and the rollup midpoint-sum behavior without `multi-spec-design-routing` or `internal/snapshot/rollup.go`.

8. **De-dogfood pm skills.**
   - `domains/pm/skills/pm-preset-detection/SKILL.md:26,41` — "`internal/vocabulary/` resolver" → "Hero's vocabulary resolver (configured via `hero.json`)".
   - `domains/pm/skills/handoff-protocol/SKILL.md:127` — drop "(in `internal/spec/`)".
   - `domains/pm/skills/story-writing-invest/SKILL.md:118` — "from `core/spec-types/`" → "from the registered spec types".

9. **Fix the two harness-skewed agents.**
   - `core/agents/project-context-builder.md:13,25` (core-only post-dedup, installs on six targets) — "future OpenCode sessions" → "future agent sessions"; `opencode.json` `instructions` becomes a scoped example of "your harness's instruction-file mechanism (e.g. `opencode.json` `instructions` on OpenCode, `CLAUDE.md` imports on Claude Code)".
   - `domains/engineering/agents/roadmap-reviewer.md:62-64` — `mcp__hero__hero_warnings/hero_list/hero_search` → bare `hero_warnings`/`hero_list`/`hero_search`, matching every sibling agent.

10. **Gate the /release docs pre-flight (F14) — `domains/engineering/commands/release.md:8-13`.**
    - Make the `hero docs check` step conditional: "If the repo has a `hero docs check`-managed docs surface (README.md plus GETTING-STARTED.md and root `agents/`/`commands/`/`skills/` dirs), run it and fix findings; otherwise skip this step." No behavior change for the hero repo; no error path for user projects.

11. **Mechanical frontmatter strip — one scripted change, ~96 files (final commit).**
    - Strip `compatibility:` from all 67 skill files (`rg -l '^compatibility:' core domains` — 13 core, 35 engineering, 19 pm), handling all three value forms: `compatibility: opencode` (64), `compatibility: opencode, cursor, claude` (2: auto-knowledge-capture, note-capture), and the multi-line YAML list in `domains/engineering/skills/code-scrub/SKILL.md:4-7` (delete key + 3 item lines).
    - Strip `role:` from the 16 agent files (`rg -l '^role:'` — 15 in `domains/engineering/agents/`, plus `domains/pm/agents/pm-reviewer.md`).
    - Strip `domains:` from the 13 pack agent files (8 `domains/engineering/agents/`, 5 `domains/sales/agents/`) where it is provably inert (install merges core + own domain only; the field filters only on `core/agents/`, where no file sets it). Do **not** touch the `readAgentDomainsFrontmatter` engine code — the capability stays for core agents.
    - Implement as a small script (frontmatter-block-aware, not whole-file sed) run once and discarded or parked in `scratch/`; verify with `rg -c '^compatibility:|^role:' core domains` → zero and `rg -l '^domains:' domains/*/agents` → empty; then `go test ./...`.

## Boundaries

- **`routing-file-completeness` (sibling child, planned from audit theme "l") owns AGENTS.md roster/structure work**: routing rows for the 11 unlisted commands, agent/skill rosters, the three-pack skeleton and heading-depth alignment, install-dead link sweeps in pm/sales AGENTS.md. This spec touches the engineering AGENTS.md **only** for (a) the Internal Lookups harness-scoping and (b) the parity table — both inside the same dual-edit pair. Whichever spec lands second rebases over the other's AGENTS.md + `agents_md.go` edits; coordinate before starting delivery.
- **`agent-safety-conventions` (in flight, planning)** adds new harness-agnostic behavioral conventions to the AGENTS.md preamble — additive new content, while this spec corrects existing content; no shared line ranges, but same files (see Risks).
- **No engine semantics for stripped fields.** Wiring `compatibility:` into install filtering, or `role:` into roster generation, is explicitly rejected here (audit verdict: fossil — strip). A future spec can reintroduce either with a consumer.
- **No new hook machinery.** If the drive-skill fix (Change 5) reveals that no shipped Stop-hook wiring exists for `hero goal --check` on claude/codex, this spec scopes the *text* to manual supervision and files the hook install as a separate enhancement — it does not build it.
- **Content-accuracy findings stay out**: pm/sales phantom CLI invocations (F2/F3/F10), core/hero.md drift (F16), token-efficiency cuts (F12/F13/F17), chat pack (F9), `installFlat` README exclusion (code change) — all belong to other content-remediation children.
- **This repo's own installed CLAUDE.md** is not hand-edited; it regenerates from the pack source on the next `hero install` run here.

## Risks

- **Dual-edit lockstep.** Changes 1-2 must land in `domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody` simultaneously or `TestEngineeringPackBodyMatchesGoFallback` fails. Mitigation: edit pack file first, port to Go, run the test after each of the two changes — never batch both changes across both files in one blind pass.
- **Collision with in-flight AGENTS.md work.** `agent-safety-conventions` (planning, targets the AGENTS.md preamble on all harnesses) and `routing-file-completeness` both edit the same dual-edit pair. Risk is merge conflicts and fallback-test whiplash, not semantic conflict. Mitigation: sequence via the initiative; claim the pair in one delivery window.
- **Mechanical strip blast radius (~96 files).** A frontmatter-unaware script could delete body lines (code-scrub's list form is the trap) or break YAML. Mitigation: frontmatter-block-aware script, `git diff --stat` sanity (expected: 67+16+13 files minus overlaps — e.g. architecture-reviewer carries both `role:` and `domains:`), full `go test ./...` including `content_parity_test.go` and install snapshot tests, plus a `hero install` smoke.
- **Parity table re-drift.** The table drifted once because it lived outside pack source; moving it in fixes provenance but not future accuracy. Mitigation: acceptance check comparing table entries against `core/commands/` + `domains/engineering/commands/` + `hero --help` at delivery time; consider a docs-check assertion as a follow-on, not in scope.
- **Rewrites weakening working guidance.** The Internal Lookups section and mock/deliver phrasing encode hard-won anti-confabulation rules; neutralizing tool names must not blur the rules themselves. Mitigation: keep every rule sentence; only the tool nouns get scoped.
- **Install snapshot/golden tests.** Codex/copilot render paths (`renderCodexToml`, `renderCopilotPromptFile`) and any golden files may assert on current body text; expect to regenerate fixtures.

## Acceptance Criteria

- WHEN `hero install` runs for any of the six targets THE SYSTEM SHALL emit an instructions body whose Internal Lookups section names harness-exclusive tools (`Explore`, `ToolSearch`, `mcp__` prefixes) only inside explicitly scoped examples
- THE SYSTEM SHALL generate the slash/CLI parity table from the engineering pack source and Go fallback so a fresh install reproduces it without hand-edits
- THE SYSTEM SHALL list `/resume`, `/blocked`, and `/peer` as both-surface and `/roadmap-review` as slash-only in the parity table
- THE SYSTEM SHALL annotate the `/import` and `/handoff` parity rows with their differing slash-vs-CLI semantics
- WHEN a session on a hookless harness loads `next-md` or `next-handoff-emit` THE SYSTEM SHALL instruct it to run `hero next checkpoint` itself and to treat auto-emit as absent
- WHEN `/release` runs in a project without the hero-repo docs layout THE SYSTEM SHALL skip the `hero docs check` pre-flight instead of erroring
- THE SYSTEM SHALL ship no installed skill, agent, or command that references hero-engine-repo-only artifacts (`internal/…`, `scripts/…`, `core/spec-types/`, `CROSS-REPO-PEERING.md`, sibling spec slugs)
- THE SYSTEM SHALL phrase subagent delegation in diagnose.md, deliver.md, and mock.md in harness-neutral terms with harness names appearing only as scoped examples
- THE SYSTEM SHALL contain zero `compatibility:` keys in pack skill frontmatter and zero `role:` keys in pack agent frontmatter
- THE SYSTEM SHALL contain zero `domains:` keys in domain-pack agent frontmatter while `readAgentDomainsFrontmatter` remains functional for core agents
- WHILE CI runs THE SYSTEM SHALL pass `TestEngineeringPackBodyMatchesGoFallback` and `content_parity_test.go` with the revised content

## Validation

- `go test ./internal/install/...` — Go-fallback parity (`TestEngineeringPackBodyMatchesGoFallback`), hook-install, and render tests.
- `go test ./...` — full suite including root `content_parity_test.go` (no unannotated core↔domain shadows introduced).
- Grep-zero checks: `rg -c '^compatibility:' core domains` → no matches; `rg -l '^role:' core domains` → empty; `rg -l '^domains:' domains/*/agents` → empty; `rg -n 'mcp__hero__' domains core --glob '!**/AGENTS.md'` → only scoped-example occurrences; `rg -n 'internal/|scripts/drive|CROSS-REPO-PEERING' core domains` (skill/agent/command bodies) → no unscoped hits.
- Install smoke: `hero install project <tmpdir> --target cursor` and `--target codex` into a scratch workspace; confirm the managed AGENTS.md region contains the parity table, the neutral Internal Lookups section, and no `Explore`/`ToolSearch` outside scoped examples; confirm codex additionally gets its hook wiring while cursor content instructs manual `hero next checkpoint`.
- Manual read-through of the five rewritten skills (drive, cross-repo-peering, roadmap-review, spec-composition, next-md) confirming each rule survived with only tool/path nouns changed.
- `hero spec lint harness-agnosticism-sweep` — EARS classification clean.
