---
title: "Cold-Start Trust Hardening — Fail Loud, Never Mislead, at First Use"
type: explainer
synthesized_from:
  - cold-start-trust-hardening
last_synthesized: 2026-06-23
source_initiative: cold-start-trust-hardening
tags: [cold-start, trust, relations, scan, check, verify, dx]
---
# Cold-Start Trust Hardening

## What it is

A set of **affordance fixes** that make Hero fail loudly and clearly at first
use, instead of degrading silently, crying wolf, or letting an unrelated signal
read as the cause. It is not a re-architecture — it's a coordinated pass over the
commands a brand-new user hits first (relation parsing, `scan`, `check`,
`verify`, `queue`/`blocked`) so that when Hero can't do what the user expects, it
says so at the point of failure, naming the real cause. The work was grounded in
a real first-use session (the `candy` project) where an agent spent ~an hour
reaching a confident-but-wrong conclusion because Hero's failures were silent or
misleading.

## Surfaces / entry points

- **Spec frontmatter** — relation declarations (`relations:` block, plus
  shorthands `parent:`/`initiative:`/`depends_on:` and block-style `child:` lists).
- **`hero check`** — severity-aware summary; `[[wikilink]]` advisory.
- **`hero scan`** — Tier-2 (LLM enrichment) labeling.
- **`hero spec verify`** — lifecycle-gated delivery checks; initiative
  auto-completion.
- **`hero queue` / `hero blocked`** — blocker source parity.
- **`domains/engineering/AGENTS.md`** — documents the canonical relation syntax.

## How it works

Each shipped fix removes one class of silent/misleading failure:

- **Relation recognition** (`internal/spec/spec.go`). The parser recognizes the
  shorthands first-use agents reach for — `initiative:` (→ a `parent` relation),
  `depends_on:` (→ `depends-on`), and newline block-style `child:` lists — and
  maps `relates-to` to a real edge. Relations that resolve to nothing are
  surfaced rather than dropped at ingest.
- **Wikilink advisory** (`internal/cli/check.go`). `hero check` warns when a spec
  body contains `[[wikilinks]]` that look like intended relations, because
  wikilinks never become graph edges — pointing the user at the `relations:`
  block instead.
- **Tier-2 labeling** (`internal/cli/scan.go`, `internal/extract/auto.go`). The
  scan output labels LLM enrichment as *optional* ("enrichment (optional LLM …)")
  so the missing-API-key line can never read as "the structural graph is off."
  The deterministic Tier-1 edge graph needs no key.
- **`check` severity** (`internal/cli/check.go`). The human output is
  severity-aware and collapses the dominant benign category (scaffolds missing
  `## Kickoff`) so a healthy scaffold-heavy workspace no longer reads as ~100
  failures.
- **Lifecycle-gated verify** (`internal/cli/verify.go`). `hero spec verify` runs
  the full delivery gates (Completion Ledger, delivery-audit) only for
  delivering/in-review/completed specs; a `planning` draft gets a lifecycle-aware
  message instead of a misleading "no Completion Ledger section found."
- **Initiative auto-completion guard** (`internal/cli/verify.go`,
  `internal/cli/complete.go`). An initiative no longer auto-completes when it has
  declared-but-unmaterialized children; the declared roster must be complete
  first. Archiving moves the whole spec directory (sibling `delivery-audit.md`
  included), not just `spec.md`.
- **queue/blocked parity** (`internal/cli/brief.go`). `hero blocked` derives
  blockers from frontmatter so it agrees with `hero queue` on a fresh clone,
  where `graph.db` hasn't been reingested yet.

## Data & state

- **Frontmatter `relations:` is canonical (Tier-1).** Edges derive from it
  deterministically; it travels in git.
- **`graph.db` is a cache**, gitignored and reingested — which is why
  `hero blocked` was changed to fall back to frontmatter for cold-start parity.
- **`.hero/events.log` is durable and left tracked by design** — it backs
  `hero feed`/`velocity`/clusters; untracking it would lose cross-machine
  activity history. (The `cst-gitignore-events-log` stub was resolved as
  *corrected, not implemented* for this reason.)
- **`hero check` severity tiers** already existed in JSON; the work brought that
  tiering to the human output.

## Gotchas

- **Relation syntax is still exacting.** Inline flow style like
  `- { kind: related, target: X }` silently produces zero edges; the proven
  forms are the top-level shorthands and the block-style `relations:` list with
  `target:`/`kind:` on separate lines. `kind: related` maps to an edge;
  `relates-to` now does too.
- **`events.log` stays tracked on purpose** — its churn on state-changing
  commands is expected for a tracked ledger, not a bug to "fix" by gitignoring.
- **One stub shipped as a correction, not code** (`cst-gitignore-events-log`).
- An open product question was deferred to the user: whether `events.log` should
  remain durable-shared (tracked) or become per-machine (gitignored).

## Related decisions

No `decision` entries were directly referenced by the initiative; its rationale
lives in the initiative's own "Guiding principles" (fail loud at the point of
failure; never let an unrelated signal read as the cause; recognize intent or
reject it; surgical, not a rewrite).

## Developer Notes
