---
title: Domain Routing and Agents — Active-Pack AGENTS.md and Agent Loader
slug: domain-routing-and-agents
type: feature
status: completed
priority: P0
tags: [platform, domains, routing, agents, refactor]
created: 2026-05-15
designed: 2026-05-19
relations:
  - target: hero-domains
    kind: parent
depends-on:
  - domain-plugin-architecture
horizon: next
smoke: deferred
completed_at: 2026-05-19T22:38:13Z
---

## Kickoff

Make agent routing and agent discovery domain-aware. The natural-language
routing table and the agent roster live in the **active domain pack**, not
hardcoded in `internal/install/agents_md.go`. A PM project routes "ship
this" → `/handoff` (PM canonical) instead of `/deliver` (engineering), and
`feature-delivery-lead` does not get materialized into `.claude/agents/`
when a PM project runs `hero install`.

**Status:** designed — 2026-05-19. Loader resolution order locked. The
routing table moves from Go-string-builder code into the pack's
`AGENTS.md` body as the source of truth; `internal/install/agents_md.go`
becomes a loader that splices that body into the managed region rather
than synthesizing it. Engineering content already exists at
`domains/engineering/AGENTS.md` on disk (committed) — the same file
becomes load-bearing.

**Pick up at:** `/deliver domain-routing-and-agents`. Phase 1 is the
loader cutover (engineering only; behavior bit-identical to today).
Phase 2 wires PM through the same loader. Phase 3 narrows the agent
materialization so packs don't leak engineering agents into PM
workspaces.

→ `/deliver domain-routing-and-agents`

**Files:** .hero/planning/features/domain-routing-and-agents/spec.md,
.hero/planning/initiatives/hero-domains/spec.md, content.go,
domains/engineering/AGENTS.md, domains/pm/AGENTS.md,
domains/sales/AGENTS.md, internal/install/agents_md.go,
internal/install/claude_md.go, internal/install/content.go,
internal/install/dialect.go, internal/install/install.go,
internal/install/target_claude.go, internal/install/target_codex.go,
internal/install/target_copilot.go, internal/install/target_cursor.go,
internal/install/target_generic.go,
internal/install/target_opencode.go, internal/cli/install.go,
internal/cli/domain.go.

**Skip:** Multi-active-domain coexistence in a single workspace
(handled by `domain-scoped-knowledge-graph`). Third-party domain packs
loaded from disk. Per-user agent overrides. Cross-pack agent sharing
beyond the universal `core/agents/` layer that already exists.

## Goal

Move the natural-language routing table and the agent roster out of
hardcoded repo-root files and Go-string-builder code into the active
domain pack. The agent loader reads from the active pack's
`domains/<active>/agents/` (merged with `core/agents/`) and splices the
active pack's `AGENTS.md` body into the managed region of the project's
top-level `AGENTS.md` and `CLAUDE.md`. A PM project routes to PM
agents (`product-strategist`, `story-writer`, `roadmap-curator`,
`intake-triager`) and presents the PM-shaped routing table; an
engineering project keeps today's behavior bit-identical.

The pack's `AGENTS.md` is the **single source of truth** for routing
content. Go code is responsible for path resolution, managed-region
splicing, and dialect block injection — not for the routing-table
text itself.

## Why now

Without this, two things break the moment any non-engineering domain
is active:

1. **Routing voice is wrong.** `internal/install/agents_md.go:285-312`
   builds the engineering routing table inside `generateAgentsMdBody`
   with `sb.WriteString(...)` calls. Every install — regardless of
   `opts.Domain` — writes that same engineering table into AGENTS.md
   and CLAUDE.md. A PM workspace gets engineering verbs.
2. **Agent surface is wrong.** Today's content materialization
   (`target_claude.go:30`, `target_opencode.go:33`,
   `target_generic.go:28`, etc.) calls `installFlat(opts, result,
   "agents", dest)` against `opts.ContentFS`, which already swaps by
   domain. But the PM pack's `domains/pm/agents/` is partial — and
   nothing in the install pipeline excludes engineering agents that
   live in `core/agents/` or that might still be present in the
   active pack's directory. The intent is "engineering agents
   disappear from PM workspaces"; the implementation today doesn't
   enforce that.

The parent initiative names this primitive as P0 — without it, the
PM domain pack ships but the user still sees engineering routing and
an engineering-shaped agent list, and the platform narrative
("Hero adapts to your function") fails on first contact.

## Design

### Loader resolution order

When `hero install` runs (project mode), the loader resolves the
**active pack's** `AGENTS.md` body content through a fixed chain:

```
1. Explicit override (highest priority)
   - opts.AgentsMdBodyOverride []byte — used by tests + future
     third-party-pack loader. When set, skip all FS lookups.
2. Active pack on disk
   - <opts.PackContentFS>/AGENTS.md      (if pack FS provided as Sub)
   - or:  domains/<opts.Domain>/AGENTS.md inside opts.ContentFS
3. Engineering fallback
   - domains/engineering/AGENTS.md inside the embedded engineeringContent
4. Hardcoded fallback
   - generateEngineeringAgentsMdBody() — the existing Go-string-builder
     code in agents_md.go, kept as the "embedded binary lost its
     filesystem" floor. This is what today's installer always emits;
     after this work it is a last-resort safety net only.
```

The resolved bytes are the **body** that gets spliced into the managed
region of the project's `AGENTS.md` and `CLAUDE.md`, the same way
`generateAgentsMdBody` produces a body today. The H2 wrapper
(`## Hero — Spec-Driven AI Engineering` for engineering;
`## Hero PM — Spec-Driven AI Product Management` for PM) comes from
the pack file's H1 line — the loader strips the file's leading `# ...`
H1 and uses the H1 text (minus the leading `# `) as `SectionTitle()`
for the managed-region orchestrator. This keeps the existing
`managed.Writer` flow byte-identical — only the body content changes
per domain.

**Why route through the pack's `AGENTS.md` file rather than a
structured (YAML/JSON) routing-table file:** the pack's `AGENTS.md` is
already what the user reads and edits (`domains/pm/AGENTS.md` exists
on disk today, committed, with the PM routing table written in
markdown). Forcing a second structured-file source of truth would
double the surface area for pack authors and force the loader to
re-render markdown from data — a layer of indirection with no
upside. The routing table's audience is the model reading
markdown; markdown is the natural shape.

If a future need emerges to query the routing table programmatically
(e.g. a `hero route "ship this"` CLI that resolves intent without
loading the model), add a structured **shadow** file
(`domains/<name>/routing.yaml`) without changing this design. The
shadow becomes the queryable form; the markdown table in
`AGENTS.md` stays the user-facing source.

### What stays in the install Go code

`internal/install/agents_md.go`'s **structure** stays — the managed
region, the H2 owned by the orchestrator, the section composer, the
snapshot pointer contributor, the dialect block appended at the end.
The Go code remains responsible for:

1. **Splicing.** Wrapping the pack body inside the managed-region
   markers in the project's `AGENTS.md` / `CLAUDE.md`.
2. **Dialect injection.** `renderActiveDialectBlock(opts)` continues
   to append the "Active workspace dialect" section after the pack
   body (see `dialect.go:21`). This is workspace-level state
   (`cfg.Vocabulary`, `cfg.Methodology`) and remains a workspace
   concern, not a pack concern.
3. **Snapshot pointer.** `snapshot.NewPointerSection(...)`
   continues to append the `## Project snapshot` pointer
   (`agents_md.go:71`).
4. **Pack-not-found fallback.** When the active pack's `AGENTS.md`
   is missing (or empty / corrupt), fall through to
   `generateEngineeringAgentsMdBody()` and emit a one-line warning
   to stderr: `warning: domain "<name>" has no AGENTS.md — falling
   back to engineering routing table`.

### What moves out of the install Go code

`generateAgentsMdBody(paths contentPathsForBody) string` is renamed
to `generateEngineeringAgentsMdBody(paths contentPathsForBody) string`
and becomes the engineering-pack fallback only. Its content does **not**
change in this work — it is bit-identical to today. The
**non-fallback** path is the new pack loader.

The `contentPathsForBody` struct stays (the engineering fallback
still uses it). The non-fallback path renders whatever the pack's
`AGENTS.md` file says about its directory layout; pack authors are
responsible for keeping that section accurate.

### Engineering content migration

`domains/engineering/AGENTS.md` already exists on disk (committed),
and its content is what `generateEngineeringAgentsMdBody` would emit
today plus a small amount of pack-specific phrasing already in the
file. The migration is a **content reconciliation** rather than a move:

1. Diff `domains/engineering/AGENTS.md` against the body that
   `generateEngineeringAgentsMdBody()` produces today. Reconcile in
   favor of the on-disk file (it's already the desired shape; the Go
   code is the legacy). One round of byte-comparison plus targeted
   edits.
2. Add a one-time test that asserts the two stay in lockstep until the
   Go fallback is removed:
   `internal/install/agents_md_test.go::TestEngineeringPackBodyMatchesGoFallback`.
   The test fails when either diverges — caught in CI before either
   becomes silently wrong.
3. Once the loader is in place and reading from the on-disk file,
   delete `generateEngineeringAgentsMdBody()` and the parity test
   in a follow-up PR. This work keeps the fallback as a safety net;
   removal is a clean follow-up after a release window where no
   bug reports surface.

**Repo-root `AGENTS.md`** (the file at `/AGENTS.md` in the hero repo —
the one this CLAUDE.md describes as committed today) is a *workspace
file in the hero project itself*, not the engineering pack's source.
The repo-root file already has a managed region written by the
`hero install` that the hero project itself runs. That file is
**unchanged** by this work — the same managed region keeps getting
re-written; the only difference is what gets spliced inside it
(now sourced from `domains/engineering/AGENTS.md` rather than from
inline Go strings).

### Agent loader (materialization narrowing)

Today the install pipeline materializes whatever's in `opts.ContentFS`
into the harness directory (e.g. `.claude/agents/`, `.opencode/agents/`).
`opts.ContentFS` already swaps with domain (see `install.go:183-194`),
so a `--domain pm` install reads from `domains/pm/agents/` — that part
works. What doesn't work is the **engineering leakage** path: the
PM pack's directory under `domains/pm/agents/` may or may not include
copies of universal agents (engineer, reviewer, etc.) and there is
no contract about what should appear there. Today, if a PM pack ships
without an agent named `engineer`, the PM workspace gets no `engineer`
agent; if it ships with one, that's an engineering agent that
shouldn't be in a PM workspace.

The narrowing rule:

1. **Active pack agents.** `domains/<active>/agents/` is materialized
   verbatim. Whatever the pack author ships is what the user gets.
2. **Universal core agents.** `core/agents/` is also materialized
   (the `coreContent` embed already exists; see `content.go:25`).
   Core agents are domain-agnostic by contract (e.g. an agent that
   only orchestrates spec-format and tracker conventions). Each core
   agent declares `domains: [*]` in its frontmatter; the loader skips
   any core agent whose `domains` field exists and doesn't list
   `<active>` or `*`. This is the seam by which a future agent can
   restrict itself to specific packs without ending up in unrelated
   workspaces.
3. **No cross-pack leakage.** Engineering pack agents are **never**
   materialized into a PM workspace, regardless of whether the
   PM pack itself ships a stub. The loader reads only
   `<active>` + `core`. The current Go code (`target_claude.go:30`
   etc.) already enforces this because `opts.ContentFS` is the
   domain-scoped FS — the bug is theoretical at the install layer
   today; the new test
   (`internal/install/install_test.go::TestPMInstallExcludesEngineeringAgents`)
   makes the invariant explicit so future content drift can't
   reintroduce the leak silently.

The merge happens at materialization time, not at routing time. The
end-state on disk under `.claude/agents/` is the union of
`core/agents/` (filtered by `domains:` frontmatter if present) and
`domains/<active>/agents/`. Name collisions are resolved
**pack-wins-over-core** — a pack may shadow a core agent by shipping
an agent with the same filename. This is the same precedence model
used by skills already.

### What happens when a user types a verb the active pack doesn't recognize

The active pack's routing table is **authoritative**. If the user
types "ship this" in a PM workspace and the PM table has no entry
matching "ship" (it instead has `/handoff` mapped to "hand off, send
to engineering, ready for dev"), the model:

1. **Does not silently route to another pack's table.** There is no
   global cross-pack fallback table. Other packs' verbs are not
   discoverable from inside the active pack.
2. **Asks the user.** The model presents the top 2-3 closest matches
   from the active pack's table and asks which they meant. This is
   already the behavior CLAUDE.md prescribes today
   ("If the intent is ambiguous, present the top 2-3 options and
   ask").
3. **Suggests `hero domain switch <name>` only when the user's verb
   is structurally engineering-shaped in a PM workspace.** The
   pack's `AGENTS.md` may include a one-line opt-in hint:

   > If the user's intent looks engineering-shaped (deploy, refactor,
   > debug a stack trace), suggest `hero domain switch engineering`
   > or `hero peer call <eng-alias>` instead of routing locally.

   This sentence is **pack-authored** content — engineering ships
   its own version aimed at PM-shaped asks. The loader does not
   inject this; it's just guidance to pack authors and lives in
   each pack's `AGENTS.md`.

The rejection-not-fall-through stance is the right v1 default
because:

- Silent cross-pack routing breaks the "pack is the unit of swap"
  invariant from the parent initiative.
- A PM user asking the model to "refactor this Go file" probably
  doesn't actually want it routed to engineering — they're in the
  wrong tool. Asking forces clarity.
- The pack switch is cheap (`hero domain switch engineering`
  re-installs content from the engineering pack; `.hero/` data is
  preserved per `domain-plugin-architecture`).

### `hero install` provisioning (the full picture)

The provisioning sequence when `hero install` runs (project mode):

```
1. Resolve domain.
   opts.Domain (flag) > cfg.Domain (hero.json) > "engineering"
   This logic already exists in internal/cli/install.go:180-195.
2. Resolve ContentFS.
   hero.DomainFS(domain) returns the FS rooted at domains/<domain>/
   (or legacyContent for engineering today; the legacy
   fallback decision in domain-plugin-architecture stays in place).
3. Materialize agents/commands/skills.
   For each per-target installer, installFlat reads from opts.ContentFS
   into the harness directory. Core overlay merge happens here
   (Phase 3 of this work; see below).
4. Render AGENTS.md / CLAUDE.md.
   installAgentsMd / installClaudeMd splice the pack body into the
   managed region. The pack body is now loaded from
   domains/<domain>/AGENTS.md (Phase 1 of this work). The dialect
   block and the snapshot pointer are appended after the pack body
   as today.
5. Register MCP server, register project, stamp install version.
   Unchanged.
```

**`hero domain switch <name>` reuses this path.** The existing
`runDomainSwitch` (`internal/cli/domain.go:77-129`) already
re-invokes `install.Run` for every installed harness target after
writing the new `cfg.Domain`. The only behavior change after this
spec lands is that the spliced AGENTS.md body is now the new
pack's body — `runDomainSwitch` itself is unchanged.

### Backwards compatibility — workspaces without `domain` in `hero.json`

`cfg.Domain == ""` resolves to `"engineering"` everywhere in the
codebase already (see `domain.go:49-52`, `install.go:183-194`,
`domain.go:61-64`). This work preserves that contract. After this
ships:

1. A workspace whose `hero.json` has no `domain` key continues to
   render the engineering routing table — sourced from
   `domains/engineering/AGENTS.md` (Phase 1) which is content-
   identical to what `generateEngineeringAgentsMdBody` emits today.
2. A workspace whose `hero.json` has `domain: pm` renders the PM
   routing table — sourced from `domains/pm/AGENTS.md`.
3. A workspace that's never run `hero install` against this binary
   version: untouched. The managed region on next install picks up
   the new body. Outside-the-region content is preserved
   byte-for-byte by `installManagedMarkdown`.

There is **no migration step for users**. A re-run of `hero install`
on an existing workspace just regenerates the managed region with
the new pack-sourced content. The version bump in the managed-region
marker (`<!-- hero:managed-start v=<heroVersion> -->`) ensures the
region gets refreshed even when its content shape changes.

### Routing-table format — markdown stays, structured form deferred

The pack's `AGENTS.md` body is plain markdown. The routing table
inside it is a markdown table with two columns (`User intent` |
`Command`), exactly as it appears in
`domains/engineering/AGENTS.md` today.

This work does **not** introduce a structured routing-table file
(`routing.yaml`, etc.). Rationale:

- The audience is the model reading the markdown body of
  `AGENTS.md` at session start. The model parses markdown tables
  natively.
- No code path today consumes the routing table as data. There is
  no `hero route <intent>` CLI that would benefit from a structured
  form.
- Adding a second source of truth doubles the surface area for
  pack authors and creates a synchronization burden (markdown
  drifts from YAML the moment one is edited without the other).

If a downstream feature (e.g. a deterministic intent classifier in
the daemon) needs a structured form, the right move is to **derive**
the YAML from the markdown table at pack-build time, with the
markdown remaining the source of truth. That deferral is recorded
in Risks.

### Phase 1 — Pack loader (engineering only)

One PR. Behavior bit-identical to today; the loader path is wired
but only the engineering pack actually flows through it.

1. `internal/install/agents_md.go` — extract `loadPackAgentsMdBody(opts
   Options) (body string, packTitle string, err error)`. Implements
   the resolution chain above. The legacy
   `generateEngineeringAgentsMdBody()` becomes a private fallback.
2. `internal/install/agents_md.go` — `newAgentsMdBodySection` reads
   the body via the loader and uses `packTitle` as `SectionTitle()`.
   When `packTitle == ""` (loader fell through to the Go fallback),
   the existing `"Hero — Spec-Driven AI Engineering"` H2 is used as
   today.
3. `internal/install/dialect.go` — no changes. `renderActiveDialectBlock`
   already returns its block as appended bytes; the new loader
   concatenates it after the pack body.
4. `domains/engineering/AGENTS.md` — reconcile against the current
   Go fallback output. Targeted edits only — preserve the existing
   file's structure.
5. `internal/install/agents_md_test.go` — new
   `TestEngineeringPackBodyMatchesGoFallback`. Builds the loader
   body against engineering and the Go fallback body; asserts
   string equality modulo trailing newlines. CI gate.
6. Run `hero install` against the hero repo itself. Diff the
   resulting `AGENTS.md` and `CLAUDE.md` managed regions before
   and after. Expected: zero diff (modulo the H2 source moving
   from `SectionTitle()` to the file's H1 — content-identical).

### Phase 2 — PM and Sales packs flow through the loader

One PR per non-engineering pack (PM is the only populated one for
v1; sales has a stub `AGENTS.md`).

1. `domains/pm/AGENTS.md` — already exists; the file's body is what
   the PM workspace's managed region will splice. Audit for
   markdown-table shape and for the "translate display terms back
   to canonical" guidance.
2. `domains/sales/AGENTS.md` — already exists as a scaffold; leave
   as-is. Sales pack is non-populated per the embed.go comments.
3. Manual smoke: `hero init --domain pm /tmp/pm-test` (or `hero
   domain switch pm` on a throwaway workspace). Inspect
   `/tmp/pm-test/AGENTS.md` — managed region body is sourced from
   `domains/pm/AGENTS.md`.
4. Manual smoke: `hero domain switch engineering` on the same
   workspace. Managed region flips back to engineering content.
5. New test:
   `internal/cli/domain_test.go::TestDomainSwitchUpdatesManagedRegion`.
   Stand up a workspace with the engineering domain, install,
   switch to PM, assert AGENTS.md managed region contains the PM
   table header (`| New feedback, customer ask, support escalation...`).

### Phase 3 — Agent materialization narrowing

One PR.

1. `internal/install/content.go::installFlat` — when `kind ==
   "agents"`, filter entries by frontmatter `domains:` field.
   An agent file with `domains: [engineering]` is materialized
   only when the active domain is engineering; absent field means
   "all domains" (today's behavior).
2. Add `domains:` frontmatter to a small handful of unambiguously
   engineering-only agents in `domains/engineering/agents/` —
   `feature-delivery-lead`, `debug-investigator`, `database-engineer`,
   `devops-engineer`, `release-engineer`, `dependency-analyst`,
   `migration-engineer`, `architecture-reviewer`. These are the
   agents that have **no PM counterpart** and should never appear
   in a PM workspace.
3. `core/agents/` — every agent in this directory keeps its existing
   `compatibility: opencode` frontmatter (which is harness-targeting,
   not domain-targeting). If a `core/agents/` file should not appear
   in PM, add `domains: [engineering]` to its frontmatter as part of
   this PR. (Survey reveals a handful of candidates; left to the
   delivery to enumerate at the moment of edit.)
4. Core agents in `core/agents/` and pack agents in `domains/<active>/agents/`
   are merged with **pack-wins-over-core** filename precedence.
   Implement in `installFlat` by reading core first, then pack,
   with a per-filename last-write-wins map.
5. New test:
   `internal/install/install_test.go::TestPMInstallExcludesEngineeringAgents`.
   Materialize PM into a temp dir; assert
   `feature-delivery-lead.md` is not present in
   `<dest>/.claude/agents/`.
6. New test:
   `internal/install/install_test.go::TestPackAgentShadowsCoreAgent`.
   Stand up a fixture with a core agent and a domain agent of the
   same name; assert the domain agent's bytes land at the
   destination.

### Routing test — does the model actually route differently

Smoke test the actual model loop, not just the installer:

1. Author `.hero/planning/features/domain-routing-and-agents/smoke.md`
   — a paste-into-Claude-Code script that runs in a temp workspace
   with `domain: pm`, asks "ship this", and asserts the model
   either (a) routes to `/handoff` (PM canonical) or (b) asks for
   clarification with the top 2-3 options drawn from the PM
   routing table — NOT `/deliver`.
2. This is a manual smoke today; if `hero smoke` evolves into a
   driver for in-CI model smokes, the same script gets wired in.

## Acceptance Criteria

- WHEN `hero install` runs against an engineering-domain workspace THE SYSTEM SHALL splice the body of `domains/engineering/AGENTS.md` into the managed region of `AGENTS.md` and `CLAUDE.md`.
- WHEN `hero install` runs against a `pm`-domain workspace THE SYSTEM SHALL splice the body of `domains/pm/AGENTS.md` into the managed region of `AGENTS.md` and `CLAUDE.md`.
- WHEN the active domain pack has no `AGENTS.md` file THE SYSTEM SHALL fall back to `generateEngineeringAgentsMdBody()` AND print `warning: domain "<name>" has no AGENTS.md — falling back to engineering routing table` to stderr.
- WHEN a workspace's `hero.json` has no `domain` key THE SYSTEM SHALL behave identically to an `engineering`-domain workspace.
- WHEN `hero domain switch pm` runs against a workspace currently on the engineering domain AND a harness target is installed THE SYSTEM SHALL rewrite that target's `AGENTS.md` and `CLAUDE.md` managed regions to contain the PM routing table.
- WHEN `hero install` runs against a `pm`-domain workspace THE SYSTEM SHALL NOT materialize `feature-delivery-lead.md`, `debug-investigator.md`, `database-engineer.md`, `devops-engineer.md`, `release-engineer.md`, `dependency-analyst.md`, `migration-engineer.md`, or `architecture-reviewer.md` into the harness `agents/` directory.
- WHEN a `core/agents/` file declares `domains: [engineering]` in its frontmatter AND `hero install` runs against a `pm`-domain workspace THE SYSTEM SHALL NOT materialize that file into the harness `agents/` directory.
- WHEN a `core/agents/` file and a `domains/<active>/agents/` file share a filename THE SYSTEM SHALL materialize the bytes from the `domains/<active>/agents/` file (pack-wins-over-core).
- WHEN the `Active workspace dialect` block is present (because `hero.json` declares a vocabulary or methodology) THE SYSTEM SHALL append it after the pack-sourced body inside the same managed region.
- WHEN the snapshot pointer section is appended THE SYSTEM SHALL place it after both the pack body and the dialect block, preserving today's section ordering.
- THE SYSTEM SHALL keep `domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody()`'s rendered body content-identical until the Go fallback is removed, enforced by `TestEngineeringPackBodyMatchesGoFallback`.
- WHEN content outside the `<!-- hero:managed-start v=... -->` markers exists in a project's `AGENTS.md` or `CLAUDE.md` THE SYSTEM SHALL preserve that content byte-for-byte across the loader cutover.

## Boundaries

- **Not** designing PM agents or PM commands — `hero-pm` owns the PM
  content pack.
- **Not** supporting multiple active domains in one workspace —
  `domain-scoped-knowledge-graph` handles read-side cross-domain
  concerns; this spec stays single-active.
- **Not** changing the agent contract (system prompt + tool surface).
  Only how agents are **discovered**, **filtered**, and **routed to**.
- **Not** introducing a structured routing-table file format. Markdown
  table in `AGENTS.md` stays the source of truth. A future derived
  YAML may exist; that's not this spec.
- **Not** touching skills — `domains/<active>/skills/` is already
  materialized today and the skill discovery seam is already
  domain-aware (see `internal/cli/install.go:188-195` and
  `installSkillsNested`).
- **Not** introducing third-party domain packs loaded from disk.
  Future work; the loader's `AgentsMdBodyOverride` seam is shaped to
  accept it without redesign.
- **Not** changing the global-mode (`hero install --global`) flow —
  the global-mode AGENTS.md continues to render the engineering body
  because global installs are not associated with a workspace
  `hero.json` and have no active domain.
- **Not** renaming any existing CLI or MCP commands. Routing-table
  entries point at the same `/design`, `/deliver`, `/diagnose` slash
  commands engineering uses today; PM commands like `/refine`,
  `/triage`, `/handoff` live alongside, not replacing.
- **Not** rewriting the repo-root hero project's own `AGENTS.md` —
  it's a workspace file like any other and picks up the new managed
  region content on the next `hero install` run.

## Risks

1. **Engineering content drift between Go fallback and on-disk pack.**
   The Phase 1 parity test catches it in CI, but a developer editing
   one without the other will see CI fail with no obvious migration
   path. Mitigation: the parity test failure message tells the dev
   exactly which file to edit and points at the planned removal of
   the Go fallback in a follow-up. The window between Phase 1
   landing and the fallback's removal is the only period this risk
   applies.

2. **PM `AGENTS.md` markdown shape doesn't quite match the engineering
   shape.** The engineering body has specific section ordering
   (Session Title, Natural Language Routing, Log significant events,
   Key Workflow, CLI Commands, Project Structure, Important Rules,
   Keep NEXT.md current, Survive context compaction, Capture
   execution plans) that downstream skills and harnesses may
   implicitly rely on. Mitigation: when authoring or auditing
   `domains/<name>/AGENTS.md`, keep the H3 section headers' ordering
   stable across packs. Documented in a follow-up convention
   `pack-agents-md-shape` (out of this spec's scope; flagged in the
   spawned-task slot).

3. **Pack-wins-over-core silently shadows a core agent the user
   expected.** A future pack ships an agent named `engineer.md`,
   shadowing the core `engineer.md`. The user sees a different agent
   than they expected. Mitigation: install output prints
   `  shadowed by pack: <filename>` for every shadowed core agent.
   Bounded to the install step so it's not noisy at runtime.

4. **`hero domain switch` is destructive to harness-materialized
   content but not to specs.** The user could expect the switch to
   also flip a PM workspace's spec frontmatter, or to refuse to
   switch when in-flight specs use the old domain's spec types. This
   spec doesn't change `runDomainSwitch`'s behavior — it remains a
   re-install of content with `.hero/` preserved. Documented as a
   v1 boundary in `domain-plugin-architecture`. The spec lifecycle
   warning is `spec-type-registry`'s concern, not this spec's.

5. **Stderr warning on missing pack `AGENTS.md` may mask real
   misconfigurations.** A user who deliberately deletes
   `domains/pm/AGENTS.md` from their build and gets the engineering
   table back may not notice the stderr line. Mitigation: also
   stamp the fallback into the managed region as an HTML comment
   (`<!-- hero: fell back to engineering routing because domain
   "pm" has no AGENTS.md -->`) so it's visible in the rendered
   file. Out of scope to escalate further; the embedded
   `AGENTS.md` for shipped packs is always present.

6. **Core-agent `domains:` frontmatter migration risk.** Adding
   `domains: [engineering]` to a `core/agents/` file that today is
   present in every workspace is a silent restriction the moment
   the new build ships. Mitigation: the audit in Phase 3 step 3
   surfaces every candidate and the migration is a deliberate,
   reviewed change. Default for any agent without `domains:` is
   "all domains" so silence stays the safe default.

7. **The model's pack-switch hint ("suggest `hero domain switch`")
   is in pack-authored text and may be inconsistent across packs.**
   This is intentional — each pack tunes the message. Risk is the
   user gets different quality of redirection in different packs.
   Mitigation: nothing in v1; documented as a `pack-agents-md-shape`
   follow-up convention.

8. **MCP-server-launched harnesses may cache the AGENTS.md body
   across `hero domain switch` calls.** Today's daemon does not
   cache AGENTS.md content (it's a file read by the harness, not
   by Hero), so this is a theoretical concern. Documented because
   any future "AGENTS.md cache in the daemon" change must invalidate
   on domain switch.

## Resolved open questions

1. **Multi-active-domain loader behavior.** Resolved: single-active
   v1. The loader resolves one pack at a time. The parent initiative's
   `domain-scoped-knowledge-graph` handles cross-domain graph reads;
   nothing in the install path needs to be multi-active.

2. **`CLAUDE.md` vs `AGENTS.md` ownership.** Resolved: both files get
   the same managed-region body (today's behavior; see
   `internal/install/claude_md.go` and `defaultSections` in
   `agents_md.go:67-72`). The pack ships **one** `AGENTS.md`; the
   installer writes that same body into both project-level files.
   Each pack does not ship a separate `CLAUDE.md` — Hero treats the
   harness fragmentation as an installer concern, not a pack-author
   concern. This stance is bit-identical to today.

3. **Cross-domain agent reuse.** Resolved: `core/agents/` is the
   universal agent tier. Agents that belong to every domain live
   there. Agents that belong only to a specific domain live in
   `domains/<name>/agents/`. Pack-wins-over-core for name
   collisions. No third "shared" tier — sharing is the universal
   core. This is also already how `core/agents/` is wired
   (`content.go:25`).

4. **Dashboard agent picker.** Resolved as **deferred / N/A for this
   spec**. The hero dashboard does not expose an agent picker today.
   When `dashboard-view-registry` (parent initiative item #4) lands,
   that work owns any agent-list UI and must enumerate from the
   active pack only. Flagged in that spec's scope; not in this
   spec's.

5. **Symlink vs generated file.** Resolved: **neither.** The pack's
   `AGENTS.md` body is **spliced** into the project's `AGENTS.md`
   inside the managed region, via the existing managed-markdown
   pipeline. No symlinks (Windows-hostile, fragile). No file
   generation that races with user edits (managed region preserves
   user content outside the markers). This is the same approach
   today's single-source-install pipeline uses for every other
   instruction file.

6. **Routing-table format — markdown vs structured.** Resolved:
   markdown. The pack's `AGENTS.md` is the source of truth. A
   structured shadow may be derived later; not in v1.

7. **Cross-domain command surface.** Resolved: the active pack's
   routing table is authoritative. Pack tables route to whichever
   slash commands they want — including engineering ones (`/design`,
   `/deliver`, `/diagnose`) if the pack chooses. The commands
   themselves are materialized from `domains/<active>/commands/`
   merged with `core/commands/`. A pack that wants `/design`
   accessible from a PM workspace must ship it in its own
   `commands/` (or rely on `core/commands/`). The PM pack today
   ships `/refine`, `/triage`, `/handoff`, `/prioritize`,
   `/prd`, `/pitch`, `/roadmap`, etc.; engineering commands are
   not reused in PM workspaces. The killer-demo case
   (PM story → engineering feature via `/design`) is handled by
   the cross-repo peer-call mechanism or by a PM-side `/design`
   command that delegates — that's a `hero-pm` concern, not this
   spec's.

## Touchpoints

- `domains/engineering/AGENTS.md` — already on disk; reconcile content
  with `generateEngineeringAgentsMdBody()` output during Phase 1.
- `domains/pm/AGENTS.md` — already on disk; audit for shape and
  routing-table markdown correctness during Phase 2.
- `domains/sales/AGENTS.md` — already on disk as a scaffold; no edits
  required.
- `domains/engineering/agents/*.md` — add `domains: [engineering]`
  frontmatter to: `feature-delivery-lead.md`, `debug-investigator.md`,
  `database-engineer.md`, `devops-engineer.md`,
  `release-engineer.md`, `dependency-analyst.md`,
  `migration-engineer.md`, `architecture-reviewer.md` (Phase 3).
- `core/agents/*.md` — audit-driven `domains:` frontmatter additions
  (Phase 3 step 3); enumerated at edit time.
- `internal/install/agents_md.go` — extract `loadPackAgentsMdBody`;
  rename `generateAgentsMdBody` → `generateEngineeringAgentsMdBody`;
  rewire `newAgentsMdBodySection.Render` to call the loader; thread
  `packTitle` into `SectionTitle()` so the H2 comes from the pack
  file's H1 line.
- `internal/install/install.go` — add `AgentsMdBodyOverride []byte`
  to `Options` (test seam; future third-party-pack seam).
- `internal/install/content.go` — `installFlat`: when `kind ==
  "agents"`, filter by frontmatter `domains:` field; implement the
  core + pack merge with pack-wins-over-core precedence.
- `internal/install/claude_md.go` — no changes (it already routes
  through `defaultSections`, which picks up the new pack body via
  `newAgentsMdBodySection`).
- `internal/install/dialect.go` — no changes.
- `internal/install/target_claude.go`,
  `internal/install/target_codex.go`,
  `internal/install/target_copilot.go`,
  `internal/install/target_cursor.go`,
  `internal/install/target_generic.go`,
  `internal/install/target_opencode.go` — no changes (they all call
  `installFlat`/`installSkillsNested` and inherit the new filtering
  + merge behavior).
- `internal/cli/install.go` — no changes (already domain-aware).
- `internal/cli/domain.go` — no changes (already re-invokes
  `install.Run` on switch).
- `content.go` — no changes (the embed already covers
  `domains/<name>/agents` for engineering, pm, sales).
- `internal/install/agents_md_test.go` — new tests:
  `TestEngineeringPackBodyMatchesGoFallback`,
  `TestLoadPackAgentsMdBody_PackPresent`,
  `TestLoadPackAgentsMdBody_PackMissingFallsBack`,
  `TestLoadPackAgentsMdBody_OverrideShortCircuits`.
- `internal/install/install_test.go` — new tests:
  `TestPMInstallExcludesEngineeringAgents`,
  `TestPackAgentShadowsCoreAgent`,
  `TestCoreAgentDomainsFrontmatterFilters`.
- `internal/cli/domain_test.go` — new test:
  `TestDomainSwitchUpdatesManagedRegion`.
- `.hero/planning/features/domain-routing-and-agents/smoke.md` —
  new manual smoke script for the model-loop routing check.
