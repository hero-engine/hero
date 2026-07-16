# Tracker Integration

Tracker commands live under `hero sync`. Hero supports **GitHub Issues,
Jira, Linear, and GitLab** as issue trackers. (Confluence is also a
provider, but for one-way *wiki publishing* via `hero publish`, not
`hero sync` — see [Tracker Setup → Confluence](../configuration/tracker-setup.md#confluence-wiki-publishing).)

`hero sync` is bidirectional — it pulls tracker state into specs and pushes
spec changes back out. See [Tracker Setup](../configuration/tracker-setup.md)
for the full `hero.json` connection schema and per-provider auth.

## Connect

```bash
hero sync connect github
hero sync connect jira
hero sync connect linear
hero sync connect gitlab
```

Configuration is stored in `.hero/hero.json` or the project config path
created by `hero init`. GitLab requires a `base_url` (use
`https://gitlab.com` for SaaS) and a token with the `api` scope.

## Import Issues

```bash
hero sync import
hero sync import --type bug
hero sync import --jql "priority = High"
hero sync import --dry-run
```

Imported specs are scaffolds in `.hero/planning/` and include
tracker-prefixed frontmatter such as `jira_status`, `jira_priority`, or
GitHub/Linear equivalents.

## Sync Specs

```bash
hero sync spec .hero/planning/features/auth-flow/spec.md
hero sync link .hero/planning/features/auth-flow/spec.md PROJ-1234
hero sync pull .hero/planning/features/auth-flow/spec.md
hero sync comment PROJ-1234 "Root cause found"
hero sync attach PROJ-1234 diagnosis.md
```

Always run `hero sync pull <spec>` before starting tracker-backed work
so closed or reassigned issues are not investigated again.

## Push & Status Transitions

`hero sync pull` brings tracker state *in*; these push Hero state back *out*:

```bash
hero sync push .hero/planning/features/auth-flow/spec.md  # field-level diff → tracker
hero sync jira                                            # bulk-push status transitions to Jira
```

`sync push` writes only the fields that changed (a field-level diff), and
`sync spec` creates the issue first if the spec has no `tracker_id` yet.
`sync jira` maps Hero spec statuses onto your Jira workflow transitions in
bulk (configure transition names under `jira.transitions` — see
[Tracker Setup](../configuration/tracker-setup.md#status-sync-jira)).

## Team Server & Cloud Sync

For teams running the Hero team server / Hero Cloud, `hero sync` also
exchanges state with the server:

```bash
hero sync cloud    # push specs to the Hero team server / Cloud
hero sync graph    # push / pull knowledge-graph deltas with Hero Cloud
```

These require a configured team server or `hero login` to Hero Cloud;
they are independent of your issue-tracker connection.

## Sprint Loading

```bash
hero sprint load
hero sprint load --tracker jira --board PROJ --sprint 42
hero sprint load --tracker linear --iteration "2026-05-01"
```

Sprint loading creates or updates local spec scaffolds and records the
plan in the knowledge base.
