# Getting Started with Hero

Hero is a spec-driven workflow and context engine for AI-augmented
engineering. The habit is simple: **design before you build, diagnose
before you fix, hand off before context disappears**.

This guide covers the current commands in the binary and the installed
harness content.

---

## 1. Install

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

All install options — direct downloads, build-from-source, version pinning —
are documented in [web/docs/src/getting-started/installation.md](web/docs/src/getting-started/installation.md).

Verify:

```bash
hero --version
hero --help
```

---

## 2. Initialize a Project

From the project root:

```bash
hero init
```

This creates `.hero/`, including planning folders, knowledge folders,
configuration, graph/search stores, and handoff surfaces. Commit the
workspace files that contain specs and knowledge.

Install Hero into your AI coding tool:

```bash
hero install project . --target opencode
hero install project . --target cursor
hero install project . --target claude
hero install project . --target codex
hero install project . --target copilot
hero install project . --target generic
```

The installer copies 27 slash command definitions, 34 agents, and 45
skills from the active **domain pack**, then registers the `hero mcp`
server so the harness can call Hero's 41 MCP tools.

Hero ships content in layers: a shared `core/` pack and one or more
domain packs (`domains/engineering/` for engineering teams,
`domains/sales/` scaffolded for sales). The active domain comes from
`hero.json` or `--domain`:

```bash
hero install project . --target claude --domain engineering
hero domain                    # show / switch active domain
```

For monorepos where the harness runs from a sub-folder, register each
sub-folder as a satellite of the root install:

```bash
hero install project . --target cursor --workspace services/auth
hero install satellites list
hero install satellites add services/auth
hero install --repair          # verify symlinks/markers
```

Other useful install flags:

```bash
hero install --migrate         # reconcile drifted copies across harnesses
hero install --no-touch-claude-md  # leave CLAUDE.md alone
hero verify-install            # audit install state
```

---

## 3. Seed Context

In your AI tool, run:

```text
/scan
```

Or from a terminal:

```bash
hero scan
```

`hero scan` detects the stack, indexes code symbols, ingests planning
and knowledge files, updates the graph, and opportunistically syncs
team state when configured.

Capture a few high-value conventions:

```text
/convention codify our API error response format
/convention document our test naming pattern
```

Those entries become reusable context for future sessions.

---

## 4. Start Every Session Warm

```text
/resume
```

`/resume` calls the graph-backed session brief: mission, in-flight work,
recent changes, blockers, relevant conventions, and dead ends to skip.
Natural language such as "pick up where we left off" or "catch me up"
routes here too.

Before switching tools or compacting context:

```text
/handoff
```

This refreshes NEXT state so the next session starts with the latest
ask, current direction, blockers, and machine state.

---

## 5. Build a Feature

Design first:

```text
/design add CSV export for user data
```

Hero writes a spec under `.hero/planning/features/<slug>/spec.md` with
context, goals, approach, acceptance criteria, and a `## Kickoff`
section for cold-starting future sessions.

Deliver against the spec:

```text
/deliver .hero/planning/features/csv-export/spec.md
```

The delivery workflow loads relevant conventions, past work, risks, and
acceptance criteria, then coordinates implementation and verification.

Useful CLI checks:

```bash
hero spec score csv-export
hero spec verify csv-export
hero diff .hero/planning/features/csv-export/spec.md
hero drift csv-export
hero ac list csv-export
hero coverage csv-export
```

Complete a delivered spec:

```bash
hero spec complete .hero/planning/features/csv-export/spec.md
```

---

## 6. Fix a Bug

Diagnose first:

```text
/diagnose login times out after 30 seconds
```

Hero investigates the root cause and creates a bug/fix spec under
`.hero/planning/bugs/<slug>/spec.md`.

Then deliver:

```text
/deliver .hero/planning/bugs/login-timeout/spec.md
```

For imported tracker bugs, sync before starting:

```bash
hero sync pull .hero/planning/bugs/login-timeout/spec.md
```

---

## 7. Query the Corpus

```bash
hero ask "what is our retry policy?"
hero search "OAuth session handling"
hero relevant src/auth/session.go src/auth/middleware.go
hero impact internal/auth/session.go
hero recap --since 2d
hero why csv-export
hero blocked
hero queue --format kickoff
```

`hero relevant` is the current CLI command for file-aware context.
The MCP tool is still named `hero_nudge` because it returns nudge-style
context to agents.

---

## 8. Track and Coordinate Work

```bash
hero status
hero list --ready --sort priority
hero spec claim csv-export --agent codex
hero spec claim csv-export --release
hero spec claims
hero feed --since 1h
hero next
hero next team
```

For dependency and graph state:

```bash
hero graph csv-export
hero graph stats
hero graph reingest specs
hero ac status
```

Working across sibling repos (backend + web client + desktop, etc.):
register each peer and use sync calls or async handoffs to coordinate
across workspaces.

```bash
hero repos add app ../app
hero peer call app --mode=advisory "What's your error envelope?"
hero handoff order-failure app --reason "Root cause is the API"
```

See [CROSS-REPO-PEERING.md](CROSS-REPO-PEERING.md) for the full setup
and three-tier ladder.

---

## 9. Tracker, Wiki, and Cloud

Connect a tracker:

```bash
hero sync connect jira
hero sync connect github
hero sync connect linear
```

Sync issues and specs:

```bash
hero sync import
hero sync spec .hero/planning/features/csv-export/spec.md
hero sync link .hero/planning/features/csv-export/spec.md PROJ-123
hero sync pull .hero/planning/features/csv-export/spec.md
hero sync comment PROJ-123 "Root cause found"
hero sync attach PROJ-123 diagnosis.md
```

Publish outward:

```bash
hero publish wiki .hero/specs/csv-export/spec.md
hero publish pages
```

Cloud/team graph sync:

```bash
hero login
hero sync graph push
hero sync graph pull
hero logout
```

---

## 10. Headless Work and Automations

Headless agent execution now lives under `hero agent`:

```bash
hero agent run deliver csv-export --dry-run
hero agent run diagnose login-timeout --provider openai
hero agent jobs
hero agent jobs <job-id>
hero agent approve <job-id>
hero agent automate list
```

Batch pipeline:

```bash
hero pipeline
hero pipeline --run diagnose
hero pipeline --run deliver
```

---

## 11. MCP and Server

`hero install` configures MCP automatically. Manual command:

```bash
hero mcp
```

This is a stdio server launched by the harness, not something you
normally run yourself.

For dashboard/API/team mode:

```bash
hero serve
hero serve --add .
hero serve --list
hero serve --team --workers 2
```

---

## 12. Configuration

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

Generated/local files usually ignored:

```gitignore
.hero/index.db
.hero/graph.db
.hero/hero.local.json
.hero/next/*.local.md
.hero/sessions/
```

---

## Quick Reference

| I want to... | Run this |
|---|---|
| Warm up a session | `/resume` |
| Preserve handoff state | `/handoff` |
| Design a feature | `/design <description>` |
| Diagnose a bug | `/diagnose <description>` |
| Deliver work | `/deliver <spec-path-or-slug>` |
| Route natural language | `/hero <request>` or `hero do "<request>"` |
| Scan a repo | `/scan` or `hero scan` |
| Search the corpus | `hero search "<query>"` |
| Ask a question | `hero ask "<question>"` |
| Get file-aware context | `hero relevant <paths>` |
| See ready work | `hero queue --format kickoff` |
| Create a spec from CLI | `hero spec new <slug>` |
| Claim a spec | `hero spec claim <slug>` |
| Verify a spec | `hero spec verify <slug>` |
| Complete a spec | `hero spec complete <spec-path>` |
| Find blockers | `hero blocked` |
| Trace origins | `hero why <target>` |
| Generate tests | `hero test generate <slug>` |
| Record a demo | `hero spec demo record <slug>` |
| Find high-churn undocumented areas | `hero suggest --top 10` |
| Run headlessly | `hero agent run deliver <slug>` |
| Validate docs | `hero docs check` |
| Upgrade installed harness files | `hero upgrade` |
