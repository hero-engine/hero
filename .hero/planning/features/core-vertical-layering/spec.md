---
title: Core / Vertical Layering — Make the Conceptual Split Physical
type: feature
status: planning
priority: P0
tags: [architecture, refactor, core, verticals, triangulation, foundational]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: domain-plugin-architecture
    kind: extends
  - target: project-charter
    kind: depends-on
  - target: multi-domain-core
    kind: completes
mission_alignment: |
  The mission distinguishes core (engine, manifesto, universal) from
  verticals (domain-specific toolkits riding on the engine). The repo
  layout doesn't reflect that distinction — engineering-specific
  agents, commands, and skills live at the root pretending to be core.
  This forces every contributor to re-derive what's universal each
  time. Making the layering physical means new verticals (Hero Sales
  next) can be built without forking, and core engineering can stop
  blending with vertical concerns.
principles_check: |
  Serves #1 (it just works — when the layering is physical, agent
  routing, install scaffolding, and content discovery all become
  trivially correct). Serves #5 (the practitioner sees a clean
  separation: "I want to extend Hero Code" vs. "I want to add a new
  vertical" are obviously different paths). Risks none directly; the
  refactor itself is one-shot work, not new ongoing surface.
horizon: next
smoke: deferred
---

## Goal

Make the conceptual core/vertical layering physical in the repo. Move
engineering-specific content out of root into `domains/engineering/`,
keep core engine at root, scaffold a second vertical (`domains/sales/`)
to triangulate, and update install + embed paths to match.

## Why now

Three reasons:

1. **The audit established the layering matters.** Half the v2 drift
   was core+vertical concerns blending indistinguishably (e.g.,
   `internal/codescan` is engineering-specific but ships as a core
   package; `agents/engineer.md` is engineering-specific but lives at
   the root next to `agents/session-primer.md` which is core).
2. **`domain-plugin-architecture` is 80% done** (per audit 3) — the
   `domains/engineering/` directory exists with agents/commands/skills,
   but the `hero domain` CLI is missing and the install flow doesn't
   yet select-by-domain. Finishing this is small.
3. **Hero Sales is next** (per the recovery conversation). Without
   the layering being physical, building Hero Sales would either
   fork the repo or pollute the root again. With two verticals
   side-by-side, it becomes obvious what's universal.

## Proposed structure

```
/                                   # repo root
├── cmd/                            # CORE — binary entry
├── internal/                       # CORE — engine implementation
│   ├── graph/                      # core
│   ├── index/                      # core
│   ├── projection/                 # core
│   ├── extract/                    # core
│   ├── sync/                       # core
│   ├── serve/                      # core (MCP, HTTP, watcher)
│   ├── codescan/                   # ⚠️ engineering-specific
│   │                                  → consider domains/engineering/
│   │                                    or rename to highlight scope
│   └── ...
├── cloud/                          # CORE — team-mode anchor
├── core/                           # CORE — universal agents/commands/skills
│   ├── agents/                     # session-primer, project-context-builder,
│   │                                  documentation-engineer (the cross-vertical ones)
│   ├── commands/                   # /note, /handoff, /resume, /prime,
│   │                                  /scan, /check (corpus-management commands)
│   └── skills/                     # auto-knowledge-capture, context-injection,
│                                       nudge-awareness, next-md, spec-format
├── domains/
│   ├── engineering/                # Hero Code vertical
│   │   ├── mission.md              # ✅ created 2026-04-28
│   │   ├── AGENTS.md               # already moved
│   │   ├── agents/                 # already moved (engineer, debug-investigator,
│   │   │                              brownfield-architect, scrubbers, reviewers, etc.)
│   │   ├── commands/               # already moved (/design, /diagnose, /deliver,
│   │   │                              /review, /scrub, /release, /sprint, /retro)
│   │   ├── skills/                 # already moved (language stacks, workflow patterns)
│   │   └── spec-types.md           # NEW: feature, bug, decision, convention, sprint
│   ├── sales/                      # Hero Sales vertical (scaffold today)
│   │   ├── mission.md              # ✅ created 2026-04-28 (scaffold)
│   │   ├── AGENTS.md               # ✅ created 2026-04-28 (scaffold)
│   │   ├── agents/                 # placeholder
│   │   ├── commands/               # placeholder
│   │   ├── skills/                 # placeholder
│   │   └── spec-types/             # placeholder (account, deal, objection,
│   │                                  debrief, etc. — defined when built)
│   └── (more verticals as added)
├── .hero/                          # this project's own workspace
│   ├── mission.md                  # this project's CORE charter
│   ├── planning/                   # in-flight work
│   ├── knowledge/                  # captured corpus
│   └── ...
├── docs/                           # repo-level meta
└── (existing root agents/, commands/, skills/ removed or symlinked)
```

## Triangulation: the actual core/vertical split for existing content

Walking the existing artifacts with both Hero Code and Hero Sales
mission files in mind. *"Would this serve a sales workflow?"* If yes,
core. If no, engineering vertical.

### Currently at root, should be CORE

- `agents/session-primer.md` — session-start guidance (any vertical)
- `agents/project-context-builder.md` — corpus-context assembly
- `agents/documentation-engineer.md` — docs is universal, not
  engineering-specific
- `commands/note.md` — capture (universal)
- `commands/handoff.md` — session checkpoint (universal)
- `commands/resume.md` — session restore (universal)
- `commands/prime.md` — session priming (universal)
- `commands/scan.md` — master ingest (universal substrate)
- `commands/check.md` — workspace health (universal)
- `commands/capture.md` — learning capture (universal)
- `commands/decide.md`, `commands/convention.md` — knowledge artifacts
  exist in any vertical (just with different names)
- `skills/auto-knowledge-capture.md` — core mechanism
- `skills/context-injection.md` — core mechanism
- `skills/nudge-awareness.md` — core mechanism
- `skills/next-md.md` — core artifact
- `skills/spec-format.md` — spec format is core; spec *types* are vertical
- `skills/knowledge-flywheel.md` — core mechanism
- `skills/note-capture.md` — core mechanism

### Currently at root, should be ENGINEERING vertical

- All language-stack skills (`go-stack`, `python-stack`, `react-stack`,
  `java-stack`, `rust-stack`, `javascript-stack`, `groovy-stack`)
- All engineering-workflow skills (`api-design-and-contracts`,
  `architecture-principles`, `database-stack`, `debugging-investigation`,
  `dependency-analysis`, `devops-and-operations`,
  `documentation-practices` (engineering-flavored),
  `greenfield-scaffolding`, `html-mockup-generation`,
  `implementation-principles`, `integration-boundaries`,
  `migration-safety`, `performance-optimization`, `pr-review`,
  `project-context-generation` (engineering-flavored),
  `release-and-deployment`, `root-cause-classification`,
  `security-review`, `stack-detection`, `test-strategy`,
  `testing-and-validation`)
- All engineering-specific agents (`engineer`, `api-engineer`,
  `brownfield-architect`, `greenfield-architect`, `database-engineer`,
  `debug-investigator`, `devops-engineer`, `feature-delivery-lead`,
  `platform-delivery-lead`, `migration-engineer`,
  `performance-engineer`, `release-engineer`, `security-reviewer`,
  `pr-reviewer`, `architecture-reviewer`, `design-reviewer`,
  `functional-qa-engineer`, `test-architect`, `dependency-analyst`,
  `comment-scrubber`, `deadcode-scrubber`, `dedup-scrubber`,
  `defensive-scrubber`, `dependency-scrubber`, `legacy-scrubber`,
  `type-scrubber`, `convention-author`, `integration-engineer`,
  `issue-tracker`, `product-ideator`, `ui-designer`)
- All engineering commands (`/design`, `/diagnose`, `/deliver`,
  `/review`, `/scrub`, `/release`, `/sprint`, `/retro`, `/compose`,
  `/split`, `/mock`, `/discover`, `/docs`)
- `AGENTS.md` (the root-level one is engineering-specific)

### Ambiguous, needs decision

- `agents/issue-tracker.md` — issue tracking happens in many verticals,
  but the current implementation is Jira/GitHub/Linear which is
  engineering-flavored. **Lean: core interface, engineering plugins.**
- `agents/convention-author.md` — every vertical has conventions/
  playbooks. **Lean: core, with vertical-specific naming overrides.**
- `internal/codescan/` — clearly code-specific. **Lean: move to
  `domains/engineering/codescan/` OR keep in `internal/` but rename
  to highlight scope (`internal/engineering/codescan/`).**

## Acceptance criteria

**AC-1:** New top-level layout exists: `core/agents/`, `core/commands/`,
`core/skills/` populated with the universal artifacts above.

**AC-2:** `domains/engineering/` is complete: all engineering-specific
agents, commands, skills moved in. `domains/engineering/AGENTS.md` is
the engineering-vertical AGENTS file (current root `AGENTS.md` content).

**AC-3:** `domains/sales/` scaffold exists with mission + AGENTS +
empty subdirs (✅ created in this spec's prep work). When Hero Sales
is built, real content lands here without restructure.

**AC-4:** Root-level `agents/`, `commands/`, `skills/`, `AGENTS.md`
either removed or symlinked to the active configuration. Backward
compat: existing tooling that reads from root paths continues to work
during a transition period.

**AC-5:** `embed.go` updated to embed `core/*/` and `domains/*/`
separately. `hero install` selects which to install based on
`hero.json` `domain:` (or `domains:` for multi-vertical use).

**AC-6:** `hero domain` CLI exists (the missing 20% from
`domain-plugin-architecture` audit finding):
- `hero domain list` — installed/available verticals
- `hero domain switch <name>` — change active vertical
- `hero domain show` — current active vertical and its contents

**AC-7:** `hero init` defaults to `--domain engineering`. `hero init
--domain sales` would write `"domain": "sales"` into hero.json (won't
work substantively until the vertical is built; this AC just checks
the plumbing).

**AC-8:** Triangulation table in this spec is reviewed and any
disagreements resolved before move execution. The "Ambiguous, needs
decision" section is empty when this AC passes.

ACs accrete as the move surfaces edge cases (e.g., a skill that turns
out to be more cross-vertical than expected).

## Approach

**Phase 1 — triangulation review** (~½ day): Walk the table above
with the user; resolve ambiguous items. Produces the canonical
"core vs. engineering" inventory.

**Phase 2 — move + symlinks** (~1 day): Execute the file moves.
Keep root-level paths working via symlinks for one release cycle.
Update `embed.go`. All existing tests continue to pass.

**Phase 3 — `hero domain` CLI** (~½ day): The missing 20% from
`domain-plugin-architecture`. Wire `list`, `switch`, `show`.

**Phase 4 — install-flow update** (~½ day): `hero install` reads
domain config, copies the right content. Generic target +
domain-specific overlay.

**Phase 5 — symlink removal** (~½ day, deferred): One release cycle
later, remove the back-compat symlinks at root. Until then, root
paths work but are deprecated.

## Verticals can have dedicated UIs (added 2026-04-28)

User feedback during the spec write: *"i think we'll have finance,
accounting?, marketing, management, down the road. each with their
own desktop app or web app."*

Implication: a vertical is not just a content pack (agents +
commands + skills). It can also include its **own dedicated user
interface** — a desktop app, a web app, or a domain-tailored
experience. Hero Code's UI today is implicit (the harness + the
dashboard); future verticals will build dedicated apps.

This sharpens the layering: **Hero is a platform; each vertical is a
product**. Same shape as Office shipping Word/Excel/PowerPoint as
distinct products on a shared foundation. The directory layout
proposed above already accommodates this — a vertical's `domains/<name>/`
directory can grow a `ui/` (or `app/`) subtree alongside its
agents/commands/skills/spec-types when it gets its own UI:

```
domains/
  sales/
    mission.md
    AGENTS.md
    agents/
    commands/
    skills/
    spec-types/
    ui/                    # NEW (when vertical has dedicated app)
      web/                 # web-app source
      desktop/             # desktop-app source (Tauri / Electron)
      shared/              # shared components
```

Vertical UIs consume core via the same APIs as the harness
(MCP tools + HTTP API + SSE event stream from `hero serve`). The
core engine doesn't know or care which UI is reading it. Cross-
vertical context (a sales conversation informing an engineering
decision) flows through the shared corpus automatically — that's
core's job.

**Vertical UI work is out of scope for this spec.** This spec only
ensures the directory layout accommodates dedicated UIs when they
arrive. Building any specific UI is a separate, per-vertical spec.

## Out of scope

- Building actual Hero Sales content (agents, commands, skills, UI)
  — this spec only scaffolds the directory and mission
- Refactoring `internal/codescan/` package location — flagged as
  ambiguous; resolve in triangulation review, execute as separate
  small spec if needed
- Multi-domain workspaces (one project using both Hero Code AND Hero
  Sales simultaneously) — interesting but defer to v2.1; today the
  assumption is one domain per workspace, with cross-vertical
  context flowing only through the shared corpus
- Building any specific vertical UI (Hero Sales rep app, Hero
  Finance dashboard, etc.) — each is its own spec under its own
  vertical when prioritized

## Open questions

- Symlink vs. generated-at-install for backward compat? Lean:
  symlinks during transition (zero copies, easy rollback);
  generated-at-install once `hero domain switch` is the canonical way
  to change active vertical.
- Should `core/` exist as a top-level directory, or do we leave the
  universal agents/commands/skills at root and only move the
  engineering-specific ones into `domains/engineering/`? Lean: top-
  level `core/` is more honest and matches the conceptual layering.
  Symmetric: `core/` and `domains/<name>/` both contain
  agents/commands/skills/.
- For projects using Hero (not Hero itself), where does their
  `mission.md` live? Lean: `.hero/mission.md` is the project's
  *active* charter (which inherits core + selected vertical).
  Verticals' missions are read-only references the project inherits
  from but doesn't edit.
