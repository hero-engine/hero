---
title: Tracker-ID-Shaped Search Falls Through to FTS, Returning Unrelated Specs
slug: bug-tracker-id-search-falls-through-to-fts
type: bug
status: completed
tags: [index, search, tracker-id, fts, relevance]
created: 2026-05-01
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Symptom

`TestSearchByTrackerID` in [internal/index/index_test.go:817](internal/index/index_test.go:817) fails:

```
Search('MORPH-999') returned 1 results, want 0
```

The test sets up two specs (one with `TrackerID: "MORPH-123"`, one without) and searches for `MORPH-999` — a tracker-ID-shaped query that matches nothing. It expects 0 results because no indexed spec has that tracker ID. Instead, the FTS fallback matches the spec whose slug contains "morph".

## Root Cause

`Search()` calls `looksLikeTrackerID(query)`, which returns `true` for `MORPH-999`. The tracker-ID lookup runs and finds no match. Rather than returning empty, the search falls through to FTS, which decomposes `MORPH-999` into the tokens `"morph"` and `"999"` (via `SanitizeFTSQuery`) and matches the `morph-123-fix-login` spec because its slug, title, and body contain "morph".

The intent encoded in the test is: a tracker-ID-shaped query that doesn't match any indexed tracker should return zero results, not a fuzzy text match. This avoids surprising behavior where a typo in a ticket ID returns unrelated specs whose names happen to share a prefix.

## Suggested Fix Approach

In [internal/index/index.go](internal/index/index.go), update `Search()` so that when `looksLikeTrackerID(query)` is true and the tracker-ID lookup returns empty, the function returns `nil, nil` (or `[]SearchResult{}, nil`) without falling through to FTS.

Pseudocode for the change:

```go
if looksLikeTrackerID(query) {
    rs, err := idx.searchByTrackerID(query)
    if err != nil {
        return nil, err
    }
    if len(rs) > 0 {
        return rs, nil
    }
    // Tracker-ID-shaped query with no match — return empty rather
    // than a noisy FTS fallback that just matches partial tokens.
    return nil, nil
}
// ... existing FTS path for non-tracker-ID queries
```

Confirm that the existing test `Search('morph-123')` (line 808) still passes (case-insensitive match), and that the new behavior doesn't regress non-tracker-ID FTS queries.

## Acceptance Criteria

- WHEN `Search` is called with a tracker-ID-shaped query THAT matches an indexed `tracker_id` THE SYSTEM SHALL return matching specs
- WHEN `Search` is called with a tracker-ID-shaped query THAT does not match any indexed `tracker_id` THE SYSTEM SHALL return zero results (no FTS fallback)
- WHEN `Search` is called with a free-text query THE SYSTEM SHALL use the existing FTS path unchanged
- WHEN `TestSearchByTrackerID` runs THE SYSTEM SHALL pass all three assertions (positive match, case-insensitive match, non-match returns 0)

## Out of Scope

- Changing `looksLikeTrackerID`'s heuristic
- Cross-indexing tracker IDs into FTS so they're discoverable both ways
- Other search relevance tuning
