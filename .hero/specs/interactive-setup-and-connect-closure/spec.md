---
title: "Interactive Setup and Connect Closure"
slug: interactive-setup-and-connect-closure
type: feature
status: completed
created: 2026-08-03
domain: engineering
size: large
priority: high
parent: interactive-cli-input-scoped-completion
depends-on: [prompt-and-tty-contract-closure]
relates-to:
  - connect-writer-unification
  - prompt-adoption-setup-commands
  - connect-help-accuracy
  - uninstall-target-parity
tags: [cli, prompt, connect, setup, uninstall]
delivery_method: manual
completed_at: 2026-08-03T21:25:41Z
---

# Interactive Setup and Connect Closure

## Context

This is the bounded setup child of `interactive-cli-input-scoped-completion`.
It starts only after `prompt-and-tty-contract-closure` verifies, and it starts
from the successor branch based on `main`. The old
`design/interactive-cli-input` branch is evidence and a selective donor, not a
branch to merge or replay wholesale.

Four donor deliveries contain the original user-facing outcome:

- `12a6775` added safe `copilot` and `generic` uninstall paths;
- `485c32f` unified connect persistence and made `--role` effective;
- `8bfe053` made both connect help surfaces describe their real flags;
- `4b1163b` added missing-value prompts to the setup commands.

Those deliveries also contain useful audit findings that belong here: provider
options must derive from the provider registry; role/capability/default
resolution must have one authority; a failed committed-config write must not
leave an orphaned credential; both `hero connect` and `hero sync connect` need
the same flag help; and deleting under `.github/` is safe only through the
install manifest.

The donor is not correct as-is. `runConnectInteractive` prints its banner and
enters `collectFields` without first proving that stdin is a terminal. A closed
reader returns an error, so its test passes, but a live open pipe blocks waiting
for input. The curated implementation must gate the complete interactive
collector before its first output or read. It must not port the donor's generic
`promptfield.go`/`skill.go` coupling merely because those files make a commit
easy to cherry-pick.

## Goal

Deliver the original guided setup experience without changing machine-driven
CLI behavior: connect and the named setup commands ask only for missing values
at a terminal; connect has one persistence path with correct role, capability,
and default selection; install and uninstall support the same six harnesses;
and the help accurately teaches both interactive and flag-driven use. Closed
stdin and a live non-TTY pipe fail promptly without a prompt or read.

## Kickoff

Closes interactive connect and setup flows while leaving automation paths non-blocking and manifest-safe.

**Status:** completed — the bounded preservation/baseline repair passed its
independent SHIP audit and Hero verification.

**Pick up at:** consult the archived audit and Completion Ledger only when
investigating connect/setup, six-target, or manifest-removal regressions.

→ `.hero/specs/interactive-setup-and-connect-closure/delivery-audit.md`

**Files:** `internal/cli/connect.go`, `internal/cli/uninstall.go`, `internal/cli/prompt_args.go`, `internal/cli/connect_writer_unification_test.go`
**Skip:** reopening delivery, generic prompt fields, skill coupling,
init/index/alias expansion, network calls, and unlisted uninstall repairs.

## Problem

### Connect has two meanings for the same connection

On `main`, `runConnectNonInteractive` writes role selectors, capabilities, and
default state, while the interactive provider-specific functions use
`saveConnection` plus `updateHeroJSON` and disagree on all three. `--role` is
also absent from `runConnect`'s non-interactive routing predicate, so
`hero connect github --role code-host` accepts the flag and then silently
creates a delivery connection. Without an explicit `code-host` capability,
`Config.ResolveCodeHostConnection` rejects the result.

The donor fixed the semantic split, but its interactive collector can still
consume or wait on non-TTY input. The safety decision must happen before the
banner, role picker, first field prompt, or provider verification.

### Setup requires values users should be able to answer

The original PROMPT-class setup surface is deliberately small: connect,
install/uninstall target selection, `admin repos add`, `admin users add`,
`admin users passwd`, `trust`, and the existing satellite confirmation flow.
Today several of these fail Cobra argument validation before a person at a
terminal can supply the missing value. Relaxing them indiscriminately would
make scripts hang, so only an argument shortfall at a real input TTY may reach a
prompt; extra arguments and every non-TTY invocation keep their existing
failure.

### Harness removal and help are asymmetric

`hero install` supports `opencode|cursor|claude|copilot|codex|generic`, but
`main` uninstall supports only four. `copilot` removal is particularly
sensitive because its files share `.github/` with user workflows and prompts.
The manifest, not a glob, must authorize every deletion.

Connect's help also describes `--project` as a remove-only flag even though
flag-driven connect requires it, and it omits the automation flags a user needs
after secure prompting refuses non-TTY input. The top-level alias and
`hero sync connect` register overlapping flags separately, so correcting only
one surface still leaves the command a user types wrong.

## Approach

### Collect values twice, validate and persist once

Keep interactive and flag-driven collectors separate because they obtain values
differently, but make both construct one private `connectionInput` and call one
`writeConnection`. The writer is the sole authority for:

- provider/role validation through `config.ValidateProviderRole`;
- provider settings validation through `config.ValidateConnectionSettings`;
- capability derivation and preservation of existing capabilities;
- `roles.<role>` selection;
- default selection, with `code-host` selected only by its role and therefore
  never stealing `integrations.default`;
- committed/local/global write ordering.

Patch non-secret configuration before storing a credential. If the config
patch fails, return the error and persist no credential. Preserve the existing
store choices (`--global`, project-local credentials, and `--local-only`) and
the existing non-interactive verification contract.

Use a connect-private `connectField` table and `collectConnectFields` helper in
`connect.go` for the real provider conditionality (`base_url`, `project`,
`user_email`, token). It may support only one provider-equality condition,
defaults, required messages, and secret-vs-text reads. Do not port the donor's
shared `promptfield.go`, do not rewire `skill.go`, and do not turn this into a
form/schema abstraction.

### Gate before interactive work

When the provider argument is omitted, offer a TTY-only choice derived from the
same provider registry used by both collectors. When a provider is supplied but
required fields are missing, `runConnectInteractive` must check
`prompt.IsInputTTY(cmd.InOrStdin())` before printing or reading anything. On
closed or live non-TTY input it returns the provider's first missing-field error
immediately, emits no prompt, and performs no network or persistence work.

`--json` always refuses the interactive form. `--token-stdin` is different: the
caller explicitly requested a stream read, so reading it through EOF remains
the supported automation path and is not covered by the no-wait rule.

Add `cmd.Flags().Changed("role")` to the flag-driven routing predicate. The
default value cannot distinguish an omitted `--role` from an explicit
`--role delivery`, so checking the value is insufficient. Interactive role
choices come only from roles the chosen provider can serve; a provider with one
valid role is not asked.

### Keep setup prompts flat and additive

Add one `promptableArgs` validator in `internal/cli/prompt_args.go`. It relaxes
an existing positional rule only when there are too few arguments and stdin is
a terminal. It does not relax extra arguments and does not own any prompt.

The flat setup commands call the prompt primitives directly:

- `admin repos add`: ask for alias, then path, skipping either value already
  supplied;
- `admin users add`: ask for a missing username before opening the job queue;
  keep `--password` as the supplied, non-prompting path;
- `admin users passwd`: keep the username required and use the verified secure
  prompt plus confirmation for the new password; do not invent a password flag;
- `trust`: ask only for the missing `codex|claude` target; keep the optional
  scope default and explicit scope behavior unchanged;
- `install` and `uninstall`: choose from one six-target enumeration;
- satellite setup: preserve the original add-subproject/candidate-walk
  confirmation through Cobra streams, defaulting to no and remaining silent
  for non-TTY, dry-run, and JSON paths.

No setup command may use `connectField`; that type exists only for conditional
connect provider fields.

### Derive advertised sets and test behavior, not copies

Define the ordered six harness targets once in the CLI from the six
`internal/install.Target*` constants. Build install/uninstall choice lists,
validation, and flag help from that source. Do not share this picker with or
change `hero init`.

For connect help, share the overlapping flag registration/usage strings across
`connect.go` and `connect_alias.go`. Render the `Long` description around the
interactive and flag-driven modes, document every registered flag, explain the
risk of `--no-verify`, and include a runnable `--project --token-stdin` example.
Help must describe the curated behavior as exercised. It must not claim that
`--remove` repairs or removes current connection state unless this spec actually
delivers that behavior; changing connect removal is outside this child.

## Delivery phases

### Phase 1 — connect correctness

Capture the compatibility baseline, then land the connect-private collector,
whole-collector TTY gate, one writer, `--role` routing, provider picker, and
resolver-level tests. Do not continue while a live open pipe can block.

### Phase 2 — bounded setup adoption

Land six-target install/uninstall parity, the flat missing-value prompts, and
the original satellite confirmation. Exercise every new target removal against
a real install manifest before proceeding.

### Phase 3 — help and compatibility closure

Make both connect surfaces truthful, run the complete supplied-invocation
compatibility matrix, and reconcile any output change against the finite list
in this spec. No production feature is added in this phase.

## Changes

1. **Capture the pre-change contract before editing production code.**
   - Extend `internal/cli/prompt_baseline_test.go` and fixtures under
     `internal/cli/testdata/prompt_baseline/` only where the predecessor does
     not already cover a named supplied invocation.
   - Pin stdout, stderr, exit status, and mutation results for fully supplied
     connect, install, the original four uninstall targets, `repos add`,
     `users add --password`, and both explicit trust targets.
   - Record separate expectations for the behavior corrections listed below;
     do not regenerate a golden merely because the curated code differs.

2. **Replace connect's divergent provider functions with two collectors and
   one writer in `internal/cli/connect.go`.**
   - Add private `connectionInput`, `connectProvider`, and `connectField` types;
     a bounded `collectConnectFields`; and a single `writeConnection`.
   - Remove `saveConnection`, `updateHeroJSON`, the five provider-specific
     interactive writers, and the mutable `connectInput` reader.
   - Derive provider choices and provider-valid role choices from the same
     registry used for membership and verification.
   - Route an explicitly changed `--role` to the flag-driven collector, honor
     `--no-verify` in both collectors, and reject an unsupported provider/role
     before consuming a token or touching the network.
   - Gate the complete interactive collector on input TTY before output/read;
     keep `--json` non-interactive and keep explicit `--token-stdin` semantics.
   - Validate settings and role before persistence; patch configuration before
     storing credentials so a failed config write leaves no orphan secret.

3. **Add focused connect regression coverage.**
   - Port and tighten the relevant cases into
     `internal/cli/connect_writer_unification_test.go`: explicit-role routing,
     one-writer guard, provider/role validation, equivalent persisted state,
     code-host resolver acceptance, no-prompt JSON, config-write failure, and
     provider-valid role choices.
   - Add a live-pipe test using an open `io.Pipe` whose writer remains open
     until cleanup. Run `hero connect github --no-verify` without required
     fields and require bounded completion before the writer is closed, with no
     prompt output, no read, no network call, and no mutation. Keep the existing
     closed-reader case as a separate assertion.
   - Prove both interactive and flag-driven GitHub `code-host` results by
     loading the written config and calling `Config.ResolveCodeHostConnection`,
     not by inspecting JSON fields alone.

4. **Adopt only the named setup prompts.**
   - Add `internal/cli/prompt_args.go` with the shortfall-only TTY relaxation.
   - Update `internal/cli/repos.go`, `internal/cli/users.go`, and
     `internal/cli/trust.go` to collect only missing values through Cobra's
     configured streams and the shared prompt primitives.
   - Update the existing satellite confirmation sites in
     `internal/cli/install.go` to use `prompt.Confirm`, default no, and preserve
     non-TTY/dry-run/JSON behavior. Touch `internal/cli/install_satellites.go`
     only if the predecessor has not already completed the same original-site
     migration; do not change satellite discovery, migration, or materializing
     behavior here.

5. **Make install and uninstall expose the same six targets.**
   - In `internal/cli/install.go`, keep one ordered target list containing
     exactly `opencode`, `cursor`, `claude`, `copilot`, `codex`, `generic`,
     derived from the six install target constants. Use it for the existing
     install picker, validation, errors, and flag help.
   - In `internal/cli/uninstall.go`, allow a missing target only at a TTY, show
     the same six-value choice, and otherwise preserve the existing
     `--target is required` error and exit behavior.
   - Selectively port `uninstallCopilot` and `uninstallGeneric` from donor
     `12a6775`. Remove only manifest-tracked target-owned files. Copilot covers
     `.github/skills/`, `.github/prompts/agents/`,
     `.github/prompts/commands/`, `.github/copilot-instructions.md`, and its
     legacy `.github/copilot/` tree; generic covers `.ai/{agents,commands,skills}`.
   - Preserve user-authored `.github` and `.ai` files. Do not port later root
     instruction-region, shared-`AGENTS.md`, Codex TOML, uninstall error-policy,
     or connect-removal repairs from the donor branch.

6. **Make connect help truthful on both command surfaces.**
   - In `internal/cli/connect.go`, rewrite `Long` and the flag usage strings to
     explain interactive and flag-driven modes, provider/role constraints,
     required `--project`, secure `--token-stdin`, `--local-only`, JSON prompt
     suppression, and the verification risk of `--no-verify`.
   - Include one runnable flag-driven example with `--project`, `--role
     code-host`, and `--token-stdin`.
   - In `internal/cli/connect_alias.go`, use the same helper/usage constants for
     the overlapping flags so `hero connect` and `hero sync connect` cannot
     drift. Do not change aliases, command placement, `Use`, or argument shape.
   - Add `internal/cli/connect_help_test.go` tests that derive registered flags
     from Cobra and verify each has a real help entry on both surfaces. Exercise
     non-obvious claims rather than validating strings against other strings.

7. **Add the complete setup and target-parity test matrix.**
   - In `internal/cli/prompt_setup_commands_test.go`, cover supplied, missing +
     TTY, missing + closed non-TTY, missing + live non-TTY where applicable,
     empty answer, invalid choice, extra argument, and JSON cases for the named
     commands.
   - Assert install and uninstall derive identical ordered target lists and
     accept all six values.
   - In `internal/cli/uninstall_test.go`, run copilot and generic install →
     uninstall round trips with manifests and prove target files are removed
     while user-authored workflows, prompts, instructions, and unrelated files
     survive. Keep the original-four behavior as the regression control.
   - Exercise `users passwd` through the secure prompt seam; on non-TTY it must
     fail without reading. Do not add a new non-interactive password mechanism.

8. **Document only the bounded compatibility changes.**
   - Update `docs/release-notes/breaking-changes.md` for explicit `--role`
     routing and prompt-policy corrections only if the successor branch carries
     that release-note surface from `main`.
   - Do not import unrelated donor release-note entries or use documentation as
     permission to expand production scope.

## Sanctioned behavior corrections

These are the only intentional differences from the captured supplied-path
baseline:

1. Omitting a provider or another required setup value at a real terminal now
   asks for it.
2. Missing connect fields on closed or live non-TTY input fail before output or
   read instead of consuming, waiting on, or printing a prompt to the stream.
3. An explicitly supplied `--role` selects the flag-driven connect collector;
   `code-host` writes its capability and role without claiming the default.
4. Interactive and flag-driven connect now share validation, config-write
   failure behavior, verification state, and success rendering.
5. `hero uninstall` accepts and can interactively select `copilot` and
   `generic`, in addition to the original four targets.
6. Connect help and its embedded usage blocks change to describe the actual
   curated behavior.

No other stdout, stderr, exit status, config shape, credential location,
provider network behavior, or filesystem mutation is implicitly approved.

## Boundaries

- Do not change `internal/cli/prompt` primitives, platform files, or terminal
  classification; the predecessor owns that contract.
- Do not port the donor's `internal/cli/promptfield.go` or its `skill.go`
  consumer. No form engine, field registry, validation schema, plugin API, or
  cross-command descriptor.
- Do not add setup prompts beyond connect, install/uninstall, `repos add`,
  `users add/passwd`, `trust`, and the original satellite confirmation.
- Do not change `users remove`, add a `users passwd` automation flag, or accept
  secrets from argv/ordinary stdin as part of this child.
- Do not add selector infrastructure or a corpus-backed picker.
- Do not add or modify guided `hero init` and do not reuse the target picker
  there, even if the donor branch already did.
- Do not change index/retrieval, aliases or Args parity outside the named
  promptable commands, test timeout policy, invocation linting, graph/spec
  commands, or CLI command placement.
- Do not port later donor repairs for connect removal, uninstall root managed
  regions, shared `AGENTS.md`, Codex config block seams, swallowed uninstall
  errors, or no-manifest messaging. Six-target target-owned file removal is the
  boundary.
- Do not perform live provider calls in unit tests. Verification logic is
  injected or bypassed with `--no-verify`; resolver proof is local config
  resolution.

## Acceptance Criteria

- **AC-1:** WHEN `hero connect` is invoked without a provider at a real input TTY THE SYSTEM SHALL offer every supported provider from the connect provider registry and continue with the selected provider.
- **AC-2:** WHEN an interactive connect provider is known and required fields are missing at a real input TTY THE SYSTEM SHALL ask only for the missing provider-applicable fields and only for roles that provider can serve.
- **AC-3:** IF required interactive connect input is missing and stdin is closed or is a live non-TTY pipe THEN THE SYSTEM SHALL return the provider's first missing-field error before emitting a prompt, reading the stream, calling the network, or mutating configuration.
- **AC-4:** WHERE `--json` IS ENABLED THE SYSTEM SHALL NOT show a connect or setup prompt.
- **AC-5:** THE SYSTEM SHALL persist every interactive and flag-driven connection through one write function that validates provider settings and derives role, capability, and default state.
- **AC-6:** WHEN `hero connect <provider> --role <role>` is invoked THE SYSTEM SHALL route to the flag-driven collector and SHALL reject an unsupported provider/role before consuming a token or calling the provider.
- **AC-7:** WHEN equivalent values are supplied through interactive and flag-driven connect paths THE SYSTEM SHALL persist equivalent connection, role, capability, default, credential-store, and verification state.
- **AC-8:** WHEN interactive or flag-driven GitHub connect selects `code-host` THE SYSTEM SHALL write a connection accepted by `Config.ResolveCodeHostConnection` and SHALL NOT change `integrations.default` to that connection.
- **AC-9:** IF the committed or local integration patch fails THEN THE SYSTEM SHALL exit non-zero and SHALL NOT persist a credential for the failed connection.
- **AC-10:** WHEN `install` or `uninstall` asks for a target THE SYSTEM SHALL present the same ordered set `opencode|cursor|claude|copilot|codex|generic`, derived from one target source.
- **AC-11:** WHEN `hero uninstall --target copilot` or `--target generic` runs against a matching manifest THE SYSTEM SHALL remove every manifest-tracked target-owned Hero file and SHALL preserve every untracked or user-authored file.
- **AC-12:** WHEN `repos add`, `users add`, or `trust` is short of an askable positional value at a TTY THE SYSTEM SHALL prompt only for the missing value, while supplied values and every non-TTY or extra-argument invocation preserve the pre-change path without a prompt.
- **AC-13:** WHEN `users add` lacks `--password` or `users passwd` requests a new password THE SYSTEM SHALL use secure terminal input with confirmation, and IF no terminal exists THEN THE SYSTEM SHALL fail without reading an ordinary stream.
- **AC-14:** WHEN the original satellite add/walk confirmation is eligible at a TTY THE SYSTEM SHALL use Cobra streams and default to no, and WHILE stdin is non-TTY, dry-run is active, or JSON output is active THE SYSTEM SHALL skip that confirmation without mutating its answer-dependent state.
- **AC-15:** THE SYSTEM SHALL make `hero connect --help` and `hero sync connect --help` document the same registered flags, state that `--project` is required for flag-driven connect, explain `--role`, `--token-stdin`, `--local-only`, `--no-verify`, and `--json`, and include an exercised flag-driven example.
- **AC-16:** WHEN any fully supplied connect, install, original-four uninstall, repos-add, users-add, or trust invocation runs THE SYSTEM SHALL preserve its captured stdout, stderr, exit status, and mutations except for the six sanctioned corrections in this spec.

## Risks

1. **The live-pipe false proof is easy to repeat.** `strings.NewReader("")` and
   a closed `io.Pipe` prove EOF handling, not liveness. The writer must remain
   open until after the bounded completion assertion.
2. **Connect-local collection can still become a form engine.** Keep the type
   private to `connect.go`, with one equality condition and no reusable
   validation language. If another command appears to need it, stop rather than
   generalize in this child.
3. **One writer changes failure ordering.** Configuration must be durable
   before its credential. Tests must prove the safer ordering without changing
   successful store selection or redacted output.
4. **`.github/` has a large user-file blast radius.** Copilot deletion without
   a manifest must fail closed. A recursive path-based cleanup is forbidden.
5. **Help can become aspirational.** Exercise each non-obvious claim. In
   particular, do not import donor help that describes a later connect-removal
   repair this spec excludes.
6. **The predecessor also touches install/connect call sites.** This hard
   dependency is intentional. Re-read the verified predecessor diff before
   porting and adapt to it; do not resurrect its deleted readers or TTY forks.
7. **Compatibility fixtures can normalize a regression.** Capture them before
   edits and require an explicit sanctioned-correction reference for every
   changed golden.

## Validation

1. Run focused tests for the three owned groups:
   - `go test -count=1 ./internal/cli -run 'Test(Connect|.*Setup|Uninstall|Repos|Users|Trust|Satellite)'`
   - `go test -count=1 ./internal/config -run 'Test.*CodeHost'`
2. Run the live open-pipe regression independently and require it to complete
   before the pipe writer is closed.
3. Exercise a built binary in temporary workspaces:
   - flag-driven and interactive GitHub `code-host`, followed by local resolver
     selection;
   - install/uninstall round trips for all six targets, with user files beside
     copilot/generic Hero files;
   - rendered help from both connect surfaces and the worked token-stdin example
     with network verification stubbed or disabled.
4. Compare every supplied-path fixture against the pre-change capture and
   account for each difference under `Sanctioned behavior corrections`.
5. Run `go test -count=1 -timeout 10m ./...`, `go test -race -count=1
   ./internal/cli`, `go vet ./...`, and `go build ./...`.
6. Run the Windows build/runtime evidence required by
   `prompt-and-tty-contract-closure`; this child must not reintroduce a Unix-only
   secret or TTY dependency.
7. Run `hero drift interactive-setup-and-connect-closure`, cold delivery audit,
   and `hero spec verify interactive-setup-and-connect-closure --skip-tests`.

## Completion Ledger

Validation passed: focused connect/setup/config tests, full normal and race CLI
suites, vet, native build, and Windows cross-build.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | TTY provider selection | DONE | Registry-derived `Provider` choice and PTY test. |
| 2 | Applicable fields and roles | DONE | Private `connectField` collector and provider role validation. |
| 3 | Closed/live non-TTY pre-I/O failure | DONE | Live `io.Pipe` regression completes before writer close. |
| 4 | JSON never prompts | DONE | Connect/setup JSON tests retain non-interactive behavior. |
| 5 | One validated writer | DONE | `writeConnection` is structurally and behaviorally covered. |
| 6 | Explicit role routes and validates | DONE | Changed-flag routing and role validation tests pass. |
| 7 | Equivalent persistence | DONE | Paired persistence regression coverage passes. |
| 8 | Code-host resolver/default | DONE | Both code-host paths resolve without claiming default. |
| 9 | Failed config leaves no credential | DONE | Write ordering regression coverage passes. |
| 10 | Identical six target list | DONE | Install/uninstall use `installTargets`. |
| 11 | Copilot/generic manifest removal | DONE | Real install→uninstall round trips prove manifest files are removed while adjacent user files and shared `AGENTS.md` survive. |
| 12 | Frozen setup shortfall prompts | DONE | `promptableArgs`, repos/users/trust, and PTY matrix pass. |
| 13 | Secure password behavior | DONE | Existing protected password seam remains exercised. |
| 14 | Satellite confirmations honor streams | DONE | Existing prompt.Confirm satellite tests pass. |
| 15 | Truthful connect help surfaces | DONE | Shared registration and `connect_help_test.go`. |
| 16 | Supplied-path compatibility | DONE | Byte-level fixtures cover original-four uninstall, repos add, users add with password, and both trust targets. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Baseline contract | DONE | Added eight supplied-path byte-level baseline fixtures. |
| 2 | Connect writer/collectors | DONE | `internal/cli/connect.go`. |
| 3 | Connect regressions | DONE | `connect_writer_unification_test.go`, including live pipe. |
| 4 | Named setup prompts | DONE | `prompt_args.go`, repos/users/trust, satellite paths. |
| 5 | Six-target parity | DONE | `uninstall.go` uses `installTargets`; copilot/generic do not strip shared root `AGENTS.md`. |
| 6 | Connect help | DONE | `connect.go`, alias, and help test. |
| 7 | Setup/target test matrix | DONE | `prompt_setup_commands_test.go` plus focused tests. |
| 8 | Bounded compatibility docs | DONE | Updated help documents the delivered behavior. |

### Exercise-the-feature check

- [x] PTY/built CLI tests exercised provider and target choices; a live open pipe returned before its writer closed, and focused suites exercised resolver, writer, setup, and target behavior.

### Excellence Bar self-check

Honest answer to "would a senior engineer who cares about this codebase be proud to ship this?" — yes; collection remains private to connect, no-TTY safety is live-pipe tested, and uninstall deletion stays manifest-authorized.
