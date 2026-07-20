# Hero Code spec-out prompt — Durable Attention consumer

Use this as the body of:

```bash
hero peer call hero-code \
  --mode=spec-out \
  --related-spec durable-attention \
  --reason "Hero is defining the Project Mail and Personal Focus contracts; Hero Code owns the native Attention/Today consumer experience" \
  "<prompt body>"
```

Fallback while the current peer runner is unavailable or the Hero Code
worktree is carrying unrelated active changes: paste `/design` followed by the
prompt body below into a Hero Code task. The design will then run natively
against that repository's Swift conventions without a cross-repo write from
Hero.

## Prompt body

Design Hero Code's consumer side of Hero's `durable-attention` initiative.
Inspect the current Swift implementation before designing, especially:

- Advisor/Dashboard/Now/inbox composition and navigation;
- `AdvisorViewData.FocusItem`, `FocusItemCard`, `NeedsAttention`, and their
  current data sources;
- `AdvisorSessionLauncher` and the project/task launch flow;
- Intake presentation and action handling;
- Hero service, CLI, MCP, and schema/fixture client boundaries;
- event subscription, refresh, and stale-action behavior.

Hero core will own two distinct durable primitives:

1. **Project Mail:** project-addressed immutable messages with threads,
   acknowledgements, provenance, explicit promotion to Intake/Spec, and no
   automatic execution.
2. **Personal Focus:** user-global prompt-backed intentions with states
   `inbox`, `today`, `later`, and `done`.

Harness task lists remain ephemeral, run-owned state. Intake remains project
pre-commitment. Specs remain committed work. Jobs/Runs remain runtime execution.
Do not merge these lifecycles.

Hero will expose a versioned Attention read model with stable source IDs, source
kind, project reference, display fields, timestamps, unread/Today state,
provenance, and explicitly supported actions. Hero will also publish JSON Schema
and golden fixtures. Hero Code must consume that boundary rather than parse
Hero-owned Mail/Focus storage directly, infer capabilities from status strings,
or define parallel Swift-only lifecycle values.

There are two important existing semantic collisions to resolve explicitly:

- `AdvisorViewData.FocusItem` is currently a derived, spec-backed advisor
  candidate. It is not the new durable user-owned Focus primitive.
- `NeedsAttention` currently represents backlog-health signals. It is not the
  new combined Mail and Focus read model.

The existing `AdvisorSessionLauncher` launches a spec slug through a fixed Hero
workflow action. A durable Focus launch instead supplies project identity and an
arbitrary saved executable prompt. Reuse presentation and launch infrastructure
where it fits, but do not force the new contracts into the old semantics.

Design the smallest Hero Code-owned work that:

- Adds one user-global Attention/Today surface combining Project Mail and
  Personal Focus without merging their underlying models.
- Presents mail actions such as Reply, Acknowledge, Add to Today, Promote, and
  Dismiss only when advertised by Hero's contract.
- Presents deferred-work suggestions as Do next, Today, Later, and Dismiss; no
  durable Focus Item exists until the user accepts.
- Launches a Focus Item into the correct project as a new task/session using its
  saved prompt; the new harness run owns its internal task list.
- Shows unread mail and Focus updates from Hero's event/read-model boundary.
- Handles missing projects, unavailable Hero service, unknown additive fields,
  and stale action attempts safely.
- Extends the existing Now/inbox/provider architecture if it fits instead of
  creating a second dashboard inbox.
- Includes consumer compatibility tests against Hero's golden fixtures and
  mocked action responses.

Explicitly out of scope for Hero Code:

- Mailbox or Focus storage authority;
- cross-project transport and delivery;
- promotion/provenance logic;
- autonomous execution, approval, scheduling, or monitoring;
- synchronization with Claude, Codex, Copilot, Cursor, or other harness todo
  systems;
- Swift-only mail/focus statuses or direct parsing of Hero storage.

After designing, report:

1. The Hero Code spec slug.
2. Dependencies on `durable-attention-contracts`,
   `project-mail-triage-and-provenance`,
   `deferred-work-suggestion-contract`, and `attention-read-model-v1`.
3. Any fields, actions, schemas, or fixtures Hero must provide.
4. Whether the current Advisor/Now/inbox architecture can absorb this cleanly.
5. Any terminology changes needed to avoid the existing `FocusItem` and
   `NeedsAttention` collisions.
