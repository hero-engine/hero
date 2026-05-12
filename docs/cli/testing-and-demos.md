# Testing & Demos

Commands for generating acceptance tests from specs and recording demos.

## Test Generation

### `hero test generate`

Generate acceptance tests from a spec's success criteria.

```bash
hero test generate auth-flow
hero test generate --all                        # Generate for all specs
hero test generate auth-flow --mode autonomous  # Fully automated
hero test generate auth-flow --mode assisted    # Human-in-the-loop
hero test generate auth-flow --mode agent       # Agent-driven execution
```

**Flags:**

| Flag | Description |
|---|---|
| `--all` | Generate tests for all specs that don't have them yet |
| `--mode` | Test generation mode (see below) |

**Generation modes:**

| Mode | Description |
|---|---|
| `autonomous` | Fully automated — Hero reads the spec, generates tests, and writes them to disk with no human input. Best for well-defined specs with clear acceptance criteria. |
| `assisted` | Human-in-the-loop — Hero generates a draft and prompts you to review, edit, and confirm each test before writing. Good for nuanced specs where judgment is needed. |
| `agent` | Agent-driven — produces test specifications that an AI agent executes at runtime rather than static test files. Suited for integration and end-to-end scenarios that require dynamic setup. |

### `hero test run`

Run the generated tests for a spec.

```bash
hero test run auth-flow
```

Executes the test suite associated with the spec and reports pass/fail against the spec's success criteria.

### `hero test list`

List all specs that have generated tests.

```bash
hero test list
```

## Smoke Verification

### `hero smoke`

Run per-feature smoke scripts that exercise a feature's happy path against a real workspace. Each script reads the feature's acceptance criteria, runs the steps each AC requires, and emits a `results.json` that flips AC status in the graph.

```bash
hero smoke <feature-slug>     # run one feature's smoke
hero smoke --since <ref>      # run smokes for features touched since <git-ref>
hero smoke --area <area>      # run all smokes matching an area tag/slug
hero smoke --all              # run every per-feature smoke
hero smoke status             # show last-run status for all smokes
```

`--since` is the killer use: pre-commit / pre-push hook runs only the smokes affected by the diff. Fast, targeted, never wasteful.

Results are written to `.hero/smoke/last-run.json`. Per-run artifacts (logs, captured stdout/stderr, results) live under `tmp/e2e/<slug>-<timestamp>/`.

### Per-command `--smoke`

Every Hero command has a built-in `--smoke` flag that runs its own happy-path verification and exits 0/1 with a one-line result.

```bash
hero scan --smoke
hero status --smoke
```

This means even a command without a per-feature smoke script has *something* runnable.

### CI integration

`.github/workflows/smoke.yml` runs:
- **Pull requests:** `hero smoke --since <base-sha>` — only smokes for features touched by the PR. A failed smoke blocks the merge.
- **Push to main:** same, against the previous commit.
- **Nightly (07:00 UTC):** `hero smoke --all`.
- **Manual dispatch:** pick `since` (with a ref) or `all`.

Failed runs upload `tmp/e2e/**` and `.hero/smoke/last-run.json` as a 14-day artifact for diagnostics.

External-dependency smokes (Jira, GitHub API, team server) skip by default. Set `HERO_SMOKE_EXTERNAL=1` in a workflow that has the credentials.

### Local equivalents

```bash
make smoke       # mirrors PR CI: hero smoke --since origin/main
make smoke-all   # mirrors nightly CI: hero smoke --all
```

## Demo Recording

### `hero spec demo record`

Record a demo of a delivered feature based on its spec.

```bash
hero spec demo record auth-flow
```

Produces a structured demo script with steps derived from the spec's user-facing scenarios.

### `hero spec demo list`

List all recorded demos.

```bash
hero spec demo list
```

### `hero spec demo clean`

Remove stale or orphaned demo recordings.

```bash
hero spec demo clean
```
