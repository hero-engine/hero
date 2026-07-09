# Findings — surface pass 3c (SKILLS)

Scope: 94 inventory rows with surface=skill = 92 `SKILL.md` files (core 14, engineering 52,
pm 19, sales 7) + 2 pack READMEs (pm, sales — freshness-only). All 94 read/verified at
git `bc86ad9`. CLI references verified against the installed `hero` binary; engine behavior
verified against `internal/` source. Findings only — no edits made to `core/` or `domains/`.

Severity counts: **S1 × 10 · S2 × 19 · S3 × 16** (45 findings; corrected after
cold-audit re-verification — an executive-report S1 claiming `hero sprint
status --week` doesn't exist was **withdrawn**: the flag is real,
`internal/cli/pulse.go:33`, verified in live `--help`.)

---

## (a) Coverage table

| # | File | Verdict |
|---|------|---------|
| 1 | core/skills/agent-reliability/SKILL.md | flagged |
| 2 | core/skills/auto-knowledge-capture/SKILL.md | flagged |
| 3 | core/skills/context-injection/SKILL.md | flagged |
| 4 | core/skills/convention-writing/SKILL.md | flagged |
| 5 | core/skills/documentation-practices/SKILL.md | flagged |
| 6 | core/skills/executive-report/SKILL.md | flagged |
| 7 | core/skills/explainer-format/SKILL.md | flagged |
| 8 | core/skills/knowledge-flywheel/SKILL.md | flagged |
| 9 | core/skills/next-handoff-emit/SKILL.md | flagged |
| 10 | core/skills/next-md/SKILL.md | flagged |
| 11 | core/skills/note-capture/SKILL.md | flagged |
| 12 | core/skills/nudge-awareness/SKILL.md | flagged |
| 13 | core/skills/project-context-generation/SKILL.md | flagged |
| 14 | core/skills/spec-format/SKILL.md | flagged |
| 15 | domains/engineering/skills/agent-reliability/SKILL.md | flagged |
| 16 | domains/engineering/skills/api-design-and-contracts/SKILL.md | clean |
| 17 | domains/engineering/skills/architecture-principles/SKILL.md | clean |
| 18 | domains/engineering/skills/auto-knowledge-capture/SKILL.md | flagged |
| 19 | domains/engineering/skills/challenge-diagnosis/SKILL.md | flagged |
| 20 | domains/engineering/skills/code-scrub/SKILL.md | flagged |
| 21 | domains/engineering/skills/context-injection/SKILL.md | flagged |
| 22 | domains/engineering/skills/convention-writing/SKILL.md | flagged |
| 23 | domains/engineering/skills/cross-repo-peering/SKILL.md | flagged |
| 24 | domains/engineering/skills/database-stack/SKILL.md | clean |
| 25 | domains/engineering/skills/debugging-investigation/SKILL.md | flagged |
| 26 | domains/engineering/skills/deep-code-enrichment/SKILL.md | clean |
| 27 | domains/engineering/skills/delivery-audit/SKILL.md | flagged |
| 28 | domains/engineering/skills/dependency-analysis/SKILL.md | flagged |
| 29 | domains/engineering/skills/devops-and-operations/SKILL.md | clean |
| 30 | domains/engineering/skills/documentation-practices/SKILL.md | flagged |
| 31 | domains/engineering/skills/drive/SKILL.md | flagged |
| 32 | domains/engineering/skills/executive-report/SKILL.md | flagged |
| 33 | domains/engineering/skills/go-stack/SKILL.md | clean |
| 34 | domains/engineering/skills/greenfield-scaffolding/SKILL.md | clean |
| 35 | domains/engineering/skills/groovy-stack/SKILL.md | clean |
| 36 | domains/engineering/skills/html-mockup-generation/SKILL.md | flagged |
| 37 | domains/engineering/skills/implementation-principles/SKILL.md | flagged |
| 38 | domains/engineering/skills/incident-response/SKILL.md | clean |
| 39 | domains/engineering/skills/integration-boundaries/SKILL.md | clean |
| 40 | domains/engineering/skills/issue-list-report/SKILL.md | clean |
| 41 | domains/engineering/skills/java-stack/SKILL.md | clean |
| 42 | domains/engineering/skills/javascript-stack/SKILL.md | clean |
| 43 | domains/engineering/skills/kickoff-prompt/SKILL.md | flagged |
| 44 | domains/engineering/skills/knowledge-flywheel/SKILL.md | flagged |
| 45 | domains/engineering/skills/migration-safety/SKILL.md | clean |
| 46 | domains/engineering/skills/next-handoff-emit/SKILL.md | flagged |
| 47 | domains/engineering/skills/next-md/SKILL.md | flagged |
| 48 | domains/engineering/skills/note-capture/SKILL.md | flagged |
| 49 | domains/engineering/skills/nudge-awareness/SKILL.md | flagged |
| 50 | domains/engineering/skills/performance-optimization/SKILL.md | clean |
| 51 | domains/engineering/skills/pr-review/SKILL.md | clean |
| 52 | domains/engineering/skills/project-context-generation/SKILL.md | flagged |
| 53 | domains/engineering/skills/python-stack/SKILL.md | clean |
| 54 | domains/engineering/skills/react-stack/SKILL.md | clean |
| 55 | domains/engineering/skills/release-and-deployment/SKILL.md | clean |
| 56 | domains/engineering/skills/roadmap-review/SKILL.md | flagged |
| 57 | domains/engineering/skills/root-cause-classification/SKILL.md | flagged |
| 58 | domains/engineering/skills/rust-stack/SKILL.md | clean |
| 59 | domains/engineering/skills/security-review/SKILL.md | clean |
| 60 | domains/engineering/skills/spec-composition/SKILL.md | flagged |
| 61 | domains/engineering/skills/spec-format/SKILL.md | flagged |
| 62 | domains/engineering/skills/spec-sizing/SKILL.md | flagged |
| 63 | domains/engineering/skills/stack-detection/SKILL.md | clean |
| 64 | domains/engineering/skills/swiftui-mockup-renderer/SKILL.md | flagged |
| 65 | domains/engineering/skills/test-strategy/SKILL.md | clean |
| 66 | domains/engineering/skills/testing-and-validation/SKILL.md | clean |
| 67 | domains/pm/skills/README.md | doc-only |
| 68 | domains/pm/skills/acceptance-criteria-ears/SKILL.md | flagged |
| 69 | domains/pm/skills/continuous-discovery-cadence/SKILL.md | flagged |
| 70 | domains/pm/skills/cross-domain-graph-query/SKILL.md | flagged |
| 71 | domains/pm/skills/cycle-planning/SKILL.md | flagged |
| 72 | domains/pm/skills/dependency-mapping/SKILL.md | flagged |
| 73 | domains/pm/skills/duplicate-detection/SKILL.md | flagged |
| 74 | domains/pm/skills/evidence-synthesis/SKILL.md | flagged |
| 75 | domains/pm/skills/handoff-protocol/SKILL.md | flagged |
| 76 | domains/pm/skills/intake-classification/SKILL.md | flagged |
| 77 | domains/pm/skills/metrics-design/SKILL.md | flagged |
| 78 | domains/pm/skills/opportunity-solution-trees-torres/SKILL.md | flagged |
| 79 | domains/pm/skills/pitch-writing-shape-up/SKILL.md | flagged |
| 80 | domains/pm/skills/pm-preset-detection/SKILL.md | flagged |
| 81 | domains/pm/skills/prd-anti-patterns/SKILL.md | flagged |
| 82 | domains/pm/skills/prd-structure/SKILL.md | flagged |
| 83 | domains/pm/skills/prioritization-frameworks/SKILL.md | clean |
| 84 | domains/pm/skills/roadmap-framing/SKILL.md | flagged |
| 85 | domains/pm/skills/sprint-planning/SKILL.md | flagged |
| 86 | domains/pm/skills/story-writing-invest/SKILL.md | flagged |
| 87 | domains/sales/skills/README.md | doc-only |
| 88 | domains/sales/skills/competitive-positioning/SKILL.md | flagged |
| 89 | domains/sales/skills/deal-qualification/SKILL.md | flagged |
| 90 | domains/sales/skills/deal-strategy/SKILL.md | flagged |
| 91 | domains/sales/skills/discovery-questioning/SKILL.md | flagged |
| 92 | domains/sales/skills/forecast-methodology/SKILL.md | flagged |
| 93 | domains/sales/skills/objection-handling/SKILL.md | flagged |
| 94 | domains/sales/skills/pipeline-management/SKILL.md | flagged |

Note: the 13 core↔engineering same-named pairs are marked flagged on both sides because
they are all rows of the dedup/drift matrix (finding S2-1), even where the content itself
is otherwise sound.

---

## (b) Dedup/drift matrix — core/skills ↔ domains/engineering/skills

### Engine precedence (which copy wins)

`hero install` builds the content FS as `hero.OverlayFS(domainFS, hero.CoreFS())` —
domain pack on top, core underneath; "**Domain wins on file-level path conflicts**"
(internal/cli/install.go:188–200; lookup semantics in the repo-root `content.go`
`overlayFS.Open`, top-then-bottom). `internal/install/content.go` reads whatever the
merged FS surfaces. Consequences:

- **Engineering-domain installs (the default)** get the `domains/engineering/skills/` copy
  for all 13 collisions. The core copy is invisible to them.
- **pm / sales / chat installs** get the `core/skills/` copy (those packs don't ship these
  13 names, so the overlay falls through to core).
- For the **8 byte-identical pairs**, the engineering copy is pure dead weight: deleting it
  changes nothing (overlay falls through to the identical core file). For the **5 diverged
  pairs**, each copy is live for a different audience — a two-master problem with no sync
  mechanism.

Fork origin: commit `92c94aa` "chore(domains): sync root → domains/engineering/ for
legacy-fallback parity" duplicated the files; later commits updated whichever copy the
author touched, in both directions.

### The matrix

| Skill | State | Direction / classification | Evidence (one line) |
|---|---|---|---|
| auto-knowledge-capture | identical (md5 b5c89c62) | — | eng copy = dead weight |
| convention-writing | identical (8f67c707) | — | eng copy = dead weight (shared S1 path bug, see S1-1) |
| documentation-practices | identical (266eb238) | — | eng copy = dead weight |
| executive-report | identical (06d18066) | — | eng copy = dead weight |
| knowledge-flywheel | identical (6db39ff5) | — | eng copy = dead weight |
| note-capture | identical (b3ea54de) | — | eng copy = dead weight |
| nudge-awareness | identical (f01c382b) | — | eng copy = dead weight |
| project-context-generation | identical (6648eb69) | — | eng copy = dead weight |
| **agent-reliability** | diverged (621w vs 1182w) | eng ahead — **accidental-drift** (thin intentional layer) | `1457107` ("read-don't-guess") and `1b340d1` (Honesty-about-scope + Persistence sections) landed on eng only; the rules are domain-universal, yet core still carries the old "Distinguish facts from assumptions" bullet. Only the `engineer.md`/Completion-Ledger cross-ref is legitimately engineering-specific. pm/sales installs get the weaker 2-generations-old reliability rules. |
| **context-injection** | diverged (1319w vs 1189w) | **core ahead — accidental-drift, stale copy wins** | `d7fd9e9` (superseded-specs soft-archive) added "### Handling superseded specs" (incl. the `hero supersede <old> --by <new>` instruction and the do-not-follow-superseded rule) to core only; git log for the eng copy shows no commit after `92c94aa`. Since eng shadows core, **engineering installs never see the supersede guidance** — the highest-blast-radius drift casualty. |
| **next-handoff-emit** | diverged (1445w vs 1494w) | eng ahead — **accidental-drift** (partial sync) | `c8bc9ac` and `468974a` landed on both copies, but `0cfe403` (don't write scratch into `.hero/next/<user>.local.md`; it's rebuilt wholesale each checkpoint) landed on eng only. The rule is engine behavior, not engineering-specific — pm/sales installs will hand-write into a file the Stop hook wipes. |
| **next-md** | diverged (1520w vs 1654w) | eng ahead — **accidental-drift** (same commit) | `0cfe403` added the "`.local.md` is machine-state-only / never hand-edit" section + What-not-to-do bullet to eng only; core never got it. Same universal-engine-behavior argument as above. |
| **spec-format** | diverged (2015w vs 2482w) | **bidirectional — accidental-drift both ways** | Eng-only: `4a62fea` (folder-per-spec rationale, flat-spec tolerance, authoritative `slug:`), `9ea7bf2` (`completed_at` row + `hero admin backfill-completed-at`), `29b4716` (Mockups sections). Core-only: `d7fd9e9` (supersede genealogy — `superseded_by` row, "Superseding a spec" section, `hero supersede --scan`). The eng copy — the one engineering installs get — still says `supersedes`: "The superseded spec should have its status set to `superseded`", the exact legacy mechanism core deprecates ("the `status: superseded` enum value is legacy"). pm/sales installs conversely lack folder-per-spec / `slug:` / `completed_at` facts. |

**Bottom line:** 8/13 duplicates are byte-identical dead weight in the engineering pack;
5/13 have accidentally forked, and in two of the five (context-injection, spec-format) the
copy that wins for the default engineering install carries **stale supersede guidance**
that contradicts the current engine (`hero supersede` exists and is the documented path).
No pair shows evidence of deliberate specialization beyond ~1 paragraph of
engineering-specific cross-refs in agent-reliability.

---

## (c) Frontmatter / `compatibility` verdict

**`compatibility:` is inert — nothing consumes it.** The engine parses exactly one piece
of content frontmatter at install time: the agents' `domains:` field
(`readAgentDomainsFrontmatter`, internal/install/content.go:101). `rg -i compatibility`
across `internal/` and `cmd/` finds no reader for the skill key; skills are copied
byte-for-byte to every target. So `compatibility: opencode` is (1) not load-bearing,
(2) **wrong on its face** given six install targets — the same file is stamped
"opencode" and installed into claude/cursor/copilot/codex/generic harnesses, where an
agent reading the frontmatter can only be misled — and (3) inconsistently valued:

- `opencode` — 79 files (v0.8.0 scaffold default)
- `opencode, cursor, claude` (comma string) — core+eng auto-knowledge-capture, note-capture
- YAML list `[opencode, cursor, claude]` — eng code-scrub
- absent — 12 files (see outliers)

**The 14 outliers** (rubric's 80/94): 2 are the pm/sales READMEs (no frontmatter,
correctly). The 12 SKILL.md outliers:

| Files | Present keys | Missing |
|---|---|---|
| domains/sales/skills/{deal-qualification, deal-strategy, discovery-questioning, objection-handling, pipeline-management, forecast-methodology, competitive-positioning} | name, description, metadata | compatibility |
| domains/engineering/skills/drive | name, description, metadata | compatibility |
| domains/engineering/skills/{swiftui-mockup-renderer, html-mockup-generation, root-cause-classification}, core/skills/explainer-format | name, description | compatibility, metadata |

Every post-v0.8.0 skill omits the key — authors already treat it as dead. **Verdict:
stale scaffolding fossil; the consistent fix is to drop `compatibility:` pack-wide (or
define an engine semantic for it), not to add it to the 12 outliers.** Filed as S2-6.

---

## (d) Findings

### S1 (actively misleading)

### [S1] Convention specs pointed at a directory that doesn't exist — core/skills/convention-writing/SKILL.md (+ byte-identical domains/engineering/skills/convention-writing/SKILL.md)
- Surface/dimension: skill / 4 (freshness)
- Evidence: "Convention specs live in `.hero/conventions/{slug}/spec.md`" and "Check for prior art. Search the `.hero/conventions/` directory". Engine truth: `ConventionsDir` = `<hero>/knowledge/conventions` (internal/config/config.go:1686–1688); `hero init` creates `.hero/knowledge/conventions/` (internal/cli/init.go:88); no `.hero/conventions/` exists in an initialized workspace. Neighboring skills (auto-knowledge-capture, greenfield-scaffolding) use the correct `.hero/knowledge/conventions/` path — the pack contradicts itself. Blast radius: all six targets, all four domains (identical copy in both packs), loaded by `/convention` + convention-author.
- Fix shape: s/.hero\/conventions/.hero\/knowledge\/conventions/ in both copies (then dedupe per matrix).

### [S1] Engineering installs get legacy supersede guidance that contradicts the engine — domains/engineering/skills/spec-format/SKILL.md
- Surface/dimension: skill / 4 (freshness) + matrix drift
- Evidence: eng copy's frontmatter table: "`supersedes` … The superseded spec should have its status set to `superseded`." Core copy (post-`d7fd9e9`) says the opposite: "run `hero supersede <old> --by <new>` rather than hand-editing frontmatter … the `status: superseded` enum value is legacy and not required for new work." `hero supersede` exists (verified, incl. `--scan`). Because the eng copy wins the overlay, every default install teaches the deprecated mechanism.
- Fix shape: port `d7fd9e9`'s supersede genealogy into the eng copy (or collapse the pair; see matrix).

### [S1] Engineering installs never see superseded-spec handling — domains/engineering/skills/context-injection/SKILL.md
- Surface/dimension: skill / 4 + matrix drift
- Evidence: core copy has "### Handling superseded specs" (`[SUPERSEDED by <slug>]` markers, do-not-follow rule, `hero supersede` command); eng copy — the one installed — lacks the entire section (diff verified; eng copy untouched since `92c94aa` while core got `d7fd9e9`). Agents on engineering installs will treat superseded past-work entries as live guidance.
- Fix shape: backport the section (or collapse the pair).

### [S1] /drive arming instructions reference a script that doesn't exist outside this repo — domains/engineering/skills/drive/SKILL.md
- Surface/dimension: skill / 4 + 6
- Evidence: "ensure the Stop hook (`scripts/drive/stop-hook.sh` → `hero goal <init> --check`) is armed with `$HERO_DRIVE_INITIATIVE=<init>`." `scripts/drive/stop-hook.sh` exists only in the hero-engine repo; no engine code installs or references it (rg across `internal/` — zero hits), so in any user workspace arming step 4 is unexecutable as written. Additionally "paste the emitted condition into the harness `/goal`" — no `/goal` command ships in any pack (`find core domains -name "goal*"` is empty) and no such standard command exists across the six targets; both `drive.md` command copies repeat the claim.
- Fix shape: have the skill emit/install the hook itself (or point at a `hero`-shipped mechanism), and scope or define what "the harness `/goal`" is per target.

### [S1] Handoff verification steps use nonexistent CLI flags/commands — domains/pm/skills/handoff-protocol/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "`hero queue --owner engineering --status ready`" and "`hero queue --owner pm`" — `hero queue --help` has only `--format, --horizon, --limit, --subproject`. Also "the event log row from `hero event handoff …`" — `hero event` is not a command.
- Fix shape: rewrite with real invocations (`hero list` filters); delete the `hero event` step.

### [S1] `hero search --themes` does not exist — domains/pm/skills/intake-classification/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "Check `hero search --themes` first." — `hero search --help` has `--type/--status/--tag/--list/…`, no `--themes`.
- Fix shape: `hero search --list --tag <theme>` or plain text query.

### [S1] Asserts a phantom skill exists — domains/pm/skills/acceptance-criteria-ears/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "(`acceptance-criteria-gherkin` exists as an alternate AC skill — load it when the team prefers Given/When/Then…)" — no such skill in domains/pm/skills/ or core/skills/; the pm README lists Gherkin AC as P1/v1.5+.
- Fix shape: rescope to "planned, not yet shipped."

### [S1] Contradicts its own unified-model claims with legacy node types — domains/pm/skills/cross-domain-graph-query/SKILL.md
- Surface/dimension: skill / 2 + 4
- Evidence: states "There is no separate `kind: handoff` edge under the unified model" / "There is no longer a separate `feature` type", yet the `hero_why` section instructs "`hero_why feature:X` returns the cross-domain handoff edge and the upstream story / epic / initiative" (three `feature:X` occurrences). A cold agent gets both models in one file.
- Fix shape: rewrite the `hero_why` bullets in `spec:` + `owner_history` terms.

### [S1] `hero note --type knowledge` flag does not exist — domains/sales/skills/objection-handling/SKILL.md
- Surface/dimension: skill / 4
- Evidence: lines 235–237: `hero note "<objection> response" --type knowledge` — `hero note --help` exposes only `--from`.
- Fix shape: drop `--type knowledge`.

### [S1] Claims BANT coverage that isn't in the file — domains/sales/skills/deal-qualification/SKILL.md
- Surface/dimension: skill / 3 + 4
- Evidence: "What this skill covers" bullet "BANT as an alternative for SMB deals" — zero BANT/SMB content in the 299-line body.
- Fix shape: delete the bullet or add the section.

---

### S2 (material waste or drift)

### [S2] 13 same-named skills duplicated across core and engineering; 8 byte-identical, 5 forked — matrix above
- Surface/dimension: skill / 1 + 2 (roster duplication + drift)
- Evidence: dup-map.txt confirmed by md5 + diff + git log (full matrix in section (b)). ~9,900 words of duplicated source; the 8 identical engineering copies are provably dead weight under `OverlayFS(domain, core)`; the 5 forks have produced two S1s above and the three drift rows below.
- Fix shape: delete the 8 identical engineering copies (overlay falls through to core); for the 5 forks, pick a single master per skill — universal content lives in core, engineering-only deltas (engineer.md refs, Mockups) either merge up or become a small eng-only overlay — and add a CI check that same-named core/domain files are either identical or intentionally annotated.

### [S2] agent-reliability drift: pm/sales installs get 2-generations-old reliability rules — core/skills/agent-reliability/SKILL.md
- Surface/dimension: skill / 4 (matrix row)
- Evidence: eng-only commits `1457107` (Ground-before-you-guess grounding check) and `1b340d1` (Two-Reading Rule, no-silent-reclassification, Persistence/true-blocker rules) are domain-universal but never reached core, which still serves every non-engineering domain.
- Fix shape: backport the universal sections to core; keep only `engineer.md`/Completion-Ledger refs in the eng layer.

### [S2] next-md + next-handoff-emit drift: `.local.md` wipe warning missing from the copies non-engineering installs get — core/skills/next-md/SKILL.md, core/skills/next-handoff-emit/SKILL.md
- Surface/dimension: skill / 4 (matrix rows)
- Evidence: `0cfe403` added "never hand-edit `.hero/next/<user>.local.md`; it is rebuilt wholesale on every checkpoint" to the engineering copies only. The wipe is engine behavior on every domain — pm/sales agents following the core copy can lose hand-written notes.
- Fix shape: backport `0cfe403` to both core copies.

### [S2] Phantom skill reference: `end-of-turn-recap` — domains/engineering/skills/kickoff-prompt/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "Composing with related skills … **`end-of-turn-recap`** — governs the closing message of the current turn." No skill of that name exists in any pack (`rg -l end-of-turn-recap core domains` → only this file).
- Fix shape: delete the bullet or point at where the recap rule actually lives.

### [S2] Dogfood leakage: installed skills reference hero-engine-repo-only artifacts — domains/engineering/skills/cross-repo-peering/SKILL.md, roadmap-review/SKILL.md, spec-composition/SKILL.md
- Surface/dimension: skill / 3 + 4
- Evidence: cross-repo-peering: "The full protocol convention lives at `.hero/knowledge/conventions/peering-protocol.md`", "Setup guide: `CROSS-REPO-PEERING.md` (workspace root)", "Decision: `.hero/knowledge/decisions/cross-repo-peering-local-first.md`" — these exist in *this* repo's workspace, not in a user's. roadmap-review: "All three quote the canonical hint from `internal/sizing/ambient.go`", "Sibling spec `roadmap-review-ambient-surfacing` reads these records". spec-composition: "per the midpoint-sum mechanic in `internal/snapshot/rollup.go`", "sibling spec `multi-spec-design-routing`". A cold agent in a target repo cannot resolve any of these. (Same pattern flagged in pm: `internal/vocabulary/`, `internal/spec/`, `core/spec-types/`.)
- Fix shape: describe the behavior without repo paths; move protocol depth into the skill or a shipped doc.

### [S2] `compatibility:` frontmatter is inert, self-contradictory, and wrong for 5 of 6 targets — 80 files pack-wide
- Surface/dimension: skill / 7 (+6)
- Evidence: section (c). No engine consumer; three different value formats; value "opencode" shipped verbatim into claude/cursor/copilot/codex/generic installs.
- Fix shape: drop the key pack-wide (one-line strip per file) or give it real filtering semantics like agents' `domains:`.

### [S2] Stop-hook-dependent handoff machinery presented as universal, but hooks install only for claude + codex — core/skills/next-md/SKILL.md, core+eng next-handoff-emit/SKILL.md
- Surface/dimension: skill / 6 (harness-agnosticism)
- Evidence: next-md: "Machine half … run by a host-tool Stop hook … The harness keeps it fresh on every turn — no agent discipline required." next-handoff-emit: "the end-of-turn Stop checkpoint … automatically records …". Engine installs hooks only via internal/install/claude_hooks.go and codex_hooks.go; cursor/copilot/generic get no Stop hook, so the machine half never refreshes and auto-UserAsk never fires there. Only the transcript_path aside ("Claude Code does") is scoped.
- Fix shape: one scoping paragraph: on hook-less harnesses the agent must run `hero next checkpoint` itself / treat auto-emit as absent.

### [S2] Duplicated drift-handling sections + delivery-plan leakage — domains/engineering/skills/spec-sizing/SKILL.md (2,120w, >p90)
- Surface/dimension: skill / 2 + 4
- Evidence: "## Drift handling" (leaf vs container, bump via `hero size <slug> <tier>`) and "## What to do on drift" (`hero size --check` … bump it) cover the same procedure twice (~230 words). Also "slice 5 wires the adapter side; until then default to 'no tracker' behavior" — an undated internal delivery-plan reference a cold agent can't evaluate; and "Don't re-litigate this choice; the spec rejected the alternatives explicitly" points at a design spec not shipped with the skill.
- Fix shape: merge the two drift sections; replace "slice 5" with the actual current behavior.

### [S2] "Loaded by" rosters contradict actual agent loads — domains/sales (objection-handling, pipeline-management, discovery-questioning, competitive-positioning)
- Surface/dimension: skill / 5 + 4
- Evidence: e.g. objection-handling description "Loaded by deal-strategist and qualification-analyst" but qualification-analyst doesn't list it; its `audience: deal-strategist, buyer-researcher` contradicts the description too (ground truth: agents' Required-skills sections).
- Fix shape: regenerate descriptions/audience from the agents' Required-skills lists.

### [S2] Staleness/risk thresholds duplicated with numeric drift — domains/sales (pipeline-management vs forecast-methodology vs deal-qualification)
- Surface/dimension: skill / 1 + 2
- Evidence: Negotiation staleness 7 days (pipeline-management) vs 10+ days (forecast-methodology); MEDDPICC qualify-out "< 25 after 2+ conversations" vs red-flag "< 25 after 3 conversations"; "$50K ARR single-threaded" rule in 3 files.
- Fix shape: single owner (pipeline-management) for thresholds; others cross-reference.

### [S2] deal-qualification duplicates discovery-questioning's question banks and deal-strategy's champion test — domains/sales/skills/deal-qualification/SKILL.md (1,721w, at p90)
- Surface/dimension: skill / 1 + 2
- Evidence: six per-dimension "Questions that surface it:" lists (~25 lines) replicate discovery-questioning's job; "Signs of a true champion" restates deal-strategy's 5-point champion definition nearly verbatim.
- Fix shape: one exemplar question per dimension + pointers; ~300 words saved.

### [S2] Unified type model described inconsistently across the pm pack — domains/pm/skills/handoff-protocol/SKILL.md, dependency-mapping/SKILL.md
- Surface/dimension: skill / 2 + 4
- Evidence: handoff-protocol "PM authors a feature (`type: feature`)" and dependency-mapping's "story:enable-saml … blocked-by feature:saml-provider" vs story-writing-invest / pm-preset-detection / cross-domain-graph-query: "the canonical type is `spec` (with `kind: feature`)".
- Fix shape: normalize both files to `spec` + `kind` + `owner`.

### [S2] Eleven phantom skill cross-references across 8 pm files — domains/pm pack-wide
- Surface/dimension: skill / 4
- Evidence: `shape-up-cadence`, `hill-chart-reasoning` (pitch-writing-shape-up, cycle-planning), `outcomes-over-outputs`, `risk-surfacing` (roadmap-framing), `stakeholder-communication` (prd-structure), `discovery-interview-design`, `assumption-testing` (continuous-discovery-cadence, opportunity-solution-trees-torres), `iteration-planning`, `capacity-planning` (sprint-planning), `domain-glossary-maintenance` (intake-classification) — none exist in pm or core.
- Fix shape: delete or mark "(P1, not yet shipped)".

### [S2] Phantom agents referenced in bodies and metadata.audience — domains/pm pack-wide
- Surface/dimension: skill / 4 + 7
- Evidence: `pitch-author`, `cycle-planner`, `capacity-planner`, `epic-framer`, `metrics-analyst`, `competitive-analyst`, `dependency-mapper`, `portfolio-curator`, `stale-roadmap-scrubber` do not exist in domains/pm/agents/ (12 shipped agents); `roadmap-reviewer` exists only in the engineering pack.
- Fix shape: map to shipped agents (pm-reviewer, roadmap-curator, prd-author, pm-delivery-lead) or mark P1.

### [S2] Betting table / cooldown / appetite told twice — domains/pm/skills/pitch-writing-shape-up/SKILL.md ↔ cycle-planning/SKILL.md (both >p90)
- Surface/dimension: skill / 1 + 2
- Evidence: near-identical betting-table mechanics, cooldown is/isn't lists, and appetite-vs-estimate contrast in both (~500 of 4,100 combined words are retellings).
- Fix shape: cycle-planning owns the mechanics; pitch-writing keeps author-facing implications + cross-ref.

### [S2] Pitch quality bars tripled across prd-structure / pitch-writing / prd-anti-patterns — domains/pm/skills/prd-structure/SKILL.md (1,731w, >p90)
- Surface/dimension: skill / 2
- Evidence: identical Appetite/Rabbit-Holes/No-Gos examples and the five-adjective test fully enumerated in prd-structure and prd-anti-patterns, sourced from pitch-writing.
- Fix shape: single owner + one-line pointers.

### [S2] Weekly roadmap reconciliation described three times — domains/pm/skills/roadmap-framing/SKILL.md (1,818w, >p90)
- Surface/dimension: skill / 2
- Evidence: "Reconciling against engineering reality" (L124–133) and "How `roadmap-curator` reads the graph weekly" (L172–182) repeat the same rules in-file; third telling in cross-domain-graph-query.
- Fix shape: merge the two internal sections; cross-ref the query mechanics.

### [S2] OST ↔ discovery-cadence mutual restatement — domains/pm/skills/opportunity-solution-trees-torres/SKILL.md, continuous-discovery-cadence/SKILL.md (both >p90)
- Surface/dimension: skill / 1 + 2
- Evidence: OST's "Continuous discovery cadence" section restates the cadence skill's core cadence; the cadence skill restates OST levels and evidence-synthesis clustering ("Phrase opportunities in the user's voice" in both).
- Fix shape: cut each restated section to a cross-ref.

### [S2] Query patterns with no runnable command — domains/pm/skills/cross-domain-graph-query/SKILL.md (1,709w, >p90)
- Surface/dimension: skill / 3
- Evidence: "Query patterns" shows rendered output blocks but never names the tool that produces them; a cold agent cannot execute "walk to all child specs … read owner, status, latest owner_history row".
- Fix shape: name the concrete surface per pattern (`hero graph`, `hero_why`, `hero list`) or mark conceptual.

---

### S3 (polish)

### [S3] Verbatim Legend block duplicated within one file — domains/engineering/skills/delivery-audit/SKILL.md (1,720w, at p90)
- Surface/dimension: skill / 2
- Evidence: the 4-line `✓ / ✗ / ~` legend appears twice, word-for-word (after "Report file format" and again after "Surface decision rules").
- Fix shape: delete the second copy.

### [S3] "Ground before you guess" paragraph duplicated verbatim across two skills — domains/engineering/skills/debugging-investigation/SKILL.md ↔ domains/engineering/skills/agent-reliability/SKILL.md
- Surface/dimension: skill / 2
- Evidence: the full ~110-word grounding-check paragraph (from `1457107`) appears identically in both files; agents loading both (debug-investigator does) pay for it twice.
- Fix shape: one owner (agent-reliability) + one-line pointer.

### [S3] Excellence-Bar / anti-punt vs Honesty-about-scope overlap — domains/engineering/skills/implementation-principles/SKILL.md ↔ domains/engineering/skills/agent-reliability/SKILL.md
- Surface/dimension: skill / 1
- Evidence: implementation-principles' "Anti-punt — hard items are not grounds for silent descoping" and agent-reliability's "No silent reclassification / no soft completion language" state the same rule set in different words; both are loaded together by engineer-facing flows.
- Fix shape: pick one home for scope-honesty rules; the other cross-references.

### [S3] Broken relative cross-links under the installed nested layout — core+eng next-handoff-emit/SKILL.md, core+eng next-md/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "[skills/next-md.md](next-md.md)" and "see skills/next-handoff-emit.md" — installed layout is `<dest>/<name>/SKILL.md` (installSkillsNested), so `next-md.md` resolves nowhere; correct relative path would be `../next-md/SKILL.md`.
- Fix shape: use skill *names*, not file links.

### [S3] Kickoff placement references a section spec-format doesn't define — domains/engineering/skills/kickoff-prompt/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "write a fresh `## Kickoff` … between `## Goal` and `## Problem`" — neither spec-format template (core or eng) has a `## Problem` section (templates go Context → Goal → Approach → Changes). Neither template includes `## Kickoff` either, despite kickoff-prompt saying it "lives in every spec."
- Fix shape: align the two skills on one template.

### [S3] `go vuln check` is not a tool — domains/engineering/skills/dependency-analysis/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "Use tools like `npm audit`, `cargo audit`, `pip audit`, or `go vuln check`" — the Go tool is `govulncheck`.
- Fix shape: rename.

### [S3] Stale hedge: `hero size --ack` now exists — domains/engineering/skills/roadmap-review/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "`hero size --ack giant <slug>` (or write `size_ack: giant` to frontmatter if the CLI flag doesn't exist yet)" — the flag exists (`hero size --help`: "--ack <tier> <slug> stamp `size_ack:`").
- Fix shape: drop the parenthetical.

### [S3] Trigger-free / overbroad description — core+eng documentation-practices/SKILL.md
- Surface/dimension: skill / 5
- Evidence: description: "Documentation guidance for writing accurate, concrete, and operationally useful technical docs." — circular (guidance for docs), no cue for *when* to load; the 89-word body adds little a competent agent doesn't do by default. Weakest earns-its-place case in the core pack.
- Fix shape: add trigger cues ("load when writing or updating README/docs files…") or fold into documentation-engineer agent.

### [S3] nudge-awareness ↔ context-injection describe the same command twice — core/skills/nudge-awareness/SKILL.md
- Surface/dimension: skill / 1
- Evidence: both skills teach `hero relevant`; nudge-awareness's "Relationship to context injection" section itself admits they are the same command at two verbosity levels. Defensible split (different audiences), but the output-format education is duplicated.
- Fix shape: keep the split, but move output interpretation to one skill and cross-reference.

### [S3] "This skill does not add Go code" — spec-authoring voice in shipped content — domains/engineering/skills/challenge-diagnosis/SKILL.md
- Surface/dimension: skill / 3
- Evidence: "Does not add Go code — this is instructions in markdown" — a design-time boundary statement meaningless to a cold agent loading the skill.
- Fix shape: delete the bullet.

### [S3] 12 frontmatter outliers vs pack schema — files listed in section (c)
- Surface/dimension: skill / 7
- Evidence: section (c) table (sales ×7 + drive missing `compatibility`; swiftui-mockup-renderer, html-mockup-generation, root-cause-classification, explainer-format missing `compatibility` + `metadata`).
- Fix shape: resolve together with the compatibility S2 — either strip `compatibility` everywhere or add it everywhere; add `metadata.audience/purpose` to the four bare files.

### [S3] `/scrub` referenced but not shipped to pm-domain installs — domains/pm/skills/duplicate-detection/SKILL.md, intake-classification/SKILL.md
- Surface/dimension: skill / 4
- Evidence: "`/scrub intake`" — scrub.md exists only in domains/engineering/commands/.
- Fix shape: reword to a pm-available surface or ship the command.

### [S3] Agent mislabeled as skill — domains/pm/skills/continuous-discovery-cadence/SKILL.md
- Surface/dimension: skill / 7
- Evidence: "The `intake-triager` and `intake-classification` skills" — intake-triager is an agent.
- Fix shape: fix the label.

### [S3] Repo-internal source paths in pm installable content — domains/pm/skills/pm-preset-detection/SKILL.md, handoff-protocol/SKILL.md, story-writing-invest/SKILL.md
- Surface/dimension: skill / 3 + 6
- Evidence: "`internal/vocabulary/` resolver", "spec store (in `internal/spec/`)", "`core/spec-types/`" — unresolvable in a user workspace.
- Fix shape: describe behavior without engine paths.

### [S3] Repeated boilerplate: "What this skill covers" restating the description (sales ×7) and unlinked "PM domain mission — principle #N" closers (pm ×~12)
- Surface/dimension: skill / 2
- Evidence: every sales skill opens with 4–6 bullets paraphrasing its own frontmatter description (~40–60 words × 7). ~12 pm skills close with "PM domain mission — principle #N (…)" pointing at a mission document whose path is never given.
- Fix shape: delete the sales section pack-wide; give the pm mission line a real path once or drop it.

### [S3] Bare-filename cross-references — domains/sales/skills/competitive-positioning/SKILL.md, forecast-methodology/SKILL.md
- Surface/dimension: skill / 3 + 4
- Evidence: "See `competitive-intel.md` for the full battlecard template" / "the format defined in `forecast.md` (the command file)" — targets exist but with no path, and a skill pointing into an agent file inverts the load direction.
- Fix shape: pack-relative paths.

---

## Notes / clean checks worth recording

- Verified real (no flag): `hero sprint status --week/--since/--md` (executive-report's recipe is correct),
  `hero relevant` positional + `--files` + `--peer/--surface`,
  `hero next ask/suggest/reflection/goal/checkpoint/path/team/shared/migrate/migrate-to-projection/ingest`,
  `hero spec lint`, `hero supersede --scan`, `hero size --check/--ack`, `hero list --format kickoff`,
  `hero spec mock detect`, `hero goal --emit/--check/--dry-run`, `hero scan --code`,
  `hero install project . --target …`, `hero admin backfill-completed-at`,
  `hero.json` keys `knowledge.auto_capture`, `team.nudge_level`, `next.projected`, `auto_context`,
  `roadmap.ambient_recency_days`, `roadmap.stop_nagging_hours`, `mockups.renderer`.
- Token-efficiency review done for all >p90 files: eng spec-format (2,482), spec-sizing (2,120),
  pm pitch-writing (2,078), cycle-planning (2,028), core spec-format (2,015), prd-anti-patterns
  (1,923), swiftui-mockup-renderer (1,890), roadmap-framing (1,818), prd-structure (1,731),
  deal-qualification (1,721), continuous-discovery-cadence (1,721), delivery-audit (1,720),
  cross-domain-graph-query (1,709), OST (1,708). Cuts named per finding above; the
  swiftui-mockup-renderer and both spec-format templates are long but template-dense — no cut
  beyond the drift consolidation.
- The 15 micro-skills (87–167w: stack packs, api-design, security-review, etc.) earn their
  place as stack-detection routing targets; no padding found.
- pm/sales READMEs: freshness-clean (all listed skills/paths exist; P1/P2 items correctly framed
  as future).
