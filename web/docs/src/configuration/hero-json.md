# `hero.json` configuration

Shared non-secret settings live in `.hero/hero.json`. Personal credentials and
overrides belong in `.hero/hero.local.json`, which must not be committed.
`hero init` creates a starter file; `hero connect` configures integrations.

## Decoder-backed full example

This example is mirrored by `internal/config/testdata/public-hero.json` and is
loaded through the production decoder by
`TestPublicHeroConfigFixtureLoadsThroughProductionDecoder`. Update that fixture
and test before changing this block.

```json
{
  "folder": ".hero",
  "team": {
    "require_review": true,
    "stale_days": 14,
    "auto_context": true,
    "nudge_level": "assertive"
  },
  "integrations": {
    "default": "jira-delivery",
    "roles": {
      "delivery": "jira-delivery"
    },
    "connections": {
      "jira-delivery": {
        "provider": "jira",
        "settings": {
          "project": "PROJ",
          "base_url": "https://example.atlassian.net",
          "user_email": "developer@example.com"
        },
        "auth": {
          "token_env": "JIRA_API_TOKEN"
        }
      }
    }
  },
  "import": {
    "default_type": "bug",
    "limit": 25,
    "base_filter": {
      "jql": "status != Done AND assignee = currentUser()"
    },
    "auto_refresh": true,
    "refresh_interval": "30m"
  },
  "sync": {
    "target": "none",
    "auto": false
  },
  "conventions": {
    "enforce": true,
    "scope_default": "*"
  },
  "knowledge": {
    "auto_capture": true,
    "explainer_synthesis": "review"
  },
  "models": {
    "roles": {
      "design": "claude-opus-4",
      "execution": "claude-sonnet-4-5",
      "review": "o3"
    },
    "default_model": "claude-sonnet-4-5"
  },
  "hooks": {
    "branch_patterns": [
      "feat/{{slug}}",
      "feature/{{slug}}",
      "fix/{{slug}}",
      "{{slug}}"
    ],
    "slug_transform": "kebab",
    "inject_commit_prefix": true
  },
  "tracking": {
    "stale_claim_days": 2,
    "default_agent": "developer"
  },
  "sessions": {
    "retention_days": 30,
    "gitignore": true
  },
  "testing": {
    "framework": "playwright",
    "mode": "autonomous",
    "test_dir": "e2e",
    "runner_command": "npx playwright test",
    "base_url": "http://localhost:3000"
  },
  "demos": {
    "mode": "manual",
    "framework": "playwright",
    "output_dir": ".hero/demos",
    "video_size": {
      "width": 1280,
      "height": 720
    },
    "on_deliver": false
  },
  "embeddings": {
    "enabled": true,
    "scope": [
      "spec",
      "knowledge",
      "convention",
      "event",
      "code"
    ],
    "model": "hero-embed-v1"
  },
  "serve": {
    "tool_filter": {
      "allow": [
        "hero_context",
        "hero_search",
        "hero_status"
      ],
      "deny": [
        "hero_demo_record"
      ],
      "profiles": {
        "minimal": [
          "hero_context",
          "hero_status"
        ]
      }
    }
  },
  "specs": {
    "layout": "single"
  },
  "delivery": {
    "default_mode": "supervised",
    "autopilot_halt_on": [
      "drift",
      "test",
      "boundary",
      "lint"
    ]
  },
  "verify": {
    "run_tests": true,
    "test_command": "go test ./..."
  }
}
```

## Important shapes

- `import.base_filter` is an object containing a provider-native filter such as
  `jql`; it is not a string-valued `import.filter`.
- `hooks.branch_patterns` is a list of templates using `{{slug}}`.
- `models.roles` uses `design`, `execution`, and `review`.
- `serve.tool_filter.profiles` maps a profile name directly to a list of allowed
  tool names.
- Integration connections use stable IDs under
  `integrations.connections`. Shared provider settings live here; personal
  `auth.token` belongs at the same path in `.hero/hero.local.json`.

## Integration and action boundary

Supported delivery providers are GitHub, Jira, Linear, and GitLab. Configure
them with `hero connect` or follow [Tracker Setup](tracker-setup.md). Connection
configuration authorizes access to a provider; it does not grant blanket
consent for mutations. A write still requires a uniquely resolved target and
operation-specific user intent.

Never place a token directly in committed configuration or command arguments.
Automation should pass credentials on standard input or reference an environment
variable through `token_env`.

## Validation

```bash
go test ./internal/config -run TestPublicHeroConfigFixtureLoadsThroughProductionDecoder -count=1
hero check
```

`hero check` validates the active workspace. The Go test is the release guard
that keeps the published full example synchronized with the production decoder.
