# Hero

**Your project remembers. Your agents deliver.**

Hero gives AI coding tools durable project memory and a verified delivery
system, so decisions survive sessions and agents finish work against evidence.

## Two connected paths

### 1. Project memory

Hero keeps project-owned intent, decisions, corrections, conventions, evidence,
failures, and current state in a local `.hero/` corpus. Supported harnesses and
the CLI can retrieve the relevant slice instead of rebuilding context in every
prompt.

[Follow the project-memory path](concepts/knowledge-base.md)

### 2. Verified delivery

Specs bound intent and acceptance. Specialized agents implement the work. A
Completion Ledger, a fresh cold audit, and build/test gates establish whether
the result can be closed.

[Follow the delivery-system path](concepts/core-loop.md)

The reinforcing cross-session, cross-tool loop is currently **preview**: the
components ship, but the repeatable continuity proof is not yet public. Hero
preserves the artifacts needed for that loop; it does not claim that every tool
or session applies them perfectly.

## Capability status at a glance

| Capability | Availability | Prerequisite | Action boundary |
|---|---|---|---|
| Project memory and retrieval | Shipped | Hero workspace plus supported harness or CLI | Reads project-owned corpus; writes happen only through explicit capture workflows |
| Spec-and-agent delivery | Shipped | Engineering setup and an active harness | Implementation follows an approved spec; verification is evidence-gated |
| Attention, Mail, and Focus | Shipped | User-global Hero state | Attention is bounded; Mail bodies require explicit reads; Focus is private |
| `hero serve` project intelligence | Shipped | Local daemon startup | Local by default; team/external access needs separate auth and configuration |
| Tracker and code-host operations | Optional | Configured provider, credentials, repository identity | Mutations require explicit operation-specific consent |
| Cross-repository peering | Optional | Registered reachable siblings and peer manifests | One graph per project; calls and handoffs cross an explicit Mail boundary |
| Headless runtime | Preview | Model provider, credentials, execution environment | Approval-gated jobs pause before protected actions |

See [Capability status and evidence](reference/capability-status.md) for the
implementation authorities behind these labels.

## Quick start

<!-- hero-quickstart -->
```bash
brew install hero-engine/tap/hero
cd your-project
hero init
hero install project . --target codex
hero check
```

Prebuilt binaries do not require Go. A source build requires the Go version in
`go.mod`. Hero renders workflows into each supported target's native surfaces;
they are not universal slash commands.

## Current release and documentation freshness

Release notes are generated from published releases. The docs artifact also
publishes its exact source revision at [Build information](about/build.md) and
[`/revision.json`](revision.json). The current release is derived from the
latest release tag during the build; it is not maintained as narrative copy.

## Product and repository boundary

This site documents the `hero` CLI repository. Hero Code and Hero Cloud are
separate proprietary products. Sprout is a separate MIT-licensed project and
is not covered by Hero's future license grant. Apache-2.0 preparation for this
repository is authorized, but Hero must not be described as open source until
the explicit license gate adds the root license.

## Next steps

- [Installation](getting-started/installation.md)
- [Project memory](concepts/knowledge-base.md)
- [Verified delivery](concepts/core-loop.md)
- [Attention, Mail, and Focus](cli/attention.md)
- [`hero serve`](cli/server-and-mcp.md)
- [Build information](about/build.md)
