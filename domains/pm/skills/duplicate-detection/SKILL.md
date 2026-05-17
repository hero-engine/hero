---
name: duplicate-detection
description: Detect near-duplicate PM artifacts using overlap signals beyond lexical similarity — theme, segment, source-cluster, and cross-domain overlap. Never auto-merge.
compatibility: opencode
metadata:
  audience: duplicate-detector, intake-triager, pm-reviewer
  purpose: intake-curation
---

## What I do

Provide the detection logic for finding near-duplicate and related PM artifacts (intakes, initiatives, stories) — and the rules for what to do with the candidates once found. Lexical similarity is the cheap first pass; the verdict requires looking at theme overlap, segment overlap, source-cluster overlap, and cross-domain overlap (the engineering feature may already exist).

The output is a ranked list of candidates with a confidence score and the **specific field overlap** that triggered each match. The merge decision is always human-confirmed.

## When to use me

Load this skill when:

- `duplicate-detector` runs at write-time of a new `intake` (the default trigger)
- `intake-triager` is comparing a triaged item against existing initiatives
- `pm-reviewer` is auditing for accumulated dupes during a `/scrub intake`
- a PM searches a theme and wants to know "is this already on the roadmap?"

## The overlap signals

### Lexical similarity (the starting point, not the verdict)

Token overlap, content shingles, or embedding cosine similarity over title + body. Cheap to compute; runs on every write.

**Why it's only a starting point:**

- Two intakes can use entirely different vocabulary for the same need ("export to spreadsheet" vs "csv download"). Lexical similarity misses this.
- Two intakes can share heavy vocabulary by accident ("user profile" appears in dozens of unrelated asks).
- Lexical match across customer segments may be the same need or two different needs with the same surface ask. The lexical layer cannot tell.

Use lexical similarity to **filter candidates**, not to **declare matches**. A 0.85 cosine similarity is a candidate; the verdict requires more signals.

### Theme overlap

If two items share themes (per the `intake-classification` skill), and the themes are *specific* (not generic catchalls like "performance"), that is strong corroborating signal. Two intakes both tagged `csv-export` + `enterprise` are far more likely duplicates than two items both tagged `general-improvement`.

### Segment overlap

Same theme + same segment = stronger merge candidate. Different segments asking for the same theme = often *related* but not *duplicate* — enterprise SSO and prosumer Google-login are theme-adjacent but solve different jobs.

### Source-cluster overlap

If two intakes both cite the same customer, same support ticket cluster, or same sales call recording, they are almost certainly the same signal captured twice. Source-cluster overlap is the highest-precision signal — the underlying source is literally the same.

When `intake-triager` runs daily on the `new` queue, source-cluster overlap often catches the "support filed three tickets from one customer" pattern.

### Cross-domain overlap

**An engineering-owned spec might already exist.** A PM intake asking for "rate limiting on the api" may overlap an existing `spec` already engineering-owned and `delivering`. Under the unified type model the type is the same (`spec`); the boundary is the `owner` field. Without cross-owner detection, the PM creates a new initiative that duplicates committed engineering work.

Use the `cross-domain-graph-query` skill to search `spec` artifacts across owner values when classifying a high-confidence theme. A cross-owner match is a different kind of finding — it is not "merge with engineering"; it is "link the intake to the existing spec and update its evidence section." The PM-side intake still exists; the resolution is the link, not the merge.

## Near-duplicate vs related

Two distinct decisions:

- **Near-duplicate** — same need captured twice. Merge into the canonical item; the merged item sets `merged_into` and stays accessible (never deleted). Source attribution from both flows into the survivor.
- **Related** — different needs that share theme, customer, or initiative. Link via `linked_intake` on the parent initiative or via cross-references in Notes. Do not merge — the distinct signals matter for downstream prioritization.

**The rule:** if the two items would generate *exactly the same downstream action* (same story, same engineering work), they are duplicates. If they would generate *different actions even on the same theme*, they are related.

## Confidence scoring and field overlap

Every duplicate candidate is reported with:

1. **A confidence score** (0.0 - 1.0) combining lexical, theme, segment, and source signals.
2. **The specific field overlap** that triggered the match. Examples:
   - "Title cosine 0.87 + shared theme `csv-export` + same customer `acme-corp`"
   - "Body shingle overlap 0.72 + shared themes [`sso`, `enterprise`] + adjacent support tickets from same account"
   - "Cross-domain: engineering `feature` spec `rate-limit-api` in `delivering` matches theme + segment"

The field overlap is what the human needs to make the call. An opaque score ("confidence 0.83") is useless; "shared customer + shared theme" is actionable.

## Never auto-merge — recall over precision

**The rule.** Duplicate detection surfaces candidates; humans confirm. This is non-negotiable.

Why:

- A false-positive auto-merge destroys source attribution and loses signal. The merged-away item cannot be recovered without a graph repair.
- A false-negative ("we missed a dupe") is recoverable — the next sweep catches it.
- The asymmetry means **recall matters more than precision**. Surface aggressively; let the human decide.

`duplicate-detector` proposes; `intake-triager` (or a PM) accepts. The merge is a deliberate gesture, not a background event.

## How `duplicate-detector` runs at write-time

The standard sequence on `intake` creation:

1. New intake lands in `new` state.
2. `duplicate-detector` runs synchronously before the item is fully persisted.
3. Lexical pass produces top-N candidates.
4. Each candidate is enriched with theme, segment, source-cluster, and cross-domain signals.
5. Candidates above a confidence threshold are written to the new item's Investigation section as "possible duplicates" with their field overlap.
6. `intake-triager` picks up the item in the normal queue; the candidates are visible before triage decides.

If the top candidate has very high confidence (e.g. source-cluster match), `duplicate-detector` may annotate the item as `suggested_merge: <slug>` — but the actual merge waits on human confirmation.

## Anti-patterns

- **Reporting opaque similarity scores.** "Confidence 0.83" alone is useless. Always include the field overlap.
- **Auto-merging on high-confidence matches.** Recall over precision; humans confirm.
- **Treating lexical similarity as the verdict.** It's a filter, not an answer.
- **Ignoring cross-domain duplicates.** The engineering feature might already exist. Skipping the cross-domain search creates roadmap items that duplicate committed engineering work.
- **Conflating near-duplicate with related.** Related items link; near-duplicates merge. Confusing them destroys distinct signals.
- **Destroying source attribution on merge.** Both items' source attribution flows into the survivor. The merged-away spec stays accessible via `merged_into`.
- **Running duplicate detection only at intake write-time.** Background sweeps catch dupes that accumulate as themes evolve. Both passes are needed.
