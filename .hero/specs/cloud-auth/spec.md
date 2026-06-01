---
title: Cloud Authentication and Org Management
slug: cloud-auth
type: feature
status: completed
tags: [cloud, auth, foundation]
created: 2026-04-12
parent: hero-cloud
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Provide authentication and organization/team management for Hero Cloud. Users
authenticate via email+password or OAuth (GitHub, Google), are organized into
orgs, and have role-based access to repos and specs.

## Design

### Auth Flow

1. `hero login` opens a browser to the Hero Cloud login page
2. User authenticates (email/password or OAuth)
3. Browser callback stores a token locally at `~/.hero/credentials.json`
4. CLI includes the token in subsequent cloud API calls
5. Token refreshes automatically; `hero logout` revokes it

### Org Model

```
Org
├── Members (role: owner | admin | member | viewer)
├── Teams (optional grouping)
│   └── Members
└── Repos (linked by push URL or manual config)
    └── Specs (synced from CLI)
```

### Token Format

JWT with org_id, user_id, roles. Short-lived access tokens (1h) with
long-lived refresh tokens (30d). Refresh tokens are rotated on use.

### Enterprise Extension Point

The auth layer is designed so that Enterprise tier can add SSO/SAML by
implementing an additional identity provider adapter without changing the
core auth flow.

## Changes

- Cloud service: `auth/` package — login, signup, OAuth callback, token issuance
- Cloud service: `org/` package — org CRUD, member management, role checks
- CLI: `hero login`, `hero logout` commands
- CLI: `~/.hero/credentials.json` token storage
- Cloud service: `middleware/auth.go` — JWT validation middleware

## Acceptance Criteria

- Users can sign up with email/password or GitHub OAuth
- Users can create orgs and invite members by email
- Members have roles (owner, admin, member, viewer)
- `hero login` opens browser, stores token, confirms auth
- `hero logout` revokes token and removes local credentials
- JWT tokens expire after 1 hour, refresh tokens after 30 days
- Invalid/expired tokens return 401, CLI prompts re-login
