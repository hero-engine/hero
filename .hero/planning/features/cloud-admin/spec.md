---
title: Cloud Admin and Enterprise Controls
type: feature
status: planning
tags: [cloud, admin, enterprise, sso]
created: 2026-04-12
parent: hero-cloud
depends-on: [cloud-auth, cloud-billing]
horizon: next
smoke: deferred
---

## Goal

Provide org administration tools and Enterprise-tier features: SSO/SAML,
audit logging, compliance controls, and org-level policy management.

## Design

### Admin Features (Team + Enterprise)

- **Member management**: invite, remove, change roles
- **Repo management**: link/unlink repos, configure sync settings
- **Usage dashboard**: API usage, storage, sync frequency

### Enterprise-Only Features

#### SSO/SAML
- SAML 2.0 identity provider integration (Okta, Azure AD, OneLogin)
- SCIM provisioning for auto member sync from IdP
- Enforce SSO — disable password login for org members

#### Audit Log
- Immutable log of all org actions: member changes, spec access, sync events,
  setting changes, login/logout
- Searchable by actor, action type, date range
- Exportable as CSV/JSON for compliance
- 1-year retention (configurable)

#### Compliance Controls
- Data residency selection (US, EU) — controls where org data is stored
- IP allowlist — restrict API access to corporate networks
- Session policies — max session duration, forced re-auth interval

### Org Settings Page

```
Settings
├── General (name, slug, logo)
├── Members (invite, roles)
├── Repos (linked repos, sync config)
├── Billing (subscription, seats, invoices)
├── Notifications (Slack webhook, defaults)
├── Security (SSO, IP allowlist, session policy)  [Enterprise]
└── Audit Log                                     [Enterprise]
```

## Changes

- Cloud service: `admin/` package — org settings CRUD
- Cloud service: `audit/` package — audit log storage and query
- Cloud service: `sso/` package — SAML 2.0 service provider
- Cloud service: `scim/` package — SCIM provisioning endpoint
- Cloud dashboard: admin settings pages

## Acceptance Criteria

- Org admins can manage members, repos, and settings via dashboard
- Enterprise tier enables SSO/SAML with at least Okta and Azure AD
- Audit log captures all significant org actions
- Audit log is searchable and exportable
- IP allowlist enforced at API middleware level
- SCIM provisioning syncs members from IdP
- Non-enterprise orgs see upgrade prompts for enterprise features
