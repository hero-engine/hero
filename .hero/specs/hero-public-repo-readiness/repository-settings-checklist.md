# Repository Settings Checklist

Prepared: 2026-08-23

This is a launch-gate checklist, not a record of applied settings. The current
Git identity can reach `origin` over its configured SSH host alias, but GitHub's
API returns `404` for `hero-engine/hero`; an anonymous HTTP request also returns
`404`, while `https://docs.heroengine.ai/` returns `200`. Host settings remain
unverified and unchanged. Repository visibility must stay private until every
blocking row below is closed and the owner gives explicit approval.

## Public identity

| Setting | Required value | Current evidence | Gate |
|---|---|---|---|
| Owner/name | `hero-engine/hero` | Configured `origin`; authenticated `git ls-remote` succeeds | VERIFY in GitHub settings |
| Description | `Durable project memory and verified delivery for AI-assisted engineering.` | API unavailable to current identity | SET before visibility change |
| Homepage | Deployed canonical landing URL | Landing source is prepared; deployment/DNS is a separate gate | BLOCKED until deployment is proven |
| Topics | `ai`, `developer-tools`, `project-memory`, `spec-driven-development`, `golang` | API unavailable | SET before visibility change |
| Visibility | Private until explicit owner approval | Anonymous repository request returns `404`; no visibility mutation was dispatched | HOLD |

## Collaboration

| Setting | Required value | Current evidence | Gate |
|---|---|---|---|
| Default branch | `main` | `git ls-remote --symref origin HEAD` resolves to `main` | READY |
| Issues | Enabled | Bug and feature forms are prepared under `.github/ISSUE_TEMPLATE/` | ENABLE at launch |
| Discussions | Disabled initially | Support is intentionally routed through documentation and issues; no moderation plan exists yet | CONFIRM at launch |
| Wiki | Disabled | Documentation belongs in tracked Markdown and the hosted docs site | CONFIRM at launch |
| Merge methods | Squash enabled; merge commits/rebase per maintainer preference | Not observable | DECIDE at launch |
| Auto-delete head branches | Enabled | Not observable | ENABLE at launch |

## Main-branch protection

| Control | Required value | Current evidence | Gate |
|---|---|---|---|
| Pull requests | Required; no direct pushes except an explicitly documented emergency path | Protection API unavailable | ENABLE before visibility change |
| Approvals | At least 1 approving review; dismiss stale approvals | Protection API unavailable | ENABLE before visibility change |
| Conversation resolution | Required | Protection API unavailable | ENABLE before visibility change |
| Force pushes/deletion | Disabled | Protection API unavailable | VERIFY before visibility change |
| Required checks | `Test / test`, `Smoke / smoke`, `Deploy Docs / build`, `Build Landing / build` | All four workflow jobs exist and run on pull requests | REQUIRE before visibility change |

Do not require the tag-triggered `Release / goreleaser` job on pull requests; it
publishes only from the separately approved release path.

## Security

| Control | Required value | Current evidence | Gate |
|---|---|---|---|
| Private vulnerability reporting | Enabled | API unavailable; `SECURITY.md` deliberately labels the route unavailable until enabled | BLOCKER |
| Secret scanning | Enabled for the repository | API unavailable | BLOCKER |
| Push protection | Enabled | API unavailable | BLOCKER |
| Dependency graph and Dependabot alerts | Enabled | API unavailable | ENABLE before visibility change |
| Actions default token | Read-only; elevate per job only | Existing workflows declare narrow permissions | VERIFY in repository settings |

## Final launch sequence

1. Rewrite reachable history to remove the exact blocked paths in
   `exposure-audit.md`, then prove a fresh mirror contains none of their object
   fingerprints.
2. Resolve or explicitly accept the redacted personal-data review findings.
3. Close the Apache grant and third-party notice preconditions; confirm the
   root license exists before accepting contributions.
4. Deploy and verify the canonical landing/docs destinations.
5. Apply and re-read every setting above through an authorized GitHub owner
   session. Capture values, not credentials.
6. Run `scripts/public-readiness-scan.sh --all` against a fresh clone. It must
   exit zero with only exact, reviewed path/type/fingerprint baseline matches;
   any new or mutated match remains a blocker.
7. Obtain explicit owner approval, then—and only then—change visibility.
