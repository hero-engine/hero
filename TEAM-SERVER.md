# Hero Team Server

Hero works locally by default. Team mode adds a shared server for graph
sync, cross-session activity, job queue workers, user management, and a
dashboard/API surface.

Start locally:

```bash
hero serve --team --workers 2
```

With token auth:

```bash
hero serve --team --auth-token "$HERO_TEAM_TOKEN"
```

Default address: `http://localhost:7437`.

---

## Project Registry

Register projects with the daemon:

```bash
hero serve --add .
hero serve --add /path/to/project
hero serve --list
hero serve --remove my-project
```

The daemon serves registered projects through dashboard/API endpoints
and file watchers.

---

## Admin

```bash
hero admin team status
hero admin users list
hero admin users add alice
hero admin users remove alice
```

Use `hero login` / `hero logout` for Hero Cloud credentials when using
the managed service.

---

## Graph Sync

```bash
hero sync graph push
hero sync graph pull
hero sync graph status
```

`hero scan` also performs opportunistic team-server sync when a team
server is configured.

---

## Headless Jobs

Headless execution lives under `hero agent`:

```bash
hero agent run deliver csv-export --dry-run
hero agent run diagnose login-timeout --provider openai
hero agent jobs
hero agent jobs <job-id>
hero agent approve <job-id>
hero agent jobs submit deliver csv-export
```

Team mode routes submitted jobs through the server queue and workers.

---

## Handoff and Activity

```bash
hero next
hero next team
hero feed --since 1h
hero recap --since 2d
```

Team mode can render per-user handoff briefings and ingest meaningful
events into the shared graph. "Tried and failed" entries and explicit
`hero agent events` / MCP `hero_event` calls become reusable context for
future sessions.

---

## Publishing

```bash
hero publish pages
hero publish wiki .hero/specs/csv-export/spec.md
```

`publish pages` renders a static site from graph/project state.
`publish wiki` pushes completed specs to a configured wiki target.

---

## HTTP API

Useful endpoints:

| Method | Path |
|---|---|
| `GET` | `/health` |
| `GET` | `/api/projects` |
| `GET` | `/api/{project}/status` |
| `GET` | `/api/{project}/specs` |
| `GET` | `/api/{project}/specs/:slug` |
| `GET` | `/api/{project}/search?q=` |
| `GET` | `/api/{project}/context?f=` |
| `GET` | `/api/{project}/check` |
| `GET` | `/api/{project}/knowledge` |
| `GET` | `/api/events` |

---

## Local-Only Is Still Valid

Hero's default local mode requires no server. Use team mode only when
you want shared graph sync, centralized activity, or server-backed
headless jobs.
