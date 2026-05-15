# Tracker Integration

Tracker commands live under `hero sync`. Hero supports GitHub Issues,
Jira, and Linear.

## Connect

```bash
hero sync connect github
hero sync connect jira
hero sync connect linear
```

Configuration is stored in `.hero/hero.json` or the project config path
created by `hero init`.

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

## Sprint Loading

```bash
hero sprint load
hero sprint load --tracker jira --board PROJ --sprint 42
hero sprint load --tracker linear --iteration "2026-05-01"
```

Sprint loading creates or updates local spec scaffolds and records the
plan in the knowledge base.
