---
title: Config/Env Map — Environment Variable and Configuration Index
slug: config-env-map
type: feature
status: completed
priority: low
tags: [code-intelligence, codescan, config, environment]
created: 2026-04-18
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Detect and index all configuration values and environment variables read by the codebase, mapping each to where it's consumed. Helps diagnose environment-class bugs ("it works locally but not in staging").

## Problem

Config-related bugs are common and hard to trace. An engineer sees an error, doesn't realize it's caused by a missing or wrong env var, and spends time debugging application logic. Knowing which env vars a file or module depends on shortens this investigation.

## Design

### Detection

Add a parser pass in `internal/codescan/` that detects config reads:

- **Go**: `os.Getenv("FOO")`, `os.LookupEnv("FOO")`, viper/envconfig struct tags
- **JS/TS**: `process.env.FOO`, `process.env["FOO"]`, dotenv usage
- **Python**: `os.environ["FOO"]`, `os.getenv("FOO")`, pydantic `Settings`
- **Ruby**: `ENV["FOO"]`, `ENV.fetch("FOO")`
- **Config files**: `.env`, `.env.example`, `docker-compose.yml` environment blocks

### Output

```go
type ConfigVar struct {
    Name     string // DATABASE_URL
    Source   string // env, config-file, struct-tag
    File     string // internal/db/connect.go
    Line     int
    Default  string // optional, if detectable
    Required bool   // true if no default/fallback
}
```

### Integration

Surface config dependencies in `hero_context` responses. When a file is queried, include any env vars it reads. Optionally provide a `hero config` CLI command that lists all detected config vars with their sources.

## Boundaries

- Static detection only — no runtime tracing.
- Best-effort extraction; won't catch dynamically constructed env var names (e.g., `os.Getenv(prefix + key)`).
- Config files are scanned for variable names but not parsed for values (don't leak secrets).
- V1 focuses on env vars. Structured config files (YAML, TOML, JSON) with schema detection are future work.
