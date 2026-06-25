// Package tracker provides an abstract interface for work tracker integrations
// (GitHub Issues, Jira, Linear). The hero binary bridges specs to tracker issues
// without replacing the tracker itself.
package tracker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// ErrSizeUpdateNotSupported is returned by adapters that don't write
// size values back to the tracker. Real adapters (jira/linear/github)
// never return this; it exists so future stub trackers can degrade
// gracefully and so callers (notably runSync) can distinguish "this
// adapter just doesn't write sizes" from real network/auth errors.
var ErrSizeUpdateNotSupported = errors.New("tracker: UpdateSize not supported by this adapter")

// Issue represents an issue in an external tracker.
type Issue struct {
	ID          string // tracker-native ID (e.g. "42" for GitHub, "PROJ-123" for Jira)
	Title       string
	Status      string            // tracker-native status string
	URL         string            // web URL to the issue
	Priority    string            // e.g. "high", "medium", "low" (Jira built-in priority field)
	Severity    string            // convenience: first severity-like value found in CustomFields
	Assignee    string            // display name or email of assignee
	Labels      []string          // label names
	IssueType   string            // tracker issue type name (e.g. "Story", "Bug", "Epic")
	EpicKey     string            // epic/parent key (Jira: epic link custom field)
	SprintName  string            // sprint or iteration name
	Reporter    string            // display name or email of reporter
	CreatedAt   string            // creation date (ISO 8601 or tracker-native)
	Description string            // issue description/body text
	CustomFields map[string]string // custom field values keyed by lowercase field name (e.g. "severity" → "critical")
}

// SearchQuery represents a structured query for filtering issues from a tracker.
// Trackers translate this into their native query format (JQL, GitHub search, Linear filters).
type SearchQuery struct {
	// RawQuery is a tracker-native query string (e.g. JQL for Jira).
	// When set, overrides all other fields.
	RawQuery string

	// FilterID is a saved filter/view ID in the tracker (e.g. Jira saved filter).
	// When set, overrides field-level filters but not RawQuery.
	FilterID string

	// IssueType filters by issue type (e.g. "Bug", "Story", "Task").
	IssueType string

	// Assignee filters by assignee. "unassigned" or "none" means no assignee.
	Assignee string

	// Labels filters by label names.
	Labels []string

	// Status filters by tracker-native status name (e.g. "New", "Open").
	Status string

	// Priority filters by priority name (e.g. "Critical", "High").
	Priority string

	// OrderBy controls sort order (tracker-native, e.g. "created DESC" for Jira).
	OrderBy string

	// Limit is the maximum number of results to return.
	Limit int
}

// Tracker is the interface all provider implementations satisfy.
type Tracker interface {
	// CreateIssue creates a new issue from a spec. Returns the tracker-native issue ID.
	CreateIssue(s *spec.Spec) (string, error)

	// UpdateStatus updates the status of an existing issue to reflect spec lifecycle changes.
	UpdateStatus(issueID string, status spec.Status) error

	// UpdateSize writes the mapped size value to the tracker for an
	// existing issue. The local tier is mapped via MapSize and emitted
	// in the adapter's native shape (Jira numeric custom field, Linear
	// `estimate`, GitHub `size/<tier>` label). Adapters without a real
	// implementation return ErrSizeUpdateNotSupported so callers can
	// degrade gracefully. Real network / auth / 4xx-5xx errors are
	// returned as-is, not swallowed.
	//
	// Callers (notably runSync) gate this behind PlanSizePush so the
	// non-destructive contract holds: never call UpdateSize on a
	// SizeSyncConflict.
	UpdateSize(issueID, localTier string) error

	// GetIssue retrieves current issue info from the tracker.
	GetIssue(issueID string) (*Issue, error)

	// UpdateFields writes a set of canonical field values to an existing
	// tracker issue. Keys are canonical hero-side field names (e.g.
	// "title", "description", "points", "priority", "labels"); the
	// adapter resolves them to provider-native fields and encodes the
	// Value tagged union into the provider's wire shape. Called only
	// with the diff (the fields that actually changed) — see the
	// field-level push path in internal/cli/sync_push.go.
	//
	// Errors are classified via FieldError so the CLI can map 401/403 →
	// exit 2 and apply the 429 retry policy. An empty patch is a no-op
	// (no network call); callers should not invoke UpdateFields with an
	// empty map.
	UpdateFields(issueID string, fields map[string]Value) error

	// GetFields fetches the current tracker-side values for the
	// canonical content fields, keyed by canonical hero-side name. Used
	// by the diff path (`hero sync push <slug>` with no --field flags)
	// to compute what differs from local. Only content-classified
	// fields the adapter knows how to read are returned; unknown or
	// org-state fields are omitted.
	GetFields(issueID string) (map[string]Value, error)

	// ListIssues fetches open issues from the tracker. Returns up to limit issues.
	// If label is non-empty, filters by that label.
	ListIssues(label string, limit int) ([]Issue, error)

	// Search fetches issues using a structured query. Supports raw queries,
	// saved filters, and field-level filters depending on the tracker.
	Search(query SearchQuery) ([]Issue, error)

	// AddComment posts a comment to an existing issue.
	AddComment(issueID, body string) error

	// AttachFile uploads a file as an attachment to an existing issue.
	// Not all trackers support attachments natively — implementations that
	// don't support attachments will post the file contents as a comment instead.
	AttachFile(issueID, filePath, fileName string) error

	// Name returns the tracker type name (e.g. "github", "jira", "linear").
	Name() string

	// SupportsHierarchy reports whether the tracker natively models
	// parent/child relationships (Jira epics, Linear projects, GitHub
	// sub-issues). The spec-sizing skill reads this flag to choose how
	// aggressively to nudge: trackers with strong hierarchy raise the
	// promotion threshold; flat trackers (and `tracker.type: "none"`)
	// nudge more aggressively. See domains/engineering/skills/spec-sizing.
	SupportsHierarchy() bool

	// MapSize translates a Hero local-tier name (one of the 6-tier
	// ladder: trivial / small / medium / large / x-large / giant) to
	// the corresponding tracker-side value (e.g. story points for Jira,
	// the `size/<tier>` label for GitHub). Returns ("", err) when no
	// size_mapping is configured (and no default applies) or the tier
	// is unknown — callers surface that as a conflict rather than
	// silently writing.
	MapSize(localTier string) (string, error)

	// ReverseMapSize translates a tracker-side value back to a local
	// tier. Inverse of MapSize: numeric values are matched against the
	// configured threshold bands; label-style values strip the
	// configured prefix and match against the ladder.
	ReverseMapSize(trackerValue string) (string, error)
}

// New creates a Tracker from the given config. Returns an error if the tracker
// type is unrecognized or if required configuration is missing.
func New(cfg *config.TrackerConfig) (Tracker, error) {
	if cfg == nil || cfg.Type == "" || cfg.Type == "none" {
		return nil, fmt.Errorf("no tracker configured (type is %q)", safeType(cfg))
	}

	token, err := cfg.ResolveToken()
	if err != nil {
		return nil, err
	}

	switch cfg.Type {
	case "github":
		g, err := newGitHub(cfg.Project, token, cfg.BaseURL)
		if err != nil {
			return nil, err
		}
		g.configuredSizeMapping = cfg.SizeMapping
		return g, nil
	case "jira":
		j, err := newJira(cfg.Project, token, cfg.UserEmail, cfg.BaseURL)
		if err != nil {
			return nil, err
		}
		j.configuredSizeMapping = cfg.SizeMapping
		return j, nil
	case "linear":
		l, err := newLinear(cfg.Project, token, cfg.BaseURL)
		if err != nil {
			return nil, err
		}
		l.configuredSizeMapping = cfg.SizeMapping
		return l, nil
	default:
		return nil, fmt.Errorf("unknown tracker type: %q", cfg.Type)
	}
}

// NewWithJiraConfig creates a Jira tracker with advanced JiraConfig settings.
// Falls back to New() for non-Jira tracker types.
// trackerKnowledgeDir is the path to .hero/knowledge/tracker/ for field cache persistence.
func NewWithJiraConfig(cfg *config.TrackerConfig, jiraCfg *config.JiraConfig, trackerKnowledgeDir string) (Tracker, error) {
	if cfg == nil || cfg.Type == "" || cfg.Type == "none" {
		return nil, fmt.Errorf("no tracker configured (type is %q)", safeType(cfg))
	}

	token, err := cfg.ResolveToken()
	if err != nil {
		return nil, err
	}

	if cfg.Type == "jira" {
		j, err := newJiraWithConfig(cfg.Project, token, cfg.UserEmail, cfg.BaseURL, jiraCfg, trackerKnowledgeDir)
		if err != nil {
			return nil, err
		}
		j.configuredSizeMapping = cfg.SizeMapping
		return j, nil
	}
	return New(cfg)
}

func safeType(cfg *config.TrackerConfig) string {
	if cfg == nil {
		return "nil"
	}
	return cfg.Type
}

// StatusLabel maps a spec lifecycle status to a human-readable label suitable
// for issue comments or status updates.
func StatusLabel(s spec.Status) string {
	switch s {
	case spec.StatusPlanning:
		return "Planning"
	case spec.StatusInReview:
		return "In Review"
	case spec.StatusDelivering:
		return "Delivering"
	case spec.StatusCompleted:
		return "Completed"
	case spec.StatusDraft:
		return "Draft"
	case spec.StatusActive:
		return "Active"
	case spec.StatusProposed:
		return "Proposed"
	case spec.StatusAccepted:
		return "Accepted"
	case spec.StatusSuperseded:
		return "Superseded"
	default:
		return string(s)
	}
}

// IssueBody builds a Markdown body for a tracker issue from a spec.
func IssueBody(s *spec.Spec) string {
	body := fmt.Sprintf("**Spec:** %s\n**Type:** %s\n**Status:** %s\n",
		s.Slug, string(s.Type), StatusLabel(s.Status))

	if goal, ok := s.Sections["goal"]; ok && goal != "" {
		body += fmt.Sprintf("\n## Goal\n\n%s\n", goal)
	}
	if approach, ok := s.Sections["approach"]; ok && approach != "" {
		body += fmt.Sprintf("\n## Approach\n\n%s\n", approach)
	}
	if context, ok := s.Sections["context"]; ok && context != "" {
		body += fmt.Sprintf("\n## Context\n\n%s\n", context)
	}

	body += "\n---\n*Managed by [Hero](https://github.com/hero-engine/hero)*\n"
	return body
}

// truncateDescription shortens a description to maxLen characters, breaking at a word boundary.
func truncateDescription(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Find last space before maxLen
	truncated := s[:maxLen]
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		truncated = truncated[:idx]
	}
	return truncated + "..."
}

// SearchQueryFromConfig builds a SearchQuery from an ImportFilter configuration.
func SearchQueryFromConfig(f *config.ImportFilter, limit int) SearchQuery {
	if f == nil {
		return SearchQuery{Limit: limit}
	}
	return SearchQuery{
		RawQuery:  f.JQL,
		FilterID:  f.FilterID,
		IssueType: f.IssueType,
		Assignee:  f.Assignee,
		Labels:    f.Labels,
		Status:    f.Status,
		Priority:  f.Priority,
		OrderBy:   f.OrderBy,
		Limit:     limit,
	}
}
