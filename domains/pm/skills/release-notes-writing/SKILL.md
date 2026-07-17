---
name: release-notes-writing
description: The two release-note shapes — customer-facing (user-benefit first, grouped by theme) and internal (spec slugs, owners, traceability back to the originating PRDs/initiatives). Turn a shipped window into notes that read like value delivered, not a changelog dump.
metadata:
  audience: stakeholder-communicator, pm-delivery-lead
  purpose: release-notes
---

## What I do

Provide the discipline for writing release notes in the two shapes a PM actually needs: the **customer-facing** cut that tells users what they can now do, and the **internal** cut that tells the team what shipped and where it came from. Both draw from the same shipped-window truth; they differ in what they lead with and who they're for. This skill packages the rules so release notes read like value delivered rather than a commit log pasted into a doc.

## When to use me

Load this skill when:

- drafting release notes for a shipped window (`/release-notes`, "what shipped this week")
- announcing a launch or a set of shipped stories to customers
- writing the internal "what shipped" update for the team or leadership
- deciding what belongs in a customer note versus an internal one

## Shipped-status source of truth

Pull "what shipped" from the **graph**, not from tracker workflow status or PR-merge dates. A spec is shipped only when its `status` is `completed` **and** the most recent `owner_history` row records engineering close-out. Specs whose work merged but whose `owner_history` never recorded a close-out are orphan deliveries — surface the gap, don't include them in the notes.

## Customer-facing shape

Written for the user. The rules:

- **Lead with the user benefit, not the feature name.** "You can now export filtered CSVs from any list view" — not "Added CSVExport handler to ListController." The reader should learn what they can *do*, not what the team *built*.
- **Call out behavior changes that affect existing workflows.** A changed default, a moved control, a removed option — anything that alters what an existing user already does — gets an explicit callout. Silent behavior changes generate support tickets.
- **Group by theme, not by spec.** Users don't think in spec slugs. Cluster related items under a capability heading ("Faster exports", "Better search") rather than listing one bullet per merged spec.
- **Link to docs.** Where a capability has a help article or a how-to, link it. The note announces; the docs explain.

Write to `.hero/planning/release-notes/<window-or-tag>/customer.md`. Tone is outcomes-not-features throughout.

## Internal update shape

Written for the team and leadership — the traceable "what shipped" record. The rules:

- **Keep the spec slugs.** Internal readers trace by slug; the customer note drops them, the internal note keeps them.
- **Name the owners.** Who shipped each item, for follow-up and credit.
- **Link back to the originating PRD / initiative.** Every shipped item points back to the bet it delivered, so the team can trace outcome → delivery and `/retro` knows which metrics to evaluate.
- **Plain and specific.** The internal cut is a status record, not launch copy — no marketing framing.

Write to `.hero/planning/release-notes/<window>/internal.md` (via `--internal` or when the ask is "what shipped this week"-style).

## Anti-patterns

- **Changelog dump with no narrative.** A flat list of merged specs with no theme and no benefit is a git log, not release notes. Group and frame.
- **Release notes that read like commit messages.** "Refactored the export pipeline" tells a customer nothing. Translate to the user-visible benefit.
- **Marketing-flavor everything.** Overselling a small fix as a headline launch erodes trust in every future note. Match the tone to the size of the change.
- **Silent behavior changes.** Shipping a changed default without a callout in the customer note. If existing users will notice, say so.
- **Including orphan deliveries.** Listing a spec whose `owner_history` never recorded engineering close-out. It isn't shipped by the graph's record — surface the gap instead.

## Cross-references

- `stakeholder-communication` — customer and internal notes are two specific audience cuts; the parent skill covers the full four-audience discipline.
- `outcomes-over-outputs` — "lead with the user benefit" is outcome-framing applied to a shipped item.
- `pm-agent-doctrine` — no fabricated metrics or testimonials in a note; shipped-status is grounded in the graph, not asserted.
- `cross-domain-graph-query` — how to read the shipped window (`completed` + `owner_history` close-out) from the graph.
