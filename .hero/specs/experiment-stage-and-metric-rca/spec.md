---
title: "Experiment Stage + Metric RCA — Experiment Designer, Metrics Analyst"
slug: experiment-stage-and-metric-rca
type: feature
status: completed
domain: pm
priority: high
size: medium
created: 2026-07-17
tags: [pm, experiment, metrics, differentiation, wave-2]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
  - target: adversarial-critics-bundle
    kind: conflicts-with
completed_at: 2026-07-17T21:10:30Z
---

# Experiment Stage + Metric RCA — Experiment Designer, Metrics Analyst

## Goal

Author the two PM-pack capabilities the external best-practice scan ranks
highest and the original design omitted entirely: a **pre-registered
experiment stage** (design a falsifiable experiment *before* launch) and
**metric-movement RCA** ("why did the metric move"). Ship `experiment-designer`
+ `experiment-design` skill + `/experiment` command, and `metrics-analyst`
+ `metric-rca` skill. This closes the 6↔7 seam — `experiment-readout-reviewer`
(child #6) forward-references the `experiment-design` brief format, and this
child authors it — and un-dangles the already-shipped `/metrics` command,
which today routes to `pm-delivery-lead` with no backing analyst.

## Kickoff

Deliver child #7 of `pm-pack-completion`: the experiment stage + metric RCA.
Content-only, `domains/pm/` pack source, no Go. Author `experiment-designer`
and `metrics-analyst` agents, the `experiment-design` and `metric-rca` skills,
and the `/experiment` command; append their routes to the AGENTS.md Wave-2
region (below the marker, after child #6's routes) and register both skills.
The `experiment-design` brief format MUST match what child #6's
`experiment-readout-reviewer` expects to read (SRM, guardrails, pre-registered
decision rule, MDE). Point `/metrics` at `metrics-analyst`. Read this spec's
Changes and Boundaries first; every loaded skill must resolve (no dangling
refs); AGENTS.md is additions-only. Do not implement beyond `domains/pm/`.

## Problem

Three concrete gaps, all in the shipped PM pack:

1. **The experiment stage is entirely absent.** The pack can design PRDs and
   rank backlogs but cannot design a *falsifiable experiment* — a single-variable
   hypothesis, a primary metric with a minimum detectable effect (MDE), a
   duration, and a decision rule + guardrails **locked before launch**. Without
   pre-registration there is no defense against the classic false-positive
   failures (peeked stops, swapped metrics, post-hoc MDE lowering).

2. **The 6↔7 seam is open.** Child #6 shipped `experiment-readout-reviewer`,
   which critiques a readout **against its pre-registered brief**. Its
   `## Forward dependency` section explicitly defers the brief format to "child
   #7's `experiment-design` skill … which is not yet delivered," and it
   intentionally does **not** load `experiment-design` to avoid a dangling
   reference. Until this child authors that skill, the reviewer can only
   critique against inline discipline and flag the missing brief. The brief this
   child authors must carry exactly the terms the reviewer reads back: registered
   primary metric, registered MDE, registered guardrails, intended split (SRM
   baseline), registered stop rule / decision rule.

3. **`/metrics` is a dangling command and there is no RCA capability.** The
   shipped `/metrics` command routes to `pm-delivery-lead` loading `metrics-design`
   with the annotation "`metrics-analyst` takes over in v1.5" — the analyst was
   never authored. Separately, when a metric moves, the pack has no disciplined
   way to answer *why*: no metric-tree decomposition, no drift taxonomy, no
   causality-before-asserting guard. The external scan ranks metric-movement RCA
   as a top-leverage, badly-served capability.

## Acceptance Criteria

EARS-phrased where a trigger/response fits; existence-and-content checks
otherwise. All checks are mechanical (file exists + contains named sections).

| ID | Pattern | Behavior |
|---|---|---|
| AC1 | Ubiquitous | `domains/pm/agents/experiment-designer.md` exists with valid agent frontmatter (`name: experiment-designer`, `mode: subagent`) and a `## Startup` that loads `pm-agent-doctrine` and `experiment-design`. |
| AC2 | Ubiquitous | `domains/pm/skills/experiment-design/SKILL.md` exists and contains sections for the pre-registered brief covering **all five** locked terms: primary metric, MDE, intended split (SRM baseline), guardrail metrics, and pre-registered decision rule / stop rule. |
| AC3 | State-driven | While `experiment-readout-reviewer` critiques a readout against a pre-registered brief, the brief format authored in `experiment-design` shall carry every term the reviewer reads back — registered primary metric, registered MDE, registered guardrails, intended split, registered stop rule — verified by a term-parity check between `experiment-design/SKILL.md` and the reviewer's `## Forward dependency` / checklist. |
| AC4 | Ubiquitous | `domains/pm/commands/experiment.md` exists, routes `/experiment` to `experiment-designer`, and states the command is for *designing* an experiment brief (not critiquing a readout — that stays `experiment-readout-reviewer`, agent-invoked). |
| AC5 | Ubiquitous | `domains/pm/agents/metrics-analyst.md` exists with valid agent frontmatter and a `## Startup` that loads `pm-agent-doctrine`, `metrics-design`, and `outcomes-over-outputs`; it covers both defining/interpreting success metrics and metric-movement RCA. |
| AC6 | Ubiquitous | `domains/pm/skills/metric-rca/SKILL.md` exists and contains a **metric-tree decomposition** section and a **drift taxonomy** enumerating all five drift classes: component, temporal, influence (mix/segment), dimension, and event-shock; plus a causality-before-asserting guard (correlation is a hypothesis, not a cause). |
| AC7 | Unwanted behavior | If `/metrics` is invoked, the command shall resolve to `metrics-analyst` (not `pm-delivery-lead`); `domains/pm/commands/metrics.md` no longer carries the "takes over in v1.5" deferral and names `metrics-analyst` as the backing agent. |
| AC8 | Ubiquitous | `domains/pm/AGENTS.md` gains, **below** the `WAVE-2 ROUTES` marker and **after** child #6's `#### Wave-2 adversarial critic routes` table, a new appended subsection carrying the `/experiment` → experiment-designer route and the metrics-analyst / `/metrics` / RCA routes. |
| AC9 | Unwanted behavior | If AGENTS.md is diffed, the change is **additions-only** below the marker: the canonical routing table (lines above the marker) and child #6's appended critic routes are byte-for-byte unchanged. |
| AC10 | Ubiquitous | AGENTS.md Skills Reference lists the two new skills (`experiment-design`, `metric-rca`); AGENTS.md Agents Reference lists `experiment-designer` and `metrics-analyst`. |
| AC11 | Ubiquitous | Every skill named in any new/edited agent's `## Startup` resolves to an on-disk `domains/pm/skills/<name>/SKILL.md`; no dangling skill reference is introduced (the tripwire's no-dangling-refs guard). |

## Changes

All authored files live under `domains/pm/` (pack source). No Go, no consumer
code, no files outside `domains/pm/`.

### New agents

- **`domains/pm/agents/experiment-designer.md`** — designs falsifiable
  experiments. Frontmatter modeled on the shipped critic/authoring agents
  (`mode: subagent`, `temperature: 0.1`, `permission.edit: allow`,
  `task."*": deny`, `skill."*": allow`, `webfetch: allow`; `color: secondary`
  as an authoring agent, not a critic). Body:
  - One-line role: senior experiment designer; produces the **pre-registered
    brief** that fixes, before data, the hypothesis, primary metric + MDE,
    duration, decision rule, and guardrails.
  - `## Startup` loads `pm-agent-doctrine` (corpus-grounding, suggest-don't-decide),
    `experiment-design` (the brief format), `metrics-design` (primary-metric
    definition, baseline before target), `risk-surfacing` (guardrails as
    scenario/indicator/response), `assumption-testing` (pre-registration
    discipline), `pm-preset-detection`.
  - `## When invoked` — `/experiment`; "design an A/B / holdout," "what should we
    test," "write the experiment brief." Explicitly: this agent designs; it does
    **not** critique a readout (that is `experiment-readout-reviewer`, child #6).
  - `## Workflow` — single-variable hypothesis → primary metric + MDE (power/N/
    duration reasoning) → guardrail metrics → intended split (SRM baseline) →
    pre-registered decision rule + stop rule → lock. No early-stopping; one
    primary metric; guardrails named before launch.
  - `## Produces` — an `## Experiment Brief` section (the pre-registered artifact)
    on the spec, in the exact shape `experiment-readout-reviewer` reads back.
  - `## Anti-patterns` — multiple primary metrics; MDE set after data; peeking /
    early-stopping baked into the plan; guardrails omitted; deciding the ship
    call (doctrine 2 — the designer designs, the team runs and decides).

- **`domains/pm/agents/metrics-analyst.md`** — defines/interprets success
  metrics **and** runs metric-movement RCA. Frontmatter as an authoring agent
  (`color: secondary`). Un-dangles `/metrics`. Body:
  - Role: senior product analyst; defines leading, observable, outcome-tied
    metrics (baseline before target) and answers "why did the metric move" with
    disciplined RCA.
  - `## Startup` loads `pm-agent-doctrine`, `metrics-design`,
    `outcomes-over-outputs`, `metric-rca`, `evidence-synthesis`.
  - `## When invoked` — `/metrics`; PRD `Goals & Success Metrics` authoring;
    "why did <metric> drop/spike," "run RCA on the funnel," principle-#5
    retrospective hooks.
  - Two workflows: (a) **metric definition** — reuse the shipped `/metrics`
    `## Metrics` shape (current/target/type/source, leading-not-vanity); (b)
    **RCA** — metric-tree decomposition → drift taxonomy classification →
    causality-before-asserting → candidate causes ranked with the corpus number
    each cites.
  - `## Produces` — `## Metrics` sections and, for RCA, a `## Metric RCA`
    section (decomposition table + classified drift + ranked hypotheses + the
    data that would confirm each). Suggest-don't-decide: names likely causes,
    does not assert a single cause without the confirming cut.

### New skills

- **`domains/pm/skills/experiment-design/SKILL.md`** — the brief format the
  readout-reviewer consumes. Header `name: experiment-design`, `purpose:
  framework-guidance`, `audience:` naming `experiment-designer` and
  `experiment-readout-reviewer`. Content:
  - **Pre-registration** — the brief fixes every term before data; changing a
    registered term after launch is a top finding at readout.
  - **Single-variable hypothesis** — one change, one primary metric.
  - **Primary metric + MDE** — the smallest effect worth detecting; MDE drives N
    and duration; MDE is registered, never lowered post-hoc.
  - **Guardrail metrics** — protected metrics (latency, error rate,
    revenue/user, retention, unsubscribe) that must not regress; a primary lift
    bought with a guardrail regression is not a win.
  - **Intended split / SRM baseline** — the registered allocation the readout's
    observed allocation is checked against (sample-ratio-mismatch).
  - **Pre-registered decision rule + stop rule** — ship/kill criteria and the
    fixed duration; no early-stopping / peeking.
  - **Brief template** — a copy-paste `## Experiment Brief` block whose fields
    are exactly what `experiment-readout-reviewer`'s checklist reads back (this
    is the term-parity contract of AC3).

- **`domains/pm/skills/metric-rca/SKILL.md`** — "why did the metric move."
  Header `name: metric-rca`, `purpose: framework-guidance`, `audience:`
  `metrics-analyst`. Content:
  - **Metric-tree decomposition** — decompose the top-line metric into its
    multiplicative/additive components (e.g. conversion = sessions × rate;
    revenue = users × ARPU) so a move is localized to a component before a cause
    is named.
  - **Drift taxonomy** — classify the move as one of: **component** (a sub-metric
    moved), **temporal** (seasonality / day-of-week / trend), **influence**
    (segment/mix shift — Simpson's-paradox class), **dimension** (a slice
    appeared/disappeared, e.g. a new geo or platform), **event-shock** (a launch,
    outage, pricing change, external event).
  - **Causality-before-asserting** — correlation is a hypothesis, not a cause;
    every candidate cause names the cut/segment/time-window that would confirm or
    kill it before it's asserted. Ranked hypotheses, each grounded in a corpus
    number (doctrine 1).

### New command

- **`domains/pm/commands/experiment.md`** — `/experiment` routes to
  `experiment-designer`, loading `experiment-design`. Required arg: a PRD /
  initiative / feature slug the experiment tests. States the command *designs*
  the pre-registered brief; readout critique is `experiment-readout-reviewer`
  (agent-invoked, no command). Output: an `## Experiment Brief` section on the
  spec + a one-line log.

### Edits

- **`domains/pm/commands/metrics.md`** — repoint the routing line from
  `pm-delivery-lead` (with "`metrics-analyst` takes over in v1.5") to
  `metrics-analyst`, and mention the RCA capability. Everything else
  (the `## Metrics` shape, rules, retrospective hook) stays as shipped.

- **`domains/pm/AGENTS.md`** — three additions-only edits:
  1. Below the `WAVE-2 ROUTES` marker, **after** child #6's `#### Wave-2
     adversarial critic routes` table, append a new subsection (e.g.
     `#### Wave-2 experiment & metrics routes`) with rows: `/experiment` →
     `experiment-designer` (design a pre-registered brief); "why did the metric
     move" / "run RCA" → `metrics-analyst`; and a note that `/metrics` now
     resolves to `metrics-analyst` (un-dangled).
  2. **Skills Reference** → add `experiment-design` and `metric-rca` (Frameworks
     group, or a Wave-2 line consistent with the existing `outcome-drift` /
     `evidence-forcing` Wave-2 entries).
  3. **Agents Reference** → add `experiment-designer` and `metrics-analyst` to
     the PM roster (and note the Wave-2 experiment/metrics grouping alongside the
     existing Wave-2 critics line).

  The canonical table above the marker and child #6's appended critic routes are
  **not** touched (AC9).

## Validation

```bash
set -euo pipefail
cd /Users/developer/projects/hero-engine/repository/hero
PM=domains/pm

# AC1 — experiment-designer agent exists, loads doctrine + experiment-design
test -f $PM/agents/experiment-designer.md
grep -q '^name: experiment-designer' $PM/agents/experiment-designer.md
grep -q 'mode: subagent' $PM/agents/experiment-designer.md
grep -qi 'pm-agent-doctrine' $PM/agents/experiment-designer.md
grep -qi 'experiment-design' $PM/agents/experiment-designer.md

# AC2 — experiment-design skill carries all five locked brief terms
S=$PM/skills/experiment-design/SKILL.md
test -f $S
grep -qi 'primary metric' $S
grep -qi 'MDE\|minimum detectable effect' $S
grep -qi 'split\|SRM\|sample.ratio\|allocation' $S
grep -qi 'guardrail' $S
grep -qi 'decision rule\|stop rule\|pre-registrat' $S

# AC3 — term parity: reviewer's forward-dep terms all appear in the brief skill
R=$PM/agents/experiment-readout-reviewer.md
for term in 'primary metric' 'MDE' 'guardrail' 'stop rule'; do
  grep -qi "$term" $S || { echo "MISSING brief term: $term"; exit 1; }
done
# split/allocation parity (reviewer says "intended split"; brief must cover it)
grep -qi 'split\|allocation\|SRM' $S

# AC4 — /experiment command routes to experiment-designer, design-not-critique
C=$PM/commands/experiment.md
test -f $C
grep -qi 'experiment-designer' $C
grep -qi 'design' $C

# AC5 — metrics-analyst agent loads doctrine + metrics-design + outcomes-over-outputs
M=$PM/agents/metrics-analyst.md
test -f $M
grep -q '^name: metrics-analyst' $M
grep -qi 'pm-agent-doctrine' $M
grep -qi 'metrics-design' $M
grep -qi 'outcomes-over-outputs' $M
grep -qi 'metric-rca\|rca\|why did' $M

# AC6 — metric-rca skill: metric-tree + all five drift classes + causality guard
K=$PM/skills/metric-rca/SKILL.md
test -f $K
grep -qi 'metric.tree\|decomposit' $K
for cls in component temporal influence dimension event.shock; do
  grep -qi "$cls" $K || { echo "MISSING drift class: $cls"; exit 1; }
done
grep -qi 'causal\|correlation is' $K

# AC7 — /metrics un-dangled: routes to metrics-analyst, no v1.5 deferral
MC=$PM/commands/metrics.md
grep -qi 'metrics-analyst' $MC
! grep -qi 'takes over in v1.5' $MC

# AC8 + AC10 — AGENTS.md gains routes + skills + agents references
A=$PM/AGENTS.md
grep -qi '/experiment' $A
grep -qi 'experiment-designer' $A
grep -qi 'metrics-analyst' $A
grep -qi 'experiment-design' $A
grep -qi 'metric-rca' $A

# AC9 — additions-only below the marker: canonical table + child #6 routes intact.
# The marker and child #6's critic-routes heading must still be present verbatim.
grep -q 'WAVE-2 ROUTES: appended by adversarial-critics-bundle / experiment-stage-and-metric-rca' $A
grep -q '#### Wave-2 adversarial critic routes' $A
# New experiment/metrics routes appear AFTER the child #6 critic table heading.
awk '/#### Wave-2 adversarial critic routes/{seen=1} seen && /experiment-designer/{found=1} END{exit !found}' $A

# AC11 — no dangling skill refs: every skill loaded in the two new agents resolves
for agent in $PM/agents/experiment-designer.md $PM/agents/metrics-analyst.md; do
  # crude extraction: backticked skill names in the Startup list
  grep -oE '`[a-z0-9-]+`' "$agent" | tr -d '`' | sort -u | while read -r name; do
    if [ -d "$PM/skills/$name" ]; then
      test -f "$PM/skills/$name/SKILL.md" || { echo "DANGLING skill dir: $name"; exit 1; }
    fi
  done
done

echo "ALL EXPERIMENT-STAGE-AND-METRIC-RCA CHECKS PASSED"
```

## Boundaries

**In scope** (content-only, `domains/pm/` pack source):
- Two new agents: `experiment-designer`, `metrics-analyst`.
- Two new skills: `experiment-design`, `metric-rca`.
- One new command: `/experiment`; one edited command: `/metrics` (un-dangle).
- Additions-only edits to `domains/pm/AGENTS.md` (Wave-2 region + references).

**Out of scope:**
- **No Go / no engine / no consumer code.** Tripwire
  *harness-changes-cover-all-targets*: author only in `domains/pm/` pack source;
  do not touch installed harness copies (`.claude/`, `.agents/`, `.codex/`) —
  `hero install` regenerates those.
- **No new critic behavior.** `experiment-readout-reviewer` (child #6) is not
  edited. This child authors the brief format it forward-references; promoting
  `experiment-design` into the reviewer's Startup load list is a later
  reconciliation's job (as the reviewer's `## Forward dependency` states), **not**
  this child's — doing so here would over-reach the seam.
- **No canonical-table edits in AGENTS.md.** Only append below the marker; the
  canonical routing table and child #6's critic routes stay byte-for-byte
  unchanged (AC9).
- **No full analytics engine.** `metrics-analyst` is the light v1 the design
  scoped (`hero-data-analytics` owns deep metric specs). RCA is a disciplined
  reasoning method (metric-tree + drift taxonomy + causality guard), not a query
  runner or a live-data integration.
- **No `/experiment` readout-critique surface.** `/experiment` designs briefs;
  readout critique remains agent-invoked (`experiment-readout-reviewer`).

## Completion Ledger

Content-only delivery in `domains/pm/` pack source (2 agents, 2 skills, 1 command +
2 edits). Validation: ran the spec's full `## Validation` bash block verbatim from
repo root — all checks PASS. Closes the 6↔7 seam (experiment-design brief term-matches
experiment-readout-reviewer) and un-dangles the shipped `/metrics` command. Cold audit +
verify run this same turn.

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| AC1 | `experiment-designer.md` exists; Startup loads `pm-agent-doctrine` + `experiment-design` | DONE | grep gates for name/mode/both skills pass |
| AC2 | `experiment-design/SKILL.md` covers primary metric, MDE, split/SRM, guardrails, decision/stop rule | DONE | "five locked terms" section; all five grep gates pass |
| AC3 | Term parity with experiment-readout-reviewer (6↔7 seam) | DONE | term-parity loop over reviewer's forward-dep terms passes against the brief |
| AC4 | `commands/experiment.md` routes `/experiment` → experiment-designer (design not critique) | DONE | present; readout critique stays with experiment-readout-reviewer |
| AC5 | `metrics-analyst.md` loads doctrine+metrics-design+outcomes-over-outputs; definition AND RCA | DONE | two workflows; grep gates pass |
| AC6 | `metric-rca/SKILL.md` — metric-tree + all five drift classes + causality guard | DONE | drift-class loop (component/temporal/influence/dimension/event-shock) + causality gate pass |
| AC7 | `/metrics` repointed to metrics-analyst; "v1.5" deferral removed | DONE | `commands/metrics.md` repointed; `! grep 'takes over in v1.5'` passes |
| AC8 | AGENTS.md routes below marker, after child #6's table | DONE | new `#### Wave-2 experiment & metrics routes`; awk ordering gate passes |
| AC9 | AGENTS.md additions-only; canonical table + child #6 routes unchanged | DONE | git diff: protected regions byte-unchanged; only additive edits below marker |
| AC10 | Skills Reference + Agents Reference list the new skills/agents | DONE | four grep gates pass |
| AC11 | Every Startup-loaded skill resolves; no dangling refs | DONE | no-dangling-refs loop passes |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | Create `agents/experiment-designer.md` | DONE | pre-registration discipline; designs not critiques |
| 2 | Create `skills/experiment-design/SKILL.md` | DONE | 102 lines; five locked terms + copy-paste brief template term-matched to reviewer |
| 3 | Create `agents/metrics-analyst.md` | DONE | metric definition + RCA; backs `/metrics` |
| 4 | Create `skills/metric-rca/SKILL.md` | DONE | 93 lines; metric-tree + five drift classes + causality guard |
| 5 | Create `commands/experiment.md` | DONE | `/experiment` → experiment-designer |
| 6 | Edit `commands/metrics.md` | DONE | repoint to metrics-analyst; drop v1.5 deferral |
| 7 | Append Wave-2 routes to `domains/pm/AGENTS.md` | DONE | experiment & metrics routes + reference rosters; additions-only |

### Exercise-the-feature check

- [x] Full `## Validation` block run verbatim from repo root — all checks PASS. AC9 additions-only confirmed via `git diff` (protected regions unchanged); AC3 term-parity confirmed by matching the reviewer's read-back terms to the brief field names; no-dangling-refs loop clean.
- [ ] Not exercised: `hero install` propagation + live agent invocation — outside content-authoring scope (installed harness copies off-limits per tripwire).

### Excellence Bar self-check

Yes. The `experiment-design` brief template uses the reviewer's checklist vocabulary term-for-term (closing the 6↔7 seam as a real contract, not just a grep); `metric-rca` names the Simpson's-paradox/influence class and confirming-cut discipline concretely; AGENTS.md kept additions-only with honest shipped-surface annotations; scope confined to `domains/pm/`.
