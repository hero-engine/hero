# Delivery audit — experiment-stage-and-metric-rca

**Audited:** working tree vs `HEAD` (new files untracked; `git diff HEAD` for edits)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC1 — `domains/pm/agents/experiment-designer.md:2-6` (`name: experiment-designer`, `mode: subagent`); Startup loads `pm-agent-doctrine` + `experiment-design` (`:26-27`).
- [✓] AC2 — `domains/pm/skills/experiment-design/SKILL.md:35-57` "The five locked terms": primary metric, MDE, intended split (SRM), guardrails, decision/stop rule.
- [✓] AC3 (6↔7 seam) — reviewer reads back registered primary metric, MDE, guardrails, intended split, stop rule (`experiment-readout-reviewer.md:51,55`); the brief declares all five (`experiment-design/SKILL.md:39-57`, template `:67-80`). Every reviewer read-back term is a registered brief field. No seam gap.
- [✓] AC4 — `domains/pm/commands/experiment.md:4-6` routes to `experiment-designer`, states it designs (not critiques).
- [✓] AC5 — `domains/pm/agents/metrics-analyst.md:2,25-30` loads doctrine + metrics-design + outcomes-over-outputs + metric-rca; two workflows (definition `:42-43`, RCA `:45-50`).
- [✓] AC6 — `domains/pm/skills/metric-rca/SKILL.md`: metric-tree decomposition (`:20-45`), all five drift classes with confirming cuts (`:51-55`), causality-before-asserting guard (`:59-65`).
- [✓] AC7 — `domains/pm/commands/metrics.md:4` repointed to `metrics-analyst`; "takes over in v1.5" deferral removed (grep confirms absent).
- [✓] AC8 — AGENTS.md `#### Wave-2 experiment & metrics routes` (line 82) appended after child #6's table (line 67), below marker (line 62).
- [✓] AC9 — additions-only: diff is 18 insertions + 1 deletion, the deletion being the reference-roster Commands line gaining `/experiment` (permitted). Canonical table and child #6's critic-routes table byte-unchanged.
- [✓] AC10 — Skills Reference adds `experiment-design` + `metric-rca`; Agents Reference adds `experiment-designer` + `metrics-analyst` (AGENTS.md diff).
- [✓] AC11 — no-dangling-refs loop clean; every Startup skill resolves on disk.

## Changes
- [✓] Create `agents/experiment-designer.md` — pre-registration authoring agent, designs not critiques.
- [✓] Create `skills/experiment-design/SKILL.md` — five locked terms + copy-paste brief template term-matched to reviewer checklist.
- [✓] Create `agents/metrics-analyst.md` — metric definition + RCA; backs `/metrics`.
- [✓] Create `skills/metric-rca/SKILL.md` — metric-tree + five drift classes + causality guard.
- [✓] Create `commands/experiment.md` — `/experiment` → experiment-designer.
- [✓] Edit `commands/metrics.md` — repointed to metrics-analyst, v1.5 deferral dropped.
- [✓] Append Wave-2 routes to `AGENTS.md` — experiment/metrics routes + reference rosters, additions-only.

## Open items
None.

## Audit notes
- Full `## Validation` block ran verbatim from repo root: `ALL EXPERIMENT-STAGE-AND-METRIC-RCA CHECKS PASSED`.
- Substance is genuine, not grep-bait: `experiment-design` carries real pre-registration discipline (single-variable, MDE-drives-N/duration, guardrails-before-launch, no-peeking, lock-and-restart). `metric-rca` is a real metric-tree + five-class drift taxonomy (each class with its own confirming cut, Simpson's-paradox named) plus a causality guard with a concrete saw-it-vs-didn't example. `metrics-analyst` covers both definition and RCA.
- Scope clean: only `domains/pm/` + the initiative spec dir + Hero projection/handoff files (`.hero/NEXT.md`, `QUEUE.md`, `events.log`, `drive/`). No Go, no installed harness copies (`.claude/`/`.agents/`/`.codex/`), no sibling-child files. The `experiment-stage-and-metric-rca.md` → `experiment-stage-and-metric-rca/` change is a flat-file-to-dir move of this spec, expected.
