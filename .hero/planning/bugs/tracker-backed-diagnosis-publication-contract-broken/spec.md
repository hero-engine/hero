---
title: "Hero Code blocks the existing tracker-backed diagnosis publication commands"
slug: tracker-backed-diagnosis-publication-contract-broken
type: bug
status: planning
domain: engineering
size: medium
priority: critical
severity: high
root_cause_class: design
created: 2026-07-19
parent: hero-in-hero-code-parity
tags: [diagnose, tracker, jira, hero-code, workflow-parity, integrations, security]
---

# Hero Code blocks the existing tracker-backed diagnosis publication commands

## Kickoff

Restore tracker-backed `/diagnose` completion by letting Hero Code follow the
canonical diagnose workflow and run Hero's existing `sync attach` and
`sync comment` commands through its app-owned command runner.

**Status:** planning — the root cause remains confirmed; the fix direction was
simplified after engineer challenge. No live tracker was queried or mutated.

**Pick up at:** expose `comment` and `attach` as narrow operations on Hero
Code's existing tracker command runner, preserve normal process/external-
mutation approval, and make every diagnose entry point load or repeat the
canonical workflow's postback step.

→ `.hero/planning/bugs/tracker-backed-diagnosis-publication-contract-broken/spec.md`

**Files:** `internal/cli/diagnose.go:123`, `internal/serve/mcp_tools.go:2288`, `domains/engineering/agents/debug-investigator.md:129`, sibling `packages/hero-swift/Sources/HeroSharedApplication/Engine/ToolExecutor.swift:576`, sibling `packages/hero-swift/Sources/HeroSharedApplication/Engine/AgentLoop.swift:3167`
**Skip:** do not add MCP, a new publication subsystem, or a new Hero CLI wrapper unless implementation proves the existing two commands cannot satisfy the workflow.

**Paste-ready implementation prompt:**

> Restore tracker-backed diagnosis postback with the smallest coherent change.
> In Hero Code, extend the existing structured `hero_tracker` runner and schema
> with validated `comment` and `attach` operations that invoke Hero's existing
> `sync comment` and `sync attach` commands through the app-selected
> `HeroProcessRunner`, `HeroTrackerProcessEnvironment`, and redaction path.
> Keep generic `hero_cli sync` and Bash Hero execution blocked. Preserve
> `hero_cli`/`hero_tracker` as `.process` tools; do not make them unconditional
> first-party silent allows or bypass plan/process approval. In Hero, update the
> short CLI/MCP diagnose instructions so natural-language diagnosis receives
> the same tracker attachment/comment close as the canonical
> debug-investigator workflow. Report the two command outcomes truthfully. Do
> not add MCP, a combined publish command, adapter logic in Swift, an
> idempotency subsystem, or stable-integration migration. Verify focused Swift
> executor/permission tests, Hero diagnose-output tests, and installed-content
> parity.

## Summary

### Categorization

| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — a tracker-backed diagnosis can claim success while silently omitting the external artifact that makes the workflow useful to the rest of the team. |
| **Ease of Fix** | moderate — Hero already owns both tracker writes and Hero Code already owns the compatible runner, credentials, redaction, and process permissions; the missing work is narrow command exposure plus workflow parity. |
| **Caused by our codebase?** | Yes — Hero's instructions promise publication while other Hero surfaces omit it, and Hero Code makes the promised operation unreachable. |
| **Needs more research?** | No — the current contracts, history, local spec, and Hero Code session establish the failure before any Jira request could be made. |

### Background

The user's memory is substantially correct. Since the first retained Hero
commit (`982742d`, 2026-05-12), the debug-investigator has ended tracker-backed
diagnoses by attaching the report and posting a root-cause summary. The Go CLI
has supported `hero sync attach` and `hero sync comment` for the same period.
This behavior was never a background engine hook: it was an explicit final
agent step. In a harness where the agent could run those commands, it looked
automatic from the user's perspective and worked as remembered.

The current Hero Code result is different. A Jira bug was imported into the
Morpheus workspace, retaining `tracker_id: MORPH-14509`, and was diagnosed
locally. The active Hero Code transcript then claimed that no tracker-posting
step existed and that publication must be manual. That claim contradicts both
the embedded `/diagnose` command and embedded debug-investigator content.

### Analysis

Three surfaces have drifted apart:

1. Hero's canonical `/diagnose` command and debug-investigator require tracker
   publication.
2. Hero's CLI/MCP `diagnose` preparation output lists only local spec-writing
   steps, making it an attractive but incomplete shortcut around the command.
3. Hero Code blocks every `sync` call through generic `hero_cli`, blocks direct
   Hero execution through Bash, and exposes only `operation: import` through
   the protected `hero_tracker` tool.

Therefore even perfect model compliance after loading the canonical workflow
would fail in the shipped Hero Code build. Conversely, exposing the commands
without making natural-language and shortcut diagnosis load the canonical
workflow would leave a route that still stops after the local spec.

### Root Cause

**Classification: `design`.** Hero Code's credential hardening correctly moved
tracker execution behind an app-owned structured runner, but the blanket
`sync` denial plus import-only `hero_tracker` allowlist failed to carry forward
the two existing commands required by the canonical diagnose workflow. Hero's
short CLI/MCP diagnose instructions compound the regression by omitting the
same closing step. The failure is command-surface drift, not a missing tracker
engine, adapter, MCP service, or publication subsystem.

### Source

- Hero workflow content: `command-diagnose` and `debug-investigator`.
- Hero shortcut surfaces: CLI `hero diagnose` and MCP `hero_diagnose`.
- Hero tracker primitives: `sync comment`, `sync attach`, and Jira's REST
  adapter.
- Hero Code routing and capability boundary: explicit-slash expansion,
  `skill_run`, `hero_cli`, `hero_tracker`, and process permissions.

### Fix Direction

Use the commands Hero already has. Hero Code should extend its existing
structured `hero_tracker` runner with narrow `comment` and `attach` operations
that map to `hero sync comment` and `hero sync attach`. That runner already
selects the app-compatible Hero binary, executes in the workspace, supplies the
tracker child environment, and redacts output. Validate the issue ID, comment,
and workspace-contained attachment path before invoking it.

Keep `hero_tracker` under the existing process/external-mutation permission
path. Do not silently mark `hero_cli` or `hero_tracker` first-party trusted
ahead of plan-mode and process checks. Hero Code must load/follow the same
canonical diagnose workflow as Claude Code; Hero's shorter CLI/MCP preparation
output should explicitly include or direct callers to that canonical postback
step.

Do not require MCP. Do not add `hero diagnose publish <slug>` or
`sync publish-diagnosis` for this repair: the canonical agent already owns the
attachment path and summary composition, and the existing commands already own
all tracker adapter logic. A combined command can be designed later only if a
separate requirement for retry/idempotency or non-agent callers emerges.

The Hero anchor check on 2026-07-19 supports this direction. The
`harness-changes-cover-all-targets` tripwire requires the canonical workflow
change to propagate and be verified across all six install targets.

---

## Issue

Session-originated report; no Hero tracker issue is linked.

Observed workspace:
`/Users/developer/projects/hpe/repository/morpheus`

Observed local Jira-backed spec:
`.hero/planning/bugs/morph-14509-backup-creation-for-a-vm-with-shared-g/spec.md`

Observed Hero Code in-progress transcript:
`~/.config/hero-code/sessions/8E62F58E-CBE3-4276-BC07-CA752C1FCBC5/in_progress.json`

The transcript records:

1. User: “can you diagnose this bug?”
2. Hero Code: reports a completed local root-cause diagnosis.
3. User asks whether it was posted to Jira.
4. Hero Code says it was local-only.
5. User recalls the `/diagnose` tracker-posting contract and asks for a check.
6. Hero Code incorrectly “confirms” that no posting step exists and notes that
   `hero_tracker` is import-only.

No credential value was needed or read for this diagnosis. No Jira request was
made.

## Problem Statement

### Expected

When a spec has a tracker identity and the selected integration is ready,
`/diagnose` finishes by publishing:

1. the full diagnosis document as `<tracker_id>-diagnosis.md`; and
2. a concise comment naming the root cause, fix location, severity, and
   attachment.

The final response says whether publication was posted, not applicable, denied,
failed, or partially completed.

### Actual

Hero Code wrote the diagnosis only to the local Markdown spec and still
reported diagnosis completion. It had no executable tracker-write operation,
then incorrectly told the user that Hero had no such workflow contract.

### Minimal reproduction

1. In Hero Code, configure a Jira integration and import bugs.
2. Select an imported bug with `tracker_id`.
3. Ask in natural language: “can you diagnose this bug?”
4. Let the model call `hero_diagnose`/investigate and edit the local spec.
5. Observe that no tracker publication operation is available:
   - Bash rejects direct Hero execution.
   - `hero_cli` rejects all `sync` subcommands.
   - `hero_tracker` accepts only `import`.
6. The local spec is updated; the Jira publication step cannot execute.

## Environment Details

- Hero repository HEAD inspected: `ca6006e`
- Hero Code repository HEAD inspected: `e5b6c619`
- Observation date: 2026-07-19
- Hero Code surface: native Swift desktop app
- Tracker: Jira, connected through layered `integrations.connections`
- Imported issue examined: `MORPH-14509`
- Formal `hero peer call hero-code --mode=advisory` was unavailable because the
  configured Claude CLI backend was not logged in; local read-only sibling
  inspection was used as the documented fallback.

---

## Root Cause Analysis

### Confirmed findings

1. `.agents/skills/command-diagnose/SKILL.md:16-18` says the
   debug-investigator owns tracker posting.
2. `.agents/skills/command-diagnose/SKILL.md:40-43` defines a batch diagnosis as
   complete only when the spec is written and the tracker is posted.
3. `domains/engineering/agents/debug-investigator.md:129-137` explicitly
   requires a diagnosis attachment and summary comment when `tracker_id`
   exists.
4. Hero Code's embedded copies retain the same requirements at
   `Resources/hero-content/domains/engineering/commands/diagnose.md:8-10` and
   `.../agents/debug-investigator.md:129-137`.
5. `internal/cli/sync.go:20-29,72-78` still registers `sync comment` and
   `sync attach`.
6. `internal/cli/tracker_ops.go:15-91` implements both operations.
7. `internal/tracker/jira.go:548-621` implements Jira comments and native
   attachments.
8. `internal/cli/diagnose.go:180-188` and
   `internal/serve/mcp_tools.go:2335-2343` emit only local investigation/write
   instructions; tracker publication, kickoff, indexing, anchor checks, and
   publication outcome are omitted.
9. Hero Code expands a command body deterministically only when the user's text
   literally begins with `/` (`ChatView.swift:1758-1771`). Natural-language
   routing relies on model compliance.
10. Hero Code's `skill_run` can load the complete command, but the observed
    natural-language turn used the shorter diagnose surface instead and never
    acquired the canonical completion contract.
11. `ToolExecutor.swift:595-600` rejects every `hero_cli` call whose first
    argument is `sync`.
12. `ToolExecutor.swift:636-640` rejects every `hero_tracker` operation except
    `import`.
13. `AgentLoop.swift:3641-3659` advertises only that import operation, so the
    model cannot even request publication structurally.
14. Hero Code commit `51286a85` (2026-07-15) changed `hero_cli` from blocking
    only `sync connect` to blocking all `sync`, while introducing the
    import-only `hero_tracker`. This made previously exposed comment/attach
    commands unreachable through the app.
15. Hero commit history does not show removal of tracker posting. The retained
    history begins with the agent step and low-level commands together in
    `982742d`; later edits retained the step.
16. `internal/cli/sync_import.go:1303-1323` writes both canonical `tracker_id`
    and provider-specific `<provider>_id`.
17. The observed Morpheus spec contains both `tracker_id: MORPH-14509` and
    `jira_id: MORPH-14509`; missing tracker identity did not cause this failure.
18. `HeroProcessRunner.swift:170-230` already executes the app-selected Hero
    binary without a shell from a structured argument array.
19. `HeroTrackerProcessEnvironment.swift:53-108` already supplies the tracker
    child environment and redacts layered local credentials from output. The
    current single-source implementation does not inject a duplicate token.
20. At Hero Code HEAD, `ToolClass.swift:37-40` classifies `hero_cli` and
    `hero_tracker` as `.process`, and `AgentLoop.swift:3250-3252` excludes them
    from unconditional first-party trust.
21. The current uncommitted Hero Code draft proves the narrow executor change
    is sufficient by adding bounded `comment` and workspace-contained `attach`
    operations. Its separate change that silently first-party trusts both Hero
    tools is unsafe and not required by this diagnosis.

### Load-bearing claim ledger

| Claim | Grounding | Confidence |
|------|-----------|------------|
| The user remembers a real behavior | Canonical and embedded agent instructions plus retained git history | confirmed/read |
| It was agent-executed, not a background hook | Only the agent step invokes the low-level commands; no diagnose engine hook calls tracker APIs | confirmed/read |
| Hero still supports Jira publication | CLI registration, implementations, Jira adapter, and tests remain | confirmed/read |
| The imported bug retained its tracker identity | Current generated-import code and `MORPH-14509` frontmatter | confirmed/read |
| Hero Code cannot perform the promised operation | Bash, `hero_cli`, `hero_tracker`, and advertised schema all close the route | confirmed/read |
| The observed diagnosis was not published by Hero Code | Active transcript reports no attempt; no callable publication route exists | confirmed/read locally; live Jira intentionally not queried |
| The failure is primarily a capability/contract bug, not bad credentials | It occurs before a tracker request can be constructed | confirmed/read |

### Why the misleading answer occurred

The assistant's statement was not a truthful reading of the installed content.
The complete embedded command and agent do contain tracker posting. The shorter
`hero_diagnose` result, however, omits it; the app's native tool catalog then
shows an import-only tracker surface. From that incomplete local view, the model
generalized “I cannot do this” into the false claim “Hero does not do this.”

That is a system defect even though model compliance contributed: a required
workflow side effect must not disappear depending on which first-party Hero
entry point the model chooses.

### Primary classification

`design`: credential hardening replaced direct tracker command access with a
structured app-owned runner, but its allowlist omitted the two existing
commands required by diagnose. Separate short diagnose instructions made that
capability omission look like intended workflow behavior.

### Contributing code defects

- Hero CLI and MCP diagnose renderers omit required closing steps.
- Hero Code blanket-blocks all sync while its protected replacement is
  import-only.
- Hero Code's natural-language routing promise has no deterministic convergence
  gate; only literal slash commands are expanded by the app.

---

## Code Flow (End to End)

1. `internal/cli/sync_import.go:1287-1340` — Jira issue import creates a local
   bug spec with canonical `tracker_id` and Jira-prefixed fields.
2. Hero Code `ChatView.swift:811-817,1758-1771` — a literal `/diagnose` is
   expanded to the full command; natural language remains unchanged and must be
   routed by the model.
3. Hero Code embedded
   `Resources/.../engineering/commands/diagnose.md:8-20` — the correct command
   delegates the complete workflow to debug-investigator.
4. Hero MCP `internal/serve/mcp_tools_def.go:306-315` — separately advertises
   `hero_diagnose` as preparation plus investigation instructions.
5. Hero MCP `internal/serve/mcp_tools.go:2328-2345` — returns spec content and
   local-only instructions, omitting the canonical close.
6. The model investigates and writes the local spec.
7. Canonical debug-investigator
   `domains/engineering/agents/debug-investigator.md:129-137` would next attach
   and comment.
8. Hero Code `AgentLoop.swift:3641-3659` offers no publication-shaped tracker
   operation.
9. Hero Code `ToolExecutor.swift:595-600` rejects the low-level route through
   `hero_cli`.
10. Hero Code `ToolExecutor.swift:636-702` runs only tracker import through the
    protected environment.
11. Hero `internal/cli/tracker_ops.go:40-91` and
    `internal/tracker/jira.go:548-621` are never reached.
12. Jira is never contacted; Hero Code reports only local completion.

---

## Key Files

### Hero workflow contract

| File | Lines | Relevance |
|------|-------|-----------|
| `.agents/skills/command-diagnose/SKILL.md` | 16-30, 40-45 | Canonical Codex workflow promises tracker posting |
| `domains/engineering/agents/debug-investigator.md` | 123-137 | Defines the attachment and summary-comment close |
| `domains/engineering/skills/debugging-investigation/SKILL.md` | 1-110 | Required diagnosis report shape |
| `domains/engineering/skills/spec-format/SKILL.md` | 506-510 | General tracker posting rule for specs |

### Hero incomplete shortcut surfaces

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/diagnose.go` | 123-190 | CLI diagnosis preparation omits the closing contract |
| `internal/serve/mcp_tools.go` | 2288-2345 | MCP diagnosis duplicates the incomplete instructions |
| `internal/serve/mcp_tools_def.go` | 306-315 | Tool description makes the shortcut attractive |

### Hero tracker capability and identity

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/sync.go` | 15-30, 63-79 | Registers comment and attach |
| `internal/cli/tracker_ops.go` | 15-121 | Implements operations and stable integration selection |
| `internal/tracker/jira.go` | 548-621 | Jira REST comment and attachment implementations |
| `internal/cli/sync_import.go` | 1287-1340 | Writes `tracker_id` plus provider fields |
| `internal/config/integrations.go` | 54-83, 421-454 | Resolves explicit, role, and default stable integration IDs |

### Hero Code client boundary

| File | Lines | Relevance |
|------|-------|-----------|
| `packages/hero-swift/Sources/HeroSharedApplication/Chat/ChatView.swift` | 811-817, 1758-1771 | Only literal slash syntax is deterministically expanded |
| `packages/hero-swift/Sources/HeroSharedApplication/Engine/ToolExecutor.swift` | 585-702 | Blocks generic sync; protected tracker tool is import-only |
| `packages/hero-swift/Sources/HeroSharedApplication/Engine/HeroProcessRunner.swift` | 170-230 | Runs the app-selected Hero binary from structured arguments without a shell |
| `packages/hero-swift/Sources/HeroSharedApplication/Sync/HeroTrackerProcessEnvironment.swift` | 53-108 | Supplies the tracker child environment and redacts layered credentials |
| `packages/hero-swift/Sources/HeroSharedApplication/Engine/AgentLoop.swift` | 3021-3096, 3143-3252, 3600-3659 | Permission behavior and advertised native tool schemas |
| `packages/hero-swift/Sources/HeroSharedApplication/Engine/ToolClass.swift` | 14-40 | Classifies `hero_cli` and `hero_tracker` as process tools |
| `packages/hero-swift/Sources/HeroSharedApplication/Engine/ConversationToolRuntime.swift` | 292-330 | `skill_run` can load the full command, but is not enforced |
| `apps/hero-desktop-mac/Tests/HeroDesktopTests/ToolExecutorTests.swift` | 180-400 | Tests deliberate blanket sync rejection and import-only credential path |
| `apps/hero-desktop-mac/Tests/HeroDesktopTests/AgentLoopPermissionTests.swift` | 287-316 | Confirms native Hero process tools are not silently first-party trusted |

---

## Secondary Defects

1. **Shortcut instructions omit the canonical close.** CLI/MCP diagnosis
   preparation can still lead an agent to stop after writing the local spec.
2. **Natural-language routing is not deterministic.** Literal `/diagnose`
   expands the command; natural language depends on the model loading the same
   installed workflow through `skill_run`.
3. **Permission widening would create a second bug.** `hero_cli` and
   `hero_tracker` are process tools in the shipped code. Treating every native
   Hero call as silently first-party trusted would bypass plan/process policy
   for the tracker mutation and is neither necessary nor part of this fix.
4. **Stable integration provenance is a separate enhancement.** Imported specs
   do not retain the stable connection selected by `--integration`. That did
   not cause this single-integration reproduction and must not enlarge this
   parity repair.
5. **Imported spec hygiene drift.** The observed imported spec has a placeholder
   kickoff, no `domain`, no frontmatter `root_cause_class`, and
   `severity: p3 low` rather than Hero's severity enum. These are separate import
   and diagnosis-quality defects, not the cause of missing publication.
6. **The current assistant gave a factually false audit result.** It claimed the
   embedded debug-investigator lacked tracker posting even though step 9 is
   present. Tool/capability self-inspection did not distinguish “unavailable in
   this client” from “absent from Hero.”

---

## Goal

A tracker-backed diagnosis follows the same canonical workflow in Hero Code as
in Claude Code. Hero Code executes the workflow's existing first-party Hero CLI
attachment and comment commands through its protected, version-compatible
tracker runner, with normal mutation approval and truthful success/failure
reporting. No credential or arbitrary command surface is exposed.

## Suggested Fix Approach

### 1. Expose the existing tracker commands through Hero Code's existing runner

**Sibling files:**
`packages/hero-swift/Sources/HeroSharedApplication/Engine/ToolExecutor.swift`,
`packages/hero-swift/Sources/HeroSharedApplication/Engine/AgentLoop.swift`, and
focused `ToolExecutorTests.swift` coverage.

Extend the structured `hero_tracker` operation allowlist from `import` to
`import`, `comment`, and `attach`:

```swift
case "comment":
    arguments = ["sync", "comment", issueID, message]
case "attach":
    arguments = ["sync", "attach", issueID, workspaceFile, displayName]
```

Run those arrays with the already injected `HeroProcessRunning` instance and
`HeroTrackerProcessEnvironment.childEnvironment()`, then redact through
`HeroTrackerProcessEnvironment.redact`. Validate bounded issue IDs/messages;
resolve and canonicalize attachment paths inside the selected workspace.

**Why:** this is the same Hero CLI execution path Claude Code reaches from the
canonical agent instructions, while retaining Hero Code's bundled-binary,
credential, structured-argument, and redaction guarantees. It does not expose
arbitrary `sync` or copy adapter logic into Swift.

### 2. Make every diagnose route use the canonical workflow

**Files:** `internal/cli/diagnose.go`, `internal/serve/mcp_tools.go`,
`internal/serve/mcp_tools_def.go`, Hero Code's routing prompt/installed skill
loading, and canonical/generated diagnose content only if wording changes.

Claude Code reads the diagnose command, delegates to debug-investigator, then
the agent runs the two Hero CLI commands in step 9. Hero Code must do the same:
literal `/diagnose` already expands the command, while natural-language
diagnosis must load `command-diagnose`/debug-investigator through `skill_run`
instead of treating the shorter `hero_diagnose` preparation result as the
whole workflow.

At minimum, make CLI/MCP preparation output include the canonical postback
step and state that preparation is not completion. Prefer pointing to/loading
the installed canonical skill over maintaining a second long renderer.

### 3. Preserve the existing permission model

**Sibling files:** `ToolClass.swift`, `AgentLoop.swift`, permission tests.

Keep `hero_cli` and `hero_tracker` classified as `.process`. Plan mode denies
them; other modes follow the existing process/external-mutation decision and
approval policy. Do not make these tools unconditional `first_party` silent
allows merely because the app owns the runner. A denied command leaves the
local diagnosis intact and must be reported as not posted.

No new permission class or external-service subsystem is required for this
repair.

### 4. Report the two command outcomes truthfully

The workflow already calls attachment and comment separately. It should call
tracker publication complete only after both commands return success. If one
fails or is denied, state exactly which artifact was posted and which was not;
do not claim an atomic or retry-safe outcome that the tracker APIs cannot
provide.

### 5. Verify installed workflow and Hero Code tool parity

Verify the canonical diagnose step across all six Hero install targets and the
Hero Code embedded pack. Add a focused Hero Code contract test that the
diagnose workflow's required `comment` and `attach` operations are both present
in the native `hero_tracker` schema and executor.

---

## Changes

1. Add `comment` and `attach` to Hero Code's existing structured
   `hero_tracker` allowlist and tool schema, mapping directly to the existing
   Hero CLI commands.
2. Keep execution on the app-selected Hero runner with the existing tracker
   child environment and output redaction.
3. Make natural-language, CLI, and MCP diagnosis paths load or repeat the
   canonical diagnose workflow's tracker-posting close.
4. Preserve normal `.process`/external-mutation permission behavior and
   truthful per-command completion reporting.
5. Add focused Hero, Hero Code, and installed-content parity tests.

## Acceptance Criteria

- **AC-1:** WHEN `/diagnose` completes for a spec with `tracker_id` and a ready tracker integration THE SYSTEM SHALL run Hero's existing `sync attach` and `sync comment` commands through Hero Code's structured tracker runner
- **AC-2:** WHEN Hero Code invokes tracker comment or attachment THE SYSTEM SHALL use the app-selected Hero runtime, existing tracker child environment, structured arguments, and redacted output
- **AC-3:** IF either publication command is denied or fails THEN THE SYSTEM SHALL identify the failed artifact and SHALL NOT claim that tracker publication completed
- **AC-4:** WHEN a bug has no `tracker_id` THE SYSTEM SHALL preserve local-only diagnosis behavior and report publication as `not_applicable` without an external request
- **AC-5:** WHEN CLI `hero diagnose`, MCP `hero_diagnose`, literal `/diagnose`, or natural-language diagnosis routing is used THE SYSTEM SHALL surface the canonical tracker attachment/comment close
- **AC-6:** WHEN Hero Code evaluates comment or attach THE SYSTEM SHALL preserve `.process` classification, deny in plan mode, and honor the active mode's normal approval policy
- **AC-7:** THE SYSTEM SHALL NOT expose tracker credentials to Bash, generic `hero_cli`, command arguments, transcripts, or diagnostics
- **AC-8:** THE SYSTEM SHALL keep arbitrary `hero sync` blocked while allowing only validated `import`, `comment`, and `attach` tracker operations
- **AC-9:** THE SYSTEM SHALL use the existing Hero tracker adapters and SHALL NOT reimplement provider publication logic in Swift
- **AC-10:** THE SYSTEM SHALL verify canonical workflow propagation across `opencode`, `cursor`, `claude`, `copilot`, `codex`, `generic`, and the Hero Code embedded pack

## Boundaries

- Do not contact Jira or retroactively post `MORPH-14509` during diagnosis or
  delivery tests.
- Do not expose arbitrary `hero sync` through Hero Code.
- Do not pass tracker credentials through Bash, generic `hero_cli`, command
  arguments, transcript text, or committed configuration.
- Do not reimplement Jira/GitHub/Linear/GitLab adapters in Swift.
- Do not require MCP for Hero Code to run a first-party Hero CLI command.
- Do not add a combined diagnosis-publication command, result protocol,
  idempotency subsystem, or stable-integration migration in this fix.
- Do not silently auto-trust `hero_cli`/`hero_tracker` ahead of the existing
  plan/process permission checks.
- Do not make tracker publication a hidden background hook unrelated to the
  explicit diagnose workflow.
- Do not fold the imported-spec severity/domain/kickoff cleanup into this fix;
  record it separately if prioritized.

## Risks

- Attachment and comment remain two external writes; partial success must be
  reported honestly and retries can duplicate tracker artifacts.
- A client may bundle an older Hero binary than the workflow content it embeds;
  parity tests must exercise the app-selected runtime.
- Natural-language routing remains probabilistic if it can still treat the
  short diagnose preparation result as the complete workflow.
- Broadly marking native Hero tools first-party trusted would silently bypass
  intended mutation approval; leaving all tracker operations blocked would
  preserve the original regression.

## Validation

### Existing test review

- `internal/tracker/tracker_test.go:1884-1953` covers Jira comment and
  attachment HTTP requests.
- `internal/tracker/tracker_test.go:1954-2007` covers GitHub comment and
  attachment-as-comment fallback.
- `internal/cli/sync_import_test.go:200-310` covers imported `tracker_id`.
- `internal/cli/diagnose_test.go` covers diagnosis preparation output but does
  not assert tracker publication instructions or outcomes.
- Hero Code `ToolExecutorTests.swift:180-400` covers sync rejection,
  import-only protected execution, and credential isolation.
- Hero Code `AgentLoopPermissionTests.swift:287-316` covers native Hero process
  approval behavior.
- Hero Code `TrackerRoundTripTests.swift` covers connect/import/field sync, not
  diagnosis publication.

### Test changes needed

1. Hero CLI/MCP output tests asserting the tracker attachment/comment close is
   present and preparation is not represented as completion.
2. Hero Code ToolExecutor tests that accept validated `comment`/`attach`, reject
   arbitrary sync and path traversal, use the app-selected runner/tracker
   environment, and redact output.
3. Hero Code permission tests confirming process classification, plan denial,
   and the active modes' existing approval behavior.
4. A Hero Code seam test: local diagnosis fixture → approved attachment →
   approved comment, with independent command failure coverage.
5. A packaging contract test comparing embedded diagnose requirements with
   native `hero_tracker` operations.
6. Six-target install propagation verification.

### Regression scope

- Tracker import and refresh
- Jira/GitHub/Linear/GitLab publication behavior
- Hero Code credential redaction and bundled-binary selection
- Natural-language versus literal slash routing
- Permission and partial-failure reporting
- CLI/MCP output parity

## Notes

- No live Jira status/comment/attachment read was performed, so the remote
  absence is not independently queried. The local transcript says no post was
  attempted, and the inspected shipped Hero Code HEAD has no callable route
  that could have made one.
- The existing uncommitted Hero Code planning spec
  `.hero/planning/bugs/desktop-diagnosis-tracker-writeback-unreachable/spec.md`
  independently reaches the same client-side conclusion. The current Hero Code
  working tree also contains an uncommitted comment/attach executor draft; this
  diagnosis did not edit it. Keep that narrow executor/schema work, but reject
  its unconditional first-party permission bypass and rewrite those tests to
  preserve the shipped process policy.
- Stable integration provenance, atomic publication receipts, and retry
  idempotency remain possible follow-up designs, not prerequisites for restoring
  the remembered behavior.

## Recap

The remembered tracker postback did not disappear from Hero's intended
workflow or Go implementation; it was always an explicit agent closing step,
not a background hook. Hero's shortcut diagnose surfaces now omit that step,
and Hero Code's protected tracker boundary makes it impossible to execute
anyway, so `MORPH-14509` correctly stayed local while the assistant incorrectly
claimed the feature never existed. The engineer's challenge supersedes the
original fix complexity: repair requires only canonical workflow parity plus
validated access to the existing attach/comment commands through Hero Code's
existing runner and permission boundary. MCP and a new publication command are
not required.

## Investigation History

### Round 1 — Initial diagnosis
- **Date**: 2026-07-19T14:16:51Z
- **Agent**: debug-investigator
- **Root cause**: Hero's tracker-posting workflow remained intact, but Hero Code's blanket `sync` block and import-only protected tracker surface made the required publication commands unreachable while short diagnose instructions omitted the close.
- **Confidence**: High
- **Key evidence**: Canonical and embedded diagnose content require attachment/comment; Hero still implements both commands; Hero Code HEAD rejects all generic sync and advertises only tracker import.

### Round 2 — Challenged (layer)
- **Date**: 2026-07-19T20:14:02Z
- **Agent**: debug-investigator
- **Challenged by**: engineer
- **Feedback**: "seems like we f'd up here - i mean its just a command - and hero-code can run a command"
- **Revised root cause**: The root-cause evidence stands, but the repair is a missing allowlisted command path plus incomplete diagnose instructions, not a missing publication subsystem.
- **What changed**: Replaced the proposed combined publisher/MCP bridge, structured result protocol, idempotency work, stable-integration provenance, and special permission subsystem with direct validated exposure of existing `sync attach` and `sync comment` through Hero Code's existing app-selected runner; explicitly preserved shipped process approval behavior.
- **Confidence**: High
