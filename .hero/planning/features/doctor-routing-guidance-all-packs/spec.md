---
title: "Extend hero doctor / MCP-surface routing guidance to all domain packs (pm, sales, chat)"
slug: doctor-routing-guidance-all-packs
type: enhancement
status: planning
priority: low
domain: engineering
created: 2026-07-13
origin: session
tags: [harness, install, domain-packs, doctor, mcp-routing, agents-md]
relates-to:
  - agent-hero-version-schema-confusion
---

# Extend hero doctor / MCP-surface routing guidance to all domain packs

## Summary

The `agent-hero-version-schema-confusion` delivery (v0.25.0) added routing guidance that steers agents to **prefer Hero's MCP tools over shelling a bare `hero`**, and to **run `hero doctor` on any schema/version mismatch instead of confabulating a migration story or running `hero upgrade`**. That guidance was authored once in `generateEngineeringAgentsMdBody` (`internal/install/agents_md.go`) and mirrored to `domains/engineering/AGENTS.md` — so it propagates to all six install **targets** (opencode|cursor|claude|copilot|codex|generic) for the **engineering** domain.

But domain **packs** are an orthogonal axis. The `pm`, `sales`, and `chat` packs have their **own** AGENTS.md bodies (independent generators / pack files) and did **not** receive this paragraph. A user working in a non-engineering domain (a PM or CS pack) gets no guidance to prefer the MCP surface or to run `hero doctor` — so an agent there can hit exactly the same version-confusion trap this fix was meant to close.

## Motivation
The version/schema confusion is harness- and domain-agnostic: any agent that shells a stale `hero` and reads a newer graph can confabulate. Non-engineering domains are explicitly a place where we want **more** proactive, integrated assistance, not less. Leaving the guidance in engineering-only means the floor is raised for engineers but not for PM/Sales/CS/Chat users — the opposite of the mission's "raise the floor for everyone."

## Scope
- **In scope:** add the same routing guidance to the `pm`, `sales`, and `chat` domain pack bodies, propagated to all six targets within each pack. Keep it identical in intent; adapt tone only if a pack's voice differs.
- **Out of scope:** any change to the engineering pack (already done); any new guidance content beyond the doctor/MCP-surface routing.

## Suggested Approach
1. Locate each pack's AGENTS.md body source. For engineering it's `generateEngineeringAgentsMdBody` + `domains/engineering/AGENTS.md`. Find the analogues for `pm`, `sales`, `chat` (grep `internal/install/` for `generate*AgentsMdBody` / per-pack body generators, and the `domains/<pack>/AGENTS.md` mirrors + their regen mechanism — engineering uses `HERO_REGEN_PACK_AGENTS=1`).
2. Add the same routing paragraph to each pack's author-once source, mirrored to each `domains/<pack>/AGENTS.md`. Reuse the exact wording from the engineering pack for consistency (consider extracting the paragraph to a shared constant if the pack generators can share one — evaluate; don't force it if the bodies are independent by design).
3. Keep it self-contained and imperative (hookless-harness rule).

## Acceptance Criteria
- [ ] The doctor/MCP-surface routing guidance appears in the rendered instruction surface for `pm`, `sales`, and `chat` packs, across all six targets.
- [ ] A per-pack propagation test (mirroring `TestHarnessNative_DoctorRoutingGuidanceAllTargets`) fails if any pack drops the guidance.
- [ ] The engineering pack is unchanged; pack-body sync tests (e.g. the `HERO_REGEN_PACK_AGENTS` fallback-match test) stay green for every pack.

## Test Plan
- Extend / parametrize the existing `TestHarnessNative_DoctorRoutingGuidanceAllTargets` to cover each pack × each target.
- Keep each pack's Go-fallback-vs-mirror sync test green.

## Notes
- This is a small, mechanical extension of a shipped fix. Governed by the same tripwire discipline (`harness-changes-cover-all-targets`) — but here the axis is packs, not targets.
- If the four pack bodies are meant to stay independent, a shared constant for the paragraph may not fit; a small duplicated block with a shared test is acceptable.

## Kickoff

v0.25.0 added "prefer the MCP surface; run `hero doctor` on version/schema confusion, not a migration story" guidance to the **engineering** pack only. The `pm`/`sales`/`chat` packs have independent AGENTS.md bodies and didn't get it.

**Status:** planning — a disclosed, deliberately-deferred scope item from `agent-hero-version-schema-confusion`.

**Pick up at:** find the per-pack AGENTS.md body generators in `internal/install/` (grep `generate*AgentsMdBody`) and the `domains/{pm,sales,chat}/AGENTS.md` mirrors + regen path. Add the same routing paragraph used in `generateEngineeringAgentsMdBody` to each, mirror it, and extend `TestHarnessNative_DoctorRoutingGuidanceAllTargets` to assert per-pack × per-target coverage.

**Files:** `internal/install/agents_md.go` (+ any per-pack body generators), `domains/pm/AGENTS.md`, `domains/sales/AGENTS.md`, `domains/chat/AGENTS.md`, `internal/install/harness_native_test.go`.
**Skip:** touching the engineering pack (done); adding any new guidance beyond the doctor/MCP-surface routing.
