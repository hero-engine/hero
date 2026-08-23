# Third-Party License Inventory

Audit date: 2026-08-23

This inventory distinguishes what the release binary redistributes, what the hosted documentation delivers, and what is only used to build or test the project. “Compatible” means no conflict was found with distributing Hero under Apache-2.0 when the recorded notice obligations are met; it is not a claim that the dependency itself becomes Apache-2.0.

## Hero release binary

The release closure was taken from `go list -deps ./cmd/hero` across Darwin, Linux, and Windows and checked with `google/go-licenses v1.6.0`. GoReleaser ships only the `hero` executable in each archive.

| Component | Version | License | Source / license evidence | Redistribution obligation | Conclusion |
|---|---:|---|---|---|---|
| Go runtime and linked standard library | 1.26.4 | BSD-3-Clause | `go.dev/LICENSE` | Reproduce the Go copyright notice, conditions, and disclaimer with binary distributions | Compatible |
| BurntSushi/toml | 1.6.0 | MIT | `github.com/BurntSushi/toml`, `COPYING` | Retain license and copyright | Compatible |
| dustin/go-humanize | 1.0.1 | MIT | `github.com/dustin/go-humanize`, `LICENSE` | Retain license and copyright | Compatible |
| google/uuid | 1.6.0 | BSD-3-Clause | `github.com/google/uuid`, `LICENSE` | Reproduce notice and disclaimer | Compatible |
| inconshreveable/mousetrap | 1.1.0 | Apache-2.0 | `github.com/inconshreveable/mousetrap`, `LICENSE` | Include Apache license; preserve notices; Windows release only | Compatible |
| mattn/go-isatty | 0.0.20 | MIT | `github.com/mattn/go-isatty`, `LICENSE` | Retain license and copyright | Compatible |
| ncruces/go-strftime | 1.0.0 | MIT | `github.com/ncruces/go-strftime`, `LICENSE` | Retain license and copyright | Compatible |
| remyoudompheng/bigfft | 24d4a6f8daec | BSD-3-Clause | `github.com/remyoudompheng/bigfft`, `LICENSE` | Reproduce notice and disclaimer | Compatible |
| spf13/cobra | 1.10.2 | Apache-2.0 | `github.com/spf13/cobra`, `LICENSE.txt` | Include Apache license; preserve notices | Compatible |
| spf13/pflag | 1.0.9 | BSD-3-Clause | `github.com/spf13/pflag`, `LICENSE` | Reproduce notice and disclaimer | Compatible |
| golang.org/x/crypto | 0.50.0 | BSD-3-Clause | Go source repository, `LICENSE` | Reproduce notice and disclaimer | Compatible |
| golang.org/x/sys | 0.44.0 | BSD-3-Clause | Go source repository, `LICENSE` | Reproduce notice and disclaimer | Compatible |
| golang.org/x/term | 0.42.0 | BSD-3-Clause | Go source repository, `LICENSE` | Reproduce notice and disclaimer | Compatible |
| gopkg.in/yaml.v3 | 3.0.1 | MIT | `github.com/go-yaml/yaml`, `LICENSE` | Retain license and copyright | Compatible |
| modernc.org/sqlite | 1.53.0 | BSD-3-Clause | `gitlab.com/cznic/sqlite`, `LICENSE` | Reproduce notice and disclaimer | Compatible |
| modernc.org/libc | 1.73.4 | BSD-3-Clause plus bundled third-party terms | `gitlab.com/cznic/libc`, `LICENSE` and `LICENSE-3RD-PARTY.md` | Ship the main license and complete third-party notice file | Compatible with mandatory bundled notices |
| modernc.org/mathutil | 1.7.1 | BSD-3-Clause | `gitlab.com/cznic/mathutil`, module `LICENSE` | Reproduce notice and disclaimer; automated classifier reported Unknown, manually resolved from the shipped license text | Compatible |
| modernc.org/memory | 1.11.0 | BSD-3-Clause | `gitlab.com/cznic/memory`, `LICENSE-GO` | Reproduce notice and disclaimer | Compatible |

### Embedded model asset

| Asset | Local evidence | Upstream | License | Transformation | Obligation | Conclusion |
|---|---|---|---|---|---|---|
| `hero-embed-v1` weights, vocabulary, config | `internal/embeddings/defaultmodel/SOURCE.md`; SHA-256 `bf344479...cce49` (weights), `28f351e6...ef6b` (vocab), `a56e14a3...b046` (config) | `minishlab/potion-base-8M` at revision `bf8b056651a2c21b8d2565580b8569da283cab23`, distilled from `BAAI/bge-base-en-v1.5` | Both lineages are MIT: Model2Vec copyright 2024 Thomas van Dongen; FlagEmbedding copyright 2022 staoxiao; texts captured in `potion-base-8M-MIT.txt` | Pinned `tools/distill-embeddings.py` exports the upstream Model2Vec tensors and removes unused subword/special-token rows | Credit Minish Lab / Model2Vec and BAAI / FlagEmbedding, link both models, and reproduce both MIT notices with binary distributions | Compatible with mandatory attribution; local hashes reproduced from the pinned source |

The model source, revision, license declaration, transformation, and output hashes are captured beside the asset. A clean regeneration from that pinned revision reproduced all three local hashes.

## Sprout and non-release Go tooling

| Component | Use | Version | License evidence | Redistributed by Hero release? | Conclusion |
|---|---|---:|---|---|---|
| `github.com/bdwheeler/sprout/go` | Offline mock-tracker seed loader and tests | `v0.1.1-0.20260822024445-cd3f0c4a2208` | The exact module ZIP contains root `LICENSE`; MIT; source commit `cd3f0c4a2208` | No; `cmd/mock-tracker-server` is not in GoReleaser archives | Compatible; exact consumed artifact is licensed |
| `go.uber.org/goleak` | Tests | 1.3.0 | MIT, upstream `LICENSE` | No | Compatible |
| `github.com/shopspring/decimal` | Test/support closure | 1.4.0 | MIT, upstream `LICENSE` | No | Compatible |

Other modules present only through unused, platform, or test graph edges are not linked into the released `hero` executable. The release gate must regenerate the package-level report for all target platforms and fail on any new Unknown, reciprocal, or restricted license.

## Hosted documentation build

`requirements-docs.txt` currently resolves the following build environment. These Python packages are build inputs, not included in Hero CLI archives. Material for MkDocs emits JavaScript, CSS, search code, and icons into the hosted site, so its MIT notice and identifiable bundled component notices must be carried on a public third-party notices page.

| Package(s) | Resolved version(s) on audit date | License(s) | Distribution treatment |
|---|---|---|---|
| MkDocs | 1.6.1 | BSD-2-Clause | Build input; preserve notice in site attribution packet |
| Material for MkDocs; material extensions | 9.7.7; 1.3.1 | MIT | Generated CSS/JS/icons are delivered to browsers; preserve MIT attribution |
| Babel; Click; Jinja2; Markdown; MarkupSafe | 2.18.0; 8.4.2; 3.1.6; 3.10.3; 3.0.3 | BSD-3-Clause | Build/runtime inputs to generator; compatible |
| Pygments | 2.21.0 | BSD-2-Clause | Build input; generated highlighting output does not relicense the package |
| ghp-import; Requests; Watchdog | 2.1.0; 2.34.2; 6.0.0 | Apache-2.0 | Build/deploy inputs; compatible |
| packaging | 26.3 | Apache-2.0 OR BSD-2-Clause | Build input; compatible |
| pathspec | 1.1.1 | MPL-2.0 | Build input only; no package source is copied into the site |
| certifi | 2026.7.22 | MPL-2.0 | Build/network input only; CA bundle is not copied into the site |
| backrefs; charset-normalizer; mergedeep; mkdocs-get-deps; paginate; platformdirs; pymdown-extensions; PyYAML; pyyaml-env-tag; six; urllib3 | 8.0; 3.5.1; 1.3.4; 0.2.2; 0.5.7; 4.11.3; 11.0.2; 6.0.3; 1.1; 1.17.0; 2.7.0 | MIT | Build inputs; compatible |
| colorama | 0.4.6 | BSD | Build input; compatible |
| idna | 3.19 | BSD-3-Clause | Build/network input; compatible |
| python-dateutil | 2.9.0.post0 | Apache-2.0 OR BSD | Build input; compatible |

The current `>=` requirements are not a reproducible lock. The hosted-docs remediation child must bound or lock the two direct requirements and rerun this inventory before the final deploy. Until then, the table is an audited snapshot rather than a permanent approval of future resolver output.

### Generated site bundles and external assets

The untracked `web/docs/site/` build output was inspected in addition to the Python package metadata. The final notices page must cover the emitted artifact, not only the build environment.

| Delivered or referenced asset | Source | License | Treatment |
|---|---|---|---|
| Material for MkDocs compiled CSS and JavaScript | Material for MkDocs 9.7.7 | MIT | Delivered to browsers; reproduce the Material MIT notice |
| Lunr 2.3.9 search engine | MkDocs search template / `lunrjs.com` | MIT | Delivered to browsers; retain license header and notices |
| Lunr Languages and Snowball-derived stemmers | `github.com/MihaiValentin/lunr-languages`; copied by Material 9.7.7 | MPL-1.1 | Material copies all language bundles into the output even for this English site. This is a reciprocal source-availability obligation, so it is not cleared for the intended notices-only site distribution; remove the unused bundles or satisfy MPL-1.1 in full |
| TinySegmenter | Taku Kudo TinySegmenter 0.1 | BSD-3-Clause-style “new BSD” | Delivered even when the site language is English; reproduce its notice |
| Wordcut browser bundle and its bundled dependencies | `github.com/veer66/wordcut`; copied by Material 9.7.7 | LGPL-3.0 for Wordcut plus retained MIT notices for bundled dependencies | Delivered even when unused by the English index. This introduces reciprocal source/relinking obligations; remove the unused bundle or satisfy LGPL-3.0 and every embedded notice in full |
| GitHub brand icon | Font Awesome Free 7.1.0 | CC BY 4.0 for SVG icons | The generated HTML retains Font Awesome's copyright/license comment; public notices must provide attribution and license link |
| Light/dark toggle icons | Pictogrammers Material Design Icons | Apache-2.0 | Generated inline SVG; preserve attribution/license in public notices |
| Inter and JetBrains Mono | Google Fonts links emitted by Material | SIL Open Font License 1.1 | No font files are hosted in this repository, but browsers fetch them from Google; document the external dependency or self-host with OFL notices |

## First-party and generated visual assets

| Category | Paths | Source | License treatment |
|---|---|---|---|
| Hero bolt logos and favicons | `internal/serve/shell/static/favicon.svg`, `web/docs/src/assets/*.svg`, `web/landing/site/favicon.svg` | Hero-authored SVG paths | Included in proposed Hero grant |
| Landing social card | `web/landing/site/og-image.svg` | Hero-authored SVG/text | Included; its current “MIT” wording is a known public-copy correction, not a third-party asset issue |
| Shell JavaScript and CSS | `internal/serve/shell/static/` | Hero-authored source embedded in the binary | Included in proposed Hero grant |
| Generated MkDocs site | `web/docs/site/` | Not tracked; generated from Hero docs plus Material for MkDocs | Deploy artifact only; regenerate from bounded dependencies and publish notices |
| Fonts | No font files tracked; docs HTML references Google Fonts for Inter and JetBrains Mono | External Google Fonts requests; both families are OFL-licensed | Record the external dependency; if later self-hosted, include OFL texts |
| Screenshots and raster marketing media | None tracked in public site source | N/A | No public redistribution row required at this audit point |
| Examples and fixtures | tracked source under repository test/example paths | Hero-authored unless a file-local notice states otherwise | Included; no third-party vendored example found |
| Vendored package directories | None | `vendor/`, `node_modules/`, and third-party source trees are absent | No vendored-license row required |
