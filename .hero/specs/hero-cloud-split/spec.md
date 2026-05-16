---
title: "Hero Cloud Repo Split — Carve hero-cloud Out of the hero Monorepo"
type: feature
status: completed
priority: high
tags: [platform, migration, repo-split, contracts, structure]
created: 2026-05-15
relations:
  - target: hero-governance
    kind: depends-on
  - target: hero-cloud
    kind: parent
  - target: cloud-admin
    kind: related
  - target: agent-outposts
    kind: related
horizon: now
---

# Hero Cloud Repo Split — Carve hero-cloud Out of the hero Monorepo

## Goal

Split the current single `hero` repository into two sibling repositories —
`hero` (CLI + local engine + contracts) and `hero-cloud` (all server code) —
without losing history on the moved files, without breaking dogfooding,
and without forcing a publishing/versioning overhead before it earns its
keep.

Done means:

- A new `contracts/` package exists inside the current `hero` repo and is
  the *only* place `hero-cloud` is allowed to import from.
- A new `hero-cloud` repo exists at `~/projects/hero-engine/repository/hero-cloud`,
  containing everything currently under `cloud/`, the cloud-side bits of
  `internal/`, and the `cmd/hero-cloud/` entrypoint, with file history
  preserved.
- `hero-cloud` builds against a sibling checkout of `hero` for local dev
  (path dependency), and against a pinned commit SHA in CI (the pin file
  drives a clone of `hero` at that SHA before build).
- Both repos have working `hero scan` self-dogfooding, their own
  `.hero/` workspaces, and CI green.
- The rollback path is a single revert of the "cut" commit on `hero-cloud`
  plus a re-merge of `hero-cloud`'s tree back into `hero` — explicitly
  documented and tested in a dry run.

## Kickoff

Structural migration spec that splits the cloud server code out of the
`hero` monorepo into a sibling `hero-cloud` repo, with a shared
`contracts/` package as the seam and a `hero.ref` pin for reproducible
builds.

**Status:** delivering — Phase 0 and Phase 1 both landed. Phase 0
carved the `contracts/` leaf package (governance + Node + Event +
ContractsVersion / ServerMinContractsVersion) inside `hero` and stood
up the two symmetric boundary tests
(`contracts/contracts_boundary_test.go` and
`cloud/cloud_boundary_test.go`). Phase 1 cut a new private repo at
`github.com/hero-engine/hero-cloud`, sibling-checked-out at
`~/projects/hero-engine/repository/hero-cloud`. Mechanism: per-prefix
`git subtree split` for `cloud/` and `cmd/hero-cloud/` in a throwaway
clone, each branch filter-branch-rewritten to bake in its destination
prefix, then merged into `hero-cloud`'s `main` as a single inaugural
commit with both subtree histories as parents. Per-file
`git log --follow` traces every moved file back to its v0.8.0 root
commit in `hero` (acceptance criterion verified). The cloud trees in
`hero` are untouched — Phase 3 deletes after Phase 2 cutover + soak.

Seam infrastructure on `hero-cloud`: `go.mod` with module
`github.com/hero-engine/hero-cloud` and a committed
`replace github.com/hero-engine/hero => ../hero` directive; `hero.ref`
pinned at `1763417` (Phase 0 head) with bump-rule doc comment;
`scripts/hero-pin-fetch.sh` idempotent and safe for local and CI;
minimal `.github/workflows/ci.yml` (one job, Go 1.26.1, ubuntu-latest:
fetch pin, build, vet, test); README; `.gitignore`.

Seam smoke canary: `cloud/internal/seam_smoke.go` +
`seam_smoke_test.go` actually import
`github.com/hero-engine/hero/contracts/governance`, exercise
`Classification`, `Subject`, `SubjectType`, and the `Compare`
ordering. This is the first real cross-repo dependency on the
contracts package; previously the boundary tests fired zero
violations only because cloud code happened not to import contracts
at all. The smoke is the canary going forward — if the seam ever
breaks in either direction, this package's compile/test fails first.

`go build`, `go vet`, `go test ./...` all green in both repos at the
end of Phase 1.

Phase 1 workspace follow-ups landed 2026-05-15 (hero-cloud commit
`4f93ed3`): `.hero/` workspace bootstrapped in hero-cloud with a
hero-cloud-specific `mission.md`, CLAUDE.md committed, and two
scaffold specs filed (`ci-token-setup` for the GitHub PAT, and
`seam-canary-expansion` for ongoing seam coverage). Convention
specs (`contracts-import-discipline.md` here,
`cross-repo-workflow.md` in hero-cloud) are still pending and
tracked as a separate follow-on.

**Completed 2026-05-16.** Phase 2 collapsed to a no-op (no `hero-cloud`
deployment exists in production yet — nothing to cut over). Phase 3
landed: cloud trees deleted from `hero`, Makefile pruned of cloud
targets, `docker-compose.yml` moved to `hero-cloud` alongside a new
`hero-cloud/Makefile` carrying the dev/setup/build targets. `HERO_REPO_TOKEN`
configured; hero-cloud CI ran green end-to-end (run `25937347391`) — the
first real proof of the cross-repo seam under CI conditions. Soak
window waived given no deployment exists. Both repos now self-contained;
contract changes flow `hero → hero.ref` bump in `hero-cloud` per the
`cross-repo-workflow` convention.

→ `.hero/planning/initiatives/hero-cloud-split/spec.md`

**Files:** `contracts/` (new), `cloud/` (all — to be moved), `cmd/hero-cloud/`,
`internal/` (selective — see scope rule), `go.mod`, `.hero/`,
`hero.ref` (new in hero-cloud), `CLAUDE.md`, `AGENTS.md`,
`.hero/planning/features/hero-governance/spec.md` (read for contract shapes).
**Skip:** designing anything *inside* `hero-cloud`; Community Edition
packaging; licensing decisions; tracker integration changes.

## Problem

Today everything lives in `github.com/hero-engine/hero`:

- CLI binary at `cmd/hero/`
- Cloud server binary at `cmd/hero-cloud/`
- Cloud-only code at `cloud/` (API handlers, middleware, web UI, migrations,
  auth, GitHub app integration)
- ~55 packages under `internal/`, of which most are local-engine and a
  handful are cloud-only or cloud-shared

This is fine right now for solo iteration, but the trajectory is wrong:

- The `hero-governance` spec just landed and explicitly mandates a
  `pkg/contracts/governance/` package as the OSS/paid seam. That package
  is the precise thing `hero-cloud` will import; it must live where
  it can be cleanly consumed by a separate codebase.
- Eight follow-on cloud-flavored specs are in flight (`cloud-admin`,
  `cloud-billing`, `cloud-sync`, `cloud-mcp`, `cloud-api`, `cloud-auth`,
  `cloud-notifications`, `agent-outposts` server side). Building these
  in the same repo as the CLI guarantees cross-coupling drift.
- Future Community Edition is a stripped build of `hero-cloud`, not a
  fork of `hero`. The CE/Cloud/Enterprise distinction must live on the
  server side; keeping the server in the CLI repo blurs that line.
- A future OSS license decision on `hero` is harder to negotiate while
  cloud code shares the same git history. Even though both repos stay
  private for now, separation keeps options open.

The cost of waiting: every cloud-feature PR touches `internal/` and
muddies the boundary. The cost of rushing: a clumsy split breaks
dogfooding, loses history, and adds publishing overhead before it earns
its keep.

The shape this spec picks — **path-dependency from `hero-cloud` to a
sibling `hero` checkout, with a committed SHA pin for CI** — is the
minimum viable separation. No registry publishing, no module proxy
setup, no v0.x.y churn. Two repos that know about each other via
`../hero`.

## Design

Four pieces: the scope rule, the contracts package, the pin mechanism,
and the dogfooding/workspace story. Migration sequencing is in the next
section.

### 1. Scope rule — what moves, what stays

The classification rule for any module, file, or future addition:

> Anything that runs as a process serving other processes goes to
> `hero-cloud`. Anything that runs on a developer's machine in support
> of their own work stays in `hero`.

Tie-breakers when the rule is ambiguous:

- If the module is *imported by* both the CLI binary and the server
  binary today, it stays in `hero` (the CLI is the smaller consumer
  surface; pulling it out would invert the dependency).
- If the module has *no consumers* in the current CLI binary but has
  consumers in the current cloud binary, it moves to `hero-cloud`.
- If the module *defines a wire shape* (request/response types, event
  envelopes, token claims, audit shapes, policy schema, retriever API),
  it goes into `contracts/` (see §2) regardless of which binary uses
  it today.

**Concrete: what moves to `hero-cloud`:**

- `cloud/` entire tree: `cloud/api/`, `cloud/auth/`, `cloud/github/`,
  `cloud/middleware/`, `cloud/migrations/`, `cloud/store/`, `cloud/web/`,
  `cloud/Dockerfile`, `cloud/web.go`
- `cmd/hero-cloud/` (the server binary entrypoint)
- Future `internal/enforcement/`, `internal/sync/server/`, `internal/audit/server/`,
  `internal/outpost/server/` packages as they land
- `docker-compose.yml` if its sole purpose is local-cloud-up (audit
  during migration; split if it serves both)
- `TEAM-SERVER.md`, `MCP-SETUP.md` (cloud-flavor portions only)

**Concrete: what stays in `hero`:**

- `cmd/hero/` (CLI binary)
- All slash commands in `commands/`
- All agents in `agents/`
- All skills in `skills/`
- All local-engine packages under `internal/` *except* those listed
  above — graph, scan, index, search, spec, traversal, retrieval,
  knowledge, watch, etc.
- `contracts/` (new, see §2)
- `core/`, `domains/`, `hero/` top-level go packages (CLI surface)
- `docs/`, `mkdocs.yml`, `wrangler.toml` — public-facing docs stay in
  `hero` (the cloud-specific marketing site can split later if it
  earns its own home)
- `.hero/` workspace for `hero` itself

**Concrete: ambiguous, audit during migration:**

- `internal/serve/` — looks like a server, but if it's the local MCP
  server it stays in `hero`. The cloud MCP server is a separate
  package that lives in `hero-cloud`.
- `internal/tracker/` — local tracker sync stays; any cloud-side
  webhook receiver moves.
- `internal/sessions/`, `internal/handoff/`, `internal/feed/` —
  determine by import graph; if any of them are imported by `cloud/`
  packages today, they either stay (and the cloud imports them via
  the path dep) or get an exported counterpart in `contracts/`.

### 2. The contracts package

**Location:** `contracts/` at the root of `hero`, importable as
`github.com/hero-engine/hero/contracts/...`.

Rejected: `pkg/contracts/`, `internal/contracts/`, `api/`.

- `pkg/` adds a Go folder convention this repo doesn't use elsewhere
  and signals "library code for external consumers," which we want.
  Acceptable second choice, but the repo doesn't have a `pkg/` tree
  today and adding one for one directory is noise.
- `internal/` is unusable — `hero-cloud` is a separate module and
  cannot import `internal/` paths from `hero`. This is a Go-toolchain
  hard rule.
- `api/` is too narrow — contracts cover graph shapes, audit, policy,
  and tokens, not just request/response.

`contracts/` reads as what it is. The governance spec already names
`pkg/contracts/governance/`; that becomes `contracts/governance/`
in this layout. Reconcile during migration.

**Surface (from `hero-governance`):**

```
contracts/
  governance/
    classification.go    // Classification enum + extension
    subject.go           // Subject{Type, ID, Label}
    principal.go         // Principal, Scope, AgentToken (JWT claims)
    policy.go            // PolicyNode, Rule, RuleKind, Matcher, Action
    audit.go             // AuditEvent, AuditToken, Purpose
    retriever.go         // Retriever interface, Query, NodeDecision
  graph/
    node.go              // Graph node base shape: ID, Kind, Classification,
                         //   Subjects, Origin, CreatedAt, UpdatedAt, ContentHash
    edge.go              // Edge shape
    ref.go               // NodeRef
  events/
    envelope.go          // Event protocol envelope (CLI <-> cloud stream)
    kinds.go             // Event kind enum (capture, sync, audit, outpost, ...)
  auth/
    token.go             // User session token shape (separate from AgentToken)
  version.go             // ContractsVersion constant + min-version check helper
```

The governance shapes are already specified; this spec doesn't re-design
them. The non-governance shapes (`graph/`, `events/`, `auth/`) are
additive and minimal — only what `hero-cloud` actually needs to import.
Nothing speculative.

**Import discipline:**

- `hero-cloud` MAY import from `github.com/hero-engine/hero/contracts/...`
  and from its own packages. Nothing else from `hero`.
- The CLI in `hero` MAY import from `contracts/` (it does — it produces
  and consumes the same shapes).
- `contracts/` MUST NOT import from anywhere else inside `hero`. It is
  a leaf package. No imports of `internal/`, no imports of `core/`,
  nothing.

**Enforcement (three layers, ordered by cost):**

1. **Lint script** in `hero-cloud`'s CI: a tiny Go program (or
   `go list -deps` + grep) that walks every import in the `hero-cloud`
   module and asserts every import path matching `github.com/hero-engine/hero/`
   begins with `github.com/hero-engine/hero/contracts/`. Fails CI on
   violation. ~30 lines of Go.
2. **Lint script** in `hero`'s CI: walks every import in `contracts/`
   and asserts no import path matches `github.com/hero-engine/hero/`
   except for other paths within `contracts/`. Keeps the package as a
   leaf.
3. **Convention spec** at `.hero/knowledge/conventions/contracts-import-discipline.md`
   in the `hero` repo that documents the rule in human-readable form
   for reviewers and agents.

Run both lint scripts via `make lint` / `go test ./...` so they fire
locally too. CI is the backstop, not the only gate.

### 3. Pin mechanism

`hero-cloud` references `hero` two ways: path dependency for dev,
committed SHA for reproducible builds.

**Files in `hero-cloud`:**

- `hero.ref` at repo root. Plain text, two lines:
  ```
  sha: 3176736abc1234deadbeef...
  tag: v0.4.2-contracts   (optional, human-readable)
  ```
- `go.mod` with a `replace` directive:
  ```
  replace github.com/hero-engine/hero => ../hero
  ```
  This makes local dev "just work" when both repos are siblings.
- `scripts/hero-pin-fetch.sh` — clones `hero` at the SHA in `hero.ref`
  into `../hero` if `../hero` doesn't already exist. Used by CI.
- `.gitignore` ignores `../hero` (it's outside the repo anyway, but
  document the assumption).

**CI flow (hero-cloud):**

1. Checkout `hero-cloud` at the PR head.
2. Run `scripts/hero-pin-fetch.sh` — clones `hero` at `hero.ref`'s SHA
   into `$GITHUB_WORKSPACE/../hero` (or wherever the runner places it).
3. `go build ./...` from `hero-cloud` — Go resolves the `replace` to
   the sibling checkout, which is now at the pinned SHA.
4. `go test ./...`.

**Local dev flow:**

1. Developer has `~/projects/hero-engine/repository/hero` and
   `~/projects/hero-engine/repository/hero-cloud` side by side.
2. `replace` directive points to `../hero`. Whatever is checked out
   there is what `hero-cloud` builds against. The pin in `hero.ref`
   is ignored locally unless the developer chooses to honor it.
3. To match CI exactly, developer runs `scripts/hero-pin-fetch.sh`
   in a clean sibling directory.

**Bumping the pin:**

Workflow when a contract changes:

1. PR to `hero` lands the contract change.
2. PR to `hero-cloud` updates `hero.ref` to the new SHA *and* updates
   any consuming code in the same PR. CI runs against the new pin.
3. Merge both. The "I'm adopting the new contract version" moment is
   exactly the `hero.ref` change in the cloud PR. Reviewers see one
   line change in `hero.ref` and the corresponding adoption code in
   the same diff.

This is approximately as fast as same-repo work for someone with both
checkouts. The friction is real but bounded: one extra commit per
contract change.

**Version skew (CLI in field older than server expects):**

`contracts/version.go` defines a single constant:

```go
const ContractsVersion = 3
```

- Every event envelope carries `contracts_version: int`.
- The server, on receipt, rejects envelopes with
  `contracts_version < ServerMinContractsVersion` with a structured
  error pointing the CLI at an upgrade.
- The server logs (and audits) the rejection but does not silently
  ignore — the CLI must surface "your client is too old."
- Bumping `ContractsVersion` is a deliberate act, documented as a
  breaking-change marker in `contracts/CHANGELOG.md` (file to be
  added in this migration).
- Min-server-version tolerance: the server SHOULD accept up to N
  versions behind (N=3 default, configurable). Older clients get a
  deprecation warning in the response envelope.

This is the minimum sufficient skew protocol. A future spec may
formalize semver, deprecation windows, and tooling. Not now.

### 4. Workspaces and dogfooding

**Two separate `.hero/` workspaces, not one shared.**

- `hero/.hero/` continues to be the workspace for `hero` itself.
- `hero-cloud/.hero/` is a new workspace for the cloud repo. Created
  via `hero init` after the split.
- Each repo has its own `hero.json`, its own spec set, its own
  knowledge base.
- Cross-repo specs (most non-trivial work) live in *one* of the two
  workspaces, picked by where the **primary** code change lands. The
  spec body references the other repo's PR by URL. There is no
  cross-workspace spec linking in v1; it's done by hand.

Why separate workspaces:

- A merged workspace requires `hero scan` to walk both trees, which
  requires either symlinks or a multi-root config we don't have.
- Spec status semantics get muddied if one tracker issue spans two
  repos.
- Knowledge entries naturally cluster by repo concern (CLI conventions
  don't apply to cloud code and vice versa).

**`hero scan` self-dogfooding after split:**

- `hero scan` from inside `hero/` works exactly as today — the CLI
  binary builds from the same tree it's scanning.
- `hero scan` from inside `hero-cloud/` requires the `hero` CLI to be
  on PATH (or built from the sibling checkout). Path dep makes this
  trivial in dev (`go run github.com/hero-engine/hero/cmd/hero ...`
  resolves through the replace directive).
- CI for `hero-cloud` that wants to run `hero check` etc. invokes the
  CLI from the pinned sibling checkout. Document in
  `hero-cloud/CLAUDE.md`.

### 5. Cross-repo refactor workflow (the hot path during early life)

Most non-trivial work during the first few months will touch both
repos. The flow:

1. Open a branch on `hero` with the contract change.
2. Open a branch on `hero-cloud` with `replace` pointing at the local
   checkout (already the default).
3. Iterate freely — both repos build from local source.
4. When ready, PR `hero` first. Merge.
5. Bump `hero.ref` in the `hero-cloud` PR to the merged SHA.
6. Merge `hero-cloud`.

The user's stated concern — "contracts will be refactored across both
repos frequently in the near term" — is the reason path-dep is the
right shape. The `hero.ref` pin only gates CI reproducibility; local
dev moves at the speed of single-repo work.

Add to `hero-cloud/CLAUDE.md`: a Cross-repo workflow section with
exactly this flow, so future agents don't re-derive it.

## Migration Plan

Expand-contract migration with the contracts package as the expand
phase, the repo cut as the migrate phase, and the cleanup of moved
files from `hero` as the contract phase. Sequenced to keep both
binaries buildable at every checkpoint.

### Phase 0 — Pre-split: carve `contracts/` inside `hero`

Goal: get the boundary right while still in one repo, where mistakes
are cheap.

1. Land `hero-governance` foundation contracts at `contracts/governance/`
   (the governance spec already targets this — coordinate so the
   directory name matches the chosen layout).
2. Add `contracts/graph/`, `contracts/events/`, `contracts/auth/`,
   `contracts/version.go` per the surface in §2. Move only the shapes
   `hero-cloud` will actually need on day one. Resist adding
   speculative types.
3. Migrate current cloud-server imports inside the repo to consume
   from `contracts/` instead of from wherever the shapes live today.
   Each such migration is a small refactor PR.
4. Add the two lint scripts (`scripts/check-contracts-leaf.sh`,
   `scripts/check-cloud-imports.sh`) and wire them into `make lint`.
   The cloud-imports script targets the current `cloud/` and
   `cmd/hero-cloud/` trees as a stand-in for the future `hero-cloud`
   repo — same rule, different boundary.
5. Verify: `make lint` green, both binaries build, all tests pass.

**Checkpoint:** repo still single, but the boundary is testable.
Continue here for as long as it takes to iron out contract shape
issues. Rolling this back is `git revert`.

### Phase 1 — Cut: create `hero-cloud` repo with history

Goal: produce a new repo containing the cloud subtree with its history
intact, leaving `hero` untouched for now.

**Method: `git subtree split` (recommended over `git filter-repo`).**

Reasoning: `git subtree split --prefix=<path>` produces a new branch
containing only the history of files under `<path>`, preserving
commit history for those files. It is non-destructive on the source
repo. `git filter-repo` is more powerful but rewrites the source
repo's history, which is unnecessary here and adds risk.

The subtree-split limitation: it works cleanly for *one* prefix. We
have multiple paths to move (`cloud/`, `cmd/hero-cloud/`, and
eventually some `internal/` packages). Handle by either:

- (a) Run `git subtree split` once per prefix, push each to a branch in
  the new repo, then merge them on the new repo's `main`. History is
  preserved per-prefix.
- (b) Pre-stage: in a throwaway clone of `hero`, move all
  to-be-extracted files under a single `_split/` prefix in one commit,
  then `git subtree split --prefix=_split` and move them back to
  their proper locations on the new repo's `main`. Cleaner history on
  the destination side, at the cost of one synthetic move commit.

Choose (b) — single subtree split, cleaner destination, the synthetic
move commit is a small price for a coherent `main` history in the new
repo.

Concrete steps:

1. Create empty `hero-cloud` repo on the user's git host. No initial
   commit.
2. Clone `hero` to a throwaway directory: `~/tmp/hero-split-work/`.
3. In the throwaway clone, on a `split-prep` branch, `git mv` all
   to-be-moved paths under `_split/`. Commit as
   "chore(split): stage cloud files for subtree split".
4. `git subtree split --prefix=_split -b cloud-only`. This creates a
   branch `cloud-only` containing only the staged files' history.
5. In a fresh clone of the empty `hero-cloud` repo, fetch the
   `cloud-only` branch from the throwaway and `git pull` it as the
   initial history.
6. On `hero-cloud`'s `main`, `git mv` files out of `_split/` back to
   their proper locations (`cloud/api/` → `api/` or wherever the
   target layout puts it). Commit as "chore(split): place files at
   final paths".
7. Add `go.mod` for `hero-cloud` (module name TBD — suggest
   `github.com/hero-engine/hero-cloud`).
8. Add `hero.ref` with the SHA of `hero`'s `main` at split time.
9. Add `scripts/hero-pin-fetch.sh`, the `replace` directive in
   `go.mod`, and a minimal CI workflow.
10. Add `hero-cloud/.hero/` via `hero init` (run from a sibling
    `hero` checkout).
11. Add `CLAUDE.md`, `AGENTS.md` for the new repo with the cross-repo
    workflow section.
12. First green build on `hero-cloud` CI.

**Checkpoint:** new repo exists, builds, has history. `hero` is
unchanged. Rollback is `rm -rf hero-cloud` — no impact on `hero`.

### Phase 2 — Migrate: cut over

1. Open a PR on `hero-cloud` that brings it to feature parity with
   the current cloud subtree on `hero/main` (any commits between
   split prep and cutover get cherry-picked or re-applied). Merge.
2. Smoke-test the cloud binary built from `hero-cloud`: run it
   against a local DB, hit a few endpoints, verify it behaves
   identically to the same binary built from `hero` pre-split.
3. Update any external CI/deployment that targets the cloud binary
   to build from `hero-cloud` instead of `hero`.

**Checkpoint:** `hero-cloud` is the canonical source for the cloud
binary. `hero` still has the cloud tree but it's no longer the
source of truth. Rollback is "revert the deployment change," zero
data risk.

### Phase 3 — Contract: remove from `hero`

Soak period: at least one full development week with `hero-cloud` as
the source of truth before deletion. If anything goes wrong, the old
tree in `hero` is still there to fall back to.

1. PR on `hero` that deletes `cloud/`, `cmd/hero-cloud/`, and any
   `internal/` packages that moved.
2. Update `hero/CLAUDE.md`, `hero/AGENTS.md`, `hero/README.md`,
   `hero/Makefile` to remove cloud-related sections and targets.
3. Update `hero/hero.json` and `hero/.hero/` if any references
   point at the removed tree.
4. Verify `make lint` and `go test ./...` both green in `hero`.
5. Verify `hero scan` self-dogfood still produces a sane result.

**Checkpoint:** clean separation. `hero` is CLI + contracts only.
`hero-cloud` is the server.

### Phase 4 — Post-split documentation and conventions

1. Write `.hero/knowledge/conventions/contracts-import-discipline.md`
   in `hero` (the rule).
2. Write `.hero/knowledge/conventions/cross-repo-workflow.md` in
   `hero-cloud` (the bump-the-pin flow).
3. Update tracker integration: any tracker labels or routing that
   assumed single-repo issues need to handle two repos. If both
   repos hit the same Jira/Linear project, this is mostly fine;
   document the convention "title-prefix with `[cloud]` if the work
   is in `hero-cloud`."
4. Update `.hero/AGENTS.md` (or equivalent) in both repos with the
   path-dep assumption and the sibling-checkout requirement.

### Rollback plan

**If Phase 0 goes wrong** (contract shapes feel wrong):
Revert the relevant commits in `hero`. Single-repo state restored.
Zero infrastructure impact.

**If Phase 1–2 goes wrong** (split mechanics fail, history confused,
build broken on the new repo):
- The new `hero-cloud` repo is untouched-as-canonical until Phase 2
  cuts deployments over. Delete it and retry the split from a fresh
  throwaway.
- `hero` still has the cloud tree. No production impact.

**If Phase 3 goes wrong** (something we missed in `hero-cloud` and
we already deleted from `hero`):
- Revert the deletion commit on `hero`. The cloud tree returns,
  identical to its state at deletion (git history preserved on
  both sides at this point).
- Treat `hero` as the source of truth again, fix the gap, retry
  Phase 3 after the gap is closed.

**Full unwind** (we want to undo the split entirely):
1. On `hero`, revert the Phase 3 deletion commit. Cloud tree returns.
2. Apply any post-split commits from `hero-cloud` back onto `hero` —
   manually or via `git format-patch` + `git am` from the
   `hero-cloud` repo.
3. Archive `hero-cloud` repo.
4. Resume single-repo development.

Test the rollback before Phase 3: in a throwaway, run through Phase 3
and then the unwind. Confirm both binaries build at the end. Time-box
to one afternoon.

## Acceptance Criteria

- THE SYSTEM SHALL place all governance, graph, event, and token
  contract shapes used by `hero-cloud` under `contracts/` in the
  `hero` repo.
- THE SYSTEM SHALL forbid imports into `contracts/` from any other
  path in `hero`, enforced by a lint script run on every CI build of
  `hero`.
- THE SYSTEM SHALL forbid imports from `hero-cloud` into
  `github.com/hero-engine/hero/<anything except contracts/>`, enforced
  by a lint script run on every CI build of `hero-cloud`.
- WHEN a developer checks out `hero` and `hero-cloud` as siblings
  THE SYSTEM SHALL resolve `hero-cloud`'s dependency on `hero` via
  the `replace` directive without network access.
- WHEN CI builds `hero-cloud` THE SYSTEM SHALL clone `hero` at the
  SHA specified in `hero.ref` into a sibling path before running
  `go build`.
- IF `hero.ref` is missing or malformed THEN THE SYSTEM SHALL fail
  the CI build with a clear error pointing at the file.
- WHEN the `hero` CLI emits an event with a `contracts_version` lower
  than the server's `ServerMinContractsVersion` THE SYSTEM SHALL
  reject the event with a structured error naming the required
  minimum version.
- THE SYSTEM SHALL preserve git history for every file moved from
  `hero` to `hero-cloud` during the split, verified by
  `git log --follow` returning the pre-split history on the new repo.
- THE SYSTEM SHALL maintain a separate `.hero/` workspace in each
  repo, with its own `hero.json`, specs, and knowledge base.
- WHEN `hero scan` runs inside `hero-cloud` with `hero` available as
  a sibling checkout THE SYSTEM SHALL produce a complete scan of the
  cloud codebase using the path-resolved CLI.
- THE SYSTEM SHALL document the cross-repo refactor workflow
  (PR `hero` first, bump pin in `hero-cloud` PR) in
  `hero-cloud/CLAUDE.md` and in a convention spec.
- WHERE the developer is offline THE SYSTEM SHALL still build both
  repos locally via the `replace` directive, provided both sibling
  checkouts exist.

## Risks

- **Contracts seam under-exercised.** Phase 0's boundary tests fired
  zero violations only because cloud-side code did not import
  `contracts/...` at all. Phase 1's `cloud/internal/seam_smoke.go`
  canary is the first real cross-repo dependency, but it is
  deliberately minimal (Classification, Subject, SubjectType,
  Compare). As real cloud features begin to depend on more of the
  contracts surface — audit events, agent tokens, policy nodes,
  retriever interface, event envelopes — each new dependency must be
  scrutinized for cross-repo coupling that should not have crossed
  the seam. Mitigation: as cloud features land that touch new
  contract types, extend the seam smoke (or accept new genuine
  callers as replacement canaries) so any future contracts
  rename/move/ABI break still surfaces immediately on a fresh CI
  build of `hero-cloud`. The smoke is the start, not the finish, of
  seam validation.
- **Subtree split misses files or history.** Files referenced by
  import but not staged under `_split/` will fail to build on the new
  repo. Mitigation: dry-run the split into a throwaway destination
  before the real cut; run `go build ./...` on the throwaway; only
  proceed when green. (Phase 1 resolved: per-prefix `git subtree split`
  + filter-branch prefix rewrite preserved per-file history through
  `git log --follow`; the single-prefix `_split/` approach the spec
  originally recommended loses per-file history at the staging-move
  commit and was rejected.)
- **`replace` directive footgun for non-sibling layouts.** If a
  developer has `hero` checked out at a different relative path, the
  build silently fails or picks up a stale module. Mitigation:
  `scripts/hero-pin-fetch.sh` checks for `../hero` and errors with a
  clear "expected sibling checkout at ../hero" message; document
  prominently in both CLAUDE.md files.
- **Pin staleness.** Forgetting to bump `hero.ref` when a contract
  changes leaves CI building against an old contract while local dev
  works fine. Mitigation: add a `hero check` rule (or a simple grep
  in CI) that warns when `hero-cloud`'s checked-in code references
  contract shapes newer than the pinned `hero` SHA exposes. Future
  hardening.
- **Cross-repo PR latency in early life.** A contract change requires
  two PRs, two reviews, two merges. For solo work this is fine; for a
  team it adds friction. Mitigation: explicit workflow doc; eventually
  a CLI helper (`hero contract bump`) that automates the pin update
  and opens the second PR. Out of scope for v1.
- **Workspace divergence.** Two `.hero/` workspaces means knowledge
  silos — a convention discovered in `hero-cloud` doesn't reach
  `hero` automatically. Mitigation: accept it for v1; revisit if it
  bites. The fix is cross-workspace knowledge federation, which is
  itself a future spec.
- **Module name churn.** If we later want `hero-cloud` under a
  different organization or vanity URL, every import in the cloud
  codebase changes. Mitigation: pick the module name carefully
  (`github.com/hero-engine/hero-cloud` recommended); avoid bikeshedding;
  accept that this is a small-blast-radius rename if it ever happens.
- **Tooling that assumes single repo.** CI pipelines, deployment
  scripts, Hero's own MCP federation may assume `hero` is one
  codebase. Mitigation: audit before Phase 2; update or call out as
  follow-on work.
- **History rewrite anxiety.** Even though `git subtree split` is
  non-destructive on the source, the operation feels scary.
  Mitigation: do the split in a throwaway clone, never on the user's
  working tree; only push to `hero-cloud` after verification.
- **`replace` directive in committed `go.mod`.** Most Go ecosystems
  consider committed `replace` directives an antipattern because they
  break downstream consumers. We accept it because `hero-cloud` has
  no external consumers; if that ever changes (e.g., third-party
  plugins importing from `hero-cloud`), the `replace` moves to a
  developer-only `go.work` or `.local` mechanism. Flag for future
  reconsideration.

## Out of Scope

- Designing anything *inside* `hero-cloud` — handlers, schema,
  enforcement engine, dashboard. Those are existing or future feature
  specs.
- Community Edition packaging. Sibling spec; references this one as a
  dependency.
- Licensing decisions for either repo. Both stay private now; OSS for
  `hero` is a future decision informed by, but not made by, this split.
- Publishing `contracts/` as its own Go module or its own repo. The
  user's working assumption is path-dep; we honor it. A future split
  to a `hero-protocol` repo can be done after a real third consumer
  exists.
- TypeScript or other-language bindings for contract shapes. Flagged
  in Open Questions; not designed here.
- Cross-workspace spec federation. The two `.hero/` workspaces remain
  independent in v1.
- Tracker integration changes beyond the documentation hint about
  prefixing cloud-side issues.
- Automating contract version bumps. Manual for now.
- Docs site split. `docs/` stays in `hero` for now.

## Open Questions

- **Module name for `hero-cloud`.** Resolved in Phase 1:
  `github.com/hero-engine/hero-cloud`.
- **Repo hosting.** Resolved in Phase 1: both live under the
  `hero-engine` GitHub org; `hero-cloud` is private. Cross-repo
  permissions inherit org membership.
- **Single-prefix vs. multi-prefix subtree split.** Resolved in
  Phase 1: per-prefix split was chosen over the single-prefix
  `_split/` staging approach because the latter loses per-file
  history through `git log --follow` at the staging-move commit.
  Per-prefix splits + filter-branch prefix rewrite preserve full
  per-file history while still producing a clean inaugural commit
  on `hero-cloud`'s `main` (single first-parent merge with both
  subtree histories as additional parents). The acceptance
  criterion ("`git log --follow` returning the pre-split history on
  the new repo") is met.
- **Should `docker-compose.yml` move?** Audit during Phase 1: if its
  sole purpose is bringing up the cloud server for local testing, it
  moves. If it also brings up local-MCP dependencies that `hero` uses
  for its own dev loop, it stays or gets split into two files.
- **Does `internal/serve/` move?** Pending audit of what the package
  actually serves. Local MCP server stays in `hero`; cloud MCP server
  is its own new package in `hero-cloud`. Today's `serve/` may be a
  mix.
- **Eventual `hero-protocol` third repo.** Spec recommends *not* doing
  this until there's a real third consumer (e.g., OSS community
  tooling). Confirm we're comfortable revisiting in 12+ months.
- **Polyglot future.** If `hero-cloud` ever grows a TS frontend, the
  `contracts/` package needs TS bindings (probably codegen from Go).
  Flag now, design later when the frontend lands.
- **Enterprise Edition as a build flag, not a repo.** Spec recommends
  EE features as build flags on `hero-cloud`, not a forked
  `hero-cloud-ee`. Confirm; if confirmed, this informs how
  `hero-cloud`'s build matrix is set up from day one.
- **`hero check` extension.** Should `hero check` (run inside
  `hero-cloud`) verify that the local sibling `hero` checkout's SHA
  matches `hero.ref`, with a warning if drifted? Useful guardrail;
  not required for v1.
- **CI provider.** Resolved in Phase 1: GitHub Actions. The
  pin-fetch step is wired in
  `hero-cloud/.github/workflows/ci.yml` and depends on a
  `HERO_REPO_TOKEN` secret (falls back to `GITHUB_TOKEN` for the
  public-clone path once `hero` opens up; while `hero` is private,
  the secret must be configured on the `hero-cloud` repo before CI
  green is reached).

## Phase 1 Follow-Ups

Captured during Phase 1 execution; deferred from the inaugural cut
to keep its scope clean. None block Phase 2 or Phase 3.

- **~~`hero-cloud/.hero/` workspace.~~** **Landed 2026-05-15** in
  hero-cloud commit `4f93ed3`. Workspace bootstrapped with
  `hero.json`, a hero-cloud-specific `mission.md` (operational
  layer charter, relationship to core hero), empty `planning/`,
  `specs/`, `knowledge/`, `next/` (placeholder `.gitkeep`s), and
  the local `.hero/.gitignore`. `hero status` and `hero list` work
  inside hero-cloud.
- **~~`hero-cloud/CLAUDE.md` and `AGENTS.md`.~~** **Landed
  2026-05-15** in the same hero-cloud commit. CLAUDE.md documents
  the cross-repo workflow, import discipline, boundary tests, and
  the seam smoke canary, plus the standard Hero slash-command
  routing table. AGENTS.md was previously written by `hero init`'s
  auto-install and is in place.
- **`HERO_REPO_TOKEN` secret in hero-cloud repo settings.**
  Captured as a scaffold spec at
  `hero-cloud/.hero/planning/bugs/ci-token-setup/spec.md`
  (status `planning`, priority `high`). Still a one-time manual
  GitHub setup; blocks CI green and therefore Phase 2.
- **Convention specs.** Phase 4 of the spec calls for
  `contracts-import-discipline.md` in `hero` and
  `cross-repo-workflow.md` in `hero-cloud`. Phase 4 has not started.
  Tracked as a separate follow-on (next round); the CLAUDE.md
  in hero-cloud documents the rule informally in the meantime.
- **Seam smoke broadening.** Captured as a scaffold spec at
  `hero-cloud/.hero/planning/features/seam-canary-expansion/spec.md`
  (status `planning`, priority `low`). Trigger: cloud features
  begin to depend on contracts shapes beyond
  `Classification`/`Subject`/`SubjectType`/`Compare`. The expansion
  is feature-driven, not scheduled directly.
