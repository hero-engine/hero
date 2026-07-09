# Mock-tracker-server seed format — the shared contract

The mock server seeds an **in-memory SQLite DB** from **Sprout seed YAML**
(`github.com/bdwheeler/sprout/go`, the shipped engine), conforming to
sprout-polyglot's normative `seed-json-schema`. This is the **same format**
hero-code's `acme-checkout-dataset` authors and the real-account seeders
consume — author once, three consumers read it.

## Tables (the tracker-neutral schema)

The "tables" the seed YAML targets are the mock server's SQLite schema
(`internal/mocktracker/schema.go`). Each tracker handler projects these rows
into its own wire shape (GitHub `number`, Jira `key`, Linear `identifier`,
GitLab `iid`):

| Table       | Columns |
|-------------|---------|
| `issue`     | `global_id` (PK), `type`, `title`, `body`, `epic_id`, `milestone_id`, `iteration_id`, `status`, `assignee`, `weight`, `severity` |
| `epic`      | `global_id` (PK), `title`, `parent_id` |
| `milestone` | `global_id` (PK), `title`, `due` |
| `iteration` | `global_id` (PK), `name`, `start`, `end` |
| `label`     | `issue_id`, `name` |
| `app_user`  | `username` (PK), `email`, `display` |
| `id_alias`  | `global_id` (PK), `iid` — managed by the server (`/__admin/rotate-ids`), not seeded |

## FK references — IMPORTANT divergence to reconcile with hero-code

Sprout's nested-map FK directive (`epic: { code: EPIC-1 }`) resolves by
issuing `SELECT id FROM epic WHERE …` — it requires the referenced table to
have an **`id`** column and emits `<field>_id`. This schema keys every table
on **`global_id`**, not `id`, so the embedded fixture uses **scalar foreign
keys** instead:

```yaml
issue:
  - key: global_id
    data: { global_id: ACME-101, epic_id: EPIC-1, milestone_id: MILE-1,
            iteration_id: ITER-3, ... }   # scalar refs, NOT { epic: {…} }
```

hero-code's `acme-checkout-dataset` spec sketches the nested-map form with a
`code` key (`epic: { code: EPIC-1 }`). For that to apply against THIS schema,
one of two reconciliations is needed before wiring `--seed` at the full Acme
dataset:

1. **Add an `id` column** (alias of `global_id`) to each container table so
   sprout's FK lookup resolves, keeping the nested-map authoring style; or
2. **Author the full dataset with scalar `*_id` references** (as the embedded
   fixture does), matching this schema verbatim.

This is flagged as an open cross-repo contract item — see the
`mock-tracker-server` spec's Completion Ledger. The embedded tiny fixture
(option 2) unblocks hero's own tests today; the full dataset is not built yet.

## Embedded default vs `--seed <dir>`

- No flag → the embedded `fixtures/seed/` fixture (2 epics, 1 milestone, 1
  iteration, 6 issues, labels, 2 users) so `go test ./internal/mocktracker/…`
  is self-contained.
- `--seed <dir>` → a real directory of Sprout seed YAML (hero-code's full
  Acme dataset, once it lands and the FK reconciliation above is settled).

Re-applying is idempotent (sprout checksum-skip); a malformed seed surfaces
sprout's validation error verbatim. `/__admin/reset` re-seeds with
`sprout.WithForce()` for a deterministic mid-run reset.
