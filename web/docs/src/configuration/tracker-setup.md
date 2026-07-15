# Tracker Setup

Hero integrates with **GitHub Issues**, **Jira**, **Linear**, **GitLab**, and **Confluence** through stable integration IDs.

!!! tip "Interactive setup"
    Run `hero sync connect jira` for guided setup, or use protected automation: `printf '%s' "$JIRA_TOKEN" | hero connect jira --integration-id jira-delivery --project PROJ --base-url https://example.atlassian.net --user-email you@example.com --token-stdin`.

---

## Quick Start

=== "GitHub Issues"

    ```json title="hero.json"
    {
      "integrations": {
        "default": "github-delivery",
        "roles": {"delivery": "github-delivery"},
        "connections": {"github-delivery": {"provider": "github", "settings": {"project": "owner/repo"}, "auth": {"token_env": "GITHUB_TOKEN"}}}
      }
    }
    ```

    ```bash
    export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"
    ```

    The token needs the `repo` scope for private repositories, or `public_repo` for public ones.

=== "Jira"

    ```json title="hero.json"
    {
      "integrations": {
        "default": "jira-delivery",
        "roles": {"delivery": "jira-delivery"},
        "connections": {"jira-delivery": {"provider": "jira", "settings": {"project": "PROJ", "base_url": "https://myorg.atlassian.net", "user_email": "you@company.com"}, "auth": {"token_env": "JIRA_API_TOKEN"}}}
      }
    }
    ```

    ```bash
    export JIRA_API_TOKEN="your-api-token"
    export JIRA_USER_EMAIL="you@company.com"
    ```

    For Jira Cloud, generate a token at [id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens). For Jira Server, use a personal access token.

=== "Linear"

    ```json title="hero.json"
    {
      "integrations": {
        "default": "linear-delivery",
        "roles": {"delivery": "linear-delivery"},
        "connections": {"linear-delivery": {"provider": "linear", "settings": {"project": "TEAM-KEY"}, "auth": {"token_env": "LINEAR_API_KEY"}}}
      }
    }
    ```

    ```bash
    export LINEAR_API_KEY="lin_api_xxxxxxxxxxxxxxxxxxxx"
    ```

    Generate a personal API key at [linear.app/settings/api](https://linear.app/settings/api).

---

## Authentication

Literal tokens are forbidden in committed `hero.json`. Put `auth.token` in the matching connection in gitignored `hero.local.json`, save it globally by stable ID, use `auth.token_env`, or pipe it through `hero connect ... --token-stdin`. Precedence is local token, stable-ID global credential, then `token_env`; status output never prints token-derived masks.

| Tracker | Required env vars |
|---|---|
| GitHub | `GITHUB_TOKEN` |
| Jira Cloud | `JIRA_API_TOKEN`, `JIRA_USER_EMAIL` |
| Jira Server | `JIRA_API_TOKEN` |
| Linear | `LINEAR_API_KEY` |

!!! warning "Keep tokens out of version control"
    Set tokens in your shell profile (`~/.zshrc`, `~/.bashrc`), a `.env` file excluded from git, or a secrets manager.

### Shared settings with personal credentials

Both files use the same schema. A teammate may commit the connection settings above, while `.hero/hero.local.json` supplies only:

```json
{"integrations":{"connections":{"jira-delivery":{"auth":{"token":"<personal token>"}}}}}
```

An integration can instead be entirely personal by defining its `default`, role, provider, settings, and auth only in `hero.local.json`. Multiple connections—even two Jira projects—use distinct stable IDs. Local objects merge recursively; scalars including `false`, `0`, and `""` replace shared values, and `null` deletes an inherited optional field or connection in the effective view. A dangling selector is rejected with its JSON path.

Inspect the redacted effective view with `hero connect --list` or `hero connect --list --json`. Override delivery selection for one sync with `hero sync --integration <id> …`.

---

## Import Configuration

Control how issues are imported with the `import` section:

```json title="hero.json"
{
  "import": {
    "default_type": "bug",
    "limit": 25,
    "filter": "status != Done AND assignee = currentUser()",
    "auto_refresh": true,
    "refresh_interval": "30m"
  }
}
```

| Key | Description |
|---|---|
| `default_type` | Spec type assigned to imported issues (`"bug"`, `"feature"`, `"chore"`) |
| `limit` | Max issues imported per run |
| `filter` | Tracker-native query — JQL for Jira, search syntax for GitHub, filter syntax for Linear |
| `auto_refresh` | Re-sync imported specs from the tracker periodically |
| `refresh_interval` | Refresh frequency (e.g. `"15m"`, `"1h"`) |

### Filter examples

=== "GitHub"

    ```json
    "filter": "is:open is:issue label:bug assignee:@me"
    ```

=== "Jira"

    ```json
    "filter": "project = PROJ AND status != Done AND assignee = currentUser() ORDER BY priority DESC"
    ```

=== "Linear"

    ```json
    "filter": "assignee:me state:started,unstarted"
    ```

---

## Jira Custom Fields

Jira's custom field IDs vary by instance. Map them in the `jira` section:

```json title="hero.json"
{
  "jira": {
    "custom_fields": {
      "epic_link": "customfield_10014",
      "sprint": "customfield_10020",
      "story_points": "customfield_10028",
      "acceptance_criteria": "customfield_10035"
    }
  }
}
```

!!! info "Finding custom field IDs"
    Use the Jira REST API: `GET /rest/api/3/field` returns all fields with their IDs and names.

---

## Status Sync (Jira)

Hero can automatically transition Jira issues as specs move through the workflow. Configure transition names in the `jira.transitions` section:

```json title="hero.json"
{
  "jira": {
    "transitions": {
      "in_progress": "In Progress",
      "in_review": "In Review",
      "done": "Done"
    }
  }
}
```

These map Hero spec statuses to Jira workflow transition names. Hero calls the Jira transition API when a spec status changes — for example, moving an issue to "In Progress" when `/deliver` starts.

!!! note
    Transition names must match your Jira workflow exactly (case-sensitive). Check your project's workflow configuration if transitions fail.

---

## Posting Updates

Hero can post comments to tracker issues at key workflow points:

```json title="hero.json"
{
  "tracker": {
    "post_on_design": true,
    "post_on_deliver": true
  }
}
```

| Event | Comment content |
|---|---|
| `post_on_design` | Links to the spec and summarizes the design |
| `post_on_deliver` | Summarizes what was delivered, files changed, and test results |

---

## Syncing Spec Status

Pull the latest status from the tracker for a specific spec:

```bash
hero sync pull <slug>
```

Or refresh all imported specs:

```bash
hero sync import --refresh
```

With `auto_refresh: true`, Hero refreshes specs in the background at the configured interval.

---

## Verify Connection

```bash
hero status
```

This displays the configured tracker, connection status, and a count of imported specs.
