---
title: Cloud REST API
type: feature
status: completed
tags: [cloud, api, foundation]
created: 2026-04-12
parent: hero-cloud
depends-on: cloud-auth
horizon: now
---

## Goal

Expose a REST API that the Hero CLI and Cloud Dashboard consume to read and
write spec data, org settings, and activity events. This is the central data
plane for all cloud features.

## Design

### API Structure

```
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
DELETE /api/v1/auth/logout

GET    /api/v1/orgs
POST   /api/v1/orgs
GET    /api/v1/orgs/:org_id
PUT    /api/v1/orgs/:org_id
GET    /api/v1/orgs/:org_id/members

GET    /api/v1/orgs/:org_id/repos
POST   /api/v1/orgs/:org_id/repos
GET    /api/v1/orgs/:org_id/repos/:repo_id

POST   /api/v1/orgs/:org_id/repos/:repo_id/sync    (push specs)
GET    /api/v1/orgs/:org_id/repos/:repo_id/specs
GET    /api/v1/orgs/:org_id/repos/:repo_id/specs/:slug

GET    /api/v1/orgs/:org_id/search?q=...            (cross-repo search)
GET    /api/v1/orgs/:org_id/activity                (event feed)
```

### Data Model (Postgres)

```sql
orgs            (id, name, slug, created_at)
users           (id, email, name, avatar_url, created_at)
org_members     (org_id, user_id, role, joined_at)
repos           (id, org_id, name, push_url, last_sync_at)
specs           (id, repo_id, slug, title, type, status, claimed_by,
                 tracker_id, tags, sections_json, files_touched,
                 created_at, modified_at, synced_at)
activity_events (id, org_id, repo_id, user_id, event_type,
                 payload_json, created_at)
```

### Per-Org Isolation

All queries are scoped to org_id. The middleware injects org context
from the JWT. No cross-org data leakage is possible at the query layer.

### Rate Limiting

- Authenticated: 1000 req/min per user
- Sync endpoint: 60 req/min per repo (prevent rapid-fire pushes)
- Search: 120 req/min per user

## Changes

- Cloud service: `api/v1/` package — route handlers
- Cloud service: `store/` package — Postgres data access layer
- Cloud service: `migrations/` — schema migrations
- Cloud service: `middleware/` — auth, rate limiting, org scoping

## Acceptance Criteria

- All endpoints require valid JWT (except login/signup)
- Spec data is stored in Postgres with org-level isolation
- Cross-repo search returns results from all repos in the org
- Activity feed captures sync events, spec status changes, member actions
- Rate limits enforced and return 429 with Retry-After header
- API versioned at /api/v1/, backward-compatible changes only within version
