# Capability status and evidence

Availability labels on this page are part of the public contract:

- **Shipped** — executable implementation or runtime evidence exists.
- **Optional** — shipped behind explicit setup, credentials, or composition.
- **Preview** — implementation exists, but the support or public-proof boundary
  is not yet release-complete.
- **Planned** — do not describe as available.

## Project memory

- **Availability:** Shipped
- **Prerequisite:** A Hero workspace plus a supported harness or CLI entry point.
- **Evidence:** `internal/knowledge/`, `internal/graph/`, `internal/cli/next_handoff.go`,
`hero search --help`, and `hero next --help`.
- **Action boundary:** Search and retrieval are reads. Capturing a note, decision,
correction, convention, or handoff is an explicit write to the project corpus.

Memory includes intent, decisions, corrections, conventions, evidence,
failures, and current state. See [Project memory](../concepts/knowledge-base.md)
and [Continuity](../concepts/continuity.md).

## Verified delivery

- **Availability:** Shipped
- **Prerequisite:** Engineering setup, an active harness, a delivery workflow, and
a testable project.
- **Evidence:** `.agents/skills/command-deliver/SKILL.md`,
`.agents/skills/completion-ledger/SKILL.md`,
`.agents/skills/delivery-audit/SKILL.md`, `internal/cli/verify.go`, and
`hero spec verify --help`.
- **Action boundary:** Work starts from an approved spec. A fresh auditor must be
independent of the implementing agent. Verification mutates status and archives
only after the hard gates pass.

See [Verified delivery](../concepts/core-loop.md).

## Attention, Project Mail, and Focus

- **Availability:** Shipped
- **Prerequisite:** User-global Hero state; peer identity is required to send
Mail across configured local projects.
- **Evidence:** `contracts/attention/`, `internal/cli/attention.go`,
`internal/cli/mail.go`, and `internal/cli/focus.go`.
- **Action boundary:** Attention snapshots are metadata-bounded. Mail bodies are
untrusted data and require an explicit read. Focus is private user-global state.
Row actions must use the advertised action ID and revision; mutations require a
fully resolved recipient, content, and destination and are not replayed merely
to confirm success.

See [Attention, Mail, and Focus](../cli/attention.md).

## `hero serve` project intelligence

- **Availability:** Shipped locally
- **Prerequisite:** Start the local daemon and register project paths. Team mode
or external access requires separate auth and network configuration.
- **Evidence:** `internal/cli/serve.go`, `internal/serve/`, and
`hero serve --help`.
- **Action boundary:** The default binds local project intelligence. Enabling
team mode, workers, or an auth token changes the operating boundary and should
be deliberate.

See [Server and MCP](../cli/server-and-mcp.md).

## Tracker evidence and mutations

- **Availability:** Optional
- **Prerequisite:** A configured GitHub, Jira, Linear, or GitLab delivery
connection with provider credentials.
- **Evidence:** `contracts/trackerbroker/`, `internal/tracker/`, and
`hero tracker contract`.
- **Action boundary:** Reads may collect tracker evidence. Mutations require
explicit semantic consent for the exact issue and operation. Credentials belong
in environment variables or `.hero/hero.local.json`, never committed config or
command arguments.

See [Tracker setup](../configuration/tracker-setup.md).

## Code-host operations

- **Availability:** Optional
- **Prerequisite:** A supported code-host connection, credentials, repository
identity, and operation-specific consent.
- **Evidence:** `contracts/codehostbroker/contract.go`,
`internal/serve/mcp_tools_code_host.go`, and `hero code-host contract`.
- **Action boundary:** Brokered reads and writes remain provider-neutral, but
every mutation is bound to the requested repository and operation. Approval is
not inferred from a previous read or a broad connection grant.

See [Code-host operations](../cli/code-host.md).

## Cross-repository peering

- **Availability:** Optional
- **Prerequisite:** Registered sibling repositories with reachable paths and
committed peer manifests.
- **Evidence:** `internal/peering/`, `internal/attention/mail/`,
`hero peer --help`, and `hero handoff --help`.
- **Action boundary:** One graph remains authoritative per project. Peer calls are
asynchronous Project Mail requests. Work transfer does not mutate the receiver
until it explicitly promotes the message through Intake.

See [Cross-repository peering](../concepts/cross-repo.md).

## Headless agent runtime

- **Availability:** Preview
- **Prerequisite:** A configured model provider, credentials, and a supported
execution environment.
- **Evidence:** `internal/runner/` and `hero agent --help`.
- **Action boundary:** Approval-aware jobs pause at configured gates. Preview
status means support boundaries still need release-level validation; do not
describe the runtime as unsupervised or production-ready.

## Domain composition

**Engineering default:** Shipped. A default project receives Core plus the
Engineering setup, including lightweight PM and QA assistance used inside
coding workflows.
**Focused PM, QA, and Sales setups:** Optional. Select the primary domain or
supported extension composition, then re-run installation. Availability of a
focused pack is not proof of a broad departmental platform.

Run `hero doctor` for the exact installed target inventory at the current
revision. Run MCP `tools/list` for the exact runtime tool registry after any
configured filtering.
