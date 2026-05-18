---
title: Core / Vertical Triangulation — Full Classification of Existing Artifacts
slug: core-vertical-triangulation
type: note
status: active
tags: [triangulation, core, vertical, classification, layering, refactor-prep]
created: 2026-04-28
relations:
  - target: core-vertical-layering
    kind: classification-for
  - target: get-back-on-track
    kind: supports
horizon: next
---

# Core / Vertical Triangulation

Full enumeration of every existing agent, command, and skill at root,
classified as **core** (universal — serves any vertical) or
**engineering** (specific to Hero Code). Done with both
[Hero Code](../../../domains/engineering/mission.md) and
[Hero Sales](../../../domains/sales/mission.md) mission files in
mind.

**The classification rule:** *"Would this serve a sales (or finance,
marketing, accounting, management) workflow as well? If yes →
**core**. If only engineering → **engineering**. If unclear → ask."*

**For ambiguous items:** default to **engineering** for now (since
the existing implementation is engineering-flavored), with a note
that promotion to core can happen when a second vertical concretely
needs it.

## Counts

| Category | Core | Engineering | Total |
|---|---|---|---|
| Agents | 4 | 31 | 35 |
| Commands | 14 | 11 | 25 |
| Skills | 12 | 32 | 44 |
| **Total** | **30** | **74** | **104** |

About a third is core. Two-thirds is the engineering vertical that's
been pretending to be core for a year.

---

## Agents (35 total)

### CORE (4)

| Agent | Why core |
|---|---|
| `convention-author` | Every vertical has conventions/playbooks (sales playbooks, finance procedures, etc.) |
| `documentation-engineer` | Documentation is universal — sales reps document deals, finance documents processes |
| `project-context-builder` | Building project context is the core engine's job |
| `session-primer` | Session priming is universal — every vertical starts cold every session |

### ENGINEERING (31)

| Agent | Why engineering |
|---|---|
| `api-engineer` | API design is software-specific |
| `architecture-reviewer` | Software architecture |
| `brownfield-architect` | Software architecture |
| `comment-scrubber` | Code comments |
| `database-engineer` | DB schema/queries |
| `deadcode-scrubber` | Dead code analysis |
| `debug-investigator` | Software debugging |
| `dedup-scrubber` | Code duplication |
| `defensive-scrubber` | Defensive coding patterns |
| `dependency-analyst` | Software dependencies |
| `dependency-scrubber` | Dep cleanup |
| `design-reviewer` | Spec-design review (engineering-flavored today) |
| `devops-engineer` | CI/CD, infra |
| `engineer` | The implementer agent |
| `feature-delivery-lead` | Engineering feature delivery |
| `functional-qa-engineer` | Software QA |
| `greenfield-architect` | Greenfield software |
| `integration-engineer` | Software integrations |
| `legacy-scrubber` | Legacy code |
| `migration-engineer` | Code/data migrations |
| `performance-engineer` | Software performance |
| `platform-delivery-lead` | Platform = software platform |
| `pr-reviewer` | Pull request review |
| `release-engineer` | Software releases |
| `security-reviewer` | Code security |
| `test-architect` | Test architecture |
| `type-scrubber` | Type systems |
| `ui-designer` | HTML/CSS UI mockups (software UI) |

### Ambiguous (lean engineering for now, may promote later)

| Agent | Question |
|---|---|
| `issue-tracker` | Sales tracks issues differently (CRM cases, not Jira issues). Current impl is Jira/GitHub/Linear → engineering. Later: extract a core "tracker" interface, vertical-specific implementations. |
| `product-ideator` | Product ideation could happen in any business context, but here it's product-as-software-product. Lean engineering; promote when sales needs equivalent. |

---

## Commands (25 total)

### CORE (14)

| Command | Why core |
|---|---|
| `/capture` | Capture session learnings — universal |
| `/check` | Workspace health — universal |
| `/convention` | Codify a convention/pattern — universal |
| `/decide` | Record a decision — universal |
| `/discover` | Discovery as a verb is universal (sales discovery, product discovery, market discovery) |
| `/docs` | Documentation — universal |
| `/handoff` | Session checkpoint — universal |
| `/hero` | Meta/routing command |
| `/import` | Pull external content — universal |
| `/note` | Quick note capture — universal |
| `/prime` | Session priming — universal |
| `/resume` | Session resume — universal |
| `/retro` | Retrospective — every vertical has them |
| `/scan` | Master ingest — the engine's verb |

### ENGINEERING (11)

| Command | Why engineering |
|---|---|
| `/challenge` | Currently scoped to "challenge a bug diagnosis" |
| `/compose` | Compose multi-spec engineering initiative |
| `/deliver` | Currently means "ship the spec by writing code" |
| `/design` | Currently means "design a feature/bug spec for software" |
| `/diagnose` | Software bug investigation |
| `/mock` | HTML mockup for software UI |
| `/release` | Software release |
| `/review` | Code/PR review |
| `/scrub` | Code scrubbing |
| `/split` | Spec splitting (engineering-flavored) |
| `/sprint` | Sprint planning (engineering-uses Jira-style sprints) |

### Ambiguous (lean engineering for now)

None today. Several engineering commands have universal verb-shape
(`/deliver`, `/design`, `/sprint`) — when a second vertical builds
its own version, we may promote the verb to core with vertical
dispatch. Defer until needed.

---

## Skills (44 total)

### CORE (12)

| Skill | Why core |
|---|---|
| `agent-reliability` | Agent quality is universal |
| `auto-knowledge-capture` | The session-end residue mechanic |
| `context-injection` | The session-start injection mechanic |
| `convention-writing` | Writing conventions — universal |
| `documentation-practices` | Universal — every vertical documents |
| `executive-report` | Executive reporting in any vertical |
| `knowledge-flywheel` | The corpus-compounds-with-every-turn mechanic |
| `next-md` | NEXT.md format — universal session artifact |
| `note-capture` | Note capture — universal |
| `nudge-awareness` | Nudging the agent during reasoning — universal |
| `project-context-generation` | Any vertical needs project context |
| `spec-format` | Spec format is core; spec *types* are vertical |

### ENGINEERING (32)

| Skill | Why engineering |
|---|---|
| `api-design-and-contracts` | API design |
| `architecture-principles` | Software architecture |
| `challenge-diagnosis` | Bug diagnosis |
| `code-scrub` | Code cleanup |
| `database-stack` | DB systems |
| `debugging-investigation` | Software debugging |
| `deep-code-enrichment` | Code analysis |
| `dependency-analysis` | Software deps |
| `devops-and-operations` | DevOps |
| `go-stack` | Go language |
| `greenfield-scaffolding` | Greenfield code |
| `groovy-stack` | Groovy language |
| `html-mockup-generation` | Software UI mocks |
| `implementation-principles` | Software implementation |
| `integration-boundaries` | Software integrations |
| `java-stack` | Java language |
| `javascript-stack` | JS language |
| `migration-safety` | Code/data migration |
| `performance-optimization` | Software performance |
| `pr-review` | PR review |
| `python-stack` | Python language |
| `react-stack` | React framework |
| `release-and-deployment` | Software release |
| `rust-stack` | Rust language |
| `security-review` | Code security |
| `stack-detection` | Tech stack detection |
| `test-strategy` | Test strategy |
| `testing-and-validation` | Software testing |

### Ambiguous (lean engineering for now)

| Skill | Question |
|---|---|
| `incident-response` | Universal pattern (security incident, sales incident, financial incident). Current impl is software-incident-flavored. Promote to core when a second vertical needs it. |
| `issue-list-report` | Issues exist in many domains; current impl uses Jira/GitHub. Engineering for now. |
| `root-cause-classification` | Universal RCA pattern; current impl is engineering-flavored. Engineering for now. |

---

## Engine code (in `internal/`)

These aren't classified for *move* but for the conceptual record. Most
of `internal/` is core engine. One package is engineering-specific
and worth flagging:

| Package | Classification | Note |
|---|---|---|
| `internal/graph/` | core | The graph substrate |
| `internal/index/` | core | FTS5 index |
| `internal/extract/` | core | Tier-2 extraction |
| `internal/projection/` (planned) | core | Projection engine |
| `internal/sync/` | core | Team-server sync |
| `internal/serve/` | core | MCP + HTTP + watcher |
| `internal/spec/` | core | Spec parser + writer |
| `internal/sessions/` | core | Session lifecycle |
| `internal/recap/` | core | Activity digest |
| `internal/impact/` | core | Impact graph |
| `internal/drift/` | core | Spec-vs-code drift |
| `internal/scan/` | core | Stack analysis (universal) |
| `internal/codescan/` | **engineering** ⚠️ | Code-specific. Move to `domains/engineering/codescan/` OR rename to `internal/engineering/codescan/`. Resolve in execution phase. |
| `cmd/`, `cloud/` | core | Binary + cloud server |

---

## Action plan (post-sign-off)

Once this classification is locked:

1. **Phase 1 — symlink-based move.** Move all engineering-classified
   files into `domains/engineering/{agents,commands,skills}/` — keep
   root paths working via symlinks. All existing tests/install flows
   continue. Single commit.
2. **Phase 2 — populate `core/`.** Create `core/agents/`,
   `core/commands/`, `core/skills/` and move (or symlink) the
   core-classified files there. Single commit.
3. **Phase 3 — `embed.go` + `hero install` updates.** Update the
   embedded filesystem to read from `core/` + selected `domains/<x>/`.
   Tests verify install copies the right things.
4. **Phase 4 — `internal/codescan/` decision.** Review whether to
   leave in `internal/`, rename to `internal/engineering/codescan/`,
   or move to `domains/engineering/codescan/`. Tradeoff: import-path
   churn vs. honest layering.
5. **Phase 5 — symlink removal** (one release later). Remove root-
   level `agents/`, `commands/`, `skills/`, `AGENTS.md` symlinks
   after confirming no external dependents.

Each phase is its own commit. Execution work tracked under
`core-vertical-layering` spec.

## Locked decisions (2026-04-28)

User confirmed:

- All five ambiguous items → **engineering for now**:
  - `agents/issue-tracker` (Jira/GitHub-flavored; sales will have
    its own — Salesforce — when built)
  - `agents/product-ideator`
  - `skills/incident-response`
  - `skills/issue-list-report`
  - `skills/root-cause-classification`
- All four core agents confirmed: `convention-author`,
  `documentation-engineer`, `project-context-builder`,
  `session-primer`
- 14 core commands confirmed
- 12 core skills confirmed
- `internal/codescan/` — defer to execution phase

## Future considerations (captured during triangulation)

Two insights worth holding onto for when more verticals come online:

### Sync subsystem may become core (with vertical plugins)

User: *"perhaps sync — some of will become core as we sync other
domain tools?"*

The sync subsystem (today: Jira, GitHub Issues, Linear, Confluence,
GitHub Pages, GitHub Wiki, the cloud team server) is wired
engineering-flavored because every external integration so far is
an engineering tool. When Hero Sales adds Salesforce sync, Hero
Marketing adds Mailchimp/HubSpot sync, Hero Finance adds QuickBooks/
Xero sync, the sync *mechanism* will look like a candidate to
extract:

- **Core**: a generic sync framework — auth, rate-limiting, retry,
  conflict detection, scope filtering, schema mapping
- **Vertical plugins**: per-tracker / per-tool implementations that
  declare their entity types, field mappings, and update semantics

This is exactly the kind of refactor a second concrete vertical
forces into focus. Defer until Hero Sales (or another vertical)
actually ships its first integration.

### Possible Hero Product vertical (distinct from Hero Code)

User: *"the product idea stuff - perhaps that is engineering for now
- we need to figure out if a domain falls out around product design
and management separate from coding? and we can reconcile then. is
there ideation flows in other domains more general?"*

Product/design ideation is currently bundled into Hero Code (the
`product-ideator` agent, `/discover` command's product-discovery
flavor, the `ui-designer` agent for mockups). But product
managers, product designers, and design researchers do work that
*isn't* engineering work — even though it precedes engineering.

The open question: is there a **Hero Product** vertical (PM + design
+ research) sitting between Hero Code and the future Hero Sales /
Hero Marketing / etc., with its own:
- Spec types: persona, journey, hypothesis, experiment, design-spec
- Agents: product-strategist, ux-researcher, design-critic
- Skills: discovery-research, prototyping, journey-mapping
- Commands: /persona, /journey, /hypothesize, /experiment, /critique

Tangentially: ideation flows ARE more general than product —
sales has account-strategy ideation, marketing has campaign
ideation, finance has scenario planning, management has org
ideation. So *"ideation"* might also be a candidate for a core
verb (`/ideate`?) with vertical-specific implementations.

Both questions are deferred. Revisit when (a) a second vertical is
real enough to triangulate against, or (b) a user actually wants to
use Hero for product/design work without the engineering surface.

## Action plan — unchanged, now unblocked

The action plan (Phase 1 → 5 above) is now unblocked by the locked
decisions above. Phase 1 (symlink-based moves) can begin
immediately.
