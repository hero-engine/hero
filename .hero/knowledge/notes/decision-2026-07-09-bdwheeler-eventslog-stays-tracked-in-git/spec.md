---
title: DECISION (2026-07-09, bdwheeler): events.log stays TRACKED in git (cst-gitign...
slug: decision-2026-07-09-bdwheeler-eventslog-stays-tracked-in-git
type: note
created: 2026-07-09
tags: []
---
# DECISION (2026-07-09, bdwheeler): events.log stays TRACKED in git (cst-gitign...

DECISION (2026-07-09, bdwheeler): events.log stays TRACKED in git (cst-gitignore-events-log open question resolved). Rationale: it is the durable backing store for hero feed / hero velocity / activity clusters across machines and clones; making it per-machine gitignored would lose cross-machine activity + velocity history. The working-tree churn from state-changing commands is accepted as normal for a tracked ledger. No code change — confirms existing behavior. Revisit only if multi-machine/team-mode usage makes per-machine the clear win.
