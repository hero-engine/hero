---
title: Integration Configuration Uses Stable IDs and Layered Selection
type: decision
status: accepted
tags: [config, integrations, credentials, architecture]
created: 2026-07-15
---

## Decision

Hero integrations are keyed by stable user-defined IDs, with provider type stored on each entry and `default`/semantic roles selecting entries. `.hero/hero.json` and `.hero/hero.local.json` use the same schema; local data overlays committed data by ID, while layer validation forbids literal secrets in committed config.

## Context

The singular `tracker.type` model made a natural Jira-keyed local token disappear silently and cannot cleanly represent Jira delivery plus a future roadmap provider or two integrations using the same provider. Stable IDs separate identity from provider type, while explicit selection prevents ambiguous map-order behavior. A shared layered shape lets teams commit non-secret settings and individuals supply credentials or an entirely local connection.

## Alternatives Considered

Provider-keyed singleton objects were rejected because they cannot represent two projects or identities for one provider. Keeping the flat tracker object plus one-off Aha/Confluence sections was rejected because it perpetuates incompatible schemas. Automatic load-time rewriting was rejected because it creates surprising diffs and weakens rollback.
