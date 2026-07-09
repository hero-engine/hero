# Hero Sales — Spec Types

Deal spec schema definitions for the Hero Sales domain pack.

| Spec Type | File | Description |
|---|---|---|
| `deal` | [deal.md](deal.md) | An active sales opportunity — frontmatter fields, status values, CRM integration fields |

### Deal status values

| Status | Meaning |
|---|---|
| `prospect` | Lead identified, outreach not yet started |
| `qualifying` | In active discovery, MEDDPICC in progress |
| `demo` | Technical and business evaluation underway |
| `proposal` | Proposal submitted |
| `negotiation` | Commercial terms being negotiated |
| `won` | Closed Won |
| `lost` | Closed Lost |

Deal statuses are registry-backed lifecycle states: `prospect → qualifying → demo → proposal → negotiation → won|lost`.

Deal specs live at `.hero/planning/deals/<slug>/spec.md`.

See [deal.md](deal.md) for the full schema with all frontmatter fields.
See [../AGENTS.md](../AGENTS.md) for how agents work with deal specs.
