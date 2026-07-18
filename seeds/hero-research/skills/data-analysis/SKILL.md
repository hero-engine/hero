---
name: data-analysis
description: How to analyze structured or tabular data the user provides — compute honestly, summarize the shape, surface real patterns, and caveat the limits (sample size, missingness, correlation is not causation). Analysis that states its own uncertainty instead of overclaiming.
metadata:
  audience: data-analyst
  purpose: grounded-data-analysis
---

## What I do

I give the discipline for analyzing **data the user supplies or points at** — a
table, a CSV, a spreadsheet, a pasted set of numbers. The job is to compute
correctly, describe what the data actually shows, surface genuine patterns, and be
honest about what the data cannot support. The failure mode I exist to prevent is
confident analysis that overclaims — reading a trend into noise, inferring
causation from correlation, or generalizing from a sample too small to carry the
conclusion.

## When to use me

Load me when the user provides structured data and asks what it shows —
"analyze this", "what's the trend", "summarize these numbers", "is there a pattern
here". The `data-analyst` agent loads me automatically.

## Workflow

### 1. Understand the data before analyzing it

Before computing anything, establish what you are looking at: what each column or
field means, the units, the row count, the time span, and — critically — what is
*missing*. Data with unlabeled columns, ambiguous units, or unexplained gaps needs
those resolved (ask the user) before analysis, not assumed past.

### 2. Compute honestly

Do the arithmetic the question needs and show the computation, not just the
result, so the user can check it. State the operation ("mean of column B over the
142 non-null rows"), not a bare number. If a computation requires an assumption
(how to treat nulls, which rows to include), name the assumption.

### 3. Describe the shape

Summarize what the data looks like: central tendency, spread, range, notable
outliers, distribution shape where it matters. The shape often answers the
question more honestly than a single headline statistic — a mean hides a bimodal
split; a total hides that one row dominates.

### 4. Surface patterns — and test them against the data

Point out real patterns: trends over time, correlations between fields,
clusters, outliers worth attention. For each, check it against the data before
asserting it — a two-point "trend" is not a trend, and an apparent correlation on
twelve rows may be noise. State the pattern *and* how strongly the data supports
it.

### 5. Caveat the limits

Every readout closes with what the data cannot support:

- **Sample size.** A conclusion from a handful of rows is a hypothesis, not a
  finding. Say so.
- **Missingness.** Gaps and nulls bias results; name which columns had them and
  how you handled them.
- **Correlation is not causation.** Two fields moving together is not one causing
  the other. Never phrase a correlation as a cause.
- **Selection.** If the data is a non-random slice (only converted users, only
  one region), the conclusion generalizes only to that slice.

## The honesty rules

- **Compute, then caveat — never the reverse.** State what the data shows, then
  immediately state what it cannot show. The caveat is not a footnote; it is part
  of the answer.
- **Show the work.** A number with no visible computation is unauditable. The
  user should be able to redo your arithmetic.
- **Small n is a hypothesis.** Do not dress a pattern from thin data as a
  conclusion. Name the sample size next to the claim.
- **Never launder correlation into causation.** This is the single most common
  overclaim in data analysis. Guard it explicitly.

## Anti-patterns

- **The headline statistic that hides the shape.** A mean or total presented
  without the distribution behind it, when the distribution is the real story.
- **Trend from two points.** Calling any up-and-to-the-right pair a trend.
- **Causal language on correlational data.** "X drives Y" when the data only
  shows they move together.
- **Silent null handling.** Dropping or imputing missing values without telling
  the user, so the result looks cleaner than the data is.
- **Generalizing past the sample.** Stating a slice's pattern as a universal
  truth.
