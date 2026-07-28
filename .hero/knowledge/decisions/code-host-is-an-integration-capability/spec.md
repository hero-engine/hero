---
title: "Code hosting is an integration capability, not a tracker subtype"
slug: code-host-is-an-integration-capability
type: decision
status: accepted
domain: engineering
created: 2026-07-27
relates-to:
  - hero-code-host-broker-capabilities
  - integration-config-uses-stable-ids
tags: [integrations, code-host, tracker, github, gitlab]
---

# Code hosting is an integration capability, not a tracker subtype

## Context

Hero historically introduced external work systems through tracker-oriented
providers and roles. Pull-request lifecycle work adds repository hosting, but
provider names do not map one-to-one to product capabilities: GitHub and GitLab
can provide both issue tracking and code hosting, while Jira and Linear provide
tracking without repositories.

A workspace may deliberately track delivery in Jira or Linear while hosting
pull requests on GitHub. Another workspace may use one GitHub connection and
credential for both.

## Decision

Model `tracker`, `code-host`, and `docs` as semantic capabilities of stable
integration connections.

- Provider metadata declares the capabilities a provider kind is eligible to
  serve.
- Each connection declares the subset it actually serves.
- Roles select connections by required capability; provider kind alone never
  satisfies a role.
- One stable connection and credential may serve multiple compatible roles.
- Tracker and code-host brokers remain separate domain contracts even when they
  resolve the same connection.
- Existing omitted capability declarations infer only their legacy role so a
  GitHub tracker does not silently become a code host.

## Consequences

GitHub and GitLab may be configured as tracker-only, code-host-only, or both.
Jira and Linear may be trackers but cannot satisfy `roles.code-host`.
Confluence remains docs-only. Runtime operation discovery remains separate from
static provider eligibility, so a valid GitLab code-host connection may report
no implemented PR operations until a GitLab adapter ships.

Hero reuses `integrations.connections`, layered stable-ID configuration,
`config.Secret`, and existing credential precedence. It does not create a
second code-host credential system or expand issue-tracker APIs into PR
lifecycle APIs.

## Rejected alternatives

- **Treat every GitHub tracker as a code host:** surprising privilege expansion
  and ambiguous selection for existing workspaces.
- **Make code host a tracker subtype:** cannot represent Jira-tracker plus
  GitHub-host and leaks issue semantics into repository operations.
- **Create separate code-host connections and credentials:** duplicates stable
  identity, local overlays, redaction, and credential resolution.
- **Infer code host from provider or ambient CLI authentication:** non-
  deterministic and bypasses workspace selection and credential policy.
