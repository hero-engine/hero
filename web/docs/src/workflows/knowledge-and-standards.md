# Knowledge & Standards

Hero maintains a persistent knowledge base in `.hero/knowledge/` that grows
with your project. These commands help you capture, organize, and query
project knowledge.

---

## `/convention` — Codify Patterns

Analyzes a codebase pattern and produces a convention spec documenting the
canonical approach. Delegated to the **convention-author** agent.

```bash
# Document an existing pattern
/convention error handling in API handlers

# Standardize a fragmented pattern
/convention how we write database migrations

# Naming conventions
/convention file and function naming in the services layer
```

The convention author:

1. Searches the codebase for existing instances of the pattern
2. Assesses whether usage is consistent or fragmented
3. Identifies or synthesizes the canonical approach
4. Produces a convention spec with examples, anti-patterns, and adoption notes

Conventions are saved to `.hero/conventions/{slug}/spec.md` and are
automatically loaded by delivery leads during `/deliver`.

---

## `/note` — Capture Ideas

Saves the current conversation as a note in the knowledge base. Use this for
brainstorms, design explorations, debugging sessions, and architecture
discussions.

```bash
# Capture with a topic
/note auth flow brainstorm

# Capture without a topic (Hero will ask for a title)
/note
```

Notes preserve **both sides** of the conversation — the user's input and the
AI's analysis. They're stored at `.hero/knowledge/notes/{slug}/spec.md` with
tags for discoverability.

!!! tip "Don't lose good thinking"
    Run `/note` before ending a session that produced valuable back-and-forth.
    Chat history disappears; notes persist.

---

## `/capture` — Extract Session Learnings

Reviews the current session and identifies knowledge worth persisting. Unlike
`/note` (which captures conversation), `/capture` extracts **insights** —
conventions discovered, decisions made, gotchas found, and rules established.

```bash
/capture
```

For each learning, `/capture`:

1. Classifies it (convention, decision, rule, context, or note)
2. Checks for duplicates via `hero search`
3. Writes the entry to the appropriate path under `.hero/knowledge/`
4. Runs `hero index` to make it searchable

!!! note "Capture threshold"
    Not every session produces knowledge. `/capture` only persists learnings
    that would save someone time, prevent a mistake, or record a non-obvious
    decision. If nothing meets the bar, it reports "No new knowledge to
    capture from this session."

### What gets captured

| Category | Example |
|---|---|
| Conventions | "FTS5 MATCH chokes on slugs with hyphens — use table scan for exact lookups" |
| Decisions | "Chose `modernc.org/sqlite` over CGo for pure-Go portability" |
| Gotchas | "Cobra's `--help` flag processing corrupts subcommand flag state" |
| Rules | "No new Go dependencies — implement protocols with stdlib" |
| Context | "The hero binary is a corpus manager; AI work happens in the agent tool" |

---

## `/scan` — Stack Detection & Knowledge Seeding

Scans the codebase to detect the technology stack and generates initial
knowledge base entries — project overview, conventions, rules, and context.

```bash
# Preview what will be detected
/scan --dry-run

# Generate knowledge entries
/scan

# Re-scan after stack changes
/scan --force
```

Generated entries include:

- **project-overview** (context) — languages, frameworks, architecture
- **use-\*** (conventions) — linter configs, formatter settings, build tools
- **ci-\*** (rules) — CI pipeline requirements, bypass policies
- **testing-with-\*** (conventions) — test frameworks, coverage expectations

!!! tip "Enrich after scanning"
    `/scan` generates stubs. Review each entry and add project-specific
    details — architecture diagrams, deployment topology, team structure,
    and exception policies.

---

## `/check` — Workspace Health

Runs a workspace health check covering convention compliance, stale specs, and
project hygiene.

```bash
# General health check
/check

# Convention compliance (deep — uses architecture-reviewer)
/check conventions

# Stale spec detection
/check stale

# Dependency audit (uses dependency-analyst)
/check deps
```

| Check type | Method |
|---|---|
| General health | Direct analysis of planning folder and knowledge base |
| Convention compliance | Delegated to `architecture-reviewer` |
| Stale specs | Scans `.hero/planning/` for inactive specs |
| Dependencies | Delegated to `dependency-analyst` |

For quick automated checks from the terminal, use `hero check` directly.

---

## `/docs` — Technical Documentation

Routes documentation requests to the appropriate specialist:

| Request type | Agent |
|---|---|
| Project context, `AGENTS.md`, repo instructions | `project-context-builder` |
| Technical docs, API docs, operational docs | `documentation-engineer` |

```bash
# Update project context
/docs update AGENTS.md with the new authentication flow

# Generate API documentation
/docs document the REST API endpoints in services/api

# Operational runbook
/docs write a runbook for the database failover process
```

---

## Auto-Capture

After major workflows (`/design`, `/deliver`, `/diagnose`, `/retro`), Hero
silently reviews the session for novel learnings. If anything meets the capture
threshold, it writes entries to `.hero/knowledge/` and runs `hero index`
without prompting.

This is controlled by the `knowledge.auto_capture` setting in `hero.json`
(enabled by default). You'll see a brief mention of what was saved at the end
of each workflow.

---

## `hero ask` — Query the Knowledge Base

Ask questions against the accumulated knowledge base.

```bash
# Find relevant knowledge
hero ask "how do we handle authentication tokens"

# Check for existing conventions
hero ask "error handling patterns"
```

Hero searches across all knowledge entries — conventions, decisions, rules,
context, and notes — and returns relevant matches with source references.
