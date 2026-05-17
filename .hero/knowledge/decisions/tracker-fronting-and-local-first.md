---
title: Tracker Fronting Is Local-First; Tracker Is a Backing Store
type: decision
status: proposed
created: 2026-05-16
tags: [tracker, integration, local-first, architecture, decision, hero-wide]
relations:
  - target: hero-domains
    kind: applies-to
  - target: hero-pm
    kind: applies-to
  - target: cross-repo-peering-local-first
    kind: related
---

# Tracker Fronting Is Local-First; Tracker Is a Backing Store

## Decision

Hero is the working surface. Trackers (Jira, Linear, GitHub) and Hero
Cloud are **backing stores**, not front doors. Three operating modes
share one UX:

1. **Standalone** — `.hero/` filesystem is source of truth.
2. **Hero Cloud-backed** — Cloud is the sync layer for multi-user
   workspaces; the local filesystem stays authoritative for the
   working session.
3. **Tracker-fronted** — the org system of record is a tracker; Hero
   fronts it transparently. Writes land locally first; propagation
   to the tracker is async; tracker-side changes flow back via
   webhook + poll.

The UX is identical in all three modes. The user does not see a
"connected to" or "syncing" status bar, no spinners on write, no
"refresh from Jira" button. Hero artifacts are round-trippable
markdown specs in `.hero/`; nothing about the mode determines what
the user types into.

This principle applies hero-wide — to every domain pack, not just
PM.

## Context

Hero's first integrations (Jira, Linear, GitHub) were modeled as
**import**: pull tracker issues down into `.hero/specs/` as scaffolds
the user works against, then push status back up. The user interacted
with Hero as the front door and the tracker as the database.

The PM domain pack design (`hero-pm/spec.md`) forces a sharper
question: a PM team running Hero on a Jira-mandated org cannot
relegate Jira to "occasional sync." Jira is the org system of record;
people need it to reflect reality continuously. But Hero is also where
the actual work happens — drafting PRDs, refining stories, accepting
inline agent proposals, executing the cross-domain handoff.

The risk: if the architecture treats the tracker as a peer that the
user occasionally "syncs with," the user experience fractures (manual
sync gestures, stale-state confusion, conflict modals on every write).
If it treats the tracker as the primary store, latency and offline
gaps make the working surface feel unresponsive.

Local-first writes with async propagation resolves the tension: every
write is instant against the local store; propagation happens in the
background; reconciliation has a clear default policy.

## Options considered

1. **Tracker-of-record, Hero as cache.** Every write goes to the
   tracker first; Hero re-reads after each operation. The user is
   working "through Hero into Jira."
   - Pros: no reconciliation problem in steady state; the org's
     existing tooling sees changes immediately.
   - Cons: every keystroke is a network round-trip; offline is
     broken; the working surface feels like a thin client; agent
     loops (inline-propose, refine, draft AC) become unusable under
     latency.

2. **Hero-of-record, tracker as periodic sink.** All writes happen
   locally; periodic batch sync pushes to the tracker.
   - Pros: fast working surface; offline-tolerant.
   - Cons: the tracker is stale enough that stakeholders looking at
     Jira don't see PM work; "Hero workspaces feel disconnected from
     the rest of the company."

3. **Local-first writes, async propagation, clear conflict policy.**
   Writes go to the local store first; an async worker propagates to
   the tracker on a short cadence; tracker-side changes flow back via
   webhook + poll. Conflicts use a fixed policy (Hero wins for
   content, tracker wins for org-state).
   - Pros: working surface stays fast; tracker stays current within
     seconds; the user never sees a sync gesture; offline is a
     non-event (writes queue, propagate on reconnect).
   - Cons: two write paths to maintain; reconciliation logic is real
     code; the conflict-policy split has to be carefully partitioned.

## Decision

Option 3. Local-first writes, async propagation, fixed conflict
policy.

The conflict-policy split:

- **Hero wins for content** — artifact body, description, acceptance
  criteria, PRD sections, kickoff content, inline-proposed agent
  outputs. The tracker mirror is a derived view of the local content.
- **Tracker wins for org-state** — assignee, sprint, cycle, workflow
  status, priority (when org-mandated), labels (when org-enforced).
  Hero treats these as read-mostly and surfaces them as header chips,
  not as content the user authors.

## Consequences

- The `DomainIntegration` interface (per `hero-domains` initiative)
  must support **local-first write semantics** as a first-class
  contract, not as an opt-in mode. The same code path serves all
  three operating modes; only the propagation step varies.

- **No syncing spinners, no sync buttons, no "refresh from tracker"
  affordance.** If a sync indicator is needed for transparency, it
  lives in chrome status, not in artifact UI. The artifact pane
  behaves as if the local store is the only store.

- **Field schema must distinguish content fields from org-state
  fields explicitly.** Domain spec-type registrations declare which
  frontmatter fields are content (Hero-authoritative) and which are
  org-state (tracker-authoritative). The spec-type-registry primitive
  must accept this declaration.

- **Re-read on artifact open** reconciles tracker-authoritative
  fields without blocking the read. The local content is rendered
  immediately; org-state chips update when the read returns.

- **Tracker outage is non-fatal.** Local writes queue; the user keeps
  working; propagation resumes on reconnect. The artifact UI does not
  visibly degrade.

- **Initial import** (the v1 onboarding moment for PM — Jira epics
  become `epic` specs, stories become `story` specs) is a one-time
  hydration into the local store. After that, the tracker is fronted,
  not imported-from.

- **Multi-user reconciliation** lives in Hero Cloud (mode 2) for
  Hero-authoritative content, and in the tracker (mode 3) for
  org-state. Cross-domain handoff edges (`story → feature`) are
  Hero-authoritative regardless of mode — the tracker cannot
  represent them.

- **The principle applies hero-wide.** Every future domain pack (QA,
  Design, Data) inherits this stance. Integration interfaces designed
  during `hero-domains` must not assume tracker-of-record.

## Out of scope (for this decision)

- The wire format of the propagation queue and the polling cadence
  belong to integration-implementation specs, not this decision.
- Tracker → Hero schema mapping per provider (Jira custom fields,
  Linear cycle objects, GitHub issue labels) is provider-specific
  design, captured per-integration.
- Multi-tracker workspaces (PM project on Jira handing off to an
  engineering project on Linear — see `hero-pm/spec.md` unknown #2)
  inherit this decision but add scope; resolve there.
