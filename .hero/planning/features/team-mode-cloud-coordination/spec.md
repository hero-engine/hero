---
title: "Team-Mode Cloud Coordination — Sync Mutable State via Hero Cloud, Keep Git for Source"
type: feature
status: planning
slug: team-mode-cloud-coordination
domain: engineering
priority: medium
created: 2026-06-23
tags: [hero-cloud, team-server, sync, coordination, next-mode, storage-topology, claims, events]
parent: hero-team-experience
relates-to: hero-team-server
---

# Team-Mode Cloud Coordination

## Goal

In team mode, sync Hero's **mutable coordination and projection state** through Hero Cloud / the team server instead of smuggling it through git. Git stays the home for **versioned source** (specs, knowledge). The transport for a given artifact should follow `next_mode`: git-as-transport for solo/offline (today's behavior), cloud-as-transport for teams — at which point those artifacts leave git entirely.

## Problem

Hero today uses git as a poor-man's sync layer for state that changes faster than people pull — the "handoff travels with commits" pattern. That is fine for a solo user but actively wrong for a team:

- **Claims (`claimed_by`)** — two teammates claiming the same spec produce a merge conflict; there is no server arbitration of who holds the lock.
- **Activity events (`.hero/events.log`)** — append-only mitigates conflicts but every machine only sees up to its last pull; `hero feed` / `hero velocity` / pulse are therefore stale, never live.
- **Projections (`SNAPSHOT.md`, `QUEUE.md`, `NEXT.md`, `next/*.md`)** — these are *regenerable* rollups, git-tracked **only** to fake cross-machine handoff. In a team they churn constantly and conflict, for no versioning benefit.
- **No presence** — there is no "who's active right now"; `claimed_by` is the closest proxy and it is stale by a push/pull cycle.

The nature of each artifact is fixed; what should change by mode is its **transport**.

## Data classification (from the 2026-06-23 survey)

| Tier | Artifacts | Today | Team-mode target |
|---|---|---|---|
| **Source** | `planning/`, `specs/`, `knowledge/`, `hero.json`, `AGENTS.md` | git | **git (unchanged)** |
| **Coordination** | claims (`claimed_by`), `events.log`, handoff/peering trail | git ✗ | **Cloud — drop from git** |
| **Projection** | `SNAPSHOT.md`, `QUEUE.md`, `NEXT.md`, `next/*.md` | git ✗ | **Cloud-served — gitignore in team mode** |
| **Cache** | `graph.db`, `index.db`, `cache/`, `knowledge/code/` | gitignored | local (graph.db already cloud-federates) |
| **Secret/local** | `hero.local.json`, `~/.hero/credentials.json`, `~/.hero/team.json`, `*.local.md` | gitignored | never shared |

## What benefits most from cloud (ranked)

1. **Claims / locks** — server-arbitrated, atomic; kills the "two people grab the same spec" race. *(Team server already exposes `/api/claims`.)*
2. **Activity events** — server-ingested stream → live feed/velocity/pulse; local `events.log` becomes a write-ahead buffer that syncs up.
3. **Projections** — regenerable, so serve from cloud and gitignore in team mode; removes the worst git churn.
4. **Handoff / peering trail** — server-mediated so originator and receiver see one shared state.
5. **Presence** (new capability) — only possible with a server.

## Don't greenfield — extend what exists

The cloud primitives are already in the codebase; this is an extension, not a new system:

- `hero login` — GitHub OAuth, token at `~/.hero/credentials.json` (`internal/cli/cloud_auth.go:16-264`).
- `hero sync cloud` — spec-metadata push, incl. `claimed_by` (`internal/cli/sync_cloud.go:20-182`), but push-only and non-arbitrating.
- `hero sync graph push/pull` — knowledge-graph **federation with `local`/`unit`/`team` scopes** (`internal/cli/sync_graph.go`). `graph.db` already bypasses git via this path — it is the template to copy.
- **Hero Team Server** — opt-in (`hero connect team`, `~/.hero/team.json`), exposes `/api/claims`, `/api/feed`, `/api/team/status` and a job queue (`internal/serve/team_coordination.go`, `internal/cli/connect_team.go`).

## Approach

Single lever: **make git-tracking of the Coordination + Projection tiers conditional on `next_mode == team`.** The `team`-scope already exists in graph federation; tag coordination data `team`-scope and route it to cloud, gitignored, instead of committed.

Phased, roughly in dependency order:

1. **Claims via server** — route `hero claim`/release through `/api/claims` with atomic arbitration when a team server / cloud is connected; fall back to frontmatter+git in solo mode.
2. **Events streaming** — stream `events.log` appends to the server; local file becomes a buffer. Feed/velocity/pulse read the server when connected.
3. **Projection transport** — when `next_mode == team`, gitignore `SNAPSHOT.md`/`QUEUE.md`/`NEXT.md`/`next/*.md` and serve/sync them via cloud; keep git-tracking in solo mode.
4. **Presence** — derive live presence from server-side claims + recent events.
5. **Handoff/peering** — move the cross-repo handoff trail to server-mediated state.

## Grounding facts

- Claims: `internal/cli/claim.go:107` (frontmatter) + `internal/tracking/tracking.go:24` (events.log) — no arbitration.
- Events: `internal/feed/feed.go:54-58` (AppendEvent); consumed by `internal/cli/velocity.go:43`, `internal/serve/metrics/metrics.go:119`, `internal/serve/pages/people/data/pulse.go:54`.
- Projections written by `internal/snapshot/projector.go:179`, `internal/cli/queue.go:99`, `internal/snapshot/pointers.go:41`; staged by the pre-commit hook (`internal/cli/next_hooks.go:44`).
- Graph federation scopes (`local`/`unit`/`team`): `internal/cli/sync_graph.go`.
- `next_mode` config key gates solo vs team handoff today.

## Acceptance Criteria

- AC-1: WHEN a team server / cloud is connected, THE SYSTEM SHALL arbitrate claims server-side so two users cannot both hold the same spec; solo mode keeps frontmatter+git.
- AC-2: WHEN connected, activity events SHALL stream to the server and `hero feed`/`velocity`/pulse SHALL read live server state rather than a local-only log.
- AC-3: WHEN `next_mode == team`, projection artifacts (`SNAPSHOT.md`/`QUEUE.md`/`NEXT.md`/`next/*.md`) SHALL be gitignored and synced via cloud; solo mode SHALL retain git-tracking unchanged.
- AC-4: Source-tier artifacts (specs, knowledge, `hero.json`) SHALL remain git-tracked in all modes.
- AC-5: Presence (who is active) SHALL be derivable from server state when connected.

## Open questions

- Offline behavior in team mode: buffer-and-reconcile semantics when a teammate works disconnected, then syncs.
- Whether the dubious-today git-tracked metadata (`install-state.json`, `version.json`) should be reclassified out of source regardless of mode.
- Decision-record candidate: "Storage transport follows `next_mode`" — capture as an ADR to anchor this and future cloud work.

## Kickoff

**Pick up at:** decide the ADR ("transport follows `next_mode`") first, then start with phase 1 (claims via `/api/claims`).

Cold-start prompt:
> Implement team-mode cloud coordination so mutable state (claims, events, projections) syncs via Hero Cloud / the team server instead of git, keyed on `next_mode`. The cloud primitives already exist — `hero sync graph push/pull` federates `graph.db` with `local`/`unit`/`team` scopes (`internal/cli/sync_graph.go`), and the team server exposes `/api/claims` and `/api/feed` (`internal/serve/team_coordination.go`). Start with phase 1: route `hero claim`/release through server-side arbitration when connected, falling back to frontmatter+git in solo mode. See the data classification and grounding facts above. This is a feature under the `hero-team-experience` initiative and relates to `hero-team-server`; it likely decomposes into the 5 phases listed — consider `/compose` to break it out.
