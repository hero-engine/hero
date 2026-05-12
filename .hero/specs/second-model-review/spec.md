---
title: Second Model Review — Automated Design Critique via Independent Model
type: feature
status: completed
milestone: v0.2
tags: [ai-models, design-review, quality, automation, review]
created: 2026-04-12
relations:
  - target: model-role-config
    kind: depends-on
horizon: now
---

## Goal

When an agent produces a design proposal, spec, or significant architectural decision, automatically route that output to a second model for independent review before the team proceeds. The second model critiques the approach, surfaces blind spots, identifies risks, and proposes alternatives — without having participated in the original reasoning chain.

This is an automated "second set of eyes" quality gate. It mirrors how engineering teams use design reviews: the author is too close to the work; a reviewer with fresh eyes catches what the author misses. The same principle applies to AI-generated designs.

## Context

This feature is sourced from the `second-model-design-review` knowledge note. The key insight there: models have confirmation bias toward their own reasoning. A second model that hasn't been anchored by the first model's framing catches what the first rationalized away. Hero already has `architecture-reviewer`, `security-reviewer`, and other specialized reviewer agents — this spec provides the triggering and routing infrastructure to use them automatically.

## Design

### Command Surface

```bash
# Explicit design review — review a spec
hero review --spec sprint-from-tracker

# Review content from a file
hero review --file path/to/design.md

# Review the last /design output (reads from spec in-progress)
hero review --last-design

# With a specific reviewer agent
hero review --spec sprint-from-tracker --reviewer architecture-reviewer

# Flag on design command to trigger review automatically
hero design --with-review sprint-from-tracker
```

The review output is either:
1. Printed to stdout (interactive mode)
2. Appended to the spec as a `## Review Notes` section (with `--append`)
3. Written to a separate `review.md` file in the spec directory (with `--save`)

### Review Flow

```
1. hero review --spec <slug>
   ↓
2. Load spec content + relevant project context
   (conventions, decisions, related specs — same as hero context)
   ↓
3. Route to reviewer model (resolved via model-role-config: role=review)
   ↓
4. Reviewer receives:
   - The proposed design/spec
   - Project context (conventions, active decisions)
   - Review prompt (see below)
   ↓
5. Review output produced:
   - Critique of the approach
   - Identified risks and blind spots
   - Missing considerations
   - Alternative approaches worth considering
   ↓
6. Output surfaced to user (stdout, appended to spec, or saved)
```

### Review Prompt Template

The reviewer model is invoked with a structured prompt:

```
You are performing an independent design review. You have NOT participated in producing this design. Your job is to provide honest, critical feedback.

Review the following design critically:

---
[SPEC CONTENT]
---

Project context (conventions and decisions relevant to this spec):

---
[CONTEXT]
---

Provide:
1. **Risk assessment**: What could go wrong? What assumptions are risky?
2. **Blind spots**: What has the author not considered?
3. **Missing edge cases**: What scenarios aren't addressed?
4. **Alternatives**: Are there simpler or better approaches?
5. **Questions**: What needs clarification before implementation?

Be direct. Do not validate the approach — find the problems.
```

The prompt is stored in `agents/design-reviewer.md` and can be customized by the team like any other Hero agent.

### Integration with `/design`

The `/design` agent command gets an optional review step at the end of its flow:

```
# In commands/design.md
After generating the spec, if the user requested --with-review or if
auto_review is enabled in hero.json, invoke:
  hero review --spec <slug> --append
```

The review notes are appended to the generated spec as:

```markdown
## Review Notes

*Auto-generated review by [model-name] on [date]. Not a human review.*

### Risks
...

### Blind Spots
...

### Questions Before Implementation
...
```

### Auto-Review Configuration

```json
{
  "review": {
    "auto_review_designs": false,
    "reviewer_agent": "design-reviewer",
    "append_to_spec": true,
    "complexity_threshold": null
  }
}
```

`complexity_threshold`: if set (e.g. `"large"`), only auto-review specs above a certain size/complexity. Values: `"any"`, `"medium"`, `"large"`. Complexity is estimated by spec body word count + number of relations.

### Reviewer Agent Definition

```markdown
---
name: design-reviewer
role: review
description: Independent design reviewer. Critiques specs for risks, blind spots, and missing considerations.
---

You are an independent design reviewer...
[full reviewer prompt]
```

The `role: review` frontmatter tag means `model-role-config` will route this agent to the `review` model — ideally a different provider than the design model to avoid correlated blind spots.

### Multi-Round Review (Optional)

When `--dialogue` is passed, the primary model sees the review and responds:

```
1. Primary model produces design
2. Reviewer critiques
3. Primary model responds to critique, updates design
4. (Optional) Reviewer does final pass
```

This is a multi-turn design dialogue. Useful for complex architectural decisions. Off by default — the single-pass review is the standard path.

## Changes

- `commands/review.md` — `/review` command definition invoking `design-reviewer` agent
- `agents/design-reviewer.md` — new reviewer agent with `role: review`
- `internal/cli/review.go` — `hero review` command with spec/file/last-design inputs
- `internal/review/reviewer.go` — review flow: context loading, model routing, output handling
- `commands/design.md` — add `--with-review` documentation

## Acceptance Criteria

- `hero review --spec <slug>` produces a structured critique of the spec
- Review output includes risks, blind spots, alternatives, and clarifying questions
- `--append` mode appends review notes to the spec under `## Review Notes`
- `hero design --with-review` auto-triggers review after spec generation
- `auto_review_designs: true` in config makes `--with-review` the default
- Reviewer uses the `review` model role (from `model-role-config`) if configured
- The review prompt is customizable by editing `agents/design-reviewer.md`
- `--dialogue` enables multi-round review/response flow

## Boundaries

- Does **not** block implementation — review is advisory, not a gate
- Does **not** modify the spec content itself — only appends `## Review Notes`
- Does **not** require `model-role-config` — falls back to default model if no `review` role is configured
- Does **not** review code output — this is for designs and specs, not implementation files
