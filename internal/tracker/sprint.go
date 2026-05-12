package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// SprintItem represents a single work item from a tracker sprint or iteration.
type SprintItem struct {
	ID                 string
	Title              string
	Description        string // body / description from tracker
	Type               string // "story", "bug", "task", "epic", "subtask"
	Status             string // tracker-native status string
	Priority           string // "highest", "high", "medium", "low", "lowest" (built-in priority)
	Severity           string // convenience: first severity-like value found in CustomFields
	Assignee           string
	Reporter           string
	Labels             []string
	SprintName         string
	EpicID             string // parent epic ID, if any
	EpicTitle          string
	LinkedIDs          []LinkedItem
	URL                string
	AcceptanceCriteria string
	StoryPoints        float64
	CustomFields       map[string]string // custom field values keyed by lowercase field name
}

// LinkedItem represents an issue link from the tracker.
type LinkedItem struct {
	ID       string
	LinkType string // "blocks", "is-blocked-by", "relates-to", "duplicates", "clones"
	URL      string
}

// SprintInfo contains metadata about a sprint or iteration.
type SprintInfo struct {
	ID    string
	Name  string
	Goal  string
	Start string
	End   string
	State string // "active", "future", "closed"
}

// SprintLoader extends Tracker with sprint/iteration loading capabilities.
type SprintLoader interface {
	// LoadSprint loads all items in a named or ID-referenced sprint.
	LoadSprint(sprintRef string) ([]SprintItem, *SprintInfo, error)

	// LoadActiveSprint loads the currently active sprint for a board/team.
	LoadActiveSprint(boardRef string) ([]SprintItem, *SprintInfo, error)

	// LoadIteration loads all items in a Linear iteration (date-based ref like "2026-04-14").
	LoadIteration(iterationRef string) ([]SprintItem, *SprintInfo, error)
}

// NewSprintLoader creates a SprintLoader for the given tracker config.
// Returns an error if the tracker does not support sprint loading.
// The optional jiraCfg parameter provides advanced Jira field mappings.
// trackerKnowledgeDir is the path to .hero/knowledge/tracker/ for field cache persistence.
func NewSprintLoader(cfg *config.TrackerConfig, jiraCfg *config.JiraConfig, trackerKnowledgeDir string) (SprintLoader, error) {
	if cfg == nil || cfg.Type == "" {
		return nil, fmt.Errorf("no tracker configured")
	}

	token, err := cfg.ResolveToken()
	if err != nil {
		return nil, err
	}

	switch cfg.Type {
	case "jira":
		return newJiraSprintLoader(cfg.Project, token, cfg.UserEmail, cfg.BaseURL, jiraCfg, trackerKnowledgeDir)
	case "linear":
		return newLinearSprintLoader(cfg.Project, token, cfg.BaseURL)
	default:
		return nil, fmt.Errorf("sprint loading is not supported for tracker type %q (supported: jira, linear)", cfg.Type)
	}
}

// ---------------------------------------------------------------------------
// Jira sprint loader
// ---------------------------------------------------------------------------

type jiraSprintLoader struct {
	projectKey          string
	token               string
	userEmail           string // required for Jira Cloud (basic auth: email + token)
	baseURL             string
	client              *http.Client
	jiraCfg             *config.JiraConfig
	trackerKnowledgeDir string            // path to .hero/knowledge/tracker/ for field cache
	resolvedCustom      map[string]string // resolved custom fields: lowercase name → field ID
	fieldDiscoveryDone  bool
}

func newJiraSprintLoader(project, token, userEmail, baseURL string, jiraCfg *config.JiraConfig, trackerKnowledgeDir string) (*jiraSprintLoader, error) {
	if project == "" {
		return nil, fmt.Errorf("jira project key is required")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("jira base_url is required")
	}
	return &jiraSprintLoader{
		projectKey:          project,
		token:               token,
		userEmail:           userEmail,
		baseURL:             strings.TrimRight(baseURL, "/"),
		client:              &http.Client{Timeout: 30 * time.Second},
		jiraCfg:             jiraCfg,
		trackerKnowledgeDir: trackerKnowledgeDir,
	}, nil
}

// storyPointsField returns the configured custom field ID for story points.
func (j *jiraSprintLoader) storyPointsField() string {
	if j.jiraCfg != nil && j.jiraCfg.StoryPointsField != "" {
		return j.jiraCfg.StoryPointsField
	}
	return "customfield_10016"
}

// acceptanceCriteriaField returns the configured custom field ID for acceptance criteria.
// Returns empty string if not configured (acceptance criteria won't be imported).
func (j *jiraSprintLoader) acceptanceCriteriaField() string {
	if j.jiraCfg != nil && j.jiraCfg.AcceptanceCriteriaField != "" {
		return j.jiraCfg.AcceptanceCriteriaField
	}
	return ""
}

// customFieldMap returns the resolved custom field map (name → ID) for the sprint loader.
// Uses the same resolution logic as jira.customFields(): config → cache → discovery.
func (j *jiraSprintLoader) customFieldMap() map[string]string {
	if !j.fieldDiscoveryDone {
		j.resolveCustomFields()
	}
	return j.resolvedCustom
}

// customFieldIDs returns just the field IDs to include in API requests.
func (j *jiraSprintLoader) customFieldIDs() []string {
	cf := j.customFieldMap()
	ids := make([]string, 0, len(cf))
	for _, id := range cf {
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// resolveCustomFields mirrors jira.resolveCustomFields() for the sprint loader.
func (j *jiraSprintLoader) resolveCustomFields() {
	j.fieldDiscoveryDone = true
	j.resolvedCustom = map[string]string{}

	// Config-supplied fields.
	var needDiscovery []string
	if j.jiraCfg != nil {
		for name, id := range j.jiraCfg.CustomFields {
			nameLower := strings.ToLower(name)
			if id != "" {
				j.resolvedCustom[nameLower] = id
			} else {
				needDiscovery = append(needDiscovery, nameLower)
			}
		}
		if j.jiraCfg.SeverityField != "" {
			if _, exists := j.resolvedCustom["severity"]; !exists {
				j.resolvedCustom["severity"] = j.jiraCfg.SeverityField
			}
		}
	}

	// Cached discoveries.
	autoDiscoveryDone := false
	if j.trackerKnowledgeDir != "" {
		if cached := loadFieldCache(j.trackerKnowledgeDir); cached != nil {
			autoDiscoveryDone = cached.AutoDiscoveryDone
			for name, id := range cached.Fields {
				if _, exists := j.resolvedCustom[name]; !exists {
					j.resolvedCustom[name] = id
				}
			}
		}
	}

	// Determine what still needs discovery.
	var stillNeed []string
	for _, name := range needDiscovery {
		if _, exists := j.resolvedCustom[name]; !exists {
			stillNeed = append(stillNeed, name)
		}
	}
	if !autoDiscoveryDone {
		for _, name := range severityFieldNames {
			if _, exists := j.resolvedCustom[name]; !exists {
				stillNeed = append(stillNeed, name)
			}
		}
	}
	if len(stillNeed) == 0 {
		return
	}

	// Discover via API.
	discovered := j.discoverFieldIDs(stillNeed)
	for name, id := range discovered {
		if _, exists := j.resolvedCustom[name]; !exists {
			j.resolvedCustom[name] = id
		}
	}
	if j.trackerKnowledgeDir != "" {
		saveFieldCache(j.trackerKnowledgeDir, j.resolvedCustom, j.projectKey)
	}
}

// discoverFieldIDs calls /rest/api/3/field for the sprint loader.
func (j *jiraSprintLoader) discoverFieldIDs(wantNames []string) map[string]string {
	if len(wantNames) == 0 {
		return nil
	}

	apiURL := fmt.Sprintf("%s/rest/api/3/field", j.baseURL)
	var fields []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := j.get(apiURL, &fields); err != nil {
		return nil
	}

	want := map[string]bool{}
	for _, n := range wantNames {
		want[strings.ToLower(n)] = true
	}

	result := map[string]string{}
	for _, f := range fields {
		nameLower := strings.ToLower(f.Name)
		if want[nameLower] && strings.HasPrefix(f.ID, "customfield_") {
			result[nameLower] = f.ID
		}
	}
	return result
}

// LoadSprint loads a Jira sprint by name or numeric ID.
func (j *jiraSprintLoader) LoadSprint(sprintRef string) ([]SprintItem, *SprintInfo, error) {
	info, err := j.findSprint(sprintRef, "")
	if err != nil {
		return nil, nil, err
	}
	items, err := j.loadSprintItems(info.ID, info.Name)
	if err != nil {
		return nil, nil, err
	}
	return items, info, nil
}

// LoadActiveSprint loads the active sprint for the given board name or ID.
func (j *jiraSprintLoader) LoadActiveSprint(boardRef string) ([]SprintItem, *SprintInfo, error) {
	boardID, err := j.resolveBoardID(boardRef)
	if err != nil {
		return nil, nil, err
	}
	info, err := j.findSprint("", boardID)
	if err != nil {
		return nil, nil, err
	}
	items, err := j.loadSprintItems(info.ID, info.Name)
	if err != nil {
		return nil, nil, err
	}
	return items, info, nil
}

// LoadIteration is not applicable to Jira — returns an error.
func (j *jiraSprintLoader) LoadIteration(ref string) ([]SprintItem, *SprintInfo, error) {
	return nil, nil, fmt.Errorf("use LoadSprint or LoadActiveSprint for Jira; LoadIteration is for Linear")
}

// resolveBoardID resolves a board name to its numeric ID using the Agile API.
func (j *jiraSprintLoader) resolveBoardID(boardRef string) (string, error) {
	apiURL := fmt.Sprintf("%s/rest/agile/1.0/board?projectKeyOrId=%s",
		j.baseURL, url.QueryEscape(j.projectKey))

	var result struct {
		Values []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"values"`
	}
	if err := j.get(apiURL, &result); err != nil {
		return "", fmt.Errorf("listing boards: %w", err)
	}

	for _, b := range result.Values {
		if fmt.Sprint(b.ID) == boardRef || strings.EqualFold(b.Name, boardRef) {
			return fmt.Sprint(b.ID), nil
		}
	}

	if len(result.Values) > 0 {
		// Fall back to the first board for this project
		return fmt.Sprint(result.Values[0].ID), nil
	}

	return "", fmt.Errorf("no board found for project %q matching %q", j.projectKey, boardRef)
}

// findSprint finds a sprint by name/ID (if sprintRef is set) or active sprint on a board (if boardID is set).
func (j *jiraSprintLoader) findSprint(sprintRef, boardID string) (*SprintInfo, error) {
	// If sprintRef looks like a number, try direct sprint lookup
	if sprintRef != "" {
		isNumeric := true
		for _, c := range sprintRef {
			if c < '0' || c > '9' {
				isNumeric = false
				break
			}
		}
		if isNumeric {
			var sprintResult struct {
				ID    int    `json:"id"`
				Name  string `json:"name"`
				Goal  string `json:"goal"`
				Start string `json:"startDate"`
				End   string `json:"endDate"`
				State string `json:"state"`
			}
			apiURL := fmt.Sprintf("%s/rest/agile/1.0/sprint/%s", j.baseURL, sprintRef)
			if err := j.get(apiURL, &sprintResult); err == nil {
				return &SprintInfo{
					ID:    fmt.Sprint(sprintResult.ID),
					Name:  sprintResult.Name,
					Goal:  sprintResult.Goal,
					Start: sprintResult.Start,
					End:   sprintResult.End,
					State: sprintResult.State,
				}, nil
			}
		}
	}

	// Search via board
	if boardID == "" {
		boardID, _ = j.resolveBoardID("")
	}

	var sprintState string
	if sprintRef == "" {
		sprintState = "active"
	}

	apiURL := fmt.Sprintf("%s/rest/agile/1.0/board/%s/sprint", j.baseURL, boardID)
	if sprintState != "" {
		apiURL += "?state=" + sprintState
	}

	var result struct {
		Values []struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Goal  string `json:"goal"`
			Start string `json:"startDate"`
			End   string `json:"endDate"`
			State string `json:"state"`
		} `json:"values"`
	}
	if err := j.get(apiURL, &result); err != nil {
		// Fall back to JQL for installations without Agile API
		return j.findSprintViaJQL(sprintRef)
	}

	for _, s := range result.Values {
		if sprintRef == "" || strings.EqualFold(s.Name, sprintRef) || fmt.Sprint(s.ID) == sprintRef {
			return &SprintInfo{
				ID:    fmt.Sprint(s.ID),
				Name:  s.Name,
				Goal:  s.Goal,
				Start: s.Start,
				End:   s.End,
				State: s.State,
			}, nil
		}
	}

	if sprintRef == "" && len(result.Values) > 0 {
		s := result.Values[0]
		return &SprintInfo{
			ID:    fmt.Sprint(s.ID),
			Name:  s.Name,
			Goal:  s.Goal,
			Start: s.Start,
			End:   s.End,
			State: s.State,
		}, nil
	}

	return nil, fmt.Errorf("sprint %q not found on board %s", sprintRef, boardID)
}

// findSprintViaJQL falls back to JQL-based sprint querying for Jira Server without Agile API.
func (j *jiraSprintLoader) findSprintViaJQL(sprintRef string) (*SprintInfo, error) {
	if sprintRef == "" {
		return &SprintInfo{
			ID:   "jql",
			Name: "active sprint",
		}, nil
	}
	return &SprintInfo{
		ID:   "jql:" + sprintRef,
		Name: sprintRef,
	}, nil
}

// loadSprintItems fetches all issues in the sprint (using Agile API or JQL fallback).
func (j *jiraSprintLoader) loadSprintItems(sprintID, sprintName string) ([]SprintItem, error) {
	var jql string
	if strings.HasPrefix(sprintID, "jql:") {
		ref := strings.TrimPrefix(sprintID, "jql:")
		jql = fmt.Sprintf(`project = %s AND sprint = %q ORDER BY rank ASC`, j.projectKey, ref)
	} else if sprintID == "jql" {
		jql = fmt.Sprintf(`project = %s AND sprint in openSprints() ORDER BY rank ASC`, j.projectKey)
	} else {
		jql = fmt.Sprintf(`sprint = %s ORDER BY rank ASC`, sprintID)
	}

	// Build dynamic fields list including configurable custom field IDs.
	fieldList := "summary,description,issuetype,status,priority,assignee,reporter,labels,components,fixVersions,parent,issuelinks"
	spField := j.storyPointsField()
	acField := j.acceptanceCriteriaField()
	fieldList += "," + spField
	if acField != "" {
		fieldList += "," + acField
	}
	for _, cfID := range j.customFieldIDs() {
		fieldList += "," + cfID
	}
	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&maxResults=100&fields=%s",
		j.baseURL, url.QueryEscape(jql), fieldList)

	var result struct {
		Issues []struct {
			Key    string          `json:"key"`
			Fields json.RawMessage `json:"fields"`
		} `json:"issues"`
	}

	if err := j.get(apiURL, &result); err != nil {
		return nil, fmt.Errorf("fetching sprint issues: %w", err)
	}

	var items []SprintItem
	for _, issue := range result.Issues {
		// Parse known fields via struct.
		var f struct {
			Summary     string          `json:"summary"`
			Description json.RawMessage `json:"description"`
			IssueType   struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
			Priority struct {
				Name string `json:"name"`
			} `json:"priority"`
			Assignee *struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
			Reporter *struct {
				DisplayName string `json:"displayName"`
			} `json:"reporter"`
			Labels     []string `json:"labels"`
			Components []struct {
				Name string `json:"name"`
			} `json:"components"`
			FixVersions []struct {
				Name string `json:"name"`
			} `json:"fixVersions"`
			Parent *struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string `json:"summary"`
				} `json:"fields"`
			} `json:"parent"`
			IssueLinks []struct {
				ID   string `json:"id"`
				Type struct {
					Name    string `json:"name"`
					Inward  string `json:"inward"`
					Outward string `json:"outward"`
				} `json:"type"`
				InwardIssue *struct {
					Key string `json:"key"`
				} `json:"inwardIssue"`
				OutwardIssue *struct {
					Key string `json:"key"`
				} `json:"outwardIssue"`
			} `json:"issuelinks"`
		}
		if err := json.Unmarshal(issue.Fields, &f); err != nil {
			continue // skip malformed issues
		}

		// Parse custom fields dynamically (field IDs are configurable).
		var customFields map[string]json.RawMessage
		if err := json.Unmarshal(issue.Fields, &customFields); err != nil {
			customFields = nil
		}

		item := SprintItem{
			ID:         issue.Key,
			Title:      f.Summary,
			Type:       jiraTypeToHero(f.IssueType.Name),
			Status:     f.Status.Name,
			Priority:   strings.ToLower(f.Priority.Name),
			SprintName: sprintName,
			URL:        fmt.Sprintf("%s/browse/%s", j.baseURL, issue.Key),
		}

		if f.Assignee != nil {
			item.Assignee = f.Assignee.DisplayName
		}
		if f.Reporter != nil {
			item.Reporter = f.Reporter.DisplayName
		}

		item.Labels = append(item.Labels, f.Labels...)
		for _, c := range f.Components {
			item.Labels = append(item.Labels, c.Name)
		}
		for _, v := range f.FixVersions {
			item.Labels = append(item.Labels, v.Name)
		}

		if f.Parent != nil {
			item.EpicID = f.Parent.Key
			item.EpicTitle = f.Parent.Fields.Summary
		}

		for _, link := range f.IssueLinks {
			linkType := jiraLinkType(link.Type.Name, link.Type.Outward)
			if link.OutwardIssue != nil {
				item.LinkedIDs = append(item.LinkedIDs, LinkedItem{
					ID:       link.OutwardIssue.Key,
					LinkType: linkType,
					URL:      fmt.Sprintf("%s/browse/%s", j.baseURL, link.OutwardIssue.Key),
				})
			} else if link.InwardIssue != nil {
				item.LinkedIDs = append(item.LinkedIDs, LinkedItem{
					ID:       link.InwardIssue.Key,
					LinkType: invertLinkType(linkType),
					URL:      fmt.Sprintf("%s/browse/%s", j.baseURL, link.InwardIssue.Key),
				})
			}
		}

		if f.Description != nil {
			item.Description = extractADFText(f.Description)
		}

		// Extract custom fields by configurable ID.
		if customFields != nil {
			if acField != "" {
				if raw, ok := customFields[acField]; ok && string(raw) != "null" {
					var s string
					if json.Unmarshal(raw, &s) == nil {
						item.AcceptanceCriteria = s
					}
				}
			}
			if raw, ok := customFields[spField]; ok && string(raw) != "null" {
				var v float64
				if json.Unmarshal(raw, &v) == nil {
					item.StoryPoints = v
				}
			}
			// Parse severity custom field independently (not a priority fallback).
			// Populate CustomFields map with all resolved custom field values.
			cfMap := j.customFieldMap()
			if len(cfMap) > 0 {
				item.CustomFields = map[string]string{}
				for name, fieldID := range cfMap {
					if raw, ok := customFields[fieldID]; ok && string(raw) != "null" {
						val := parseCustomFieldValue(raw)
						if val != "" {
							item.CustomFields[name] = val
						}
					}
				}
				// Set Severity convenience field: first match from known severity-like names.
				for _, sevName := range severityFieldNames {
					if val, ok := item.CustomFields[sevName]; ok {
						item.Severity = val
						break
					}
				}
			}
		}

		items = append(items, item)
	}

	return items, nil
}

func (j *jiraSprintLoader) get(apiURL string, out interface{}) error {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	if j.userEmail != "" {
		// Jira Cloud: basic auth with email + API token
		req.SetBasicAuth(j.userEmail, j.token)
	} else {
		// Jira Server/Data Center: Bearer token (PAT)
		req.Header.Set("Authorization", "Bearer "+j.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Jira API %s returned %d: %s", apiURL, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// jiraTypeToHero maps Jira issue type names to Hero spec types.
func jiraTypeToHero(jiraType string) string {
	switch strings.ToLower(jiraType) {
	case "epic":
		return "initiative"
	case "bug":
		return "bug"
	case "story", "feature request":
		return "feature"
	case "sub-task", "subtask":
		return "feature"
	default:
		return "feature"
	}
}

// jiraLinkType maps Jira link type names to Hero relation kinds.
func jiraLinkType(typeName, outward string) string {
	name := strings.ToLower(typeName)
	out := strings.ToLower(outward)
	switch {
	case strings.Contains(name, "block") || strings.Contains(out, "block"):
		return "blocks"
	case strings.Contains(name, "duplicate") || strings.Contains(out, "duplicate"):
		return "duplicate-of"
	case strings.Contains(name, "clone") || strings.Contains(out, "clone"):
		return "derived-from"
	default:
		return "related"
	}
}

func invertLinkType(lt string) string {
	switch lt {
	case "blocks":
		return "blocked-by"
	case "blocked-by":
		return "blocks"
	default:
		return lt
	}
}

// ---------------------------------------------------------------------------
// Linear sprint (iteration) loader
// ---------------------------------------------------------------------------

type linearSprintLoader struct {
	teamKey string
	token   string
	apiURL  string
	client  *http.Client
}

func newLinearSprintLoader(project, token, baseURL string) (*linearSprintLoader, error) {
	if project == "" {
		return nil, fmt.Errorf("linear team key is required")
	}
	if baseURL == "" {
		baseURL = defaultLinearAPI
	}
	return &linearSprintLoader{
		teamKey: project,
		token:   token,
		apiURL:  strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (l *linearSprintLoader) LoadSprint(sprintRef string) ([]SprintItem, *SprintInfo, error) {
	return l.LoadIteration(sprintRef)
}

func (l *linearSprintLoader) LoadActiveSprint(boardRef string) ([]SprintItem, *SprintInfo, error) {
	return l.LoadIteration("") // load current/active iteration
}

// LoadIteration loads a Linear iteration by date (e.g. "2026-04-14") or name.
func (l *linearSprintLoader) LoadIteration(iterationRef string) ([]SprintItem, *SprintInfo, error) {
	// Query the team's cycles (Linear's term for iterations/sprints)
	query := `query GetCycle($teamId: String!) {
		team(id: $teamId) {
			cycles(filter: { isActive: { eq: true } }) {
				nodes {
					id
					name
					number
					startsAt
					endsAt
					issues {
						nodes {
							id
							identifier
							title
							description
							state { name }
							priority
							priorityLabel
							assignee { displayName }
							labels { nodes { name } }
							parent { identifier title }
							relations {
								nodes {
									type
									relatedIssue { identifier }
								}
							}
							url
						}
					}
				}
			}
		}
	}`

	vars := map[string]interface{}{"teamId": l.teamKey}
	result, err := l.graphql(query, vars)
	if err != nil {
		return nil, nil, fmt.Errorf("loading Linear iteration: %w", err)
	}

	team, ok := result["team"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("team not found")
	}
	cycles, _ := team["cycles"].(map[string]interface{})
	nodes, _ := cycles["nodes"].([]interface{})

	if len(nodes) == 0 {
		return nil, nil, fmt.Errorf("no active cycle found for team %q", l.teamKey)
	}

	// Take first active cycle (or match by iterationRef if specified)
	var cycle map[string]interface{}
	for _, n := range nodes {
		c, ok := n.(map[string]interface{})
		if !ok {
			continue
		}
		if iterationRef == "" {
			cycle = c
			break
		}
		name, _ := c["name"].(string)
		startsAt, _ := c["startsAt"].(string)
		if strings.Contains(strings.ToLower(name), strings.ToLower(iterationRef)) ||
			strings.HasPrefix(startsAt, iterationRef) {
			cycle = c
			break
		}
	}

	if cycle == nil {
		cycle = nodes[0].(map[string]interface{})
	}

	cycleName, _ := cycle["name"].(string)
	cycleID, _ := cycle["id"].(string)
	startsAt, _ := cycle["startsAt"].(string)
	endsAt, _ := cycle["endsAt"].(string)

	info := &SprintInfo{
		ID:    cycleID,
		Name:  cycleName,
		Start: startsAt,
		End:   endsAt,
		State: "active",
	}

	issuesMap, _ := cycle["issues"].(map[string]interface{})
	issueNodes, _ := issuesMap["nodes"].([]interface{})

	var items []SprintItem
	for _, n := range issueNodes {
		node, ok := n.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := node["identifier"].(string)
		title, _ := node["title"].(string)
		desc, _ := node["description"].(string)
		nodeURL, _ := node["url"].(string)
		priorityLabel, _ := node["priorityLabel"].(string)

		statusName := ""
		if state, ok := node["state"].(map[string]interface{}); ok {
			statusName, _ = state["name"].(string)
		}

		assignee := ""
		if a, ok := node["assignee"].(map[string]interface{}); ok {
			assignee, _ = a["displayName"].(string)
		}

		var labels []string
		if lbls, ok := node["labels"].(map[string]interface{}); ok {
			if lnodes, ok := lbls["nodes"].([]interface{}); ok {
				for _, ln := range lnodes {
					if lm, ok := ln.(map[string]interface{}); ok {
						if name, ok := lm["name"].(string); ok {
							labels = append(labels, name)
						}
					}
				}
			}
		}

		epicID, epicTitle := "", ""
		if parent, ok := node["parent"].(map[string]interface{}); ok {
			epicID, _ = parent["identifier"].(string)
			epicTitle, _ = parent["title"].(string)
		}

		var linked []LinkedItem
		if rels, ok := node["relations"].(map[string]interface{}); ok {
			if rnodes, ok := rels["nodes"].([]interface{}); ok {
				for _, rn := range rnodes {
					rm, ok := rn.(map[string]interface{})
					if !ok {
						continue
					}
					relType, _ := rm["type"].(string)
					if ri, ok := rm["relatedIssue"].(map[string]interface{}); ok {
						relID, _ := ri["identifier"].(string)
						linked = append(linked, LinkedItem{
							ID:       relID,
							LinkType: linearRelationType(relType),
						})
					}
				}
			}
		}

		items = append(items, SprintItem{
			ID:          id,
			Title:       title,
			Description: desc,
			Type:        "feature",
			Status:      statusName,
			Priority:    strings.ToLower(priorityLabel),
			Assignee:    assignee,
			Labels:      labels,
			SprintName:  cycleName,
			EpicID:      epicID,
			EpicTitle:   epicTitle,
			LinkedIDs:   linked,
			URL:         nodeURL,
		})
	}

	return items, info, nil
}

func (l *linearSprintLoader) graphql(query string, vars map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{"query": query, "variables": vars}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", l.apiURL, bytes.NewReader(bodyBytes))
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
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("linear API returned %d: %s", resp.StatusCode, string(b))
	}

	var gqlResp struct {
		Data   map[string]interface{}     `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, err
	}
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("linear GraphQL error: %s", gqlResp.Errors[0].Message)
	}
	return gqlResp.Data, nil
}

func linearRelationType(t string) string {
	switch strings.ToLower(t) {
	case "blocks":
		return "blocks"
	case "blocked_by":
		return "blocked-by"
	case "duplicate_of":
		return "duplicate-of"
	case "clones":
		return "derived-from"
	default:
		return "related"
	}
}
