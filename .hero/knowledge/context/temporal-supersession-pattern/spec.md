---
title: Temporal Supersession Pattern — Detecting Stale Facts at Read Time
type: context
status: active
created: 2026-05-06
tags: [architecture, retrieval, temporal, episodic-memory]
---

## Pattern

When a memory/knowledge system returns a stored fact, check at retrieval time
whether a newer fact on the same topic exists that contradicts or supersedes
it. Attach a staleness warning rather than silently returning outdated content.

## Origin

Observed in `f00stx/episodic-memory` (MIT, Python, 2026-05), which implements
a `ContradictionDetector` that checks cosine similarity >= 0.75 between
recalled episode summaries and newer episodes, flagging temporal supersession
when an age gap >= 1 day exists.

## Key Design Decisions

1. **Read-time, not write-time** — Write-time validation (like Hero's
   `FindGraphConflicts`) catches concurrent edits. Read-time detection
   catches a different class of problem: stale reads where the data changed
   between index updates or between sessions.

2. **Warnings, not blocks** — Contradictions are advisory. The consumer
   decides whether to act on them. This avoids false-positive fatigue.

3. **BM25 before vectors** — Topical overlap detection via FTS5 BM25 is
   sufficient for same-type same-topic supersession without requiring
   embedding infrastructure. Vector similarity is a Phase C enhancement.

4. **Budget caps** — Topical overlap queries are O(N) per result; capping
   N (default 5) keeps retrieval latency predictable.

## Applicability to Hero

Hero's bitemporal graph already stores the temporal history needed for
signals 1-3 (explicit supersession edges, revised rows, concurrent edits).
FTS5 provides signal 4 (topical overlap). Signal 5 (index/graph status
mismatch) is a consistency check unique to Hero's dual-substrate architecture.

See spec: `retrieval-contradiction-detection`.
