---
title: "Multi-Domain Core Engine"
slug: multi-domain-core
type: feature
status: draft
priority: low
horizon: next
smoke: deferred
created: 2026-05-12
---

# Multi-Domain Core Engine

## Problem
The spec-driven pattern (structured input → AI execution → human review → tracked output) generalizes beyond code to legal, compliance, research, marketing, and other domains. But diluting Hero into a generic tool would kill what makes it good.

## Proposed Solution
Separate products sharing a core engine:
- Core: spec management, knowledge accumulation, tracker integration, quality scoring, async execution
- Hero (engineering): code-aware agents, CI/CD integration, PR workflows
- Future products: domain-specific agents, commands, conventions on the same core

### Domains with highest AI upside:
- Legal document review and contract analysis
- Compliance workflows (SOC2, HIPAA, GDPR audit prep)
- Research synthesis (literature review → structured findings)
- Infrastructure/DevOps runbooks
- Data pipeline design and validation

### Key insight:
Each domain needs its own "code scan" equivalent — a way to understand the existing artifacts (contracts, policies, papers, pipelines) and build structured knowledge. The pattern is the same, the parsers are different.

### Status: futures list. Focus on engineering first, validate the core, then explore.
