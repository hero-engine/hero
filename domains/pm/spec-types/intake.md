---
name: intake
title: Intake
domain: pm
description: Inbound feedback / request / signal. The Productboard-shaped surface. Promoted-to or merged-with an initiative, or rejected with reason. Source attribution is the trust signal.
location: .hero/planning/intake/{slug}/spec.md
kind:
  values: [customer, support, sales, internal, competitive]
  default: customer
  note: |
    Canonical sources. Drives the inbox swimlanes on the Intake board and the
    segment-weighting used by `prioritization-strategist`. The legacy `source`
    enum on the frontmatter is preserved as a finer-grained classifier
    (e.g. `support-escalation` vs `user-research`); `kind` is the registry-level
    rollup.
lifecycle:
  states: [new, triaged, linked, rejected]
  initial: new
  terminal: [linked, rejected]
  transitions:
    - from: new
      to: triaged
      gate: "intake-triager classified and clustered (target SLA: 24h)"
    - from: triaged
      to: linked
      gate: "Linked to an initiative OR merged into another intake"
    - from: triaged
      to: rejected
      gate: "Rejected with reason"
    - from: new
      to: rejected
      gate: "Obvious reject at intake (spam, off-product, duplicate of recently-rejected)"
frontmatter:
  required:
    - { name: title, type: string, classification: content }
    - { name: type, type: enum, values: [intake], classification: content }
    - { name: kind, type: enum, values: [customer, support, sales, internal, competitive], classification: content }
    - { name: status, type: enum, values: [new, triaged, linked, rejected], classification: org-state }
    - { name: source, type: enum, values: [customer-feedback, support-escalation, sales-note, internal, competitive, user-research, other], classification: content, note: "Finer-grained source than `kind`; preserved for backward compatibility and segment weighting." }
  optional:
    - { name: owner, type: enum, values: [pm, engineering, qa, design, ...], classification: org-state, note: "Intake is PM-owned; owner stays `pm` for the lifecycle." }
    - { name: source_url, type: string, classification: content, note: "Link to the inbound source (ticket, call recording, Slack thread)" }
    - { name: source_quote, type: string, classification: content, note: "The customer's own words. Preserved verbatim." }
    - { name: customer, type: string, classification: content, note: "Customer or contact name; respects PII conventions" }
    - { name: customer_segment, type: string, classification: content, note: "For segment-weighted prioritization" }
    - { name: tracker_id, type: string, classification: org-state }
    - { name: tags, type: list[string], classification: content }
    - { name: themes, type: list[string], classification: content }
    - { name: linked_initiative, type: ref(initiative), classification: content }
    - { name: merged_into, type: ref(intake), classification: content, note: "Set when duplicate-detector merges this into another" }
    - { name: rejection_reason, type: string, classification: content, note: "Required when status=rejected" }
    - { name: triaged_by, type: string, classification: org-state }
    - { name: triaged_at, type: date, classification: org-state }
    - { name: created, type: date, classification: org-state }
sections:
  required: [Signal]
  optional: [Investigation, Tasks, Linked decision]
  template: |
    # {Title}

    ## Signal
    The inbound signal as faithfully as possible. Customer quote
    verbatim where available. Don't paraphrase into product
    vocabulary — that loses fidelity.

    Source: {source} · {source_url}
    Customer: {customer} · {customer_segment}

    ## Investigation
    Optional. Populated by pm-investigator on ambiguous signals.
    Distinguishes evidence from hypothesis.

    ## Tasks
    Triage follow-ups (additional outreach to the customer, evidence to
    gather before linking, internal stakeholders to consult). Parsed
    identically to AC; `hero task add | list | done`.

    ## Linked decision
    The initiative this was linked to, or the reason it was
    rejected. Set on triage.
authoring_agent: intake-triager
investigation_agent: pm-investigator
duplicate_agent: duplicate-detector
relations:
  - { kind: links, target_type: initiative, cardinality: zero-or-one }
  - { kind: merged-into, target_type: intake, cardinality: zero-or-one }
---

# Intake spec type

An **intake** is an inbound signal — a customer asking for
something, a support escalation, a sales note, a competitive
observation. It's the surface that Productboard popularized: the
funnel from raw signal to roadmap decision.

The defining property: **source attribution is the trust signal**.
The customer's own words, the link back to the ticket, the segment
they're in — these are what make intake usable for prioritization
later. Paraphrasing erases trust.

## When to use

- Anything inbound that *could* shape the roadmap but hasn't been
  evaluated yet.
- Customer feedback in any form (ticket, call, NPS comment, sales
  note).
- Internal asks from sales, support, leadership, engineering.
- Competitive signals worth weighing.

## When NOT to use

- A clear, well-scoped customer request that maps obviously to an
  existing initiative — link it directly via the existing item's
  evidence section, then create the intake if there's any
  ambiguity worth preserving for future analysis.
- Bug reports with reproduction steps — those route to engineering's `/diagnose` (engineering pack) in
  the engineering domain.

## Lifecycle SLA

Every intake gets a status within **24 hours**. That's the
operational standard. `intake-triager` runs against the `new` queue.
The four outcomes:

- **linked** — to an existing or new initiative.
- **merged** — into another intake (duplicate). `merged_into`
  is set; the original stays accessible.
- **rejected** — with a reason. Visible in the funnel's rejected
  lane; the customer-facing communication is a separate workflow.
- **escalated** — for high-signal items that warrant immediate
  promotion (not a status; an `intake-triager` annotation that
  surfaces the item to PM leadership).

## Authoring rules

- `intake-triager` is the default agent. Loads
  `intake-classification`, `duplicate-detection`,
  `evidence-synthesis`, `customer-segment-weighting`.
- `duplicate-detector` runs at create-time. Surfaces duplicate
  candidates with confidence and the specific field overlap.
- `pm-investigator` is invoked on ambiguous signals to populate the
  Investigation section before triage decides.
- The Signal section must include the source attribution. Authoring
  agents must preserve `source_quote` verbatim — no rewriting.

## Anti-patterns

- Intakes without source attribution — useless for
  prioritization later.
- Intakes with paraphrased customer quotes — destroys trust
  signal.
- Auto-merging duplicates without human confirmation — recall matters
  more than precision; humans confirm.
- Triage SLA breach (>24h in `new`) — surfaces as a `/triage` sweep
  finding.

## Conflict policy

`content` fields (title, signal, source, source_quote, customer,
segment, themes, links, rejection_reason) — **Hero wins**. `org-state`
(status, triaged_by, triaged_at, tracker_id) — **tracker wins**.
