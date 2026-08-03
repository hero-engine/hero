# Scope provenance — interactive CLI successor

Baseline: `main` at `bf18979ab3b05fd86c1f09b0ff380d92bdfc909f`.
Reviewed range: `main...HEAD` on the clean successor branch. The command below
uses zero context so the hunk count is an exact, reproducible unit:

```sh
git diff --unified=0 --no-ext-diff main...HEAD -- internal/cli internal/ptytest
```

It identifies 233 non-test Go hunks. Every production hunk is listed below.
For a file with multiple historical contributors, "owner" means the final child
accountable for the behavior in the curated successor, rather than the commit
that first touched the file. Hunk ranges are the ordinal hunks produced by the
command above; together they cover every hunk in each file.

| File and zero-context hunk range | Parent AC | Final owning child | Why it is in scope |
|---|---|---|---|
| internal/cli/brief.go H1-H3 | AC-1, AC-2 | prompt-and-tty-contract-closure | Uses the shared output-terminal predicate. |
| internal/cli/connect.go H1-H65 | AC-3, AC-5, AC-6, AC-7, AC-8 | interactive-setup-and-connect-closure | One prompt-gated collector and one effective connection writer preserve safe machine paths and role/capability/default resolution. |
| internal/cli/connect_alias.go H1 | AC-5, AC-8 | interactive-setup-and-connect-closure | Keeps the alias help surface equivalent to connect. |
| internal/cli/export.go H1-H12 | AC-1, AC-3, AC-5, AC-6, AC-7 | prompt-and-tty-contract-closure | Moves the conflict prompt onto the shared stream and policy contract. |
| internal/cli/handoff.go H1-H18 | AC-10, AC-11, AC-12 | corpus-selector-closure | Adds missing-only local selection for handoff and handoff accept. |
| internal/cli/install.go H1-H18 | AC-1, AC-3, AC-5, AC-6, AC-7 | prompt-and-tty-contract-closure | Routes target and satellite confirmations through the shared input contract. |
| internal/cli/install_satellites.go H1-H14 | AC-1, AC-3, AC-5, AC-6, AC-7 | prompt-and-tty-contract-closure | Applies the same gate and Cobra-stream handling to satellite interaction. |
| internal/cli/new.go H1-H4 | AC-1, AC-3, AC-5 | prompt-and-tty-contract-closure | Removes the package-global reader while retaining explicit interactive opt-in. |
| internal/cli/note.go H1-H5 | AC-1, AC-3, AC-5 | prompt-and-tty-contract-closure | Uses Cobra input instead of the inverted stdin path. |
| internal/cli/prompt/prompt.go H1 | AC-1, AC-2, AC-3, AC-4 | prompt-and-tty-contract-closure | Defines the sole prompt authority and stream classification. |
| internal/cli/prompt/secret_terminal.go H1 | AC-4 | prompt-and-tty-contract-closure | Keeps the protected-terminal abstraction independent of ordinary streams. |
| internal/cli/prompt/secret_terminal_unix.go H1 | AC-4 | prompt-and-tty-contract-closure | Uses the Unix protected terminal implementation. |
| internal/cli/prompt/secret_terminal_windows.go H1 | AC-4 | prompt-and-tty-contract-closure | Uses Windows console handles and non-echoing password reads. |
| internal/cli/prompt/secret_terminal_windows_seam.go H1 | AC-4 | prompt-and-tty-contract-closure | Makes Windows console acquisition executable at the platform seam. |
| internal/cli/prompt_args.go H1 | AC-3, AC-5, AC-6, AC-7 | interactive-setup-and-connect-closure | Relaxes only missing arguments at an input TTY for named setup commands. |
| internal/cli/prompt_line.go H1 | AC-1, AC-3 | prompt-and-tty-contract-closure | Supplies bytewise shared line reading without reader read-ahead. |
| internal/cli/repos.go H1-H6 | AC-5, AC-7 | interactive-setup-and-connect-closure | Adds bounded missing-only repository setup prompts. |
| internal/cli/score.go H1-H2 | AC-10, AC-11, AC-12 | corpus-selector-closure | Adds the frozen score selector target. |
| internal/cli/selector.go H1 | AC-10, AC-11, AC-12 | corpus-selector-closure | Provides bounded full-corpus reachability, cancellation, and noninteractive bypass. |
| internal/cli/size.go H1-H5 | AC-10, AC-11, AC-12 | corpus-selector-closure | Adds the frozen size selector target. |
| internal/cli/skill.go H1-H24 | AC-10, AC-11, AC-12 | corpus-selector-closure | Adds selectors only to the five specified skill verbs. |
| internal/cli/spec_move.go H1-H3 | AC-10, AC-11, AC-12 | corpus-selector-closure | Adds missing-source selection while retaining flag validation. |
| internal/cli/supersede.go H1-H7 | AC-10, AC-11, AC-12 | corpus-selector-closure | Adds independent omitted-position selection without changing modes. |
| internal/cli/trust.go H1-H8 | AC-5, AC-7 | interactive-setup-and-connect-closure | Adds a TTY-only missing target prompt. |
| internal/cli/uninstall.go H1-H15 | AC-5, AC-7, AC-9 | interactive-setup-and-connect-closure | Shares the six-target picker and deletes only target-owned manifest surfaces. |
| internal/cli/users.go H1-H8 | AC-4, AC-5, AC-7 | interactive-setup-and-connect-closure | Adds missing-only setup prompts while preserving protected secret entry. |
| internal/cli/verify.go H1-H3 | AC-10, AC-11, AC-12 | corpus-selector-closure | Adds the frozen verify selector target. |

The four internal/ptytest files contribute four additional non-test Go hunks,
but are test-only portable PTY support: every importer is a _test.go file.
They are therefore validation infrastructure, not production behavior.

## Boundary result

The range contains no production hunks in index/search, guided init, alias
parity/derivation, invocation-string guards, form/schema engines, timeout
policy, graph/spec-terminal behavior, or a new selector target outside the
frozen set. The root-instruction preservation change in uninstall is necessary
to AC-9's manifest-safe removal: it removes no shared root instruction file.
No production hunk was rejected or needed routing back to an implementation
child during this gate.
