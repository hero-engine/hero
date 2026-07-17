---
description: Produce a tiered launch plan + phased checklist for a roadmap-item or PRD — detect the launch tier, then emit the five-phase checklist scoped to that tier. Routes to stakeholder-communicator.
---
Route this launch request to `stakeholder-communicator` and produce a **launch plan + checklist** for the target roadmap-item / PRD.

`/launch <target>` sizes the launch by impact, then emits the go-to-market motion the tier warrants — not a one-size campaign. It loads `launch-gtm-tiering` for the tier rubric and the five-phase checklist.

## Pre-flight: identify the target

Read the launch target from `$ARGUMENTS`.

- A roadmap-item / initiative / PRD slug → plan the launch for that item.
- No argument → ask which shipped-or-shipping item to plan a launch for. Do not guess.
- Read the item's bet, target segment, and delivery state first — the tier is grounded in real blast-radius / revenue / competitive signal, not gut feel (`pm-agent-doctrine`).

## Tier detection

Using `launch-gtm-tiering`, assign the launch to **Tier 1** (major / company-moving — full GTM motion), **Tier 2** (standard feature — positioning + enablement + announcement), or **Tier 3** (minor / incremental — release note + in-product surface). Size against the rubric dimensions — blast radius, revenue/segment impact, net-new vs. incremental, competitive stakes — and name the dimension that sets the tier. The highest-hitting dimension pulls the tier up. The tier is a **recommendation** the PM confirms, not an auto-decision.

## Emit the phased checklist

Emit the five-phase checklist — `alignment → positioning → enablement → launch → post-launch` — **scoped to the detected tier** (Tier 3 collapses phases; Tier 1 runs them all). For each phase that runs, list its concrete artifacts / gates and the owner.

## Routing

Route to **`stakeholder-communicator`** as the default owner — it owns the announcement and the exec/customer cut (loads `launch-gtm-tiering`). Per phase, name a sensible owner where it differs (positioning → `stakeholder-communicator` with `positioning-canvas`; post-launch metric → `metrics-analyst`). **Note the route** in the output so the PM sees who owns what.

## Output

- The detected **tier** + the rubric dimension that set it.
- The **five-phase checklist**, scoped to the tier, with artifacts/gates and owner per phase.
- The **route** (default `stakeholder-communicator`, plus per-phase owners).
- Report-only: this is a launch *plan*, human-owned; it does not schedule or fire anything.

Request: $ARGUMENTS
