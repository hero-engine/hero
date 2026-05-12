---
title: Project Overview
type: context
status: active
created: 2026-04-29
tags: [auto-generated, project-scan]
---

## What is Hero

Spec-driven AI engineering workflow. Design before you build, diagnose before you fix.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.26.1+ |
| Build | go build (`go.mod`) |
| Build | Make (`Makefile`) |
| Build | GoReleaser (`.goreleaser.yaml`) |
| Package manager | Go Modules |
| Testing | `go test` (137 test files) |
| CI/CD | GitHub Actions |

## Key Dependencies

- `pgx/v5` v5.9.2
- `spf13/cobra` v1.10.2
- `gopkg.in/yaml.v3` v3.0.1
- `modernc.org/sqlite` v1.48.2

## Package Organization

- `agents/`
- `cloud/` — contains: api, auth, github, middleware, +3 more
- `cmd/` — CLI entry points (main packages) (hero, hero-cloud)
- `commands/`
- `core/` — contains: agents, commands, skills
- `dist/` — distribution output (hero_darwin_amd64_v1, hero_darwin_arm64_v8.0, hero_linux_amd64_v1, hero_linux_arm64_v8.0, +3 more)
- `docs/` — documentation (agents, cli, commands, concepts, +3 more)
- `domains/` — contains: engineering, sales
- `internal/` — internal packages (not exported) (acceptance, active, async, automations, +49 more)
- `scripts/` — build and utility scripts (e2e, smoke)
- `skills/`
- `tmp/` — contains: e2e, e2e-smoke

### Internal Packages

- `cloud/auth/` — provides JWT token generation, validation, and OAuth flows.  github.go implements the GitHub OAuth2 authorization code flow.
- `cloud/github/` — provides GitHub App integration for Hero Cloud.  A GitHub App authenticates using a private key to generate JWTs, then exchanges them for installation access tokens scoped to specific repositories.…
- `cloud/middleware/` — provides HTTP middleware for Hero Cloud.
- `internal/active/` — manages a registry of active spec sessions so that context injection and compaction recovery can prioritize the right spec.
- `internal/cost/` — provides effort calibration by comparing estimated vs actual delivery signals from the completed spec corpus and git history.
- `internal/coverage/` — maps spec acceptance criteria to test files, reporting which criteria have test coverage and which are gaps.
- `internal/demos/` — provides pluggable demo recording from Hero spec test files.
- `internal/digest/` — is hero's per-turn context digester.  The principle: hero captures everything (the graph is unbounded); the model sees a bounded, ranked, pruned brief tailored to the current turn. As the corpus…
- `internal/errpattern/` — manages a catalog of common error patterns accumulated from diagnose sessions. Patterns are stored as markdown files with YAML-like frontmatter under .hero/knowledge/error-patterns/.
- `internal/feed/` — provides a cross-session activity feed built on .hero/events.log. It extends the existing ClaimEvent format with richer event types while remaining backward-compatible with the old format.
- `internal/gitutil/` — provides Git helper functions for status reconciliation. All functions shell out to git and gracefully return empty results if git is unavailable or the directory is not a git repository.
- `internal/handoff/` — is the read/write layer for the per-user durable handoff state — Last user ask, Suggested next prompt, in-flight Session reflections.  These are the agent-narrative pieces of NEXT.md that…
- `internal/herotest/` — provides pluggable test generation from Hero spec acceptance criteria.
- `internal/knowledge/` — ingests files from .hero/knowledge/ into the unified knowledge graph. Today this covers raw/ — the immutable audit copy of `hero ingest` source bytes — which become Document nodes whose key is…
- `internal/memory/` — ingests per-project Claude memory files (~/.claude/projects/<project-key>/memory/) into the unified knowledge graph as Memory nodes scoped local — these never leave the machine.
- `internal/mission/` — parses .hero/mission.md (the project charter) and upserts it as a Mission graph node so every context-emitting command can inject the mission as the highest-priority block in every agent session. …
- `internal/nextdoc/` — parses .hero/NEXT.md and writes its session-scoped signals into the knowledge graph.  NEXT.md is a per-session projection. The "Tried and failed" section is the most graph-worthy: each bullet is an…
- `internal/projection/` — renders graph state into the user-visible markdown surfaces hero already exposes: NEXT.md, sprint views, future GitHub Pages output, etc.  Projections are deterministic and fast (<100ms target). They…
- `internal/reconcile/` — compares spec status fields against git evidence and produces findings about status drift.
- `internal/retrieval/` — provides a unified facade over the graph and FTS5 search indexes. Every caller that needs ranked content goes through here; engine- specific SQL is contained in this package rather than scattered…
- `internal/score/` — provides heuristic quality scoring for specs. It evaluates completeness, clarity, and deliverability without requiring an LLM — scoring is fast, deterministic, and local.
- `internal/sitegen/` — renders the unified knowledge graph as a static HTML site — phase 9's "publish a living team narrative" pillar.  Output goes to a directory (default ./site) that's deployable as GitHub Pages,…
- `internal/templates/` — extracts spec authoring patterns from the completed spec corpus and generates learned templates for hero new.
- `internal/traversal/` — implements multi-hop graph queries that read across subgraphs — the showcase that justifies the v2 graph substrate over flat files.  `hero why <target>` answers "where did this come from": resolves…
- `internal/version/` — manages Hero workspace version tracking, mismatch detection, and file checksum tracking for smart upgrades.
- `internal/watch/` — provides polling-based file change detection for the hero workspace. It monitors .hero/ directory for spec changes, triggering reindex and health checks.

## Project Structure

Entry points:
- `cmd/hero/main.go`
- `cmd/hero-cloud/main.go`
- `tmp/e2e-smoke/go-task-task/cmd/release/main.go`

Go module: `github.com/hero-engine/hero`

## Documentation

- `README.md`
- `GETTING-STARTED.md`

## Architecture Summary

Detailed architecture documentation is available in the architecture-overview knowledge entry.

- `commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)
- `agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)
- `skills/` — Domain-specific knowledge and patterns
- `.hero/planning/` — Active specs being worked on
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)
- `hero.json` — Project configuration

## Current Gaps

- **No linters** — no linter or formatter configuration detected
- **No Dockerfile** — no containerized build/deploy configuration

<!-- Add project-specific context here:
- Architecture overview and key design patterns
- Deployment topology (cloud provider, regions, etc.)
- Important environment variables
- Third-party service dependencies
-->
