---
title: "Architectural Drift Detection"
type: feature
status: draft
priority: high
tags: [cloud, enterprise, billion-dollar]
horizon: next
smoke: deferred
---

# Architectural Drift Detection — Continuous Architecture Compliance

## Problem
Architecture diagrams rot in Confluence. Dependency rules exist in people's heads. AI generates code faster, so architectural drift accelerates faster. Existing tools (ArchUnit, etc.) check imports, not intent.

## Proposed Solution
Architects define constraints as specs. Hero continuously validates the entire codebase against them.

### Constraint types:
- Dependency rules: "service A must never depend on service B"
- Layer enforcement: "all database access goes through the repository layer"
- Domain boundaries: "no direct HTTP calls from domain logic"
- Pattern mandates: "all API endpoints use the middleware chain defined in convention-api-patterns"
- Security boundaries: "no PII logging outside the audit module"

### How it works:
- Constraints stored as specs in .hero/knowledge/architecture/
- Code scan provides the structural understanding
- Continuous validation: not per-PR, across the whole org
- Violations caught BEFORE the PR is opened (during /deliver)
- Cloud dashboard shows drift over time across all repos

### Massive effort:
Requires deep semantic understanding of code, not just regex. But it's unsolved and every large org needs it.
