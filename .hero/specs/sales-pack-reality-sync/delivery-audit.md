# Delivery audit — sales-pack-reality-sync

**Audited:** `git diff 65e3333 -- domains/sales internal/spectypes` (uncommitted working tree)
**Verdict:** SHIP
**Surface:** noteworthy

Cold audit from artifacts on disk. Ran the phantom sweep, the loader test, and
every reachable CLI/lookup check against the fresh binary
(`scratchpad/hero-sales`) in the temp workspace `/tmp/sales-ws`. All nine ACs
are met with real evidence.

**AC#5 — initial HOLD, now resolved.** The first pass held on AC#5: two files
carried un-cross-referenced deal-staleness numbers, one
(`deal-strategy/SKILL.md`) never opened by the delivery. Those were fixed and
independently re-verified — every deal staleness/risk day-threshold now defers
to `pipeline-management` with zero local numbers (`forecast-analyst.md:87`,
`deal-strategy/SKILL.md:256-257`, `forecast.md:58`, `pipeline.md:23/50/73`), and
the `forecast.md:68` output path was aligned to `.hero/reports/forecasts/` to
match its agent. The only remaining `N days` hits outside `pipeline-management`
are non-deal-staleness families it does not own: battlecard 90-day freshness, a
worked-example table cell ("Stale 21 days"), and the unresponsive-champion
"2 attempts in 7 days" signal. Build clean; no new Go changes since the first
pass (prose only).

The headline-risk deviation (AC#6 path-based lookup) is **sound** and was
verified independently — see Audit notes. Surface stays **noteworthy** to record
the AC#5 hold→fix cycle and the AC#6 deviation; both are things a reader of this
delivery should know.

## Acceptance criteria

- [✓] **AC#1** No phantom CLI under `domains/sales/` — Validation §1 sweep returns **zero hits** over the whole tree. Only surviving `hero search --type` uses are `deal` (a now-registered type). `hero note` appears once (`objection-handling:236`) with no `--type`.
- [✓] **AC#2** `Load("sales")` registers `deal` with 7-state lifecycle, no load error — `TestLoad_SalesOverlay_DealLifecycle` PASS (asserts states `prospect,qualifying,demo,proposal,negotiation,won,lost`, initial `prospect`, terminal `won,lost`). `deal.md` transitions are referentially sound: every `from`/`to`/`initial`/`terminal` is a declared state. `hero status` in `/tmp/sales-ws` exits 0.
- [✓] **AC#3** Every cited `hero …` exits without unknown-command/flag — verified against the binary: `sprint status --week`, `list --type deal --status <comma>`, `list --type deal --stale 14`, `next path`, `search --type deal "<q>"` all exit 0. (Bare `search --type deal` without a query exits 1 with "requires 1 arg" — but that is a usage error, not unknown-command/flag; the sole literal invocation, `buyer-researcher.md:36`, supplies a query. `deal.md:58` is prose, not a runnable line.)
- [✓] **AC#4** No `.hero/hero.json` key read that `config.go` doesn't — Domain Configuration JSON block deleted, replaced with honest prose (`AGENTS.md:227-236`). `qualify.md` straggler repointed from `.hero/hero.json` `qualification.framework` → deal frontmatter. No `.hero/hero.json` reference remains under `domains/sales/`.
- [✓] **AC#5** Staleness/risk numeric thresholds in exactly one file — **met (after fix).** `pipeline-management` is the sole owner of the deal stale-deal table (SKILL.md:175-181) and declares itself so (SKILL.md:172). Every sibling now defers by cross-reference with no local day-number: `forecast-analyst.md:87` ("past the stage's stale threshold (see `pipeline-management`)"), `deal-strategy/SKILL.md:256-257` ("past the stage's stale window (thresholds owned by `pipeline-management`)"), `forecast.md:58`, `forecast-methodology.md`, and `pipeline.md:23/50/73` (rule + output row + `--stale` flag desc). Re-verified by sweep: no `stale|dark|same-stage|no-activity + N days` rule survives outside `pipeline-management`. Remaining `N days` hits are other families it does not own (battlecard freshness; a worked-example cell; the champion-outreach signal).
- [✓] **AC#6** Battlecard/playbook save+lookup agree — met by **path-based** lookup (deviation from spec). Independently verified: `hero search "battlecard rival"` returns "No results found" in `/tmp/sales-ws` (the file exists on disk with those words). Path-based lookup is honest and functional: `ls .hero/knowledge/battlecards/` lists `acme-rival.md`, which is readable. Deviation judged correct — see Audit notes.
- [✓] **AC#7** Battlecard per template NOT surfaced in `hero list` — verified: `acme-rival.md` (template-authored: no `type:` line, titled "Battlecard — Hero vs. RivalCorp") is **absent** from `hero list`, which shows only the `acme-corp` deal.
- [✓] **AC#8** Install render emits no repo-only relative link — `hero install project <tmp> --domain sales --target claude` rendered AGENTS.md + CLAUDE.md: **zero** `](commands/|agents/|skills/|spec-types/` hits, zero config JSON keys, honest Domain Configuration prose present.
- [✓] **AC#9** Every skill's `audience` matches an agent's Required-skills — cross-checked both directions for all 7 skills; every audience maps exactly (`competitive-positioning`→competitive-intel+deal-strategist; `deal-qualification`→qualification-analyst+deal-strategist; `deal-strategy`→deal-strategist; `discovery-questioning`→buyer-researcher; `forecast-methodology`→forecast-analyst; `objection-handling`→deal-strategist; `pipeline-management`→forecast-analyst+/pipeline).

## Changes

- [✓] **1** `deal.yaml → deal.md` — `deal.yaml` `git rm`'d, `deal.md` written in `parseRecord` shape (top-level `type:`+`category: work`, lifecycle with referential integrity, canonical stage-probability defaults). Loader test green.
- [✓] **2** `spec-types/README.md` repoint to `deal.md` — links + closing line updated, lifecycle-states note added.
- [✓] **3–9** AGENTS.md sweep — real CLI surfaces, config JSON deleted, CLI/slash blocks split, relative links stripped (confirmed by install-render grep).
- [✓] **10–14** Agent files — deal-strategist/forecast-analyst/qualification-analyst/buyer-researcher/competitive-intel updated; battlecard template de-typed + retitled (verified in `/tmp/sales-ws`); forecasts → `.hero/reports/forecasts/`. `forecast-analyst.md:87` staleness number now cross-references `pipeline-management` (AC#5 fix).
- [✓] **15** Skills — phantom refs cleared, roster fixed (AC#9 ✓). `deal-strategy/SKILL.md:256-257` (the file omitted from Change 15) now defers its staleness thresholds to `pipeline-management` (AC#5 fix).
- [✓] **16** Commands — real CLI, `/review`→`/qualify`; `pipeline.md` staleness rule/output/flag genericized to `pipeline-management` (AC#5 fix); `forecast.md:68` output path aligned to `.hero/reports/forecasts/` to match `forecast-analyst.md:156`.

## Open items

- **AC#5 (RESOLVED)** — the three sites flagged in the first pass are fixed and re-verified:
  - `forecast-analyst.md:87` — now "past the stage's stale threshold (see `pipeline-management`)"; number removed.
  - `deal-strategy/SKILL.md:256-257` — now "past the stage's stale window (thresholds owned by `pipeline-management`)"; the file (omitted from Change 15) was opened and fixed.
  - `forecast.md:58` + `pipeline.md:23/50/73` — genericized to defer to `pipeline-management`.
  Sweep confirms no deal-staleness day-threshold rule survives outside `pipeline-management`.
- **`deal-qualification/SKILL.md:250`** — `MEDDPICC score < 25 after 3 qualification conversations — canonical thresholds: pipeline-management`. Retains the number alongside its cross-ref. **Not an AC#5 blocker:** this is a MEDDPICC qualify-out score, not a day-based staleness threshold, and the same value is the canonical one in `pipeline-management` (SKILL.md:65) and the `deal.md` lifecycle gate — they agree, so there is no drift. Left as-is (matches the spec's own Change #15 wording, which kept the number with a cross-ref).

## Audit notes

- **AC#6 deviation is the right call.** Independently reproduced the spec's false premise: `hero search "battlecard rival"` returns "No results found" against a battlecard that exists on disk containing those exact words. `nonWorkFlatTypes` excludes knowledge from discovery and `hero ask` ingests only `.hero/knowledge/raw/`, so no content-search surface reaches `.hero/knowledge/battlecards/`. Path-based lookup (list/read the known save dir) is the one mechanism that makes save+lookup genuinely agree with zero engine change — verified functional. Filing the "should `.hero/knowledge/` be content-searchable" question as a separate all-domains follow-up (rather than smuggling an engine change into a content delivery) is the correct scoping. This deviation strengthens the delivery; it is not a gap.
- **Forecast-path inconsistency (RESOLVED).** `forecast.md:68` was aligned to `.hero/reports/forecasts/`, matching `forecast-analyst.md:156`. Command and agent now agree on the output path.
- **Scope is clean.** Only `domains/sales/`, `internal/spectypes/loader_test.go`, and the `deal.yaml → deal.md` move changed (plus `.hero/` handoff-projection housekeeping). No non-test Go change (confirmed again after the AC#5 fixes — prose only; `go build ./...` clean). No pm/chat/other-AGENTS sibling bleed. `go test ./...` = 0 FAIL, exit 0. The loader test asserts states/initial/terminal but not the transition set — acceptable, since AC#2 speaks to the 7-state lifecycle, and referential integrity was verified by inspection.
- **Why SHIP (was HOLD).** The lone blocker was AC#5's "single owner" invariant being empirically false. That is now genuinely true: all deal staleness/risk day-thresholds live only in `pipeline-management`; every sibling defers by cross-reference with no local number. Re-verified by independent sweep. AC#6's path-based deviation is sound. Surface stays `noteworthy` (not `clean`) to preserve the record of the hold→fix cycle and the deviation.
