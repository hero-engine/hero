# Hero Landing Page

Public homepage served at **heroengine.ai**. Plain HTML + inlined CSS, no
build step. Mirrors the `docs/` deploy pattern via Cloudflare Workers
static assets.

## Layout

- `site/index.html` — the page
- `site/favicon.svg`, `site/og-image.svg` — brand assets
- `site/robots.txt` — allow all
- `wrangler.toml` — Cloudflare Workers config

## Preview locally

```bash
python3 -m http.server 8080 --directory site
# then open http://localhost:8080
```

## Deploy

Manual:

```bash
cd web/landing
wrangler deploy
```

Automatic (after secrets and the `DEPLOY_LANDING=true` repo variable are
set): push to `main` touching `web/landing/**` and the
`.github/workflows/landing.yml` workflow handles the deploy.

Required secrets: `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`.
