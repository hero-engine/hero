# Hero PM — Commands

Slash commands in this directory are PM workflows, loaded by the
domain command loader when the active domain in `hero.json` is `pm`.

See the [agent-pack design](../../../.hero/planning/features/hero-pm/agent-pack-design.md)
§E for the full 22-command list. The v1 P0 command set (12) shipped
here:

### New PM-specific (10)
| File | Routes to | Purpose |
|---|---|---|
| `triage.md` | `intake-triager` | Process inbound intake |
| `refine.md` | `pm-delivery-lead` → `story-writer` / `prd-author` / `epic-framer` (P1, ships v1.5) | Refine an artifact for delivery readiness |
| `roadmap.md` | `roadmap-curator` | Open / reconcile the roadmap |
| `prioritize.md` | `prioritization-strategist` | Rank a set of initiatives or specs |
| `prd.md` | `prd-author` | Draft or refine a PRD |
| `pitch.md` | `pitch-author` (P1) → falls back to `prd-author` in v1 with pitch template | Draft a Shape Up pitch |
| `handoff.md` | `handoff-coordinator` | Flip `owner: pm → engineering` on a refined spec — the cross-domain owner-flip handoff (brand interaction; no separate engineering spec is created) |
| `discover.md` | `discovery-researcher` | Continuous-discovery research kickoff |
| `metrics.md` | `metrics-analyst` (P1) → falls back to `pm-delivery-lead` in v1 | Define success metrics for a PRD |
| `release-notes.md` | `stakeholder-communicator` (P1) → falls back to `pm-delivery-lead` in v1 | Draft customer-facing release notes |

### Reused (cross-domain / core)
- `/why` — multi-hop trace across the spec hierarchy + bitemporal `owner_history` rows that show cross-domain ownership transitions
- `hero search` (CLI; no pack ships a `/search` command) — cross-domain search (results render `owner` so the PM/engineering boundary is visible)
- `/note` — note capture
- `/deliver` — engineering-pack command — runs on the engineering side after the owner flip, not in a pm install; runs against the same spec PM authored

P1 / P2 commands (`/capacity`, `/plan-cycle`, `/plan-sprint`,
`/plan-iteration`, `/standup`, `/interview`, `/scrub roadmap`,
`/scrub intake`, `/scrub specs`) ship in v1.5+ per §E and §H of the
pack design.

Each command file follows the engineering pack's shape: YAML
frontmatter (`description` only) then a markdown body that scopes the
workflow and names the delegated agent(s).
