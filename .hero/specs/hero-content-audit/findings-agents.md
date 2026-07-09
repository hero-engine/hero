# Findings — surface pass 3a (agents)

Pass: agents (58 files) · Auditor: content-audit subagent · Date: 2026-07-05 · Base SHA: `bc86ad9`

Verification methodology: every CLI reference checked against `hero --help` /
`hero <cmd> --help` (binary at `~/go/bin/hero`); every skill reference checked
against `core/skills/` + `domains/*/skills/`; every command reference checked
against `core/commands/` + `domains/*/commands/`; frontmatter consumption
checked against `internal/install/content.go`, `internal/install/render.go`,
and a repo-wide grep of `internal/` for each field name.

---

## (a) Coverage table — 58/58 files read

| # | File | Verdict |
|---|------|---------|
| 1 | core/agents/convention-author.md | clean |
| 2 | core/agents/documentation-engineer.md | clean |
| 3 | core/agents/project-context-builder.md | flagged |
| 4 | core/agents/session-primer.md | flagged |
| 5 | domains/engineering/agents/api-engineer.md | flagged |
| 6 | domains/engineering/agents/architecture-reviewer.md | flagged |
| 7 | domains/engineering/agents/brownfield-architect.md | flagged |
| 8 | domains/engineering/agents/comment-scrubber.md | clean |
| 9 | domains/engineering/agents/convention-author.md | flagged |
| 10 | domains/engineering/agents/database-engineer.md | flagged |
| 11 | domains/engineering/agents/deadcode-scrubber.md | flagged |
| 12 | domains/engineering/agents/debug-investigator.md | clean |
| 13 | domains/engineering/agents/dedup-scrubber.md | flagged |
| 14 | domains/engineering/agents/defensive-scrubber.md | clean |
| 15 | domains/engineering/agents/dependency-analyst.md | flagged |
| 16 | domains/engineering/agents/dependency-scrubber.md | flagged |
| 17 | domains/engineering/agents/design-reviewer.md | clean |
| 18 | domains/engineering/agents/devops-engineer.md | flagged |
| 19 | domains/engineering/agents/documentation-engineer.md | flagged |
| 20 | domains/engineering/agents/engineer.md | flagged |
| 21 | domains/engineering/agents/feature-delivery-lead.md | flagged |
| 22 | domains/engineering/agents/functional-qa-engineer.md | clean |
| 23 | domains/engineering/agents/greenfield-architect.md | flagged |
| 24 | domains/engineering/agents/integration-engineer.md | flagged |
| 25 | domains/engineering/agents/issue-tracker.md | flagged |
| 26 | domains/engineering/agents/legacy-scrubber.md | flagged |
| 27 | domains/engineering/agents/migration-engineer.md | clean |
| 28 | domains/engineering/agents/performance-engineer.md | flagged |
| 29 | domains/engineering/agents/platform-delivery-lead.md | flagged |
| 30 | domains/engineering/agents/pr-reviewer.md | clean |
| 31 | domains/engineering/agents/product-ideator.md | clean |
| 32 | domains/engineering/agents/project-context-builder.md | flagged |
| 33 | domains/engineering/agents/release-engineer.md | flagged |
| 34 | domains/engineering/agents/roadmap-reviewer.md | flagged |
| 35 | domains/engineering/agents/security-reviewer.md | clean |
| 36 | domains/engineering/agents/session-primer.md | flagged |
| 37 | domains/engineering/agents/test-architect.md | clean |
| 38 | domains/engineering/agents/type-scrubber.md | flagged |
| 39 | domains/engineering/agents/ui-designer.md | flagged |
| 40 | domains/pm/agents/README.md | doc-only (freshness clean) |
| 41 | domains/pm/agents/discovery-researcher.md | flagged |
| 42 | domains/pm/agents/duplicate-detector.md | clean |
| 43 | domains/pm/agents/handoff-coordinator.md | flagged |
| 44 | domains/pm/agents/intake-triager.md | flagged |
| 45 | domains/pm/agents/pm-delivery-lead.md | flagged |
| 46 | domains/pm/agents/pm-investigator.md | flagged |
| 47 | domains/pm/agents/pm-reviewer.md | flagged |
| 48 | domains/pm/agents/prd-author.md | flagged |
| 49 | domains/pm/agents/prioritization-strategist.md | clean |
| 50 | domains/pm/agents/product-strategist.md | flagged |
| 51 | domains/pm/agents/roadmap-curator.md | flagged |
| 52 | domains/pm/agents/story-writer.md | flagged |
| 53 | domains/sales/agents/README.md | doc-only (freshness clean) |
| 54 | domains/sales/agents/buyer-researcher.md | flagged |
| 55 | domains/sales/agents/competitive-intel.md | flagged |
| 56 | domains/sales/agents/deal-strategist.md | flagged |
| 57 | domains/sales/agents/forecast-analyst.md | flagged |
| 58 | domains/sales/agents/qualification-analyst.md | flagged |

---

## (b) Roster overlap map

### Engineering (35 agents — the prime suspect)

| Agent A ↔ Agent B | Why they collide | Recommendation |
|---|---|---|
| `feature-delivery-lead` ↔ `platform-delivery-lead` | Both coordinate `/design` + `/deliver` with the identical 20-agent task allowlist. Descriptions distinguish scope (features/bugs vs migrations/platform), but the delivery *process* must be identical — and it isn't: platform is a stale fork missing the Completion Ledger validation, cold-audit pass, `hero spec verify` gate, sizing-nudge schedule, autopilot modes, and challenge handling that feature got. Step 13 of platform directly contradicts feature's step 19 (see S1-2). | **Merge** — make platform a thin delta (sequencing/rollback emphasis) on the shared delivery procedure, or fold into feature-delivery-lead with a platform mode. Maintaining two 900–2,700-word copies of the delivery pipeline guarantees drift; it has already happened. |
| `engineer` ↔ `api-engineer` / `database-engineer` / `integration-engineer` / `performance-engineer` / `devops-engineer` / `release-engineer` | The thin specialists (136–154 words each) are "engineer + load skill X" stubs: their entire body is the load-skills list plus 4 aspirational rules. `engineer.md` startup step 4 already instructs loading `api-design-and-contracts`, `integration-boundaries`, `performance-optimization`, `security-review` "if the task involves those domains" — the same skills the stubs exist to load. A session choosing between `engineer` and `api-engineer` for "change this endpoint" gets no differentiated behavior. | **Cut or consolidate.** Keep the names only if delivery-lead delegation-by-name is worth 6 near-empty files; otherwise cut the stubs and let `engineer` + stack/domain skills cover it. If kept, each stub needs at least one behavior `engineer` doesn't have. `database-engineer` is the strongest keep (operational-migration stance); `devops-engineer`/`release-engineer` have distinct subject matter — keep; `api/integration/performance` are the weakest. |
| `deadcode-scrubber` ↔ `legacy-scrubber` | Both hunt obsolete code. Deadcode's targets include unreachable paths; legacy's checklist includes "feature flags or version checks with only one active branch," "compatibility shims," "functions that exist only to support an old interface" — all of which deadcode's static-analysis pass also surfaces once the old path is unreferenced. A session with "remove the old v1 fallback" cannot pick from descriptions alone (`unused… unreachable code paths` vs `deprecated, legacy, and fallback code`). | **Merge** into one scrubber with two detection passes (reference-count vs intent-comment), or sharpen the boundary in both descriptions: deadcode = provably unreferenced; legacy = referenced but superseded. |
| `dedup-scrubber` ↔ `type-scrubber` | Direct overlap on duplicated type definitions. dedup process step 1: "Similar struct/class definitions that could share a common base"; type-scrubber "Type consolidation" section: "Identify duplicates or near-duplicates across packages… Consolidate." Same work item, two agents. | **Cut** the type-consolidation half of `type-scrubber` (leave it weak-type strengthening only) or the struct/class bullet in `dedup-scrubber`. |
| `architecture-reviewer` ↔ `design-reviewer` | Both are pre-delivery review gates over a proposed design. architecture-reviewer reviews "architecture proposals, design docs, migration plans"; design-reviewer reviews "spec designs for completeness, feasibility." A spec with architectural content matches both descriptions. In practice design-reviewer is a spec-readiness lint (EARS ratio, Changes present) and architecture-reviewer is a technical critique. | **Keep both**, but the descriptions should carry the split: design-reviewer = "is this spec deliverable as written" (structure gate); architecture-reviewer = "is this technical approach sound" (judgment gate). |
| `dependency-analyst` ↔ `dependency-scrubber` | Content is distinct (external library health vs internal import-graph structure) but the names are a coin flip for a session told "clean up our dependencies." | **Keep**, rename one (e.g. `import-graph-scrubber` or fold scrubber into the code-scrub family naming it already belongs to). S3. |
| `brownfield-architect` ↔ `greenfield-architect` | Cleanly split by "existing system vs new system." Not a collision — but ~40% of each body is the same architectural stance / scale-readiness / strict-rules text, also present in `architecture-reviewer`, all three of which *also* say "load `architecture-principles` — it contains the shared architectural stance." | **Keep both agents; cut the triplicated stance** (see S2-5). |
| `convention-author`, `documentation-engineer`, `project-context-builder`, `session-primer` (engineering) ↔ same names in `core/agents/` | Byte-identical duplicates except session-primer, which has silently diverged (core copy now stale — see S1-1). Because install overlays domain over core, the engineering copies shadow core exactly; identical copies add nothing but a second place to forget. | **Cut** the three identical engineering copies (core already installs everywhere); session-primer proves the drift hazard is real, not theoretical. |
| `product-ideator` ↔ `roadmap-reviewer` | No collision — ideation vs sizing-drift triage. Listed to record it was checked. | Keep. |

### PM (12 agents)

| Agent A ↔ Agent B | Why | Recommendation |
|---|---|---|
| `pm-investigator` ↔ `discovery-researcher` | Both "reduce uncertainty before authoring." Boundary is stated (investigator = classify an ambiguous inbound signal; researcher = design/synthesize outbound research) and each names the other in its routing table. Descriptions are distinguishable. | Keep. |
| `product-strategist` ↔ `prioritization-strategist` ↔ `roadmap-curator` | Framing / ranking / state-reconciliation — three distinct verbs on the same artifact set, each explicitly disclaiming the others' jobs. | Keep. |
| `prd-author` ↔ `story-writer` | PRD vs feature-spec authoring; clean split. | Keep. |
| (ghost roster) `pitch-author`, `epic-framer`, `dependency-mapper`, `risk-curator`, `metrics-analyst`, `cycle-planner`, `capacity-planner`, `stakeholder-communicator` | Referenced as delegation targets by pm-delivery-lead (and by prd-author, story-writer, product-strategist, prioritization-strategist) but **do not exist** in the pack (README: "P1/P2 … ship in v1.5+"). | See S1-3. |

### Sales (5 agents)

Five agents, five distinct jobs (strategy / qualification / forecast / competitive / research); descriptions are mutually exclusive; `deal-strategist` is the coordinator and names the others. No overlap findings.

---

## (c) Frontmatter schema determination (task 3)

Engine consumption (verified in code):

- **`name`, `description`** — load-bearing everywhere: consumed by `renderCodexToml`/`renderCopilotPromptFile` (`internal/install/render.go:152-200`) and required by Claude Code's subagent loader (`internal/install/target_claude.go:11`).
- **`domains:`** — load-bearing in the engine: `internal/install/content.go:56-89` filters agents out of installs whose active domain isn't listed (absent = all domains). **But inert in practice today**: install merges core + exactly one domain pack (`internal/cli/install.go:188-200`, `OverlayFS(domainFS, CoreFS())`), so a pack file only ever installs when its own domain is active — `domains: [engineering]` on an engineering-pack agent can never filter anything. The field only matters on `core/agents/*`, where **no file sets it**. Present in 11/58 files with no discernible pattern (8 engineering, 5 sales... actually 6+5; 0 core, 0 pm).
- **`mode`, `temperature`, `permission`** — never read by the engine; they are opencode agent-config fields passed through verbatim (harmless unknown keys on Claude Code; dropped on codex/copilot which render only name/description).
- **`color`** — opencode/Claude Code cosmetic; engine-inert.
- **`role:`** — consumed by **nothing**: not in the engine (repo grep of `internal/` finds no reader), not an opencode agent field, not a Claude Code field. Dead metadata in 17 files.

Canonical set: `name, description, mode, temperature, color, permission` (matches pm/agents/README.md's own statement of the pack shape). Outliers: `role:` in 17 files (dead), `domains:` in 11 files (inert as placed), `ui-designer.md` with only `name, description` (missing the opencode execution fields every sibling has — it runs with default permissions).

---

## (d) Findings

### [S1] Core session-primer instructs nonexistent CLI flags; silently diverged from its engineering twin — core/agents/session-primer.md
- Surface/dimension: agent / 4 (freshness), 7 (format consistency)
- Evidence: lines 16–17: "Run `hero status --delivering`" / "Run `hero status --claimed`". `hero status --help` has only `--all`, `--horizon`. The engineering copy (domains/engineering/agents/session-primer.md:16-17) was already fixed to `hero list --status delivering` / `hero list --mine <user>` (both flags verified in `hero list --help`), and its line 21 fixed `hero read_spec` → `hero_read_spec`; the core copy never got the fix. Blast radius: the core copy is what installs for **pm, sales, and chat** domain workspaces.
- Fix shape: port the engineering copy's corrections to core, then delete the engineering duplicate (overlay makes it redundant once identical).

### [S1] platform-delivery-lead tells the lead to hand-edit `status: completed` and move the spec, contradicting the verify-gate flow — domains/engineering/agents/platform-delivery-lead.md
- Surface/dimension: agent / 4 (freshness), 1 (earns its place)
- Evidence: line 79 (step 13): "On completion, move the spec from `planning/` to `specs/` and update its status to `completed`." feature-delivery-lead.md:136 (step 19) states the current doctrine: "**Do not edit `status: completed` directly** — `hero verify` checks four gates (ledger, audit, coverage, tests) and flips status + archives only when all pass." platform-delivery-lead also has none of the Completion Ledger validation (feature step 17), cold-audit pass (step 18), or `hero spec verify` gate — a fork frozen at an older delivery process. Any platform-scoped `/deliver` bypasses all four delivery gates.
- Fix shape: replace step 13 with the verify-gated closing sequence (or merge the two leads per the overlap map).

### [S1] pm-delivery-lead's allowlist and delegation table route to 8 agents that don't exist — domains/pm/agents/pm-delivery-lead.md
- Surface/dimension: agent / 4 (freshness), 3 (actionability)
- Evidence: frontmatter `permission.task` allowlists `pitch-author`, `epic-framer`, `dependency-mapper`, `risk-curator`, `metrics-analyst`, `cycle-planner`, `capacity-planner`, `stakeholder-communicator` (lines 22–29); the step-3 specialist table (lines 82–94) routes 8 of its 19 rows to them. None exist in `domains/pm/agents/` (README.md: "P1 / P2 agents … ship in v1.5+"). Same ghosts referenced by prd-author.md:162 (`metrics-analyst`), product-strategist.md:61, story-writer.md:75,176 (`epic-framer`, `dependency-mapper`), prioritization-strategist.md:58 (`metrics-analyst`, `capacity-planner`). A session following the table delegates into a void.
- Fix shape: trim table + allowlist to the shipped 12; where a row has no owner, name the fallback (usually pm-delivery-lead itself or a skill).

### [S1] handoff-coordinator's "brand interaction" workflow is built on surfaces that don't exist — domains/pm/agents/handoff-coordinator.md
- Surface/dimension: agent / 4 (freshness), 3 (actionability)
- Evidence (each verified against the binary/tree):
  - Line 53: read the spec at `.hero/planning/specs/<slug>/spec.md` — no such directory convention (`.hero/planning/` contains `features/ bugs/ initiatives/ intake/ audits/`; `specs/` is the *archive* at `.hero/specs/`).
  - Lines 88–92: "you do not write `owner_history` directly; updating `owner` causes the history append" — false. The append mechanism is `hero spec set-owner <slug> <owner>` ("appends the transition to its owner_history timeline", `hero spec set-owner --help`); a raw frontmatter edit records no history row, so the agent's own step-2 verification ("read it back; it should show from: pm, to: engineering") will fail every time.
  - Line 104: `hero event handoff "…" --slug <slug>` — `hero event` is not a CLI command (`Error: unknown command "event"`); the MCP tool `hero_event` exists but the file shows shell syntax.
  - Line 121: `hero queue --owner engineering --status ready` — `hero queue --help` has neither `--owner` nor `--status`.
  - Pre-flight statuses `ready` / `drafted` / `refined` (lines 56–58) are not engine statuses (`internal/spec/spec.go:45-88`).
- Fix shape: rewrite the mechanics against real surfaces — `hero spec set-owner` for the flip, MCP `hero_event` (or add the CLI) for the stream, `hero list --status …` for verification, real planning paths and statuses.

### [S1] `hero event` CLI invoked across the PM pack but the command doesn't exist — domains/pm/agents/{roadmap-curator,intake-triager,pm-delivery-lead,handoff-coordinator}.md
- Surface/dimension: agent / 4 (freshness)
- Evidence: roadmap-curator.md:44 (`hero event decision_made "…" --slug <slug>`), intake-triager.md:45, pm-delivery-lead.md:118,144, handoff-coordinator.md:104. `hero --help` lists no `event` command; `hero event --help` errors. The MCP tool `mcp__hero__hero_event` exists — the capability is real, the invocation shape is wrong. (Also present in 5 PM command files — out of this pass's scope, noted for the commands pass.)
- Fix shape: one systemic edit — either reference the MCP tool by name or ship a `hero event` CLI shim.

### [S1] deal-strategist captures playbooks with a flag `hero note` doesn't have — domains/sales/agents/deal-strategist.md
- Surface/dimension: agent / 4 (freshness)
- Evidence: lines 203–205: `hero note "Pattern: [title]" --type playbook`. `hero note --help` flags: `--from` only.
- Fix shape: correct to the real note-capture path (per `note-capture` skill) or drop `--type`.

### [S2] PM pack lifecycle vocabulary doesn't match the engine's status set — domains/pm/agents/ (systemic: story-writer, prd-author, pm-delivery-lead, pm-reviewer, roadmap-curator, handoff-coordinator)
- Surface/dimension: agent / 4 (freshness), 3 (actionability)
- Evidence: story-writer.md:145-154 defines `drafted → refined → ready → delivering …`; prd-author.md:147-153 `drafting/refining/ready/handed-off/shipped`; pm-delivery-lead.md:117,142 adds `triaging/discovering/shaping/dropped`; roadmap-curator uses `candidate/committed/shipped`. Engine canon (`internal/spec/spec.go:45-88`): planning, in-review, delivering, completed, regressed, handed_off (underscore, peer-handoff semantics), triaged, promoted, rejected, merged, superseded. Freeform statuses will load but break every engine feature keyed on canonical statuses (`hero queue` readiness, `hero list --status`, verify, blocked).
- Fix shape: map the PM lifecycle onto engine statuses (or extend the engine set deliberately) in one place — a pm skill — and have the agents cite it.

### [S2] PM agents reference engineering-only skills/commands that a pm-domain install doesn't ship — domains/pm/agents/ (systemic)
- Surface/dimension: agent / 4 (freshness), 6 (harness/install-agnosticism)
- Evidence: install merges core + one pack only (`internal/cli/install.go:188-200`). Under `--domain pm`: `kickoff-prompt` (referenced by pm-delivery-lead.md:44, prd-author.md:30,143) exists only in `domains/engineering/skills/` — not installed; `/review` (pm-reviewer.md:32), `/scrub intake` (intake-triager.md:67), `/scrub roadmap` (roadmap-curator.md:27), `/deliver` (handoff-coordinator.md:123, story-writer.md:150) are engineering-pack commands — not installed. Also `/interview` (discovery-researcher.md:30) exists in **no** pack.
- Fix shape: promote genuinely shared skills (kickoff-prompt) to core, ship pm-side commands, or scope the references ("when the engineering pack is installed …"); delete `/interview` until it ships.

### [S2] project-context-builder is written for OpenCode only, installed on six targets — core/agents/project-context-builder.md + identical domains/engineering/agents/project-context-builder.md
- Surface/dimension: agent / 6 (harness-agnosticism)
- Evidence: line 13: "the project instructions that future **OpenCode** sessions will rely on"; lines 13, 25: `opencode.json` `instructions` entries as the mechanism. Installs unmodified to claude, cursor, copilot, codex, generic (`installFlat`, all six `target_*.go`). A Claude Code session gets told to optimize for OpenCode and edit opencode.json.
- Fix shape: neutral phrasing ("future agent sessions", "the harness's instruction-file mechanism") with harness-specifics as scoped examples.

### [S2] Three byte-identical core/engineering agent duplicates (and one diverged) — domains/engineering/agents/{convention-author,documentation-engineer,project-context-builder}.md
- Surface/dimension: agent / 1 (earns its place), 2 (token efficiency)
- Evidence: `diff` confirms convention-author, documentation-engineer, project-context-builder are byte-identical to their `core/agents/` twins; the overlay already installs core into every domain, so the copies contribute nothing except a second maintenance site. session-primer — the fourth twin — has already diverged (see S1 finding #1), demonstrating the failure mode.
- Fix shape: delete the three identical engineering copies; keep pack copies only when they intentionally differ.

### [S2] Architectural stance triplicated across the three architecture agents despite living in a shared skill — domains/engineering/agents/{brownfield-architect,greenfield-architect,architecture-reviewer}.md
- Surface/dimension: agent / 2 (token efficiency)
- Evidence: each file instructs loading `architecture-principles` ("it contains the shared architectural stance and guardrails used across all architecture agents" — brownfield:16, greenfield:16) and then restates the stance anyway: brownfield "Architectural stance" + "Scale-readiness rules" + "Strict rules" (lines 35–66, ~230 words), greenfield lines 34–66 (~240 words), architecture-reviewer "Principles" + "Strict rules" (lines 27–40). Near-identical monolith-first / no-premature-distribution / no-CQRS content three times.
- Fix shape: keep 2–3 agent-specific bullets per file; move the shared stance wholly into `architecture-principles` (where the files already claim it lives).

### [S2] feature-delivery-lead (2,685 words, corpus max) restates content owned by four skills — domains/engineering/agents/feature-delivery-lead.md
- Surface/dimension: agent / 2 (token efficiency) — mandatory review (> p90)
- Evidence: step 4d (line 106, ~190 words) walks the sizing-nudge schedule that `spec-sizing` "carries the exact paste-ready phrasing" for — the step even says "quote from it rather than improvising"; step 17 (lines 124–131, ~260 words) restates the Completion Ledger contract that engineer.md defines (both sides of the contract maintained in two files); step 18 restates `delivery-audit` mechanics; "Challenge handling" (lines 170–190) restates `challenge-diagnosis` mode detection. Design-phase paragraphs 2–3 (lines 45–47) duplicate each other's nudge-precedence explanation.
- Fix shape: keep the step list and halt conditions; compress each restated block to one pointer sentence + the non-negotiable rule. ~40% reduction available with no signal loss.

### [S2] Completion Ledger format defined in an agent file but consumed by two other surfaces — domains/engineering/agents/engineer.md
- Surface/dimension: agent / 2 (token efficiency), 7 (format consistency) — mandatory review (1,723 words)
- Evidence: lines 105–169 (~450 words) define the ledger tables, status definitions, and exercise-check format. feature-delivery-lead.md:124 cross-references it as "(see `engineer.md` — 'Closing output')" and `hero spec verify` Gate 1 parses it (engineer.md:138-139). A format contract read by the lead, the engineer, and the CLI belongs in a skill, not one agent's body.
- Fix shape: extract to a `completion-ledger` skill (or fold into `delivery-audit`); both agents reference it.

### [S2] Sales agents read hero.json config keys the config schema doesn't have — domains/sales/agents/{forecast-analyst,qualification-analyst}.md
- Surface/dimension: agent / 4 (freshness)
- Evidence: forecast-analyst.md:47-49: "Read from `.hero/hero.json`: `forecast.methodology` … `forecast.stages`"; qualification-analyst.md:38: "Read `qualification.framework` from `.hero/hero.json`." `internal/config/config.go` defines neither key (grep for `forecast`/`qualification`: no hits; PM's `pm.presets` by contrast is real, config.go:335-342). A cold agent reads the file, finds nothing, and only qualification-analyst names a default.
- Fix shape: add the config fields, or have the agents state the defaults and treat hero.json keys as optional-future.

### [S2] forecast-analyst's pipeline query can't work as written — domains/sales/agents/forecast-analyst.md
- Surface/dimension: agent / 4 (freshness)
- Evidence: line 32: `hero search --type deal --status "prospect,qualifying,demo,proposal,negotiation"`. `hero search --help`: `--status string` is a single status filter (no comma-list, unlike `hero list --status strings`), and the sales stage names aren't engine statuses. Also writes to `.hero/planning/forecasts/<period>.md` — tolerated by the generic walker but a non-canonical planning dir.
- Fix shape: iterate stages or filter client-side; align stage storage (`stage:` frontmatter vs status) with how search can actually filter.

### [S2] `role:` is dead frontmatter in 17 files; `domains:` present in 11 with no pattern — schema-wide
- Surface/dimension: agent / 7 (format consistency)
- Evidence: see section (c). `role:` (design/review/execution) is read by no engine code, no opencode field, no Claude Code field — files: architecture-reviewer, brownfield-architect, comment-scrubber, deadcode-scrubber, dedup-scrubber, defensive-scrubber, dependency-scrubber, design-reviewer, engineer, greenfield-architect, legacy-scrubber, pr-reviewer, roadmap-reviewer, security-reviewer, type-scrubber (engineering) + pm-reviewer (pm). `domains:` appears on engineering + sales files where it can only match trivially, and on none of the core files where it would actually filter.
- Fix shape: decide — either wire `role:` into something (roster docs, AGENTS.md generation) or strip it; document `domains:` as core-agents-only and remove the inert pack usages.

### [S2] buyer-researcher is an encyclopedia where a skill should be — domains/sales/agents/buyer-researcher.md
- Surface/dimension: agent / 2 (token efficiency) — mandatory review (1,310 words, > p90)
- Evidence: "Company research" subsections 1–8 (lines 43–133) are a ~700-word research-dimension catalog (tech-stack signal sources, news taxonomies, trigger tables) — reference material, not agent behavior. The sales pack keeps this kind of content in skills for every other agent.
- Fix shape: move the dimension catalog to a `buyer-research` skill; agent keeps the ground-in-hero-first rule, workflow, output formats, and rules.

### [S2] story-writer restates the EARS pattern set from the skill it just loaded — domains/pm/agents/story-writer.md
- Surface/dimension: agent / 2 (token efficiency) — mandatory review (1,468 words)
- Evidence: step 5 (lines 121–135) reproduces all five EARS patterns plus three examples, duplicating `acceptance-criteria-ears` (loaded at line 35). design-reviewer.md:30-36 carries the same five-pattern list — the pattern set now lives in ≥3 places. The v1 filename note block (lines 17–24) is changelog material, not agent instruction.
- Fix shape: keep one example, point at the skill for the pattern set; drop the filename note to a comment or the README.

### [S3] ui-designer frontmatter missing the execution fields every sibling has — domains/engineering/agents/ui-designer.md
- Surface/dimension: agent / 7 (format consistency)
- Evidence: frontmatter is `name, description` only — no `mode`, `temperature`, `color`, `permission` (inventory row 77; the only non-README agent in this shape). On opencode it runs with default (unrestricted) permissions despite compiling and executing generated Swift binaries.
- Fix shape: add the standard block.

### [S3] roadmap-reviewer hedges for a flag that now exists; harness-specific MCP naming — domains/engineering/agents/roadmap-reviewer.md
- Surface/dimension: agent / 4 (freshness), 6 (harness-agnosticism)
- Evidence: line 112: "`hero size --ack giant <slug>`. If the `--ack` flag does not exist yet, write `size_ack: giant` directly…" — `hero size --help` shows `--ack` shipped. Lines 62–64: survey table names tools in Claude-Code MCP naming (`mcp__hero__hero_warnings`, `mcp__hero__hero_list`, `mcp__hero__hero_search`) where sibling agents use harness-neutral names (`hero_anchor`, `hero_code`).
- Fix shape: drop the fallback clause; normalize tool names to the bare `hero_*` form used elsewhere.

### [S3] issue-tracker: weak trigger description and tool-specific guidance without naming the tool — domains/engineering/agents/issue-tracker.md
- Surface/dimension: agent / 5 (triggering), 3 (actionability)
- Evidence: description ("Maintain local issue queue reports from the tracking system…") gives a session no trigger cue vs `/import` or `hero sync import`, which cover adjacent ground. Line 24: "use the `page_token` field for paging when the issue tracker API supports it" — a specific API's field name with no tool named; line 19's "current project workspace or sandbox path" is unanchored.
- Fix shape: description states when to reach for it ("standing unassigned-bug report refreshed each session, without re-querying the tracker"); replace `page_token` sentence with "use the tracker tool's paging mechanism."

### [S3] pm-investigator copies debug-investigator's dead-end doctrine — domains/pm/agents/pm-investigator.md
- Surface/dimension: agent / 2 (token efficiency)
- Evidence: "You won't always find the answer — and that's fine" + "The worst outcome is not 'I don't know'…" (lines 22–25) mirror debug-investigator.md:18-30; the write-verify-report choreography (steps 6–7) mirrors debug-investigator steps 3/8. Reasonable adaptation, but the investigation-persistence doctrine is now maintained twice.
- Fix shape: extract the shared investigator doctrine into a core skill (`agent-reliability` is the natural home) referenced by both.

### [S3] `{hero_folder}` placeholder used by the two delivery leads, literal `.hero/` everywhere else — domains/engineering/agents/{feature-delivery-lead,platform-delivery-lead}.md
- Surface/dimension: agent / 7 (format consistency)
- Evidence: `{hero_folder}/planning/features/{slug}/spec.md` (feature:56,66; platform:52). No engine code substitutes it (repo grep); it's defined only inside the `spec-format` skill. All other agents hardcode `.hero/`; feature-delivery-lead itself uses both forms (line 96: `.hero/planning/features/<slug>/plan.md`).
- Fix shape: pick one convention; literal `.hero/` unless configurable-workspace-location lands.

### [S3] Scrubber startup boilerplate repeated seven times — domains/engineering/agents/*-scrubber.md (7 files)
- Surface/dimension: agent / 2 (token efficiency)
- Evidence: identical 3-step "## Startup" block (load `code-scrub`, load `stack-detection`, load detected stack skill) in comment-, deadcode-, dedup-, defensive-, dependency-, legacy-, type-scrubber. Small (~25 words each); the `code-scrub` skill could own the startup sequence and the agents reference it in one line.
- Fix shape: optional consolidation; lowest priority.

### [S3] Sales pack invents spec types and knowledge dirs the engine doesn't know — domains/sales/agents/{buyer-researcher,competitive-intel}.md
- Surface/dimension: agent / 4 (freshness)
- Evidence: buyer-researcher.md:36-38 (`--type deal`, `--type knowledge`), 227-228 (knowledge types `prospect`, `persona`); competitive-intel.md:36 (`.hero/knowledge/battlecards/<slug>.md`, `type: battlecard`). The generic walker will load these files, but no engine surface (typeFromPath, statusFromPath, `hero list --type` enum) knows the types, and flat `.md` knowledge files with unrecognized work-ish types interact oddly with flat-spec discovery (`internal/spec/spec.go:1160-1170`). Works by accident, not by contract.
- Fix shape: register the sales types/dirs (spec-types files + engine awareness) or store them under recognized knowledge subdirs with `type:` documented as freeform.

---

## Freshness verifications that came back clean (so future passes don't re-check)

`hero relevant`, `hero index`, `hero drift`, `hero size --check/--summary/--ack`, `hero spec lint/verify --skip-tests/complete/mock detect/set-owner`, `hero sync pull`, `hero search --file/--list/--type/--status`, `hero list --status/--mine`, `hero anchor`, `hero scan`, `hero check` — all exist as referenced. MCP tools referenced (`hero_code`, `hero_anchor`, `hero_score`, `hero_conflicts`, `hero_drift`, `hero_warnings`, `hero_kickoff`, `hero_pulse`, `hero_read_spec`, `hero_event`) all exist. Spec-type paths cited by pm-reviewer/handoff-coordinator/roadmap-curator (`core/spec-types/{feature,epic,initiative}.md`, `domains/pm/spec-types/{prd,intake}.md`) all exist. `/mock`, `/split`, `/compose`, `/challenge`, `/roadmap-review`, `/debrief`, `/prospect`, and all cited pm commands exist in their packs. Both README.md files (pm, sales): all referenced files exist; the pm README's roster table matches the shipped 12.
