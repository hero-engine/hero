---
title: Cloud Dashboard — Web UI for Team Visibility
type: feature
status: planning
tags: [cloud, dashboard, web]
created: 2026-04-12
parent: hero-cloud
depends-on: [cloud-api, cloud-sync]
horizon: next
smoke: deferred
---

## Goal

A web application that gives teams a real-time view of spec status across all
repos in their org. The dashboard is the primary value proposition for the Team
tier — it answers "what's happening across our projects?" without requiring
everyone to run CLI commands.

## Design

### Views

1. **Org Overview**
   - Total specs across all repos, by status
   - In-flight work summary (who's working on what)
   - Recent activity feed (syncs, status changes, new specs)
   - Health indicators (stale specs, unclaimed work)

2. **Repo View**
   - Spec list with filtering/sorting (status, type, owner, tags)
   - Spec detail with full body (if synced with --full)
   - Dependency graph (rendered from spec relations)
   - Coverage metrics (tracker linked, changes section filled, etc.)

3. **Team View**
   - Work distribution by team member
   - Velocity chart (completed specs over time)
   - Workload balance indicators

4. **Search**
   - Cross-repo search over spec metadata (and body if --full synced)
   - Filter by repo, type, status, tags, owner
   - Results ranked by relevance

### Tech Stack

- **Frontend**: Static SPA (React or Svelte, TBD — lean toward Svelte for bundle size)
- **Backend**: Cloud API (already specified in cloud-api)
- **Auth**: JWT from cloud-auth, cookie-based for web sessions
- **Deploy**: Container on fly.io or Railway initially

### Design Principles

- Read-only — the dashboard never modifies specs. Specs are authored in the CLI/AI tool.
- Mobile-responsive — team leads check status on phones
- Dark mode — developers live here
- Fast — no spinner for initial load. Server-render the overview, hydrate for interactivity.

## Changes

- Cloud service: `web/` — SPA frontend
- Cloud service: `api/v1/dashboard/` — dashboard-specific aggregation endpoints
- Cloud service: deploy configuration

## Acceptance Criteria

- Org overview shows real-time spec counts and in-flight work
- Repo view supports filtering and sorting specs
- Cross-repo search works with metadata (and body when available)
- Dependency graph renders for specs with relations
- Activity feed shows recent syncs and status changes
- Page loads in under 2 seconds for orgs with 500+ specs
- Mobile-responsive layout
- Dark mode support
