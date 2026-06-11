---
title: "Landing Page & Docs Content Refresh — Peering, Semantic Search, Verify Gate"
type: feature
status: completed
size: small
priority: high
related_specs:
  - hero-landing-page
  - hero-docs-site
completed_at: 2026-06-11T02:30:50Z
---

## Goal

Bring the public landing page and docs sources in sync with features that have shipped since the last content pass: cross-repo peering, offline semantic search, the `hero verify` delivery gate, monorepo satellite installs, and the tripwire system.

## Kickoff

Deliver the content refresh spec `landing-docs-content-refresh`. Five targeted file edits — no code changes, no new files. Landing page: (1) swap stale "Tripwire" Coming Soon card for a cross-repo peering card, (2) add offline semantic search sentence to the Knowledge compounds why-card. Docs: (3) add `hero verify` delivery gate to delivery-and-debugging.md, (4) add monorepo install section to installation.md, (5) add tripwire system + spec scoring note to knowledge-base.md. Run hero verify after all five edits are in.

## Problem

Several shipped features are either absent from the website or missing from the docs, creating a gap between what Hero does and what visitors and users can learn about it:

- The "Coming Soon" section still lists Tripwire as in-progress — it shipped as `feat(context): tripwire system`.
- Cross-repo peering (`hero peer`, `hero handoff`) shipped in `feat(peering)` and is not mentioned on the landing page at all — yet it is one of the most differentiated capabilities Hero has.
- The embedded offline semantic search (Model2Vec, no API key) shipped in `feat(embeddings)` and is not surfaced in the website copy, despite being a clear differentiator.
- `hero verify` as the explicit delivery gate checkpoint shipped in `feat(verify): delivery gate enforcement` but is absent from `delivery-and-debugging.md`.
- Monorepo satellite installs shipped in `feat(install): monorepo satellite installs` but `installation.md` has no monorepo setup section.
- The tripwire system (forbidden-option guardrails) is mentioned nowhere in `knowledge-base.md`.

## Approach

All changes are targeted content edits to existing files. No new pages, no structural refactors.

### Landing page — `web/landing/site/index.html`

**Change 1 — Replace stale Tripwire "Coming Soon" card with cross-repo peering card.**

The Tripwire card reads:
> "Forbidden-option guardrails so model sessions can't wander into known dead ends. Encode 'we already tried this and it didn't work' once."

Replace with a cross-repo peering card:
- Label: `Shipped`
- Title: `Your projects know each other`
- Copy: `Hero connects sibling repos — agents can ask a peer for context, hand off specs across boundaries, and share conventions without copy-pasting. One corpus per project, linked by the graph.`

**Change 2 — Add semantic search line to the "Knowledge compounds" why-card.**

Current copy ends: "Models come and go; the corpus stays — committed to git, reviewable in PRs."

Append: "Searched semantically with built-in inference — no API key, no network call, private by default."

### Docs — `web/docs/src/`

**Change 3 — `workflows/delivery-and-debugging.md`**

Add a `hero verify` section after the delivery step. Explain: `hero verify` checks acceptance criteria against the delivered state and is the required checkpoint before a spec moves to `completed`. Include the command and a one-line description of what it validates.

**Change 4 — `getting-started/installation.md`**

Add a "Monorepo setup" section. Document: when a repo has multiple subfolder workspaces, run `hero install --target <harness>` from each subfolder. Hero installs a satellite `.hero/` scoped to that subfolder. Cross-reference the main install steps so it's findable.

**Change 5 — `concepts/knowledge-base.md`**

Add a "Tripwire guardrails" subsection explaining that conventions can encode forbidden options — approaches that were tried and ruled out. When Hero injects context, tripwires surface as explicit constraints the model respects. Add a note that spec scoring and sizing guidance also surface during knowledge review.

## Acceptance Criteria

- WHEN a visitor loads the landing page "What we're building next" section THE SYSTEM SHALL display a cross-repo peering card in place of the Tripwire card.
- WHEN a visitor reads the cross-repo peering card THE SYSTEM SHALL show the label `Shipped` (not `In progress`).
- WHEN a visitor reads the "Knowledge compounds" why-card THE SYSTEM SHALL include a sentence about offline/private semantic search with no API key required.
- WHEN a developer reads `delivery-and-debugging.md` THE SYSTEM SHALL describe `hero verify` as the explicit gate that must pass before a spec is closed, with the command shown.
- WHEN a developer reads `installation.md` THE SYSTEM SHALL find a monorepo or multi-workspace section explaining `hero install --target` for subfolder workspaces.
- WHEN a developer reads `knowledge-base.md` THE SYSTEM SHALL find an explanation of tripwire guardrails as a form of encoded institutional knowledge that agents respect.
- THE SYSTEM SHALL NOT introduce any new MIT license mentions in the landing page (those were removed in the prior pass).

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| AC-1 | Coming Soon shows cross-repo peering card (not Tripwire) | DONE | `grep "Your projects know each other" web/landing/site/index.html` → found; Tripwire card absent |
| AC-2 | Peering card label is `Shipped` (not `In progress`) | DONE | `<span class="soon-label">Shipped</span>` confirmed before peering `<h3>` |
| AC-3 | Knowledge compounds card includes offline semantic search sentence | DONE | Line added: "Searched semantically with built-in inference — no API key, no network call, private by default." |
| AC-4 | `delivery-and-debugging.md` describes `hero verify` gate with command shown | DONE | New "The verify gate" section with `hero verify <slug>` commands (+ `--skip-tests`, `--json` variants) and warning admonition; 6 occurrences |
| AC-5 | `installation.md` has monorepo section explaining subfolder installs | DONE | "Monorepo setup" section added with satellite concept, bash examples, tip admonition, and platform install cross-reference |
| AC-6 | `knowledge-base.md` explains tripwire guardrails as institutional knowledge | DONE | "Tripwire guardrails" subsection added with forbidden-option concept, `/decide` example, and spec scoring/sizing note |
| AC-7 | No new MIT mentions in landing page | DONE | `grep MIT web/landing/site/index.html` → 0 results |

### Exercise-the-feature check

- [x] Exercised: read each changed file section and ran `grep` verification — landing page Coming Soon section shows peering card ("Shipped", "Your projects know each other"); Knowledge compounds card includes semantic search sentence; delivery-and-debugging.md workflow diagram updated + "The verify gate" section present; installation.md "Monorepo setup" section present with examples and cross-reference; knowledge-base.md "Tripwire guardrails" subsection present. No MIT or "open source" strings remain in the landing page HTML.
