---
title: Claude Subagent Frontmatter Registration — Hero Agents Invisible to Task Tool
type: bug
status: completed
severity: high
priority: high
created: 2026-05-12
completed: 2026-05-12
tags: [install, claude, subagents, agents, registration, frontmatter, testing-gap]
---

# Claude Subagent Frontmatter Registration — Hero Agents Invisible to Task Tool

## Issue

After `hero install --target claude`, none of Hero's agents
(`feature-delivery-lead`, `platform-delivery-lead`, `engineer`,
`debug-investigator`, the scrubber pack, etc.) were registered as
callable `subagent_type` values in Claude Code's Task tool. Slash
commands that say "delegate to `<agent>`" silently fell back to
inline execution.

Observed in a real session running `/retro`:

> Subagent not wired in this harness — I'll do the retro inline.

Reproducible in the Hero source repo itself (which is Hero-installed
into Claude Code via `.claude/agents -> ../agents` symlink). A fresh
Claude Code session showed only the built-in `subagent_type` values:
`claude, claude-code-guide, Explore, general-purpose, Plan,
statusline-setup`. Zero Hero agents.

User-facing impact: every Hero slash command that delegates was
silently degraded to inline execution. That broke the entire
multi-agent workflow Hero is designed around — `/retro` ran without
a delivery lead, `/diagnose` ran without `debug-investigator`,
`/deliver` ran without `feature-delivery-lead`, `/review` ran
without specialized reviewers, `/scrub` couldn't dispatch the
scrubber pack. Most users did not notice the degradation because
nothing errored — they just got worse output than the design
intended.

Reporter: internal (Hero team, observed in another Claude Code
session). No tracker configured.

### Compounding meta-issue: install testing gap

This bug existed across every prior install of Hero into Claude Code
and was not caught by any test. The existing harness smoke tests
(`internal/install/harness_smoke_test.go`
`TestHarness_SmokeClaude`, `TestHarness_CanonicalAndSymlinks`)
verified **file existence** at the destination paths, but never
asserted that the installed files satisfied the consuming harness's
contract. Specifically, no test parsed the destination agent files
and checked that they met Claude Code's required-frontmatter schema
(`name:`, `description:`). That gap is what let this ship.

This spec covered both the code fix AND the missing verification
contract.

## Investigation

### Architecture context (preserved by the fix)

Per `multi-harness-install-collision` (commit 4fb54d5) and the
single-source-install P2 design documented in
`internal/install/target_claude.go:8-21`:

- Canonical content lives ONCE under `.hero/{agents,commands,skills}/`
  (or, for the Hero source repo itself, under top-level `agents/`,
  `commands/`, `skills/`).
- Per-target harness directories use **directory symlinks** pointing
  at the canonical tree. In this repo: `.claude/agents -> ../agents`.
- The fallback (when symlinks aren't supported) is `linkOrRenderDir`
  rendering a copy. That fallback also reads from canonical source,
  so the frontmatter contract is the same on both paths.
- This was an explicit anti-sprawl decision: **do not materialize
  translated copies per-harness**. One source of truth, symlinked.

The fix preserved this. No translated copies. No additional
materialization paths.

### Root cause

Claude Code's subagent registry requires at minimum `name:` and
`description:` keys in YAML frontmatter. Without `name:`, the file
is not registered as a callable subagent.

Every canonical agent file under `agents/` used
OpenCode/Crush-shaped frontmatter and omitted `name:`:

```yaml
---
description: Execute approved specs and implementation plans into ...
mode: subagent
role: execution
temperature: 0.1
color: primary
permission:
  edit: allow
  webfetch: allow
  skill:
    "*": allow
---
```

Claude Code read this file (via the `.claude/agents -> ../agents`
symlink), found no `name:`, and silently skipped registration. The
file was invisible to the Task tool.

OpenCode derives the agent name from the filename and was happy
without an explicit `name:` field. So the canonical-source files
were authored to satisfy OpenCode but never validated against
Claude Code's stricter contract.

### Why no test caught this

`internal/install/harness_smoke_test.go` `TestHarness_SmokeClaude`
verified `mustBeRegularFile` for each agent destination, but never:

- Parsed the resulting agent files.
- Asserted the YAML frontmatter had Claude Code's required keys.
- Round-tripped through any "would Claude Code accept this?"
  checker.

`TestHarness_CanonicalAndSymlinks` only checked the symlink
topology. There was no harness-contract test at all. Each target's
expected file shape was documented in code comments but never
enforced as a contract.

### Severity

- **Criticality**: high. Hero's entire delegation model was broken
  in Claude Code, the primary harness. Quality of every
  spec/diagnose output was degraded.
- **Frequency**: 100% of Claude Code users, every session, every
  delegating command.
- **Visibility**: low — Claude Code fell back gracefully to inline
  execution and did not always surface the "subagent not wired"
  message.
- **Caused by our code**: yes.

## Goal

After `hero install --target claude` (or working in any
Hero-installed Claude Code repo), every agent under `agents/` is
visible to Claude Code's Task tool as a callable `subagent_type`,
and a contract test prevents this class of regression for Claude
Code AND every other harness Hero installs into.

The single-source symlinked architecture is preserved. No
per-harness frontmatter translation. No additional materialized
copies.

## Changes (delivered)

### 1. Added `name:` to every canonical agent file

68 files updated across the three canonical content roots:

- `agents/*.md` (34 files)
- `core/agents/*.md` (4 files)
- `domains/engineering/agents/*.md` (30 files)

Each file received `name: <filename-stem>` as the first frontmatter
key, above `description:`. No other frontmatter fields were
modified — `mode:`, `permission:`, `temperature:`, `color:`,
`role:` continue to serve OpenCode and any future consumer.

### 2. Added a frontmatter contract test (root package)

New file [content_test.go](content_test.go) in the root `hero`
package, which owns the embedded canonical content via
`//go:embed`. The test walks each embedded FS
(`legacyContent`, `coreContent`, `engineeringContent`) under its
`agents/` root, parses YAML frontmatter, and asserts:

- `name:` is present, non-empty, and matches the filename stem.
- `description:` is present and non-empty.

Lives in the same package as the embedded content so there's no
import cycle with `internal/install/`. Catches the regression at
test time — any future agent added without the required fields
fails CI before release.

### 3. Added a Claude-Code-specific install assertion

Extended [internal/install/harness_smoke_test.go](internal/install/harness_smoke_test.go)
`TestHarness_SmokeClaude` with `mustBeRegisterableSubagent` calls
for `engineer.md` and `reviewer.md`, plus the helper itself in
[internal/install/harness_test.go](internal/install/harness_test.go).
The helper parses the installed file's frontmatter and asserts
`name == expectedName` and `description != ""`.

`seedSource` now writes proper YAML frontmatter for the seed
agents so the smoke test exercises the realistic shape.

### 4. Documented harness contracts (deferred)

The dedicated `HARNESS_CONTRACTS.md` doc was deferred — the
test in change 2 is the load-bearing enforcement and the
docstring at the top of `target_claude.go` already names the
relevant requirement. Add a separate doc as a followup if a
second harness target needs different rules.

### 5. Manual verification step in install output (deferred)

Skipped per the spec's low-priority note. The contract test
covers the regression path; user-facing install output stays
unchanged.

## Validation

- ✅ `go test ./...` — full suite passes.
- ✅ `go vet ./...` — clean.
- ✅ `go test . -run TestEmbeddedAgents` — new contract test passes
  for legacy/root, core, and engineering canonical roots.
- ✅ `go test ./internal/install/... -run TestHarness_Smoke` — extended
  Claude smoke test passes; opencode and cursor unaffected.
- ✅ Manual smoke 1 — fresh install:
  ```
  hero init && hero install project . --target claude
  ```
  Result: 34 agents installed. `head -3 .claude/agents/engineer.md`
  shows `name: engineer`. All 34 agents have `name:`. `.claude/agents`
  is a symlink to `../.hero/agents` — single-source architecture
  preserved.
- ✅ Manual smoke 2 — multi-harness collision regression guard:
  `hero install project . --target claude` then
  `hero install project . --target opencode` succeeds without
  `--force`. OpenCode sees 34 agents with `name:` present.
- ✅ Manual smoke 3 — OpenCode unaffected: existing
  `TestHarness_SmokeOpenCode` remains green; opencode still
  registers all agents (it derives name from filename and is
  agnostic to the new field).

## Acceptance Criteria — all met

- ✅ Every file under `.claude/agents/` (whether symlinked or
  rendered) contains `name:` matching the filename stem.
- ✅ Every Hero agent appears as a callable `subagent_type` in
  Claude Code's Task tool after install.
- ✅ Source contract test asserts `name:` and `description:` on
  every canonical agent.
- ✅ `TestHarness_SmokeClaude` parses installed agent frontmatter.
- ✅ Adding an agent without `name:` fails the new test.
- ✅ All OpenCode-specific frontmatter fields preserved.
- ✅ No per-harness frontmatter translation introduced; canonical
  symlinked architecture preserved.
- ✅ `TestHarness_SmokeOpenCode` remains green unchanged.

## Followups

(Out of scope here — separate specs.)

- **Audit harness-contract coverage end-to-end.** For every
  `target_*.go`, write down what shape of file each target consumes
  and add contract tests where missing. The Claude/agents bug is
  almost certainly mirrored in commands and skills, and probably
  in other targets we haven't checked.
- **Investigate whether `commands/*.md` has a parallel
  registration-contract gap in Claude Code.** Slash commands have
  their own frontmatter requirements; quick spot-check warranted.
- **Pre-commit hook or `hero spec lint` extension** to catch
  missing `name:` at author time rather than only at CI.
- **Document the canonical-symlink architecture decision in
  `.hero/knowledge/decisions/`** so future contributors understand
  why per-harness translation is off the table.

## Kickoff

Fixed. Hero agents now register as callable Claude Code subagents,
and a contract test prevents regression.

**Status:** completed — code committed in
`16e030f fix(agents): add name+description frontmatter to all agent files`,
full test suite green, manual smokes pass.

**What changed:**
- 68 canonical agent files updated with `name:` frontmatter
  (`agents/`, `core/agents/`, `domains/engineering/agents/`).
- [content_test.go](content_test.go) — new test enforces
  `name:` + `description:` on every embedded agent file.
- [internal/install/harness_test.go](internal/install/harness_test.go)
  — added `mustBeRegisterableSubagent` helper; updated
  `seedSource` to write realistic frontmatter.
- [internal/install/harness_smoke_test.go](internal/install/harness_smoke_test.go)
  — `TestHarness_SmokeClaude` now asserts subagent registration
  contract on installed agent files.

**Validation done:**
- `go test ./...` — full suite passes.
- `go vet ./...` — clean.
- Manual: `hero init && hero install project . --target claude` in
  throwaway dir; all 34 agents installed with `name:`, `.claude/agents`
  symlinked to `../.hero/agents`.
- Manual: `hero install project . --target opencode` after Claude
  install succeeds without `--force` (multi-harness regression guard).

**Why this matters:** every delegating slash command in Claude Code
had been silently degrading to inline execution. Quality of
`/retro`, `/diagnose`, `/deliver`, `/review`, `/design`, `/docs`,
`/scrub`, `/split`, `/check`, `/release` all step up now that
delegation actually delegates.

**Skip:** translating frontmatter at install time (rejected — keeps
single-source architecture), fixing commands/skills contract gaps
(separate spec), pre-commit hooks (followup), broader
harness-contract audit (followup).
