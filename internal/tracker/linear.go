package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

const defaultLinearAPI = "https://api.linear.app/graphql"

// linear implements the Tracker interface for Linear using its GraphQL API.
type linear struct {
	teamKey string
	token   string
	apiURL  string
	client  *http.Client
	// configuredSizeMapping is the workspace-configured size mapping
	// (hero.json: tracker.size_mapping). Nil → use the shipped default.
	// See internal/tracker/size_mapping.go.
	configuredSizeMapping *config.SizeMappingConfig
}

func newLinear(project, token, baseURL string) (*linear, error) {
	if project == "" {
		return nil, fmt.Errorf("linear team key is required")
	}
	if baseURL == "" {
		baseURL = defaultLinearAPI
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &linear{
		teamKey: project,
		token:   token,
		apiURL:  baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (l *linear) Name() string { return "linear" }

// CreateIssue creates a Linear issue from a spec. Returns the issue identifier.
func (l *linear) CreateIssue(s *spec.Spec) (string, error) {
	// Non-destructive size write on create: if the spec carries a
	// declared size and the mapping resolves it cleanly to a numeric
	// estimate, include it in the mutation. CreateIssue has nothing to
	// overwrite, so the planner isn't invoked — the overwrite-safety
	// check only matters on update.
	var sizeEstimate *float64
	if s.Size != "" {
		if v, err := l.MapSize(s.Size); err == nil && v != "" {
			if n, perr := strconv.ParseFloat(v, 64); perr == nil {
				sizeEstimate = &n
			}
		}
	}

	var query string
	variables := map[string]interface{}{
		"teamId":      l.teamKey,
		"title":       fmt.Sprintf("[%s] %s", s.Type, s.Title),
		"description": IssueBody(s),
	}
	if sizeEstimate != nil {
		query = `mutation CreateIssue($teamId: String!, $title: String!, $description: String!, $estimate: Float) {
			issueCreate(input: { teamId: $teamId, title: $title, description: $description, estimate: $estimate }) {
				success
				issue {
					id
					identifier
					url
				}
			}
		}`
		variables["estimate"] = *sizeEstimate
	} else {
		query = `mutation CreateIssue($teamId: String!, $title: String!, $description: String!) {
			issueCreate(input: { teamId: $teamId, title: $title, description: $description }) {
				success
				issue {
					id
					identifier
					url
				}
			}
		}`
	}

	result, err := l.graphql(query, variables)
	if err != nil {
		return "", fmt.Errorf("creating issue: %w", err)
	}

	data, ok := result["issueCreate"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response shape from Linear")
	}

	success, _ := data["success"].(bool)
	if !success {
		return "", fmt.Errorf("linear issueCreate returned success=false")
	}

	issue, ok := data["issue"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no issue in response")
	}

	identifier, _ := issue["identifier"].(string)
	if identifier == "" {
		// Fall back to id
		identifier, _ = issue["id"].(string)
	}

	return identifier, nil
}

// UpdateSize writes the mapped estimate to a Linear issue. Linear's
// `estimate` field is only present when the team has estimation
// enabled; if the mutation rejects with that case we log a warning
// and return nil (soft failure, per spec). Real network / auth /
// other GraphQL errors propagate.
func (l *linear) UpdateSize(issueID, localTier string) error {
	v, err := l.MapSize(localTier)
	if err != nil {
		return fmt.Errorf("mapping size %q: %w", localTier, err)
	}
	if v == "" {
		return fmt.Errorf("size %q maps to empty value", localTier)
	}
	estimate, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("parsing mapped size %q as number: %w", v, err)
	}

	internalID, err := l.resolveIssueID(issueID)
	if err != nil {
		return err
	}

	query := `mutation IssueUpdate($id: String!, $estimate: Float!) {
		issueUpdate(id: $id, input: { estimate: $estimate }) {
			success
		}
	}`
	variables := map[string]interface{}{
		"id":       internalID,
		"estimate": estimate,
	}

	result, err := l.graphql(query, variables)
	if err != nil {
		// Linear surfaces estimation-disabled as a GraphQL error
		// message. We don't have a stable error code, so substring
		// match on the message. Soft-fail in that case; everything
		// else propagates.
		if isLinearEstimationDisabled(err) {
			fmt.Fprintf(os.Stderr, "Warning: Linear team has estimation disabled; skipping size update for %s\n", issueID)
			return nil
		}
		return fmt.Errorf("updating size: %w", err)
	}

	data, ok := result["issueUpdate"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response shape from Linear")
	}
	if success, _ := data["success"].(bool); !success {
		return fmt.Errorf("linear issueUpdate returned success=false")
	}
	return nil
}

// isLinearEstimationDisabled reports whether a GraphQL error from
// Linear indicates the team has estimation turned off. Substring
// match against the surfaced message; updated if Linear ever ships a
// stable error code we can key on.
func isLinearEstimationDisabled(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "estimat") {
		return false
	}
	return strings.Contains(msg, "not enabled") ||
		strings.Contains(msg, "disabled") ||
		strings.Contains(msg, "not allowed")
}

// UpdateStatus adds a comment to the Linear issue with the new status.
func (l *linear) UpdateStatus(issueID string, status spec.Status) error {
	// First, find the issue's internal ID from its identifier
	internalID, err := l.resolveIssueID(issueID)
	if err != nil {
		return err
	}

	query := `mutation AddComment($issueId: String!, $body: String!) {
		commentCreate(input: { issueId: $issueId, body: $body }) {
			success
		}
	}`

	variables := map[string]interface{}{
		"issueId": internalID,
		"body":    fmt.Sprintf("**Hero status update:** %s", StatusLabel(status)),
	}

	result, err := l.graphql(query, variables)
	if err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	data, ok := result["commentCreate"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response shape from Linear")
	}
	success, _ := data["success"].(bool)
	if !success {
		return fmt.Errorf("linear commentCreate returned success=false")
	}

	return nil
}

// AddComment posts a comment to a Linear issue.
func (l *linear) AddComment(issueID, body string) error {
	internalID, err := l.resolveIssueID(issueID)
	if err != nil {
		return err
	}

	query := `mutation AddComment($issueId: String!, $body: String!) {
		commentCreate(input: { issueId: $issueId, body: $body }) {
			success
		}
	}`

	variables := map[string]interface{}{
		"issueId": internalID,
		"body":    body,
	}

	result, err := l.graphql(query, variables)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}

	data, ok := result["commentCreate"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response shape from Linear")
	}
	if success, _ := data["success"].(bool); !success {
		return fmt.Errorf("linear commentCreate returned success=false")
	}
	return nil
}

// AttachFile posts file contents as a comment (Linear doesn't support file attachments via API).
func (l *linear) AttachFile(issueID, filePath, fileName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	body := fmt.Sprintf("**Attached: %s**\n\n```\n%s\n```", fileName, string(content))
	return l.AddComment(issueID, body)
}

// GetIssue retrieves issue info from Linear.
func (l *linear) GetIssue(issueID string) (*Issue, error) {
	query := `query GetIssue($id: String!) {
		issue(id: $id) {
			id
			identifier
			title
			url
			state {
				name
			}
		}
	}`

	variables := map[string]interface{}{
		"id": issueID,
	}

	result, err := l.graphql(query, variables)
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}

	issueData, ok := result["issue"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}

	title, _ := issueData["title"].(string)
	identifier, _ := issueData["identifier"].(string)
	url, _ := issueData["url"].(string)

	statusName := ""
	if state, ok := issueData["state"].(map[string]interface{}); ok {
		statusName, _ = state["name"].(string)
	}

	return &Issue{
		ID:     identifier,
		Title:  title,
		Status: statusName,
		URL:    url,
	}, nil
}

// resolveIssueID takes either a Linear identifier (e.g. "ENG-123") or an internal
// UUID and returns the internal UUID needed for mutations.
func (l *linear) resolveIssueID(issueID string) (string, error) {
	query := `query GetIssue($id: String!) {
		issue(id: $id) {
			id
		}
	}`
	variables := map[string]interface{}{"id": issueID}

	result, err := l.graphql(query, variables)
	if err != nil {
		return "", fmt.Errorf("resolving issue ID: %w", err)
	}

	issueData, ok := result["issue"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("issue %q not found", issueID)
	}

	id, _ := issueData["id"].(string)
	if id == "" {
		return issueID, nil // assume it's already an internal ID
	}
	return id, nil
}

// graphql executes a GraphQL request against the Linear API.
func (l *linear) graphql(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", l.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", l.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("linear API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var gqlResp struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("linear GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data, nil
}

// ListIssues fetches active issues from Linear.
func (l *linear) ListIssues(label string, limit int) ([]Issue, error) {
	return l.Search(SearchQuery{
		Labels: func() []string {
			if label != "" {
				return []string{label}
			}
			return nil
		}(),
		Limit: limit,
	})
}

// Search fetches issues from Linear using a structured query.
func (l *linear) Search(query SearchQuery) ([]Issue, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	// Linear uses GraphQL filters — build the filter object
	filterParts := []string{`state: { type: { nin: ["completed", "canceled"] } }`}

	if query.Assignee != "" {
		switch strings.ToLower(query.Assignee) {
		case "unassigned", "none", "empty":
			filterParts = append(filterParts, `assignee: { null: true }`)
		default:
			filterParts = append(filterParts, fmt.Sprintf(`assignee: { displayName: { eq: %q } }`, query.Assignee))
		}
	}

	if query.Priority != "" {
		// Linear uses numeric priorities: 0=No priority, 1=Urgent, 2=High, 3=Medium, 4=Low
		var priorityNum string
		switch strings.ToLower(query.Priority) {
		case "urgent", "1":
			priorityNum = "1"
		case "high", "2":
			priorityNum = "2"
		case "medium", "3":
			priorityNum = "3"
		case "low", "4":
			priorityNum = "4"
		}
		if priorityNum != "" {
			filterParts = append(filterParts, fmt.Sprintf(`priority: { eq: %s }`, priorityNum))
		}
	}

	if len(query.Labels) > 0 {
		filterParts = append(filterParts, fmt.Sprintf(`labels: { name: { in: [%s] } }`,
			quoteStrings(query.Labels)))
	}

	filter := strings.Join(filterParts, ", ")

	graphqlQuery := fmt.Sprintf(`query ListIssues($teamKey: String!, $first: Int!) {
		team(id: $teamKey) {
			issues(first: $first, filter: { %s }) {
				nodes {
					id
					identifier
					title
					url
					description
					createdAt
					state {
						name
					}
					assignee {
						displayName
					}
					priority
					creator {
						displayName
					}
					labels {
						nodes {
							name
						}
					}
				}
			}
		}
	}`, filter)

	variables := map[string]interface{}{
		"teamKey": l.teamKey,
		"first":   limit,
	}

	result, err := l.graphql(graphqlQuery, variables)
	if err != nil {
		return nil, fmt.Errorf("listing issues: %w", err)
	}

	team, ok := result["team"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("team not found: %s", l.teamKey)
	}
	issuesData, ok := team["issues"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response shape")
	}
	nodes, ok := issuesData["nodes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected nodes shape")
	}

	var issues []Issue
	for _, n := range nodes {
		node, ok := n.(map[string]interface{})
		if !ok {
			continue
		}
		identifier, _ := node["identifier"].(string)
		title, _ := node["title"].(string)
		url, _ := node["url"].(string)
		description, _ := node["description"].(string)
		createdAt, _ := node["createdAt"].(string)

		statusName := ""
		if state, ok := node["state"].(map[string]interface{}); ok {
			statusName, _ = state["name"].(string)
		}

		assigneeName := ""
		if assignee, ok := node["assignee"].(map[string]interface{}); ok {
			assigneeName, _ = assignee["displayName"].(string)
		}

		creatorName := ""
		if creator, ok := node["creator"].(map[string]interface{}); ok {
			creatorName, _ = creator["displayName"].(string)
		}

		// Map Linear numeric priority to name
		priorityName := ""
		if p, ok := node["priority"].(float64); ok {
			switch int(p) {
			case 1:
				priorityName = "urgent"
			case 2:
				priorityName = "high"
			case 3:
				priorityName = "medium"
			case 4:
				priorityName = "low"
			}
		}

		var labels []string
		if labelsData, ok := node["labels"].(map[string]interface{}); ok {
			if labelNodes, ok := labelsData["nodes"].([]interface{}); ok {
				for _, ln := range labelNodes {
					if labelNode, ok := ln.(map[string]interface{}); ok {
						if name, ok := labelNode["name"].(string); ok {
							labels = append(labels, name)
						}
					}
				}
			}
		}

		issues = append(issues, Issue{
			ID:          identifier,
			Title:       title,
			Status:      statusName,
			URL:         url,
			Description: truncateDescription(description, 500),
			CreatedAt:   createdAt,
			Assignee:    assigneeName,
			Reporter:    creatorName,
			Priority:    priorityName,
			Labels:      labels,
		})
	}
	return issues, nil
}

// quoteStrings formats a slice of strings as quoted, comma-separated values for GraphQL.
func quoteStrings(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(quoted, ", ")
}
