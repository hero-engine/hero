---
title: "Cross-machine handoff loads empty when the local user slug differs between machines"
slug: cross-machine-handoff-slug-mismatch
type: bug
status: completed
severity: high
priority: high
domain: engineering
created: 2026-06-04
origin: session
root_cause_class: design
relates-to:
  - resume-brief-surfaces-handoff
  - e2e-handoff-continuity
  - next-as-projection
completed_at: 2026-06-04T17:55:55Z
---

# Cross-machine handoff loads empty when the local user slug differs between machines

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — silent, total failure of Hero's core cross-machine promise; lands on the just-shipped resume-brief-surfaces-handoff path; no error surfaced. |
| **Ease of Fix** | moderate — the fix is localized to identity resolution at the ingest/read boundary, but must be done without re-keying existing repos or breaking the same-machine path. |
| **Caused by our codebase?** | Yes — `nextUserSlug` derives identity from volatile local state (`git config user.name` → `$USER`), not from the committed/on-disk truth that travels with the repo. |
| **Needs more research?** | No — root cause confirmed against source at file:line; reproduction is real (built binary this session) and the keying mismatch is proven in code. |

### Background
Hero's handoff "magic" promises that a fresh session on another machine "just knows what's going on": finish a turn → context captured to the graph and projected to `.hero/next/<user>.md`; that file travels via git (the graph DB is gitignored and does NOT travel); on the new machine `hero next ingest` rehydrates the graph and `hero resume` surfaces the context. This loop silently produces an EMPTY handoff whenever the user slug derived locally on the second machine differs from the slug baked into the committed handoff file — which happens routinely when git `user.name` differs (or is absent) between machines.

### Analysis
The handoff content is keyed in the graph by a `(user, repo, domain)` triple. The `user` component is a slug derived at runtime from local git/OS config via `nextUserSlug(cfg)` → `gitUserName()` → `gitutil.UserName()`. That derivation is **not stable across machines**:

- Machine A: `git config user.name "BW"` → slug `bw`. The handoff file is named `bw.md`, its frontmatter records `user: bw`, and the graph nodes are keyed `bw:engineering`.
- Machine B (clone, `user.name` unset, only `user.email` set): `gitutil.UserName()` **never consults email**; it falls through to `$USER` (e.g. `bwheeler`). `nextUserSlug` returns `bwheeler`.
- `hero next ingest` on B correctly keys the rehydrated nodes under the **file's frontmatter** `user:` ("bw") — so the graph now holds `bw:engineering` nodes.
- `hero resume` on B queries the handoff singletons under the **locally-derived** slug `bwheeler:engineering`. No match. The "Where you left off" section is empty. No error, no warning.

The mismatch is a **read-time identity-derivation** divergence: ingest writes under the file's recorded identity; resume reads under a freshly re-derived local identity. They only agree when both machines derive the same slug.

> Note on the original report's mechanism: the report attributed B's slug to an email-derived "bdwheeler". In source, `gitutil.UserName()` (internal/gitutil/gitutil.go:25) does **not** derive from email at all — the precedence is `git config user.name` → `$USER` → `$USERNAME` → `"unknown"`. The slug B actually derives comes from `$USER`, not email. The root cause is unchanged: the slug is volatile local state that differs across machines. This was verified on the live repo (`git config user.name` = `chet-bellows`, `$USER` = `bwheeler` — two different slugs on the same checkout).

### Root Cause
**Design.** Handoff identity is derived from volatile local environment (git `user.name`, falling back to `$USER`) rather than from the durable, committed truth that travels with the repository. The write/projection side and the ingest side both anchor identity to the **on-disk file** (filename + frontmatter `user:`), but the read side (`hero resume` → `digest.handoffSection`) re-derives identity locally and queries with it. When the local derivation disagrees with the file's recorded identity, the read finds nothing — and because a clean miss is treated as "no handoff yet," the failure is **silent**.

### Source
- `internal/gitutil/gitutil.go:13-42` — `UserName()` precedence chain (volatile).
- `internal/cli/next.go:89-101` — `nextUserSlug(cfg)`: `tracking.defaultAgent` → `gitUserName()`.
- `internal/cli/brief.go:97` — `runResume` sets `Options.User = nextUserSlug(cfg)` (fresh local derivation).
- `internal/digest/digest.go:313-355` — `handoffSection` queries by `opts.User`.
- `internal/handoff/ingest.go:44-85` — `IngestUserFile` keys nodes under `parsed.User` (file's frontmatter `user:`).
- `internal/projection/user_handoff.go:47` — writes `user: <opts.User>` into the file's frontmatter.

### Fix Direction
Make the **read path identity-aware** so it resolves the user from what is actually present on disk (the `.hero/next/*.md` files and their `user:` frontmatter) rather than re-deriving a fresh local slug — and have `ingest` additionally re-key (or alias) the rehydrated nodes to the local slug so the local read finds them. Whatever the chosen mechanism, when identity cannot be resolved the system must fail **observably** (a warning / hint), never silently return an empty handoff. As defense-in-depth, prefer committed `tracking.defaultAgent` and consider auto-writing it to committed `hero.json` so identity travels.

---

## Problem Statement

### Confirmed reproduction (real — built binary, this session)
1. **Machine A**: git `user.name "BW"` + `user.email "277887514+chet-bellows@users.noreply.github.com"`. `hero next checkpoint` / `hero next suggest` / `hero next reflection` recorded handoff nodes (UserAsk / NextSuggestion / SessionReflection) keyed under slug **`bw`**, and the per-user file was written to `.hero/next/bw.md` with frontmatter `user: bw`.
2. `git clone` to **Machine B** (a second folder — true cross-machine, since `.hero/graph.db` is gitignored and does NOT travel). On B, set ONLY `git config user.email "277887514+chet-bellows@users.noreply.github.com"` (NO `user.name`).
3. On B: `hero next ingest` ran ("ingested bw.md") and re-created the handoff nodes; `bw.md` is present on the clone.
4. BUT `hero resume` on B showed **NO** "Where you left off" section — empty handoff, no error, no warning.
5. Root of it: `hero resume` resolves the LOCAL slug via `nextUserSlug(cfg)` → **`bwheeler`** (from `$USER`, because no git `user.name`) and queries `handoff.LatestAsk/LatestSuggestion/RecentReflections` keyed by `("bwheeler", repo, domain)` — which does NOT match the ingested nodes under `bw`.
6. **CONFIRMED fix-direction signal**: setting `git config user.name "BW"` on B made `nextUserSlug` resolve to `bw` and the handoff immediately appeared in the brief.

### Live-repo corroboration
On this checkout, `git config user.name` = `chet-bellows` while `$USER` = `bwheeler` — two different slugs from the same machine. The committed handoff file is `.hero/next/chet-bellows.md` with frontmatter `user: chet-bellows`. A clone with `user.name` unset would derive `bwheeler` and never match the `chet-bellows`-keyed nodes. The `tracking` block is absent from committed `.hero/hero.json`, so `nextUserSlug` falls straight through to the volatile `gitUserName()`. And `.hero/hero.local.json` (where `tracking.defaultAgent` would typically be set) IS gitignored (`git check-ignore` exit 0) — so a `defaultAgent` set there would NOT travel to other machines.

## Environment Details
- Solo / team `next` mode both affected (the keying is identical; only the file path differs).
- `tracking.defaultAgent` unset in committed `hero.json` for this project → falls through to git/OS identity.
- `hero.local.json` is gitignored → any `defaultAgent` set there is machine-local and does not travel.
- `graph.db` is gitignored → the `.hero/next/<user>.md` file is the ONLY cross-machine federation medium (no Cloud).

---

## Root Cause Analysis

### 1. The slug precedence chain is volatile (the divergence source)
`internal/cli/next.go:89-101`:
```go
// Priority: hero.local.json tracking.defaultAgent > hero.json tracking.defaultAgent > git user.name.
func nextUserSlug(cfg config.Config) string {
	if cfg.Tracking != nil && cfg.Tracking.DefaultAgent != "" {
		agent := cfg.Tracking.DefaultAgent
		if strings.HasPrefix(agent, "human/") {
			agent = strings.TrimPrefix(agent, "human/")
		}
		return agent
	}
	return gitUserName()
}
```
`gitUserName()` (internal/cli/session.go:324) → `gitutil.UserName()` (internal/gitutil/gitutil.go:25-42):
```go
// Precedence: git config user.name → $USER → $USERNAME → "unknown"
func UserName() string {
	if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		if name := normalizeIdentity(string(out)); name != "" { return name }
	}
	if v := os.Getenv("USER"); v != "" { ... }
	if v := os.Getenv("USERNAME"); v != "" { ... }
	return "unknown"
}
```
Three of the four sources (`user.name`, `$USER`, `$USERNAME`) are per-machine settings that commonly differ between a laptop and a desktop, or between a human's machine and a CI box. **Email is never consulted.** Only the first branch of `nextUserSlug` (committed `tracking.defaultAgent` in `hero.json`) is stable across machines — and it is empty in this project.

### 2. Write/projection anchors identity to the on-disk file (the recorded truth)
`internal/cli/checkpoint.go:397-415` — `writeUserHandoffFile`:
```go
user := nextUserSlug(cfg)                       // A: "bw"
...
content, err := projection.UserHandoffMD(store, projection.UserHandoffOptions{ User: user, ... })
```
`internal/projection/user_handoff.go:47` writes the frontmatter:
```go
fmt.Fprintf(&b, "user: %s\n", opts.User)        // "user: bw"
```
And the file is named `<user>.md` (`resolveNextPath`, next.go:105-111). So on Machine A the filename, the frontmatter `user:`, and the graph node keys all agree on `bw`. The file is a faithful record of A's identity.

### 3. Ingest keys nodes under the file's recorded identity (correct, machine-independent)
`internal/handoff/ingest.go:44-62` — `IngestUserFile`:
```go
parsed, _ := ParseUserHandoff(data)             // parsed.User = fm["user"] = "bw"
if parsed.User == "" { return nil }             // can't attribute → skip silently
...
parsed.Ask.User = parsed.User                   // keyed under "bw"
RecordAsk(store, repoKey, *parsed.Ask)
```
Ingest does the right thing: it keys the rehydrated nodes under the **file's** `user:` ("bw"), NOT a local derivation. So after ingest, Machine B's graph holds `bw:engineering` nodes. This is why the report's "likely read-time" hypothesis is correct.

### 4. Read path re-derives identity locally (the mismatch)
`internal/cli/brief.go:97` — `runResume`:
```go
opts := digest.Options{
	...
	User:   nextUserSlug(cfg),                  // B: "bwheeler" (fresh local derivation!)
	Domain: graph.DomainFor(cfg, graph.IntrinsicActive),
}
```
`internal/digest/digest.go:323-337` — `handoffSection` then queries with that slug:
```go
handoff.LatestAsk(store, opts.User, opts.RepoKey, opts.Domain)        // "bwheeler:..." → miss
handoff.LatestSuggestion(store, opts.User, opts.RepoKey, opts.Domain) // miss
handoff.RecentReflections(store, opts.User, opts.RepoKey, opts.Domain, 3) // miss
```
`singletonKey(user, domain)` (internal/handoff/handoff.go:97-99) = `user + ":" + resolveDomain(domain)`. The graph holds `bw:engineering`; the query asks for `bwheeler:engineering`. Clean miss.

### 5. The miss is silent (the second half of the bug)
`scanLatestSingleton` (handoff.go:292-305) returns `(nil, nil)` on no match — a clean miss, by design, so fresh repos stay clean. `handoffSection` (digest.go:349-352) renders an empty section when all three singletons are absent, and the caller omits empty sections. So an **identity mismatch is indistinguishable from a genuinely-empty handoff** — the user gets no signal that their context exists in the graph under a different key. This silence is itself part of the defect: it makes the failure invisible and un-diagnosable without reading source.

### Confirmed vs. hypothesis
- **Confirmed (read in source):** the slug precedence chain; ingest keying under `parsed.User`; resume keying under `nextUserSlug(cfg)`; `singletonKey` composition; the silent-miss render path; projection writing `user: <slug>`; `hero.local.json` gitignored; `tracking` absent from committed `hero.json`.
- **Confirmed (live repro, built binary):** the empty-brief symptom and the `git config user.name` fix-signal (report).
- **No remaining hypotheses.** The chain from divergent derivation → keyed-under-different-user → silent empty is fully grounded.

---

## Code Flow (End to End)

**Machine A — capture & project:**
1. `internal/cli/checkpoint.go:125` — `user := nextUserSlug(cfg)` → `bw`.
2. `internal/handoff/handoff.go:103-131` — `RecordAsk` upserts UserAsk keyed `bw:engineering`.
3. `internal/cli/checkpoint.go:397-415` — `writeUserHandoffFile` renders `.hero/next/bw.md`.
4. `internal/projection/user_handoff.go:47` — frontmatter `user: bw` written into the file.
5. File committed; `graph.db` gitignored and left behind.

**Machine B — clone, ingest, resume:**
6. Clone arrives with `.hero/next/bw.md`; B's `graph.db` is empty/new.
7. `internal/cli/next_handoff.go:94-146` — `hero next ingest` walks `.hero/next/*.md`.
8. `internal/handoff/ingest.go:40-85` — `IngestUserFile` parses `user: bw`, re-creates nodes keyed `bw:engineering`.
9. `internal/cli/brief.go:66-104` — `hero resume` runs; `Options.User = nextUserSlug(cfg)` → `bwheeler` (no `user.name`, `$USER=bwheeler`).
10. `internal/digest/digest.go:150` → `handoffSection(store, opts, ...)`.
11. `internal/digest/digest.go:323-347` — queries `bwheeler:engineering` → all three miss.
12. `internal/digest/digest.go:349-352` — empty section; caller omits it. **No "Where you left off." No error.**

---

## Key Files

### Identity derivation
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/gitutil/gitutil.go` | 13–52 | `UserName()` volatile precedence: `user.name` → `$USER` → `$USERNAME` → `"unknown"`. Email never consulted. |
| `internal/cli/next.go` | 89–111 | `nextUserSlug(cfg)` (defaultAgent → gitUserName) and `resolveNextPath` (file naming). |
| `internal/cli/session.go` | 320–326 | `gitUserName()` thin wrapper over `gitutil.UserName()`. |

### Write / projection (records identity into the file)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/checkpoint.go` | 125, 397–415 | Derives slug, projects `.hero/next/<user>.md`. |
| `internal/projection/user_handoff.go` | 31–57 | Writes frontmatter `user: <slug>`; renders ask/suggestion/reflections. |

### Ingest (rehydrates under file identity — correct)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/handoff/ingest.go` | 29–130 | `IngestUserFile` keys nodes under `parsed.User` (frontmatter `user:`); `ParseUserHandoff` reads `fm["user"]`. |

### Read path (re-derives identity — the mismatch)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/brief.go` | 66–104 | `runResume`; `Options.User = nextUserSlug(cfg)`. |
| `internal/digest/digest.go` | 302–355 | `handoffSection` queries by `opts.User`; silent empty on miss. |
| `internal/handoff/handoff.go` | 95–99, 202–305 | `singletonKey`, `LatestAsk/LatestSuggestion/RecentReflections`, `scanLatestSingleton` clean-miss. |

### Guardrail that missed it
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/handoff_continuity_test.go` | 85–138, 145–206, 214–292 | Cross-machine guardrail; pins identical identity (`teamCfg()` → `human/alice`) on BOTH machines and hardcodes `"alice"` on read and write — structurally cannot exercise a slug divergence. |

---

## Acceptance Criteria

- WHEN a handoff file authored on machine A (frontmatter `user: bw`) is ingested on machine B whose local git `user.name` is absent (email matches A) THE SYSTEM SHALL surface A's last ask, suggested-next, and recent reflections in `hero resume`'s "Where you left off" section.
- WHEN machine B's locally-derived slug differs from every `.hero/next/<user>.md` frontmatter `user:` present on disk AND no handoff can be matched THE SYSTEM SHALL emit an observable diagnostic (warning or hint naming the slug it queried vs. the slugs available), not a silent empty section.
- WHEN exactly one travel-eligible `.hero/next/<user>.md` (excluding `*.local.md`) is present and the local slug finds no handoff THE SYSTEM SHALL resolve identity to that file's `user:` so the handoff loads. (Best-match/disambiguation rule for the multi-file case is specified in Changes.)
- WHEN committed `tracking.defaultAgent` is set in `hero.json` THE SYSTEM SHALL derive the same slug on every machine regardless of local git config.
- THE SYSTEM SHALL preserve same-machine behavior: a machine whose local slug already matches its handoff file continues to load handoff exactly as today (no regression in `Test_HandoffContinuity_SameMachine`).
- IF identity is genuinely unresolvable (no files present, or ambiguous with no match) THEN THE SYSTEM SHALL fail observably and leave the rest of the brief intact (the handoff miss must never fail `hero resume`).

---

## Suggested Fix Approach

Three options evaluated. **Recommendation: Option B (identity-aware read/ingest), with Option A's `tracking.defaultAgent` auto-write as a complementary defense-in-depth.** Reasoning follows each.

### Recommended — Option B: make the ingest/read boundary identity-aware

The file on disk is the cross-machine truth. The read path should resolve identity from it rather than re-deriving a fresh local slug. The most robust single point of intervention is **ingest**, because it is the cross-machine boundary where the local graph is first populated from the traveled file.

**B-1 (primary): `ingest` records nodes under BOTH the file's `user:` and the local slug (alias), when they differ.**

File: `internal/handoff/ingest.go`, `IngestUserFile`.

*Before* (ingest.go:44-62, abbreviated):
```go
if parsed.User == "" { return nil }
if parsed.Ask != nil && parsed.Ask.Text != "" {
	parsed.Ask.User = parsed.User
	parsed.Ask.Domain = domain
	RecordAsk(store, repoKey, *parsed.Ask)
}
// ...suggestion, reflections, all keyed under parsed.User
```

*After* (intent — exact signature TBD at delivery):
```go
// Key under the file's recorded identity (machine-independent truth)
// AND, when the local reader will look under a different slug, mirror
// the singletons under the local slug so `hero resume` finds them.
fileUser := parsed.User
localUser := localSlugForIngest(cfg) // nextUserSlug(cfg), threaded in
recordFor := func(u string) { /* RecordAsk/Suggestion/Reflection under u */ }
recordFor(fileUser)
if localUser != "" && localUser != fileUser {
	recordFor(localUser)
}
```
*Why:* the local read (`hero resume` keyed by `nextUserSlug`) then finds the singletons regardless of git-config divergence. Ingest is the single chokepoint; fixing it here covers `hero resume`, `hero next ask/suggest/reflection`, and the re-projection. **Caveat to resolve at delivery:** mirroring under the local slug must not corrupt a true multi-user team workspace (two real people). Gate the alias on solo-ish context (single travel-eligible file, or `next.mode != team`), or alias only when the local slug currently has zero handoff nodes. Reflections must reuse the existing dedupe-by-text so the alias copy doesn't double-count.

**B-2 (read-side fallback, belt-and-suspenders): when `handoffSection` finds nothing under the local slug, resolve a fallback slug from the present files.**

File: `internal/digest/digest.go`, `handoffSection` (or a helper in `internal/cli/brief.go` that sets `Options.User`).

*Intent:* if `LatestAsk/Suggestion/Reflections` all miss under `opts.User`, scan `.hero/next/*.md` (excluding `*.local.md`) for frontmatter `user:` values; if exactly one distinct value exists, retry the queries under it. If multiple distinct values exist and none matches the local slug, **emit a diagnostic** ("handoff present for users [bw, dustin] but your local identity is 'bwheeler'; set tracking.defaultAgent or git user.name to match") and leave the section empty — observably, not silently.

*Why:* covers the case where ingest has already run under the file identity on a prior session (so B-1's same-call mirror didn't fire), and guarantees the **fail-loud** acceptance criterion. This also handles the "no ingest yet" path by hinting the user toward `hero next ingest`.

**B-3 (kill the silence): `handoffSection` distinguishes "no handoff exists" from "handoff exists under a different identity."**

File: `internal/digest/digest.go:349-352` and/or `internal/cli/brief.go`.

*Intent:* when the local-slug query misses but `.hero/next/*.md` files exist with handoff content under other slugs, render a one-line hint in the brief (or stderr) rather than nothing. The hint names the mismatch so the user can self-correct. Never fail `hero resume`.

### Complementary — Option A: prefer/auto-persist a stable committed identity

`nextUserSlug` already prefers `tracking.defaultAgent`. The gap is that nobody writes it to **committed** `hero.json`, so it never travels.

**A-1:** On `hero init` / first handoff write, if `tracking.defaultAgent` is unset in committed `hero.json`, write the current derived slug there (committed, not `hero.local.json`). Then every machine derives the same slug.

*Why complementary, not primary:* (i) it only helps repos initialized/migrated after the fix — existing repos with divergent live identities still need Option B to recover; (ii) writing identity into committed config is a product decision (some teams may not want a single `defaultAgent` committed); (iii) it does nothing for a repo already in the broken state. It is the right **forward-looking** stabilizer but not a sufficient **fix** on its own.

*Migration / back-compat note:* changing slug **derivation** (e.g. switching the fallback from `user.name` to an email-derived slug) would re-key existing repos and orphan current `.hero/next/<slug>.md` files — assessed as **not worth it**. The tiny user base permits a near-clean break, but Option B avoids re-keying entirely by matching on the file's recorded `user:` rather than changing how slugs are derived. Keep `gitutil.UserName()`'s precedence as-is; fix the matching, not the derivation.

### Rejected — Option C: change `gitUserName()` to derive from email
Deriving the slug from `git config user.email` (more stable than name) was considered. Rejected: (i) it re-keys every existing repo and orphans every existing `.hero/next/<name-slug>.md`; (ii) email-derived slugs are ugly (`bdwheeler-at-gmail-com`) and still differ if a person uses different emails per machine; (iii) it doesn't fix repos already in the broken state. Option B's file-anchored matching is strictly better.

### Net recommendation
Land **B-1 + B-2 + B-3** as the fix (identity-aware ingest + read-fallback + fail-loud). Add **A-1** as a follow-on stabilizer so new repos never enter the broken state. The combination satisfies every acceptance criterion: cross-machine load works regardless of git-config divergence (B-1/B-2), unresolvable identity is observable not silent (B-3), same-machine path is untouched (matching is additive), and new repos get a stable committed identity (A-1).

---

## Test Plan

### Existing test review
- `internal/cli/handoff_continuity_test.go` — the cross-machine guardrail (`Test_HandoffContinuity_CrossMachine`, `_AutoEmit`, `_Idempotent`, `_GuardrailBites`). **This is the test that should have caught the bug and structurally cannot.** Both `teamCfg()` instances set `tracking.defaultAgent = "human/alice"`, and both `seedMachineA` (write) and `assertHandoffReconstructed` (read) hardcode the slug `"alice"`. The slug is identical on both sides by construction, so a slug **divergence** between machines is never exercised. **Explicit gap.**
- `internal/cli/checkpoint_test.go` — `Test_writeCheckpoint_TeamMode_RoundTripIsIdempotent`: same-machine, same-graph round-trip; never severs the graph, never varies identity.
- `internal/gitutil/gitutil_test.go:233-302` — already exercises `UserName()` when `user.name` and `$USER` disagree; documents the live repro but does not connect it to the handoff load path.
- `internal/projection/user_handoff_test.go` — projection rendering, hardcoded `"alice"`.

### Test changes needed (the new test the guardrail is missing)
1. **`Test_HandoffContinuity_CrossMachine_SlugDivergence`** (new, in `handoff_continuity_test.go`). Machine A uses identity that derives slug `bw` (e.g. `teamCfg()` with `DefaultAgent: "human/bw"`, OR drive a real checkpoint with A's git `user.name`). Capture the committed `bw.md`. Machine B is constructed so that its **local** `nextUserSlug` derives a DIFFERENT slug (e.g. no `tracking.defaultAgent`, controlled `$USER`/`user.name` so `gitUserName()` returns `bwheeler`). Drop only `bw.md` on B, run the real `hero next ingest`, then assert `hero resume`'s brief (and the `hero next ask/suggest/reflection` query surface) surfaces A's content **under B's local identity**. This is the assertion the current guardrail cannot make because it pins identity equal on both sides.
   - Implementation note: B's local slug must be controllable in-test. Either thread a config-driven override, or seed B's `hero.json` WITHOUT `tracking.defaultAgent` and set the env (`$USER`) / a temp git `user.name` so `nextUserSlug` returns the divergent value deterministically. The existing `gitutil_test.go` helpers (`run(t, dir, "config", "user.name", ...)`) show the pattern for controlling git config in a temp repo.
2. **`Test_HandoffContinuity_UnresolvableIdentity_IsObservable`** (new). B has files present under slugs that match neither the local slug nor each other (ambiguous), no match found. Assert the brief / stderr carries the diagnostic hint (naming queried-vs-available slugs) and that `hero resume` still succeeds (non-fatal). Guards the fail-loud criterion.
3. **Guardrail-bites update**: ensure at least one assertion would go RED if a future change reverted B-1/B-2 (i.e. the divergence test must genuinely depend on the new matching, mirroring the spirit of `Test_HandoffContinuity_CrossMachine_GuardrailBites`).

### Regression scope
- Same-machine load path (`Test_HandoffContinuity_SameMachine`) must stay green — the fix is additive matching, not derivation change.
- Idempotence (`_Idempotent`) must stay green — B-1's alias write must reuse reflection dedupe so no double-count; assert reflection count == 1 after round-trip under both the file slug and the alias slug.
- True multi-user team workspaces: confirm B-1's alias gating does not bleed one real user's handoff into another's key. Add a team-mode test with two distinct real users + two files and assert no cross-contamination.
- `hero next ask/suggest/reflection` read surfaces (next_handoff.go) inherit the same `nextUserSlug` keying — verify they also resolve via the new matching (or are explicitly out of scope and documented).

---

## Boundaries
- Do NOT change `gitutil.UserName()`'s derivation precedence — that re-keys existing repos and orphans existing handoff files (Option C, rejected). Fix the matching, not the derivation.
- Do NOT introduce Cloud/federation as the fix — the file-on-disk medium is the contract for solo-no-Cloud continuity and must keep working standalone.
- Do NOT auto-migrate or rename existing `.hero/next/<slug>.md` files. Existing filenames must continue to resolve.
- The `domain` component of the key is correct and out of scope; this bug is purely the `user` component.
- Team-server / multi-real-user disambiguation beyond "don't cross-contaminate" is out of scope for this fix; the recommended gating is conservative (alias only when unambiguous).

## Risks
- **Multi-user cross-contamination** (B-1): mirroring nodes under the local slug in a genuine team workspace could attribute one person's handoff to another. Mitigate by gating the alias to unambiguous/solo contexts and by aliasing only when the local slug has zero existing handoff nodes.
- **Diagnostic noise** (B-3): a fail-loud hint that fires too eagerly (e.g. on genuinely fresh repos) would be annoying. Gate the hint on "handoff content exists under other slugs on disk" so fresh repos stay silent.
- **Reflection double-count** (B-1): aliasing reflections must reuse the existing dedupe-by-text; otherwise the alias copy doubles entries. Covered by the idempotence regression.
- **Committed-identity product decision** (A-1): writing `tracking.defaultAgent` into committed `hero.json` is a behavior change some teams may not want; make it opt-out-able or only-on-init.

## Validation
- Reproduce the original failure on the built binary BEFORE the fix (A slug `bw`, B with no `user.name`) → empty brief. After the fix → brief surfaces A's ask/suggestion/reflection.
- `go test ./internal/cli/ ./internal/handoff/ ./internal/digest/ ./internal/projection/` green, including the two new tests.
- Confirm `hero resume` on B prints an observable hint (not silence) when identity is unresolvable.
- Confirm same-machine and idempotence guardrails remain green.

## Notes
- Session-originated; no tracker configured (`hero.json` tracker.type = "none"), so no tracker posting. No `tracker_id`.
- The original report's "derived from email → bdwheeler" mechanism is corrected here: the divergent slug actually comes from `$USER`, not email (`gitutil.UserName` never reads email). Root cause and severity are unchanged.

## Delivered (files changed)
- `internal/handoff/ingest.go` — B-1: `IngestUserFile` takes a `localSlug` param and mirrors singletons under it when safe; added `safeAliasSlug` (zero-existing-node gate). Reflection dedupe reused so the alias copy never double-counts.
- `internal/cli/next_handoff.go` — threads `nextUserSlug(cfg)` into `IngestUserFile` (the real `hero next ingest` entry point).
- `internal/cli/scan.go` — threads `nextUserSlug(cfg)` into `IngestUserFile` (the `hero scan` ingest entry point).
- `internal/cli/brief.go` — B-2/B-3: `resolveHandoffIdentity` + `handoffHasContent` + `nextFileUserSlugs`; `runResume` reconciles the local slug against on-disk file identities and emits a fail-loud diagnostic when unresolvable.
- `internal/cli/checkpoint.go` — A-1: `persistDefaultAgentIfUnset` writes the derived slug to the COMMITTED `hero.json` (surgical, never `hero.local.json`) once when unset.
- Tests: `internal/handoff/ingest_test.go` (alias + gate units), `internal/cli/handoff_continuity_test.go` (`Test_HandoffContinuity_CrossMachine_SlugDivergence`, `_SlugDivergence_FallbackWithoutMirror`, `_UnresolvableIdentity_IsObservable`, `Test_persistDefaultAgentIfUnset_*`, `Test_IngestUserFile_TeamModeGate_NoCrossContamination`). Existing callers updated to the new signature (pass `""` = no alias).
- Team-mode anti-corruption gate chosen: **alias only when the local slug currently has ZERO handoff nodes**. Works in any mode; a real second team member already owns nodes, so their handoff is never clobbered.
- Verified end-to-end with the built binary: A (`user.name "BW"` → slug `bw`) → clone → B (email-only, divergent `$USER`): pre-ingest `hero resume` emits the diagnostic (no silent empty); post-ingest it surfaces A's ask/suggestion/reflection. A-1 path also verified (committed `default_agent` travels, B derives `bw`).

## Kickoff

Cross-machine handoff silently loads EMPTY because `hero resume` re-derives the user slug from volatile local git/`$USER` config, which differs from the slug baked into the traveled `.hero/next/<user>.md` file that `ingest` keyed nodes under.

**Status:** planning — root cause confirmed against source (file:line) and a live repro; no code written.

**Pick up at:** implement Option B — make `IngestUserFile` mirror handoff nodes under the local slug when it differs from the file's `user:` (B-1), add a read-side fallback + fail-loud hint in `handoffSection`/`runResume` (B-2/B-3), and write `Test_HandoffContinuity_CrossMachine_SlugDivergence` that pins DIFFERENT slugs on A and B (the current guardrail pins them equal and cannot catch this).

→ `.hero/planning/bugs/cross-machine-handoff-slug-mismatch/spec.md`

**Files:** `internal/handoff/ingest.go:44`, `internal/cli/brief.go:97`, `internal/digest/digest.go:313`, `internal/cli/handoff_continuity_test.go:85`, `internal/cli/next.go:91`
**Skip:** changing `gitutil.UserName()` derivation or deriving from email — re-keys existing repos, orphans existing handoff files (Option C, rejected).
