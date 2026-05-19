# Hero

Hero is the sidekick brain for AI-augmented knowledge work.

For engineering teams, Hero adds a spec-driven workflow to AI coding
tools: **design before you build, diagnose before you fix**. It captures
specs, decisions, conventions, attempts, failures, tests, and recent
activity into a local `.hero/` corpus so each new model session starts
with the right project context.

**New here? Start with [GETTING-STARTED.md](GETTING-STARTED.md).**

Hero currently installs into OpenCode, Cursor, Claude Code, Codex,
GitHub Copilot, and generic MCP-capable tools.

---

## What Hero Does

Hero has two layers:

| Layer | Purpose |
|---|---|
| Core Hero | Capture, structure, query, project, inject, and sync the project corpus. |
| Hero Code | The engineering vertical: specs, commands, agents, skills, tests, tracker sync, delivery workflows. |

The spec workflow is the surface. The durable value is the corpus that
compounds across sessions, tools, teammates, and time.

```text
/discover  ->  /design  ->  /deliver
                           ^
              /diagnose ---|
```

| Workflow | Use it when |
|---|---|
| `/resume` | Starting a fresh session and loading current context. |
| `/discover` | Exploring product direction or possible work. |
| `/design` | Creating a feature, platform, or documentation spec before building. |
| `/diagnose` | Investigating a bug and producing a fix spec. |
| `/deliver` | Implementing and validating an approved spec. |
| `/handoff` | Preserving state before switching tools or context gets tight. |
| `/review` | Reviewing code, PRs, specs, architecture, security, or tests. |
| `/scrub` | Removing dead code, duplication, weak types, stale comments, and legacy cruft. |

Natural language is the default interface. In a Hero-aware harness, say
what you want and Hero routes to the right workflow. The CLI remains the
escape hatch for scripting, inspection, sync, and verification.

---

## Install

**macOS / Linux (Homebrew):**

```bash
brew install hero-engine/tap/hero
```

**Linux (install script):**

```bash
curl -fsSL https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.sh | sh
```

**Windows (Scoop):**

```powershell
scoop bucket add hero-engine https://github.com/hero-engine/scoop-bucket
scoop install hero
```

**Windows (PowerShell install script):**

```powershell
irm https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.ps1 | iex
```

Full options — including direct downloads and build-from-source — are in
the [installation guide](web/docs/src/getting-started/installation.md).

Initialize a project and install Hero into your coding tool:

```bash
cd /path/to/project
hero init
hero install project . --target opencode
hero install project . --target cursor
hero install project . --target claude
hero install project . --target codex
hero install project . --target copilot
```

For monorepos where the AI tool runs from a subfolder, register each
subfolder as a satellite of the root install:

```bash
hero install project . --target cursor --workspace services/auth
hero install satellites list
hero install satellites add services/auth
hero install --repair
```

Hero ships content in two layers: a shared `core/` pack and one or more
domain packs (currently `domains/engineering/` and a scaffolded
`domains/sales/`). The active domain comes from `hero.json` or the
`--domain` flag:

```bash
hero install project . --target claude --domain engineering
hero domain
```

`hero install` copies the active domain's command, agent, and skill
files into the target harness format and registers `hero mcp` so the
tool can call Hero directly. Use `--migrate` to reconcile drifted
copies across multiple harnesses, and `hero check` to audit workspace
and install state. The Hero-managed sections inside `AGENTS.md`/`CLAUDE.md`
are regenerated in place on every install — the markers signal that
the content between them is owned by Hero.

Current installed content counts:

| Surface | Count |
|---|---:|
| Slash command definitions | 28 |
| Agent definitions | 34 |
| Skill definitions | 45 |
| MCP tools | 42 |

Run `hero docs check` to validate these counts against the repo.

---

## Daily Use

Start every session:

```text
/resume
```

Seed a project:

```text
/scan
/convention codify our API error response format
```

Design and deliver:

```text
/design add CSV export for user data
/deliver .hero/planning/features/csv-export/spec.md
```

Investigate and fix:

```text
/diagnose login times out after 30 seconds
/deliver .hero/planning/bugs/login-timeout/spec.md
```

Preserve context:

```text
/handoff
```

Ask the corpus:

```bash
hero ask "what is our error response format?"
hero search "OAuth session handling"
hero relevant src/auth/session.go src/auth/middleware.go
hero why csv-export
hero blocked
```

---

## Cross-Repo Peering

When you run sibling Hero workspaces — backend, web client, desktop
client — they can work as a tag team. A session in one repo can ask
another's Hero a question, hand off a spec, or pull peer-surface
conventions into context. Provenance travels with every operation.

Three interaction modes, plus a passive boundary detector:

| Mode | Writes? | Pick when |
|---|---|---|
| Sync peer call — advisory | Nothing | You need a fact from peer B: "does this break you?", "what's your convention for X?" |
| Sync peer call — spec-out | Spec on B | The work is really B's. B's Hero designs the spec natively; its conventions kick in. |
| Async handoff | Spec on B (scaffolded) | You already did the investigation; drop it on B's queue. |
| Convention import (fallback) | Nothing | Work stays in A but must respect B's surface. |

Quick cheat sheet:

```bash
hero init                                # mints a stable peer_id UUID
hero admin repos add app ../app          # register a sibling peer
hero peer call app --mode=advisory "What's your error envelope?"
hero handoff order-failure app --reason "Root cause is the API"
```

V1 runs on a developer laptop with three sibling checkouts and no
cloud. Full-delivery peer calls, boundary nudges, and cloud transport
are deferred.

See [CROSS-REPO-PEERING.md](CROSS-REPO-PEERING.md) for setup, the full
ladder, lifecycle reference, troubleshooting, and the dogfood checklist.

---

## Current CLI Map

The binary is organized around a few stable groups.

| Area | Commands |
|---|---|
| Session context | `hero resume`, `hero next`, `hero recap`, `hero feed`, `hero relevant`, `hero ask`, `hero search`, `hero do` |
| Spec lifecycle | `hero spec new`, `hero spec deliver`, `hero spec verify`, `hero spec complete`, `hero spec claim`, `hero spec plan`, `hero diff`, `hero drift`, `hero list`, `hero queue`, `hero suggest` |
| Acceptance criteria | `hero ac list`, `hero ac record`, `hero ac status`, `hero ac history`, `hero coverage`, `hero spec contract` |
| Workspace health | `hero status`, `hero dashboard`, `hero check`, `hero docs check`, `hero smoke`, `hero ci`, `hero anchor`, `hero tripwire` |
| Graph and retrieval | `hero scan`, `hero graph`, `hero extract`, `hero impact`, `hero why`, `hero blocked`, `hero snapshot` |
| Tracker and sync | `hero sync connect`, `hero sync import`, `hero sync pull`, `hero sync spec`, `hero sync link`, `hero sync comment`, `hero sync attach`, `hero sync graph` |
| Cross-repo peering | `hero admin repos`, `hero peer manifest`, `hero peer list`, `hero peer show`, `hero peer call`, `hero handoff`, `hero handoff status`, `hero handoff accept`, `hero context imports` |
| Automation and headless work | `hero agent run`, `hero agent jobs`, `hero agent approve`, `hero agent automate`, `hero pipeline`, `hero watch` |
| Publishing and server | `hero serve`, `hero mcp`, `hero publish wiki`, `hero publish pages`, `hero login`, `hero logout` |
| Installation | `hero install`, `hero install satellites`, `hero upgrade`, `hero uninstall`, `hero verify-install`, `hero trust`, `hero domain` |

Useful examples:

```bash
hero status --all
hero list --ready --sort priority
hero queue --format kickoff
hero spec new csv-export
hero spec claim csv-export --agent codex
hero spec score csv-export
hero spec verify csv-export
hero spec complete .hero/planning/features/csv-export/spec.md
hero ac list csv-export
hero spec contract status csv-export
hero coverage csv-export
hero suggest --top 10
hero agent run deliver csv-export --dry-run
hero agent jobs
hero agent approve job-abc123
hero sync connect jira
hero sync import
hero sync pull .hero/planning/bugs/login-timeout/spec.md
hero publish pages
hero serve --add .
```

Use the grouped forms above for spec lifecycle, tracker sync, publishing,
and headless execution.

---

## MCP Tools

`hero mcp` is a hidden stdio server launched by AI tools. The current
tool set is:

`hero_context`, `hero_search`, `hero_status`, `hero_check`,
`hero_nudge`, `hero_list`, `hero_queue`, `hero_kickoff`,
`hero_knowledge`, `hero_read_spec`, `hero_ask`, `hero_anchor`,
`hero_pulse`, `hero_skill_run`, `hero_claim`, `hero_velocity`,
`hero_test_generate`, `hero_demo_record`, `hero_code`,
`hero_error_pattern`, `hero_enrich`, `hero_score`, `hero_diagnose`,
`hero_verify`, `hero_conflicts`, `hero_sequence`, `hero_warnings`,
`hero_insights`, `hero_contract`, `hero_plan`, `hero_impact`,
`hero_recap`, `hero_drift`, `hero_ci`, `hero_feed`, `hero_event`,
`hero_active`, `hero_coverage`, `hero_why`, `hero_blocked`,
`hero_expand`, `hero_snapshot`.

Most tools are read-only. Tools that intentionally mutate local state
include claim/event/plan/enrich/test/demo helpers.

---

## Workspace Layout

```text
.hero/
├── mission.md                  # project charter and first principles
├── NEXT.md                     # shared projected handoff in solo mode
├── QUEUE.md                    # ready-work queue for cold starts
├── SNAPSHOT.md                 # project-shape rollup (managed by `hero snapshot --project`)
├── planning/
│   ├── features/
│   ├── bugs/
│   └── initiatives/
├── specs/                      # completed specs
├── knowledge/
│   ├── conventions/
│   ├── decisions/
│   ├── rules/
│   ├── context/
│   ├── notes/
│   ├── templates/
│   └── external/
├── next/                       # per-user and local handoff projections
├── smoke/                      # per-feature smoke metadata
├── events.log                  # cross-session activity feed source
├── graph.db                    # generated graph store
├── index.db                    # generated search index
└── hero.json                   # project config
```

Specs and knowledge entries are committed. Generated state such as
`graph.db`, `index.db`, local overlays, and per-machine files are ignored.

---

## Repository Layout

```text
cmd/hero/                 CLI entrypoint
internal/                 Go implementation packages
internal/serve/           HTTP daemon and MCP server
internal/graph/           SQLite graph substrate and traversal support
internal/retrieval/       BM25/TF-IDF retrieval layer
internal/scan/            master ingest and codebase scanning
internal/traversal/       why/blocked graph queries
core/                     universal core pack (agents, commands, skills, vocabularies)
domains/engineering/      Hero Code engineering domain pack
domains/pm/               Hero PM domain pack
domains/sales/            scaffolded Hero Sales domain pack
web/                      public web surfaces (docs, landing)
cloud/                    team server and cloud backend
```

Engineering agents, commands, and skills live under
`domains/engineering/`. The install pipeline overlays the active domain
pack on top of the universal `core/` layer; domain wins on file
conflicts.

---

## Installed Content Inventory

Agents:

`api-engineer`, `architecture-reviewer`, `brownfield-architect`,
`comment-scrubber`, `convention-author`, `database-engineer`,
`deadcode-scrubber`, `debug-investigator`, `dedup-scrubber`,
`defensive-scrubber`, `dependency-analyst`, `dependency-scrubber`,
`design-reviewer`, `devops-engineer`, `documentation-engineer`,
`engineer`, `feature-delivery-lead`, `functional-qa-engineer`,
`greenfield-architect`, `integration-engineer`, `issue-tracker`,
`legacy-scrubber`, `migration-engineer`, `performance-engineer`,
`platform-delivery-lead`, `pr-reviewer`, `product-ideator`,
`project-context-builder`, `release-engineer`, `security-reviewer`,
`session-primer`, `test-architect`, `type-scrubber`, `ui-designer`.

Slash commands:

`blocked`, `capture`, `challenge`, `check`, `compose`, `convention`,
`decide`, `deliver`, `design`, `diagnose`, `discover`, `docs`,
`handoff`, `hero`, `import`, `mock`, `note`, `peer`, `prime`,
`release`, `resume`, `retro`, `review`, `scan`, `scrub`, `split`,
`sprint`, `why`.

Skills:

`agent-reliability`, `api-design-and-contracts`,
`architecture-principles`, `auto-knowledge-capture`,
`challenge-diagnosis`, `code-scrub`, `context-injection`,
`convention-writing`, `database-stack`, `debugging-investigation`,
`deep-code-enrichment`, `dependency-analysis`, `devops-and-operations`,
`documentation-practices`, `executive-report`, `go-stack`,
`greenfield-scaffolding`, `groovy-stack`, `html-mockup-generation`,
`implementation-principles`, `incident-response`,
`integration-boundaries`, `issue-list-report`, `java-stack`,
`javascript-stack`, `kickoff-prompt`, `knowledge-flywheel`,
`migration-safety`, `next-handoff-emit`, `next-md`, `note-capture`,
`nudge-awareness`, `performance-optimization`, `pr-review`,
`project-context-generation`, `python-stack`, `react-stack`,
`release-and-deployment`, `root-cause-classification`, `rust-stack`,
`security-review`, `spec-format`, `stack-detection`, `test-strategy`,
`testing-and-validation`.

---

## Configuration

Common `.hero/hero.json` fields:

```json
{
  "folder": ".hero",
  "team": {
    "auto_context": true,
    "nudge_level": "gentle",
    "stale_days": 14
  },
  "knowledge": {
    "auto_capture": true
  },
  "next": {
    "mode": "personal",
    "projected": true
  },
  "testing": {
    "framework": "playwright",
    "mode": "autonomous",
    "test_dir": "e2e"
  }
}
```

See [web/docs/src/configuration/hero-json.md](web/docs/src/configuration/hero-json.md)
for the full reference.

---

## More Docs

- [What Is Hero?](web/docs/src/what-is-hero.md) — plain-English overview
- [Why Hero](web/docs/src/why-hero.md) — deeper technical evaluation
- [Getting Started](GETTING-STARTED.md)
- [Docs Index](web/docs/src/index.md)
- [Commands Reference](web/docs/src/commands/index.md)
- [Project Structure](web/docs/src/project-structure.md)
- [MCP Setup](MCP-SETUP.md)
- [Team Server](TEAM-SERVER.md)
- [Cross-Repo Peering](CROSS-REPO-PEERING.md)

---

## Build

```bash
make build
make test
go build ./...
go test ./...
```

Hero requires Go 1.21+.
