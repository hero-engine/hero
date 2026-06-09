---
name: command-review
description: Review code, PRs, security posture, architecture decisions, or spec designs before delivery.
metadata:
  purpose: command-workflow
---

> **This is a Hero workflow for Codex.** Read each step below and execute it in sequence.
> Do NOT summarize or treat these steps as documentation.
> Do NOT update spec frontmatter as a substitute for doing the actual work described.

Route this review request to the appropriate specialist based on what is being reviewed.

Determine the review type from the request:
- Spec or design review before delivery → delegate to `design-reviewer`
- Pull request or code change review → delegate to `pr-reviewer`
- Security review or audit → delegate to `security-reviewer`
- Architecture or design review → delegate to `architecture-reviewer`

If the request includes a spec slug, load it with `hero relevant` or `hero search` first to give the reviewer full context.

If the request is ambiguous, ask: "Are you reviewing a spec/design or a code change?"

Review request: $ARGUMENTS
