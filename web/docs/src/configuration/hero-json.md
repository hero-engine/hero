# hero.json Configuration

Hero is configured through a `hero.json` file at the root of your project (or inside `.hero/`). This file controls tracker integration, team workflows, import behavior, conventions, and more.

!!! tip "Generate a starter config"
    Run `hero init` to generate a `hero.json` with sensible defaults, or `hero sync connect` for interactive tracker setup.

---

## Full Example

```json
{
  "folder": ".hero",

  "team": {
    "require_review": true,
    "stale_days": 14,
    "auto_context": true,
    "nudge_level": "moderate"
  },

  "tracker": {
    "type": "jira",
    "project": "PROJ",
    "token_env": "JIRA_API_TOKEN",
    "base_url": "https://myorg.atlassian.net",
    "post_on_design": true,
    "post_on_deliver": true
  },

  "import": {
    "default_type": "bug",
    "limit": 25,
    "filter": "status != Done AND assignee = currentUser()",
    "auto_refresh": true,
    "refresh_interval": "30m"
  },

  "jira": {
    "custom_fields": {
      "epic_link": "customfield_10014",
      "sprint": "customfield_10020",
      "story_points": "customfield_10028",
      "acceptance_criteria": "customfield_10035"
    },
    "transitions": {
      "in_progress": "In Progress",
      "in_review": "In Review",
      "done": "Done"
    }
  },

  "sync": {
    "target": "github",
    "auto": false
  },

  "conventions": {
    "enforce": true,
    "scope_default": "project"
  },

  "knowledge": {
    "auto_capture": true
  },

  "models": {
    "roles": {
      "plan": "claude-sonnet-4-20250514",
      "build": "claude-sonnet-4-20250514",
      "review": "claude-sonnet-4-20250514"
    }
  },

  "hooks": {
    "branch_patterns": {
      "feature": "feature/{slug}",
      "bug": "fix/{slug}",
      "chore": "chore/{slug}"
    },
    "slug_transform": "kebab-case",
    "inject_commit_prefix": true
  },

  "tracking": {},

  "sessions": {},

  "pulse": {},

  "testing": {
    "framework": "jest",
    "mode": "unit",
    "test_dir": "tests/",
    "runner_command": "npm test",
    "base_url": "http://localhost:3000"
  },

  "demos": {},

  "embeddings": {
    "enabled": true,
    "scope": ["spec", "knowledge", "convention", "event", "code"],
    "model": "hero-embed-v1"
  },

  "serve": {
    "tool_filter": {
      "allow": ["hero_context", "hero_search", "hero_status"],
      "deny": ["hero_demo_record"],
      "profiles": {
        "minimal": {
          "allow": ["hero_context", "hero_status"]
        },
        "full": {
          "deny": []
        }
      }
    }
  }
}
```

---

## Section Reference

### `folder`

The directory where Hero stores specs, knowledge, and reports. Defaults to `".hero"`.

```json
"folder": ".hero"
```

---

### `team`

Team workflow settings that control collaboration behavior.

| Key | Type | Default | Description |
|---|---|---|---|
| `require_review` | `bool` | `false` | Require review approval before a spec can be delivered |
| `stale_days` | `int` | `14` | Number of days before an in-progress spec is flagged as stale |
| `auto_context` | `bool` | `true` | Automatically load project context at session start |
| `nudge_level` | `string` | `"moderate"` | How aggressively Hero nudges on process gaps: `"off"`, `"gentle"`, `"moderate"`, `"strict"` |

---

### `tracker`

External issue tracker connection. See [Tracker Setup](tracker-setup.md) for detailed guides.

| Key | Type | Description |
|---|---|---|
| `type` | `string` | Tracker type: `"github"`, `"jira"`, or `"linear"` |
| `project` | `string` | Project key or repository (e.g. `"PROJ"`, `"owner/repo"`) |
| `token_env` | `string` | Environment variable name containing the API token |
| `base_url` | `string` | Base URL for self-hosted instances (Jira Server, GitHub Enterprise) |
| `post_on_design` | `bool` | Post a comment to the tracker issue when a spec is designed |
| `post_on_deliver` | `bool` | Post a comment when delivery is complete |

---

### `import`

Controls how `/import` and `hero import` pull issues from the tracker.

| Key | Type | Default | Description |
|---|---|---|---|
| `default_type` | `string` | `"bug"` | Default spec type for imported issues: `"bug"`, `"feature"`, `"chore"` |
| `limit` | `int` | `25` | Maximum number of issues to import per run |
| `filter` | `string` | — | Tracker-native filter query (JQL for Jira, search query for GitHub) |
| `auto_refresh` | `bool` | `false` | Automatically refresh imported specs from the tracker |
| `refresh_interval` | `string` | `"30m"` | How often to auto-refresh (e.g. `"15m"`, `"1h"`) |

---

### `jira`

Jira-specific configuration for custom fields and workflow transitions.

#### `custom_fields`

Map logical field names to Jira custom field IDs:

| Key | Description |
|---|---|
| `epic_link` | Epic link field |
| `sprint` | Sprint field |
| `story_points` | Story points / estimation field |
| `acceptance_criteria` | Acceptance criteria field |

#### `transitions`

Map Hero status changes to Jira transition names:

| Key | Description |
|---|---|
| `in_progress` | Transition name when work starts |
| `in_review` | Transition name when review begins |
| `done` | Transition name when work completes |

!!! note
    Use `push_status_transitions` in your Jira config to enable automatic status syncing. Hero will transition issues as specs move through the workflow.

---

### `sync`

Spec synchronization settings.

| Key | Type | Description |
|---|---|---|
| `target` | `string` | Sync target: `"github"`, `"jira"`, `"linear"` |
| `auto` | `bool` | Automatically sync spec status changes to the tracker |

---

### `conventions`

Convention enforcement settings.

| Key | Type | Default | Description |
|---|---|---|---|
| `enforce` | `bool` | `false` | Enforce conventions during `/check` and `/deliver` |
| `scope_default` | `string` | `"project"` | Default scope for new conventions: `"project"` or `"team"` |

---

### `knowledge`

Knowledge base behavior.

| Key | Type | Default | Description |
|---|---|---|---|
| `auto_capture` | `bool` | `true` | Automatically capture learnings at the end of major workflows |

---

### `models.roles`

Assign models to Hero's operational roles.

| Role | Description |
|---|---|
| `plan` | Model used for planning, design, and analysis (read-only agents) |
| `build` | Model used for implementation (read-write agents) |
| `review` | Model used for review agents |

```json
"models": {
  "roles": {
    "plan": "claude-sonnet-4-20250514",
    "build": "claude-sonnet-4-20250514",
    "review": "claude-sonnet-4-20250514"
  }
}
```

---

### `hooks`

Git workflow integration.

| Key | Type | Description |
|---|---|---|
| `branch_patterns` | `object` | Branch naming templates per spec type. Use `{slug}` as placeholder |
| `slug_transform` | `string` | How to transform spec titles to slugs: `"kebab-case"`, `"snake_case"` |
| `inject_commit_prefix` | `bool` | Prefix commit messages with the tracker ID (e.g. `PROJ-142: ...`) |

---

### `testing`

Test runner configuration used by delivery and QA agents.

| Key | Type | Description |
|---|---|---|
| `framework` | `string` | Test framework: `"jest"`, `"pytest"`, `"go"`, `"vitest"`, etc. |
| `mode` | `string` | Default test mode: `"unit"`, `"integration"`, `"e2e"` |
| `test_dir` | `string` | Directory containing tests |
| `runner_command` | `string` | Command to run tests |
| `base_url` | `string` | Base URL for integration/e2e tests |

---

### `embeddings`

Semantic embedding engine for hybrid search. The embedding model ships
inside the binary — no download or external tooling required.

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | `bool` | `true` | Enable semantic embeddings. Set to `false` to use BM25-only search |
| `scope` | `string[]` | `["spec", "knowledge", "convention", "event", "code"]` | Which corpora to embed. Remove entries to reduce index size |
| `model` | `string` | `"hero-embed-v1"` | Model name. Override to use a custom model from `~/.hero/models/embeddings/<name>/` |

The embedding index lives inside `index.db` (the `vec_chunks` table) and
is rebuilt incrementally on every `hero scan`. Use `hero embeddings status`
to inspect the index and `hero embeddings rebuild` to force a full re-embed.

---

### `serve`

Configuration for `hero serve` and `hero mcp`. See [MCP Setup](mcp-setup.md).

#### `tool_filter`

Controls which MCP tools are exposed.

| Key | Type | Description |
|---|---|---|
| `allow` | `string[]` | Allowlist of tool names. If set, only these tools are exposed |
| `deny` | `string[]` | Denylist of tool names. These tools are hidden |
| `profiles` | `object` | Named filter profiles for different use cases |
