---
title: "Agent Outposts — Operable External Systems with Scoped Credentials and Audit-by-Construction"
slug: agent-outposts
type: feature
status: delivering
priority: medium
horizon: next
tags: [agent-runtime, credentials, audit, knowledge-corpus]
relations:
  - target: tripwire-system
    kind: related
  - target: cloud-mcp
    kind: related
created: 2026-05-12
---

# Agent Outposts — Operable External Systems with Scoped Credentials and Audit-by-Construction

## Problem

Agents in real engineering work routinely need to act against external systems during a run — call a staging API to verify a fix, read pod logs in a k8s cluster, run a one-shot command on a remote host, query a remote database for a sample row. Today, every path to making that possible is bad in a different way:

1. **Paste credentials into chat.** Tokens, kubeconfigs, SSH keys end up in transcripts. They leak into Hero's own event log, into the harness scrollback, into shared screenshares. Rotation becomes mandatory after every demo.
2. **Pre-inject env vars.** The model can't tell which credentials are scoped to which system. `KUBECONFIG=/path/to/prod` and `KUBECONFIG=/path/to/staging` look identical from inside the model. There is no way to say "for this run, you may only act on staging." Mistakes are silent.
3. **Shell out to the user's local CLI** (`gh`, `kubectl`, `aws`). The agent inherits the user's full ambient authority — every context, every account, every cluster. Audit is whatever shell history happens to capture, which is unstructured and per-machine.

There is also a deeper problem specific to Hero. **None of these paths produce structured events.** When the agent acts on external systems via env vars or shell, the actions are invisible to the event stream — they don't compound into corpus, they don't show up in `hero recap`, they can't be traced via `hero why`. A future session asking "what did the agent touch in staging last week?" cannot be answered from Hero's data; the answer is buried in shell history if it exists at all.

The gap: a way for a user to say *"the agent can act on this specific external system, with these specific credentials, at these specific verbs, and every action lands in the event stream as a structured event."*

## Goal

Give Hero a typed, two-scope registry of operable external systems — **outposts** — with scoped credentials, opt-in per run, and audit-by-construction. Every action against an outpost becomes a structured event in Hero's stream so external work compounds into corpus exactly like internal work does.

**Mission-fit.** This raises the floor: a junior dev gets a guardrailed path to wire up agent access to staging or prod without having to invent their own secrets-management story. And it makes the next session smarter: the answer to "what did the agent touch externally last week" stops being lost in shell history and becomes part of the corpus future sessions inherit automatically.

The non-goal is to be a general secrets manager or replace 1Password / Vault / cloud secret stores. The standalone version exists so Hero is useful to people who haven't installed any of those. The pluggable backend seam exists so people who *have* installed those can plug in.

## Design

Six components. The first three are the standalone product. Components four through six are the team-and-enterprise extensions, designed-in but phased out.

### 1. Outpost Registry — Project and Global Scopes

**Storage**: encrypted file at one of two locations:

- **Project scope**: `<repo>/.hero/outposts.enc` — outposts scoped to the work in this repo
- **Global scope**: `~/.hero/outposts.enc` — outposts that follow the dev across all repos (homelab kubectl context, GitHub PAT, npm publish key, personal SSH hosts)

Each file is encrypted with a symmetric key stored in the OS keychain — macOS Keychain, libsecret on Linux, DPAPI on Windows. Project and global use distinct keychain entries so leaking one does not leak the other. If the keychain is unavailable, fall back to a passphrase prompt cached for the session.

**Resolution at run start**: merge global + project. **Project wins on name collision.** This means a repo can shadow a permissive global `prod` with a stricter project-local `prod` if needed, and the dev does not have to remember which scope a given name lives in — the project answer is always the more specific one.

**Default scope for `add`** is `project`. Dev opts into global with `--global`:

```
hero outpost add prod-api --type http ...           # project (default)
hero outpost add gh-token --type http --global ...  # global
```

**Schema** (decrypted in-memory shape, identical at both scopes):

```yaml
outposts:
  - name: prod-api
    type: http
    description: "Production REST API"
    config:
      base_url: https://api.example.com
      headers:
        X-Client: hero-agent
      auth:
        kind: bearer
        token_ref: keyring://prod-api-token
    permissions:
      - method: GET
      - method: POST
        path_prefix: /v1/feedback/
    created: 2026-05-01
    last_used: 2026-05-01T14:32:00Z
```

**Three outpost types in v1.** Each defines its own `config` schema, verb set, and permission model.

**`http`** — REST/HTTP endpoint
- Config: `base_url`, default headers, `auth: {kind: bearer | basic | api_key | none, token_ref | user_ref+pass_ref | header+key_ref}`
- Verbs: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`
- Permissions: list of allow rules, each with `method` and optional `path_prefix`. No matching rule = deny.

**`ssh`** — SSH host
- Config: `host`, `port`, `user`, `key_ref` (path to identity, or keyring entry)
- Verbs: `exec`, `read_file`, `write_file`
- Permissions: command-prefix allowlist for `exec` (e.g., `kubectl get`, `systemctl status`), path-prefix allowlist for file verbs.

**`kubectl-context`** — a named kubectl context
- Config: `kubeconfig_path` (defaults to `~/.kube/config`), `context`, optional `namespace` default
- Verbs: `get`, `describe`, `logs`, `exec`, `apply`
- Permissions: `(verb, namespace_glob, resource_kind)` triples. E.g., `(get|describe|logs, "staging-*", *)` is read-only across all staging namespaces.

**Credential refs.** Wherever a credential value is needed, the schema stores a *ref*, never a value:

| Scheme | Resolves from | Phase |
|---|---|---|
| `keyring://<name>` | OS keychain entry | 1 |
| `env://<VAR>` | Process environment at call time | 1 |
| `file://<path>` | File on disk (e.g., SSH key at `~/.ssh/id_ed25519`) | 1 |
| `op://<vault>/<item>[/<field>]` | 1Password CLI | 5 |
| `vault://<path>` | HashiCorp Vault | 5 |
| `aws-sm://<arn>`, `gcp-sm://<id>` | Cloud secret managers | 5 |

The agent never sees raw credential values. Hero resolves the ref at tool-call time and injects auth into the outbound request. Refs are opaque to the registry layer — they get handed to a backend chain at call time.

### 2. Scoped Tools Per Run

A run opts in to specific outposts via `--with`:

```
hero deliver --with prod-api,staging-cluster
hero ask --with prod-api "is /health responding?"
```

`--with` resolves names against the merged (global + project) registry. If a name appears in both scopes, the project entry wins.

For each named outpost, Hero registers a single MCP tool whose schema is **generated from the outpost type and its permissions**. Outposts not named in `--with` are not visible to the model — the tool list contains only what the user authorized for this run.

Generated tool example (`prod-api`, http, GET-only on any path plus POST on `/v1/feedback/`):

```
Tool: outpost_prod_api_call
Description: Make an HTTP request to the prod-api outpost. Auth is
             injected automatically. Allowed: GET on any path; POST
             on paths starting with /v1/feedback/.
Inputs:
  method: enum["GET", "POST"]
  path: string
  query?: object
  body?: object
```

The enum and description are *derived* from the permissions, so the model sees only the allowed verbs and is told the constraints up front. Permissions are also enforced at call time — a model that ignores its instructions and tries `DELETE` gets a structured rejection, not a leaked outbound request.

**Naming**: `outpost_<name>_<action-noun>`. The name is sanitized (kebab → underscore) so it's a valid tool identifier.

### 3. Audit-by-Construction (the mission-fit core)

Every tool call against an outpost writes a structured event to `.hero/events.log`:

```json
{
  "ts": "2026-05-01T14:32:00.123Z",
  "kind": "outpost.action",
  "run_id": "deliver-2026-05-01-...",
  "session_id": "...",
  "outpost": "prod-api",
  "outpost_scope": "project",
  "outpost_type": "http",
  "verb": "GET",
  "args_redacted": {"method": "GET", "path": "/health"},
  "result_summary": "200 OK, 142 bytes, content-type=application/json",
  "duration_ms": 47,
  "outcome": "ok"
}
```

Permission rejections produce `outpost.action.rejected` events with the rule that blocked the call. Credential-resolution failures produce `outpost.action.cred_unavailable`.

**These events feed every existing Hero surface for free**:

- `hero recap` — "during this session the agent made 7 calls against `prod-api` (project), all GETs, all 2xx; 2 calls against `gh-token` (global), both POSTs."
- `hero feed` — team-wide visibility into which outposts are in active use.
- `hero why` — when tracing a decision, outpost actions appear in the chain alongside file edits and spec updates.
- Knowledge graph — `Outpost` nodes connected by `acted_on` edges from run nodes; queries like "what runs touched prod-api this month" are graph traversals.

This is the part that earns the mission-fit claim. Without structured events, this is a credential bundle. With them, every external action becomes corpus that compounds into future sessions.

**Cross-scope audit caveat**: `outpost.action` events for *global* outposts are still written into the *current project's* `events.log` (the project the run is happening in), not a global event log. This is the right call — the corpus is project-scoped, and the event lives where the work happened — but it means "show me everywhere I used my global gh-token across all my repos this month" is not a v1 query. It can be answered later by federating event streams across projects (out of scope for this spec).

**Redaction rule**: `args_redacted` strips known credential-bearing fields (auth headers, tokens) before logging. Result bodies are summarized (status, size, content-type), never logged in full — there is no situation where an outpost *response* belongs verbatim in `events.log`.

### 4. CLI Surface

```
hero outpost add <name> --type <http|ssh|kubectl-context> [--global]
    [--base-url ... --auth-bearer-from keyring://...]
    Default scope is project; --global writes to ~/.hero/outposts.enc.
    Interactive prompts fill anything not given on the flag-line.

hero outpost list [--global-only | --project-only]
    Name, scope, type, last-used. Never shows credential values.
    Default lists both, project entries marked as shadowing on collision.

hero outpost show <name> [--scope <project|global>] [--reveal]
    Full config including cred refs. Resolved cred VALUES only with
    --reveal and an interactive confirmation. If --scope is omitted
    and the name exists in both scopes, the command lists both and
    asks the user to pick.

hero outpost rm <name> --scope <project|global>
    Scope is REQUIRED for rm — refusing to make destructive choices
    implicit. Removes the outpost from that scope and offers to
    delete any keyring:// entries it owned.

hero outpost test <name> [--scope <project|global>]
    Non-destructive probe: HEAD /health for http, ssh-keyscan for
    ssh, `kubectl version` for kubectl-context. Reports OK or a
    structured error (auth | network | permission | config).

hero outpost audit <name> [--since 7d]
    Recent outpost.action events for this outpost in the current
    project's events.log.
```

Special handling: `--with prod-api` does not require a scope — the resolution rule (project wins, then global) is unambiguous and matches what the user almost always means. Only destructive commands (`rm`) and disambiguating commands (`show` on collision) demand the scope explicitly.

### 5. Pluggable Backends (Phase 5, but designed in now)

The credential ref scheme is the seam. The Go interface is:

```go
type CredentialBackend interface {
    Scheme() string
    Resolve(ctx context.Context, ref string) (string, error)
    Available() bool   // e.g. is the op CLI installed?
}
```

Backends register at startup. v1 ships `keyring`, `env`, `file`. Adding `op://` later is a new backend file — no changes to the registry or tool layer.

An outpost referencing a backend that isn't installed locally fails clearly at first *use*, not at registration. This is intentional: it means a teammate can publish an outpost definition referencing `op://Eng/staging-token` and a colleague without the 1Password CLI gets a useful error ("the `op` backend is not available; install 1Password CLI or change the cred ref") rather than a silent fallback to no auth.

### 6. Team Distribution (Phase 6)

`.hero/outposts.public.yaml` — an optional, *gittable-but-gitignored-by-default* file at the project scope only (global scope is by definition personal and has no team-distribution analog) containing outpost definitions **without credential values** (refs only):

```yaml
outposts:
  - name: prod-api
    type: http
    config:
      base_url: https://api.example.com
      auth: {kind: bearer, token_ref: op://Eng/prod-api-token}
    permissions:
      - {method: GET}
```

`hero outpost import` reads a teammate's `outposts.public.yaml`, prompts the local user to set up cred storage for each ref (or accepts an existing local entry), and writes the result to the local encrypted project registry.

This means a team can converge on "everyone can talk to staging-cluster" without any credential ever leaving an individual machine. The shape is shared; the secrets are not.

### Design decisions

**Why "outpost" and not "target", "destination", or "endpoint"?** "Target" sounds combative and is too narrow (you fire at targets; you don't audit verbs against them). "Destination" is wordy and carries email/networking baggage. "Endpoint" is HTTP-specific. "Outpost" captures (a) it's external/remote, (b) it's an established place with known protocols, (c) there's a trust relationship to home base, (d) you have multiple, each its own mini-territory with its own authority. CLI ergonomics: `hero outpost list`, `--with prod-api` reads as "the outposts this run can reach."

**Why two scopes (project + global) instead of project-only?** Personal outposts (homelab kubectl context, GitHub PAT, npm publish key) are not project-specific. Forcing N copies across N repos turns rotation into a chore. Global solves that without giving up the per-project intent of project-scoped outposts.

**Why "project wins on name collision"?** The project answer is the more specific one. A repo author who registers a `prod` outpost is making an intentional choice to override whatever the dev's global `prod` is. The opposite rule (global wins) would mean a careless personal config silently overrides a project's safety boundary.

**Why does `add` default to project, not global?** Most outposts are project-scoped in practice. Defaulting to global would put personal credentials into team-shared definitions when someone copies a flag-line from a teammate. The default should be the safer-and-more-common choice.

**Why does `rm` require an explicit `--scope` while `--with` does not?** `--with` is read-only and the resolution rule is unambiguous. `rm` is destructive — silently picking the project entry when the user meant global (or vice versa) is the kind of mistake that ends with the wrong key getting deleted. Make it impossible to do by accident.

**Why per-project event logs even for global outposts?** Events live where the work happened. A global gh-token used in repo A vs. repo B genuinely produced different work — separating those events by project is correct. The cost is that "show me everywhere I used gh-token across all my repos" is not a single query in v1; that comes later via cross-project event federation, which is a Hero-wide concern not unique to outposts.

**Why opt-in `--with` rather than "all outposts always available"?** Principle of least authority. If you have 30 outposts registered, the model should not see 30 tools every run; it should see exactly the ones the user named for this work. This also keeps the prompt budget small and the model's choices unambiguous.

**Why not just shell out to `gh`, `kubectl`, `aws`?** Three reasons that compound: (a) **scope** — a shellout inherits the user's full ambient authority, with no way to say "only staging." (b) **audit** — shell history is unstructured and per-machine; events.log is structured and team-syncable. (c) **leakage** — env vars and config files end up in tool transcripts.

**Why an encrypted local file plus a keychain key, instead of putting everything in the keychain?** Keychains store secrets, not arbitrary structured data. The non-secret half (outpost name, type, base_url, permission rules) belongs in a file we can version-pattern. The keychain holds the encryption key plus any `keyring://` cred values. This split is also what makes the team-distribution story work — definitions can be shared in a file; secrets cannot.

**Why audit events in `events.log`, not a separate audit log?** Hero's value is the corpus that compounds. A separate audit log doesn't compound — it's a parallel artifact that needs its own viewers and its own sync story. Events in the existing log flow through `recap`, `feed`, `why`, and the graph for free, which is what makes external action behave like internal action.

**Why three types in v1 (http, ssh, kubectl-context)?** Coverage check: REST API → http; remote machine → ssh; k8s cluster → kubectl-context. These cover the high-frequency cases. Postgres, Redis, S3, cloud SDKs come later as native types or via plugins. The cost of supporting one fewer type now is small; the cost of getting the registry/permissions/audit primitives wrong with five types is large.

**Why `kubectl-context` as a separate type and not "ssh with extra config"?** The verb model differs (`get|describe|logs|exec|apply` vs. arbitrary commands), the permission model differs (namespace + resource kind vs. command-prefix), and the auth model differs (kubeconfig context vs. SSH key). Modeling it as ssh-with-special-config would force users to write k8s permission rules in command-prefix syntax, which doesn't fit the operable unit.

**Why "scoped permissions" at all instead of "the agent can do whatever the credential allows"?** Tokens are blunt. A staging API token usually has more capability than a given run needs. Permissions narrow the agent's authority to the slice required for the work, so a prompt-injection or hallucination cannot escalate to a full-token-can-do. This is also what makes the model's tool list legible — it sees only allowed verbs.

**Why fail at call time, not registration time, when a backend is missing?** Registration must succeed for definitions to be portable across machines. A teammate without 1Password installed should still be able to *see* an outpost definition referencing `op://`; they just cannot *use* it without setting up the backend. Failing at call time gives a useful, actionable error in the right place.

## Acceptance Criteria

- WHEN a user runs `hero outpost add <name> --type <type>` THE SYSTEM SHALL prompt for type-specific config and credential refs, validate the resulting outpost, and persist it to the project registry at `<repo>/.hero/outposts.enc`, encrypted with a key from the OS keychain.
- WHEN a user runs `hero outpost add <name> --global ...` THE SYSTEM SHALL persist the outpost to the global registry at `~/.hero/outposts.enc` instead of the project registry.
- WHEN `hero outpost list` is run THE SYSTEM SHALL display name, scope, type, description, and last-used timestamp for every registered outpost across both scopes, marking project entries that shadow a global entry of the same name, and SHALL NEVER display credential values or resolved auth material.
- WHEN `hero outpost show <name>` is run and the name exists in only one scope THE SYSTEM SHALL display full config including cred refs from that scope; values shown only with `--reveal` and interactive confirmation.
- WHEN `hero outpost show <name>` is run and the name exists in both scopes without `--scope` THE SYSTEM SHALL list both entries and prompt the user to pick rather than guessing.
- WHEN `hero outpost rm <name>` is run THE SYSTEM SHALL require `--scope project` or `--scope global` and SHALL fail with a clear error if scope is omitted, regardless of how many scopes contain the name.
- WHEN `hero outpost test <name>` is run THE SYSTEM SHALL perform a non-destructive probe appropriate to the outpost type and report `ok` or a structured error categorized as `auth | network | permission | config`.
- WHEN a Hero run is invoked with `--with <names>` THE SYSTEM SHALL resolve each name against the merged registry with project-wins precedence, register one MCP tool per resolved outpost with input schema derived from the outpost's type and permissions, and SHALL NOT register tools for outposts not named.
- IF a name passed to `--with` exists at neither scope THEN THE SYSTEM SHALL fail run startup with a clear error listing available outpost names.
- WHEN an agent invokes an outpost tool THE SYSTEM SHALL resolve all credential refs through the backend chain, inject auth into the outbound action, execute, and return the result.
- WHEN any outpost tool call completes (success or failure) THE SYSTEM SHALL write an `outpost.action` event to the current project's `.hero/events.log` containing outpost name, scope, type, verb, redacted args, result summary, duration, and outcome — including for outposts that resolved from global scope.
- IF an outpost tool is invoked with arguments that violate the outpost's permission rules THEN THE SYSTEM SHALL reject the call before any outbound action and emit an `outpost.action.rejected` event naming the rule that blocked it.
- IF a credential ref cannot be resolved (backend unavailable, entry missing) THEN THE SYSTEM SHALL fail the tool call with a structured error naming the backend and ref, emit an `outpost.action.cred_unavailable` event, and SHALL NOT silently fall back to unauthenticated calls.
- THE SYSTEM SHALL never write resolved credential values to event logs, transcripts, recaps, knowledge files, or graph nodes.
- THE SYSTEM SHALL ingest outpost definitions as `Outpost` nodes in the knowledge graph (with a `scope` property), and SHALL create `acted_on` edges from run nodes to outpost nodes whenever an `outpost.action` event occurs.
- WHEN `hero recap` runs over a session THE SYSTEM SHALL include a section summarizing outpost actions, grouped by outpost (with scope shown), with verb counts and outcome counts.
- WHERE a pluggable credential backend is configured THE SYSTEM SHALL resolve refs through that backend at call time and SHALL surface backend unavailability as the actionable error described above.
- THE SYSTEM SHALL provide CLI commands `hero outpost add` (with `--global`), `list`, `show`, `rm` (with required `--scope`), `test`, and `audit`.

## Changes

### New files
- `internal/outposts/registry.go` — load, save, encrypt, decrypt registry files at both scopes; merge with project-wins precedence
- `internal/outposts/types.go` — `Outpost` struct, `Scope` enum (project, global), type-specific config schemas, permission rule shapes
- `internal/outposts/keychain/keychain_darwin.go` — macOS Keychain integration
- `internal/outposts/keychain/keychain_linux.go` — libsecret integration
- `internal/outposts/keychain/keychain_windows.go` — DPAPI integration
- `internal/outposts/credbackend/backend.go` — `CredentialBackend` interface + chain registry
- `internal/outposts/credbackend/keyring.go` — `keyring://` backend
- `internal/outposts/credbackend/env.go` — `env://` backend
- `internal/outposts/credbackend/file.go` — `file://` backend
- `internal/outposts/types/http.go` — http verb handlers + permission check + tool schema generator
- `internal/outposts/types/ssh.go` — ssh verb handlers + permission check + tool schema generator
- `internal/outposts/types/kubectl.go` — kubectl-context verb handlers + permission check + tool schema generator
- `internal/cli/outpost.go` — `hero outpost` CLI subcommand tree
- `<repo>/.hero/outposts.enc` — encrypted project outpost registry (gitignored)
- `~/.hero/outposts.enc` — encrypted global outpost registry (per-user)

### Modified files
- `internal/serve/mcp.go` — accept a list of outpost names at session start; resolve via merged registry; register one MCP tool per named outpost via the per-type schema generator; emit `outpost.action*` events on tool invocation
- `internal/serve/run.go` (or run-bootstrap equivalent) — accept `--with` flag and propagate outpost list to MCP registration
- `internal/cli/deliver.go`, `internal/cli/ask.go`, `internal/cli/diagnose.go`, `internal/cli/design.go` — add `--with` flag; validate names against the merged registry before starting the run
- `internal/events/log.go` — register `outpost.action`, `outpost.action.rejected`, `outpost.action.cred_unavailable` event kinds and their schemas
- `internal/graph/schema.go` — add `Outpost` node type (with `scope` property) and `acted_on` edge kind
- `internal/graph/ingest.go` — ingest both registries as `Outpost` nodes; ingest `outpost.action` events as edges from run nodes to outpost nodes
- `internal/recap/recap.go` — surface an Outpost Actions section with grouped summaries (scope shown)
- `internal/feed/feed.go` — include recent outpost activity in feed output
- `.gitignore` — add `.hero/outposts.enc`; add a commented-out template `.hero/outposts.public.yaml` line for Phase 6

## Phasing

### Phase 1 — Registry + http outpost end-to-end (project scope only)
Encrypted project-scope registry, OS keychain integration on macOS (Linux + Windows follow), `http` outpost type only, `keyring://` / `env://` / `file://` cred backends. CLI: `add`, `list`, `show`, `rm`, `test` (no scope flags yet — all project). Scoped MCP tool registration via `--with`. `outpost.action*` events written to `events.log`. This is the minimum slice that proves the model end-to-end and is shippable on its own.

### Phase 2 — Global scope + scope-aware CLI
Add the global registry, merge resolution with project-wins precedence, `--global` on `add`, `--global-only` / `--project-only` on `list`, required `--scope` on `rm`, `--scope` on `show` for collisions. Updated event payload includes `outpost_scope`.

### Phase 3 — ssh and kubectl-context outpost types
Adds the other two v1 types. Most of the registry/CLI/events/keychain plumbing is reused; this phase is mainly the per-type verb handlers, permission models, and tool-schema generators.

### Phase 4 — Graph + recap + feed surfacing
`Outpost` nodes (with scope property) and `acted_on` edges in the graph. Recap and feed integration. This is what earns the mission-fit claim end-to-end — external actions become first-class corpus visible in the same surfaces as everything else Hero captures.

### Phase 5 — Pluggable credential backends
1Password (`op://`), HashiCorp Vault (`vault://`), AWS Secrets Manager (`aws-sm://`), GCP Secret Manager (`gcp-sm://`). Each is a new file behind the existing backend interface; no core changes.

### Phase 6 — Team distribution
`.hero/outposts.public.yaml` (project scope only) for sharing outpost definitions (no secrets) across a team. `hero outpost import` to consume a teammate's definitions and prompt for local credential setup. This makes the on-ramp for new teammates "import the team file, set up your creds, you're operational" instead of "ask someone for the staging token in Slack."
