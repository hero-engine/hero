---
title: handoff-over-engineering NEXT/handoff subsystem got over-built: ~18 moving pa...
slug: handoff-over-engineering-nexthandoff-subsystem-got-over-buil
type: note
created: 2026-06-03
tags: []
---
# handoff-over-engineering NEXT/handoff subsystem got over-built: ~18 moving pa...

handoff-over-engineering NEXT/handoff subsystem got over-built: ~18 moving parts, 9 files, most graph-derived caches no code reads back. The two felt pains (drift, 'not in my commit') have one root cause each — no auto-emit (projections re-render stale graph) and staging in only one of two hook installers. Both are pure re-wires of existing code. Only .hero/next/<user>.md is load-bearing for code (cross-machine federation via ingest). SNAPSHOT/QUEUE are accretions on the projection framework, justified independently, but their committed files are read by no Go code. Lesson: handoff is 'persist at end of turn, load at start, travel via git' — keep it to one persist/one load/one stage and 2 files. Collapse toward graph-as-truth + on-demand commands, not committed projections.
