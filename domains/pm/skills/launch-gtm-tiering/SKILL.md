---
name: launch-gtm-tiering
description: Size a launch into Tier 1/2/3 by impact, then run the right amount of go-to-market motion — the five-phase checklist (alignment → positioning → enablement → launch → post-launch) scoped to the tier. A Tier-3 patch does not get a Tier-1 motion, and a company-moving launch does not get a release note.
metadata:
  audience: stakeholder-communicator, product-strategist, roadmap-curator, and the /launch command
  purpose: operational
---

## What I do

Match the go-to-market effort to what the launch actually warrants. The two failure modes are symmetric: over-launching a minor change (a full GTM motion, exec reviews, and a field enablement session for a settings-page tweak) burns credibility and calendar; under-launching a major one (a company-moving capability shipped with a one-line release note) wastes the build. This skill supplies the **tier rubric** that sizes a launch by impact and the **five-phase checklist** that says which motion each tier runs. When `stakeholder-communicator` plans an announcement or `/launch` builds a launch plan, this is the scoping discipline.

Like every PM artifact, a launch plan is **corpus-grounded and decision-gated** (`pm-agent-doctrine`): the tier is a *recommendation* the PM confirms, blast-radius and revenue claims cite real segment/analytics data (not "this feels big"), and the checklist is a proposal owned by a human, not an auto-scheduled campaign.

## When to use me

- planning any launch or announcement (`/launch`, or a `stakeholder-communicator` exec/customer cut for a shipped item)
- deciding how much GTM motion a shipped or about-to-ship item warrants
- auditing a launch plan for over- or under-investment relative to impact
- sequencing the phases and gates before a ship date

## The tier rubric — size by impact

Assign every launch to exactly one tier. Size against the **rubric dimensions**, not gut feel:

| Dimension | Tier 1 (major / company-moving) | Tier 2 (standard feature) | Tier 3 (minor / incremental) |
|---|---|---|---|
| **Blast radius** | All segments, new market, or platform-level shift | A defined segment or major workflow | Narrow slice; existing users of one feature |
| **Revenue / segment impact** | Moves a company metric; opens/defends a segment | Meaningful for a segment; upsell/retention lever | Negligible direct revenue effect |
| **Net-new vs. incremental** | Net-new capability or category | New feature on an existing surface | Enhancement, fix, or polish |
| **Competitive stakes** | Table-stakes gap closed or a differentiator opened | Keeps pace / modest edge | None |

**Reading the rubric:** a launch that lands in the Tier-1 column on **any** of blast radius, revenue, or competitive stakes is Tier 1 — the highest-hitting dimension pulls the tier up (a niche feature that closes a deal-blocking competitive gap is Tier 1 even if its blast radius is small). Tier 2 is the default for a real feature launch. Tier 3 is reserved for changes with negligible impact on all four dimensions. When two people disagree on the tier, the rubric dimension they disagree on is the conversation — ground it in data (which segment, how much revenue, whose competitive claim), don't split the difference.

**The three tiers, in one line each:**

- **Tier 1 — major / company-moving.** Full GTM motion: exec + field + comms alignment, positioning, enablement, coordinated launch, measured post-launch. Every phase runs.
- **Tier 2 — standard feature launch.** Positioning + enablement + announcement. The middle phases run; alignment is lighter, post-launch is a metric check.
- **Tier 3 — minor / incremental.** Release note + in-product surface. Phases collapse into a single "announce it where users will see it" step.

## The five-phase checklist

Every launch moves through the same five phases; the **tier decides which phases run in full and which collapse.**

1. **Alignment** — agree on what's shipping, to whom, and why now. *Artifacts/gates:* one-line launch goal + target segment; tier assignment (this skill); named launch owner; a go/no-go date. *Tiers:* Tier 1 runs a full alignment (exec sign-off, cross-functional owners per phase); Tier 2 runs a light version (owner + goal + date); Tier 3 collapses it into the release-note decision.
2. **Positioning** — the message: what it is, who it's for, why it beats the alternative. *Artifacts/gates:* positioning statement (route to `positioning-canvas`), the core message, proof points grounded in real evidence. *Tiers:* Tier 1 + Tier 2 require positioning; Tier 3 uses a one-line "what changed and why you care."
3. **Enablement** — make the people who touch customers ready. *Artifacts/gates:* internal FAQ, sales/support talking points, demo or docs, objection handling. *Tiers:* Tier 1 runs full field + support enablement; Tier 2 runs a lightweight support-doc + FAQ; Tier 3 skips enablement (the release note is the enablement).
4. **Launch** — ship the announcement on the coordinated date. *Artifacts/gates:* announcement copy (blog / email / in-app), in-product surface, the go-live checklist, comms sequencing. *Tiers:* Tier 1 coordinates a multi-channel launch on a set date; Tier 2 ships an announcement + in-app surface; Tier 3 ships a release note + in-product callout.
5. **Post-launch** — did it land, and what do we do next. *Artifacts/gates:* the success metric + baseline (route to `metrics-design`), adoption read at a set checkpoint, feedback loop back to discovery, a retro on the launch itself. *Tiers:* Tier 1 runs an instrumented post-launch review; Tier 2 checks the adoption metric at a checkpoint; Tier 3 watches for regressions only.

**Tier → phase coverage, at a glance:**

| Phase | Tier 1 | Tier 2 | Tier 3 |
|---|---|---|---|
| Alignment | Full | Light | Collapsed into release-note decision |
| Positioning | Full | Full | One-liner |
| Enablement | Full | Light | Skipped |
| Launch | Multi-channel | Announcement + in-app | Release note + callout |
| Post-launch | Instrumented review | Metric checkpoint | Regression watch |

## Worked example

A team ships SAML SSO. Blast radius: enterprise segment only. Revenue: unblocks two pipeline deals + defends renewals. Competitive stakes: the gap is deal-blocking in enterprise. Two dimensions (revenue, competitive) sit in the Tier-1 column → **Tier 1**, even though blast radius is one segment. So it runs the full motion: exec alignment (this defends renewal revenue), positioning against the competitor whose SSO was the wedge, field + support enablement (sales needs to reach back to the stalled deals), a coordinated announcement to the enterprise segment, and an instrumented post-launch read on the two deals + renewal cohort. Sizing it Tier 2 "because it's one segment" would under-launch a revenue-defending capability — the rubric catches that.

## Anti-patterns

- **Tiering by effort built, not impact delivered.** A six-month project can be a Tier-3 launch; a one-week feature can be Tier 1. Size by the four impact dimensions, not the engineering cost.
- **Splitting the difference on a contested tier.** "Somewhere between 1 and 2" is a dodge. Name the dimension in dispute and ground it in segment/revenue/competitive data.
- **Running every phase for every launch.** Tier 3 with a full enablement motion burns the team's credibility for the next Tier-1 ask. Collapse the phases the tier doesn't warrant.
- **Skipping post-launch on a Tier 1.** A company-moving launch with no baseline and no adoption read can't tell success from noise — the most expensive launches are the ones no one measured.
- **Ungrounded impact claims.** "This is huge" with no segment, revenue number, or competitive citation is the free-association `pm-agent-doctrine` forbids. The tier rubric runs on evidence.
- **Auto-scheduling the campaign.** The launch plan is a recommendation; the PM owns the go/no-go and the date. The checklist proposes; it does not fire.

## Cross-references

- `pm-agent-doctrine` — the tier is a grounded, human-gated recommendation, not an auto-decision.
- `positioning-canvas` — the positioning phase's core artifact (Dunford five-component positioning).
- `metrics-design` — the post-launch success metric + baseline.
- `stakeholder-communication` — the audience-shaped announcement cuts (exec / customer / internal) the launch phase ships.
- `release-notes-writing` — the Tier-3 (and Tier-2 announcement) release-note format.
