---
title: "Attention Lifecycle Read Awareness — Bounded Context at Chat Boundaries"
slug: attention-lifecycle-read-awareness
type: feature
status: completed
domain: engineering
priority: high
size: medium
horizon: now
created: 2026-07-23
parent: conversational-attention-operability
depends-on: [attention-interaction-consent-contract]
conflicts-with: [attention-conversational-routes]
tags: [attention, lifecycle, resume, recap, context]
relations:
  - target: agent-end-of-turn-recap
    kind: related
  - target: active-context-management
    kind: related
claimed_by: chet-bellows
claimed_at: 2026-07-23T16:00:00-06:00
delivery_method: manual
completed_at: 2026-07-23T22:25:02Z
---

# Attention Lifecycle Read Awareness — Bounded Context at Chat Boundaries

## Context

Hero has one authoritative Attention projection and an MCP
`hero_attention_snapshot` read. It does not yet tell chat agents when to read
that projection, how much of it to place in context, how to distinguish empty
from unavailable or stale state, or when Attention belongs in an end-of-turn
recap.

The current MCP snapshot also returns the projection without a window. Its rows
can contain `body` and unbounded `summary` values, including complete Project
Mail bodies. That is suitable for trusted native projection consumers but not
for automatic chat-boundary awareness. Installing prose alone would therefore
leave the model-facing read unbounded and expose content that should require the
explicit `hero_mail_show` operation.

## Goal

Give every supported chat harness one deterministic, bounded, side-effect-free
Attention read at useful lifecycle boundaries, backed by the existing
projection authority and safe to summarize without loading full Mail bodies
into model context or turning every turn into an inbox poll.

## Kickoff

Adds a compact MCP snapshot window plus portable session, post-mutation, and
recap rules across all six harnesses.

**Status:** in-review — compact projection, lifecycle guidance, all-target
propagation, and full repository validation are complete.

**Pick up at:** cold-audit the Completion Ledger, repair any HOLD finding, then
run `hero spec verify attention-lifecycle-read-awareness`.

→ `/deliver attention-lifecycle-read-awareness`

**Files:** `contracts/attention/projection.go`,
`contracts/attention/schema/v1/attention-snapshot.schema.json`,
`internal/attention/projection/service.go`,
`internal/serve/mcp_tools_attention.go`, `internal/serve/mcp_tools_def.go`,
`core/commands/resume.md`,
`domains/engineering/skills/attention-lifecycle-awareness/SKILL.md`,
`internal/install/attention_guidance.go`
**Skip:** per-turn polling, background hooks, automatic Mail show/read,
generic recap formatting, and any second Attention storage/query authority.

## Design

### One projection authority, two presentation depths

Keep `projection.Service.Snapshot()` and its existing source queries as the
only Attention read authority. Add a pure compact-window projection over its
result for model-facing MCP reads; do not query Mail, Focus, suggestions, or
storage a second time.

`hero_attention_snapshot` always returns the compact window. Its optional
`limit`:

- defaults to 8 rows;
- accepts integers from 1 through 20;
- returns a structured `validation` error for zero, negative, fractional, or
  over-limit values;
- is applied after the existing deterministic Attention ordering.

The compact result preserves the full snapshot revision, generated timestamp,
refresh token, per-source counts, and total count. An additive `window` object
publishes:

- `state`: `current` when the source is available and total is nonzero, or
  `empty` when the authoritative total is zero;
- `limit`;
- `returned`;
- `truncated`.

`counts.total` always describes the authoritative full projection, while
`window.returned` describes the delivered window. Existing API/native projection
consumers retain the full snapshot; only the model-facing MCP adapter applies
the compact presentation.

### Metadata-only rows

The compact MCP view copies rows and strips `body` from every source kind. It
also strips Mail `summary`, because the existing Mail projector copies the
entire message body into both fields. Mail rows therefore expose subject,
sender/project identity, timestamps, unread state, revision, provenance, and
advertised actions, but no body excerpt.

Focus and suggestion summaries are useful for recognizing an intention or
proposal, so the compact view retains them only as a UTF-8-safe 240-byte
excerpt. It strips their `body` copies as well. Full Project Mail inspection
remains `hero_mail_show`; the awareness read must never call it automatically.

The compact transformation is pure and must not mutate the full snapshot or
its source rows. It performs no receipt, lifecycle, suggestion, or commitment
action.

### Three deterministic lifecycle boundaries

Author one reusable
`domains/engineering/skills/attention-lifecycle-awareness/SKILL.md` contract
and reinforce its essential imperatives in the native root instruction file.
The rules are:

1. **Fresh or resumed session.** After the normal Hero resume/context load, if
   `hero_attention_snapshot` is advertised, call it exactly once with
   `limit: 8` before claiming that nothing is pending. Do not block unrelated
   work when Attention is unavailable.
2. **Successful Attention mutation.** Trust the structured mutation result and
   perform at most one `hero_attention_snapshot(limit: 8)` refresh. Never replay
   a write merely to confirm it. If refresh fails, keep the successful source
   result and classify the aggregate view as stale/unavailable.
3. **End of turn.** Do not poll solely to populate a recap. Mention Attention
   only when a known item changed during the turn or the already-read bounded
   snapshot is materially relevant to the user's next step. Use those known
   facts; never append a generic inbox dump.

There is no per-prompt polling, timer, watcher, receipt hook, or mailbox-
triggered model execution.

### Empty, unavailable, and stale are different

Lifecycle guidance and tests use these exact interpretations:

- **empty:** a successful current snapshot with `state: "empty"` and
  `counts.total == 0`;
- **unavailable:** `hero_attention_snapshot` returns structured
  `ActionResult.error.code == "unavailable"` and no successful snapshot;
- **stale:** a previously successful snapshot exists, but a required
  post-mutation refresh later returns unavailable. Preserve its
  `generated_at` and `revision`, label it stale, and do not call it current or
  empty.

Stale is deliberately a client/chat lifecycle classification, not a second
persisted server state. A fresh successful read always replaces it.

### Portable installation without a hook dependency

Add an engineering-only managed section contributor for the concise lifecycle
imperatives. `defaultSections` includes it for the engineering domain before
the shared operational and snapshot-pointer sections. This authors the root
guidance once in Go and renders it into the native file each installed target
actually reads:

- Claude → `CLAUDE.md`;
- OpenCode, Cursor, Copilot, Codex, Generic → `AGENTS.md`.

Install the reusable skill through the existing domain skill pipeline and
update the canonical `core/commands/resume.md` workflow to perform the optional
bounded read. The same sources propagate to all six harness-native skill and
command surfaces. Root guidance remains self-contained for hookless harnesses;
the skill carries the detailed state table and examples. No behavior depends on
Claude Stop/PreCompact hooks.

### Recap ownership boundary

This feature emits only Attention-specific facts and trigger rules. It does not
add a universal recap heading, format, frequency, or suggestion mechanism.
`agent-end-of-turn-recap` may later consume:

- whether Attention changed this turn;
- whether a bounded current snapshot was materially relevant;
- current/empty/unavailable/stale classification;
- counts and the bounded row metadata already in context.

If neither trigger is true, the correct Attention recap contribution is empty.

## Changes

1. Extend `AttentionSnapshot` additively with state and window metadata plus
   validation for coherent totals, returned rows, and limits.
2. Add a pure projection helper that creates the deterministic metadata-only
   window, strips all bodies and Mail summaries, and UTF-8-safely bounds other
   summaries to 240 bytes.
3. Give `hero_attention_snapshot` an optional `limit` input, default it to 8,
   reject values outside 1–20 as structured validation results, and return the
   compact window without changing the HTTP/native full projection.
4. Update MCP tool descriptions and input schema so clients can discover the
   bounded metadata-only behavior.
5. Add the canonical `attention-lifecycle-awareness` skill with fresh/resume,
   post-mutation, recap, side-effect, and state-classification rules.
6. Update `core/commands/resume.md` to invoke one optional bounded snapshot
   after the normal context load when the MCP tool is advertised.
7. Add an engineering-only managed root-section contributor with self-contained
   imperative guidance and wire it into the shared AGENTS/CLAUDE generator.
8. Add focused projection/MCP tests and six-target golden propagation tests for
   the root section, skill, and native resume workflow.

## Acceptance Criteria

- **AC-1:** WHEN `hero_attention_snapshot` is invoked without a limit THE
  SYSTEM SHALL return at most 8 deterministically ordered rows while preserving
  authoritative full counts and snapshot revision.
- **AC-2:** WHEN a caller supplies a valid limit from 1 through 20 THE SYSTEM
  SHALL return at most that many rows and SHALL report `window.limit`,
  `window.returned`, and `window.truncated` coherently; invalid limits SHALL return a
  structured field-specific validation error.
- **AC-3:** WHEN a compact snapshot contains Mail THE SYSTEM SHALL expose no
  Mail body or summary text; WHEN it contains Focus or suggestions THE SYSTEM
  SHALL expose no body and at most a UTF-8-safe 240-byte summary.
- **AC-4:** WHEN compact awareness is derived THE SYSTEM SHALL use the existing
  Attention projection snapshot exactly once, preserve its ordering and
  revision, leave the source snapshot unchanged, and perform no lifecycle or
  receipt mutation.
- **AC-5:** WHEN a successful snapshot has zero authoritative rows THE SYSTEM
  SHALL identify it as `empty`; IF projection authority is unavailable THEN it
  SHALL return structured `unavailable` and SHALL NOT represent that condition
  as empty.
- **AC-6:** WHEN a Hero-aware session starts or resumes and the snapshot tool is
  advertised THE INSTALLED GUIDANCE SHALL require exactly one bounded
  `limit: 8` read after normal resume context and before claiming nothing is
  pending.
- **AC-7:** WHEN an Attention mutation succeeds THE INSTALLED GUIDANCE SHALL
  trust its authoritative structured result, perform at most one bounded
  refresh, and SHALL NOT replay the mutation to confirm it.
- **AC-8:** IF a required post-mutation refresh is unavailable after an earlier
  successful read THEN THE INSTALLED GUIDANCE SHALL classify the earlier view
  as stale using its timestamp/revision and SHALL NOT call it current or empty.
- **AC-9:** WHEN no Attention item changed and the already-read snapshot is not
  relevant during a turn THE INSTALLED GUIDANCE SHALL contribute no Attention
  inbox dump to the end-of-turn recap and SHALL NOT poll solely for recap.
- **AC-10:** WHEN Hero installs OpenCode, Cursor, Claude, Copilot, Codex, or
  Generic THE SYSTEM SHALL place equivalent lifecycle rules in that target's
  native root instruction, skill, and resume workflow surfaces without relying
  on a harness-specific hook.

## Boundaries

- No background watcher, timer, push notification, or mailbox-triggered model.
- No generic recap format, frequency, or suggestion changes.
- No ranking algorithm beyond the existing Attention projection ordering.
- No automatic `hero_mail_show`, mark-read, acknowledge, dismiss, accept,
  promote, create, or other mutation.
- No second Attention digest, storage authority, or source query.
- No requirement that Attention be available for unrelated Hero workflows.
- No native Hero Code UI changes or cached-client persistence contract.

## Risks

- **Compatibility:** changing the full projection would break native clients.
  The compact window therefore lives at the MCP presentation boundary and uses
  additive snapshot metadata.
- **False emptiness:** a transport or authority failure can look like zero
  results if errors are flattened. Structured unavailable results and explicit
  lifecycle language keep the states distinct.
- **Content leakage:** Mail currently duplicates its complete body into
  `summary` and `body`. The compact transform must clear both, with tests using
  recognizable secret/tool-shaped content.
- **UTF-8 truncation:** byte slicing can produce invalid output. Summary bounds
  must preserve valid UTF-8 and never exceed 240 bytes.
- **Guidance drift:** root, skill, and resume surfaces serve different loading
  paths. All-target tests must assert the same numeric limits, boundaries, and
  state vocabulary on each surface.
- **Recap noise:** a start-of-session read does not imply every response should
  repeat the inbox. Relevance/change triggers and no end-only polling keep the
  recap sparse.

## Validation

- `go test ./contracts/attention ./internal/attention/projection ./internal/serve ./internal/install`
- `go test ./...`
- Projection tests with more than 20 mixed rows proving deterministic limits,
  full counts, truncation metadata, immutable input, body removal, Mail summary
  removal, and UTF-8-safe Focus/suggestion summaries.
- MCP tests for omitted/default, minimum, maximum, fractional, zero, negative,
  and over-limit inputs plus unavailable-versus-empty results.
- Spy-source tests proving one snapshot source pass and zero action/mutation
  calls during awareness.
- Six-target install matrix checking native root filenames and equivalent
  lifecycle markers in root guidance, installed skill, and generated resume
  command/prompt/skill locations.
- Repository search proving no new hook, timer, watcher, automatic Mail show, or
  lifecycle action is wired to awareness.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Omitted limit returns at most 8 with full counts/revision | DONE | `TestAttentionMCPLimitValidationAndUnavailableAreStructured` invokes the real MCP handler with omitted input and verifies an 8-row window over 22 authoritative rows; the HTTP/MCP authority test proves revision and counts are preserved. |
| 2 | Valid 1–20 limits and invalid structured errors | DONE | The MCP test exercises 1, 8, and 20 plus zero, negative, fractional, and 21; valid windows report coherent metadata and invalid calls return field-specific `validation` without opening Attention authority. |
| 3 | Compact rows exclude bodies and bound non-Mail summaries | DONE | `TestCompactBoundsRowsAndRemovesBodiesWithoutMutatingSource` proves all bodies are removed, Mail summary is removed, and multibyte Focus summary remains valid UTF-8 at no more than 240 bytes; MCP tests use recognizable private Mail content. |
| 4 | One existing projection pass, immutable compact transform, no mutations | DONE | The compact unit test proves the source snapshot is unchanged; the MCP source spy records one Inbox read per invocation and zero Action calls while the handler delegates once to `Service.Snapshot`. |
| 5 | Empty and unavailable remain distinct | DONE | MCP coverage decodes a successful empty snapshot with `window.state == empty` and a separate structured `ActionResult.error.code == unavailable`; window contract validation rejects state/count mismatch. |
| 6 | Fresh/resume guidance performs one bounded read | DONE | `core/commands/resume.md`, the reusable skill, and root section require exactly one advertised-tool call with `limit: 8`; `TestAttentionLifecycleGuidanceReachesAllHarnessNativeSurfaces` verifies every installed form. |
| 7 | Successful mutation trusts result and refreshes at most once | DONE | Root and skill guidance explicitly preserve the authoritative mutation result, permit at most one bounded refresh, and prohibit replay for confirmation; six-target golden checks pin the wording. |
| 8 | Failed refresh classifies prior view stale | DONE | The reusable skill defines stale from the prior `generated_at` and `revision`, and both root/skill propagation checks require unavailable/stale never be translated into empty. |
| 9 | Irrelevant turns do not poll or dump Attention | DONE | Root and skill guidance prohibit recap-only polling and generic inbox dumps, and restrict output to changed or materially relevant known facts; all six installed skill/root surfaces are asserted. |
| 10 | Equivalent root, skill, and resume rules reach six harnesses | DONE | The install matrix passes for OpenCode, Cursor, Claude, Copilot, Codex, and Generic at each target's native root, skill, and command/prompt path; PM exclusion proves the contributor is engineering-scoped. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add snapshot state/window contract and validation | DONE | Added raw-compatible state constants, `AttentionWindow`, additive schema, and coherence validation while old snapshots remain valid without `window`. |
| 2 | Add pure deterministic compact projection | DONE | `projection.Compact` preserves authority metadata/order, copies rows, removes bodies/Mail summaries, bounds other summaries, and publishes window metadata. |
| 3 | Bound `hero_attention_snapshot` with structured validation | DONE | The handler defaults to 8, accepts 1–20, rejects invalid values before service construction, and compacts the single authoritative snapshot result. |
| 4 | Publish discoverable MCP input/behavior | DONE | Tool schema advertises optional numeric `limit`; description states default/max, metadata-only rows, and explicit `hero_mail_show` for full Mail bodies. |
| 5 | Add canonical lifecycle-awareness skill | DONE | Added session, mutation, recap, state, stale, content, and side-effect rules in one engineering domain skill. |
| 6 | Integrate bounded awareness into resume | DONE | The canonical core resume workflow conditionally performs one advertised MCP read and preserves the empty/unavailable and no-side-effect boundaries. |
| 7 | Add engineering root section contributor | DONE | One managed contributor renders the essential imperative policy into each native root file; engineering pack/fallback skill catalogs remain byte-aligned. |
| 8 | Add runtime, contract, docs, and six-target validation | DONE | Focused and full `go test ./...` suites pass; installer count docs now reflect 57 skills, and Hero Code handoff documents full HTTP versus compact MCP depth. |

### Exercise-the-feature check

- [x] The model-facing handler was exercised with 22 real projected Mail rows
  at omitted, minimum, and maximum limits; results preserved full counts and
  revision, exposed no body text, performed no action calls, distinguished
  empty from unavailable, and installed equivalent lifecycle instructions into
  all six harnesses.

### Excellence Bar self-check

- [x] Yes — the implementation reuses one authority, is additive for native
  consumers, defaults safe for chat, bounds both rows and content, fails
  structurally, avoids hook dependence, and is covered by focused plus complete
  repository validation.
