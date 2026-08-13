---
title: "MCP Tool Category & Tier Metadata — one label set every harness defers on"
slug: mcp-tool-category-metadata
type: feature
status: completed
domain: engineering
size: medium
priority: medium
created: 2026-08-12
tags: [mcp, tools, progressive-disclosure, harness-agnostic, token-cost, contract]
relates-to: [harness-agnosticism-sweep]
completed_at: 2026-08-13T01:15:04Z
---

# MCP Tool Category & Tier Metadata — one label set every harness defers on

## Context

Hero's MCP server advertises ~59 `hero_*` tools. Every harness that connects
sends all of their schemas to the model on every turn, and tool schemas are
the single largest fixed token cost at the start of a request. hero-code hit
this at scale: its Codex provider **stalls** once the tool payload is large
(verified live — 40 tools stream, 80 stall), because desktop AgentLoop sends
the full ~71-tool catalog (30 native + 41 Hero MCP) every turn. Its fix,
`tool-deferred-schema-loading` (owned in the **hero-code** repo, app-side), is
the Claude-Code pattern: broadcast a small **eager** set of full schemas, list
the rest by name and load their schemas on demand via a `discover_tools`
meta-tool.

hero-code sent an **advisory** peer call
(`mail_e9747189f7385569888f405d`, non-blocking) asking two questions before it
finalizes: (1) should the discover contract and the tool-category surface
follow a **harness-agnostic shape Hero owns**, so Claude / Codex / hero-code
stay consistent; and (2) are there constraints on how Hero's tools should be
categorized for progressive disclosure that clients should **mirror rather than
invent**. This spec is Hero's answer to both.

**The gap, verified this session:** `internal/serve/mcp_dispatch.go` already
splits tools into read / mutate / analyze — but only as *source-file
organization and comments*, invisible in the wire contract. `ToolDefinition`
(`internal/serve/mcp_protocol.go:92`) carries `Annotations` (the closed MCP set:
readOnly / destructive / idempotent / openWorld) and an `_meta`
`map[string]interface{}` passthrough — but exactly **one** of 59 tools sets
either today. A harness that wants to defer Hero's tools has nothing to read,
so it must hardcode its own eager list and its own grouping. Every harness
inventing that independently is fleet-wide drift by construction — the same
hand-maintained-inventory rot the CLI invocation guard was just built to kill,
one layer up.

## Goal

Broadcast the labels once, in the place every MCP client already reads. Hero's
`tools/list` emits, for **every** `hero_*` tool, two pieces of metadata: a
**category** (which functional family it belongs to) and a **tier** (a hint at
whether it is hot-path enough to keep eager, or safe to defer). Each harness
then builds its own broadcast-plus-lookup on top of those labels — Claude Code
via `ToolSearch`, hero-code via `discover_tools`, Codex however it defers —
reading the *same* Hero labels instead of hardcoding a divergent guess. Hero
does not do the broadcasting or the lookup; it supplies the label set both run
on. The labels are declared once, co-located with each tool, and a guard fails
the build when a tool ships without them.

## Kickoff

Emit `category` + `tier` metadata in `_meta` for every `hero_*` MCP tool in
`tools/list`, so any harness can defer Hero tool schemas from one shared label
set instead of hardcoding its own. Answers hero-code's advisory
(`tool-deferred-schema-loading`, their repo, app-side); Hero owns the labels,
each harness owns its own deferral mechanics.

**Status:** planning — contract designed, no code. Carrier decided (`_meta`,
not `annotations`); taxonomy + tiers still to be finalized against the live
tool list during delivery.

**Pick up at:** define the category enum + tier values, co-locate a declaration
with each tool in `internal/serve/mcp_tools_def.go` / the `mcp_dispatch.go`
table, emit them under a namespaced `_meta` key, and add the every-tool-declares
guard test. Falsify the guard before trusting it.

→ `.hero/planning/features/mcp-tool-category-metadata/spec.md`

**Files:** `internal/serve/mcp_protocol.go`, `internal/serve/mcp_tools_def.go`,
`internal/serve/mcp_dispatch.go`
**Skip:** hero-code's app-side eager/deferred split and `discover_tools` tool —
their spec, their repo.

## Problem

Nothing in Hero's wire contract tells a client which tools matter or how they
group. A harness that wants to cut base token cost by deferring Hero's schemas
has three bad options: (a) send everything (the status quo that stalls Codex),
(b) hardcode an eager allowlist of Hero tool names it will have to re-sync every
time Hero adds a tool, or (c) guess groupings from tool-name prefixes. Options
(b) and (c) are per-harness inventories of Hero's surface — they drift the
moment Hero changes, and they drift *differently* in each harness, so the same
`hero_synthesize` might be eager in one client and deferred in another for no
principled reason. There is no single place a client can read "these are the
handful you almost always need; here is how the rest divide up."

## Approach

**Carrier: `_meta`, not `annotations`, not a new top-level field.** MCP's
`annotations` is a *closed, typed* set (title, readOnlyHint, destructiveHint,
idempotentHint, openWorldHint); adding a `category` there is non-conformant and
a strict client may reject the tool. A brand-new top-level `ToolDefinition`
field is equally off-spec. `_meta` is exactly MCP's blessed namespace for
server-defined metadata: it already exists on `ToolDefinition`, and a client
that does not understand a key ignores it. So the metadata is additive and
backward-compatible **by construction** — `tools/list` stays valid and every
tool stays callable for a client that reads none of it. Keys are namespaced
(e.g. `hero.dev/category`, `hero.dev/tier`) per MCP convention to avoid collision
with other servers' `_meta`.

**Two orthogonal axes; do not conflate them.**

- **Category** — the functional family, for *lookup quality*: which tools a
  client groups together and which a discovery query like "spec lifecycle" or
  "cross-repo" should surface. This is new. Seed it from the real surface, not
  invented buckets — the `mcp_dispatch.go` read/mutate/analyze split plus the
  natural families already visible in the tool list (search/knowledge,
  spec-lifecycle, attention/mail, cross-repo/peering, coverage/quality,
  handoff/session). Finalize the enum against the live 59 tools in delivery.
- **Safety class** — read / mutate / destructive — already has a home in MCP
  `annotations` (readOnlyHint / destructiveHint / idempotentHint). This spec
  keeps that axis in annotations and **backfills it** (only 1 of 59 tools sets
  annotations today), rather than duplicating it into `_meta`. Category answers
  "what family," annotations answer "is it safe to call unprompted" — separate
  questions, separate carriers.

**Tier is an advisory recommendation, never a directive.** Hero has real priors
on which tools are session-warmup essentials (`hero_context`, `hero_anchor`,
`hero_search`, `hero_status`) versus rarely-called (`hero_demo_record`,
`hero_synthesize`, `hero_test_generate`). It emits a `tier` — `eager` or
`deferrable` — as its recommendation. But which tools are hot depends on the
harness and the workflow, so the contract states plainly that a harness **may
override**: the tier is a labeled default, in the same spirit as MCP annotations
being "advisory, never authorization." This keeps Hero's opinion useful without
being prescriptive, and gives hero-code a non-arbitrary eager set to start from
instead of a hand-picked one.

**Discovery mechanics stay with the harness — Hero ships no `discover_tools`.**
Managing an eager/deferred window, deciding when to load a schema, and the
`discover_tools` / `ToolSearch` meta-tool are all functions of the harness's own
context window, which Hero does not control. Hero's contribution is *static
metadata in `tools/list`* plus a documented taxonomy. A harness filters
`tools/list` by `_meta` itself; no separate Hero discovery endpoint is needed,
and adding one would duplicate state Hero cannot keep in sync with a client's
live window. This boundary is the answer to the advisory's implicit "should Hero
own discovery" — no; Hero owns the *labels*, the harness owns the *mechanism*.

**Single source of truth + a drift guard.** The category and tier for a tool are
declared once, co-located with that tool's definition, and flow into `_meta`
from there — never a second hand-maintained list keyed by tool name. A test
asserts every registered tool declares a category in the enum and a valid tier;
a tool added without them fails the build. This is the direct carry-over of the
lesson from the CLI invocation guard delivered this session: a hand-maintained
inventory of the surface rots silently, so the surface must *carry* its own
metadata and a guard must make an omission loud. Falsify that guard per the
project bar (add a tool with no category, confirm red) before trusting it.

**Documented, versioned taxonomy.** The category enum and tier semantics are
written down as a stable contract clients can pin to — so hero-code, Claude
Code, and Codex mirror one vocabulary. A knowledge entry (or a doc under
`web/docs/`) states the categories, the tier meaning, the `_meta` key names, and
the backward-compat guarantee. hero-code's advisory reply points at it.

## Changes

1. **Model the metadata** — `internal/serve/mcp_protocol.go`.
   - Add typed representation for category (closed enum) and tier
     (`eager` | `deferrable`), and the namespaced `_meta` keys they serialize to.
   - Emit them into each `ToolDefinition._meta` at `tools/list` build time.
2. **Declare category + tier once per tool** — `internal/serve/mcp_tools_def.go`
   / `internal/serve/mcp_dispatch.go`.
   - Co-locate the declaration with each tool's existing definition/dispatch
     entry. Finalize the category enum against the live 59-tool list; assign
     tiers from Hero's warmup-vs-rare priors.
3. **Backfill MCP annotations** for the safety axis — the read/mutate/destructive
   class that `mcp_dispatch.go` already knows — so category and safety class are
   both present and orthogonal.
4. **Add the drift guard** — a test that every registered tool declares an
   in-enum category and a valid tier, and that categories stay within the
   documented set. Falsify it per-site before trusting it.
5. **Document the taxonomy** — a knowledge entry (and/or `web/docs/`) stating the
   categories, tier semantics, `_meta` key names, versioning, and the
   backward-compat guarantee; the paste-ready contract clients mirror.
6. **Reply to hero-code's advisory** — answer `mail_e9747189f7385569888f405d`
   with the contract link and the eager set Hero recommends (user-approved send;
   out of this spec's code scope but noted as the closing coordination step).

## Delivered files (code scope)

- `internal/serve/mcp_protocol.go` — `ToolCategory`/`ToolTier` closed enums (with
  per-const meaning comments), `Category`/`Tier` fields on `ToolDefinition`
  (both `json:"-"`), `toolCategorySet` + `ToolCategoryValid`/`ToolTierValid`,
  and the `MetaKeyCategory`/`MetaKeyTier` namespaced keys.
- `internal/serve/mcp_dispatch.go` — `toolSafetyClass` + `toolSafetyClasses()`,
  the read/mutate/analyze grouping lifted from the `toolHandlers()` comments into
  data, and `annotationsForSafety()`.
- `internal/serve/mcp_tools_def.go` — inline `Category:`/`Tier:` on every tool
  literal, category/tier set in `directAttentionTool`, and the single
  `finalizeToolMetadata` fold (writes `_meta` + backfills annotations).
- `internal/serve/mcp_tools_code_host.go` — category/tier on the generated
  code-host tool family.
- `internal/serve/mcp_tool_metadata_test.go` — the drift guard (AC-4/AC-5/AC-8),
  the eager-set pin, and the additive/backward-compat wire test (AC-6).
- `internal/serve/api_attention.go` — `attentionBundleManifestSHA256` bumped for
  the additive `_meta` (HTTP/MCP runtime-parity constant).
- `contracts/attention/conformance/v1/**` — regenerated bundle (`mcp-tools.json`,
  `manifest.json`, `HERO-CODE-HANDOFF.md`) via `cmd/attention-conformance`.

Changes 5 (taxonomy doc) and 6 (advisory reply) are the lead's coordination
steps, out of this engineer's code scope — the category enum is left
well-commented in `mcp_protocol.go` for the doc to derive from.

## Boundaries

Explicitly out of scope:

- **hero-code's app-side eager/deferred split and `discover_tools` tool.** That
  is `tool-deferred-schema-loading` in the hero-code repo. This spec only makes
  the labels it (and every other harness) reads.
- **No Hero-side `discover_tools` / runtime discovery endpoint.** Discovery
  mechanics belong to the harness (see Approach). Hero emits static metadata
  only.
- **No change to which tools exist, their names, schemas, or dispatch behavior.**
  Purely additive metadata; a tool call resolves exactly as before.
- **No per-harness instruction-file content.** This is a wire-contract change on
  one server surface, which is *why* it satisfies the
  `harness-changes-cover-all-targets` tripwire by construction — one contract,
  read identically by all six install targets, rather than six instruction-file
  edits. (See Risks for the honest caveat.)

## Risks

- **Category taxonomy bikeshedding / churn.** A wrong or unstable enum is worse
  than none — clients pin to it. Mitigate: seed from the *existing*
  read/mutate/analyze split and visible families rather than inventing, keep the
  set small, and version it so a later addition is additive, not a rename.
- **Tier priors are Hero's guess.** Marking the wrong tool eager wastes a client's
  budget; marking a hot tool deferrable adds a discovery round-trip. Mitigate:
  tier is explicitly advisory and overridable, and the eager set is small and
  defensible (session-warmup tools only).
- **Drift guard theater.** A guard that passes because it checks nothing is the
  failure mode this project has hit repeatedly. AC-4/AC-5 require per-site
  falsification: add a tool with no category, confirm the guard goes red naming
  it, before trusting green.
- **Tripwire honesty.** `harness-changes-cover-all-targets` is *satisfied* here
  because the change is one server-side contract every harness reads — but that
  cuts both ways: the metadata only helps a harness that actually reads `_meta`.
  Hero-code will; a harness that ignores `_meta` is no worse off than today
  (backward-compatible), but gets no benefit until it opts in. That is acceptable
  and stated, not hidden.
- **Coordination, not blocking.** hero-code is proceeding with a hardcoded eager
  set regardless. If this ships after they finalize, they adopt the labels in a
  follow-up rather than at first delivery. The cost of delay is temporary
  divergence, not a broken feature — hence `priority: medium`, important-not-urgent.
- Rollback is trivial: the metadata is additive; dropping the `_meta` emission
  returns `tools/list` to its current bytes. No schema, no data, no client break.

## Acceptance Criteria

- **AC-1:** WHEN the MCP server responds to `tools/list` THE SYSTEM SHALL include,
  in each `hero_*` tool's `_meta` under a namespaced Hero key, a `category` value
  drawn from a documented closed taxonomy.
- **AC-2:** WHEN the MCP server responds to `tools/list` THE SYSTEM SHALL include,
  in each `hero_*` tool's `_meta`, a `tier` of `eager` or `deferrable`, marking
  session-warmup tools `eager` and the remainder `deferrable`.
- **AC-3:** THE SYSTEM SHALL treat the `tier` value as an advisory recommendation
  a harness may override, and SHALL document it as such.
- **AC-4:** THE SYSTEM SHALL derive each tool's category and tier from a single
  in-code declaration co-located with the tool's definition, with no separate
  name-keyed inventory as the source of truth.
- **AC-5:** IF a registered `hero_*` tool has no category or tier declaration, OR
  declares a category outside the documented taxonomy, THEN a guard test SHALL
  fail identifying that tool — demonstrated during delivery by adding one
  undeclared tool and confirming the failure before relying on green.
- **AC-6:** IF an MCP client does not read the Hero `_meta` keys THEN `tools/list`
  SHALL remain valid and every tool SHALL remain callable unchanged (additive,
  backward-compatible).
- **AC-7:** THE SYSTEM SHALL carry the read / mutate / destructive safety class in
  MCP `annotations` (not `_meta`), keeping functional category and safety class
  as separate axes, and SHALL backfill annotations for every tool.
- **AC-8:** THE SYSTEM SHALL publish the category taxonomy, tier semantics,
  `_meta` key names, and backward-compat guarantee as a versioned document a
  harness can pin to and mirror.
- **AC-9:** THE SYSTEM SHALL NOT expose a Hero-side runtime tool-discovery
  endpoint or meta-tool; discovery mechanics remain the harness's responsibility.

## Validation

- Inspect a live `tools/list` response: every `hero_*` tool carries
  `_meta` category + tier under the namespaced key; annotations present for the
  safety axis; JSON validates.
- Backward-compat: a client (or test harness) that ignores `_meta` lists and
  calls a tool with byte-identical behavior to today.
- Drift guard: run on the full tool set (all declare category+tier, green), then
  the AC-5 falsification — add an undeclared tool, confirm the guard names it,
  revert.
- Taxonomy doc reviewed; category set matches what the code emits (a test that
  the emitted categories are exactly the documented enum — no undocumented value,
  no documented-but-unused drift).
- `go test ./internal/serve/...` and `go build ./...` green.
- Coordination: advisory reply drafted to hero-code with the contract link and
  recommended eager set (user approves the send).

## Completion Ledger

Delivered by an engineer subagent against a decided design, then lead-verified:
independently re-ran build/vet/tests, confirmed the wire output on real tool
JSON, and corrected one safety-class defect the engineer flagged (below). Stack:
Go. Validation: `go build ./...`, `go vet ./...`, `go test ./... -timeout 1200s`
(103 packages, 0 failures), markdown invocation guard green, attention
conformance suite green with the regenerated bundle.

**Safety-axis correction (high-harm axis), across two passes:** the dispatch
read/mutate/analyze grouping classifies tools by their PRIMARY action, but
several read/analyze tools also write on some inputs (an `action=` parameter,
`archive:true`, `link`). Deriving annotations from that grouping therefore
advertised `readOnlyHint:true` on state-writers — a client could auto-call one
believing it safe, the exact harm this delivery claimed to close. A first lead
pass caught two (`hero_plan` → `os.WriteFile`; `hero_error_pattern` →
`SavePattern`) and wrongly cleared `hero_active` as output-only. A cold audit
then found three still misclassed (`hero_active` register/unregister/prune;
`hero_contract` link → writes tracker_id into the spec; `hero_snapshot`
`archive:true`). All five are now `safetyMutate`, verified against their
handlers and re-confirmed on the wire (`readOnlyHint:false`). `hero_score`,
`hero_verify`, `hero_diagnose` were checked and are genuinely read-only in
their MCP form (thin report versions; `score.Score` is pure). A new regression
guard `TestConditionalWritersAreNotReadOnly` pins all five so a future
regrouping that flips one back to read fails loudly (falsified: reverting
`hero_contract` to read makes it go red naming the tool). The lasting lesson,
now written into the `toolSafetyClasses` comment: the handler-section grouping
is by primary action and is not a safety oracle; every conditional writer must
be listed explicitly.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | tools/list includes `_meta` category from closed taxonomy per tool | DONE | `finalizeToolMetadata` writes `hero.dev/category`; wire-confirmed on hero_search/hero_plan/hero_error_pattern/code_host_merge |
| 2 | tools/list includes `_meta` tier eager/deferrable | DONE | eager={context,anchor,search,status,list,queue}, rest deferrable; pinned by test |
| 3 | Tier advisory + documented as overridable | DONE | `TierEager` const comment + published doc both state advisory/overridable |
| 4 | Single co-located declaration, no name-keyed inventory as source | DONE | `Category:`/`Tier:` inline on every literal (`json:"-"`); fold reads the field |
| 5 | Guard fails naming a tool missing/out-of-enum category or tier | DONE | `TestToolMetadataDriftGuard`; falsified both ways (missing + bogus), watched RED naming the tool, reverted, re-confirmed PASS ran via -v |
| 6 | Client ignoring `_meta` still sees valid callable tool; no top-level keys | DONE | `TestToolMetadataIsAdditiveAndBackwardCompatible`; wire dump shows category/tier only under `_meta` |
| 7 | read/mutate/destructive safety in `annotations`, backfilled every tool | DONE | `toolSafetyClasses()` + `annotationsForSafety`; **5 conditional-writers reclassified to mutate** across a lead pass + cold-audit fix (plan, error_pattern, active, contract, snapshot); `TestConditionalWritersAreNotReadOnly` pins them (falsified); guard asserts every tool ends with annotations |
| 8 | Publish versioned taxonomy doc to pin/mirror | DONE | `web/docs/src/serve/mcp-tool-metadata.md` — v1 contract: keys, closed category table, tier semantics, safety-axis separation, backward-compat guarantee, "Hero ships no discover_tools" |
| 9 | No Hero-side runtime discovery endpoint/meta-tool | DONE | Metadata-only; no `discover_tools`; dispatch/handlers unchanged |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Model metadata + emit into `_meta` at build time — `mcp_protocol.go` | DONE | `ToolCategory`/`ToolTier` enums+validators, `json:"-"` fields, `MetaKey*`; fold in `mcp_tools_def.go` |
| 2 | Declare category+tier once per tool | DONE | 58 inline literals + attention + code-host generator |
| 3 | Backfill annotations for safety axis from dispatch grouping | DONE | lifted `toolHandlers()` sections into `toolSafetyClasses()`; 2 writer misclassifications corrected at source by lead |
| 4 | Add drift guard, falsify per-site | DONE | `mcp_tool_metadata_test.go`; both falsifications watched RED, reverted |
| 5 | Document the taxonomy (web doc) | DONE | `web/docs/src/serve/mcp-tool-metadata.md`, written by lead |
| 6 | Reply to hero-code advisory with contract + eager set | DONE | Sent this session (mail `mail_66116582d81c7bec2a964c36`, on-thread), user-approved. Note: reply named `io.hero/` before the namespace was corrected to `hero.dev/` during delivery — a one-line follow-up correction is owed (surfaced to user) |

### Exercise-the-feature check

- [x] User-visible behavior (the `tools/list` wire output) exercised end-to-end via marshaled `ToolDefinition` JSON: `hero_search` (`_meta.hero.dev/category=search-and-knowledge`, `tier=eager`, `annotations.readOnlyHint=true`); `hero_plan` and `hero_error_pattern` (post-fix `readOnlyHint=false`, category/tier present); `hero_code_host_merge` (existing `destructiveHint/openWorldHint` + effect/consent `_meta` preserved, category/tier added alongside). Confirmed no top-level `category`/`tier` keys. Full `tools/list` round-trip and regenerated conformance bundle both green.

### Excellence Bar self-check

Yes. The metadata is co-located and drift-guarded (falsified both ways), the safety axis is derived from existing curation rather than re-guessed, and the one place that verbatim lift was wrong — two disk-writers advertised as read-only — was caught in lead verification and fixed at the source, not patched at the output. The change is provably additive on the wire, the taxonomy is published as a pinnable v1 contract, and the unavoidable conformance-bundle downstream was regenerated and re-verified. The only residual is a one-line courtesy correction to hero-code about the `_meta` key namespace, surfaced to the user rather than sent silently.
