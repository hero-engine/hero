# Hero landing page

Tracked source for the future canonical `https://heroengine.ai/` landing page.
The site is plain HTML, CSS, and JavaScript. Its visual identity remains aligned
with the hosted docs surface, while the build adds a revision marker without
mutating tracked source.

The canonical domain is not currently claimed as live. DNS, redirects, the
first production deployment, anonymous source links, and production smoke
evidence belong to `hero-public-visibility-launch-gate`.

## Layout

- `site/` — tracked landing template and static brand/social assets
- `scripts/landing_build.py` — artifact build and validation
- `scripts/test_landing_build.py` — content, accessibility, link, asset, and
  revision regressions
- `dist/` — ignored build artifact
- `wrangler.toml` — Cloudflare Worker static-assets configuration

`wrangler.toml` points only at `dist/`, so even a manual Wrangler invocation
cannot publish the tracked templates with unresolved revision placeholders.

## Build and validate

```bash
cd web/landing
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/landing_build.py check-source
python3 scripts/landing_build.py build
python3 scripts/landing_build.py check-artifact
```

The build records `source_revision`, `source_commit`, `source_digest`,
`source_dirty`, and `generated_at` in `dist/revision.json`. The digest is a
deterministic SHA-256 of every path and byte under `site/`.

In a clean CI checkout, `HERO_LANDING_REVISION` must be the exact 40-character
checked-out `HEAD`; any changed or untracked file under `site/` fails the build.
An ordinary dirty local build is labeled
`<commit>+worktree:<source-digest>` instead of claiming the commit alone.
`check-artifact` independently compares the recorded commit and digest with the
current checkout and template tree. Deployed parity still requires the launch
owner to read the production marker after deployment.

Preview the validated artifact locally:

```bash
python3 -m http.server 8080 --directory dist
```

## Launch-gated deployment

Pull requests and pushes run the build and upload `hero-landing-<revision>`;
they never deploy. A production deployment requires all of the following:

1. The repository visibility and launch gate has approved publication.
2. `LANDING_LAUNCH_APPROVED=true` is set as a repository variable.
3. Cloudflare credentials are configured as repository secrets.
4. The `Build Landing` workflow is started manually with `deploy=true`.
5. The launch owner configures and verifies `heroengine.ai` DNS, HTTPS,
   canonical redirects, anonymous destinations, and production revision parity.

Do not set the approval variable, deploy, or change DNS as part of ordinary
landing-content delivery.
