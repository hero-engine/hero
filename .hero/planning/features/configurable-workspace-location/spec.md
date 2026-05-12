---
title: Configurable Workspace Location — Hero Dir Anywhere
type: feature
status: planning
horizon: someday
priority: P2
tags: [config, deployment, workspace, future]
created: 2026-04-28
relations:
  - target: spec-prioritization
    kind: tagged-by
mission_alignment: |
  Default `.hero/` is the hidden-folder convention most projects
  expect. Some users want the corpus visible (a clearly-named folder)
  or shared across machines (network drive, shared volume) or even
  cloud-backed (S3, GCS bucket). The mission of "AI gets the right
  context at the right moment" doesn't care where the corpus
  physically lives — only that the right slice reaches the model.
  Making the location configurable widens deployment surface without
  changing the engine.
principles_check: |
  Serves #1 (it just works — defaults stay sane: `.hero/` for
  everyone unless they override). Risks none if implemented as a
  pure path-redirect. Cloud-bucket case is a much larger scope and
  may want its own spec when prioritized.
smoke: deferred
---

## Captured

User asked during the get-back-on-track recovery work
(2026-04-28): *"we should have a config setting on where the hero
folder lives - in case someone wants to keep it in a well named
folder not a hidden .hero - or a shared drive or bucket or
something - something to consider later."*

Captured here at `horizon: someday` so it isn't lost but doesn't
drown the now-actionable recovery work. Promote to `next` or `now`
when:

- A user concretely needs visible workspace folder (compliance,
  documentation, training)
- A team needs shared-drive/network-mount workspace (single corpus
  for the whole team without going through Hero Cloud sync)
- A vertical needs cloud-bucket-backed workspace (e.g., Hero Sales
  with reps across orgs sharing a corpus)

## Sketch (for future implementation)

### Filesystem-path case (small, near-term)

```yaml
# hero.json
workspace_dir: ./project-knowledge   # custom name
# or
workspace_dir: ../shared/hero        # parent / sibling dir
# or
workspace_dir: /mnt/team-share/hero  # network mount
```

`hero` looks for the workspace at `workspace_dir` if set; defaults
to `.hero/`. Existing `findProjectRoot()` logic walks up looking
for either path. Most code already uses `cfg.HeroDir(projectRoot)`
which becomes a thin wrapper.

### Cloud-bucket case (much larger)

`workspace_dir: s3://bucket/team/hero` or
`workspace_dir: gs://bucket/team/hero`. Requires:
- Bucket-fs implementation that satisfies the same interface as
  os.DirFS
- Auth (probably defer to AWS SDK / GCP SDK env-var conventions)
- Caching (don't hit the bucket on every read)
- Conflict handling for concurrent writers
- Likely overlaps significantly with team-server sync — may want
  to be a sync target rather than a primary workspace

This case is its own spec when prioritized; current scope only
covers the filesystem-path case.

## Out of scope (until promoted)

Everything. This is a someday capture, not active work.
