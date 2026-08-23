# Public Exposure Audit

Audited: 2026-08-23

## Result

**BLOCKED. Do not make `hero-engine/hero` public yet.** The current tree does
not contain a `cloud/` or `cmd/hero-cloud/` source tree, but reachable history
still contains deleted Hero Cloud source. Reachable history also contains three
deleted machine-local session databases. Both categories require a deliberate,
reviewed history rewrite before visibility changes.

The audit is evidence-only: it did not add a license, mutate repository
settings, rewrite history, inspect or modify the separate Hero Code/Hero Cloud
repositories, publish a release, or expose secret values.

## Method and evidence format

The reusable scanner checks tracked files and every named object reachable from
`--all` for high-confidence credential shapes, credential-bearing URLs,
machine-local/generated paths, proprietary Cloud paths, local absolute paths,
internal endpoints, and large blobs:

```bash
scripts/public-readiness-scan.sh --current
scripts/public-readiness-scan.sh --history
```

Output contains only severity, scope, object/ref, path, line number, finding
type, a 16-character SHA-256 fingerprint, and a redacted evidence marker. It
never prints a matched value. Exit `2` means at least one unreviewed blocking
finding. `scripts/public-readiness-baseline.tsv` converts only an exact
path/type/fingerprint match into `reviewed`; a new path or changed line gets a
new fingerprint and blocks again. The scanner self-test exercises that
mutation behavior.

Additional read-only checks covered tracked and historical path names, object
sizes, commit identity metadata, public source URLs, the completed licensing
inventory, workflow/policy destinations, `origin/HEAD`, and repository API
availability. Object fingerprints below are identifiers, not content excerpts.
An anonymous request to the repository returned `404`; the hosted docs root
returned `200`. No settings mutation was attempted.

## Blocking findings

### PR-1 — proprietary Hero Cloud source remains in reachable history

| Evidence | Value |
|---|---|
| Paths | 45 source-file blobs / 52 object-list paths including 7 tree prefixes under `cloud/**` and `cmd/hero-cloud/**` |
| Object-list fingerprint | `2c262a57c097672d` |
| Introduced | commit `982742de82e4c767fdac3f6468c301d1d0d65705` (`v0.8.0`) |
| Removed from current tree | commit `17f79c6ff2a4e4c074a474cc88456d98a7699ec3` |
| Reachability | `main`, `origin/main`, historical branches, and tags from `v0.8.0` through `v0.33.0` |
| Disposition | **BLOCKER — repository owner / release engineer** |

This is actual historical source, not a name-only reference or public
interface contract. Because Hero Cloud remains proprietary, deletion from HEAD
is insufficient: making the repository public would publish these blobs.

Remediation: make a recoverable backup, perform a reviewed `git filter-repo`
rewrite removing exactly `cloud/**` and `cmd/hero-cloud/**` from every public
ref, replace affected tags/branches deliberately, and validate a fresh mirror.
Do not run that destructive rewrite as part of readiness preparation.

### PR-2 — machine-local session databases remain in reachable history

| Historical path | Object ref | Size | Content fingerprint | Disposition |
|---|---|---:|---|---|
| `.hero/sessions/30f03a503761d2fd/refs.db` | `d3d82658f8539b77d711e0ffd4dd8a5dc8c767dd` | 49,152 B | `25e42cf09b93497c` | BLOCKER |
| `.hero/sessions/638463b8eb8ad320/refs.db` | `c59224dbd2ae3c00140e15e6055ad687591dc2fb` | 69,632 B | `099dde5d2f4f3986` | BLOCKER |
| `.hero/sessions/9f842f314f777d27/refs.db` | `e262a76f82cdbb408758f0da0fd0fd08e25f0181` | 32,768 B | `bbcc0d3cdeb2e130` | BLOCKER |

These SQLite files are ignored now and absent from HEAD, but machine-local
session state is unsuitable for anonymous publication. Remove the three exact
paths during the same reviewed history rewrite and prove the fingerprints are
absent from a fresh mirror.

## Current-tree credential review

The current scan returned four high-confidence shapes, all manually
dispositioned and recorded in the narrow fingerprint baseline without printing
values:

| File/ref | Type | Fingerprint | Disposition |
|---|---|---|---|
| `internal/config/integrations_test.go:197` / `HEAD` | credential-bearing URL | `55402dd736cce191` | False positive: invalid-provider fixture |
| `internal/config/integrations_test.go:249` / `HEAD` | credential-bearing URL | `4a9162536a95cad9` | False positive: invalid-provider fixture |
| `web/docs/src/configuration/tracker-setup.md:25` / `HEAD` | provider token | `60312e2c3ce61527` | False positive: visibly synthetic placeholder |
| `internal/wiki/wiki.go:167` / `HEAD` | credential-bearing URL | `321f18808ab399d0` | No stored secret; real runtime design review—token is interpolated into a child-process URL and should be hardened separately |

No committed current-tree credential was confirmed. The scanner reports these
lines as reviewed, while any changed or newly located match blocks again. The
wiki behavior is an operational security concern, not evidence that a
credential is stored in Git.

The authoritative reachable-history scan also found two initially unreviewed
provider-token shapes in generated documentation objects. Redacted inspection
proved both are the same 24-character synthetic placeholder (value fingerprint
`cad81719849e893c`) rendered by deployment commit
`7551c6a1224c869a5711368a43e7412a9b88982d`, reachable only from
`origin/gh-pages`. Their exact line fingerprints—`0fec53d4746fa334` in
`configuration/tracker-setup/index.html` and `f039bbb3c0355726` in
`search/search_index.json`—are now reviewed baseline entries. No real provider
token was confirmed; changing either generated line reblocks it.

## Personal and internal data review

- Reachable commit metadata has six distinct non-empty email fingerprints: one
  GitHub noreply identity and five address-bearing identities. Names and
  addresses are normal Git history metadata but become anonymous public data.
  The owner must either accept that exposure or include mail rewriting in the
  history plan. Recorded fingerprints: `8bd3c67833d2`, `73ce7451f04e`,
  `ce702afbbdcb`, `01ae27589508`, `3c205d8fc749`; one malformed/empty identity
  fingerprint is `e3b0c44298fc`.
- The current scanner found 101 local absolute-path references. Thirty-five
  occur in the generated `.hero/QUEUE.md`; the remainder are primarily test
  fixtures and archived delivery evidence. They disclose workstation layout,
  not credential values. Disposition: **OWNER REVIEW**—sanitize authored public
  prose where practical and explicitly accept fixture/audit evidence before
  launch.
- Reachable history contains 25,937 absolute-path review rows across 54 paths,
  645 objects, and 608 line fingerprints. Repeated generated snapshot revisions
  dominate the count: 532 historical `.hero/QUEUE.md` objects account for
  25,761 rows. This is not a secret match, but it materially expands
  workstation-layout exposure. Disposition: **OWNER REVIEW**—either accept that
  historical project evidence as public or include affected projected/evidence
  paths in the rewrite plan.
- Two current internal-endpoint shapes were found in a team-server spec and a
  connection implementation example. Their fingerprints are
  `e14d4f20f0d032f2` and `3ed726289bf74d2d`; both require owner review but no
  credential value was detected.
- Reachable history contains 15 internal-endpoint review rows across 4 paths,
  11 objects, and 5 fingerprints. They are examples/spec evidence rather than
  confirmed stored credentials, but remain **OWNER REVIEW** before exposure.

## Public-link and policy review

- Public-facing root docs, hosted-doc source, landing source, and new policy
  files contain no direct `github.com/hero-engine/hero` source/issue links while
  the repository is private.
- Contribution, support, issue, pull-request, conduct, and security surfaces
  are present. Each states the current private/no-license boundary where it
  matters.
- The security intake is intentionally labeled unavailable until GitHub private
  vulnerability reporting is enabled. That setting is a visibility-gate
  blocker, not an implied live route.
- Issue forms require reporters to remove credentials/private data and redirect
  vulnerabilities to `SECURITY.md`.

## Proprietary names versus proprietary content

Current tracked specs, decisions, contracts, and handoff documents refer to
Hero Code and Hero Cloud. Those name-only product boundaries and public
interface contracts are not automatically copied proprietary source. The
completed licensing packet includes Hero-authored tracked planning material in
the proposed grant while explicitly excluding the separate products.

The historical `cloud/**` and `cmd/hero-cloud/**` source is categorically
different and blocks visibility. No current tracked path under either source
tree was found. Sprout remains a separate MIT dependency; this audit neither
inspected nor modified its repository.

## Large, generated, and third-party material

- Current tracked files over 1 MiB are the 24,263,680-byte embedded model and
  the 1,384,588-byte landing social image. The model's pinned source, hashes,
  MIT lineage, and notice obligations are already recorded in the licensing
  inventory; the social image is a first-party launch asset.
- The history scan has one large-blob review:
  `internal/embeddings/defaultmodel/weights.bin`, fingerprint
  `7bddd1e5a1990fc1`. It is the same licensed embedded-model category already
  cleared with mandatory attribution in the licensing packet.
- No tracked `dist/`, `bin/`, `.build/`, `.hero/cache/`, graph/index database,
  environment, key, or certificate path exists at HEAD.
- Third-party redistribution remains governed by the completed licensing
  packet. Public visibility does not waive its required Apache/third-party
  notices or the final generated-docs validation.

## Readiness decision

Policy and template preparation is complete, but repository exposure is not.
Close PR-1 and PR-2, resolve the owner-review items, satisfy the licensing and
host-settings checklists, require zero unreviewed scanner blockers against a
fresh clone, and obtain explicit visibility approval. Until then, fail closed.

The completed authoritative history scan emitted 45 proprietary-path blockers,
3 machine-local-path blockers, 2 initially unreviewed synthetic provider-token
shapes (now exactly dispositioned), 16 reviewed credential-URL shapes, 7
already-reviewed provider-token shapes, 25,937 absolute-path review rows, 15
internal-endpoint review rows, and 1 licensed large-history-blob review. It
exited `2` because PR-1 and PR-2 remain unresolved, as intended.
