---
name: acceptance-criteria-gherkin
description: Given/When/Then as the alternate acceptance-criteria shape when a team prefers it over EARS — clause structure, data tables, scenario outlines, and the anti-pattern of Gherkin novels.
metadata:
  audience: story-writer, pm-reviewer
  purpose: ac-authoring
---

## What I do

Provide the Given/When/Then (Gherkin) acceptance-criteria format as the **alternate** to EARS. EARS is the pack default (see `acceptance-criteria-ears`); Gherkin is the right tool when a team already runs BDD, has a Cucumber/SpecFlow/Behave suite the AC should map to, or simply thinks in scenarios. Both formats serve the same goal — criteria that survive handoff to engineering with a single unambiguous pass/fail. This skill supplies the clause shapes, the data-table and scenario-outline mechanics for compressing repetitive cases, and the discipline that keeps scenarios from bloating into unreadable novels. When `story-writer` is asked for Gherkin, or `pm-reviewer` checks Gherkin AC, this is the bar.

## When to use me

- the user (or the team's active convention) explicitly asks for Gherkin over EARS
- the story's AC will feed a BDD test suite (Cucumber, SpecFlow, Behave, Gauge)
- the behavior is naturally scenario-shaped — distinct states leading to distinct outcomes
- reviewing existing Gherkin AC for structure and testability (`pm-reviewer`)

Default to EARS otherwise. Don't impose Gherkin on a team that isn't using it — the format only pays off when scenarios map to something (tests, a shared BDD vocabulary).

## The clause shape

A scenario is three clauses. Each has a precise job:

```gherkin
Scenario: New account exports data within the latency budget
  Given a signed-in user on a new account with at least one dataset
  When they click "Export CSV"
  Then a downloadable CSV is delivered within 5 seconds
  And the file contains every row visible in the current view
```

- **Given** — the precondition / starting state. What must be true before the trigger. Multiple Givens with `And`.
- **When** — the trigger. The single action or event under test. Keep it to *one* When per scenario; two triggers means two scenarios.
- **Then** — the expected, observable outcome. What the system does that you could verify. Additional expectations with `And`.

Each clause describes **behavior, not implementation**. "Then a CSV is delivered within 5 seconds" is behavior; "Then the `ExportController` calls `s3.putObject`" is implementation leaking into the criterion — cut it.

## Data tables — compress repetitive cases

When the same behavior holds across several inputs, a data table beats copy-pasting the scenario. Use a `Scenario Outline` with `Examples`:

```gherkin
Scenario Outline: Export respects the row-count limit by plan tier
  Given a user on the "<tier>" plan
  When they export a dataset of <rows> rows
  Then the export <result>

  Examples:
    | tier       | rows    | result                              |
    | free       | 5000    | succeeds                            |
    | free       | 50001   | fails with an upgrade prompt        |
    | enterprise | 500000  | succeeds                            |
```

The `<placeholder>` tokens in the outline bind to columns in `Examples`. One outline covers the whole truth table — far more reviewable than three near-identical scenarios, and it maps cleanly onto parameterized tests.

Inline data tables (a table argument attached to a step) work when a single step needs structured input:

```gherkin
  Given the following intake items exist:
    | id    | segment    | status  |
    | i-241 | enterprise | new     |
    | i-256 | smb        | triaged |
```

## Tags for cross-cutting concerns

Tag scenarios to group and filter them — smoke sets, work-in-progress, feature areas:

```gherkin
@export @smoke
Scenario: ...
```

Tags let the suite run a subset (`@smoke` before every deploy) and let reviewers see at a glance which cross-cutting concern a scenario belongs to. Keep the tag vocabulary small and team-agreed; a tag per scenario is noise.

## Gherkin ↔ EARS

They're interchangeable in intent; pick one per team, don't mix within a story:

| EARS | Gherkin |
|---|---|
| `WHEN <trigger> THE SYSTEM SHALL <response>` | `When <trigger> / Then <response>` |
| `WHILE <state> THE SYSTEM SHALL <response>` | `Given <state> / When … / Then <response>` |
| `IF <unwanted> THEN THE SYSTEM SHALL <response>` | `Given <unwanted condition> / When … / Then <response>` |

If a team has no BDD suite and no strong preference, EARS is terser and the pack default. Gherkin earns its extra ceremony when the scenarios *run* as tests.

## Anti-patterns

- **Gherkin novels.** A scenario with 10+ steps, nested Givens, and a paragraph of Whens is unreadable and untestable. If a scenario needs that many steps, the behavior is too coarse — split it. Aim for 3–7 steps.
- **Implementation in steps.** "When the user clicks the button with id `#export-btn`" or "Then the `exports` table gets a row" — that's testing the *how*. Describe observable behavior; leave selectors and tables to the test code.
- **Multiple Whens.** Two triggers in one scenario means you're testing two things and can't tell which failed. One When per scenario.
- **Vague Thens.** "Then it works" / "Then the user is happy" — not verifiable. Name the observable outcome and its threshold.
- **Copy-pasted scenarios.** Three scenarios differing only in an input value should be one Scenario Outline with an Examples table.
- **Tag sprawl.** A unique tag on every scenario defeats the point; tags group, they don't label.

## Cross-references

- `acceptance-criteria-ears` — the pack default AC format; use it unless the team runs BDD or asks for Gherkin.
- `story-writing-invest` — Testable is the INVEST letter this format serves; Gherkin makes "done" unambiguous.
- `pm-preset-detection` — some teams pin an AC format by convention; read it before choosing.
- Prior art: Gherkin / Cucumber (Aslak Hellesøy, Matt Wynne, *The Cucumber Book*); Dan North on BDD.
