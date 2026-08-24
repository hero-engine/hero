# Why Hero

Long-lived codebases accumulate more truth than one prompt or chat can carry:
decisions, exceptions, corrections, failed approaches, conventions, active
work, and evidence. Hero makes that project truth durable and connects it to a
completion workflow.

## When Hero is useful

Hero fits AI-native engineers and hands-on technical leads who switch sessions
or tools while remaining accountable for correctness. It is especially useful
when a project has non-obvious history, several concurrent work items, or a
completion standard stronger than “the agent says it is done.”

## What it adds to existing tools

| Existing tool | Keep using it for | Hero adds |
|---|---|---|
| Harness rule files | Stable instructions the tool should always load | Evolving decisions, evidence, current work, scoped retrieval, and handoff state |
| Chat history | Continuity within one thread | Project-owned artifacts that survive thread and tool boundaries |
| Wiki | Authoritative human documentation | Agent-relevant retrieval and execution context; Hero should link to the wiki, not replace it |
| Issue tracker | Commitments and coordination | Local evidence, specs, and bounded optional sync operations |
| AI coding harness | Reasoning and code changes | Durable context plus an evidence-gated delivery workflow |

## The completion bar

Hero's specs are a delivery mechanism, not the product category. A normal
delivery ends with one command:

```bash
hero spec verify <slug>
```

That command checks the Completion Ledger, cold audit, acceptance-criterion
test mapping, and build/test result. If the hard gates pass, it marks the spec
complete and archives it. `hero spec complete` is not the normal delivery
close.

## Honest limits

- Workflows are harness-native, not universal slash commands.
- External spec-system adapters are planned, not shipped.
- Tracker, code-host, peering, and headless operations are optional or preview
  and require setup, credentials, and explicit action boundaries.
- Hero Code and Hero Cloud are separate proprietary products.

## Evaluate it from evidence

Start with [Capability status and evidence](reference/capability-status.md),
then follow [Project memory](concepts/knowledge-base.md) or
[Verified delivery](concepts/core-loop.md). Mutable command, agent, skill, and
MCP-tool counts are intentionally absent from this narrative; inspect the
installed target with `hero doctor` and the runtime registry with MCP
`tools/list` when exact inventory matters.
