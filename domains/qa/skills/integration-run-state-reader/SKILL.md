---
name: integration-run-state-reader
description: Read optional external run history defensively and distinguish fresh, stale, partial, and unavailable evidence.
---
# Integration run-state reader

Check connector configuration, source identity, last successful synchronization,
pagination or truncation, environment, and selected history window. Preserve source
links and raw status while normalizing for analysis. Label stale, partial, or
unavailable history instead of treating it as failure or success. Cache only what
the local artifact needs for traceability. A connector error must not prevent local
plan, case, triage, or release-gate work.

