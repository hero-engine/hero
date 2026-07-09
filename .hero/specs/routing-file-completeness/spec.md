---
title: "Routing File Completeness — full rosters, one skeleton, installable links for the domain AGENTS.md files"
slug: routing-file-completeness
type: enhancement
status: completed
priority: P2
size: small
domain: engineering
created: 2026-07-06
tags: [content-audit, routing, agents-md, domain-packs, install]
relations:
  - target: content-remediation
    kind: parent
  - target: hero-content-audit
    kind: related
  - target: pm-pack-phantom-surfaces
    kind: follows
  - target: sales-pack-reality-sync
    kind: follows
completed_at: 2026-07-09T00:05:26Z
---

# Routing File Completeness — full rosters, one skeleton, installable links for the domain AGENTS.md files

## Context

The hero-content-audit routing pass (`.hero/specs/hero-content-audit/findings-routing.md`) found that the three domain AGENTS.md files — the always-loaded body of every installed session's `AGENTS.md`/`CLAUDE.md` — under-list what ships and diverge structurally:

- **Engineering under-routes its own surface.** `domains/engineering/AGENTS.md` routes 18 commands; an engineering install ships 30 (13 pack + 17 core, recounted post-dedup at commit `177e8a1` and verified by `ls` on 2026-07-06). Eleven commands appear nowhere in the file (`/blocked` `/capture` `/challenge` `/peer` `/prime` `/resume` `/roadmap-review` `/scrub` `/split` `/why` `/hero`), and `/handoff` appears only inside prose. 33 of 35 installed agents (31 pack + 4 core) and all 53 installed skills (39 pack + 14 core) are unnamed — sessions can't route to what isn't listed. Sales proves rosters are affordable: its AGENTS.md carries complete command/agent/skill rosters at 1,233 words.
- **The three files diverge structurally, and two break the installed heading hierarchy** (audit S3). At install, `splitPackAgentsMd` (`internal/install/agents_md.go`) strips the pack H1 and the orchestrator re-emits it as an **H2** inside the managed region. Engineering's `###` sections therefore nest correctly; pm's and sales' `##` sections render as *siblings* of the pack title and visually escape Hero's managed section. Beyond depth, section order and shared obligations (session title, "run don't suggest", CLI disclaimer, compaction survival) appear in three different shapes — divergence that reads accidental, not domain-motivated. No convention tells pack authors otherwise.
- **Sales links die at install** (audit S2). ~20 relative links (`commands/qualify.md`, `agents/deal-strategist.md`, `skills/deal-qualification/SKILL.md`, `spec-types/deal.yaml`) are transplanted verbatim into the user's project-root AGENTS.md, where they 404. Engineering's `<harness>/` placeholder convention is the working pattern.
- **PM points installed users at hero-engine source paths** (audit S1, structural half): its Project Structure section lists `domains/pm/agents/`, `core/vocabularies/`, bare `hero.json`, and a `../../.hero/knowledge/...` relative link — none of which exist in a user's repo.
- **One-char roster bug:** `domains/pm/skills/README.md` says "Writing (5)" but lists six skills.

**Dual-edit constraint:** `domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody` (`internal/install/agents_md.go:379`) are held byte-identical by `TestEngineeringPackBodyMatchesGoFallback` (`internal/install/agents_md_test.go`); the regenerator (`HERO_REGEN_PACK_AGENTS=1`) rewrites the pack file from the Go fallback's output. Every engineering body change is a paired edit.

**Sibling ownership:** `pm-pack-phantom-surfaces` owns PM's phantom-command/CLI claim fixes; `sales-pack-reality-sync` owns sales content-truth claims (wrong CLI, dead hero.json schema, `deal.yaml` loading); `token-efficiency-pass` owns extracting PM's ~500 words of inline skill re-teaching; `core-commands-domain-neutral` owns `core/commands/hero.md`. This spec owns **structure, rosters, skeleton, and link/path hygiene** across all three packs — and deliberately lands after the two content-truth siblings (`follows`) so it restructures corrected text.

## Goal

All three domain AGENTS.md files share one documented skeleton at `###` heading depth, so every pack section nests inside the installed managed region. Engineering's routing table covers all 30 installed commands (core + pack) and compact roster lines name all 35 agents and 53 skills, with the Go fallback regenerated in lockstep and a new test locking roster completeness. Sales and pm carry no repo-relative links or source-tree paths — installed-content pointers use `<harness>/` placeholders and `.hero/` paths. `domains/pm/skills/README.md` says "Writing (6)". Sibling-owned claim content is relocated, never rewritten.

## Kickoff

Makes the three domain AGENTS.md files complete and uniform: engineering routes all 30 installed commands and names all agents/skills (dual-edited with the Go fallback); pm/sales adopt one `###`-depth skeleton; sales' ~20 dead relative links become `<harness>/` placeholders.

**Status:** planning — spec authored from hero-content-audit routing findings; no edits yet.

**Pick up at:** confirm both `follows` siblings landed, then write the skeleton convention entry and edit `generateEngineeringAgentsMdBody`, regenerating the pack file.

→ `HERO_REGEN_PACK_AGENTS=1 go test -run TestEngineeringPackBodyMatchesGoFallback ./internal/install/`

**Files:** internal/install/agents_md.go:379, domains/engineering/AGENTS.md, domains/pm/AGENTS.md, domains/sales/AGENTS.md
**Skip:** hand-editing engineering's AGENTS.md alone — the parity test fails unless the Go fallback moves with it.

## Approach

Three moves, smallest surface that fixes the findings:

1. **One skeleton, written down.** A knowledge convention defines the canonical section order and `###` depth for pack AGENTS.md bodies, with the renderer rationale (H1 → H2 demotion inside the managed region) so the rule survives its authors. A pointer comment goes in `agents_md.go` next to the existing on-disk-layout warning, where pack-facing authoring guidance already lives. No pointer ships inside the pack bodies themselves — an installed comment referencing a hero-engine path would recreate the 404 defect this spec removes.

2. **Completeness for engineering, structure-only for pm/sales.** Engineering gains routing rows for the 12 unrouted/prose-only commands and two compact roster sections (grouped one-liners, not per-row tables — 35 agents and 53 skills in sales-style tables would blow the always-loaded token budget). The edit is made in `generateEngineeringAgentsMdBody` and the pack file is regenerated, keeping the test green by construction. A new roster-completeness test walks the shipped content dirs and asserts coverage, so the roster can't silently rot. PM and sales adopt the skeleton (heading demotion, section reordering, roster completion for pm's 3 unlisted agents and 17 unlisted skills) with claim text relocated verbatim.

3. **Installable pointers everywhere.** Sales' relative links become backticked names with a single `<harness>/` location note per roster section (matching engineering's placeholder prose, which links nothing); pm's Project Structure is rewritten to `<harness>/` + `.hero/` placeholders, `hero.json` → `.hero/hero.json`, and the relative knowledge link becomes a plain `.hero/knowledge/decisions/...` path.

Sequencing: this spec `follows` both content-truth siblings. Restructuring first would force them to re-make edits inside moved sections; landing last means the skeleton pass operates on corrected claims and its diff is reviewable as pure moves.

## Changes

1. **`.hero/knowledge/conventions/domain-agents-md-skeleton.md` (new)** — authoring convention for pack AGENTS.md files:
   - Heading rule: one `#` H1 (pack title), all body sections `###`, never `##` — with the rationale (`splitPackAgentsMd` demotes the H1 to an H2 section title inside `<!-- hero:managed-start -->`; `##` sections escape it).
   - Canonical section order: intro paragraph → Session Title/Start → Natural Language Routing → Key Workflow → Commands Reference → Agents Reference → Skills Reference → CLI Commands (opening with the shared "These are run in the terminal, not as slash commands" disclaimer) → Project Structure → Important Rules → handoff-briefing + compaction-survival sections.
   - Path rule: installed-content pointers use `<harness>/…` placeholders; workspace pointers use `.hero/…`; no repo-relative links or hero-engine source paths.
   - Dual-edit note for engineering (Go fallback + regenerator invocation).
2. **`internal/install/agents_md.go`** — extend the existing author-facing comment near `loadPackAgentsMdBody` / the on-disk-layout warning (~lines 80–108 and 283–288) with one line pointing at the skeleton convention entry.
3. **`internal/install/agents_md.go` — `generateEngineeringAgentsMdBody` (line 379)**, then regenerate `domains/engineering/AGENTS.md` via `HERO_REGEN_PACK_AGENTS=1 go test -run TestEngineeringPackBodyMatchesGoFallback ./internal/install/`:
   - Add routing rows for `/why`, `/blocked`, `/capture`, `/challenge`, `/prime`, `/resume`, `/roadmap-review`, `/scrub`, `/split`, `/peer`, `/handoff` (session-level; keep the cross-repo CLI rows and the existing disambiguation prose), and `/hero` ("which command do I use" meta-help) — bringing the table to the full 30-command installed surface (13 engineering + 17 core).
   - Add `### Agents Reference` and `### Skills Reference` as compact grouped roster lines (e.g. "Scrubbers: comment-scrubber, deadcode-scrubber, …" / "Stacks: go-stack, react-stack, …") naming all 35 agents (31 engineering + 4 core) and all 53 skills (39 engineering + 14 core), one line of role/coverage per group, no tables, no links.
   - Budget: regenerated body ≤ 2,600 words (currently 1,954).
4. **`internal/install/agents_md_test.go` — new `TestEngineeringAgentsMdRosterComplete`**: walk `domains/engineering/{commands,agents,skills}` and `core/{commands,agents,skills}` (skip `README.md`), assert every command name appears as `/name` and every agent/skill name appears verbatim in `domains/engineering/AGENTS.md`; failure message names the missing entries.
5. **`domains/pm/AGENTS.md`** — structural pass only (claim text relocated verbatim; content truth is pm-pack-phantom-surfaces' output):
   - Demote all `##` section headings to `###`; reorder sections to the skeleton.
   - Add compact Commands/Agents/Skills roster lines covering the installed pm surface (10 pm + 17 core commands, 12 pm + 4 core agents, 19 pm + 14 core skills), naming pm's currently unlisted `duplicate-detector`, `pm-delivery-lead`, `prioritization-strategist` and 17 unlisted skills.
   - Rewrite Project Structure to `<harness>/agents|commands|skills/` placeholders (drop `domains/pm/*` and `core/*` source paths); `hero.json` → `.hero/hero.json` (here and line 84's `pm.presets` mention); replace the `../../.hero/knowledge/decisions/tracker-fronting-and-local-first.md` relative link with the plain path `.hero/knowledge/decisions/tracker-fronting-and-local-first.md`; likewise de-source-tree the `core/vocabularies/*.yaml` mention (line 239).
   - Merge the two `/refine` routing rows (lines 25 and 45) into one.
6. **`domains/sales/AGENTS.md`** — structural pass only (claim truth is sales-pack-reality-sync's output):
   - Demote all `##` section headings to `###`; drop the `---` dividers; reorder sections to the skeleton.
   - Replace the ~20 repo-relative markdown links in the Commands/Agents/Skills reference tables and the `spec-types/deal.yaml` mention with backticked names, adding one `Installed under <harness>/commands|agents|skills/` note per section (engineering-style placeholder prose, no links).
   - Split "Key CLI Commands (Sales)" into a CLI list opening with the shared terminal disclaimer and a separate slash-command examples list (invocation contents as left by the sibling spec).
7. **`domains/pm/skills/README.md` line 12** — `### Writing (5)` → `### Writing (6)`.

## Boundaries

- **No claim-truth rewrites.** PM's phantom commands and wrong CLI invocations belong to `pm-pack-phantom-surfaces`; sales' wrong CLI, fabricated hero.json schema, `deal.yaml` loading, and unbacked status/queue semantics belong to `sales-pack-reality-sync`. If a sibling hasn't landed, this spec waits — it does not fix those claims in passing.
- **No compression.** PM's ~500 words of inline skill re-teaching (handoff flow, methodology presets, plan capture) are relocated as-is; extraction to pointers is `token-efficiency-pass`.
- **`core/commands/hero.md` untouched** — router drift is `core-commands-domain-neutral`.
- **No code behavior changes**: no edits to `splitPackAgentsMd`, the managed-region orchestrator, or `installFlat` (the README-installs-as-pseudo-agent finding is a separate fix-pass code item).
- **No chat AGENTS.md** — chat is not an installable domain today (audit S3, deferred).
- **No harness-agnosticism rewrite** of engineering's "Internal Lookups — Tool Routing" section (audit S2, content-shaped, not structural).

## Risks

- **Dual-edit drift.** Any hand edit to `domains/engineering/AGENTS.md` without the Go fallback breaks `TestEngineeringPackBodyMatchesGoFallback`. Mitigated by making the change Go-side and regenerating — the pack file is then correct by construction.
- **Sibling sequencing.** If this lands before `pm-pack-phantom-surfaces`/`sales-pack-reality-sync`, the restructure moves text those specs still need to rewrite, producing conflicts and re-review. The `follows` relations encode the ordering; the deliver pre-flight must check both siblings are completed.
- **Always-loaded token growth.** Rosters and 12 routing rows add weight to a body every engineering session loads on all six targets. Bounded by the 2,600-word budget and the grouped-one-liner roster format; if the budget is at risk, cut group descriptions before cutting names.
- **Roster rot.** Static rosters drift as packs gain/lose files. `TestEngineeringAgentsMdRosterComplete` locks engineering; pm/sales rosters stay manually maintained (accepted for size — the content audit is the backstop).
- **Semantic drift during relocation.** Reordering pm/sales prose can accidentally change claim meaning. Review discipline: the pm/sales diffs must read as moves, heading-depth changes, roster additions, and path/link rewrites only.

## Acceptance Criteria

- THE SYSTEM SHALL include a Natural Language Routing row in `domains/engineering/AGENTS.md` for every one of the 30 commands installed by an engineering install (13 pack + 17 core).
- THE SYSTEM SHALL name all 35 installed agents and all 53 installed skills in `domains/engineering/AGENTS.md` roster sections.
- WHEN `go test ./internal/install/` runs THE SYSTEM SHALL pass `TestEngineeringPackBodyMatchesGoFallback` with the updated body.
- IF a shipped engineering or core command, agent, or skill is absent from `domains/engineering/AGENTS.md` THEN THE SYSTEM SHALL fail `TestEngineeringAgentsMdRosterComplete` naming the missing entry.
- THE SYSTEM SHALL use `###` depth for every body section heading in all three domain AGENTS.md files, with no `##` heading below the H1.
- THE SYSTEM SHALL present sections in the skeleton's canonical order in all three domain AGENTS.md files.
- THE SYSTEM SHALL contain no repository-relative markdown links or hero-engine source-tree paths in `domains/pm/AGENTS.md` or `domains/sales/AGENTS.md` (installed-content pointers use `<harness>/` or `.hero/` forms).
- THE SYSTEM SHALL document heading depth, section order, and path-placeholder rules in `.hero/knowledge/conventions/domain-agents-md-skeleton.md`.
- THE SYSTEM SHALL keep the `domains/engineering/AGENTS.md` body at or under 2,600 words.
- THE SYSTEM SHALL state `### Writing (6)` in `domains/pm/skills/README.md`.
- WHILE restructuring `domains/pm/AGENTS.md` and `domains/sales/AGENTS.md` THE SYSTEM SHALL preserve sibling-owned claim text unchanged apart from relocation, heading depth, and path/link form.

## Validation

- `go test ./internal/install/` — parity test and new roster-completeness test pass.
- `grep -nE '^## ' domains/engineering/AGENTS.md domains/pm/AGENTS.md domains/sales/AGENTS.md` → no matches (H1 is `# `; all sections `###`).
- `grep -nE '\]\((commands|agents|skills|spec-types|\.\./)' domains/pm/AGENTS.md domains/sales/AGENTS.md` → no matches.
- Roster spot-check: for each name in `ls domains/engineering/commands core/commands domains/engineering/agents core/agents domains/engineering/skills core/skills`, confirm it appears in `domains/engineering/AGENTS.md` (the new test automates this; run it once with a name deliberately removed to confirm it fails).
- `wc -w domains/engineering/AGENTS.md` ≤ 2,600; record pm/sales counts to confirm the structural pass didn't balloon them (pm ≤ ~2,450 given roster additions; extraction savings belong to token-efficiency-pass).
- Manual install check: `hero install --target claude` into a scratch project; open the generated `CLAUDE.md`/`AGENTS.md` and confirm every pack section for each domain nests under the managed-region H2 (no section escapes it).
- `hero check` — clean.
- Diff review of pm/sales: changes read as moves, heading demotions, roster additions, and path rewrites only.

## Completion Ledger

**Pre-flight note:** both `follows` siblings (`pm-pack-phantom-surfaces`, `sales-pack-reality-sync`) had already landed on this branch's base — verified: pm's Project Structure already uses `<harness>/` + `.hero/` placeholders and a plain `.hero/knowledge/decisions/...` link, the two `/refine` rows were already merged into one, and sales' Commands/Agents/Skills reference tables already use backticked names (no repo-relative links). This delivery built the skeleton/roster/structure pass on top of that corrected content, per the spec's sequencing.

Counts re-verified against the current tree (they had shifted from the spec's authored numbers, as flagged): engineering ships 14 pack + 15 core = **29** commands (not 30), 31 pack + 4 core = **35** agents (matches), 38 pack + 16 core = **54** skills (not 53). All Go-side logic (the roster and the new test) walks the directories directly, so it is self-correcting regardless of any documented count.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | NL Routing row in engineering AGENTS.md for every installed command (29 actual) | DONE | `internal/install/agents_md.go` NL Routing table now carries all 29 commands as literal `/name` rows; verified line-by-line against `domains/engineering/commands`, `core/commands` |
| 2 | Name all installed agents (35) and skills (54) in engineering AGENTS.md rosters | DONE | New `### Agents Reference` / `### Skills Reference` sections, grouped one-liners, all names verbatim |
| 3 | `TestEngineeringPackBodyMatchesGoFallback` passes with updated body | DONE | `go test -run TestEngineeringPackBodyMatchesGoFallback ./internal/install/` green after `HERO_REGEN_PACK_AGENTS=1` regen |
| 4 | Missing roster entry fails `TestEngineeringAgentsMdRosterComplete`, naming it | DONE | `internal/install/agents_md_test.go` — new test added; verified by temporarily stripping `session-primer` from the pack file, confirming failure names it, then restoring via regen |
| 5 | `###` depth for every body section in all three files, no `##` below H1 | DONE | `grep -nE '^## ' domains/{engineering,pm,sales}/AGENTS.md` → no matches |
| 6 | Sections in skeleton's canonical order in all three files | DONE | Engineering: Agents/Skills Reference inserted before CLI Commands. PM: Commands/Agents/Skills Reference inserted before CLI Commands; `Capture execution plans` moved before the closing handoff/compaction pair. Sales: `Key CLI Commands` moved to directly follow Skills Reference; `Domain Configuration` moved before `Surviving Context Compaction` so compaction-survival is last |
| 7 | No repo-relative links or hero-engine source paths in pm/sales AGENTS.md | DONE | `grep -nE '\]\((commands\|agents\|skills\|spec-types\|\.\./)' domains/pm/AGENTS.md domains/sales/AGENTS.md` → no matches. Sales' one remaining bare `spec-types/deal.md` path mention rewritten to reference the registered `deal` spec type + `.hero/cache/spec-types.json` (a real `.hero/`-form workspace pointer, since spec-types are never installed into a user's project) |
| 8 | Skeleton documented in `.hero/knowledge/conventions/domain-agents-md-skeleton.md` | DONE | New convention file: heading rule, canonical order, path rule, renderer rationale, dual-edit note, enforcement pointers |
| 9 | `domains/engineering/AGENTS.md` ≤ 2,600 words | DONE | `wc -w` → 2,580 |
| 10 | `domains/pm/skills/README.md` says `### Writing (6)` | DONE | Line 12 updated |
| 11 | pm/sales restructuring preserves sibling-owned claim text verbatim apart from relocation/heading/path form | DONE | Diffs are moves + heading demotions + new roster sections + the one spec-types path rewrite (see AC 7 note); no other prose changed |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | New skeleton convention file | DONE | `.hero/knowledge/conventions/domain-agents-md-skeleton.md` |
| 2 | Pointer comment in `agents_md.go` near existing authoring guidance | DONE | Added at `loadPackAgentsMdBody`'s doc comment and at the on-disk-layout warning doc comment (near `generateAgentsMdBody`) |
| 3 | Extend `generateEngineeringAgentsMdBody`, regenerate pack file | DONE | 11 new NL Routing rows (`/blocked /capture /challenge /resume /roadmap-review /scrub /split /why /hero /peer /handoff`), `/hero` added to the slash/CLI parity table, new Agents/Skills Reference sections; regenerated via `HERO_REGEN_PACK_AGENTS=1 go test -run TestEngineeringPackBodyMatchesGoFallback ./internal/install/` |
| 4 | New `TestEngineeringAgentsMdRosterComplete` | DONE | Walks `domains/engineering/{commands,agents,skills}` + `core/{commands,agents,skills}` (skip README.md), asserts coverage; verified it fails-and-names on a deliberately removed entry |
| 5 | pm structural pass + roster | DONE | `##`→`###`; Commands/Agents/Skills Reference sections added (naming `pm-delivery-lead`, `prioritization-strategist`, `duplicate-detector`, and all 19 pm + 16 core skills); `Capture execution plans` relocated before the closing pair. The two `/refine` rows and the Project Structure/hero.json/knowledge-link fixes were already done by `pm-pack-phantom-surfaces` — verified, not redone |
| 6 | sales structural pass + link fix | DONE | `##`→`###`; `---` divider lines removed (the two `---` inside the Deal Spec Structure YAML example were preserved); `Key CLI Commands (Sales)` moved up; `Domain Configuration`/`Surviving Context Compaction` swapped; `spec-types/deal.md` path reference rewritten. The ~20 relative links in the reference tables were already fixed by `sales-pack-reality-sync` — verified, not redone |
| 7 | `domains/pm/skills/README.md` line 12 fix | DONE | `Writing (5)` → `Writing (6)` |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: built `/tmp/hero-rfc` from this worktree, ran `hero init --domain engineering|pm|sales` + `hero install project . --target claude` into three scratch repos, and inspected the generated `CLAUDE.md` in each — every pack section (including the new Agents/Skills Reference sections) nests as `###` under the installed `##` H2, none escapes the managed region. Also ran `hero check` in this worktree — no failures (pre-existing advisory findings only, unrelated to this spec).

### Excellence Bar self-check

Yes — the roster-completeness test is real (verified fail-and-restore), the dual-edit stays enforced by construction (Go-side edit + regen, not a hand-edit), the skeleton convention documents the renderer rationale rather than asserting a bare rule, and the pm/sales diffs are honest moves/demotions/additions with one narrowly-scoped path rewrite called out rather than silently folded in.
