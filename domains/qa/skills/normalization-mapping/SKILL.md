---
name: normalization-mapping
description: Map optional external test-management fields into a local evidence shape while preserving source identity and raw decoration.
---
# Normalization mapping

Map source identifier, link, title, status, severity, priority, assignee, age,
environment, and last evidence into canonical fields. Preserve raw source fields
under source-specific decoration and never write normalized assumptions back over
authoritative values. Record mapping version and synchronization time. Unknown or
unmapped fields remain visible. Local artifacts must remain useful when no source
record or connector exists.

