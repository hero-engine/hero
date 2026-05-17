---
description: Process inbound intakes into a triaged status — linked, merged, promoted, or rejected with reason.
---
Route this triage request to the `intake-triager` agent.

## Pre-flight: identify scope

Determine what is being triaged:
- A single intake slug → triage that one item.
- No argument → triage all `intake` specs with `status: new` in `.hero/planning/intake/`.
- A filter (e.g. `--segment enterprise`, `--source support`, `--since 7d`) → triage matching items.

Use `hero search --list --type intake --status new` to enumerate.

## Sub-routing

Before `intake-triager` runs:
- **If scope > 1 item**, invoke `duplicate-detector` first to cluster near-duplicates. The triager processes clusters as units, not items.
- **If an intake is ambiguous** (no clear ask, no source quote, or no segment tag), invoke `pm-investigator` first via `/diagnose` to extract the actual customer ask from the raw signal. Skip the item until investigation lands.

## Output

For each item, the triager flips status to one of `linked`, `merged`, `promoted`, or `rejected` and writes the rationale into the spec:
- `linked` — attached to an existing `initiative` slug.
- `merged` — merged into another `intake` slug (duplicate).
- `promoted` — became (or seeded) a new `initiative`.
- `rejected` — with a one-line reason in the spec body.

Emit a one-line log per item to chat:

```
intake-acme-csv-export → linked → roadmap-self-serve-exports (3 prior signals, enterprise segment)
intake-billing-flicker → rejected → bug, not a product ask; refiled via /diagnose
```

**SLA reminder.** Items at `status: new` for more than 24h are overdue. If any in scope are older than 24h, surface that in the run summary.

After triage, log notable promotions or rejections via `hero event decision_made` so other sessions see the inbox state move.

Request: $ARGUMENTS
