package tracker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// WriteSprintGraph ingests Jira/Linear sprint items into the unified
// knowledge graph. Each tracker issue becomes an Issue node (globally
// keyed by tracker key — same as commit-message-extracted Issue stubs,
// so they merge automatically). The sprint becomes a Sprint node;
// assignees become Person nodes (globally keyed by email or login).
//
// Edges emitted:
//   Issue ─belongs_to→ Sprint     (issue is in sprint)
//   Issue ─belongs_to→ Issue      (subtask → epic)
//   Issue ─assigned_to→ Person
//   Issue ─blocks/depends_on→ Issue   (from LinkedItems)
//
// Idempotent: re-running on the same sprint produces no history rows
// when nothing changed. Issue stubs created earlier from commit
// messages get fleshed out without losing their edges.
//
// repoKey stamps the partition column on local (intra-repo) edges.
// Issue/Sprint/Person nodes themselves stay globally-scoped (no Repo
// stamp) since trackers cross repo boundaries.
func WriteSprintGraph(items []SprintItem, info *SprintInfo, repoKey string, store *graph.Store) (*GraphSummary, error) {
	if store == nil {
		return nil, fmt.Errorf("tracker: WriteSprintGraph requires non-nil Store")
	}
	source := map[string]any{"kind": "tracker"}
	summary := &GraphSummary{}

	var sprintID int64
	if info != nil && info.ID != "" {
		props := map[string]any{
			"id":    info.ID,
			"name":  info.Name,
			"state": info.State,
		}
		if info.Goal != "" {
			props["goal"] = info.Goal
		}
		if info.Start != "" {
			props["start"] = info.Start
		}
		if info.End != "" {
			props["end"] = info.End
		}
		id, err := store.UpsertNode(&graph.Node{
			Type:        "Sprint",
			Key:         info.ID,
			Props:       props,
			ContentHash: hashFields("sprint", info.ID, info.Name, info.State, info.Start, info.End),
			Source:      source,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert Sprint: %w", err)
		}
		sprintID = id
		summary.Sprints++
	}

	// First pass: create Issue + Person nodes.
	issueIDs := make(map[string]int64, len(items))
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		issueProps := map[string]any{
			"key":      item.ID,
			"title":    item.Title,
			"type":     item.Type,
			"status":   item.Status,
			"tracker":  trackerForKey(item.ID),
		}
		if item.Priority != "" {
			issueProps["priority"] = item.Priority
		}
		if item.URL != "" {
			issueProps["url"] = item.URL
		}
		if item.SprintName != "" {
			issueProps["sprint"] = item.SprintName
		}
		if item.EpicID != "" {
			issueProps["epic"] = item.EpicID
		}
		if item.StoryPoints > 0 {
			issueProps["story_points"] = item.StoryPoints
		}
		if len(item.Labels) > 0 {
			issueProps["labels"] = item.Labels
		}
		if item.AcceptanceCriteria != "" {
			issueProps["acceptance_criteria"] = item.AcceptanceCriteria
		}

		issueID, err := store.UpsertNode(&graph.Node{
			Type:        "Issue",
			Key:         item.ID,
			Props:       issueProps,
			ContentHash: hashIssue(item),
			Source:      source,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert Issue %s: %w", item.ID, err)
		}
		issueIDs[item.ID] = issueID
		summary.Issues++

		// Sprint membership.
		if sprintID != 0 {
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: issueID, ToID: sprintID, Type: "belongs_to",
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert Issue→Sprint: %w", err)
			}
			summary.Edges++
		}

		// Assignee → Person (globally identified).
		if item.Assignee != "" {
			personID, err := store.UpsertNode(&graph.Node{
				Type:        "Person",
				Key:         strings.ToLower(item.Assignee),
				Props:       map[string]any{"display": item.Assignee},
				ContentHash: hashFields("person", strings.ToLower(item.Assignee)),
				Source:      source,
			})
			if err != nil {
				return nil, fmt.Errorf("upsert Person: %w", err)
			}
			summary.Persons++
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: issueID, ToID: personID, Type: "assigned_to",
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert assigned_to edge: %w", err)
			}
			summary.Edges++
		}
	}

	// Second pass: parent epic + linked-item edges (need both endpoints
	// to exist as graph nodes).
	for _, item := range items {
		fromID, ok := issueIDs[item.ID]
		if !ok {
			continue
		}
		// Subtask → epic (parent).
		if item.EpicID != "" {
			toID, ok := issueIDs[item.EpicID]
			if !ok {
				// Epic not in this batch — create a stub Issue node so
				// the edge has a target. Phase 8 federation will
				// reconcile when epic is loaded separately.
				epicID, err := store.UpsertNode(&graph.Node{
					Type: "Issue", Key: item.EpicID,
					Props: map[string]any{
						"key":     item.EpicID,
						"title":   item.EpicTitle,
						"type":    "epic",
						"tracker": trackerForKey(item.EpicID),
					},
					ContentHash: hashFields("issue-stub", item.EpicID),
					Source:      map[string]any{"kind": "tracker-epic-stub"},
				})
				if err != nil {
					return nil, fmt.Errorf("upsert Epic stub: %w", err)
				}
				toID = epicID
				summary.Issues++
			}
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: fromID, ToID: toID, Type: "belongs_to",
				Props:  map[string]any{"relation": "epic"},
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert subtask→epic edge: %w", err)
			}
			summary.Edges++
		}

		// Linked items (blocks / is-blocked-by / etc.).
		for _, link := range item.LinkedIDs {
			toID, ok := issueIDs[link.ID]
			if !ok {
				continue // skip cross-batch links — they'll resolve next ingest
			}
			edgeType := edgeTypeForLink(link.LinkType)
			if edgeType == "" {
				continue
			}
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: fromID, ToID: toID, Type: edgeType,
				Props:  map[string]any{"link_type": link.LinkType},
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert linked-item edge: %w", err)
			}
			summary.Edges++
		}
	}

	return summary, nil
}

// GraphSummary reports counts for diagnostics.
type GraphSummary struct {
	Sprints int
	Issues  int
	Persons int
	Edges   int
}

// WriteIssuesGraph upserts an Issue node for each issue in the slice
// (and a Person node + assigned_to edge for each non-empty assignee).
// Unlike WriteSprintGraph this does no sprint/epic/linked-item edges —
// it's the minimal best-effort path for ingesting open issues during
// `hero scan`.
//
// repoKey stamps the partition column on the assigned_to edges. Issue
// and Person nodes themselves stay globally-scoped (no Repo stamp)
// since trackers cross repo boundaries — same contract as
// WriteSprintGraph.
func WriteIssuesGraph(issues []Issue, repoKey string, store *graph.Store) (*GraphSummary, error) {
	if store == nil {
		return nil, fmt.Errorf("tracker: WriteIssuesGraph requires non-nil Store")
	}
	source := map[string]any{"kind": "tracker"}
	summary := &GraphSummary{}

	for _, item := range issues {
		if item.ID == "" {
			continue
		}
		issueProps := map[string]any{
			"key":     item.ID,
			"title":   item.Title,
			"type":    item.IssueType,
			"status":  item.Status,
			"tracker": trackerForKey(item.ID),
		}
		if item.Priority != "" {
			issueProps["priority"] = item.Priority
		}
		if item.URL != "" {
			issueProps["url"] = item.URL
		}
		if item.SprintName != "" {
			issueProps["sprint"] = item.SprintName
		}
		if len(item.Labels) > 0 {
			issueProps["labels"] = item.Labels
		}

		issueID, err := store.UpsertNode(&graph.Node{
			Type:        "Issue",
			Key:         item.ID,
			Props:       issueProps,
			ContentHash: hashIssueRecord(item),
			Source:      source,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert Issue %s: %w", item.ID, err)
		}
		summary.Issues++

		if item.Assignee != "" {
			personID, err := store.UpsertNode(&graph.Node{
				Type:        "Person",
				Key:         strings.ToLower(item.Assignee),
				Props:       map[string]any{"display": item.Assignee},
				ContentHash: hashFields("person", strings.ToLower(item.Assignee)),
				Source:      source,
			})
			if err != nil {
				return nil, fmt.Errorf("upsert Person: %w", err)
			}
			summary.Persons++
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: issueID, ToID: personID, Type: "assigned_to",
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert assigned_to edge: %w", err)
			}
			summary.Edges++
		}
	}
	return summary, nil
}

func hashIssueRecord(i Issue) string {
	return hashFields(
		"issue-record", i.ID, i.Title, i.Status, i.Priority,
		i.Assignee, i.SprintName, i.IssueType,
		strings.Join(i.Labels, ","),
	)
}

// edgeTypeForLink maps a Jira issue-link type to a graph edge type.
// Unknown link types return "" — caller skips them.
func edgeTypeForLink(linkType string) string {
	switch strings.ToLower(strings.ReplaceAll(linkType, "_", "-")) {
	case "blocks":
		return "blocks"
	case "is-blocked-by", "blocked-by":
		return "depends_on"
	case "relates-to", "relates":
		return "related_to"
	case "duplicates":
		return "supersedes"
	case "clones":
		return "related_to"
	default:
		return ""
	}
}

// trackerForKey infers the tracker (jira/github) from an issue key
// shape. Jira keys look like PROJ-123; GitHub keys carry a GH# prefix
// (set by the commit-ref parser in internal/gitutil).
func trackerForKey(key string) string {
	if strings.HasPrefix(key, "GH#") {
		return "github"
	}
	if strings.Contains(key, "-") {
		return "jira"
	}
	return "unknown"
}

func hashIssue(item SprintItem) string {
	return hashFields(
		"issue", item.ID, item.Title, item.Status, item.Priority,
		item.Assignee, item.SprintName, item.EpicID,
		strings.Join(item.Labels, ","),
		fmt.Sprintf("%.2f", item.StoryPoints),
	)
}

func hashFields(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
