---
title: "Team Connect — CLI Registration with Team Server"
type: feature
status: delivering
received_from:
  peer_id: 5770cae7-b233-45c0-8e5d-765338a6058c
  peer_alias_display: hero
  originator_slug: team-connect
  handed_off_at: 2026-06-24T22:17:54Z
  at_commit: 923152b
  reason: "All changes are hero-side code (internal/cli/, internal/config/, internal/runner/). Needs hero connect/disconnect team commands, team.json storage, runner integration for team queue routing, MCP session registration."
---

# Team Connect — CLI Registration with Team Server

## Provenance

Handed off from peer `hero` (peer_id `5770cae7-b233-45c0-8e5d-765338a6058c`).
Originator spec: `team-connect`.

**Reason:** All changes are hero-side code (internal/cli/, internal/config/, internal/runner/). Needs hero connect/disconnect team commands, team.json storage, runner integration for team queue routing, MCP session registration.

## Context

_Scaffolded by `hero handoff`. Flesh out goal, design, and acceptance criteria before delivering._

## Acceptance Criteria

1. **AC-1: Session HTTP API** — `POST /api/sessions` registers a session (201), `GET /api/sessions` lists active sessions, `DELETE /api/sessions/{id}` unregisters (200). All backed by existing `JobQueue.RegisterSession`/`UnregisterSession`/`ActiveSessions`.
2. **AC-2: CLI auto-registration** — `hero run` registers a session with the team server on start and unregisters on exit (via defer). Best-effort: team server failures do not block the run.
3. **AC-3: `hero team sessions`** — New subcommand fetches `GET /api/sessions` and displays active sessions in a formatted table.
4. **AC-4: Tests pass** — `TestSessionsAPI_RegisterListUnregister`, `TestSessionsAPI_RegisterMissingID`, `TestSessionsAPI_UserFromHeader`, `TestSessionsAPI_MethodNotAllowed` all pass.

## Completion Ledger

| AC | File | Status |
|----|------|--------|
| AC-1 | `internal/serve/api_jobs.go` | done — `/api/sessions` and `/api/sessions/` handlers added to `RegisterJobsAPI` |
| AC-2 | `internal/cli/run.go` | done — `registerTeamSession`/`unregisterTeamSession` helpers, called in `runRun` with defer |
| AC-3 | `internal/cli/team.go` | done — `teamSessionsCmd` added, `runTeamSessions` displays table |
| AC-4 | `internal/serve/api_sessions_test.go` | done — 4 tests, all passing |

## Handoff Trail

- 2026-06-24T22:17:54Z — in ← hero-cloud (peer_id: 5770cae7-b233-45c0-8e5d-765338a6058c)
  mode: async-drop
  originating_spec: team-connect
  peer_spec: hero/team-connect
  at_commit: 923152b
  reason: "All changes are hero-side code (internal/cli/, internal/config/, internal/runner/). Needs hero connect/disconnect team commands, team.json storage, runner integration for team queue routing, MCP session registration."

