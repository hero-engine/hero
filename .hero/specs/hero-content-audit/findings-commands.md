# Findings — surface pass 3b (COMMANDS)

Auditor pass over all 72 slash-command definition files (17 core, 30 engineering,
11 pm, 8 sales, 6 chat). Rubric: `.hero/planning/features/hero-content-audit/rubric.md`.
All CLI claims verified against the installed `hero` binary (`hero --help` +
per-subcommand `--help`); skill/agent cross-references verified against the
merged install surface (`OverlayFS(domain, core)` per `content.go`).

Severity counts: **S1 × 8 · S2 × 14 (incl. F9, graded S1→S2) · S3 × 7** (29 findings; header corrected to match the graded findings below).

---

## A. Coverage table (72/72 files read)

| File | Verdict |
|---|---|
| core/commands/blocked.md | clean |
| core/commands/capture.md | clean |
| core/commands/check.md | flagged (F4) |
| core/commands/convention.md | clean |
| core/commands/decide.md | flagged (F4) |
| core/commands/discover.md | flagged (F4) |
| core/commands/docs.md | clean |
| core/commands/drive.md | flagged (F4) |
| core/commands/handoff.md | flagged (F11, F22) |
| core/commands/hero.md | flagged (F4, F16) |
| core/commands/import.md | flagged (F1) |
| core/commands/note.md | clean |
| core/commands/prime.md | flagged (F15) |
| core/commands/resume.md | flagged (F15, F27) |
| core/commands/retro.md | flagged (F4) |
| core/commands/scan.md | clean |
| core/commands/why.md | clean |
| domains/engineering/commands/blocked.md | clean |
| domains/engineering/commands/capture.md | clean |
| domains/engineering/commands/challenge.md | clean |
| domains/engineering/commands/check.md | clean |
| domains/engineering/commands/compose.md | clean |
| domains/engineering/commands/convention.md | clean |
| domains/engineering/commands/decide.md | clean |
| domains/engineering/commands/deliver.md | flagged (F12, F23) |
| domains/engineering/commands/design.md | clean |
| domains/engineering/commands/diagnose.md | flagged (F17, F23) |
| domains/engineering/commands/discover.md | clean |
| domains/engineering/commands/docs.md | clean |
| domains/engineering/commands/drive.md | clean |
| domains/engineering/commands/handoff.md | flagged (F22) |
| domains/engineering/commands/hero.md | flagged (F16) |
| domains/engineering/commands/import.md | clean |
| domains/engineering/commands/mock.md | flagged (F23) |
| domains/engineering/commands/note.md | clean |
| domains/engineering/commands/peer.md | flagged (F13) |
| domains/engineering/commands/prime.md | clean |
| domains/engineering/commands/release.md | flagged (F14) |
| domains/engineering/commands/resume.md | flagged (F27) |
| domains/engineering/commands/retro.md | clean |
| domains/engineering/commands/review.md | clean |
| domains/engineering/commands/roadmap-review.md | clean |
| domains/engineering/commands/scan.md | clean |
| domains/engineering/commands/scrub.md | clean |
| domains/engineering/commands/split.md | clean |
| domains/engineering/commands/sprint.md | flagged (F8) |
| domains/engineering/commands/why.md | clean |
| domains/pm/commands/README.md | doc-only (flagged, F19) |
| domains/pm/commands/discover.md | flagged (F26) |
| domains/pm/commands/handoff.md | flagged (F3, F18) |
| domains/pm/commands/metrics.md | flagged (F25) |
| domains/pm/commands/pitch.md | flagged (F25) |
| domains/pm/commands/prd.md | clean |
| domains/pm/commands/prioritize.md | flagged (F3) |
| domains/pm/commands/refine.md | flagged (F25) |
| domains/pm/commands/release-notes.md | flagged (F3, F20) |
| domains/pm/commands/roadmap.md | flagged (F3) |
| domains/pm/commands/triage.md | flagged (F3, F7) |
| domains/sales/commands/README.md | doc-only (clean) |
| domains/sales/commands/debrief.md | clean |
| domains/sales/commands/forecast.md | flagged (F28) |
| domains/sales/commands/pipeline.md | flagged (F28) |
| domains/sales/commands/prospect.md | flagged (F21) |
| domains/sales/commands/qualify.md | clean |
| domains/sales/commands/research.md | flagged (F5) |
| domains/sales/commands/strategize.md | flagged (F6, F21) |
| domains/chat/commands/ask-corpus.md | flagged (F9, F29) |
| domains/chat/commands/capture.md | flagged (F9) |
| domains/chat/commands/discover.md | flagged (F9) |
| domains/chat/commands/note.md | flagged (F9) |
| domains/chat/commands/space.md | flagged (F9, F29) |
| domains/chat/commands/why.md | flagged (F9) |

---

## B. Routing-table alignment results

Sources checked: `domains/engineering/AGENTS.md`, `domains/pm/AGENTS.md`,
`domains/sales/AGENTS.md`, the Go fallback body in
`internal/install/agents_md.go` (`generateEngineeringAgentsMdBody`, kept
content-identical to the engineering AGENTS.md by test), and the slash/CLI
parity table found only in this repo's installed `CLAUDE.md` managed region
(lines 45-52) — **the parity table has no source in any pack AGENTS.md or in
the Go generator** (only grep hit in the whole repo is `CLAUDE.md:49`).

### Engineering (install = engineering + core, engineering wins collisions)
- **Phantom routes: none.** All 18 slash routes in the table resolve to files.
- **Unlisted shipped commands (11):** `/blocked`, `/capture`, `/challenge`,
  `/hero`, `/peer`, `/prime`, `/resume`, `/roadmap-review`, `/scrub`,
  `/split`, `/why` have no routing row (`/handoff` appears only in prose).
  Rubric: nothing shipped goes unlisted. -> F24 (S3).
- **CLI claims in the CLI Commands section:** all verified real
  (`hero status/search/snapshot/sync import/sync pull/note/check/peer
  list|show|call --mode/handoff [status|accept]/admin repos add`).

### Parity table (installed CLAUDE.md, engineering surface)
- **Slash-only list (14): correct** — none of `/capture /challenge /compose
  /convention /decide /discover /drive /mock /prime /release /retro /review
  /scrub /split` exist as top-level `hero` subcommands.
- **CLI-only list: correct** — `hero status/search/ask/list/queue/spec
  verify/spec score/diff/drift` all exist.
- **"Both" list defects:** omits `/resume` <-> `hero resume`, `/blocked` <->
  `hero blocked`, `/peer` <-> `hero peer` (all genuinely both-surface); omits
  `/roadmap-review` from slash-only. Two listed "Both" rows conflate
  different semantics: slash `/import` = tracker import (`hero sync import`)
  while CLI `hero import` = knowledge-base URL/file ingestion; slash
  `/handoff` = NEXT.md refresh while CLI `hero handoff` = cross-repo drop.
  -> F22 (S2).
- **Provenance:** table exists only inside the installed managed region, not
  in the pack source that regenerates it — the next `hero install` run
  against the current pack silently drops it. -> part of F22.

### PM (install = pm + core)
- **Phantom slash routes (11+ rows):** `/interview`, `/capacity`,
  `/plan-cycle`, `/plan-sprint`, `/plan-iteration`, `/standup`,
  `/scrub roadmap`, `/scrub intake`, `/scrub specs`, `/diagnose`, `/search`,
  `/review` — none exist in the pm+core merged command set (`/scrub`,
  `/diagnose`, `/review` are engineering-only; the rest exist nowhere). The
  commands README says these are "v1.5+" — the routing table routes to them
  today. -> F2 (S1).
- **Phantom CLI routes:** `hero new feature|bug|epic|initiative` (no `hero
  new`; scaffolding is `hero spec new` / `hero design`), `hero event ...`
  (no such subcommand; the MCP tool `hero_event` exists), `hero active
  register|list` (no such subcommand), `hero queue --owner engineering
  [--status ready]` (`hero queue` has no `--owner` or `--status` flags),
  and `hero import` described as "import issues from tracker" (that is
  `hero sync import`; `hero import` ingests URLs/files). -> F3 (S1).
- **Unlisted shipped commands:** core's `/capture /check /convention /docs
  /drive /hero /import /prime /resume /scan` install with the pm pack but
  have no routing row.

### Sales (install = sales + core)
- **Phantom slash routes: none** — all 10 routed commands exist.
- **Phantom CLI claims:** `hero pulse --week`, `hero forecast`,
  `hero read-spec <slug>`, `hero search --match "stale"` — none exist
  (verified `unknown command` / no `--match` flag). -> F10.
- Commands/Agents/Skills reference tables: all 7+5+7 targets exist. OK.

### Chat
- **No AGENTS.md at all**, and the domain is not installable: `content.go`
  embeds only engineering/sales/pm; `DomainFS("chat")` errors and
  `AvailableDomains()` = `{engineering, sales, pm}`. If it were wired,
  `loadPackAgentsMdBody` would fall back to the engineering routing table —
  whose routes point at commands the chat pack doesn't have. -> F9.

---

## C. Findings (ordered by severity)

### [S1] F1: core /import instructs a CLI command that does something else entirely — core/commands/import.md
- Surface/dimension: command / 4 (freshness), 3 (actionability)
- Evidence: "Run the appropriate `hero import` command: Preset: `hero import --preset <name>` / Raw JQL: `hero import --jql \"<query>\"`". `hero import --help` shows it ingests a URL/file/directory into the knowledge base and has **no** `--preset`/`--jql` flags; tracker import is `hero sync import` (which has both flags). The engineering copy was already fixed to `hero sync import`; the core copy was not. Blast radius: every pm and sales install (neither pack overrides import.md).
- Fix shape: port the engineering copy's `hero sync import` invocations back to core/commands/import.md.

### [S1] F2: PM routing table routes to 11+ slash commands that don't install — domains/pm/AGENTS.md
- Surface/dimension: routing / 4 (freshness)
- Evidence: routes `/interview`, `/capacity`, `/plan-cycle`, `/plan-sprint`, `/plan-iteration`, `/standup`, `/scrub roadmap|intake|specs`, `/diagnose`, `/search`, `/review`. None exist in the pm+core merged command set (`/scrub`, `/diagnose`, `/review` are engineering-pack-only). domains/pm/commands/README.md itself says the `/capacity`, `/plan-*`, `/standup`, `/interview`, `/scrub` rows "ship in v1.5+".
- Fix shape: delete or clearly mark future rows; re-point `/diagnose` -> `pm-investigator` and `/review` -> `pm-reviewer` through commands that actually ship, or ship pm variants.

### [S1] F3: PM pack instructs five nonexistent CLI invocations — domains/pm/AGENTS.md, handoff.md, prioritize.md, release-notes.md, roadmap.md, triage.md
- Surface/dimension: routing+command / 4 (freshness)
- Evidence (each verified unknown command / missing flag against the binary):
  - `hero new feature|bug|epic|initiative` (AGENTS.md routing + vocabulary tables) — no `hero new`; scaffold is `hero spec new`.
  - `hero event handoff|decision_made ...` (AGENTS.md "Log significant events"; pm/handoff.md step 4; prioritize.md; release-notes.md; roadmap.md; triage.md) — no `hero event` subcommand (the MCP tool `mcp__hero__hero_event` is the real surface).
  - `hero active register|list` (AGENTS.md "Survive context compaction") — no `hero active`.
  - `hero queue --owner engineering --status ready` (AGENTS.md + pm/handoff.md step 5) — `hero queue` has neither flag.
  - `hero import` described as tracker import (AGENTS.md CLI list) — that's `hero sync import`.
- Fix shape: swap in the real surfaces (`hero spec new`, `hero_event` MCP, unfiltered `hero queue`, `hero sync import`) or add the missing CLI verbs.

### [S1] F4: six core commands delegate to agents/skills that only exist in the engineering pack — core/commands/{check,decide,discover,retro,drive,hero}.md
- Surface/dimension: command / 4 (freshness); blast radius = every pm and sales install
- Evidence: core ships 4 agents (convention-author, documentation-engineer, project-context-builder, session-primer) and 14 skills. Yet: check.md delegates to `architecture-reviewer` and `dependency-analyst`; decide.md to `architecture-reviewer`, `brownfield-architect`, `greenfield-architect`; discover.md to `product-ideator`; retro.md to `feature-delivery-lead`/`platform-delivery-lead`; drive.md says "Load the `drive` skill" (engineering-only; core/skills has no drive); hero.md routes to `/design /diagnose /deliver /review /compose /release` (engineering-only commands). All resolve on engineering installs but dangle on pm/sales installs, where these core files are the live copies.
- Fix shape: move these commands into the engineering pack, or give them domain-neutral fallbacks ("delegate to the domain's delivery lead, else run directly").

### [S1] F5: /research examples invoke a nonexistent `hero research` CLI — domains/sales/commands/research.md
- Surface/dimension: command / 4 (freshness)
- Evidence: "**Company research** — `hero research \"Acme Corp\"` or `hero research acme-corp`" (all four research-type bullets). `hero research` -> unknown command. These should be `/research ...` slash examples.
- Fix shape: s/hero research/\/research/ in the four example bullets.

### [S1] F6: /strategize suggests `/review <slug>` which doesn't install with sales — domains/sales/commands/strategize.md
- Surface/dimension: command / 4 (freshness)
- Evidence: "Run `/review <slug>` to have the deal reviewed for gaps" — `/review` is engineering-pack-only; sales+core has no review command and no reviewer agent.
- Fix shape: drop the suggestion or point at an agent that ships with sales.

### [S1] F7: /triage routes ambiguous intakes "via /diagnose" which doesn't install with pm — domains/pm/commands/triage.md
- Surface/dimension: command / 4 (freshness)
- Evidence: "invoke `pm-investigator` first via `/diagnose`" — no /diagnose in pm+core. `pm-investigator` itself exists; the invocation path doesn't.
- Fix shape: invoke `pm-investigator` directly (it's an agent; no command shim needed).

### [S1] F8: /sprint execute mode instructs bypassing the verify gate /deliver declares mandatory — domains/engineering/commands/sprint.md
- Surface/dimension: command / 3 (actionability — contradictory instructions), 1 (overlap)
- Evidence: sprint.md step 4c: "If clean: run `hero spec complete <spec-path>`". deliver.md: "**`hero spec verify` is the only path to `completed`** ... Do not edit `status: completed` in the frontmatter directly" (and the AGENTS.md closing-gate rule). A sprint-execute run following sprint.md skips the ledger/audit/coverage/build gates. Execute mode also duplicates `/drive`, which post-dates it.
- Fix shape: replace step 4c with `hero spec verify <slug>`; consider deleting execute mode in favor of `/drive`.

### [S1->S2] F9: the entire chat pack is dead content — domains/chat/commands/* (6 files)
- Surface/dimension: command / 1 (earns its place), 4 (freshness)
- Evidence: `content.go` embeds only engineering/sales/pm (go:embed lines 17-24); `DomainFS` has no "chat" case; `AvailableDomains()` = {engineering, sales, pm}. No AGENTS.md, agents/, or skills/ in domains/chat/. The 6 command files cannot be installed by any `hero install --domain` value. Graded S2 (waste, not active harm) only because they're unreachable — the moment chat is wired, the missing AGENTS.md makes installs fall back to the engineering routing table, whose routes nearly all point at commands the chat pack lacks (pre-built S1).
- Fix shape: wire the domain (embed + DomainFS case + AGENTS.md) or move the files out of domains/.

### [S2] F10: sales routing file instructs four nonexistent CLI surfaces — domains/sales/AGENTS.md
- Surface/dimension: routing / 4 (freshness)
- Evidence: `hero pulse --week`, `hero forecast`, `hero read-spec <slug>` (Session Start section 2 and Key CLI Commands) — all unknown commands; `hero search --match "stale"` — no `--match` flag.
- Fix shape: replace with real surfaces (`hero status`, `hero_pulse` MCP, `mcp__hero__hero_read_spec`, `hero search "stale"`), or implement the verbs.

### [S2] F11: core<->engineering duplicate set has drifted accidentally, not by specialization — core/commands/* vs domains/engineering/commands/*
- Surface/dimension: command / 2 (duplication), 4 (freshness)
- Evidence: 17 shared filenames. Identical: blocked, check, discover, docs, drive, hero, retro, scan, why (9). Diverged: capture, convention, note (subproject-workspace guard added to eng only), decide (hero_anchor tripwire step eng only), handoff (QUEUE.md refresh step eng only), import (`hero sync import` fix eng only — F1), prime (queue-surfacing step eng only), resume (hero_queue/QUEUE.md guidance eng only). git log: every diverged core copy frozen at 982742d (v0.8.0) while engineering copies were updated (92c94aa and later). Direction of drift is uniformly "engineering improved, core forgot," including a genuine bug fix (import) — drift, not domain specialization.
- Fix shape: one-time re-sync core <- engineering for the domain-neutral improvements, plus a parity test, or de-duplicate by deleting the 9 identical engineering copies (core already provides them through OverlayFS fallback).

### [S2] F12: deliver.md at 2,673 words (5x the p90) — extract what its skills already own — domains/engineering/commands/deliver.md
- Surface/dimension: command / 2 (token efficiency)
- Evidence + what to cut:
  - **Definition of done stated three times**: its own section, again in the `--supervised` row of the mode table, again in the "MUST run `hero spec verify`" block (~200 redundant words). Keep one.
  - **Cold audit pass (step 6, ~600 words)** restates the `delivery-audit` skill's block formats (AUDIT_VERDICT/HEADLINE/HIGHLIGHTS), the always-vs-conditional display policy, and the signal-preservation rule. Keep the invocation + verdict routing (~150 words); the block semantics live in the skill.
  - **Completion Ledger validation (~350 words across step 5 and batch step 4)** restates `engineer.md` "Closing output" and the `agent-reliability` persistence rule it already cites. Keep the DONE/PARTIAL/SKIPPED routing rules once.
  - **Batch mode vs Queue mode** share ~70% of their loop. Merge into one multi-spec mode with a queue-ordering variant.
  - Estimated recovery: ~1,100-1,300 words without losing a rule.

### [S2] F13: peer.md restates the skill it mandates loading — domains/engineering/commands/peer.md
- Surface/dimension: command / 2 (token efficiency)
- Evidence: 979 words (p90 530). Opens with "load `skills/cross-repo-peering/SKILL.md` — the skill carries the decision tree, the prompt-composition rules, and the anti-patterns," then re-carries them: steps 3-4 prompt-composition guidance and budget defaults, the "What NOT to do" list, and the example dispatch block all duplicate skill content. Keep: sub-action detection table, pre-flight `hero peer list`, trail-discipline checklist. ~400 words recoverable.

### [S2] F14: /release pre-flight is a hero-repo-ism shipped to every engineering install — domains/engineering/commands/release.md
- Surface/dimension: command / 6 (context-agnosticism), 3 (actionability)
- Evidence: "Before assessing release readiness, **always run `hero docs check`** ... fix README.md and GETTING-STARTED.md ... Counts, tables, and reference sections must match." `hero docs check --help`: compares README numeric claims ("22 commands", "33 specialist agents") against agents/, commands/, skills/ directories — the Hero source repo's own layout. User projects have no GETTING-STARTED.md and no root agents/commands/skills dirs; the mandatory pre-flight errors or reports noise on every non-hero repo.
- Fix shape: gate the pre-flight on the dirs existing, or keep it hero-repo-local.

### [S2] F15: /prime and /resume are two session-start context loaders with no disambiguation — core/commands/prime.md, core/commands/resume.md (and eng copies)
- Surface/dimension: command / 1 (earns its place)
- Evidence: prime.md: "Load session context ... read your handoff briefing ... run `hero recap` ... be the `session-primer` agent." resume.md: "Run `hero resume` ... at the start of every fresh session ... run `hero resume` unconditionally." Both claim the session-start slot; resume claims it unconditionally, prime overlaps 3 of its 4 steps (NEXT.md read, recent activity, orientation). Neither file mentions the other; a session cannot pick between them from the descriptions alone, and the natural-language trigger sets collide.
- Fix shape: fold prime's `hero check --reconcile` and queue-surfacing into resume and retire one, or give prime a distinct trigger and state the split in both files.

### [S2] F16: /hero router's workflow list is stale — routes to 14 of 30 shipped commands, phantoms on non-eng installs — core/commands/hero.md, domains/engineering/commands/hero.md (identical)
- Surface/dimension: command / 4 (freshness), 1 (overlap with AGENTS.md routing table)
- Evidence: "Available workflows" lists 14 slash commands; missing `/mock`, `/drive`, `/sprint`, `/import`, `/peer`, `/challenge`, `/split`, `/scrub`, `/prime`, `/resume`, `/roadmap-review` — the router cannot route to over half the engineering pack (notably /mock, whose routing the AGENTS.md calls confabulation-prone). Being in core, the same file ships to pm/sales installs where 6 of its 14 routes are phantoms (F4). It also duplicates the AGENTS.md routing table wholesale.
- Fix shape: regenerate the list from the installed command set per domain, or slim /hero to defer to the instructions-file routing table + `hero do`.

### [S2] F17: diagnose.md's parallel-batch protocol is a 400-word inline skill — domains/engineering/commands/diagnose.md
- Surface/dimension: command / 2 (token efficiency)
- Evidence: 898 words. "Parallel batch mode" + "After all agents complete" (6 numbered rules, verification loop, summary-table template) are generic multi-agent batch discipline paralleling deliver.md's batch mode; the single-bug path (the common case) is ~350 words and fine.
- Fix shape: extract batch/parallel protocol to a shared batch-discipline skill; keep the selection rules in the command.

### [S2] F18: pm /handoff duplicates the handoff-protocol skill and uses a spec path the pack doesn't declare — domains/pm/commands/handoff.md
- Surface/dimension: command / 2 (token efficiency), 7 (format consistency)
- Evidence: 776 words. Loads `handoff-protocol` then re-enumerates the six pre-flight gates, five failure modes, and the "what does NOT happen" list the skill owns. Path inconsistency: reads/writes `.hero/planning/specs/<slug>/spec.md` (twice) while domains/pm/AGENTS.md declares `.hero/planning/features/` (plus epics/initiatives/prds/intake) and no planning/specs/ exists in the layout.
- Fix shape: keep argument contract + owner-flip step (+ F3 fix) in the command; defer gates/failure modes to the skill; correct the path.

### [S2] F19: pm commands README claims reused commands PM installs don't get — domains/pm/commands/README.md
- Surface/dimension: command (doc-only) / 4 (freshness)
- Evidence: "Reused (cross-domain / core)" lists `/search` (exists nowhere as a slash command in any pack) and `/deliver` ("picked up by engineering's engineer agent") — /deliver is engineering-pack-only and does not install with --domain pm. `/why` and `/note` claims are correct.
- Fix shape: correct the reused list to what pm+core actually ships.

### [S2] F20: release-notes v1 fallback loads a skill its own parenthetical says doesn't exist yet — domains/pm/commands/release-notes.md
- Surface/dimension: command / 4 (freshness), 3 (actionability)
- Evidence: "In v1, falls through to `pm-delivery-lead` loading the `stakeholder-communication` skill directly. (Note: `stakeholder-communication` is v1.5 — until it lands, the v1 fallthrough uses a baseline template.)" — instructs loading a skill (absent from domains/pm/skills/) then retracts it; "a baseline template" is unlocatable for a cold agent.
- Fix shape: state the v1 behavior only; park the v1.5 note in the README.

### [S2] F21: sales commands filter `hero search --type` on non-spec-types — domains/sales/commands/{prospect,strategize,research}.md, domains/sales/AGENTS.md
- Surface/dimension: command / 4 (freshness), 3 (actionability)
- Evidence: `hero search --type playbook`, `--type battlecard`, `--type prospect`, `--type knowledge` (prospect.md x3, strategize.md x2, research.md x2, AGENTS.md x5). `--type` filters the FTS5 index by spec type; registered types are the core nine + sales' `deal` (domains/sales/spec-types/deal.yaml). playbook/battlecard/prospect aren't spec types — battlecards are written to `.hero/knowledge/battlecards/<competitor>.md` (research.md's own save step), so these lookups return empty and the agent concludes no intel exists.
- Fix shape: use plain-text queries (`hero search "battlecard <competitor>"`) or register the types; make save and lookup locations agree.

### [S2] F22: slash/CLI parity table exists only in the installed CLAUDE.md, and is incomplete — CLAUDE.md (managed region) vs domains/engineering/AGENTS.md
- Surface/dimension: routing / 4 (freshness), 7 (consistency)
- Evidence: grep for "Slash-only" across the repo hits only CLAUDE.md:49 — the pack source and the Go fallback have no parity table, so the next install regeneration drops it. Content defects: "Both" omits `/resume`, `/blocked`, `/peer` (real CLI twins exist); slash-only omits `/roadmap-review`; the `/import` and `/handoff` "Both" rows pair slash and CLI commands with different semantics (section B). All "CLI-only" entries verified real. Related: core/handoff.md + eng/handoff.md cite `skills/next-md.md` / `skills/kickoff-prompt.md` — actual layout is `skills/<name>/SKILL.md` (folded here as the same stale-path family; see also F24 note).
- Fix shape: move the corrected table into domains/engineering/AGENTS.md so it survives regeneration; annotate the two different-semantics rows; fix the two skill paths.

### [S3] F23: unscoped Claude-specific tool vocabulary in three engineering commands — diagnose.md, deliver.md, mock.md
- Surface/dimension: command / 6 (harness-agnosticism)
- Evidence: diagnose.md "launch multiple **Task agents**"; deliver.md step 6 "Invoke a **general-purpose subagent**" (a Claude Code agent type name); mock.md "When the **Agent tool** completes". Contrast: engineering/handoff.md scopes correctly ("important for harnesses (Claude Code) that can't pop a terminal"). Copilot/codex/cursor/generic targets have no Task/Agent tool or general-purpose agent type.
- Fix shape: harness-neutral phrasing ("spawn subagents / a fresh subagent via your harness's delegation mechanism").

### [S3] F24: engineering routing table leaves 11 shipped commands unrouted — domains/engineering/AGENTS.md
- Surface/dimension: routing / completeness (rubric classification notes)
- Evidence: section B list (blocked, capture, challenge, hero, peer, prime, resume, roadmap-review, scrub, split, why). `/resume`'s own file demands NL auto-routing ("agents should auto-route these to /resume") that only works if the instruction file routes it.
- Fix shape: add rows at least for the user-facing ones (/resume, /scrub, /split, /peer, /why, /blocked).

### [S3] F25: three pm commands lead with agents that don't exist yet — domains/pm/commands/{metrics,pitch,release-notes,refine}.md
- Surface/dimension: command / 4 (freshness), 5 (triggering)
- Evidence: metrics.md -> `metrics-analyst` (P1); pitch.md -> `pitch-author` (P1); release-notes.md -> `stakeholder-communicator` (P1); refine.md -> `epic-framer` (v1.5). None exist in domains/pm/agents/ (verified by ls). Each is labeled with its fallback, so not actively misleading, but the primary instruction of each workflow is a dangling reference a cold agent must parse past.
- Fix shape: lead with the v1 agent, footnote the P1 upgrade.

### [S3] F26: pm /discover writes research under a path the pack structure doesn't declare — domains/pm/commands/discover.md
- Surface/dimension: command / 7 (format consistency)
- Evidence: "Interview guides -> written to `.hero/planning/roadmap/<slug>/research/interview-<n>.md`" — pm AGENTS.md Project Structure declares planning/{features,epics,initiatives,prds,intake}; no planning/roadmap/.
- Fix shape: align on `.hero/planning/initiatives/<slug>/research/`.

### [S3] F27: resume.md padding — trigger-phrase list and pep-talk section — core/commands/resume.md, domains/engineering/commands/resume.md
- Surface/dimension: command / 2 (token efficiency)
- Evidence: 520/554 words vs p90 530. "What to say" lists 18 trigger phrases where 6 establish the pattern; "Why this matters" (~80 words) is motivation, not instruction; the closing "run unconditionally ... >=99% useful and never wrong" paragraph repeats When-to-use bullet 1. ~150 words recoverable.
- Fix shape: trim triggers to a representative set; cut "Why this matters" to one line.

### [S3] F28: sales forecast/pipeline rely on unverified comma-list --status filter — domains/sales/commands/{forecast,pipeline}.md
- Surface/dimension: command / 4 (freshness)
- Evidence: `hero search --type deal --status "qualifying,demo,proposal,negotiation"` — `hero search --help` documents `--status string` as a single-status FTS5 filter; nothing documents comma-list support, so the filter likely matches zero rows silently. (forecast.md's `hero sync import --type deal --status open` is flag-valid.)
- Fix shape: verify comma-list behavior; if unsupported, loop per-status or drop the filter.

### [S3] F29: chat commands hardcode a specific chat client's internals — domains/chat/commands/{ask-corpus,space}.md
- Surface/dimension: command / 6 (harness-agnosticism)
- Evidence: ask-corpus.md instructs `semantic_search` / `read_file` (no install target exposes tools by those names); space.md instructs "Use the `SpaceStore` API ... if running outside the GPUI shell" — a private client implementation detail, unscoped. Moot while F9 stands; must be fixed before the pack is wired.
- Fix shape: name capabilities, not tool identifiers; scope the SpaceStore path to the specific client.

---

## D. Duplicate classification detail (task 4)

| Name | Copies | Status | Judgment |
|---|---|---|---|
| blocked, check, discover, docs, drive, hero, retro, scan, why | core + eng | byte-identical | pure duplication — eng copy redundant given OverlayFS fallback |
| capture, convention, note | core + eng | diverged (+subproject guard in eng) | accidental drift — guard is domain-neutral |
| decide | core + eng | diverged (+hero_anchor tripwire step in eng) | accidental drift — tripwires are core machinery |
| handoff, prime, resume | core + eng | diverged (+QUEUE.md steps in eng) | accidental drift — QUEUE.md is core machinery |
| import | core + eng | diverged (eng fixed hero import -> hero sync import) | **accidental drift with a live bug in core** (F1) |
| discover, handoff | core + pm | fully different content | intentional domain specialization (pm overrides via overlay) OK |
| capture, discover, note, why | core + chat | fully different content | intentional rewrite for chat, but pack is dead (F9) |

git evidence: diverged core copies last touched at 982742d (v0.8.0); engineering copies updated at 92c94aa ("sync root -> domains/engineering") and later. Every divergence direction is engineering-forward.

---

## E. Dimension notes with no finding

- **Triggering (dim 5):** frontmatter description quality across the 70
  files that have one is good — concrete, differentiated, non-circular
  (best-in-class: resume.md, mock.md, roadmap-review.md). No flags beyond
  the prime/resume overlap (F15).
- **Format (dim 7):** 70/72 command files carry exactly `description:`
  frontmatter (consistent); the 2 exceptions are the doc-only READMEs.
- **Verified-clean CLI claims** (checked both sides, no finding): hero next
  checkpoint / ask / path, hero load alias, hero graph reingest, hero check
  --reconcile, hero scan --dry-run|--code|--force, hero why --depth|--edges,
  hero index --if-stale -q, hero queue write -q, hero goal --emit|--check,
  hero spec mock detect | verify --skip-tests|--json|--force | complete |
  lint | score | deliver, hero sync import --preset|--jql|--type|--status,
  hero sync attach|comment|push, hero relevant, hero size --check,
  hero dashboard, hero do, hero peer call --mode|--related-spec|--reason,
  hero handoff --reason|--title|--type, hero admin repos add.
