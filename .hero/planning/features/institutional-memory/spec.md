---
title: "Institutional Memory"
type: feature
status: draft
priority: high
tags: [cloud, enterprise, billion-dollar, moat]
horizon: next
smoke: deferred
---

# Institutional Memory That Actually Works

## Problem
Senior engineers leave, knowledge walks out. Onboarding takes 6 months. Everyone re-discovers the same pitfalls. Confluence is write-only. Slack is a firehose.

## Proposed Solution
Hero observes engineering sessions across the org and builds proactive knowledge.

### What it learns:
- 15 developers hit the same bug in payments → auto-generate a convention
- Every time someone modifies billing pipeline, they break the same test → proactive warning
- Team London structures React components differently than Team Austin → surface the inconsistency, let them decide
- New developer touches auth module → "here are the 3 things everyone gets wrong here"

### Knowledge tiers:
1. **Explicit** (today): conventions, ADRs, specs committed to repo
2. **Observed** (new): patterns mined from session activity across the org
3. **Proactive** (new): relevant knowledge surfaced at the right moment, before the developer hits the problem

### Privacy model:
- No raw session content stored in cloud
- Only structured patterns extracted (which files cause issues, which conventions are followed/violated)
- Org controls what's captured via policy
- Self-hosted option for sensitive environments

### Why this is the moat:
Once a company has 2 years of engineering memory in Hero, they can't leave. The knowledge is the lock-in, not the tool.
