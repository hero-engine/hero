# Tracker Setup

Hero integrates with **GitHub Issues**, **Jira**, and **Linear** to import issues, sync status, and post updates. This page covers setup for each tracker.

!!! tip "Interactive setup"
    Run `hero sync connect` for a guided setup that writes your `hero.json` config and validates the connection.

---

## Quick Start

=== "GitHub Issues"

    ```json title="hero.json"
    {
      "tracker": {
        "type": "github",
        "project": "owner/repo",
        "token_env": "GITHUB_TOKEN"
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
      "tracker": {
        "type": "jira",
        "project": "PROJ",
        "token_env": "JIRA_API_TOKEN",
        "base_url": "https://myorg.atlassian.net"
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
      "tracker": {
        "type": "linear",
        "project": "TEAM-KEY",
        "token_env": "LINEAR_API_KEY"
      }
    }
    ```

    ```bash
    export LINEAR_API_KEY="lin_api_xxxxxxxxxxxxxxxxxxxx"
    ```

    Generate a personal API key at [linear.app/settings/api](https://linear.app/settings/api).

---

## Authentication

Tokens are always read from environment variables — never stored in `hero.json`. The `token_env` field specifies which env var to read.

| Tracker | Required env vars |
|---|---|
| GitHub | `GITHUB_TOKEN` |
| Jira Cloud | `JIRA_API_TOKEN`, `JIRA_USER_EMAIL` |
| Jira Server | `JIRA_API_TOKEN` |
| Linear | `LINEAR_API_KEY` |

!!! warning "Keep tokens out of version control"
    Set tokens in your shell profile (`~/.zshrc`, `~/.bashrc`), a `.env` file excluded from git, or a secrets manager.

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
<!-- drift-test:ignore (follow-up: --refresh flag moved to `hero sync import --refresh`) -->
hero import --refresh
```

With `auto_refresh: true`, Hero refreshes specs in the background at the configured interval.

---

## Verify Connection

```bash
hero status
```

This displays the configured tracker, connection status, and a count of imported specs.
