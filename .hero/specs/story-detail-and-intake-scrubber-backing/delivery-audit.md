# Delivery audit — story-detail-and-intake-scrubber-backing

**Audited:** working tree vs `HEAD` (`git diff HEAD`; NEW files untracked)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1 three files on disk — `domains/pm/agents/dependency-mapper.md`, `domains/pm/agents/duplicate-intake-scrubber.md`, `domains/pm/commands/scrub.md` all `test -f` pass.
- [✓] AC-2 both agents load `pm-agent-doctrine` — `dependency-mapper.md:24`, `duplicate-intake-scrubber.md:24` (first bullet of each `## Startup`).
- [✓] AC-3 dependency-mapper loads `dependency-mapping` + `cross-domain-graph-query` + `risk-surfacing`, read-only — `dependency-mapper.md:25-27`; `edit: deny` at `:8`; workflow walks graph and reports only (`:36-61`, "You do not mutate graph state" `:61`).
- [✓] AC-4 duplicate-intake-scrubber loads `duplicate-detection`, batch/cluster, report-only no auto-merge, differentiates from `duplicate-detector` — skill at `:25`; `edit: deny` at `:8`; "Report-only / no auto-merge (hard rule)" `:55-57`; dedicated comparison table `:31-36` + structural blind-spots prose `:38-40` citing the "only at intake write-time" anti-pattern.
- [✓] AC-5 scrub.md append-only concern-dispatch region — `<!-- SCRUB CONCERNS ... -->` marker `scrub.md:15-17` naming intake (this child) + roadmap/stories (#11); intake block `:19-23` sits above an explicit `<!-- #11 APPEND POINT -->` marker `:25-28`. #11 appends below line 28 without touching intake. Structure genuinely allows append-only.
- [✓] AC-6 `/scrub intake` routes to `duplicate-intake-scrubber`, dispatch documented — `### Concern: intake` → `duplicate-intake-scrubber` (`scrub.md:19-23`); `## Dispatch` section `:30-32`; `$ARGUMENTS` passthrough `:34`.
- [✓] AC-7 additions-only below WAVE-2 marker, canonical table + prior subsections byte-unchanged — `git diff --unified=0` shows zero removed lines; marker intact at `AGENTS.md:62`; new `#### Wave-1 backing routes` subsection inserted after the Wave-2 PRD-Editor/comms subsection, before "When routing".
- [✓] AC-8 both agents in Agents Reference, `/scrub` in Commands Reference — new "PM Wave-1 Story-Detail / Intake backing" bullet (Agents Reference) and "PM Wave-1 backing" `/scrub` bullet (Commands Reference); pure insertions in the diff.
- [✓] AC-9 no dangling refs — all 5 loaded skills (`pm-agent-doctrine`, `dependency-mapping`, `cross-domain-graph-query`, `risk-surfacing`, `duplicate-detection`) resolve to `domains/pm/skills/<name>/SKILL.md`; all routed agents resolve on disk. `stale-roadmap-scrubber`/`ambiguous-story-scrubber` appear only inside the #11 marker comment as future work — correctly not required to resolve yet.

## Changes
- [✓] 1 Create `dependency-mapper.md` — frontmatter mirrors `duplicate-detector.md` (mode/temperature/color/permission identical); forward+backward walk with `hero_why`, cross-domain traversal, hard/soft/coupling classification, transitive chains, "blocker in progress, ETA", risk-surfacing framing.
- [✓] 2 Create `duplicate-intake-scrubber.md` — frontmatter mirrors `duplicate-detector.md`; batch/cluster workflow, overlap-signal ladder, no-auto-merge hard rule, required differentiation table vs. write-time detector.
- [✓] 3 Create `scrub.md` — `description:`-only frontmatter (matches `triage.md`); pre-flight concern read; two markers delimit the append-only region; intake routes to scrubber; report-only doctrine note; `$ARGUMENTS` passthrough; roadmap/stories not pre-authored.
- [✓] 4 Append to `AGENTS.md` — one new Wave-2-region routing subsection + one Agents Reference bullet + one Commands Reference bullet; all four hunks additive, no removed lines, prior subsections intact.

## Substance judgment
- **duplicate-intake-scrubber vs. duplicate-detector: genuinely different, not a near-dup.** Shipped detector is a single-item write-time check (one item vs. corpus, recall-first, per-write-site triggers, loads `duplicate-detection`+`cross-domain-graph-query`+`evidence-synthesis`). The scrubber is a batch/cluster sweep over a window (N items clustered against each other, `/scrub intake` + "Cluster recent" + cron triggers, loads `pm-agent-doctrine`+`duplicate-detection`). Mechanism, trigger set, and framing are distinct; overlap is only the shared skill + no-auto-merge doctrine, which is correct.
- **dependency-mapper is real.** Forward walk (outbound blocks + parent), backward walk (`hero_why`), cross-domain graph traversal into engineering features, three-way hard/soft/coupling classification with explicit "don't report coupling as dependency", transitive-chain-to-terminal-state, ETA on `delivering` blockers.
- Both agents are `edit: deny` report-only in frontmatter and reinforced in body.

## Audit notes
- Working tree also contains Hero machinery outside the deliverable scope: `.hero/NEXT.md`, `.hero/QUEUE.md`, `.hero/events.log` (M), the spec-stub → directory migration (`story-detail-and-intake-scrubber-backing.md` D + new dir), and `.hero/drive/` state (untracked). These are auto-projected handoff/drive files from the /drive run, not hand edits — expected, not scope drift. All *deliverable* content is confined to `domains/pm/`.
- The spec's full `## Validation` bash block was re-run verbatim from repo root: every line printed `OK`, zero `MISSING` / `DANGLING` / `REVIEW`.
