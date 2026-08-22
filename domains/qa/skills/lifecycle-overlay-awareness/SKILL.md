---
name: lifecycle-overlay-awareness
description: Handle QA-added workflow states as optional overlays while preserving artifacts when another pack is absent.
---
# Lifecycle overlay awareness

Read the active pack composition before proposing a state transition. When QA
overlay states are available, validate entry conditions, permitted transitions,
and accountable owner. When they are absent, render stored foreign states as
historical labels and use a portable finding or handoff instead of forcing a
transition. Never redefine the owning pack's base type or silently map an unknown
state to a different one.

