---
type: convention
status: draft
scope: ["internal/serve/**", "internal/cli/serve*.go"]
tags: [serve, daemon, lifecycle, pid-file]
relations:
  - target: hero-serve-multi-project
    kind: introduced-by
---

# `hero serve` PID File Location and Format

## Pattern

The `hero serve` daemon writes a PID file alongside the existing project
registry under `~/.hero/`:

- Default port (7437): `~/.hero/serve.pid`
- Non-default port: `~/.hero/serve-<port>.pid`

The file contents are **JSON**, not a bare PID:

```json
{"pid": 12345, "port": 7437, "started_at": "2026-05-19T10:30:00Z", "version": "0.x.y"}
```

## When to apply

Any code that starts, stops, probes, or inspects the local `hero serve`
daemon process.

## How

- Resolve the directory via `registryDir()` in
  `internal/serve/registry.go` — same directory as `projects.json` so
  there's one global hero config location.
- Write the file immediately after `net.Listen` succeeds in
  `internal/serve/server.go` (`Run()`).
- Remove the file in the shutdown path after `httpServer.Shutdown`
  returns. Tolerate already-removed (no error on missing file).
- Per-port filename suffix lets two daemons on different ports coexist
  (default + scratch on `--port 7438`, for example).

## Why JSON over a bare PID

`hero serve status` should be able to print something useful even when
the daemon is wedged and not answering HTTP. Embedding port,
`started_at`, and version directly in the PID file removes the round-
trip dependency.

The HTTP `/api/status` endpoint remains the source of truth for live
state (project list, uptime, etc.) — the PID file is just enough to
locate and identify the process.

## Anti-patterns

- Writing only a bare PID (loses port/version, forces HTTP probe for
  trivial inspection).
- Writing to `/tmp` or `/var/run` (cross-platform pain; doesn't survive
  reboot in a useful way; users expect `~/.hero/`).
- Skipping cleanup on shutdown (stale files are tolerable but increase
  the burden on `status`/`stop` probe logic).

## Exceptions

Tests that spin up an in-process server may skip the PID file entirely
by passing a flag/option that disables file writes. Production code
always writes the file.
