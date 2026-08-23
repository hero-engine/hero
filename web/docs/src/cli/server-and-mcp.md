# Server and MCP

Hero exposes two shipped local project-intelligence surfaces:

| Surface | Command | Prerequisite | Boundary |
|---|---|---|---|
| MCP stdio | `hero mcp` | Installed harness MCP registration | One client process; tools filtered by current config |
| HTTP daemon | `hero serve` | Local daemon startup and registered project paths | Local by default; team/external access requires separate auth and setup |

## MCP stdio

Users normally do not launch `hero mcp` manually. `hero install project` writes
the target-specific registration:

```bash
hero install project . --target claude
hero install project . --target codex
hero install project . --target generic
```

Hero supports OpenCode, Cursor, Claude, GitHub Copilot, Codex, Generic MCP, and
Grok install targets. The generated workflow surface is harness-native: Claude
receives command files, while Codex and Grok receive command skills. Run
`hero install --help` and `hero doctor` for the current registry and installed
target inventory.

### Runtime tool registry

The authoritative MCP inventory is the server's `tools/list` response after
`serve.tool_filter` is applied. Tools are grouped by capability:

- project memory, search, knowledge, and provenance;
- spec lifecycle and verified delivery;
- planning, status, coverage, and quality;
- activity and code intelligence;
- Attention, Mail, and Focus;
- optional tracker and code-host broker operations.

Narrative documentation does not freeze a mutable tool count. See
[MCP tool metadata](../serve/mcp-tool-metadata.md) for category, tier, and
safety annotations.

### Tool filtering

`serve.tool_filter.allow` exposes only named tools, `deny` removes named tools,
and `profiles` maps a profile to a list of allowed names. The
[decoder-backed configuration example](../configuration/hero-json.md) shows the
production shape.

## Local HTTP daemon

```bash
hero serve
hero serve --add .
hero serve --list
hero serve --port 8080
hero serve --no-ui
hero serve status
hero serve stop
```

The default address is `http://localhost:7437`. The daemon provides the HTTP
API, dashboard, file watcher, and registry for local projects. Useful endpoints
include `/health`, `/api/projects`, and project-scoped status, spec, search,
context, check, and knowledge routes. See [Serve homes](../serve/homes.md) for
the browser navigation.

Team mode, workers, auth tokens, and any non-local exposure change the
operational boundary. Configure authentication and network controls before
enabling them; local availability is not evidence of a hosted team service.

## Attention API boundary

The user-global HTTP API under `/api/attention/v1` exposes bounded snapshots
and advertised row actions. Mail and Focus retain separate mutation contracts;
there is no generic mutable Attention endpoint. Clients refresh the snapshot on
mount, reconnect, foreground, and after a successful mutation. Mail bodies are
untrusted and require explicit reads.

## Optional external operations

Tracker and code-host tools appear only when their integrations and filters
allow them. Both require credentials and target identity. Provider setup is not
consent to mutate an issue, pull request, or repository; writes require explicit
operation-specific intent.

- [Tracker setup](../configuration/tracker-setup.md)
- [Code-host operations](code-host.md)

## Headless runtime

`hero agent` is **preview**. It requires a configured model provider,
credentials, and an execution environment:

```bash
hero agent run deliver auth-flow
hero agent jobs
hero agent approve <job-id>
```

Approval-aware jobs pause at protected gates. Preview status means the support
boundary still needs release-level validation; do not treat provider
credentials as permission for every external or destructive action.

## Operational checks

```bash
hero doctor   # binary/schema and installed target diagnosis
hero check    # workspace health
hero serve status
```

If a GUI-launched harness resolves a stale binary, run `hero doctor`. `hero
upgrade` regenerates workspace content; it does not replace a stale executable
on `PATH`.
