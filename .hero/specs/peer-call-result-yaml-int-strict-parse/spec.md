---
title: "peer call result fails YAML unmarshal when subagent emits tilde-approximated ints (e.g. tokens: ~22000)"
slug: peer-call-result-yaml-int-strict-parse
type: bug
status: completed
severity: high
priority: P1
created: 2026-05-19
tags: [peering, yaml, contracts, robustness, subagent]
root_cause_class: design
relations:
  - target: cross-repo-peering
    kind: child-of
---

# peer call result fails YAML unmarshal when subagent emits tilde-approximated ints (e.g. `tokens: ~22000`)

## Kickoff

**Status:** delivered — `ApproxInt` type added, four budget fields swapped, `PeeringContractsVersion` bumped 1→2, tests green.

**Shipped:**
- `contracts/peering/peercall.go` — added `ApproxInt` type with `UnmarshalYAML`/`MarshalYAML`/`UnmarshalJSON`/`MarshalJSON`. Accepts plain int, `~N`, float (truncates), quoted variants, empty/null → 0. Rejects negatives.
- Both `BudgetSpec.Turns/Tokens` and `BudgetConsumed.Turns/Tokens` now `ApproxInt` (underlying `int`, so existing `%d` formatting and zero comparisons keep working).
- `contracts/peering/version.go` — `PeeringContractsVersion` 1 → 2.
- `internal/cli/peer.go:354-356` — one-line conversion at the constructor (`peerCallBudgetTurns` is `int`).
- `contracts/peering/peercall_test.go` — new test file (package previously had none) covering YAML and JSON unmarshal forms + canonical-int marshal round-trip + negative-rejection.
- `internal/peering/peercall_test.go::TestParseResultBlock` — extended happy path with `tokens: ~1842` (the canonical bug-report scenario) and added a `tolerant budget forms` subtest covering all accepted input variants.

**Verified:** `go build ./...`, `go test ./contracts/peering/... ./internal/peering/... ./internal/cli/...`, `go vet ./...` all clean.

**Pick up at:** done — archive with `hero spec complete` and move to the sibling truncation/persistence fix.

## Issue

Reporter: **hero-code** (sibling peer repo), via `hero peer call hero --mode=advisory` while probing `hero-pm-ui` prerequisites.

Reported as: "YAML parse failure on the result envelope when the peer subagent's response contained `~22000` (tilde-prefixed token estimate) where an int was expected — `peercall.go` unmarshal at line 207. Worth a defensive int parser or accepting strings."

Reporter line cite is **off** — line 207 in `internal/peering/peercall.go` is the `events.log` `AppendEvent` for `EventCallInvoked`. The actual `yaml.Unmarshal` site is `internal/peering/peercall.go:419`, inside `parseResultBlock`. The drift is fine — same call path, same failure surface.

This bug is the same call path used by `--mode=advisory` AND `--mode=spec-out`. Both modes route through `parseResultBlock`. The failure mode is identical in both: `yaml.Unmarshal` errors → `parseResultBlock` returns wrapped error → `Call` aborts at line 254 with `fmt.Errorf("parse subagent result: %w", parseErr)` → no `Result` is returned, no trail entry written, no `peer.call.completed` event with a real result kind (only an error-message event). The investment in turns/tokens consumed by the subagent is lost; the caller sees "parse subagent result: unmarshal result block: …" and has to retry.

No tracker configured — this spec is the sole tracking artifact.

## Investigation

### End-to-end flow

1. Caller invokes `peering.Call(projectRoot, opts)` (`internal/peering/peercall.go:127`).
2. After validation and envelope rendering, `Call` exec's the subagent in the peer's workspace and captures its stdout (`runSubagent`, line 239).
3. Stdout is passed to `parseResultBlock(stdout)` (line 253).
4. `parseResultBlock` (line 408-423):
   - Locates the `<peer-call-result>...</peer-call-result>` fence via `resultFenceRE` (regex at line 44).
   - Trims the captured YAML body.
   - Calls `yaml.Unmarshal([]byte(body), &out)` at **line 419** into a `contractpeering.PeerCallResult`.
   - On error, wraps and returns: `fmt.Errorf("unmarshal result block: %w", err)`.
5. `Call` propagates the parse error and exits, emitting only a `peer.call.completed` event whose message is `"peer call result unparseable: ..."`. No `Result` ever reaches the caller.

### Target type and the strict-int fields

`contracts/peering/peercall.go`:

```go
// Lines 22-26
type BudgetSpec struct {
    Turns  int `yaml:"turns,omitempty" json:"turns,omitempty"`
    Tokens int `yaml:"tokens,omitempty" json:"tokens,omitempty"`
}

// Lines 28-32
type BudgetConsumed struct {
    Turns  int `yaml:"turns" json:"turns"`
    Tokens int `yaml:"tokens" json:"tokens"`
}
```

`PeerCallResult.BudgetConsumed` (line 114) embeds `BudgetConsumed`. `PeerCallRequest.Budget` (line 56) embeds `BudgetSpec`.

`gopkg.in/yaml.v3` decodes the literal `~22000` as a YAML node of kind scalar, tag `!!str` (the tilde is not a magic sigil in YAML the way it is in TOML or some env-var DSLs; `~` alone is null, but `~22000` parses as a plain string). Strict-int target → type-mismatch error: `cannot unmarshal !!str ` … `into int`.

### Why the subagent emits `~22000`

The envelope template (`renderEnvelope`, lines 549-568) instructs the subagent to emit:

```yaml
budget_consumed:
  turns: <actual>
  tokens: <actual>
```

The template does not constrain the *form* of `<actual>`. A subagent reporting "I used roughly 22,000 tokens" naturally writes `~22000` — Claude-class models routinely produce tilde-approximated estimates when the exact count is unknown to them (they do not have a reliable post-hoc token counter in stdin/stdout-only mode). The template should either:

- Constrain the form explicitly (`tokens: 22000  # integer, no tilde, no commas`), or
- Accept the natural form on the parsing side.

The second is more robust because we cannot prevent subagents from drifting on a templated instruction over time; defensive parsing keeps the envelope contract honest under realistic LLM behaviour.

### Blast radius

- **Call modes affected:** advisory and spec-out (both v1). full mode is rejected upstream at line 134.
- **Fields affected:** `BudgetConsumed.Turns`, `BudgetConsumed.Tokens` (output, primary), and by symmetry `BudgetSpec.Turns`, `BudgetSpec.Tokens` (input, if a subagent ever echoes the request back through a result-shaped block).
- **What gets lost on failure:** the entire parsed `Result` — `Kind`, `Findings`, `SpecSlug`, `PeerStatus`, `CommitRef`, `PRURL`, `Error` (irony: even the peer's report of its own error is lost), and `At`. The originator-side trail entry (`recordOriginatorSide`, line 280) never runs, so no record of the peer's actual output lands on the related spec. Only the events.log gets a "result unparseable" entry, which is not surfaced anywhere the user normally reads.
- **Frequency in practice:** high. Token estimates are the single most common approximation field a subagent fills in; subagents have no precise counter for tokens consumed during their own run. Once `--mode=spec-out` traffic ramps up, every long-running call is a candidate.
- **Recovery cost:** subagent runs cost real money/tokens. A failed parse forces a re-run with the same prompt, with no guarantee the next subagent run won't drift again.

### Root cause classification

**Design** — contract-too-strict. The bug is not in `yaml.Unmarshal` (working as documented), nor in `parseResultBlock` (faithfully bubbles the error). The bug is that `BudgetConsumed.Tokens` is declared as `int` when the realistic input domain — LLM-emitted budget estimates — is "non-negative integer approximation, possibly tilde-prefixed, possibly quoted, possibly missing". The contract under-models its actual input space.

Not a **code** bug because no logic is wrong; not a **race** bug; not **external** because the subagent is part of our system and we control its instructions; not **data** in the corrupt-input sense — `~22000` is well-formed YAML and a perfectly reasonable estimate format; not **user** since the user never types this field.

### Contract version implications

`PeeringContractsVersion = 1` (`contracts/peering/version.go:24`). The bump rule (lines 14-19): "every breaking change to any exported type, field, or method signature under contracts/peering/... increments this constant by one. Adding a new field with a zero-value-safe default is not a breaking change. Removing or renaming a field is."

Changing `Turns int` → `Turns ApproxInt` is a **breaking change to a field type**. However:

- The YAML wire shape is identical (still an integer literal on output).
- The JSON wire shape must remain identical (verify: `ApproxInt` must implement `MarshalJSON` that emits a plain int, not an object).
- Downstream Go consumers (`internal/cli/peer.go:380-383` reads `res.Result.BudgetConsumed.Turns` / `.Tokens`) are affected: an `int` can no longer be passed directly to `fmt.Printf("%d", ...)` if `ApproxInt` is a struct. Best path: declare `ApproxInt` as `type ApproxInt int` (named integer type), which lets `%d` formatting continue to work and keeps the value an addressable integer.

Because `ApproxInt` is `type ApproxInt int`, downstream call sites compile with at most a trivial conversion. The named type is the contract; the underlying `int` is the wire/runtime value. **Verdict:** roll `PeeringContractsVersion` from 1 → 2. Even if the wire format is unchanged, the Go-symbol surface is — and peers calling our manifest see the type name change. Conservative bump is correct here; the cost is one constant edit.

(Alt: keep `int` and override unmarshal at the *container* level via a custom `UnmarshalYAML` on `BudgetSpec` and `BudgetConsumed` that pre-processes node values. Rejected — more code, harder to test, and doesn't help the JSON path if we ever need it there.)

### Test fixtures and recorded envelopes

Searched `internal/peering/`, the root tree, and the contracts package for `testdata/` directories or YAML fixture files referencing `budget_consumed` — none found. The only test that exercises this path is `TestParseResultBlock` (`internal/peering/peercall_test.go:100-159`), which uses inline string fixtures with plain ints. Those tests stay valid (`turns: 4` is still a valid `ApproxInt`); we need to extend the table with the new forms.

No recorded peer-call envelopes on disk in this workspace — the events.log doesn't preserve the raw result block.

### Why not a result-level fallback unmarshal?

Considered: in `parseResultBlock`, on error, fall back to unmarshaling into a `map[string]any` and salvage what we can. **Rejected.** That swallows real shape bugs — a malformed `kind:` or a missing fence would silently produce a `Result{}` zero value and Call would record a successful peer call with no data. We want shape errors to fail loudly; what we want is for *valid input variants of the budget field* to succeed quietly. Field-level tolerance is the right granularity.

## Root cause

`BudgetConsumed.Turns`, `BudgetConsumed.Tokens`, `BudgetSpec.Turns`, `BudgetSpec.Tokens` (all declared as plain `int` at `contracts/peering/peercall.go:24-25, 30-31`) cannot decode any of the natural forms a peer subagent emits for an approximate budget estimate: tilde-prefixed (`~22000`), float (`22000.0`), or quoted (`"22000"`). When `yaml.Unmarshal` rejects the form, `parseResultBlock` returns an error and the entire peer-call result envelope is discarded — including findings, kind, and spec_slug fields that have nothing to do with budget reporting. The contract under-models its actual input domain.

## Severity

**High.** Affects both supported peer-call modes (advisory and spec-out). High likelihood — `~N` is the natural shape for LLM-emitted estimates. Loss is total per occurrence: every field of the result is discarded, not just the offending one. No workaround for end users; only "re-run and hope". Real cost in subagent spend per failed call. Severity is bounded only by the fact that peer calls are an early-adopter feature today; once cross-repo peering ramps, this becomes a daily papercut.

## Suggested Fix Approach

### Change 1 — introduce `ApproxInt` in `contracts/peering/peercall.go`

**File:** `contracts/peering/peercall.go`

**Before** (lines 22-32):

```go
// BudgetSpec caps how much work a peer call may consume.
type BudgetSpec struct {
    Turns  int `yaml:"turns,omitempty" json:"turns,omitempty"`
    Tokens int `yaml:"tokens,omitempty" json:"tokens,omitempty"`
}

// BudgetConsumed records actual consumption after the call returns.
type BudgetConsumed struct {
    Turns  int `yaml:"turns" json:"turns"`
    Tokens int `yaml:"tokens" json:"tokens"`
}
```

**After** (add new type above; mutate fields):

```go
// ApproxInt is a non-negative integer field that tolerates a few
// approximation forms peers (especially LLM subagents) naturally
// emit when reporting estimates:
//
//   - plain int:        22000        → 22000
//   - tilde-prefix:     ~22000       → 22000   (the "~" is dropped)
//   - float form:       22000.0      → 22000   (truncated)
//   - string forms of any of the above
//   - empty / missing → 0
//
// Round-trips back to a plain integer (no "~" preservation — the
// tilde is an *input tolerance*, not an output format). Wire shape
// is `int` for both YAML and JSON.
//
// Negative values are rejected at unmarshal time to keep the
// "budget consumed" semantics honest.
type ApproxInt int

// Int returns the underlying integer value. Convenience for callers
// that want to be explicit; equivalent to int(a).
func (a ApproxInt) Int() int { return int(a) }

// UnmarshalYAML accepts !!int, !!float, and !!str scalars whose
// string form parses as a (possibly tilde-prefixed) integer.
func (a *ApproxInt) UnmarshalYAML(node *yaml.Node) error {
    if node == nil || node.Kind != yaml.ScalarNode {
        return fmt.Errorf("approxint: expected scalar, got kind=%d", node.Kind)
    }
    s := strings.TrimSpace(node.Value)
    s = strings.TrimPrefix(s, "~")
    s = strings.Trim(s, `"'`)
    if s == "" {
        *a = 0
        return nil
    }
    if n, err := strconv.Atoi(s); err == nil {
        if n < 0 {
            return fmt.Errorf("approxint: negative value %d", n)
        }
        *a = ApproxInt(n)
        return nil
    }
    if f, err := strconv.ParseFloat(s, 64); err == nil {
        if f < 0 {
            return fmt.Errorf("approxint: negative value %g", f)
        }
        *a = ApproxInt(int(f))
        return nil
    }
    return fmt.Errorf("approxint: cannot parse %q as integer", node.Value)
}

// MarshalYAML emits a plain integer. The "~" form is input tolerance
// only; outputs are always canonical.
func (a ApproxInt) MarshalYAML() (any, error) { return int(a), nil }

// UnmarshalJSON applies the same tolerance as YAML so symmetric
// transports stay honest. Accepts JSON numbers, JSON strings of the
// same shape, and null → 0.
func (a *ApproxInt) UnmarshalJSON(data []byte) error {
    s := strings.TrimSpace(string(data))
    if s == "" || s == "null" {
        *a = 0
        return nil
    }
    // Strip surrounding quotes if present.
    if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
        s = s[1 : len(s)-1]
    }
    s = strings.TrimPrefix(s, "~")
    if s == "" {
        *a = 0
        return nil
    }
    if n, err := strconv.Atoi(s); err == nil {
        if n < 0 {
            return fmt.Errorf("approxint: negative value %d", n)
        }
        *a = ApproxInt(n)
        return nil
    }
    if f, err := strconv.ParseFloat(s, 64); err == nil {
        if f < 0 {
            return fmt.Errorf("approxint: negative value %g", f)
        }
        *a = ApproxInt(int(f))
        return nil
    }
    return fmt.Errorf("approxint: cannot parse %q as integer", string(data))
}

// MarshalJSON emits a canonical integer.
func (a ApproxInt) MarshalJSON() ([]byte, error) {
    return []byte(strconv.Itoa(int(a))), nil
}

// BudgetSpec caps how much work a peer call may consume.
type BudgetSpec struct {
    Turns  ApproxInt `yaml:"turns,omitempty" json:"turns,omitempty"`
    Tokens ApproxInt `yaml:"tokens,omitempty" json:"tokens,omitempty"`
}

// BudgetConsumed records actual consumption after the call returns.
type BudgetConsumed struct {
    Turns  ApproxInt `yaml:"turns" json:"turns"`
    Tokens ApproxInt `yaml:"tokens" json:"tokens"`
}
```

Imports added at the top of `contracts/peering/peercall.go`:

```go
import (
    "fmt"
    "strconv"
    "strings"
    "time"

    "gopkg.in/yaml.v3"
)
```

**Why:** The contract now models its real input space. `ApproxInt`'s underlying type is `int`, so call sites that read these fields with `%d` formatting continue to work. The wire shape on output is unchanged. The wire shape on input expands to cover the natural forms LLM subagents emit. Negative values are rejected — `budget_consumed` is monotonically non-negative by definition; surfacing nonsense quietly would mask other bugs.

Note: adding the `yaml.v3` import inside the `contracts/peering` package is consistent with the leaf-only rule in `contracts/peering/version.go` — that rule forbids *intra-repo* imports, not stdlib or third-party. `gopkg.in/yaml.v3` is already in `go.mod`.

### Change 2 — bump `PeeringContractsVersion`

**File:** `contracts/peering/version.go`

**Before** (line 24):

```go
const PeeringContractsVersion = 1
```

**After:**

```go
const PeeringContractsVersion = 2
```

**Why:** the Go-symbol shape of `BudgetSpec` and `BudgetConsumed` changed (`int` → `ApproxInt`), which is observable to any peer's contract-import scanner. The wire format is unchanged, but Go consumers see a type name change. Conservative bump per the rule in `version.go` lines 14-19. Required test updates: `TestPeeringContractsVersion`-style assertions if any (none found in the current tree, but `hero check` may surface a peer-manifest drift on next index — that is expected and correct).

### Change 3 — confirm `internal/cli/peer.go` printing still works

**File:** `internal/cli/peer.go`

**Before** (lines 380-383):

```go
if res.Result.BudgetConsumed.Turns != 0 || res.Result.BudgetConsumed.Tokens != 0 {
    fmt.Printf("budget: turns=%d tokens=%d\n",
        res.Result.BudgetConsumed.Turns, res.Result.BudgetConsumed.Tokens)
}
```

**After:** unchanged. Because `ApproxInt` has underlying type `int`, `%d` continues to format correctly via `fmt`'s reflection of the underlying kind. Verify with `go vet ./...`.

**Why:** No-op — proves the underlying-type choice (`type ApproxInt int`) keeps the change surgical at call sites.

Same applies to `internal/cli/peer.go:354`:

```go
Budget: contractpeering.BudgetSpec{
    Turns:  contractpeering.ApproxInt(opts.budgetTurns),
    Tokens: contractpeering.ApproxInt(opts.budgetTokens),
},
```

A one-line conversion is needed where ints are passed as constructor values. (Read the existing call site to confirm — if `opts.budgetTurns` is already `int`, the explicit conversion is required because Go does not coerce named integer types implicitly.)

### Change 4 — confirm `internal/peering/peercall.go::applyBudgetDefaults` still compiles

**File:** `internal/peering/peercall.go`

**Before** (lines 308-326):

```go
func applyBudgetDefaults(mode contractpeering.PeerCallMode, b contractpeering.BudgetSpec) contractpeering.BudgetSpec {
    switch mode {
    case contractpeering.PeerCallAdvisory:
        if b.Turns == 0 {
            b.Turns = DefaultAdvisoryTurns
        }
        if b.Tokens == 0 {
            b.Tokens = DefaultAdvisoryTokens
        }
    ...
```

`DefaultAdvisoryTurns` etc. are untyped int constants (lines 30-33), so they assign to `ApproxInt` fields without an explicit conversion. The zero-comparison `b.Turns == 0` continues to work because `0` is an untyped constant. **No source change needed** — but verify by building. If the constants ever become explicitly typed `int`, conversions become necessary.

### Change 5 — keep `runSubagent` env vars correct

**File:** `internal/peering/peercall.go`

**Before** (lines 375-376):

```go
fmt.Sprintf("HERO_PEER_CALL_BUDGET_TURNS=%d", budget.Turns),
fmt.Sprintf("HERO_PEER_CALL_BUDGET_TOKENS=%d", budget.Tokens),
```

**After:** unchanged. `%d` formats `ApproxInt` (underlying `int`) correctly. Verified mentally; verify with build.

### Change 6 — explicit instruction in the envelope (defense in depth, optional)

**File:** `internal/peering/peercall.go::renderEnvelope` (lines 564-566)

**Before:**

```go
b.WriteString("budget_consumed:\n")
b.WriteString("  turns: <actual>\n")
b.WriteString("  tokens: <actual>\n")
```

**After:**

```go
b.WriteString("budget_consumed:\n")
b.WriteString("  turns: <actual integer count>\n")
b.WriteString("  tokens: <actual integer count; approximate forms like \"~22000\" are accepted>\n")
```

**Why:** documents the now-tolerant contract directly in the envelope the subagent reads. Lowers the rate at which subagents emit drifted forms in the first place, while keeping the tolerant parser as the safety net. Optional — the parser is the real fix.

## Test Plan

### Existing test review

- `internal/peering/peercall_test.go::TestParseResultBlock` (lines 100-159) — covers happy paths for findings and spec-ref result kinds, plus the missing-fence error. After the fix, the existing assertions still hold (`turns: 4`, `tokens: 1842` decode to `ApproxInt(4)` / `ApproxInt(1842)` which compare equal to int constants because the underlying type is int — confirm with build).
- `internal/peering/peercall_test.go::TestBudgetDefaults` (lines 162-178) — covers default-budget application logic. The `b.Turns != DefaultAdvisoryTurns` comparison continues to work because untyped-int constants compare against `ApproxInt` cleanly.
- No tests currently exist for the `BudgetSpec` / `BudgetConsumed` YAML round-trip in isolation.

### Test changes needed

1. **New test:** `internal/peering/peercall_test.go::TestParseResultBlock_TolerantBudget` — table-driven test covering every accepted input form for `budget_consumed.tokens`. Each row asserts a successful parse and the expected `ApproxInt` value.

   Table rows:
   - `tokens: 22000` → 22000
   - `tokens: ~22000` → 22000
   - `tokens: 22000.0` → 22000
   - `tokens: 22000.7` → 22000 (truncation, not rounding — locked behaviour)
   - `tokens: "22000"` → 22000
   - `tokens: "~22000"` → 22000
   - `tokens:` (empty / missing key) → 0
   - `tokens: 0` → 0
   - Negative form `tokens: -1` → error (asserts the non-negative invariant)
   - Non-numeric `tokens: lots` → error
   - Same matrix for `turns:` to lock symmetric behaviour.

   Each row constructs a minimal fenced stdout containing the budget block and a stub `kind: findings` so the fence regex matches.

2. **New test:** `contracts/peering/peercall_test.go::TestApproxInt_MarshalRoundtrip` — a unit test in the contracts package itself that round-trips `ApproxInt(22000)` through YAML and JSON and asserts the output is `22000` (no quotes, no tilde). Locks the marshal-back-as-plain-int behaviour.

   (If no `contracts/peering/*_test.go` exists today, create the file. The package currently has no tests — confirmed via `ls contracts/peering/`. Adding the first test in this package is acceptable; the leaf-only rule doesn't preclude tests.)

3. **New test:** `contracts/peering/peercall_test.go::TestApproxInt_RejectsNegative` — `~-1`, `-1`, `"-1"` all return an error rather than silently zeroing or wrapping.

4. **Extend:** `internal/peering/peercall_test.go::TestParseResultBlock` "findings" subtest — change one of the existing fixture lines from `tokens: 1842` to `tokens: ~1842` and assert the parsed value is 1842. Locks the canonical bug-report scenario as a regression test directly inside the existing happy-path test.

5. **Build / vet:** add `go vet ./...` to the validation step to catch any `%d`-on-non-int formatting drift introduced inadvertently elsewhere.

### Regression scope

- **Manifest round-trip:** `internal/peering/manifest.go` and `contract_imports_test.go` exercise `yaml.Marshal` on `PeerManifest`, which transitively contains no `BudgetConsumed` / `BudgetSpec` (manifests are not call records). No expected impact. Confirm by running the full `internal/peering/...` test suite.
- **Events log:** `contracts/peering/events.go::CallEvent.BudgetConsumed` (line 66) — uses the same type. JSON round-trip of `CallEvent` continues to work via the new `MarshalJSON`/`UnmarshalJSON` on `ApproxInt`. Add a smoke test if no existing test covers CallEvent JSON.
- **Wire compat with hero-code (reporter):** verify hero-code consumes `budget_consumed` only as a display value (it reports the field, doesn't reason about it). If hero-code parses the JSON envelope independently, a `"22000"`-as-string output would be a regression. Our change emits plain ints on output (canonical), so consumers see no wire-shape change — only inputs are now more permissive. Safe.
- **Contract version bump:** consumers of `PeeringContractsVersion` (`contracts/peering/peercall.go:38, 87, 185, 228, 272` and the manifest version field) compare ints; bumping 1→2 may cause peer-manifest drift on next `hero index`. That is the intended signal of a breaking-shape change — not a bug to suppress.

### Manual verification

Once the unit tests pass, an end-to-end check from a real `hero peer call`:

1. Pick any registered peer (e.g., a `hero-code` sibling if available).
2. Run `hero peer call <alias> --mode=advisory --reason "ApproxInt regression check" "What's your favorite number?"`.
3. Observe the post-call printed budget line: `budget: turns=N tokens=M` with integer N and M.
4. Confirm `.hero/events.log` carries a `peer.call.completed` entry with `kind=findings`, not `peer call result unparseable`.

A more targeted check: drop a known-tolerant fixture into a fake subagent (script that just echoes a `<peer-call-result>...</peer-call-result>` block containing `tokens: ~22000`) and run `hero peer call --subagent.command=./fake-sub`. This avoids burning real subagent spend during the verification loop.

## Boundaries

- Not addressing the truncation / persistence concern that is being investigated in parallel as a sibling spec — that spec covers result loss when the subagent's output is well-formed but oversized, and result persistence beyond the trail entry. The two specs are related (both concern result robustness) but the fixes are independent: the tolerant-int change here is purely shape, and lands without coordination.
- Not changing the envelope template substantively (Change 6 is a small clarification, optional). The fix is in the parser, not the prompt.
- Not generalising `ApproxInt` to a workspace-wide tolerant-int — scope is limited to the peering contract. If similar drift turns up in other contracts (e.g., spec frontmatter ints), file a separate spec.
- Not collapsing `BudgetSpec` and `BudgetConsumed` into a single type. They have different YAML tags (`omitempty` differs), different semantics (request cap vs. observed consumption), and reasonable independence even if their field sets are currently identical.

## Risks

- **Contract version bump may surface peer-manifest drift in CI:** any peer that has cached `PeeringContractsVersion=1` will register the change. This is expected; the manifest re-publish is the audit signal the bump is for.
- **Tests that compare `BudgetConsumed.Turns` to a literal `int` keep working** because `ApproxInt`'s underlying type is `int` and untyped constants coerce. If any test uses a typed `int` variable (`var got int = res.Result.BudgetConsumed.Turns`), it will need an explicit cast. Grep before landing: `rg "BudgetConsumed\.(Turns|Tokens)" --type=go` and audit each site.
- **JSON consumers that expect string output:** none known. We emit plain ints both before and after. Verified by reading the contract — all JSON tags are plain `int`-shaped, no string-tag drift.
- **Other approximation forms drift in:** future subagent versions might emit `"~22k"`, `"22,000"`, or scientific notation. Out of scope for this fix; if they appear in real traffic, extend the parser. The current accept-set (`~N`, float, quoted) covers the observed report and the most likely near-term variants.

## Needs more research?

No. Root cause is confirmed against the code as it stands, fix path is concrete, all dependent files have been read end-to-end, and the test plan is sufficient. The only deferred validation is the manual end-to-end peer call, which depends on a working sibling peer environment and is properly a delivery-time check.
