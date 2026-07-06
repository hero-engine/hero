---
title: "Chat Pack Disposition — Formalize domains/chat as a Client-Embedded Pack"
slug: chat-pack-disposition
type: decision
status: planning
priority: P2
domain: engineering
created: 2026-07-06
tags: [content, domains, chat, hero-code, packs, audit-followup, decision]
relations:
  - target: content-remediation
    kind: parent
  - target: hero-content-audit
    kind: related
  - target: multi-domain-context-activation
    kind: builds-on
---

# Chat Pack Disposition — Formalize domains/chat as a Client-Embedded Pack

## Context

The content audit graded the chat pack as dead content. F9
(findings-commands.md, graded S1→S2): `domains/chat/` contains only
`commands/` — six files (ask-corpus, capture, discover, note, space, why)
with no AGENTS.md, no agents/, no skills/. `content.go` embeds only
engineering, sales, and pm (go:embed lines 17–24); `DomainFS("chat")`
returns the domain-not-found error; `AvailableDomains()` = {engineering,
sales, pm}. No `hero install --domain` value can ship the pack. F29:
ask-corpus.md instructs `semantic_search` / `read_file` (tool identifiers
no install target exposes) and space.md instructs "Use the `SpaceStore`
API … if running outside the GPUI shell" — a specific chat client's
private internals, unscoped. The routing audit (findings-routing.md, S3)
adds the trap: if chat were embedded as-is, `loadPackAgentsMdBody` would
fall back to the engineering routing table with a stderr warning
(`internal/install/agents_md.go:105-108`, verified current post-177e8a1),
routing chat sessions to `/diagnose`, `/design`, `/deliver`, `/mock` —
none of which chat ships.

**The audit's "dead content" verdict misses the build-time consumer.**
The sibling repo `../hero-code` (Rust/GPUI chat client) consumes
`domains/chat` directly: `crates/hero-core/build.rs` iterates *every*
directory under `${HERO_SRC}/domains/` (not a hardcoded list) and stages
each domain's `agents/`, `skills/`, `commands/` into `OUT_DIR/hero-content/<domain>/`
for `include_dir!` embedding. `src/hero/domains.rs` registers chat as one
of three live domains (`DOMAINS: [engineering, chat, pm]`) with its own
`UiProfile`, and `embedded.rs::load_all_for("chat")` loads the staged
chat commands. So chat is dead only for `hero install`; for hero-code it
is live, shipping content. Note two staging details verified in build.rs:
it copies only the three category directories (a root `AGENTS.md` would
*not* be staged), and it does not stage `core/` at all — so in hero-code,
chat's capture/discover/note/why are not overlay shadows of core; they
are the only copies the chat domain loads.

The accepted framing comes from the related decision
`.hero/knowledge/decisions/multi-domain-context-activation.md`: domains
resolve by context, hero-code picks the active domain per window/space,
and the engine stays single-domain-per-install with `OverlayFS(domain,
core)` unchanged. That decision already treats chat as a per-context
domain pack hero-code activates — same-named commands (chat's /capture,
/discover, /note, /why vs core/pm) are legitimate per-domain
specializations selected by activation, never merged.

One more engine fact this spec must not disturb: the parity test
(`content_parity_test.go:32`) walks `AvailableDomains()`, so chat is
outside its core-shadow enforcement today — correctly so, since chat
never overlays core at install time. The recommendation below keeps
`AvailableDomains()` unchanged, so the parity test needs no modification.

## Goal

A recorded, executed decision on the fate of `domains/chat`: the pack's
relationship to `hero install` and to hero-code is intentional and
documented in code (comment + test, not accident), F29's client-internal
wording is scoped or generalized, and the routing trap (S3) cannot fire
if chat is ever embedded later. F9, F29, and the chat-routing S3 findings
close with an auditable trail.

## Kickoff

Decides what to do with domains/chat — the six-command pack that
hero-code embeds at build time but `hero install` can't ship.
Recommendation: option (a), formalize chat as a client-embedded pack —
keep it under domains/, make the DomainFS exclusion intentional
(comment + test), fix ask-corpus/space client-internals wording, add a
minimal chat AGENTS.md.

**Status:** planning — options analyzed, recommendation drafted,
awaiting review.

**Pick up at:** review ## Options and ## Recommendation; if accepted,
execute ## Changes starting with the content.go comment and the
domain-intentionality test.

→ `.hero/planning/initiatives/content-remediation/chat-pack-disposition/spec.md`

**Files:** content.go:53-69, content_parity_test.go:32,
domains/chat/commands/space.md, domains/chat/commands/ask-corpus.md,
../hero-code/crates/hero-core/build.rs

## Options

### (a) Formalize chat as a client-embedded pack — keep under domains/, explicitly NOT installable

Keep `domains/chat/` where it is (build.rs stages it from there today).
Make the engine's exclusion intentional rather than accidental: a
comment in `content.go` naming the build-time consumer and the reason
chat has no embed/DomainFS case, plus a test asserting both that
`DomainFS("chat")` errors and that every directory under `domains/` is
either installable (`AvailableDomains()`) or on a documented
client-embedded allowlist — so the next stray domain directory fails CI
instead of silently reproducing this situation. Fix F29 by naming
capabilities instead of tool identifiers in ask-corpus.md and scoping
space.md's SpaceStore instruction to hero-code. Author a minimal
`domains/chat/AGENTS.md` routing only to the six commands chat ships.

- **Pro:** zero cross-repo breakage — build.rs keeps staging chat
  unchanged; matches the context-activation decision exactly (chat is a
  per-context pack, hero-code activates it; engine contract untouched).
- **Pro:** smallest change that converts every accident into a decision:
  the exclusion, the wording, and the routing surface all become
  deliberate and test-guarded.
- **Pro:** the AGENTS.md preempts the S3 misroute for any future
  embedder. Honest caveat: neither current consumer reads it —
  `hero install` can't ship chat, and hero-code's build.rs stages only
  category dirs while its Layer 1b roster (`hero_surface.rs`) is built
  from the content registry, not from AGENTS.md. Its immediate value is
  as the pack's routing spine (pack completeness + the audit's "author
  a chat AGENTS.md before or with embedding" fix shape), consumed the
  day anyone wires chat into an instructions-file path. If reviewers
  judge that too speculative, this one step can be deferred without
  weakening the rest of the option.
- **Con:** the engine repo permanently hosts content it never installs;
  the domains/ directory means two different things (installable pack
  vs. client-embedded pack). The comment + allowlist test is the
  mitigation — the distinction becomes legible.

### (b) Wire chat as a full installable domain — embed + AGENTS.md + DomainFS case

Add `//go:embed domains/chat/...`, a `DomainFS` case, chat in
`AvailableDomains()`, and a chat AGENTS.md.

- **Pro:** uniform treatment — every domain under domains/ is
  installable; the S3 trap closes because the AGENTS.md ships.
- **Con:** no user. Chat's commands presume a chat client's session
  model (Spaces, corpus-scoped Q&A inside a conversation shell); no CLI
  harness user has asked for `--domain chat`, and the sales pack already
  demonstrates what a shipped-but-unused domain produces: a stream of
  phantom-surface audit findings (F10 et al.) that this initiative is
  busy cleaning up.
- **Con:** real new maintenance surface. Once chat enters
  `AvailableDomains()`, the parity test walks it — chat's capture,
  discover, note, and why shadow `core/commands/*` paths and would each
  need a justified `core_fork:` annotation; the pack would also need the
  AGENTS.md heading-structure care flagged for pm/sales, README
  hygiene, and install-matrix testing across six targets.
- **Con:** cuts against the context-activation decision's grain: the
  engine stays single-domain-per-install and chat was explicitly
  anticipated as a hero-code-activated pack. Installing chat into a CLI
  workspace gives an agent commands (`/space`) that reference UI
  machinery the harness doesn't have — recreating F29 at install scale.

### (c) Relocate chat out of domains/ into a hero-code-owned location

Move the six files into hero-code's repo (or a non-domains/ path in
hero), on the theory that client-specific content should live with the
client.

- **Pro:** the engine repo stops hosting non-installable content; the
  "domains/ = installable" invariant would hold without an allowlist.
- **Con:** breaks the working consumer. build.rs stages
  `${HERO_SRC}/domains/` — relocation requires a coordinated cross-repo
  change (build.rs learns a second content root, or hero-code vendors
  the files) and forfeits what the current arrangement provides free:
  chat content versioned with the Hero content corpus, stamped by the
  same `git describe` version, staged by the same loop as engineering
  and pm.
- **Con:** contradicts the context-activation decision's architecture,
  which names hero-code's three domains (engineering, chat, pm) as Hero
  domain packs sourced from the hero repo — pm's UI also lives in
  hero-code while its pack lives here; chat would become the lone
  exception.
- **Con:** highest coordination cost of the three for zero functional
  gain — nothing a user or agent touches behaves differently afterward.

## Recommendation

**Option (a).** The audit's fix shape for F9 was "wire the domain or
move the files out" — but both horns were framed before the hero-code
consumption evidence. Chat is not dead content; it is a live
client-embedded pack whose engine-side exclusion happens to be
undocumented. Option (b) ships the pack to an audience that doesn't
exist and buys a sales-pack-shaped maintenance liability; option (c)
breaks a working build to satisfy a directory-naming instinct and walks
back the context-activation architecture. Option (a) is the only one
that changes no behavior for either consumer while converting every
accidental property — the DomainFS exclusion, the unscoped client
internals, the missing routing surface — into a documented, test-guarded
decision. It is also cheaply reversible: if a real `--domain chat`
install audience ever appears, option (b) is a small additive change on
top of (a), and the AGENTS.md and F29 fixes will already be done.

Consequences: domains/ officially holds two kinds of packs (installable
and client-embedded), legible via the content.go comment and enforced by
the allowlist test; `AvailableDomains()` and the parity test are
unchanged (chat stays outside the core-shadow walk — correct, since chat
never overlays core at install); hero-code needs no change and notices
nothing (build.rs ignores a root AGENTS.md by construction).

## Changes

1. **content.go — document the intentional exclusion.** Extend the
   package comment (or add one at `DomainFS`/`AvailableDomains`,
   content.go:53-69 and 259-265) stating: `domains/chat` is a
   client-embedded pack consumed at build time by hero-code
   (`crates/hero-core/build.rs` stages every `domains/<name>/`
   {agents,skills,commands} directory); it is deliberately not embedded,
   not in `AvailableDomains()`, and not installable, per this spec and
   the multi-domain-context-activation decision. No behavior change.
2. **New test — make the exclusion and the domains/ taxonomy enforceable**
   (in `content_parity_test.go` or a sibling `content_test.go`, package
   `hero`):
   - Assert `DomainFS("chat")` returns an error mentioning available
     domains.
   - Read the source tree's `domains/` directory (tests run at repo
     root) and assert every subdirectory is either in
     `AvailableDomains()` or in a `clientEmbeddedDomains = {"chat"}`
     allowlist with a comment pointing at this spec — a new unwired
     domain directory fails CI instead of becoming the next F9.
3. **domains/chat/commands/ask-corpus.md — fix F29 (capabilities, not
   tool identifiers).** Replace the step-1 instruction naming
   `semantic_search` / `read_file` with capability language: search the
   corpus with whatever semantic or keyword search the session exposes
   (e.g. a semantic-search tool where available, otherwise grep) and
   read the matching files. No structural changes.
4. **domains/chat/commands/space.md — fix F29 (scope the client
   internals).** Rewrite step 3 so the `SpaceStore` API path is
   explicitly scoped to the hero-code (GPUI) client, and the generic
   path — surface a brief summary the user can paste into a New-Space
   dialog — is the default for any other host. Same behavior in
   hero-code; no more unscoped private API instruction.
5. **domains/chat/AGENTS.md — author the minimal routing spine.** H1
   title plus a natural-language routing table covering only the six
   shipped commands (ask-corpus, capture, discover, note, space, why),
   `###` section headings (the pack H1 becomes an H2 at install
   assembly, per findings-routing.md — engineering's `###` structure is
   the one that nests correctly), no relative links, no references to
   commands or CLI surfaces the pack doesn't ship. This preempts the S3
   engineering-fallback misroute (`agents_md.go:105-108`) for any future
   embedding; note in the file header that no current consumer reads it.
6. **Close the audit loop.** Mark F9, F29, and the chat-routing S3
   finding resolved-by-this-spec in the content-remediation initiative's
   tracking (progress section / ledger), recording that F9's "dead
   content" framing was corrected by the hero-code build-time
   consumption evidence.

## Boundaries

- **No change to `AvailableDomains()`, `DomainFS` cases, or go:embed
  directives** — chat does not become installable; that is the decision.
- **No hero-code changes.** build.rs, domains.rs, embedded.rs, and the
  chat UiProfile are untouched; making hero-code consume the new chat
  AGENTS.md (build.rs staging + a Layer 1b or preamble surface) is
  hero-code-side design work, out of scope here.
- **No parity-test modification** — chat remains outside the
  `AvailableDomains()` walk; do not add `core_fork:` annotations to chat
  files (they never overlay core at install).
- **No new chat content** — no agents/, no skills/, no seventh command;
  this spec disposes of what exists.
- **Other packs' findings** (pm/sales AGENTS.md structure, F10 phantom
  surfaces, README shipping) belong to sibling children of
  content-remediation, not here.

## Risks

- **The allowlist test reads the source tree, not the embed.** It only
  runs where `domains/` exists on disk (repo checkouts — fine for CI);
  keep the assertion tolerant of running from the repo root only, and
  skip (with `t.Skip`) if `domains/` is absent, so downstream `go test`
  of the module as a dependency doesn't false-fail.
- **AGENTS.md with no consumer can rot.** Mitigated by keeping it
  minimal (route only the six commands) and by the audit's drift checks;
  if reviewers judge it speculative, defer step 5 — the decision holds
  without it.
- **F29 rewording could regress hero-code behavior** if its chat
  sessions rely on the literal `semantic_search`/`SpaceStore` strings.
  Low risk — the wording keeps those identifiers, scoped ("where
  available" / "in the GPUI shell") rather than removed; verify with a
  hero-code rebuild.
- **Taxonomy precedent:** blessing "client-embedded pack" as a category
  may invite more non-installable content into domains/. The allowlist
  test is the gate — additions require touching a test that cites this
  spec.

## Acceptance Criteria

- THE SYSTEM SHALL document in content.go that domains/chat is an intentionally non-installable, client-embedded pack, naming hero-code's build.rs as the consumer.
- WHEN `DomainFS("chat")` is called THE SYSTEM SHALL return the domain-not-found error, and a test SHALL assert it.
- IF a directory exists under domains/ that is neither in `AvailableDomains()` nor in the documented client-embedded allowlist THEN THE SYSTEM SHALL fail the content test suite.
- THE SYSTEM SHALL keep `AvailableDomains()` returning exactly engineering, sales, and pm.
- THE SYSTEM SHALL express ask-corpus.md's search step as capabilities rather than harness-specific tool identifiers.
- THE SYSTEM SHALL scope space.md's SpaceStore instruction to the hero-code (GPUI) client with the paste-a-summary path as the default elsewhere.
- THE SYSTEM SHALL ship a domains/chat/AGENTS.md whose routing table references only the six commands the chat pack contains.
- WHEN `go test ./...` runs at the repo root THE SYSTEM SHALL pass with the existing parity test unmodified.

## Validation

- `go test ./...` — new exclusion/allowlist assertions pass; existing
  `TestDomainPacks_NoUnannotatedCoreShadows` passes unmodified.
- `go build ./...` — content.go comment-only change compiles.
- `grep -n "semantic_search\|SpaceStore\|GPUI" domains/chat/commands/*.md`
  — remaining occurrences are scoped/capability-phrased per F29's fix
  shape (identifiers may appear, but only inside "where available" / "in
  the hero-code GPUI shell" framing).
- `hero domain list` (or equivalent) — output unchanged: engineering,
  sales, pm.
- Cross-repo smoke: `cargo build -p hero-core` in `../hero-code` — build
  log still reports chat staged with its 6 command files; the new
  domains/chat/AGENTS.md does not appear in staged output (build.rs
  copies category dirs only).
- Manual: read domains/chat/AGENTS.md against findings-routing.md's
  structural findings — H1 + `###` sections, no relative links, no
  phantom command references.
