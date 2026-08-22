---
name: blocker-policy-evaluation
description: Evaluate defects, gaps, flakes, and operational concerns against an explicit release-blocker policy.
---
# Blocker policy evaluation

Load the local blocker policy and classify each issue by severity, affected scope,
reproducibility, workaround, detectability, user impact, and recovery. Apply
defaults only when no project policy exists, and label that assumption. Every
override requires an accountable owner, rationale, expiry or follow-up, and the
specific risk being accepted. Do not downgrade an issue merely to obtain a Go
recommendation.

