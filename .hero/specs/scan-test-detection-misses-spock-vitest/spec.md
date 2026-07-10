---
title: hero scan misses Spock, dependency-only Vitest/Playwright, and Playwright .js/.mts configs
slug: scan-test-detection-misses-spock-vitest
type: bug
status: completed
completed_at: 2026-06-23
severity: medium
created: 2026-05-12
tags: [scan, detector, test-frameworks, spock, vitest, playwright, junit]
---
# hero scan misses Spock, dependency-only Vitest/Playwright, and Playwright .js/.mts configs

## Problem

`hero scan` reported a "No tests" gap on a Groovy/Spock + Vitest/Playwright project that actually has all three test frameworks wired up. False-positive gaps damage trust in the scan output and waste the host model's enrichment turn writing knowledge entries about a fictional test gap.

## Root Cause

Test-framework detection in `internal/scan/scan.go` was marker-file-only and shallow:

- **No Spock detection at all.** Spock uses no root-level config file; it lives as a Gradle/Maven dependency plus tests in `src/test/groovy/`.
- **Vitest detection required `vitest.config.{ts,js}` at root.** Projects that configure Vitest inline in `vite.config.ts` (under the `test:` key) or only via a `vitest` package.json dependency were missed.
- **Playwright detection required `playwright.config.ts` at root.** Missed `playwright.config.{js,mts,cts}` variants and projects that only declare `@playwright/test` as a dependency.
- **No JUnit, Kotest, or TestNG detection** for JVM projects.

When all detectors miss, `detectGaps` (`internal/scan/enrich.go:798`) fires the "No tests" gap based on `e.GoTestCount == 0 && len(r.TestFrames) == 0`.

## Fix

Two new detector helpers in `internal/scan/scan.go`, both called from `detectFromMarkers`:

1. **`detectTestFromPackageJSON(content, existing)`** — scans `package.json` for `"vitest"`, `"@playwright/test"`, `"jest"`, `"@jest/core"`, `"cypress"`, `"mocha"`, `"ava"` and adds matching `TestFramework` entries (skipping anything already detected via a config file).

2. **`detectJVMTestFrameworks(root, existing)`** — concatenates `build.gradle` + `build.gradle.kts` + `pom.xml` contents and looks for:
   - Spock: `spock-core` or `org.spockframework` (falls back to `src/test/groovy/` directory existence).
   - Kotest: `io.kotest`.
   - TestNG: `org.testng` or `<artifactId>testng</artifactId>`.
   - JUnit: `junit-jupiter` or `<artifactId>junit` (falls back to `src/test/{java,kotlin}/`).

Also added marker variants: `playwright.config.js`, `playwright.config.mts`, `vitest.config.mts`.

## Changes

- `internal/scan/scan.go`: added `detectTestFromPackageJSON`, `detectJVMTestFrameworks`, new Playwright/Vitest config-file marker variants, wired both helpers into `detectFromMarkers`.

## Verification

Live verification on a synthetic project (`build.gradle` declaring `spock-core` + `junit-jupiter`, `package.json` declaring `vitest` + `@playwright/test`, `src/test/groovy/SampleSpec.groovy`):

```
Test Frameworks:
  Vitest               package.json
  Playwright           package.json
  Spock                build.gradle
  JUnit                build.gradle/pom.xml
```

Empty Go project still emits the "No tests" gap correctly (`e.GoTestCount == 0 && len(r.TestFrames) == 0`).

`go test ./internal/scan/...` passes.

## Kickoff

Resume work on the test-framework detector miss. Read this spec and `internal/scan/scan.go` (the `detectFromMarkers`, `detectTestFromPackageJSON`, and `detectJVMTestFrameworks` functions). Fix is in place. Outstanding work to consider: (a) unit tests covering Spock + Vitest + Playwright detection paths (currently verified live, no unit coverage added); (b) frame detection for less-common JVM frameworks (Spek for Kotlin, ScalaTest); (c) consider parsing `package.json`'s `"scripts"` section for `test`/`e2e` patterns when no framework dep is declared.

## Resolution (2026-06-23)

**Verified already fixed** (status was stale). internal/scan/scan.go detects Spock (detectJVMTestFrameworks), dependency-only Vitest/Playwright (detectTestFromPackageJSON), and .js/.mts Playwright/Vitest config markers. Remaining: dedicated unit tests for these detectors (test-coverage debt, tracked separately — behavior verified live).
