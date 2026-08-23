# Hero

Hero gives AI coding tools durable project memory and a verified delivery
system, so decisions survive sessions and agents finish work against evidence.

Project memory is the headline: Hero keeps intent, decisions, corrections,
conventions, evidence, failures, and current state in a project-owned `.hero/`
corpus. Verified delivery is the connected execution system: specs bound the
work, specialized agents implement it, and Completion Ledgers, cold audits,
builds, and tests establish whether it is done.

Memory informs delivery. Delivery can add decisions and evidence for later
sessions. The components are shipped; the repeatable cross-tool continuity
outcome remains preview until its public proof is complete.

**New here? Follow [Getting Started](GETTING-STARTED.md).**

## Install

Prebuilt binaries do not require Go.

macOS or Linux with Homebrew:

```bash
brew install hero-engine/tap/hero
```

Linux install script:

```bash
curl -fsSL https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.sh | sh
```

Windows with Scoop:

```powershell
scoop bucket add hero-engine https://github.com/hero-engine/scoop-bucket
scoop install hero
```

Windows PowerShell install script:

```powershell
irm https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.ps1 | iex
```

The [installation guide](web/docs/src/getting-started/installation.md) covers
direct downloads and source builds.

## First project

Run these commands from the repository root:

```bash
hero init
hero install project . --target codex
hero status
```

Replace `codex` with `opencode`, `cursor`, `claude`, `copilot`, `generic`, or
`grok` for another supported target. Hero renders workflows into each target's
native surfaces. Claude receives command files; Codex and Grok receive
`command-*` workflow skills; other targets receive their supported native
surfaces and root instructions. Do not assume slash commands exist everywhere.

The default setup combines Core with Engineering, including lightweight PM and
QA assistance used inside engineering workflows. Focused PM, QA, and Sales
setups are optional and maturity-bounded:

```bash
hero domain list
hero domain enable qa
hero install project . --target codex
```

## Project memory

Start or resume work by asking your installed harness to resume the project, or
use the CLI directly:

```bash
hero resume
hero status
hero search "OAuth session handling"
hero ask "What is our retry policy?"
hero relevant src/auth/session.go src/auth/middleware.go
hero why session-retry
hero blocked
```

Capture durable context deliberately. Notes, conventions, decisions, and
handoff projections are writes to the repository-owned corpus; search and
retrieval are reads.

```bash
hero note session-retry
hero next
hero recap --since 2d
```

Inside an installed harness, natural-language requests route to its native Hero
workflow. For example: “resume this project,” “design CSV export,” or “diagnose
the login timeout.”

## Verified delivery

The normal engineering loop is:

1. Design a feature or diagnose a bug into an approved spec.
2. Deliver against its acceptance criteria and project conventions.
3. Record the Completion Ledger and run the project build/tests.
4. Run a fresh, independent cold delivery audit.
5. Close once with `hero spec verify <slug>`.

```bash
hero spec score csv-export
hero spec deliver csv-export --manual
hero spec verify csv-export
```

`hero spec verify` checks the Completion Ledger, cold audit, acceptance-criteria
coverage, and configured build/test command. When the hard gates pass, it marks
the spec complete and archives it. `hero spec complete` exists as a manual
administrative path; it is not the normal evidence-backed delivery close.

Use the active harness workflow for agent execution. The CLI is useful for
inspection, automation, scoring, and the final verification gate:

```bash
hero list --ready --sort priority
hero queue --format kickoff
hero diff .hero/planning/features/csv-export/spec.md
hero drift csv-export
hero ac list csv-export
hero coverage csv-export
```

`hero drift` and `hero coverage` exit non-zero when they find drift or missing
coverage; that is a failed gate to address, not a broken invocation.

## Monorepos: one corpus, thin satellites

Initialize Hero once at the repository root. When a harness opens inside a
subproject, materialize a thin harness-native satellite tree that points back
to the root content:

```bash
hero install satellites                    # guided candidate review
hero install satellites --yes              # accept detected candidates
hero install satellites --repair           # reconcile satellite trees
hero install satellites --migrate-nested   # print a legacy migration plan
```

A monorepo has one root `.hero` corpus. Satellites are not nested Hero
workspaces. Do not run `hero init` inside a subproject under an existing Hero
root.

Repair the intended install scope explicitly:

```bash
hero install project . --target codex --repair
hero install satellites --repair
hero check
hero doctor
```

`hero doctor` diagnoses binary, schema, and installed-target mismatches.
`hero check` reports workspace and corpus health.

## Optional integrations and runtime surfaces

| Capability | Availability | Prerequisite and boundary |
|---|---|---|
| MCP project-memory tools | Shipped | An initialized workspace and harness MCP configuration. The runtime `tools/list` response is the inventory authority after filtering. |
| Local `hero serve` dashboard/API | Shipped locally | Register local projects deliberately. Team or external access changes the trust boundary. |
| Tracker operations | Optional | Configure GitHub, Jira, Linear, or GitLab credentials. Mutations require explicit consent for the exact issue and action. |
| Code-host operations | Optional | Configure a supported code-host connection. Reads do not imply permission for writes. |
| Cross-repository peering | Optional | Register reachable sibling repositories and manifests. Requests use asynchronous Project Mail. |
| Headless agent runtime | Preview | Configure a model provider, credentials, and execution environment. Do not treat preview jobs as unsupervised or production-ready. |

See [MCP Setup](MCP-SETUP.md), [Cross-Repo Peering](CROSS-REPO-PEERING.md),
and [Team Server](TEAM-SERVER.md).

## Cross-repository work

Each Hero project keeps its own graph. Registered peers exchange asynchronous
Project Mail; sending never launches a model or writes the receiver's tree.

```bash
hero admin repos add app ../app
hero peer manifest
hero peer list
hero peer call app --mode=advisory --reason="Confirm the API contract" \
  "Is the error envelope stable?"
hero handoff order-failure app --reason="Implementation belongs to app"
```

The receiver must inspect and explicitly promote a work-transfer message:

```bash
hero mail show <message-id>
hero handoff receive <message-id> --type bug
```

## Workspace layout

```text
.hero/
├── mission.md                  # project charter and first principles
├── NEXT.md                     # projected session handoff
├── QUEUE.md                    # ready-work projection
├── SNAPSHOT.md                 # project-shape rollup
├── planning/                   # active specs
├── specs/                      # verified, archived specs
├── knowledge/                  # decisions, conventions, context, notes
├── next/                       # personal/local handoff projections
├── smoke/                      # feature smoke metadata
├── graph.db                    # generated graph store
├── index.db                    # generated search index
└── hero.json                   # committed project configuration
```

Specs, knowledge, and projected handoff files are committed. Generated
databases, credentials, sessions, and local overlays remain ignored.

## This repository

```text
cmd/hero/                 CLI entrypoint
contracts/                integration and operation contracts
internal/                 Go implementation
core/                     shared agents, workflows, skills, and vocabularies
domains/engineering/      default Engineering content
domains/pm/               optional focused PM content
domains/qa/               optional focused QA content
domains/sales/            optional focused Sales content
web/                      hosted docs and landing source
```

Hero Cloud is a separate proprietary product; there is no `cloud/` backend tree
in this repository. Hero Code is also a separate proprietary product. Sprout
(`github.com/bdwheeler/sprout`) is a separate public MIT-licensed dependency.

This `hero` repository is being prepared for a future Apache-2.0 grant, but the
grant has not happened: there is no root license file yet. Do not describe this
repository, Hero Code, or Hero Cloud as open source until the explicit license
and visibility gates land. Third-party components retain their own licenses.

## Build from source

Building from source requires the Go version declared by `go.mod`, currently
Go 1.26.4:

```bash
make build
make test
go build ./...
go test ./...
```

Prebuilt binary installation does not require Go.

## More documentation

- [Getting Started](GETTING-STARTED.md)
- [MCP Setup](MCP-SETUP.md)
- [Cross-Repo Peering](CROSS-REPO-PEERING.md)
- [Team Server and Headless Runtime](TEAM-SERVER.md)
- [Hosted documentation source](web/docs/src/index.md)
- [Configuration reference](web/docs/src/configuration/hero-json.md)
- [Capability status](web/docs/src/reference/capability-status.md)

## Project policy

- [Contributing](CONTRIBUTING.md)
- [Support](SUPPORT.md)
- [Security](SECURITY.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)

This repository remains private and does not yet carry a root open-source
license. The policy files prepare the public contribution and reporting routes;
they do not activate those routes or grant redistribution rights.
