---
title: Spec Triage on Import — Duplicate Detection and Convention Conflict Check
slug: spec-triage
type: feature
status: completed
milestone: v0.3
tags: [specs, import, triage, deduplication, conventions, intake]
created: 2026-04-13
relations:
  - target: knowledge-contradiction-detection
    kind: extends
  - target: hero-ask
    kind: related
horizon: now
---

## Goal

When a spec is added to Hero — via `hero add`, `hero prime`, or manual file drop — automatically check it for duplicates, convention conflicts, and structural issues before it enters the active corpus. Surface problems at intake rather than letting bad data silently accumulate.

## Problem

Hero's knowledge base and spec corpus degrade over time through:

1. **Duplicate specs** — two specs covering the same feature or convention, written independently; agents get conflicting signals
2. **Convention conflicts** — a new spec prescribes behavior that directly contradicts an existing convention (e.g., "use stdlib log" vs. "use zerolog")
3. **Orphaned specs** — specs imported with `relates_to` targets that don't exist in the corpus
4. **Structural violations** — missing required frontmatter fields, invalid status values, bad slug format

Without intake triage, these problems accumulate silently. The only signal is confusion when an agent acts on contradictory knowledge.

## Design

### Trigger Points

Triage runs automatically on:

1. `hero add <spec-file>` — explicit import
2. `hero prime` — during spec generation from a repo scan
3. `hero scan` — when generating knowledge entries
4. **Manual file drop** — when `hero serve` detects a new `.hero/` file via file watcher

Triage can also be run explicitly:

```
hero triage                    # triage all specs and knowledge
hero triage <slug>             # triage a specific spec
hero triage --fix              # auto-fix structural issues where safe
```

### Checks

#### 1. Duplicate Detection

Similarity check against all existing specs and knowledge entries:

- **Title similarity** — Levenshtein distance on normalized titles (lowercased, stopwords removed)
- **Tag overlap** — specs sharing 3+ tags from the same type are flagged
- **Content fingerprint** — a short bloom-filter-style hash of the first 200 words; near-identical content is flagged

Similarity thresholds (configurable in `hero.json`):
```json
{
  "triage": {
    "duplicate_threshold": 0.80,
    "tag_overlap_min": 3
  }
}
```

Output:
```
⚠  DUPLICATE CANDIDATE: conventions/go-code-style
   Similar to: conventions/go-style-guide (similarity: 0.87)
   Recommend: merge or differentiate
```

#### 2. Convention Conflict Detection

For new convention or rule specs, check for direct prescription conflicts with existing entries:

- Extract imperative sentences from the new spec body ("use X", "never do Y", "always Z")
- Match against extracted imperatives from existing specs of the same type
- Flag when the same subject has contradictory predicates

Example conflict:
```
⚠  CONFLICT: conventions/logging-zerolog prescribes "use zerolog for all logging"
   Conflicts with: conventions/stdlib-log which prescribes "use log/slog for structured logs"
   One of these must be updated or removed.
```

This extends the `knowledge-contradiction-detection` engine — same logic, applied at intake rather than on-demand.

#### 3. Orphaned Relations

Check that all `relations` frontmatter targets exist in the corpus:

```
⚠  ORPHANED RELATION: decisions/my-decision
   References target: conventions/nonexistent-pattern
   Target not found in corpus. Fix the slug or create the missing spec.
```

#### 4. Structural Validation

Enforce required frontmatter fields and valid enum values:

| Field | Required | Valid Values |
|---|---|---|
| `title` | yes | non-empty string |
| `type` | yes | `feature`, `convention`, `decision`, `context`, `rule` |
| `status` | yes | `planning`, `active`, `delivering`, `done`, `deprecated` |
| `created` | yes | ISO date |
| `milestone` | no | non-empty string if present |

Structural errors block import; all others are warnings.

### Output Modes

**Interactive (default):**
```
Triaging: conventions/logging-zerolog

  ✓ Structure valid
  ⚠ Duplicate candidate: conventions/stdlib-log (similarity: 0.82)
  ⚠ Convention conflict: contradicts conventions/stdlib-log on "logging library"
  ✓ No orphaned relations

2 warnings. Continue with import? [y/N]
```

**Non-interactive / CI mode (`--no-prompt`):**
```
hero triage --no-prompt conventions/logging-zerolog
```
Exits non-zero if any errors; exits with code 2 if warnings exist (configurable).

**JSON output (`--json`):**
```json
{
  "slug": "conventions/logging-zerolog",
  "passed": false,
  "errors": [],
  "warnings": [
    {
      "type": "duplicate_candidate",
      "target": "conventions/stdlib-log",
      "similarity": 0.82
    },
    {
      "type": "convention_conflict",
      "target": "conventions/stdlib-log",
      "subject": "logging library"
    }
  ]
}
```

### Auto-Fix (`--fix`)

Safe structural issues can be auto-fixed:
- Add missing `created` date (uses file mtime)
- Normalize `status` to closest valid value
- Strip invalid frontmatter keys

Duplicate and conflict warnings are never auto-fixed — they require human judgment.

## Changes

- `internal/triage/triage.go` — triage orchestrator, check runner
- `internal/triage/duplicate.go` — similarity scoring, bloom filter, tag overlap
- `internal/triage/conflict.go` — imperative extraction, conflict matching (extends contradiction engine)
- `internal/triage/structural.go` — frontmatter validation
- `internal/cli/triage.go` — `hero triage` command
- `internal/cli/add.go` — integrate triage into `hero add` flow
- `internal/cli/prime.go` — integrate triage into `hero prime` flow

## Acceptance Criteria

- Duplicate specs are flagged at import with similarity score and target slug
- Convention conflicts are detected and reported with the conflicting target
- Orphaned `relations` targets produce a warning with the missing slug
- Structural errors (missing required fields, invalid enums) block import
- `--fix` auto-corrects safe structural issues only
- `--no-prompt` mode exits non-zero on errors, code 2 on warnings
- `--json` output matches the schema above
- Triage runs automatically on `hero add` and `hero prime`
- Triage can be run standalone: `hero triage` and `hero triage <slug>`
- All checks are O(n) or better — no quadratic behavior on large corpora

## Boundaries

- Does **not** merge specs automatically — duplicate resolution is always manual
- Does **not** run LLM calls — all checks are deterministic text analysis
- Does **not** enforce arbitrary style rules beyond the defined structural schema
- Conflict detection applies only to `convention` and `rule` type specs, not `feature` or `decision`
