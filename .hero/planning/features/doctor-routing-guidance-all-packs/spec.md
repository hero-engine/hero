---
title: "Extract hero doctor / MCP-surface routing guidance into a shared domain-agnostic section"
slug: doctor-routing-guidance-all-packs
type: enhancement
status: delivering
priority: low
domain: engineering
created: 2026-07-13
updated: 2026-07-13
origin: session
tags: [harness, install, domain-packs, base-vs-overlay, section-contributor, doctor, mcp-routing, agents-md, refactor]
relates-to:
  - agent-hero-version-schema-confusion
---

# Extract hero doctor / MCP-surface routing guidance into a shared domain-agnostic section

## Summary

The `agent-hero-version-schema-confusion` delivery (v0.25.0) added routing guidance — **prefer Hero's MCP tools over shelling a bare `hero`; on any schema/version mismatch run `hero doctor` instead of confabulating a migration story or running `hero upgrade`** — but placed it **inside `generateEngineeringAgentsMdBody`** (`internal/install/agents_md.go`) and the `domains/engineering/AGENTS.md` mirror. That guidance is **domain-agnostic** (it's about the hero binary / MCP surface, nothing engineering-specific), yet it currently ships only to the engineering pack. This spec corrects the layering: **author it once in a shared section that renders into every domain's managed region**, and remove it from the engineering body.

### Why not "copy into pm/sales/chat" (the original approach — rejected)
Investigation of the install pipeline showed the AGENTS.md body resolves via `OverlayFS(domainFS, coreFS)` as a **whole-file override**, not a section merge — and **all four domains (`engineering`, `pm`, `sales`, `chat`) ship their own `domains/<d>/AGENTS.md`**, so a domain body fully shadows any base. There is **no `core/AGENTS.md`**. Copying the paragraph into each pack body would mean four duplicated copies that drift, and every *future* pack would silently miss it. That's the wrong layer.

### The right layer: a shared SectionContributor
The managed region is assembled by `defaultSections()` (`agents_md.go:212`) from a list of `managed.SectionContributor`s — currently the pack body section **plus** `snapshot.NewPointerSection(...)`, which is already a **domain-agnostic section authored once and rendered into every pack's managed region, across all six targets** (`defaultSections` is shared by the AGENTS.md and CLAUDE.md callers). That snapshot pointer is the precedent to follow. Adding the doctor/MCP guidance as its own contributor makes it: authored once, present in every domain (current and future), on every target — with zero per-pack maintenance.

## Motivation
The version/schema confusion is domain- and harness-agnostic: any agent in any domain that shells a stale `hero` and reads a newer graph can confabulate. A PM/Sales/CS/Chat user should get the same "run `hero doctor`, don't invent a migration story" floor as an engineer — the mission is to raise the floor for *everyone*. Doing it as a shared section (rather than N pack copies) also guarantees the next domain pack inherits it for free.

## Scope
- **In scope:** (1) a new domain-agnostic `SectionContributor` that emits the doctor/MCP-surface routing guidance; (2) wire it into `defaultSections()` so it renders for all domains + all six targets; (3) **remove** the paragraph from `generateEngineeringAgentsMdBody` and `domains/engineering/AGENTS.md` (and regenerate the mirror) so there is a single source.
- **Out of scope:** any other guidance content; restructuring AGENTS.md resolution from override to merge (bigger change, not needed here); introducing a `core/AGENTS.md`.

## Suggested Approach
1. **Create the section.** Model it on `internal/snapshot/pointers.go` `NewPointerSection` — a small type implementing `managed.SectionContributor` (`SectionID`, `SectionTitle`, `Render`). Give it a stable `SectionID` (e.g. `install:hero-operational-guidance`) so the managed-region differ tracks it. `Render` returns the routing paragraph (reuse the exact v0.25.0 wording verbatim from the current `agents_md.go:675` block).
2. **Decide placement + heading.** The current paragraph lives under the engineering body's "Internal Lookups — Tool Routing" H3. As a standalone section it needs its own heading (e.g. `### Hero Binary & MCP Surface` or fold under a domain-agnostic "Operating Hero" heading). Pick a heading that reads correctly in *every* pack, and choose its order in `defaultSections` (recommend: after the pack body, before the snapshot pointer). Confirm against the managed-region skeleton convention (`.hero/knowledge/conventions/domain-agents-md-skeleton.md`).
3. **Remove the duplicated source.** Delete the paragraph from `generateEngineeringAgentsMdBody` and from `domains/engineering/AGENTS.md`; regenerate the engineering mirror (`HERO_REGEN_PACK_AGENTS=1`) so `TestEngineeringPackBodyMatchesGoFallback` stays green with the paragraph gone from the body.
4. **Repoint the coverage test.** Update `TestHarnessNative_DoctorRoutingGuidanceAllTargets` to assert the guidance now comes from the shared section and appears for **every domain × every target** (not just engineering), and would fail if the section were dropped.

## Acceptance Criteria
- [ ] The doctor/MCP routing guidance is authored in exactly **one** place (a shared section contributor) — `grep` for the guidance text finds a single source, not four.
- [ ] It renders into the managed region for **all four domains** (engineering, pm, sales, chat) and **all six targets** (opencode, cursor, claude, copilot, codex, generic).
- [ ] It is **removed** from `generateEngineeringAgentsMdBody` and `domains/engineering/AGENTS.md`; the engineering body no longer carries a private copy.
- [ ] A future/unknown domain (no own AGENTS.md, or a new pack) inherits the guidance automatically — covered by a test that renders a non-engineering domain and asserts the section is present.
- [ ] Managed-region byte-stability tests (`TestHarnessNative_SameManagedBody`, the pack-body fallback-match test) stay green.

## Test Plan
- Unit-test the new section's `Render` output.
- Parametrize `TestHarnessNative_DoctorRoutingGuidanceAllTargets` over {engineering, pm, sales, chat} × six targets; assert the guidance is present and sourced from the shared section.
- Assert the engineering body **no longer** contains the paragraph (guards against a lingering duplicate).
- Add a case for a domain with no own AGENTS.md (fallback path) to prove the section still renders.

## Notes
- This is a small refactor + coverage win, not new behavior. It corrects the placement from `agent-hero-version-schema-confusion` (which shipped it engineering-only under time pressure) rather than piling a second copy on top.
- Governed by tripwire `harness-changes-cover-all-targets` — trivially satisfied here because a single `defaultSections` contributor is inherently all-target and all-domain.
- Watch the managed-region diff/drift machinery: adding/removing a section changes the managed block; make sure `hero upgrade` on an existing install cleanly reconciles (the section add + the engineering-body removal net out to the same rendered text, so a re-install/upgrade should be a no-op in content for engineering and an addition for the other packs).

## Completion Ledger

Delivered on branch `fix/agent-hero-version-schema-confusion`. `go build ./cmd/hero/` OK; `go test ./internal/install/... ./internal/snapshot/...` green; `go vet` clean on touched packages.

| Acceptance criterion | Status | Evidence |
|---|---|---|
| Guidance authored in exactly one place | DONE | `internal/install/operational_guidance.go` const `heroOperationalGuidance`; grep across `*.go` finds only the section def + tests; 0 occurrences in `agents_md.go`. |
| Renders for all 4 domains × 6 targets | DONE | `TestHarnessNative_DoctorRoutingGuidanceAllTargets` — 24 subtests PASS; wired via `defaultSections` (`agents_md.go:216`), shared by AGENTS.md + CLAUDE.md callers. |
| Removed from engineering body + mirror | DONE | Block deleted from `generateEngineeringAgentsMdBody`; `domains/engineering/AGENTS.md` regenerated (`HERO_REGEN_PACK_AGENTS=1`); `TestEngineeringBodyOmitsOperationalGuidance` PASS. |
| Future/unknown domain inherits automatically | DONE | `TestHarnessNative_OperationalGuidanceFallbackDomain` (`--domain widgets`, no own AGENTS.md → engineering fallback body) PASS. |
| Byte-stability tests green | DONE | `TestHarnessNative_SameManagedBody`, `TestEngineeringPackBodyMatchesGoFallback` PASS. |

**Changes:** new `internal/install/operational_guidance.go` (+ `_test.go`); `internal/install/agents_md.go` (`defaultSections` wiring, engineering-body paragraph removed); `internal/install/harness_native_test.go` (repointed matrix + 2 new tests); `domains/engineering/AGENTS.md` (regenerated mirror).

**Exercised:** real `hero install` — engineering CLAUDE.md renders the guidance **exactly once** (marker count 1, heading `## Hero Binary & MCP Surface` count 1, 0 before the section); `--domain pm` CLAUDE.md **now carries** it (count 1) where it previously did not.

**Note:** `chat` is deliberately non-installable (no `DomainFS` case, `content.go:9-21`); its coverage runs through the on-disk `domains/chat/AGENTS.md` via the `AgentsMdBodyOverride` seam. The shared-section wiring covers it automatically if/when it becomes installable.

## Kickoff

v0.25.0 put the universal "prefer the MCP surface; run `hero doctor` on version/schema confusion" guidance inside the **engineering** pack body. It's domain-agnostic and should be a shared section, not a per-pack copy — there's no `core/AGENTS.md` and all four domains ship their own body, so duplication would be the wrong call.

**Status:** planning — corrected design (was "copy into pm/sales/chat"; now "extract to a shared section contributor"). Deferred follow-up from `agent-hero-version-schema-confusion`.

**Pick up at:** model a new domain-agnostic `managed.SectionContributor` on `internal/snapshot/pointers.go` `NewPointerSection`; emit the routing paragraph (verbatim from `internal/install/agents_md.go:675`); wire it into `defaultSections()` (`agents_md.go:212`); then **remove** the paragraph from `generateEngineeringAgentsMdBody` + `domains/engineering/AGENTS.md` and regenerate the mirror (`HERO_REGEN_PACK_AGENTS=1`). Repoint `TestHarnessNative_DoctorRoutingGuidanceAllTargets` to assert all-domain × all-target coverage from the shared section.

**Files:** `internal/install/agents_md.go` (`defaultSections`, `generateEngineeringAgentsMdBody`), new section file (near `internal/snapshot/pointers.go` for the pattern), `domains/engineering/AGENTS.md`, `internal/install/harness_native_test.go`.
**Skip:** copying the paragraph into pm/sales/chat bodies (the rejected approach); introducing a `core/AGENTS.md`; converting AGENTS.md resolution from whole-file override to section merge.
