---
title: Project Overview
type: context
status: active
created: 2026-04-29
tags: [auto-generated, project-scan]
slug: project-overview
---

## What is Hero

Hero is the sidekick brain for AI-augmented knowledge work.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.26.4+ |
| Build | go build (`go.mod`) |
| Build | Make (`Makefile`) |
| Build | GoReleaser (`.goreleaser.yaml`) |
| Package manager | Go Modules |
| Testing | `go test` (387 test files) |
| CI/CD | GitHub Actions |

## Key Dependencies

- `BurntSushi/toml` v1.6.0
- `sprout/go` v0.1.0
- `google/uuid` v1.6.0
- `spf13/cobra` v1.10.2
- `go.uber.org/goleak` v1.3.0
- `x/crypto` v0.50.0
- `x/sys` v0.44.0
- `x/term` v0.42.0
- `gopkg.in/yaml.v3` v3.0.1
- `modernc.org/sqlite` v1.53.0

## Package Organization

- `bin/` — compiled binaries
- `cmd/` — CLI entry points (main packages) (hero, mock-tracker-server)
- `contracts/` — contains: governance, peering
- `core/` — contains: agents, commands, methodologies, skills, +2 more
- `docs/` — documentation (contracts)
- `domains/` — contains: chat, engineering, pm, sales
- `examples/` — contains: scrum-workspace
- `internal/` — internal packages (not exported) (acceptance, active, async, automations, +68 more)
- `scripts/` — build and utility scripts (drive, e2e, smoke)
- `testdata/` — contains: proposals
- `tmp/` — contains: e2e
- `tools/`
- `web/` — web assets and frontend code (docs, landing)

### Internal Packages

- `cmd/mock-tracker-server/` — Command mock-tracker-server is a single-binary, offline HTTP fake that speaks the GitHub / Jira / Linear / GitLab API subset hero's tracker adapters call, backed by an in-memory SQLite DB seeded by…
- `internal/active/` — manages a registry of active spec sessions so that context injection and compaction recovery can prioritize the right spec.
- `internal/clusters/` — detects "work clusters" — recurring shapes in a project's recent activity that an operator should notice. It extends the knowledge-flywheel pattern-detection thesis (which clusters captured notes /…
- `internal/cost/` — provides effort calibration by comparing estimated vs actual delivery signals from the completed spec corpus and git history.
- `internal/coverage/` — maps spec acceptance criteria to test files, reporting which criteria have test coverage and which are gaps.
- `internal/demos/` — provides pluggable demo recording from Hero spec test files.
- `internal/digest/` — is hero's per-turn context digester.  The principle: hero captures everything (the graph is unbounded); the model sees a bounded, ranked, pruned brief tailored to the current turn. As the corpus…
- `internal/driveio/` — wires the pure /drive judge (internal/drive) to the sqlite-backed index (internal/index). It exists solely so internal/drive stays free of an internal/index import: drive.Check takes an injected,…
- `internal/embeddings/defaultmodel/` — embeds the hero-embed-v1 model weights into the binary. The files are produced by tools/distill-embeddings.py (one-time export from minishlab/potion-base-8M, pruned of dead subword tokens).
- `internal/errpattern/` — manages a catalog of common error patterns accumulated from diagnose sessions. Patterns are stored as markdown files with YAML-like frontmatter under .hero/knowledge/error-patterns/.
- `internal/feed/` — provides a cross-session activity feed built on .hero/events.log. It extends the existing ClaimEvent format with richer event types while remaining backward-compatible with the old format.
- `internal/gitutil/` — provides Git helper functions for status reconciliation. All functions shell out to git and gracefully return empty results if git is unavailable or the directory is not a git repository.
- `internal/handoff/` — is the read/write layer for the per-user durable handoff state — Last user ask, Suggested next prompt, in-flight Session reflections.  These are the agent-narrative pieces of NEXT.md that…
- `internal/hooks/` — claude_settings.go — read, mutate, and write `.claude/settings.json` host-tool hook entries for Hero's compact-handoff integration.  Design constraints (from next-compact-handoff spec):  - Preserve…
- `internal/managed/` — owns the single Hero-managed region inside user-owned files (AGENTS.md, CLAUDE.md, NEXT.md, and future harness instruction files).  Two concerns live here:  1. The low-level marker primitives…
- `internal/memory/` — ingests per-project Claude memory files (~/.claude/projects/<project-key>/memory/) into the unified knowledge graph as Memory nodes scoped local — these never leave the machine.
- `internal/mission/` — parses .hero/mission.md (the project charter) and upserts it as a Mission graph node so every context-emitting command can inject the mission as the highest-priority block in every agent session. …
- `internal/nextdoc/` — parses .hero/NEXT.md and writes its session-scoped signals into the knowledge graph.  NEXT.md is a per-session projection. The "Tried and failed" section is the most graph-worthy: each bullet is an…
- `internal/propose/` — implements the inline-propose contract — the envelope schema, in-memory proposal store, and lifecycle bookkeeping the daemon needs to mediate between agents and the dashboard.  The full wire…
- `internal/reconcile/` — compares spec status fields against git evidence and produces findings about status drift.
- `internal/refs/` — implements the session-scoped ref-store backing two-tier MCP responses. A ref is a stable handle returned by a read-side tool that points to either cached content or the args needed to re-fetch it.…
- `internal/retrieval/` — provides a unified facade over the graph and FTS5 search indexes. Every caller that needs ranked content goes through here; engine- specific SQL is contained in this package rather than scattered…
- `internal/score/` — provides heuristic quality scoring for specs. It evaluates completeness, clarity, and deliverability without requiring an LLM — scoring is fast, deterministic, and local.
- `internal/serve/chat/` — implements the hero serve chat dispatcher.  hero serve does not run inference. Every chat turn either resolves to a runner-free slash (executed inline here) or dispatches to a connected Hero adapter…
- `internal/serve/edition/` — resolves the active Hero edition from the HERO_EDITION environment variable and exposes a small set of helpers the shell uses to gate routes and top-nav tabs per the deployment-and-rendering decision…
- `internal/serve/healthcache/` — holds the in-process per-project cache of `hero check` results and peer reachability probes that backs the /p/<slug>/project Health and Peers sections.  Phase 5 of the hero-serve-project-section…
- `internal/serve/mdrender/` — is a tiny block-level markdown-to-HTML renderer. It exists so the serve homes (Knowledge entry detail, Work spec detail) can render persisted spec / knowledge markdown without pulling in a…
- `internal/serve/opsrunner/` — runs an allowlisted set of `hero` CLI verbs as subprocesses on behalf of the Project section's Operations panel, streaming progress to the browser over SSE.  The package is intentionally small and…
- `internal/serve/pages/agentspage/` — hosts the Agents home — the autonomy surface at GET /agents. It composes the shell's page chrome (top nav, sub-nav, page-hero fragment, tabbed-metric-strip fragment, footer) with Agents-specific…
- `internal/serve/pages/knowledge/` — hosts the Knowledge home — the corpus surface at GET /knowledge. It composes the shell's page chrome (top nav, sub-nav row, page-hero fragment, tabbed-metric-strip fragment, footer) with…
- `internal/serve/pages/now/` — hosts the Now home — the personal cold-start surface at GET /now. It composes the shell's page chrome (top nav, page-hero fragment, tabbed-metric-strip fragment, chat-input fragment, footer) with…
- `internal/serve/pages/people/` — hosts the People & ROI home — the team-pulse and canonical Hero-ROI surface at GET /people. It composes the shell's page chrome (top nav, sub-nav row, page-hero, tabbed-metric-strip, footer) with…
- `internal/serve/pages/rollup/` — hosts the Rollup home — the project-shape rollup at GET /rollup. The home renders the surfaces table, active initiatives, recently-completed work, what's next, open risks, and (when archives exist)…
- `internal/serve/pages/work/` — hosts the Work home — the spec-and-delivery surface at GET /work. It composes the shell's page chrome (top nav, page-hero fragment, tabbed-metric-strip fragment, footer) with Work-specific section…
- `internal/serve/session/` — manages per-user shell state for hero serve: which home a user last visited, and which sub-nav tab they last picked inside any sub-nav-bearing home. Backed by a small SQLite database at…
- `internal/serve/shell/` — hosts the slim web-app chrome (top nav, page router, shared fragments) that every hero serve home rides on. The shell owns its own embedded assets — templates and static CSS / SVG — so no other…
- `internal/sitegen/` — renders the unified knowledge graph as a static HTML site — phase 9's "publish a living team narrative" pillar.  Output goes to a directory (default ./site) that's deployable as GitHub Pages,…
- `internal/sync/` — implements the shared-field 3-way merge and the persisted last-synced baseline that the merge reconciles against.  The baseline is the common ancestor for a 3-way merge: the last value of each synced…
- `internal/templates/` — extracts spec authoring patterns from the completed spec corpus and generates learned templates for hero new.
- `internal/traversal/` — implements multi-hop graph queries that read across subgraphs — the showcase that justifies the v2 graph substrate over flat files.  `hero why <target>` answers "where did this come from": resolves…
- `internal/version/` — manages Hero workspace version tracking, mismatch detection, and file checksum tracking for smart upgrades.
- `internal/watch/` — provides polling-based file change detection for the hero workspace. It monitors .hero/ directory for spec changes, triggering reindex and health checks.
- `internal/workspace/` — resolves the active Hero workspace root and scope from any directory. It centralizes the walk-up logic so satellite folders, subfolders of a workspace, and direct workspace dirs all behave the same…

## Project Structure

Entry points:
- `cmd/hero/main.go`
- `cmd/mock-tracker-server/main.go`

Go module: `github.com/hero-engine/hero`

## Documentation

- `README.md`
- `GETTING-STARTED.md`

## Architecture Summary

Detailed architecture documentation is available in the architecture-overview knowledge entry.

- `<harness>/commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)
- `<harness>/agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)
- `<harness>/skills/` — Domain-specific knowledge and patterns (each skill is a subdir with SKILL.md)
- `.hero/planning/` — Active specs being worked on
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)
-…

## Current Gaps

- **No linters** — no linter or formatter configuration detected
- **No Dockerfile** — no containerized build/deploy configuration

<!-- Add project-specific context here:
- Architecture overview and key design patterns
- Deployment topology (cloud provider, regions, etc.)
- Important environment variables
- Third-party service dependencies
-->
