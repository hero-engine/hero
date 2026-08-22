---
description: Produce and independently review an evidence-backed release-readiness recommendation.
---
# Release gate

Route to `release-readiness-strategist`, then `release-gate-reviewer`. Resolve the
candidate release and blocker policy. Aggregate committed scope, critical
coverage, regression evidence, open issues, flakes, environment limitations, and
waivers. Report Go, Conditional Go, or No-Go with confidence and named blockers.
The release owner makes the final decision and records any waiver.

Request: $ARGUMENTS

