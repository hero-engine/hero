---
title: Team OAuth — GitHub/Google SSO for Team Server Authentication
type: feature
status: planning
priority: P1
tags: [team, auth, oauth, github, google, sso]
created: 2026-04-25
relations:
  - target: hero-team-server
    kind: parent
horizon: next
smoke: deferred
---

## Goal

Replace the shared-token auth tier with OAuth so team members authenticate
with their existing GitHub or Google identity. The team server issues JWTs
after OAuth callback, and all subsequent API calls use the JWT. User
identity flows through to job attribution and usage tracking.

## Problem

Token auth works but has no user identity — everyone shares one token.
The team server tracks "submitted_by" but has no way to know WHO unless
the client self-reports. OAuth ties jobs and usage to real identities,
enables per-user budgets, and lets orgs restrict access to their members.

## Design

### Auth flow

```
hero connect team https://hero.internal:7437
  → opens browser to https://hero.internal:7437/auth/login?provider=github
  → redirects to GitHub OAuth consent
  → callback to /auth/callback with code
  → server exchanges code for GitHub user info
  → server issues JWT (30-day expiry, signed with server secret)
  → CLI receives JWT via localhost callback, stores in ~/.hero/credentials
  → all future requests include Authorization: Bearer <jwt>
```

### Configuration

```json
{
  "serve": {
    "team": {
      "auth": "github-oauth",
      "oauth": {
        "provider": "github",
        "client_id": "${HERO_OAUTH_CLIENT_ID}",
        "client_secret": "${HERO_OAUTH_CLIENT_SECRET}",
        "org": "your-github-org",
        "allowed_teams": ["engineering"]
      },
      "jwt_secret": "${HERO_JWT_SECRET}"
    }
  }
}
```

### Providers

| Provider | Scope | User info endpoint | Org restriction |
|---|---|---|---|
| GitHub | `read:user, read:org` | `/user` + `/user/orgs` | Filter by org membership |
| Google | `openid email profile` | `/oauth2/v3/userinfo` | Filter by hosted domain |

### Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/auth/login` | Redirect to OAuth provider |
| GET | `/auth/callback` | Handle OAuth callback, issue JWT |
| GET | `/auth/me` | Return current user info from JWT |
| POST | `/auth/refresh` | Refresh an expiring JWT |

### JWT payload

```json
{
  "sub": "github:alice",
  "name": "Alice Smith",
  "email": "alice@example.com",
  "org": "your-github-org",
  "iat": 1745000000,
  "exp": 1747592000
}
```

### Auth middleware update

The existing `TokenAuthMiddleware` gains a sibling `JWTAuthMiddleware`
that validates the JWT signature and expiry, extracts the user identity,
and sets `X-Hero-User` on the request for downstream handlers.

### hero connect team

```bash
hero connect team https://hero.internal:7437
# Opens browser for OAuth
# On success: "Connected as alice@example.com (your-github-org)"
# Stores JWT in ~/.hero/credentials
```

The CLI starts a temporary localhost HTTP server to receive the JWT
callback (same pattern as `gh auth login`).

## Changes

- `internal/serve/oauth.go` — OAuth flow (GitHub + Google), JWT issuance
- `internal/serve/auth.go` — add JWTAuthMiddleware alongside existing token auth
- `internal/serve/api_auth.go` — /auth/login, /auth/callback, /auth/me endpoints
- `internal/cli/connect_team.go` — `hero connect team <url>` with browser OAuth
- `internal/config/credentials.go` — store/load JWT tokens
- `internal/config/config.go` — OAuthConfig struct

## Acceptance Criteria

- WHEN `hero connect team <url>` is called THE SYSTEM SHALL open a browser to the OAuth login page and store the received JWT locally
- WHEN a valid JWT is included in the Authorization header THE SYSTEM SHALL extract the user identity and allow the request
- WHEN an expired JWT is included THE SYSTEM SHALL return 401 with a message to reconnect
- WHEN `auth: "github-oauth"` is configured with an `org` restriction THE SYSTEM SHALL reject users who are not members of that org
- WHEN a job is submitted with a valid JWT THE SYSTEM SHALL set `submitted_by` to the authenticated user's identity
- WHEN `/auth/me` is called with a valid JWT THE SYSTEM SHALL return the user's name, email, and org
- THE SYSTEM SHALL fall back to token auth if OAuth is not configured

## Boundaries

- Does **not** support SAML/OIDC (enterprise SSO) — that's a cloud feature
- Does **not** manage user roles or permissions — all authenticated users have equal access
- Does **not** store user profiles on the server — identity comes from the JWT
- Does **not** support multiple OAuth providers simultaneously — one per server
