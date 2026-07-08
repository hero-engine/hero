---
title: "Core commands are engineering-only — dangling agents, skills, routes, and storage paths on pm/sales installs"
slug: core-commands-domain-neutral
type: bug
status: completed
priority: P1
size: medium
domain: engineering
created: 2026-07-06
tags: [content, core-commands, domain-packs, overlay, audit-remediation]
relations:
  - {target: content-remediation, kind: parent}
  - {target: hero-content-audit, kind: related}
  - {target: content-dedup-resync, kind: builds-on}
delivery_method: manual
completed_at: 2026-07-08T18:09:57Z
---

# Core commands are engineering-only — dangling agents, skills, routes, and storage paths on pm/sales installs

## Context

Since content-dedup-resync (commit `177e8a1`), `core/commands/*.md` is the
single master copy installed into **every** domain: pm and sales installs get
each core command via OverlayFS fallthrough unless the pack ships an annotated
`core_fork:` shadow (enforced by `content_parity_test.go` — unannotated or
byte-identical shadows fail CI). That fix ended the core↔engineering drift
(and already ported the F1 `hero sync import` correction into
`core/commands/import.md`), but it also made a latent problem load-bearing:
several core commands are written as if every install were an engineering
install.

From the audit (`.hero/specs/hero-content-audit/findings-commands.md`):

- **F4 (S1):** `core/commands/{check,decide,discover,retro,drive,hero}.md`
  delegate to engineering-pack-only agents — `architecture-reviewer`,
  `dependency-analyst`, `brownfield-architect`, `greenfield-architect`,
  `product-ideator`, `feature-delivery-lead`, `platform-delivery-lead` — or
  load the engineering-only `drive` skill. Core ships only 4 agents
  (`convention-author`, `documentation-engineer`, `project-context-builder`,
  `session-primer`); on pm/sales installs every one of those references
  dangles. `discover.md` additionally hands off to `/design`, which is
  engineering-pack-only.
- **F16 (S2):** `core/commands/hero.md` (the meta-router) hardcodes a
  14-workflow list that is stale on both axes: 6 of its routes
  (`/design /diagnose /deliver /review /compose /release`) are phantoms on
  pm/sales installs, and it omits over half the engineering pack
  (`/mock /drive /sprint /import /peer /challenge /split /scrub /prime
  /resume /roadmap-review`). It also duplicates the per-install AGENTS.md
  routing table wholesale.
- **F15 (S2):** `core/commands/prime.md` and `core/commands/resume.md` both
  claim the session-start slot with overlapping steps and no mutual
  disambiguation; `resume.md` claims it unconditionally.
- **F22 (partial, S2):** `core/commands/handoff.md` cites
  `skills/next-md.md` and `skills/kickoff-prompt.md` — the installed layout
  is `skills/<name>/SKILL.md`, so both paths are dead. Worse,
  `kickoff-prompt` lives at `domains/engineering/skills/kickoff-prompt/`
  (engineering-only), so even a corrected path dangles on pm/sales; `next-md`
  is core (`core/skills/next-md/`).
- **Storage-path claims (verified this session):** `core/commands/decide.md`
  says decisions save to `.hero/decisions/{slug}/spec.md`; the real
  convention is flat files under `.hero/knowledge/decisions/` (verified:
  `ls .hero/knowledge/decisions/` shows flat `<slug>.md` files, no folders).
  Same family: `core/commands/check.md` loads conventions from
  `.hero/conventions/` and `core/commands/convention.md` saves to
  `.hero/conventions/{slug}/spec.md` — the real location is flat files under
  `.hero/knowledge/conventions/` (verified by `ls`). No other core command
  cites a wrong storage path (`grep -n '\.hero/' core/commands/*.md` —
  remaining hits are `.hero/QUEUE.md`, `.hero/NEXT.md`, `.hero/knowledge/`,
  all real).

All CLI invocations these files make were re-verified against the installed
binary and are real: `hero check --reconcile`, `hero drift --in-flight`,
`hero queue --limit`, `hero goal --emit|--check`, `hero next path|checkpoint`,
`hero recap`, `hero resume`, `hero do`, `hero spec new`, `hero dashboard`.
The bugs are agent/skill/command references and storage paths, not CLI claims.

## Goal

Every command remaining in `core/commands/` reads correctly on any install
(engineering, pm, sales): no unconditional reference to an agent, skill, or
slash command that the install's merged surface doesn't ship, and every
storage-path claim matches the real on-disk convention. `/hero` routes
correctly per domain by deferring to the install's routing table instead of a
hardcoded list. Exactly one command (`/resume`) owns the session-start slot,
carrying forward prime's unique steps. `/drive` ships only where its skill
and downstream commands exist (the engineering pack). `go test ./...`
(including the content parity gate) passes.

## Kickoff

Makes the core command pack safe on every domain install — today pm/sales
users get commands that delegate to engineering-only agents, route to
nonexistent slash commands, and cite storage paths that were never real.

**Status:** planning — spec authored from audit findings F4/F15/F16/F22 plus
a verified storage-path sweep; no edits yet.

**Pick up at:** Change 1 — slim `core/commands/hero.md` to defer to the
install's instruction-file routing table; then work the Changes list in order.

→ `/deliver core-commands-domain-neutral`

**Files:** core/commands/hero.md, core/commands/decide.md, core/commands/retro.md, core/commands/resume.md, content_parity_test.go
**Skip:** per-domain regeneration of /hero at install time; promoting the `drive` skill to core — relocated to the engineering pack instead (see Approach). (`kickoff-prompt` promotion is owned by [[pm-pack-phantom-surfaces]] Change 2 — initiative reconciliation.)

## Approach

Three patterns, chosen per file — prefer the cheapest that keeps
`core/commands/` a single domain-neutral master:

1. **Generic rule + scoped examples (default).** Rewrite delegation as
   "delegate to the install's <role> agent if one is installed; otherwise
   perform the work directly as the session agent," with pack-specific names
   only inside explicitly scoped parentheticals ("engineering installs ship
   `feature-delivery-lead` / `platform-delivery-lead`; pm ships
   `pm-delivery-lead`"). A scoped example can't dangle — it names its own
   precondition. This avoids per-domain `core_fork:` shadows, which the
   parity test allows but which recreate the N-copies maintenance problem
   content-dedup-resync just eliminated. No new forks are introduced by this
   spec.

2. **Relocate, don't neutralize, when the command is inherently
   engineering-shaped.** `/drive` loads the engineering-only `drive` skill
   and its protocol dispatches `/design` and `/deliver` per child spec —
   there is no honest domain-neutral fallback (the loop cannot run without
   those commands). Move the file into the engineering pack. After the move
   it is pack-native (no core counterpart), so no `core_fork:` annotation is
   needed and the parity test is unaffected.

3. **Defer to generated-per-install surfaces instead of hand-maintained
   lists.** `/hero`'s workflow list can never be simultaneously correct for
   three domains from one static file. Rather than build install-time
   templating (rejected: engine work, disproportionate for a content bug),
   slim `/hero` to read the routing table in the install's instruction file
   (AGENTS.md / CLAUDE.md managed region) — which *is* regenerated per domain
   at install time — match intent to a row, and run it. `hero do` remains
   the CLI twin.

For F15, **fold prime into resume and retire prime** (recommended over a
stated split). Rationale: `resume.md` already claims session-start
unconditionally; `hero resume` output already surfaces the handoff digest
("Where you left off") and `resume.md` step 5 already covers the queue
surface — prime's only non-duplicated steps are `hero check --reconcile`,
the team-roster glance at `.hero/NEXT.md`, and the `session-primer`
deep-orientation persona. Three lines absorbed into resume beat maintaining
two overlapping commands with a documented boundary that every agent must
parse at every session start.

For F22, reference skills by NAME (no `skills/<name>.md` paths).
**Initiative reconciliation:** the sibling [[pm-pack-phantom-surfaces]]
(Change 2) promotes `kickoff-prompt` to `core/skills/` with the
engineering command names scoped in its body — that resolves this spec's
original objection (promoting would relocate the `/design`/`/deliver`
dangling refs) at the source. This spec therefore treats `kickoff-prompt`
as core: reference it by name, no install-conditional hedging needed once
the sibling lands (both are wave 1; whichever delivers second drops the
hedge).

## Changes

1. **`core/commands/hero.md` — slim the meta-router (F16).** Delete the
   "Available workflows" section (both the 14-command list and the CLI list)
   and the hardcoded intent→command table in "Routing logic". Replace with:
   read the routing table in this install's instruction file (the managed
   region of AGENTS.md / CLAUDE.md — regenerated per domain at install time,
   so it lists exactly the commands this install ships); match the user's
   intent to a row and run that command, passing the original context as
   arguments; if ambiguous, present the top 2–3 candidate rows and ask; if
   nothing matches, show the table's command column and ask. Keep the
   existing "Important" rules (run, don't echo; pass slugs through) and the
   pointer to `hero do <request>` as the CLI equivalent.

2. **`core/commands/check.md` — conditional delegation + conventions path
   (F4 + path sweep).** In the "Convention compliance" branch: load active
   conventions from `.hero/knowledge/conventions/` (not `.hero/conventions/`),
   and delegate to the install's architecture/review agent if one is
   installed (engineering installs ship `architecture-reviewer`), otherwise
   perform the compliance review directly as the session agent. In the
   "Dependencies" branch: delegate to a dependency-analysis agent if
   installed (engineering ships `dependency-analyst`), otherwise run the
   audit directly. General/stale/drift branches are already neutral — leave
   untouched.

3. **`core/commands/decide.md` — neutral evaluation + decisions path (F4 +
   verified path fix).** Keep step 1 (`hero_anchor` tripwire check) verbatim.
   Replace the hard "Route this ... to `architecture-reviewer`" with: run the
   structured evaluation directly as the session agent, or delegate to the
   install's reviewer agent if one is installed (engineering:
   `architecture-reviewer`; pm: `pm-reviewer`). Scope the architect consults
   as engineering-install-only ("on engineering installs, also consult
   `brownfield-architect` for existing-system concerns or
   `greenfield-architect` for new subsystems"). Change the save path from
   `.hero/decisions/{slug}/spec.md` to a flat `.hero/knowledge/decisions/<slug>.md`
   using the decision (ADR) template from the `spec-format` skill.

4. **`core/commands/convention.md` — conventions path (path sweep).** Change
   the save path from `.hero/conventions/{slug}/spec.md` to a flat
   `.hero/knowledge/conventions/<slug>.md`. No other edits — the
   subproject-workspace guard and the rest of the flow are already neutral.

5. **`core/commands/discover.md` — agent fallback + handoff target (F4).**
   Replace "using the `product-ideator` agent" with: be the install's
   ideation agent if one is installed (engineering ships `product-ideator`),
   otherwise run the discovery conversation directly as the session agent.
   Change the output framing from "ready for `/design`" to "ready for the
   install's design workflow (`/design` where installed; otherwise scaffold
   with `hero spec new <slug>`)". Note: pm ships an annotated `core_fork:`
   override of discover.md, so this file's live audience is engineering and
   sales installs — do not touch the pm fork.

6. **`core/commands/retro.md` — delivery-lead fallback (F4).** Replace the
   product/platform routing preamble with: delegate to the install's
   delivery-lead agent if one is installed (engineering installs route
   product work to `feature-delivery-lead` and platform work to
   `platform-delivery-lead`; pm installs ship `pm-delivery-lead`), otherwise
   run the retrospective directly as the session agent. Steps 1–6 (spec vs
   git comparison, learnings, auto knowledge capture) are already neutral —
   keep verbatim.

7. **Move `core/commands/drive.md` → `domains/engineering/commands/drive.md`
   (F4, relocation).** `git mv`, content unchanged. The command loads the
   engineering-only `drive` skill and its run protocol dispatches `/design`
   and `/deliver` per child — engineering-pack-only by nature. After the
   move there is no core counterpart, so no `core_fork:` annotation is
   needed. pm/sales installs stop shipping a `/drive` that could never
   complete a single transition on their surface.

8. **`core/commands/resume.md` — absorb prime's unique steps (F15).** Add to
   the Steps list: (a) after reading the `hero resume` output, run
   `hero check --reconcile` to fix status drift — silent when clean; (b) in
   team mode, also glance at `.hero/NEXT.md` for the team roster; (c) a
   closing line: "For a deeper orientation on conventions, decisions, and
   risks, be the `session-primer` agent (core)." Do not otherwise rewrite the
   file — the F27 token trim is out of scope here.

9. **Delete `core/commands/prime.md` (F15).** `/resume` is now the single
   session-start command. `hero resume` output already carries the handoff
   digest and in-flight list; resume step 5 already covers the
   queue/kickoff surface; the three prime-unique steps moved in Change 8.

10. **`core/skills/knowledge-flywheel/SKILL.md` line 18 — retarget the
    `/prime` reference (F15 follow-through).** "When `/prime` detects a
    growing knowledge base" → "When `/resume` detects a growing knowledge
    base". This is the only content reference to `/prime` outside prime.md
    itself (`grep -rn "/prime" core/ domains/`; the
    `internal/version/version_test.go` hit is an inert fixture string —
    leave it).

11. **`core/commands/handoff.md` — skill references by name (F22
    partial).** Replace the closing "See `skills/next-md.md` ... and
    `skills/kickoff-prompt.md` ..." paragraph with: "See the `next-md`
    skill for the NEXT.md format and the `kickoff-prompt` skill for the
    `## Kickoff` sections that populate `hero queue` / QUEUE.md." Skill
    names only — no `skills/<name>.md` paths. (Valid on every install
    once [[pm-pack-phantom-surfaces]] Change 2 promotes `kickoff-prompt`
    to core; if this spec delivers first, add "where installed" and let
    the sibling drop it.)

## Boundaries

- **pm/sales AGENTS.md and routing tables are sibling scope.** Phantom slash
  and CLI routes in `domains/pm/AGENTS.md` / `domains/sales/AGENTS.md`
  (F2, F3, F10) belong to `pm-pack-phantom-surfaces`; the engineering
  AGENTS.md unrouted-commands gap (F24) and the slash/CLI parity table
  relocation and correction (F22's routing half) belong to
  `routing-file-completeness`. This spec must, however, hand those siblings
  two facts: `/prime` is retired and `/drive` is engineering-pack-only —
  their tables must not list either on the wrong surface.
- **No engine/Go changes** beyond `git mv` of a content file. Install-time
  per-domain generation of `/hero` was considered and rejected (Approach 3).
- **No new `core_fork:` shadows.** Existing pm forks (discover, handoff) are
  untouched.
- **Domain-pack command bodies** (F5–F8, F12–F14, F17–F21, F23, F25, F26,
  F28) and the dead chat pack (F9, F29) are other children of
  content-remediation.
- **Token-efficiency trims** (F27 resume padding) are not this spec — resume
  edits here are additive-only.
- **README / GETTING-STARTED count updates** are in scope only as far as
  `hero docs check` requires (command counts change: core 17→15,
  engineering-native 13→14).

## Risks

- **Stale installs keep the old files.** Installed harness copies of
  `/prime`, `/drive`, and the old `/hero` persist until the next
  `hero install` run; the fix ships with the next release, not
  retroactively. Mitigation: none needed beyond normal upgrade flow, but
  release notes should mention the `/prime` → `/resume` fold.
- **Routing-table dependency.** The slimmed `/hero` assumes every install
  has an instruction file with a routing table. `internal/install/agents_md.go`
  generates one for every target (including the Go fallback body), so this
  holds; if a user hand-deletes the managed region, `/hero` degrades to
  asking — acceptable.
- **Cross-spec coordination.** If `routing-file-completeness` lands first
  and adds `/prime` or `/drive` rows, this spec silently invalidates them.
  Sequence this spec first within the initiative, or flag the retirement in
  both specs' trackers.
- **`hero docs check` drift.** README claims about command/agent counts and
  the repo's own installed CLAUDE.md parity table (which lists `/prime` and
  `/drive`) may fail the docs gate after Changes 7 and 9 — fix counts in the
  same commit rather than treating it as scope creep.
- **Behavioral regression at session start.** Folding `hero check
  --reconcile` into `/resume` adds a write-capable step to every session
  start; it is a no-op when statuses are clean and was already the behavior
  for anyone using `/prime`.

## Acceptance Criteria

- THE SYSTEM SHALL reference no agent, skill, or slash command from any `core/commands/*.md` file unless it ships in core or the reference is scoped to the installs that ship it.
- WHEN a user invokes `/hero` on any domain install THE SYSTEM SHALL route by matching intent against the install's instruction-file routing table rather than a hardcoded workflow list.
- WHEN `/decide` saves a decision THE SYSTEM SHALL cite `.hero/knowledge/decisions/<slug>.md` as the save path.
- WHEN `/convention` saves a convention or `/check conventions` loads them THE SYSTEM SHALL cite `.hero/knowledge/conventions/` as the storage location.
- WHEN a fresh session starts in a hero-aware repo THE SYSTEM SHALL offer exactly one session-start command, `/resume`, which includes `hero check --reconcile` and the session-primer deep-orientation pointer.
- WHEN Hero installs the pm or sales domain THE SYSTEM SHALL not ship a `/drive` or `/prime` command.
- WHEN Hero installs the engineering domain THE SYSTEM SHALL ship `/drive` from `domains/engineering/commands/drive.md` with content unchanged from the pre-move core copy.
- THE SYSTEM SHALL reference skills in `core/commands/handoff.md` by skill name only, with no `skills/<name>.md` path forms remaining.
- IF a domain pack file shadows a core command path THEN THE SYSTEM SHALL carry a non-empty `core_fork:` annotation and differ from the core content (existing CI gate stays green).

## Validation

- `go test ./...` — must pass, in particular
  `TestDomainPacks_NoUnannotatedCoreShadows` (no new shadows introduced;
  drive.md move leaves no core counterpart) and any docs-drift tests.
- Dangling-reference sweep:
  `grep -rn "architecture-reviewer\|dependency-analyst\|brownfield-architect\|greenfield-architect\|product-ideator\|feature-delivery-lead\|platform-delivery-lead\|pm-reviewer\|pm-delivery-lead" core/commands/`
  — every hit sits inside an "if installed" / "engineering installs" /
  "pm installs" scope.
- Stale-path sweep: `grep -rn "\.hero/decisions\|\.hero/conventions" core/commands/`
  returns nothing; `grep -rn "skills/.*\.md" core/commands/` returns nothing.
- Retirement sweep: `core/commands/prime.md` and `core/commands/drive.md` do
  not exist; `grep -rn "/prime" core/ domains/` returns nothing;
  `domains/engineering/commands/drive.md` exists and matches the pre-move
  content (`git diff --find-renames` shows a pure rename).
- Install smoke test: `hero install --domain pm` (and `--domain sales`) into
  a temp dir; for each installed core command, confirm every agent/skill
  name it references either resolves in the merged install surface or is
  conditionally scoped; confirm `/drive` and `/prime` are absent and `/hero`
  contains no hardcoded workflow list.
- `hero docs check` — passes after count updates.
- Manual read-through of the slimmed `/hero` on an engineering install:
  confirm a "fix this bug" request routes to `/diagnose` via the routing
  table with no hardcoded fallback.

## Completion Ledger

Base commit: `752e516`. `kickoff-prompt` was confirmed already promoted to
`core/skills/` by the sibling `pm-pack-phantom-surfaces` (commit `65e3333`,
landed before this delivery started), so Change 11 drops the "where
installed" hedge per the spec's own instruction.

### Changes

| # | Change | Status | Evidence |
|---|---|---|---|
| 1 | `core/commands/hero.md` — slim the meta-router | DONE | Deleted "Available workflows" (14-command + CLI lists) and the hardcoded intent table; replaced with "read the routing table in this install's instruction file... match intent... run it"; kept the "Important" rules and the `hero do` CLI pointer. `git diff 752e516 -- core/commands/hero.md` shows 59 lines removed/replaced. |
| 2 | `core/commands/check.md` — conditional delegation + conventions path | DONE | Conventions path now `.hero/knowledge/conventions/`; convention-compliance and dependency branches both read "delegate ... if one is installed (engineering ships X); otherwise perform ... directly". General/stale/drift branches untouched (diff shows only 2 hunks touched). |
| 3 | `core/commands/decide.md` — neutral evaluation + decisions path | DONE | Step 1 (`hero_anchor`) kept verbatim; hard route to `architecture-reviewer` replaced with "run directly ... or delegate ... (engineering: `architecture-reviewer`; pm: `pm-reviewer`)"; architect consults scoped "On engineering installs..."; save path is now `.hero/knowledge/decisions/<slug>.md` citing the `spec-format` skill's ADR template. |
| 4 | `core/commands/convention.md` — conventions path | DONE | Save path changed to `.hero/knowledge/conventions/<slug>.md`; single-line diff, nothing else touched. |
| 5 | `core/commands/discover.md` — agent fallback + handoff target | DONE | "using the `product-ideator` agent" → "be the install's ideation agent if one is installed... otherwise run directly"; output framing now points to "the install's design workflow (`/design` where installed; otherwise `hero spec new <slug>`)"; pm's `core_fork:` discover.md left untouched (verified: only `domains/pm/commands/discover.md` and `handoff.md` carry `core_fork:` in the repo). |
| 6 | `core/commands/retro.md` — delivery-lead fallback | DONE | Preamble replaced with "delegate to the install's delivery-lead agent if one is installed... otherwise run directly"; engineering/pm names scoped in a follow-up sentence; steps 1–6 left verbatim (diff touches only the opening paragraph). |
| 7 | Move `core/commands/drive.md` → `domains/engineering/commands/drive.md` | DONE | `git mv`; `git diff --find-renames --stat` shows a pure 0-line-changed rename; byte-for-byte `diff` against the pre-move blob confirms identical content. |
| 8 | `core/commands/resume.md` — absorb prime's unique steps | DONE | Added steps 7 (`hero check --reconcile`, silent when clean) and 8 (team-mode `.hero/NEXT.md` roster glance), plus a closing `session-primer` deep-orientation pointer. Rest of file untouched. |
| 9 | Delete `core/commands/prime.md` | DONE | `git rm core/commands/prime.md`; file absent from `core/commands/` (`ls` confirms). |
| 10 | `core/skills/knowledge-flywheel/SKILL.md` — retarget `/prime` | DONE | Line 18: "When `/prime` detects..." → "When `/resume` detects...". `grep -rn "/prime" core/ domains/` returns nothing; `internal/version/version_test.go:43` fixture string left untouched as instructed. |
| 11 | `core/commands/handoff.md` — skill references by name | DONE | Closing paragraph now cites "the `next-md` skill" and "the `kickoff-prompt` skill" by name, no `skills/<name>.md` paths, no hedge (sibling already landed the core promotion). |

### Acceptance Criteria

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | No unscoped agent/skill/command ref in `core/commands/*.md` | DONE | `grep -rn "architecture-reviewer\|dependency-analyst\|brownfield-architect\|greenfield-architect\|product-ideator\|feature-delivery-lead\|platform-delivery-lead\|pm-reviewer\|pm-delivery-lead" core/commands/` — every hit is inside "if installed"/"engineering installs"/"pm:" scoping (9 hits across decide.md, check.md, retro.md, discover.md, all scoped). |
| 2 | `/hero` routes via install's instruction-file table | DONE | `core/commands/hero.md` routing logic now reads "the routing table in this install's instruction file (the managed region of AGENTS.md / CLAUDE.md...)"; no hardcoded workflow list remains. |
| 3 | `/decide` cites `.hero/knowledge/decisions/<slug>.md` | DONE | `core/commands/decide.md:14` (post-edit) — confirmed by `grep -n "knowledge/decisions" core/commands/decide.md`. |
| 4 | `/convention` and `/check conventions` cite `.hero/knowledge/conventions/` | DONE | `core/commands/convention.md` save path + `core/commands/check.md` load path both updated; `grep -rn "\.hero/decisions\|\.hero/conventions" core/commands/` returns empty. |
| 5 | Exactly one session-start command (`/resume`), with reconcile + session-primer pointer | DONE | `core/commands/prime.md` deleted; `core/commands/resume.md` carries `hero check --reconcile` (step 7), team-roster glance (step 8), and the `session-primer` closing pointer. |
| 6 | pm/sales installs ship no `/drive` or `/prime` | DONE | Install smoke: `hero install project <tmp> --target claude --domain pm --root` and `--domain sales --root` — `ls .claude/commands/drive.md .claude/commands/prime.md` → "No such file" for both domains. |
| 7 | Engineering install ships `/drive` unchanged from pre-move core copy | DONE | Install smoke: `--domain engineering --root` → `.claude/commands/drive.md` present, `.claude/commands/prime.md` absent. Content identity confirmed at the move (Change 7 evidence). |
| 8 | `core/commands/handoff.md` cites skills by name only | DONE | `grep -rn "skills/.*\.md" core/commands/` returns empty; handoff.md now reads "the `next-md` skill" / "the `kickoff-prompt` skill". |
| 9 | Shadowing domain files still carry non-empty `core_fork:` and differ from core | DONE | `go test . -run TestDomainPacks_NoUnannotatedCoreShadows -v` — PASS for engineering, sales, pm subtests. Only `domains/pm/commands/discover.md` and `domains/pm/commands/handoff.md` shadow core, both pre-existing `core_fork:` annotations, both untouched by this spec. |

### Validation

| Check | Status | Evidence |
|---|---|---|
| `go test ./...` | DONE | Full suite green: `contracts`, `contracts/peering`, and all `internal/*` packages `ok`; root package (`content_parity_test.go`) `ok` (cached, and re-run explicitly with `-run TestDomainPacks_NoUnannotatedCoreShadows -v` — 3/3 subtests PASS). Zero FAIL lines across the run. |
| Dangling-reference sweep | DONE | See AC #1 evidence — 9 hits, all scoped. |
| Stale-path sweep | DONE | `grep -rn "\.hero/decisions\|\.hero/conventions" core/commands/` and `grep -rn "skills/.*\.md" core/commands/` both empty. |
| Retirement sweep | DONE | `core/commands/prime.md` and `core/commands/drive.md` absent; `grep -rn "/prime" core/ domains/` empty; `domains/engineering/commands/drive.md` exists, `git diff --find-renames` shows pure rename (0 lines changed), byte-diff against pre-move blob is identical. |
| Install smoke test (pm, sales, engineering) | DONE | Three fresh `--root` installs built from this worktree's binary; pm and sales ship no `drive.md`/`prime.md` and `kickoff-prompt` resolves under `.claude/skills/`; engineering ships `drive.md`, no `prime.md`; every scoped agent name referenced by an installed core command resolves in that domain's installed `.claude/agents/` (spot-checked `pm-reviewer`, `pm-delivery-lead`, `session-primer` on pm; `session-primer` on sales). `/hero` contains no hardcoded workflow list on any install. |
| `hero docs check` | SKIPPED (pre-existing, out of scope) | Fails identically on base commit `752e516` before any edit in this spec (`agents: claims 34, actual 0` / `skills: claims 45, actual 0`) — `findProjectRoot()` resolves to this repo's git root, which has no top-level `agents/`/`commands/`/`skills/` dirs (this repo uses `.agents/`), so the check has been structurally broken independent of this spec's content. No literal "core 17→15" / "engineering 13→14" count claim exists anywhere in README.md or GETTING-STARTED.md to update — the spec's Boundaries note anticipated a claim that isn't present in the current tree. This repo's own root `AGENTS.md`/`CLAUDE.md` (tracked files, last touched at `3e34f72`) contain a stale hand-authored `/prime` mention inside the Hero-managed region, but `TestMarkdownInvocationsResolveAgainstRootCmd` and `content_parity_test.go` both explicitly scope their scan to `domains/engineering/AGENTS.md` (the clean source template) and exclude the repo's own root install artifacts — confirmed no test gates on this staleness. Regenerating those root files was judged out of Boundaries ("No engine/Go changes beyond `git mv`") and out of surgical-change scope for a content-only spec; flagged here rather than silently left. |
| Manual read-through of slimmed `/hero` | DONE | Read `core/commands/hero.md` post-edit: a "fix this bug" request has no hardcoded fallback table to match against — the agent is instructed to read the install's own routing table (which on an engineering install has "Bug, error, broken, fix, investigate, diagnose → `/diagnose`") and route there. No dangling logic. |
