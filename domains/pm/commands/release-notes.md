---
description: Draft customer-facing release notes for shipped items — pulls shipped status from the cross-domain graph.
---
Route to `stakeholder-communicator`, loading `release-notes-writing`. Drafts from the shapes below.

## Scope

- **`--since <duration>`** (e.g. `--since 1w`, `--since 30d`) → ship notes for everything shipped in that window.
- **A list of spec slugs** → notes for exactly those specs.
- **No arguments** → ask the user which time window or spec set; don't guess.

## Shipped-status source of truth

Pull shipped status from the **graph** — not from tracker workflow status, not from PR-merge dates.

Under the unified type model, a spec is "shipped" only if:
1. Its `status` is `completed`, **and**
2. The most recent `owner_history` row shows engineering close-out (the bitemporal history is the cross-domain record).

Specs whose engineering work merged but whose `owner_history` never recorded a close-out are **not eligible** for release notes — they're orphan deliveries. Surface the gap to the user but don't include them.

## Output

Two artifacts (the agent picks based on context or the user's request):

- **Customer-facing notes** → written to `.hero/planning/release-notes/<window-or-tag>/customer.md`. Tone: outcomes-not-features ("you can now export filtered CSVs from any list view" — not "added CSVExport handler to ListController"). Grouped by theme, not by spec.
- **Internal update variant** (optional, via `--internal` or when the user asks for "what shipped this week" style) → `.hero/planning/release-notes/<window>/internal.md`. Includes spec slugs, owners, and links back to the originating PRDs/initiatives so the team can trace.

A one-line log to chat names the file paths and the count of shipped items included.

After notes land, log `hero agent events delivery_complete` so other sessions see the release-notes pass happened (avoids duplicate drafts in parallel sessions).

Request: $ARGUMENTS
