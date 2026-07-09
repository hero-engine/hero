---
title: "Cold-Start Trust Hardening — Fail Loud, Never Mislead, at First Use"
slug: cold-start-trust-hardening
type: initiative
status: planning
domain: engineering
size: large
priority: high
created: 2026-06-23
tags: [graph, ingest, discoverability, scan, gitignore, dx, cold-start, relations, tier-2]
child:
  - cst-relation-frontmatter-fail-loud
  - cst-wikilink-edge-intent-warning
  - cst-relations-block-discoverability
  - cst-tier-labeling-clarity
  - cst-gitignore-events-log
  - cst-check-human-severity
  - cst-verify-lifecycle-scoping
  - cst-queue-blocked-source-parity
  - cst-initiative-premature-autocomplete
  - cst-archive-orphans-sibling-files
relations:
  - target: hihcp-mcp-first-turn-readiness
    kind: related
  - target: hihcp-mcp-auto-reconnect
    kind: related
  - target: hero-workspace-not-self-describing
    kind: related
  - target: spec-lifecycle-hygiene-breakdown
    kind: related
completed_at: 2026-06-23T19:57:04Z
---

## Vision

A capable agent dropped into a fresh Hero workspace should never have to reverse-engineer Hero's SQLite schema to find out why something didn't work. When Hero can't do what the user expects, it should say so **at the moment of failure, in the command they ran, naming the real cause** — not fail silently, not emit a uniform wall of benign warnings, and never surface an unrelated signal that reads as the explanation.

## Goal

Eliminate the class of first-use failures where Hero degrades silently or misleadingly, forcing the user to guess. Concretely: every relationship a user declares either becomes an edge or produces a precise error; the deterministic (Tier-1) graph is never confused with optional LLM enrichment (Tier-2); and routine Hero commands stop dirtying the tree or crying wolf.

## Problem

This initiative is grounded in a real external-user session (the `candy` project, a first-time Hero adopter). The agent spent roughly an hour and a long chain of trial-and-error trying to get spec relationships to materialize as graph edges, reached a **confident but wrong conclusion**, and shipped a fragile workaround. Root-cause analysis against this repo's code shows the friction was caused by Hero's affordances, not the agent's reasoning:

1. **Silent wrong-schema failure (root cause).** The agent declared relationships three plausible ways — top-level `initiative:` / `depends_on:` frontmatter fields and body `[[wikilinks]]`. Hero recognizes **none** of them. The deterministic Tier-1 parser only reads a `relations:` block (`internal/spec/graph_ingest.go:141`); unknown top-level keys fall through silently (`internal/spec/spec.go:547`), unresolved targets are silently skipped (`graph_ingest.go:159` — "target not in this batch — skipped"), and wikilinks are never parsed as edges at all. Three ways to ask, one works, the other two no-op without a word.

   **Audit correction (2026-06-23):** the agent further concluded "this build's `hero graph reingest work` doesn't derive edges from frontmatter." That is **false** — `reingest` calls `spec.WriteGraph`, which *does* derive edges from the `relations:` block (`internal/cli/graph_memory.go` → `graph_ingest.go:141`). Reingest was never broken; the edges simply never existed because the schema was wrong. This makes the fix below higher-value, not lower.

2. **A misleading bystander signal.** `hero scan` prominently prints `⊘ tier-2: no ANTHROPIC_API_KEY … Tier-2 extraction disabled` (`internal/cli/scan.go:446`, message at `internal/extract/auto.go:34`). The agent latched onto this as the cause of its missing edges. But **Tier-2 has nothing to do with structural edges** — it is LLM enrichment that extracts Decision/Concept nodes from prose (`internal/extract/client.go`). The deterministic edge graph (Tier-1) needs no key. The output let an unrelated "disabled" line read as "the graph is off."

3. **A volatile cache is tracked.** *(Scope corrected by audit.)* `.hero/cache/` is already ignored (`.gitignore:20`), but **`.hero/events.log` is not** — it's append-only, written on every `claim`/`complete`/`hook`/`handoff` (`internal/tracking/tracking.go:24`), so it dirties the working tree on routine commands. (The candy session's `git checkout` abort was attributed to this class of churn; the specific committed file was `events.log`, not `cache/`.)

4. **`hero check` cries wolf — in the human view only.** *(Scope corrected by audit.)* Severity tiers (pass/warn/fail) **exist in the JSON output** (`internal/cli/check.go:60`) but the **human CLI output is a flat list** (`check.go:413`) with no grouping. Missing-`## Kickoff` is one of 13+ checks (all `warn`); on a scaffold-heavy workspace it dominates the flat list, so a healthy workspace reads as ~100 failures. The fix is bringing the existing tiering/grouping to the human view, not inventing severities.

5. **Delivery gates fire on planning drafts (confirmed bug).** `hero spec verify` runs the full delivery gates (Completion Ledger, delivery-audit.md) **with no status guard** (`internal/cli/verify.go:68-137`), so a `status: planning` spec FAILs with "no Completion Ledger section found." The candy agent then hand-wrote a ledger and spawned a cold audit *on a planning draft* — wasted motion the tool should have stopped with one lifecycle-aware message. Reproduced live in this session against this very initiative spec. (Install is *not* the cause — the `delivery-audit` skill ships fine; the candy agent's "no audit agent" was looking for the wrong artifact type, since it's a skill invoked as a cold subagent, not a standalone agent.)

6. **`hero queue` and `hero blocked` disagree at cold start (confirmed divergence).** `queue` reads frontmatter `relations:` (`internal/cli/queue.go:176`); `blocked` queries graph.db edges (`internal/cli/brief.go:573`). Since graph.db is gitignored and must be reingested, on a fresh clone `blocked` shows nothing while `queue` is already correct — two sources of truth for "is this blocked."

The through-line: **silent, uniform-noisy, or lifecycle-blind failure forces reverse-engineering.** A capable model recovers from a clear error; it cannot recover from a silent no-op or a misleading gate failure except by inspecting internals — which is exactly what burned the session.

## Guiding principles

- **Fail loud at the point of failure.** The command the user ran should name the real cause, not leave them to infer it.
- **Never let an unrelated signal read as the cause.** Tier-1 (deterministic, free) and Tier-2 (LLM, keyed) must be unmistakably distinct in all output.
- **Recognize intent or reject it — never silently drop it.** A relation the user clearly meant should either work or error; it must not vanish.
- **Surgical, not a rewrite.** These are affordance fixes (messages, parsing tolerance, gitignore, severity tiers), not a graph re-architecture.

## Grounding facts

All file:line refs below were re-verified against source by a four-agent code audit on 2026-06-23.

- Tier-1 edge derivation: `internal/spec/graph_ingest.go:141-173`; relation→edge kind mapping at `:186-237` (`parent→belongs_to`, `depends-on→depends_on`, `blocks`, `supersedes`, `related→related_to`). Node keys are **type-capitalized** (`Feature:slug`, `Initiative:slug`) — `graphTypeFor()` at `:186-212`.
- `hero graph reingest work` **does** honor the `relations:` block (calls `spec.WriteGraph`): `internal/cli/graph_memory.go`. The candy "reingest is broken" claim was refuted by audit.
- Canonical relation shape: a `relations:` block of `{ target: <slug>, kind: <kind> }` (`internal/spec/spec.go:602-647`). Unknown top-level keys fall through silently: `spec.go:547`.
- Silent skip of unresolved targets: `graph_ingest.go:159`.
- Tier-2 is provider-agnostic (anthropic/openai/azure) but key-only, no subscription/OAuth: `internal/runner/provider.go:66-122`, `internal/extract/auto.go:30-35`. **Auth/provider work is explicitly deferred — see Out of scope.**
- Disabled-tier message + render: `internal/extract/auto.go:34`, `internal/cli/scan.go:446-462`, emitted from `scan.go:640`.
- Wikilinks are parsed as searchable text only; no wikilink→edge path exists.
- `.hero/cache/` already ignored (`.gitignore:20`); `.hero/events.log` **not** ignored, appended by `internal/tracking/tracking.go:24`. Managed gitignore block: `internal/cli/init.go:432-454`.
- `hero check` severity tiers exist in JSON (`internal/cli/check.go:60`); human output is flat (`check.go:413`); kickoff check at `check.go:299-318` (severity `warn`).
- Delivery gate runs with no status guard: `internal/cli/verify.go:68-137`; gate defs at `:26-43`; audit-report lookup `checkAudit()` at `:267-296`. `delivery-audit` ships as a skill: `domains/engineering/skills/delivery-audit/SKILL.md`.
- queue vs blocked source divergence: `internal/cli/queue.go:176-187` (frontmatter relations) vs `internal/cli/brief.go:573-593` (graph.db edges).

## Specs

Child stubs below are each actionable as a `/design` input. Slugs are listed in `child:` frontmatter.

### cst-relation-frontmatter-fail-loud  *(root cause — highest value)*
Make relationship declaration unmissable. When frontmatter contains relation-shaped keys Hero doesn't consume (`initiative:`, `depends_on:`, `parent:` used as a bare slug, etc.) or a `relations:` target that resolves to nothing, **emit a precise warning** naming the unrecognized field/target and pointing at the canonical `relations:` block. Decide per-key whether to *alias* common shorthands (`depends_on:` → `depends-on` relation) or strictly reject — but never silently drop. Surface unresolved relation targets in `hero check`/`hero scan` output instead of skipping at `graph_ingest.go:160`.

### cst-wikilink-edge-intent-warning
Detect `[[wikilinks]]` in spec bodies that look like intended relations and warn that wikilinks do not create graph edges, suggesting the `relations:` block. (Stretch: offer to materialize them.) Prevents the exact "I linked them, why no edges?" trap.

### cst-relations-block-discoverability
Make the `relations:` block discoverable without reading source: include a commented `relations:` example in spec scaffolds/templates, document it in AGENTS.md and the spec-type docs, and reference it from the fail-loud warning copy above.

### cst-tier-labeling-clarity
Relabel all Tier-2 output so it can never read as "relationships off." Rename the scan summary line to something like `⊘ enrichment (optional LLM, Decisions/Concepts): skipped — no provider key`, and add a one-line note that the structural graph (Tier-1) is unaffected. Audit every place "extraction"/"tier-2" is surfaced.

### cst-gitignore-events-log
Add `.hero/events.log` to the managed gitignore block (`internal/cli/init.go:432`) and stop tracking it, so routine Hero commands no longer dirty the tree or block branch switches. One-time `git rm --cached` for already-tracked instances. (`.hero/cache/` is already ignored — no change needed there.)

### cst-check-human-severity
Bring `hero check`'s existing JSON severity tiering (pass/warn/fail) to the **human CLI output**, which is currently a flat list (`check.go:413`). Group/collapse the dominant benign category (scaffolds missing `## Kickoff`) so a healthy scaffold-heavy workspace doesn't read as ~100 failures. No new severities needed — surface the ones that already exist.

### cst-verify-lifecycle-scoping  *(confirmed P0 bug)*
`hero spec verify` runs full delivery gates (Completion Ledger, delivery-audit.md) on specs of **any** status, including `planning`, with no guard (`verify.go:68-137`). Add a status guard so delivery gates only run for delivering/in-review/completed specs; for a planning draft, emit a lifecycle-aware message ("spec is in planning — delivery gates don't apply yet; run /deliver to start implementation") instead of "no Completion Ledger section found." Reproduced live in this session. Ready for `/diagnose`.

### cst-initiative-premature-autocomplete  *(found during delivery, reproduced live)*
Completing the only *materialized* child of an initiative auto-completed and archived the whole initiative, even though 7 of its 8 children are unmaterialized stubs. `autoCompleteParentIfReady` (`internal/cli/verify.go`) treats "all materialized children done" as "initiative done" — and the block-style `child:` list (`spec.go:507` shorthand only parses inline/list-on-same-line via `parseList(val)`, not a newline `- item` block) doesn't register the pending stubs. Fix: an initiative with declared-but-unmaterialized children must not auto-complete; count declared children (parse the block-style `child:` list) and/or require all declared children to exist and be completed. Coordinate with `TestVerify_UnmaterializedInitiativeChild`.

### cst-archive-orphans-sibling-files
`completeAndArchive` moved `spec.md` to `specs/<slug>/` but left its sibling `delivery-audit.md` behind in `planning/bugs/<slug>/`, orphaning the audit report from its spec. Archive should move the whole spec directory (or all known sibling artifacts), not just `spec.md`.

### cst-queue-blocked-source-parity
`hero queue` reads frontmatter `relations:` while `hero blocked` queries graph.db edges, so they disagree on a fresh clone (graph.db is gitignored until reingest). Make both read the same source of truth — or have `blocked` fall back to frontmatter / auto-reingest — so "is this blocked?" is answered consistently at cold start. Coordinate with the durable-source principle (frontmatter is canonical; graph.db is a cache).

## Dependencies

- `cst-relations-block-discoverability` should land after `cst-relation-frontmatter-fail-loud` settles the canonical shape and warning copy.
- `cst-wikilink-edge-intent-warning` builds on the same warning surface as the fail-loud work.
- The rest are independent and parallelizable.

## Related existing bugs (coordinated, not owned)

These already belong to `hero-in-hero-code-parity`; this initiative coordinates with them because they share the cold-start-trust theme. Linked as `related`, not re-parented:

- **hihcp-mcp-first-turn-readiness** — "Gate First Turn on Hero MCP Readiness." Root of *why* the candy agent never used MCP at all and fell back to CLI/sqlite archaeology.
- **hihcp-mcp-auto-reconnect** — "Auto-Recover from MCP Server Disconnect Mid-Session." Same MCP-availability theme.
- **hero-workspace-not-self-describing** — "AGENTS.md Project Structure section lies about content-path locations." Why the candy agent invented a `docs/` folder instead of using `.hero/`.
- **spec-lifecycle-hygiene-breakdown** — the delivery-gate orchestration gap (gate demands Completion Ledger + delivery-audit.md, but only the agent-mediated `/deliver` path produces them; manual `hero verify` users hit dead ends with no guidance). `cst-verify-lifecycle-scoping` fixes the *status-guard* half here; the deeper orchestration/guidance half belongs to that existing bug.

## Cross-cutting concerns & shared risks

- **Warning fatigue.** The fail-loud work (relations, wikilinks, unresolved targets) must not recreate the `hero check` wolf-crying problem. Coordinate severity/grouping with `cst-check-scaffold-noise`.
- **Alias-vs-reject is a real fork.** Aliasing `depends_on:`/`initiative:` is friendlier but grows the accepted-schema surface; strict rejection is cleaner but harsher. Resolve in `cst-relation-frontmatter-fail-loud`'s `/design`.
- **gitignore migration** touches already-committed files in real workspaces — needs an upgrade-safe `git rm --cached` path, not just a gitignore edit.

## Out of scope / deferred

- **LLM auth & provider expansion.** Subscription/OAuth support (so a Claude Pro/Max sub powers Tier-2 without a raw key) and broadening beyond anthropic/openai/azure (Gemini, Bedrock, Vertex, local). Deliberately deferred to a separate future initiative — decided 2026-06-23. The provider abstraction already exists (`provider.go`); the gap is subscription auth and under-advertised provider support, which is its own bet.

## Recommended delivery order

1. **cst-gitignore-events-log** — trivial, unblocks clean commits immediately.
2. **cst-verify-lifecycle-scoping** — confirmed P0, surgical status guard, high trust impact, reproduced live.
3. **cst-relation-frontmatter-fail-loud** — root cause, highest user-trust ROI.
4. **cst-tier-labeling-clarity** — cheap, kills the misleading-bystander trap.
5. **cst-relations-block-discoverability** — once the canonical shape/copy is settled.
6. **cst-queue-blocked-source-parity** — correctness; coordinate with the durable-source principle.
7. **cst-wikilink-edge-intent-warning** — builds on the fail-loud surface.
8. **cst-check-human-severity** — independent; sequence by capacity.

Related MCP/self-description bugs proceed in parallel under `hero-in-hero-code-parity`.

## Progress

- 2026-06-23 — Initiative drafted from candy first-use session root-cause analysis. Auth/provider scope deferred. Six child stubs defined; three existing bugs linked as related. Awaiting `/design` passes on child specs.
- 2026-06-23 — Delivered `cst-verify-lifecycle-scoping` (the P0). Delivery surfaced two more live bugs, now added as stubs #9–10: completing the bug's spec auto-completed *this whole initiative* (only 1 of 8 children materialized) and archived it — restored manually; and the archive orphaned the `delivery-audit.md` sibling. Both captured as `cst-initiative-premature-autocomplete` and `cst-archive-orphans-sibling-files`.
- 2026-06-23 — **Dogfood confirmation of finding #1.** While authoring this spec and the `cst-verify-lifecycle-scoping` bug, I declared relations in inline flow style (`- { kind: related, target: X }`). It silently produced **zero edges** — `applyRelField` (`spec.go:736`) splits on the first colon, so `{ kind` never matches `target`/`kind` and the entry is never flushed. Only two syntaxes work: top-level shorthand (`parent: <slug>`) and the block-style `relations:` list (`- target:` / `kind:` on separate lines, with `kind: related` — *not* `relates-to`, which maps to no edge). This is the exact silent-no-op trap that burned the candy session, reproduced by someone who had just read the parser. Both specs corrected to proven syntax.
- 2026-06-23 — Four-agent code audit reconciled every claim against source. Corrections folded in: (a) `reingest` is NOT broken — it honors `relations:`; candy's edges failed purely from wrong schema; (b) `.hero/cache/` is already ignored — the real gitignore gap is `.hero/events.log`; (c) `hero check` already has JSON severity tiers — the gap is the flat human view; (d) my earlier NEXT.md/SNAPSHOT.md "churn" observation was misattributed (scan/index don't touch them). Two new confirmed findings added as child stubs: `cst-verify-lifecycle-scoping` (P0 — delivery gates fire on planning drafts) and `cst-queue-blocked-source-parity` (queue/blocked read different sources, disagree at cold start). `spec-lifecycle-hygiene-breakdown` linked as related. Now eight child stubs.
- 2026-06-23 — **Initiative delivered (9 of 10 child stubs shipped; full suite green).** Commits on `main`:
  - ✅ `cst-verify-lifecycle-scoping` — v0.19.1 (lifecycle guard on `hero spec verify`).
  - ✅ `cst-relation-frontmatter-fail-loud` — recognize `initiative:`/`depends_on:` shorthands + block-style `child:` lists (`6ffff42`).
  - ✅ `cst-initiative-premature-autocomplete` — declared-roster gate before auto-completing an initiative (`9395c1e`); completed + archived (dogfooded: this initiative correctly stayed open).
  - ✅ `cst-archive-orphans-sibling-files` — archive moves the whole spec dir, not just `spec.md` (`b5b91d0`).
  - ✅ `cst-tier-labeling-clarity` — scan output labels Tier-2 as optional enrichment (`ad63073`).
  - ✅ `cst-check-human-severity` — severity-aware human summary + collapsed kickoff noise (`c446101`).
  - ✅ `cst-wikilink-edge-intent-warning` — `hero check` advisory for `[[wikilinks]]` (`074f30c`).
  - ✅ `cst-queue-blocked-source-parity` — `hero blocked` reconciles from frontmatter (`0cd04b0`).
  - ✅ `cst-relations-block-discoverability` — relation-syntax docs in AGENTS.md + `relates-to` → edge (`a6a46dd`).
  - ⊘ `cst-gitignore-events-log` — **resolved as corrected, not implemented.** On investigation `events.log` is the durable backing store for `hero feed`/`hero velocity`/clusters (consumed at `feed.go`/`velocity.go`/`clusters.go`, preserved by `satellite_migrate`); untracking it would lose cross-session/cross-machine activity & velocity history. The churn only comes from state-changing commands, normal for a tracked ledger. Left tracked by design. (`.hero/cache/` is already ignored.)
  - **Open product question deferred to the user:** whether `events.log` should remain durable-shared (tracked) or become per-machine (gitignored) — a real call with data-loss implications, not assumed here.
