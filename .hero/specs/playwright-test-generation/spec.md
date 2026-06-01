---
title: Pluggable Test Generation — Playwright-First Spec Validation
slug: playwright-test-generation
type: feature
status: completed
milestone: v0.5
tags: [testing, playwright, validation, spec-driven, e2e]
created: 2026-04-14
relations:
  - target: skills-git-native
    kind: related
  - target: git-hook-integration
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Give teams a way to generate, scaffold, or run end-to-end tests directly from Hero spec acceptance criteria — so that every delivered feature has verifiable validation coverage. Playwright is the first adapter, but the framework is pluggable from day one.

## Problem

Specs have well-defined `## Acceptance Criteria` sections, but those criteria live only in markdown. There is no automated bridge from "criteria written" to "tests exist." Engineers either skip tests, write them manually from scratch, or rely on agents to remember criteria. This creates a gap between what was designed and what gets verified.

Linear, Notion, and GitHub Projects have no concept of spec-to-test generation. Hero can close this gap by treating acceptance criteria as structured inputs for test scaffolding and generation.

## Design

### Configuration

`hero.json` gains a `testing` block:

```json
{
  "testing": {
    "framework": "playwright",
    "mode": "autonomous",
    "test_dir": "e2e",
    "runner_command": "npx playwright test",
    "base_url": "http://localhost:3000",
    "config_path": "playwright.config.ts"
  }
}
```

| Field | Type | Default | Description |
|---|---|---|---|
| `framework` | string | `"playwright"` | Test framework adapter to use |
| `mode` | string | `"autonomous"` | Generation mode: `agent`, `assisted`, `autonomous` |
| `test_dir` | string | `"e2e"` | Directory for generated test files (relative to project root) |
| `runner_command` | string | `"npx playwright test"` | Command to run tests |
| `base_url` | string | `""` | Application base URL for navigation |
| `config_path` | string | `""` | Path to framework config file |

### Three Generation Modes

**1. Agent-driven (`mode: "agent"`)** — During `/deliver`, the MCP context includes the spec's acceptance criteria formatted as test requirements. The coding agent is responsible for writing the tests as part of delivery. Hero provides the criteria and target file path; the agent does the work.

**2. Assisted (`mode: "assisted"`)** — `hero test generate <slug>` creates a scaffolded test file with `test.describe` blocks and TODO comments derived from the spec's acceptance criteria. Each criterion becomes a `test()` block with a descriptive name and a `// TODO: implement` body. The engineer fills in the implementation.

**3. Autonomous (`mode: "autonomous"`)** — `hero test generate <slug>` creates complete, runnable Playwright tests from acceptance criteria. Hero uses heuristic mapping from natural-language criteria to Playwright assertion patterns:
- Criteria mentioning URLs → `toHaveURL` assertions
- Criteria mentioning visibility/display → `toBeVisible` assertions
- Criteria mentioning text content → `toContainText` / `toHaveText` assertions
- Criteria mentioning counts → `toHaveCount` assertions
- Criteria mentioning form inputs → `toHaveValue` assertions
- Criteria mentioning existence → locator + `toBeVisible` assertions
- Fallback → `test.skip()` with the criterion text as a comment

This is a best-effort heuristic — not an LLM. It generates a starting point that can be refined.

### CLI Commands

```
hero test generate <slug>        # Generate test file for a spec
hero test generate --all         # Generate tests for all delivering/completed specs
hero test run <slug>             # Run tests for a specific spec
hero test run --all              # Run all hero-generated tests
hero test list                   # List specs with test coverage status
hero test show <slug>            # Print the generated test file path and content
```

### Acceptance Criteria Extraction

The spec parser already captures `## Acceptance Criteria` as `s.Sections["acceptance criteria"]`. The testing package parses this section:

1. Split by newline
2. Each line starting with `- ` or `* ` is a criterion
3. Strip the bullet prefix and leading/trailing whitespace
4. Each criterion becomes a `test()` block

### Generated Test File Structure

For a spec `my-feature`, the generated file is `<test_dir>/my-feature.spec.ts`:

```typescript
import { test, expect } from '@playwright/test';

// Auto-generated from Hero spec: my-feature
// Spec: My Feature — A description from the spec title
// Mode: autonomous
// Generated: 2026-04-14T10:30:00Z

test.describe('my-feature', () => {
  test('criterion 1 description', async ({ page }) => {
    await page.goto('http://localhost:3000');
    // Assertion derived from criterion text
    await expect(page).toHaveTitle(/My Feature/);
  });

  test('criterion 2 description', async ({ page }) => {
    await page.goto('http://localhost:3000');
    await expect(page.locator('.feature-element')).toBeVisible();
  });
});
```

### Test Runner Integration

`hero test run <slug>` executes:
```
npx playwright test e2e/my-feature.spec.ts
```

Exit code is passed through. Output is streamed to stdout/stderr.

### MCP Tool

```json
{
  "name": "hero_test_generate",
  "description": "Generate test files from spec acceptance criteria",
  "inputSchema": {
    "type": "object",
    "properties": {
      "slug": { "type": "string", "description": "Spec slug to generate tests for" },
      "mode": { "type": "string", "enum": ["agent", "assisted", "autonomous"], "description": "Override generation mode" }
    },
    "required": ["slug"]
  }
}
```

### Pluggable Framework Interface

The Go implementation uses an interface:

```go
type TestFramework interface {
    Name() string
    GenerateAssisted(spec *spec.Spec, criteria []string, cfg TestingConfig) (string, error)
    GenerateAutonomous(spec *spec.Spec, criteria []string, cfg TestingConfig) (string, error)
    AgentContext(spec *spec.Spec, criteria []string, cfg TestingConfig) string
    TestFilePath(slug string, cfg TestingConfig) string
    RunCommand(testFile string, cfg TestingConfig) (string, []string)
}
```

`PlaywrightFramework` implements this interface. Additional adapters (Cypress, TestCafe) can be added later by implementing the same interface.

## Changes

- `internal/config/config.go` — add `TestingConfig` struct and `Testing` field to `Config`
- `internal/testing/framework.go` — `TestFramework` interface, criteria extraction, registry
- `internal/testing/playwright.go` — Playwright adapter implementing `TestFramework`
- `internal/testing/generate.go` — orchestration: load spec, extract criteria, call framework, write file
- `internal/cli/test.go` — `hero test generate|run|list|show` commands
- `internal/cli/root.go` — register `testCmd`
- `internal/serve/mcp.go` — add `hero_test_generate` tool

## Acceptance Criteria

- `hero test generate <slug>` produces a `.spec.ts` file in the configured test directory
- Generated file imports `{ test, expect }` from `@playwright/test`
- Each acceptance criterion from the spec becomes a separate `test()` block
- In assisted mode, test bodies contain `// TODO: implement` placeholders
- In autonomous mode, test bodies contain heuristic-mapped Playwright assertions
- `hero test run <slug>` invokes the configured runner command and passes through exit codes
- `hero test list` shows specs with a checkmark for those that have generated test files
- `hero test show <slug>` prints the test file content
- `--all` flag on generate and run operates on all delivering/completed specs
- Configuration is read from `hero.json` `testing` section with sensible defaults
- Framework is pluggable — adding a new adapter requires implementing `TestFramework` interface only
- Works with no `testing` config (defaults to Playwright, autonomous mode, `e2e/` directory)
- `hero_test_generate` MCP tool generates tests and returns the file path

## Boundaries

- Does **not** make LLM API calls — generation is template/heuristic based
- Does **not** install Playwright or any npm packages — assumes the project has them
- Does **not** parse TypeScript or validate generated code — write-and-run
- Does **not** modify existing test files — only creates new ones (overwrites if re-generated)
- Does **not** run a dev server — assumes `base_url` is already running
- Autonomous mode is best-effort heuristic — not guaranteed to produce passing tests
