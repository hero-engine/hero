# Hero Team Server and Headless Runtime

Hero works locally without a server. `hero serve` provides shipped local
project intelligence: an HTTP API, dashboard, file watcher, and a registry of
local Hero projects.

Team mode, shared workers, and headless agent execution are **preview**. Their
implementation is available, but support and public-access boundaries are not
release-complete. Use them only with deliberate authentication, network,
provider, credential, and execution configuration. Do not treat preview as
unsupervised or production-ready execution.

Hero Cloud is a separate proprietary product and is not implemented by a
`cloud/` tree in this repository.

## Local project intelligence — shipped

Register projects and start the local daemon:

```bash
hero serve --add .
hero serve --add /path/to/project
hero serve --list
hero serve
```

The default address is `http://localhost:7437`. Inspect or stop the daemon with:

```bash
hero serve status
hero serve stop
```

Local endpoints include:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Process health |
| `GET` | `/api/projects` | Registered projects |
| `GET` | `/api/{project}/status` | Workspace summary |
| `GET` | `/api/{project}/specs` | Spec listing |
| `GET` | `/api/{project}/specs/:slug` | One spec |
| `GET` | `/api/{project}/search?q=` | Full-text search |
| `GET` | `/api/{project}/context?f=` | File-aware context |
| `GET` | `/api/{project}/check` | Health results |
| `GET` | `/api/{project}/knowledge` | Knowledge entries |
| `GET` | `/api/events` | Server-sent event stream |

Registering a path or starting a watcher is a local mutation. Review which
projects are registered before exposing the daemon beyond the local machine.

## Team mode — preview

Start a local preview instance with queue workers:

```bash
hero serve --team --workers 2
```

If the endpoint crosses a trusted local boundary, require authentication. Have
the process manager inject `HERO_AUTH_TOKEN` into the service environment, then
run Hero without placing the secret in argv or committed configuration:

```bash
# HERO_AUTH_TOKEN is injected by the process manager.
hero serve --team --workers 2
```

Team administration and connection inspection are available through:

```bash
hero admin team status
hero admin team sessions
hero admin team usage
hero admin users
```

User creation and password operations are mutations and may prompt for protected
input:

```bash
hero admin users add alice --email alice@example.com --role member
hero admin users passwd alice
hero admin users remove alice
```

Do not put passwords in command arguments.

## Headless jobs — preview

Headless execution requires a configured Anthropic, OpenAI, or Azure provider,
credentials, and a supported execution environment. Start with a dry run:

```bash
hero agent run deliver csv-export --dry-run
hero agent jobs
hero agent jobs <job-id>
```

Team workers can consume explicitly submitted queue jobs:

```bash
hero agent jobs submit deliver csv-export
```

Direct `hero agent run` jobs and team queue jobs currently use separate stores.
No end-to-end path that pauses a running job for approval and later resumes it
through the queue is documented or shipped. Dry-run success proves command
planning only; it does not prove provider credentials, tool authorization,
repository permissions, or a production-ready worker environment.

## Graph sync boundary

Cloud graph synchronization is a separate authenticated integration. Local mode
does not upload the project corpus. When a configured team/cloud environment is
available, inspect state before any mutation:

```bash
hero sync graph status
```

Push and pull are explicit operations:

```bash
hero sync graph push
hero sync graph pull
```

Scope-local rows never leave the machine. Hero Cloud remains proprietary and
outside this repository's future license grant.

## Local-only remains the default

Use the repository-local `.hero` corpus, installed harness workflows, CLI, and
stdio MCP server without enabling team mode. Adopt local Serve when its
dashboard or API helps; adopt preview team/headless paths only after validating
their auth, network, provider, store, rollback, and operational boundaries
for your environment.
