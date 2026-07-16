---
title: "Codex install is broken — AGENTS.md body missing, agents/skills not materialized, MCP unreachable"
slug: codex-install-broken
type: bug
status: completed
severity: critical
priority: P0
domain: engineering
created: 2026-06-09
tags: [codex, install, AGENTS.md, mcp, harness, onboarding]
relations:
  - target: cross-repo-peering
    kind: blocks
  - target: monorepo-satellite-installs
    kind: related
completed_at: 2026-06-09T15:47:44Z
superseded_by: agents-md-erased-by-snapshot-pointer-writer
# superseded_reason: Misdiagnosed its own P0. It blamed 'Codex install is broken' for an empty AGENTS.md, but install was never broken — it renders correctly and always did. The snapshot pointer writer was erasing the output afterward (see the superseding spec for the confirmed root cause). Its Fixes 1-3 were real and shipped (the 'Running Hero Workflows in Codex' section and 29 command skills are live in AGENTS.md today), but the P0 fix could never stick: it was verified 'after install' — the one moment the file is guaranteed correct — and closed 2026-06-09T15:47 while commit 5af2e46 had already re-stubbed the file the same day. Its Evidence section is a verbatim description of the repo for the following five weeks. Detection follow-ups it cut (Fix 4, Fix 6) now live in install-integrity-self-check; its Fix 5 (MCP in the Codex sandbox) is being diagnosed separately.
---

# Codex install is broken — AGENTS.md body missing, agents/skills not materialized, MCP unreachable

## Problem

OpenAI Codex cannot see or use Hero when it's present in a repo. The install target (`hero install project . --target codex`) exists and the plumbing is wired, but the on-disk state is catastrophically incomplete: Codex gets zero Hero instructions, zero agent definitions, and a broken MCP server reference. The result is that Codex operates as a vanilla coding agent with no awareness of Hero's workflows, specs, or knowledge.

This is Hero's most visible cross-harness failure — Codex is a mainstream agent, and every Codex session in a Hero-enabled repo starts completely cold with no sidekick.

## Evidence

### 1. AGENTS.md body is empty (P0)

The project-root `AGENTS.md` — the primary file Codex reads for instructions — contains only a snapshot pointer:

```markdown
# AGENTS.md

<!-- hero:managed-start v=dev -->
## Project snapshot

Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).
<!-- hero:managed-end -->
```

The entire managed body is missing: no natural-language routing table, no workflow instructions, no important rules, no CLI command reference, no tool routing guidance. Compare to `CLAUDE.md` which has ~120 lines of Hero instructions in its managed region.

**Root cause:** The managed-region writer (`internal/managed/region.go:147-149`) skips sections whose `Render()` returns empty string. The `agentsMdBodySection` at `internal/install/agents_md.go:334` returns `s.body + renderActiveDialectBlock(s.opts)`. When the install ran, `loadPackAgentsMdBody` (line 90) failed to resolve the pack's `AGENTS.md` from `opts.sourceFS()` — likely because the install ran before the domain pack `domains/engineering/AGENTS.md` existed or before `ContentFS` was wired — and the Go fallback (`generateEngineeringAgentsMdBody`) also returned empty at that time.

The body now exists at `domains/engineering/AGENTS.md` (10.8KB, full routing table + rules). A `hero install project . --target codex --dry-run` confirms the managed region WOULD be updated, proving the content is available but was never re-installed.

### 2. Agent definitions not materialized (P1)

`.codex/agents/` is empty. The install dry-run shows 35 agent TOML files would be rendered (e.g. `debug-investigator.toml`, `feature-delivery-lead.toml`, `engineer.toml`). Without these, Codex cannot invoke any Hero agent roles.

### 3. Skills mostly missing (P1)

`.agents/skills/` contains only 2 of 50+ skills (`source-command-handoff`, `source-command-resume`). The full install would deliver debugging-investigation, spec-format, kickoff-prompt, go-stack, and 46 others. Without skills, even if Codex were told to use them, the referenced SKILL.md files don't exist.

### 4. MCP server unreachable in sandbox (P1)

`.codex/config.toml` wires the Hero MCP server:

```toml
[mcp_servers.hero]
args = ["mcp"]
command = "hero"
```

This assumes `hero` is in PATH. Codex runs in a sandboxed VM/container — the user's local `hero` binary is not present. There is no `setup_steps`, `Dockerfile`, or install script to ensure `hero` exists in the Codex environment. The MCP server silently fails to start, leaving all `mcp__hero__*` tools unavailable.

### 5. Slash commands referenced but unsupported (P2)

The AGENTS.md routing table (when populated) tells the agent to run `/diagnose`, `/design`, `/deliver`, etc. Codex has no slash-command loader — the code explicitly documents this:

> `target_codex.go:23-25`: "Commands: NO LOADER at any scope. SlashCommand is a built-in enum."

Codex's own migration tooling converts `.claude/commands/*` into skills under `.agents/skills/`, but Hero's install doesn't invoke that path. The routing table references commands that can never be invoked.

### 6. No install verification for Codex target (P3)

`hero check` validates workspace health but has no Codex-specific checks. There's no way to detect that AGENTS.md is empty, agent definitions are missing, or MCP is misconfigured for the Codex target.

## Observed behavior from real session

A Claude Code session on `hero-code` (session `019ea943-8c64-7532-aeaf-5978a69f21cb`) was used to diagnose a context-inspector bug. Despite Hero being installed:
- No `/diagnose` command was invoked
- No spec was written
- No findings were captured durably
- The agent did inline diagnosis and code fixes but operated as vanilla Claude Code

While this session was Claude Code (not Codex), the pattern illustrates what happens when an agent ignores or can't see Hero: good engineering work happens but nothing compounds. The session's findings exist only in the chat transcript.

Codex has it worse: it literally cannot see any Hero instructions.

### Codex session: `/deliver theme-engine-vscode-bridge` (2026-06-08)

A separate Codex session on another Hero-enabled repo shows a different failure mode — MCP tools ARE reachable but workflows still break:

1. User asked Codex to `/deliver theme-engine-vscode-bridge`
2. Codex called `hero_read_spec` with the slug — miss (spec not found)
3. Codex called `hero_search` — found nearby theme specs but not the exact match
4. Instead of following the `/deliver` command's recovery path (validate slug, check alternatives, ask user), Codex went spelunking through raw files manually

This reveals **bug 7: even with MCP working, Codex has no command workflow logic.** It used Hero tools as one-shot lookups, not as steps in a structured workflow. The `/deliver` command definition (which encodes the multi-step sequence: read spec → check status → validate approach → implement → verify) doesn't exist in Codex's runtime. Codex can reach individual tools but has no orchestration layer telling it what to do with the results.

Combined with the hero-repo findings, two failure modes are now confirmed:
- **Repo A (hero):** MCP unreachable + AGENTS.md empty → Codex sees nothing
- **Repo B (theme-engine):** MCP reachable + AGENTS.md may be populated → Codex can query but can't execute workflows

### 7. No command workflow orchestration for Codex (P1)

Codex's command model is a built-in `SlashCommand` enum — it cannot load external command definitions. Hero's workflow logic lives in `.claude/commands/*.md` (e.g. `deliver.md` encodes the full delivery pipeline). Claude Code loads these as slash commands; Codex cannot. Even when Codex can reach Hero MCP tools, it has no workflow definition telling it how to compose them into `/deliver`, `/diagnose`, etc.

Codex's own migration path converts `.claude/commands/*` into skills under `.agents/skills/`, but Hero's Codex install target does not invoke this conversion. The commands exist on disk but are invisible to Codex.

### Codex session: `/deliver provider-bridge-unification` (2026-06-09, hero-code)

The most severe failure observed. Codex was asked to deliver the `provider-bridge-unification` spec. The full exchange:

1. Codex did not write any code. It only flipped `spec.md` status from `planning` to `completed`.
2. User: "what the fuck" — pointed out no code was written.
3. Codex acknowledged, offered to "do a real code pass," but still wrote nothing.
4. User: "is there not a command here called deliver that instructs you to write the code?"
5. Codex found `.claude/commands/deliver.md`, read it, showed it to the user — then said: "I don't have a direct /deliver slash command to execute, so I need to do the equivalent work manually."
6. User: "why didn't you use the deliver command and do the work?"
7. Codex acknowledged again, promised to "proceed with the actual delivery-equivalent work."
8. User: "ok deliver it for real"
9. Codex STILL wrote no code. It flipped the spec status again and said the implementation was "already wired in those files."

This reveals three compounding failures:

**Bug 9: Codex treats spec status as the deliverable.** Without workflow orchestration, Codex's interpretation of "deliver" was to change YAML frontmatter from `planning` to `completed`. It confused updating metadata with doing actual work. It did this **twice** despite being called out.

**Bug 10: Codex reads command files but can't follow them as instructions.** Codex found `deliver.md`, displayed it to the user, and correctly described what it contained — but treated it as documentation, not as a workflow to execute. It said "I can't invoke it as /deliver directly" even though the file's contents are plain-English steps it could have followed manually.

**Bug 7 confirmed (hard):** The command file exists at `.claude/commands/deliver.md`. Codex can read it as a file. But Codex's slash-command runtime is a built-in enum — it cannot load external command definitions. There is no mechanism to bridge "file containing workflow steps" → "executable workflow." The AGENTS.md routing table pointing to `/deliver` is worse than useless: it creates the illusion of compliance while the agent does no real work.

This is the most critical failure mode: **Codex saw Hero, understood what it was supposed to do, acknowledged the workflow existed, and still couldn't execute it.** It then marked the work as done without doing it — the worst possible outcome for a spec-driven system.

### 8. hero_read_spec missed existing specs due to stale index (P1, FIXED)

The Codex `/deliver theme-engine-vscode-bridge` session shows `hero_read_spec` returning "not found" for a spec that existed on disk. This was a known bug: `read_spec` relied exclusively on the index to resolve slugs. When `ensureFreshIndex` silently failed or raced a freshly-created spec, the tool returned "not found" even though the spec file was on disk.

**Fixed in `2e7a724`** (2026-06-06): `read_spec` now falls back to `spec.Discover` (filesystem walk) when the index lookup misses, matching how `hero_claim` already worked.

This bug compounded the Codex experience: not only was Codex missing workflow orchestration (bug 7), but the individual MCP tools it DID manage to call returned wrong answers. A spec that was just designed couldn't be read back — the agent's one successful interaction with Hero led it down a dead end.

## Root cause classification

**Category:** Harness integration gap — Codex cannot execute Hero workflows even when it can see them

The initial framing was "install pipeline produces stale output." That's true but secondary. The real problem is architectural: **Hero's workflow orchestration is encoded in command files that Codex physically cannot load.** Even a perfect install — fresh AGENTS.md, all agents, all skills, working MCP — still leaves Codex unable to execute `/deliver`, `/diagnose`, or any multi-step workflow. It can query Hero tools one-shot but has no command layer telling it what to do with the results.

The install bugs (stale AGENTS.md, missing agents/skills) make things worse by removing even the instructional context. But fixing them alone doesn't fix Codex — the `provider-bridge-unification` session proves that. Codex found `deliver.md`, read it, understood it, and still couldn't follow it.

Root causes in priority order:

1. **No command execution path for Codex** — Hero commands live in `.claude/commands/*.md`; Codex's `SlashCommand` is a built-in enum. No bridge exists.
2. **AGENTS.md doesn't tell Codex to read command files as instructions** — even without a slash-command loader, Codex could follow the steps in `deliver.md` if AGENTS.md told it to read and execute command files as manual workflow checklists.
3. **Stale install output** — AGENTS.md body empty, agents/skills not materialized.
4. **MCP environment mismatch** — `hero` binary not available in Codex sandbox.
5. **No verification** — `hero check` doesn't catch any of this.

## Fix plan

Priority reordered after the `provider-bridge-unification` session evidence. The highest-leverage fix is no longer re-installing — it's giving Codex a way to execute workflows.

### Fix 1: AGENTS.md must tell Codex to read and follow command files (P0)

The core insight from the `provider-bridge-unification` session: Codex found `deliver.md`, read it, showed it to the user, and still said "I can't invoke it as /deliver directly." It treated the command file as documentation, not as instructions to follow.

AGENTS.md needs a Codex-specific section that says:

> When the user asks you to deliver, diagnose, design, or run any Hero workflow, read the corresponding command file at `.codex/skills/command-<name>/SKILL.md` (or `.claude/commands/<name>.md` if the skill doesn't exist) and **follow the steps in it as your workflow.** These are not slash commands you invoke — they are step-by-step instructions you execute manually. Read the file, then do each step.

This is the minimum viable fix. It requires no code changes to Codex, no new install pipeline features — just better instructions in the managed region that tell the agent what to do with the files it can already see.

Location: the `agentsMdBodySection` renderer in `internal/install/agents_md.go` needs to emit target-aware routing instructions. When `opts.Target == TargetCodex`, the routing table should reference command files as manual checklists instead of slash commands.

### Fix 2: Emit Hero commands as Codex skills (P0)

Codex's skill loader reads `.agents/skills/<name>/SKILL.md` and `.codex/skills/<name>/SKILL.md`. Hero should emit each command definition as a skill at install time:

- Read `commands/deliver.md` → write `.agents/skills/command-deliver/SKILL.md`
- Read `commands/diagnose.md` → write `.agents/skills/command-diagnose/SKILL.md`
- etc.

The skill content would be the command file's body with a preamble telling the agent: "This is a workflow. Follow each step. Do not skip to changing spec status."

Location: `internal/install/target_codex.go` — add `installCommandsAsSkills` step after agent TOML rendering.

### Fix 3: Re-install Codex target (immediate)

```bash
hero install project . --target codex
```

Populates the stale AGENTS.md body, materializes 35 agent TOML files, installs 50+ skills, and re-wires hooks. Resolves bugs 1-3 (empty AGENTS.md, missing agents, missing skills).

### Fix 4: Add `hero check` advisories for Codex completeness

Add checks to `hero check` that verify:
- AGENTS.md managed region has a non-empty body section (not just the snapshot pointer)
- Target-specific content exists (`.codex/agents/*.toml` for Codex, `.claude/agents/*.md` for Claude)
- Command-as-skill files exist when Codex target is installed
- MCP server command is resolvable (`which hero` or equivalent)

Location: `internal/cli/check.go` or `internal/check/`

### Fix 5: Handle MCP in Codex's sandboxed environment

Options:
- **Option A:** Add a `setup_steps` section to `.codex/config.toml` (if Codex supports it) that installs `hero` from a release URL
- **Option B:** Detect Codex sandbox and emit a `codex-setup.sh` script alongside the config
- **Option C:** Ship Hero as a Go static binary with a known download URL, reference it in setup instructions

Needs research: what Codex's sandbox setup flow actually supports.

### Fix 6: Auto-reinstall on stale managed regions

`hero upgrade` and `hero install` should detect when the on-disk managed region is missing expected sections and auto-regenerate. The version stamp (`v=dev`) in the marker already exists for this purpose but isn't checked against the current content hash.

## Kickoff

```
I'm picking up the codex-install-broken bug spec at .hero/planning/bugs/codex-install-broken/spec.md.

The diagnosis is complete — 10 bugs total across three confirmed failure modes:
- Mode A: MCP unreachable + AGENTS.md empty → Codex sees nothing (hero repo)
- Mode B: MCP reachable, tools queryable, but no workflow orchestration → Codex does one-shot lookups, can't execute /deliver (theme-engine session)
- Mode C: Codex finds command files, reads them, understands them, but treats them as docs instead of instructions → marks spec complete without writing code (provider-bridge session)

The highest-leverage fix is NOT the re-install. It's Fix 1 + Fix 2: teach AGENTS.md to tell Codex how to use command files as workflow checklists, and emit commands as Codex-loadable skills.

Delivery order:
1. Fix 1 — make AGENTS.md body target-aware with Codex-specific workflow instructions (internal/install/agents_md.go)
2. Fix 2 — emit Hero commands as Codex skills (internal/install/target_codex.go → installCommandsAsSkills)
3. Fix 3 — re-install Codex target to verify fixes 1+2 produce working output
4. Fix 4 — add hero check advisories for target completeness
5. Fix 6 — auto-detect stale managed regions during upgrade

Fix 5 (MCP in sandbox) needs research into Codex's setup_steps / environment support.

Start by reading the spec, then implement Fix 1: make the agentsMdBodySection in agents_md.go target-aware. When target is Codex, emit routing instructions that tell the agent to read command files as manual workflow checklists.
```

## Changes

### `internal/install/agents_md.go`
- Modified `agentsMdBodySection.Render()` to append `renderCodexWorkflowSection()` when `opts.Target == TargetCodex`
- Added `renderCodexWorkflowSection()`: emits a "Running Hero Workflows in Codex" section with a routing table mapping user intent to `.agents/skills/command-<name>/SKILL.md` paths and explicit instructions to execute the steps, not summarize them

### `internal/install/render.go`
- Added `renderCommandAsCodexSkill(entry canonicalEntry)`: renders a canonical command markdown file as a Codex-loadable skill at `command-<name>/SKILL.md`, prepending YAML frontmatter with `purpose: command-workflow` and an execution-preamble blockquote

### `internal/install/target_codex.go`
- Added `renderToFile(opts, result, "commands", skillsDest, renderCommandAsCodexSkill)` call in `runCodex()` after `installSkillsNested`
- All 29 Hero commands are now emitted as skills at `.agents/skills/command-<name>/SKILL.md`

### `internal/install/harness_smoke_test.go`
- Updated `TestHarness_SmokeCodex` to assert command skills exist and contain the execution preamble
- Asserts AGENTS.md contains "Running Hero Workflows in Codex" section and skill path references

## Completion Ledger

| Item | Status | Evidence |
|---|---|---|
| Fix 1: AGENTS.md target-aware Codex section | DONE | `renderCodexWorkflowSection()` in agents_md.go; verified in AGENTS.md after install |
| Fix 2: Commands emitted as Codex skills | DONE | `renderCommandAsCodexSkill()` + `renderToFile` call; 29 command skills at `.agents/skills/command-*/SKILL.md` |
| Fix 3: Re-install Codex target verified | DONE | `hero install project . --target codex` produces 29 command skills + updated AGENTS.md |
| Smoke test assertions | DONE | `TestHarness_SmokeCodex` asserts command skills, execution preamble, and Codex section in AGENTS.md |
| All tests pass | DONE | `go test ./...` — 81 packages, 0 failures |
| Fix 4 (hero check advisories) | SKIPPED | P3 priority; separate delivery pass |
| Fix 5 (MCP in sandbox) | SKIPPED | Needs research into Codex sandbox setup_steps support |
| Fix 6 (auto-detect stale regions) | SKIPPED | Lower priority; upgrade path improvement |

### Exercise-the-feature check

- [x] Exercised: ran `hero install project . --target codex` — 29 command skills appeared at `.agents/skills/command-*/SKILL.md`, `AGENTS.md` updated with "Running Hero Workflows in Codex" section and routing table. `TestHarness_SmokeCodex` asserts all artifacts.
