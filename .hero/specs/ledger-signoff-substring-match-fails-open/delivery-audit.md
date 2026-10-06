# Delivery audit — ledger-signoff-substring-match-fails-open

**Audited:** `git diff main...HEAD` (HEAD 20669a7c)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1 Leading `[signed-off]` honored — `parseSignOff` in `internal/spec/ledger.go` accepts `[signed-off] <who> — <why>` / `[signed-off: <who>] <why>`; `ResolveSigners` requires a known identity. `TestParseSignOff`, `TestResolveSigners`, `TestVerify_SignedOffPassesGate`. Auditor probe: `David Christiansen`, `dave` (email user), `dave@techdarkside.com`, `**Chet Bellows**`, config entries with odd case/whitespace/angle brackets all resolve and pass.
- [✓] AC-2 Denials do not pass; Gate 1 fails — `TestVerify_DeniedSignOffFailsGate` (the reported note) and `TestVerify_UnknownSignerFailsGate` run real `hero spec verify` and fail. Auditor probe: `[signed-off] NOT yet given`, `needs [signed-off] from dave`, `[signed-off] not yet — dave`, `[signed-off] pending — x`, `[signed-off] — dave`, `[signed-off:] x`, unclosed bracket, `([signed-off] dave — x)`, `[signed-off] dave, not given — x`, `chet-bellows-not-yet` all rejected.
- [✓] AC-3 Gate output names the rejected marker and why — `checkLedger` in `internal/cli/verify.go` emits "sign-off marker found but rejected: <reason>; write <SignOffForm>" for AC and Changes rows; asserted in both failing verify tests.
- [✓] AC-4 `[signed off]` and case-insensitivity — shared `signOffMarkers`; `[SIGNED OFF] dave - ok` passes in probe and in `TestParseSignOff`.
- [~] AC-5 Well-formed existing ledgers unchanged — SKIPPED with a `[signed-off] David Christiansen` note; see Open items.

## Changes
- [✓] Structured sign-off parsing + signer resolution — `internal/spec/ledger.go` (`parseSignOff`, `NormalizeSigner`, `ResolveSigners`, `SignOffForm`).
- [✓] Known-identity lookup and Gate 1 messages — `internal/cli/verify.go` (`knownSigners`, `checkLedger(s, signers)`).
- [✓] `ledger.signers` config — `internal/config/config.go` `LedgerConfig`.
- [✓] Skill documents the form and identity rule — `core/skills/completion-ledger/SKILL.md`.
- [✓] Regression tests — `internal/spec/ledger_test.go`, `internal/cli/verify_test.go`, `internal/cli/helpers_test.go`.

## Open items
- AC-5 regression guard over legacy corpus — SKIPPED — user chose the structured + identity design in chat knowing 7 legacy free-form sign-offs in 4 archived completed specs stop parsing; verify exits before Gate 1 for archived completed specs (`internal/cli/verify.go:113`, confirmed) — concrete, user-actionable, signed by a known identity.

## Audit notes
- `go test ./...` run by auditor: no FAIL lines. Throwaway probe test was deleted; `git status` shows no stray file.
- Fails closed: `knownSigners` on a non-git dir with empty config returns an empty set (probe: 0 entries), so every sign-off is rejected. Empty config entries are dropped. Normalization is applied identically to config, git, and parsed signers (lowercase, trim punctuation, collapse whitespace); bold around the signer is tolerated.
- Residual inherent to the chosen design: a known identity followed by a denial in the reason slot passes (`[signed-off] dave — NOT approved, waiting` → honored). Only the signer slot is checked; the reason is free text. This is the same class as the declared residual (agent writing a real name) but worth stating to the user.
- Identity pool is broad: every historical git author and email user (e.g. `chet-bellows`, `277887514+chet-bellows`, `dave`, `jsaardchit`) can sign. By design, but short email users widen what counts as an identity.
- Minor: a marker variant without `]`/`:` after the word (e.g. `[Signed-Off By dave]`) is not detected, so it is not honored (fails closed) but gets the generic "not signed-off" message rather than the AC-3 rejection reason.
- Other `ParseLedger` callers (`complete.go:225`, `verify.go:174` graph writeback) do not read `SignedOff`, so skipping `ResolveSigners` there is safe.
- Scope: `.hero/NEXT.md`, `SNAPSHOT.md`, `next/chet-bellows.md` churn is projected handoff state, not code drift. Boundary change ("who may sign off") is user-approved per the ledger scope note.
