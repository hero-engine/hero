---
title: Sprint Plan — dashboard-fix-and-rebuild (2026-05-19)
slug: sprint-2026-05-19
type: note
status: active
tags: [sprint, planning, delivery, dashboard, hero-serve]
created: 2026-05-19
sprint:
  name: dashboard-fix-and-rebuild
  started_at: 2026-05-19
  mode: continuous-flow
  owner: chet-bellows
  specs:
    - dashboard-user-identity-os-env-mismatch
    - dashboard-delivery-events-never-emitted
    - dashboard-adapter-state-hardcoded
    - dashboard-inbox-misses-most-activity-sources
    - dashboard-now-headline-misleading-when-empty
    - hero-serve-multi-project
    - hero-serve-dashboard-redesign
    - hero-serve-project-section
---

# Sprint Plan — dashboard-fix-and-rebuild

## Sprint Goal

**Fix the dashboard data/UI bugs so the existing pages tell the truth, then land the multi-project foundation so the redesign work can begin.**

The sprint is intentionally **front-loaded with five quick bug fixes** that
restore data trust in the existing `/now`, `/work`, and inbox surfaces, then
moves to the multi-project foundation that the broader redesign work depends
on. The dashboard cannot tell a true story until "you" is the right person,
shipped-spec counts reflect reality, and the adapter chip stops lying. Once
those are in, multi-project (item 6) unblocks both the redesign (item 7) and
the project section (item 8).

## Sprint Window

- **Start:** 2026-05-19
- **Mode:** Continuous-flow, no hard end date. Solo developer (chet-bellows).
- **Realistic horizon:** 1–2 weeks for the bug cluster (items 1–5) plus the
  multi-project foundation (item 6). Items 7 and 8 are in scope but will
  likely span into a follow-on sprint — see "Carry-over expectation" below.

## Selected Specs (delivery order)

| # | Slug | Type | Sev/Pri | Size | Owner | Depends on | Summary |
|---|------|------|---------|------|-------|------------|---------|
| 1 | `dashboard-user-identity-os-env-mismatch` | bug · code | high | S | chet-bellows | — | Replace `$USER`-based identity with `git config user.name`; restores author-filtered tiles and "On your plate" cards. |
| 2 | `dashboard-delivery-events-never-emitted` | bug · design | high | S–M | chet-bellows | — | Emit `delivery_complete` events from `hero spec complete` so shipped counts are real. |
| 3 | `dashboard-adapter-state-hardcoded` | bug · code | high | S | chet-bellows | — | Source adapter chip from actual runtime state, not a hardcoded "via hero-code" string. |
| 4 | `dashboard-inbox-misses-most-activity-sources` | bug · code | medium | S | chet-bellows | — | Inbox sources beyond proposals + inbound handoffs; un-hardcode the proposals=nil shortcut. |
| 5 | `dashboard-now-headline-misleading-when-empty` | bug · design | medium | S | chet-bellows | — | Stop composing two empty signals ("no agent running · since 19h ago") into a false story. |
| 6 | `hero-serve-multi-project` | feature · foundation | — | L | chet-bellows | — | Multi-project lifecycle and dashboard awareness; foundation under items 7 and 8. |
| 7 | `hero-serve-dashboard-redesign` | feature | — | L | chet-bellows | 1, 2, 3, 6 | Redesign Now and Work pages; assumes data flows correctly and multi-project is live. |
| 8 | `hero-serve-project-section` | feature | — | L | chet-bellows | 6 | Per-project info, utilities, and operations page; depends only on the multi-project foundation. |

Size key: S = hours, S–M = up to a day, M = 1–2 days, L = multi-day.

## Dependency Graph

```
 (independent bug fixes — different root causes, sequenced for serial delivery)
  1 ──┐
  2 ──┼──► 7  hero-serve-dashboard-redesign
  3 ──┘

  4   (independent — inbox sourcing)
  5   (independent — headline copy/UX)

  6 ──┬──► 7  hero-serve-dashboard-redesign
      └──► 8  hero-serve-project-section
```

Notes on the graph:

- Items **1, 2, 3** are independent of each other (different root causes) but
  together gate item **7** — the dashboard redesign assumes data flows
  correctly. Shipping the redesign on top of broken data would re-bake
  today's bugs into the new layout.
- Items **4 and 5** are independent of everything else. They are scheduled
  inside the bug cluster purely to bank quick wins early in the sprint;
  neither blocks nor is blocked by anything in the sprint.
- Item **6** is independent of all five bug fixes but blocks both **7** and
  **8**. Multi-project must land before either downstream feature can claim
  acceptance criteria.
- Item **7** depends on items 1, 2, 3, and 6.
- Item **8** depends on item 6 only — it does *not* depend on the bug
  fixes, so if the redesign (7) slips, the project section (8) can still
  proceed in parallel-shape after 6 lands.

## Capacity Allocation

Single-developer project. All work assigned to **chet-bellows**. The
dashboard's top-nav "Solo" badge reflects this — no team-mode partitioning
is in play and the sprint plan does not assume any.

| Slot | Spec | Estimate | Notes |
|------|------|----------|-------|
| 1 | `dashboard-user-identity-os-env-mismatch` | half-day | Highest-impact prerequisite — until it lands, the dashboard literally can't tell who "you" are. |
| 2 | `dashboard-delivery-events-never-emitted` | half-to-full day | Touches `hero spec complete` writer + the dashboard's count reader; verify events surface end-to-end. |
| 3 | `dashboard-adapter-state-hardcoded` | half-day | Localized change in `internal/serve/server.go` + chip template. |
| 4 | `dashboard-inbox-misses-most-activity-sources` | half-day | Expand inbox sources; un-hardcode proposals. |
| 5 | `dashboard-now-headline-misleading-when-empty` | half-day | Headline composition logic + empty-state copy. |
| 6 | `hero-serve-multi-project` | 3–5 days | Multi-day; the keystone for items 7 and 8. |
| 7 | `hero-serve-dashboard-redesign` | 4–6 days | **Likely spans into next sprint.** |
| 8 | `hero-serve-project-section` | 4–6 days | **Likely spans into next sprint.** Can run after 6 in parallel-shape with 7 if scheduling allows. |

Total realistic in-sprint commitment (1–2 weeks): items **1–5 closed** plus
item **6 substantially landed or closed**. Items 7 and 8 are in scope but
explicitly flagged as carry-over candidates.

### Carry-over expectation

Items 7 and 8 are explicitly **expected to span into a follow-on sprint**.
This is by design: the sprint trades a tighter "everything must close"
commitment for honest sequencing — the redesign cannot start cleanly until
its prerequisites land, so it is staged inside this sprint rather than
deferred to a later one.

## Risks and Blockers

1. **Identity fix (item 1) is a hard prerequisite for credibility.** Until
   it lands, every author-filtered tile on the dashboard reads zero. If any
   other work in the sprint is being validated by "look at the dashboard," it
   must run after item 1 closes.

2. **Each item touches `internal/serve/server.go` to some degree.** This
   forces **serial delivery** — there is no parallelism between items in
   this sprint. Treat the order above as binding rather than advisory; do
   not pick up item N+1 while item N is mid-edit on `server.go`.

3. **Item 2 (delivery events) is a "design" root-cause bug, not a code
   bug.** It introduces a new event type into the lifecycle. Confirm the
   event schema and the reader/writer contract before implementing — this
   is the one bug in the cluster where a half-baked fix could pollute the
   event log retroactively.

4. **Items 6–8 are multi-day each.** The five bug fixes are hours-to-a-day;
   the three feature items are multi-day. Sprint scope is honest about this:
   five fixes + foundation **closed in-sprint**, the two large redesign
   features **likely carry over**.

5. **Bug fixes 1–3 quietly invalidate dashboard test fixtures.** Identity,
   adapter chip state, and shipped-spec counts are commonly hardcoded in
   serve-page tests. Audit and update fixtures with each fix; do not skip
   the regression-test step in each spec's Validation section.

6. **No team capacity buffer.** Solo flow. Unplanned bugs or context loss
   will push items 7 and 8 further into the carry-over window — that's
   acceptable, but means item 6 (the foundation) is the real "must-close"
   commitment after the bug cluster.

## Definition of Done for the Sprint

- All 5 bug specs (items 1–5) moved from `planning` → `completed` via
  `hero spec complete`.
- `hero-serve-multi-project` (item 6) either `completed` or `delivering`
  with at least Phase 1 acceptance criteria met.
- `/now` reflects real author-filtered data, real shipped-spec counts, an
  honest adapter chip, a populated inbox where appropriate, and a headline
  that doesn't lie when signals are empty.
- Dashboard redesign (item 7) and project section (item 8): in scope, with
  carry-over to the follow-on sprint acceptable if reached but not
  finished.
- Retros captured for each closed spec via `/retro`.

## Explicitly Out of Sprint

- Any dashboard work that depends on team-mode, multi-user identity, or
  shared workspace partitioning — that belongs to `hero-team-server`.
- Marketing / docs / landing surfaces — separate initiative.
- Other in-flight `delivering` specs in the workspace (graph-conflict
  detection, e2e suites, agent-outposts, etc.) — they continue but are not
  part of *this* sprint's commitment.

## Kickoff

The user will explicitly kick off delivery on item 1
(`dashboard-user-identity-os-env-mismatch`) after this plan is saved. Do
not auto-start `/deliver` from the sprint plan.

When ready, the kickoff command is:

```
/deliver dashboard-user-identity-os-env-mismatch
```
