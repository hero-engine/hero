---
title: "HTML Reports — Self-Contained Project Report Generation"
slug: html-report
type: feature
status: completed
tags: [cli, reporting]
created: 2026-04-12
horizon: now
---

## Goal

The `hero report` command generates a self-contained HTML report summarizing project health, spec status, and engineering velocity — viewable in any browser with no external dependencies.

## Design

The report is rendered as a single HTML file with inline CSS and JavaScript. It includes:

- **Status counts** — breakdown of specs by status (draft, in-progress, completed, blocked, etc.)
- **Velocity chart** — specs completed over time, rendered as an inline SVG chart
- **Coverage bars** — visual indicators showing how well specs cover the codebase
- **Stale warnings** — highlights specs that haven't been updated within a configurable threshold
- **In-flight table** — sortable table of all in-progress specs with assignee, age, and last activity

The HTML is fully self-contained — no CDN links, no external assets. The file can be emailed, committed, or served statically.

## Changes

- `internal/cli/report.go` — `hero report` command implementation, HTML template rendering
- `internal/cli/report_test.go` — tests for report generation, template rendering, data aggregation

## Acceptance Criteria

- `hero report` produces a valid HTML file that opens in any modern browser
- Report includes status counts, velocity chart, coverage bars, stale warnings, and in-flight table
- HTML file is fully self-contained with no external dependencies
- Stale threshold is configurable
- Tests cover data aggregation and template rendering
