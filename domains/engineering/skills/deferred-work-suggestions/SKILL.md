---
name: deferred-work-suggestions
description: Propose meaningful out-of-scope follow-up work without silently creating a personal commitment.
metadata:
  audience: all-agents
  purpose: consent-boundary
---

## When to use this skill

Create one deferred-work suggestion only when all of these are true:

- The work is meaningful and worth returning to.
- It is outside the currently accepted scope.
- The title, reason, and exact executable prompt are concrete enough for a fresh session to resume.
- It is not an unfinished required step, acceptance criterion, checklist item, Completion Ledger item, or harness todo from the current task.

Do not turn required work into a suggestion. Finish the accepted task and its closing gates.

## Consent boundary

A suggestion is advisory output, not Focus and not a personal commitment. Invoke the structured Hero operation once, then continue and finish the current task. Never call `hero focus add` on the user's behalf and never claim a suggestion was saved by mentioning it only in prose.

Use the MCP tool `hero_focus_suggest` when available. Provide `title`, `reason`, the exact `prompt`, optional `project`, typed `source_kind` and `source_id`, and a stable `idempotency_key`.

For CLI-only environments, keep the prompt out of argv:

```text
hero focus suggest --title "..." --reason-file <path> --prompt-file - --project . --source-kind run --source-id "..." --idempotency-key "..."
```

Write the exact prompt to stdin and the reason to a private file; neither body belongs in shell argv. Omit `--project` only when the proposal is intentionally unbound.

Proposal persistence is best-effort and must not fail the primary task. If the structured operation fails, report the structured error if useful; do not simulate success with assistant prose and do not retry with a new idempotency key.

Only the user may accept a pending proposal as Today, Later, or Do Next. Acceptance creates or reuses Focus. Do Next returns a launch intent but does not start a session; the client owns session creation. Dismissal creates no Focus.

Deferred suggestions are not part of the Completion Ledger and never replace required delivery steps.
