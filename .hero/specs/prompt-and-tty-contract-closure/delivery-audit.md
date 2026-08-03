# Delivery audit — prompt-and-tty-contract-closure

**Audited:** `git diff 2767452^...02c47d4`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Shared prompt API is the interactive-input authority — `internal/cli/prompt/prompt.go:52-200` defines the six public operations; `internal/cli/prompt_contract_test.go:22-36` pins their signatures, and `internal/cli/prompt_adoption_test.go:912-997` guards the retained `new --interactive` exception and removed standard-input forks.
- [✓] Input and output TTY checks remain distinct and reject `/dev/null` — `internal/cli/prompt/prompt.go:59-94`; real-PTY, pipe, regular-file, and `/dev/null` assertions in `internal/cli/prompt/prompt_test.go:32-129`.
- [✓] Replaced readers and helpers are removed — the diff removes `connectInput`, `newStdin`, `isTerminal`, `hasPipedInput`, and `exportIsTerminal`; `internal/cli/prompt_adoption_test.go:912-997` provides structural guards.
- [✓] `new --interactive` remains opt-in and reads Cobra input — `internal/cli/new.go:84-93,433-482`; PTY-backed command tests use `rootCmd.SetIn` in `internal/cli/new_test.go:540-552`.
- [✓] Closed and live non-TTY input returns without reading — `internal/cli/prompt_baseline_test.go:345-367` keeps each pipe writer open and drives all 14 migrated sites.
- [✓] Unix and Windows protected secret input uses protected handles — Unix uses `/dev/tty`; Windows opens `CONIN$`/`CONOUT$` through `openWindowsConsoleFiles` and calls `term.ReadPassword` on the input handle (`internal/cli/prompt/secret_terminal_windows.go:20-34`). The portable executed contract verifies exact handle acquisition and failure cleanup in `internal/cli/prompt/secret_terminal_test.go:47-84`; the Windows adapter and test also compile for Windows.
- [✓] No protected terminal returns `ErrNoTTY` without echoed fallback — `internal/cli/prompt/prompt.go:171-199`, `internal/cli/prompt/secret_terminal_test.go:19-45`, and `internal/cli/prompt_sanctioned_breaks_test.go:29-93`.
- [✓] JSON paths never prompt — `internal/cli/connect.go:97-103` routes `connectJSON` through the non-interactive path; `internal/cli/prompt_json_test.go:27-104` drives install and missing-value connect JSON cases under a live PTY and asserts no block or prompt.
- [✓] NEVER-PROMPT commands contain no prompt path — the complete family inventory and source-level prompt/read guard are in `internal/cli/prompt_policy_test.go:19-92`.
- [✓] Fully supplied inputs preserve the committed baseline — all 14 sites are inventoried in `internal/cli/prompt_baseline_test.go:83-342`; supplied/closed/pipe fixtures are checked byte-for-byte by `TestPromptSiteBaseline`.
- [✓] Non-TTY install without `--target` fails — positive closed-stdin and pipe assertions, including no opencode mutation, are in `internal/cli/prompt_sanctioned_breaks_test.go:95-139`.
- [✓] Password entry without a protected terminal fails — `internal/cli/users.go:184-211` and the positive refusal tests at `internal/cli/prompt_sanctioned_breaks_test.go:29-93`.
- [✓] Migrated ordinary prompts are exercised through Cobra streams — `internal/cli/prompt_streams_test.go`, `internal/cli/prompt_adoption_test.go`, `internal/cli/new_test.go:540-552`, and `internal/cli/connect_integrations_test.go:102-121` contain real stream and PTY assertions.

## Changes

- [✓] Shared prompt package and platform secret adapters — added under `internal/cli/prompt/` with primitive, predicate, acquisition, cleanup, and platform test code.
- [✓] Original prompt-site migrations — the named CLI files route ordinary reads through Cobra streams and the shared primitives; `brief.go:834-840` uses the output predicate.
- [✓] `new.go` reconciliation — `newStdin` is removed, the helper receives `cmd.InOrStdin()`, and PTY tests preserve explicit opt-in behavior.
- [✓] Two compatibility corrections — install-target refusal and protected-password refusal have positive subprocess assertions in `internal/cli/prompt_sanctioned_breaks_test.go`.
- [✓] Fixtures and standing policy tests — baseline, live-pipe, NEVER-PROMPT, install/connect JSON, and portable Windows console-contract coverage are present and were reported passing with the full normal/race/build validation set.

## Audit notes

- No non-DONE ledger rows, performative claims, or out-of-scope implementation changes remain.
