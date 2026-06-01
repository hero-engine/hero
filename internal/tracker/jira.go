package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// jira implements the Tracker interface for Jira (Cloud or Server).
type jira struct {
	projectKey          string
	token               string // API token (Cloud) or PAT (Server)
	userEmail           string // required for Jira Cloud (basic auth: email + token)
	baseURL             string // e.g. "https://mycompany.atlassian.net"
	client              *http.Client
	jiraCfg             *config.JiraConfig // optional advanced config
	trackerKnowledgeDir string             // path to .hero/knowledge/tracker/ for field cache
	resolvedCustom      map[string]string  // resolved custom fields: lowercase name → field ID
	fieldDiscoveryDone  bool               // true once we've tried to discover fields
	// configuredSizeMapping is the workspace-configured size mapping
	// (hero.json: tracker.size_mapping). Nil → use the shipped default.
	// See internal/tracker/size_mapping.go.
	configuredSizeMapping *config.SizeMappingConfig
}

func newJira(project, token, userEmail, baseURL string) (*jira, error) {
	return newJiraWithConfig(project, token, userEmail, baseURL, nil, "")
}

func newJiraWithConfig(project, token, userEmail, baseURL string, jiraCfg *config.JiraConfig, trackerKnowledgeDir string) (*jira, error) {
	if project == "" {
		return nil, fmt.Errorf("jira project key is required")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("jira base_url is required (e.g. https://mycompany.atlassian.net)")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &jira{
		projectKey:          project,
		token:               token,
		userEmail:           userEmail,
		baseURL:             baseURL,
		client:              &http.Client{Timeout: 30 * time.Second},
		jiraCfg:             jiraCfg,
		trackerKnowledgeDir: trackerKnowledgeDir,
	}, nil
}

func (j *jira) Name() string { return "jira" }

// epicLinkField returns the configured custom field for epic links (defaults to Jira Cloud standard).
func (j *jira) epicLinkField() string {
	if j.jiraCfg != nil && j.jiraCfg.EpicLinkField != "" {
		return j.jiraCfg.EpicLinkField
	}
	return "customfield_10014"
}

// sprintField returns the configured custom field for sprint membership.
func (j *jira) sprintField() string {
	if j.jiraCfg != nil && j.jiraCfg.SprintField != "" {
		return j.jiraCfg.SprintField
	}
	return "customfield_10020"
}

// storyPointsField returns the configured custom field for story points.
func (j *jira) storyPointsField() string {
	if j.jiraCfg != nil && j.jiraCfg.StoryPointsField != "" {
		return j.jiraCfg.StoryPointsField
	}
	return "customfield_10016"
}

// customFields returns the resolved custom field map (lowercase name → field ID).
// On first call, merges config-supplied IDs with auto-discovered fields.
// Any names without IDs are resolved via the Jira /rest/api/3/field endpoint.
func (j *jira) customFields() map[string]string {
	if !j.fieldDiscoveryDone {
		j.resolveCustomFields()
	}
	return j.resolvedCustom
}

// customFieldIDs returns just the field IDs to include in API requests.
func (j *jira) customFieldIDs() []string {
	cf := j.customFields()
	ids := make([]string, 0, len(cf))
	for _, id := range cf {
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// DiscoveredFields returns the resolved custom field map for the caller to persist.
// Returns nil if no fields were discovered (all were pre-configured or none found).
func (j *jira) DiscoveredFields() map[string]string {
	cf := j.customFields()
	if len(cf) == 0 {
		return nil
	}
	// Return a copy
	out := make(map[string]string, len(cf))
	for k, v := range cf {
		out[k] = v
	}
	return out
}

// severityFieldNames lists common names for the "how bad is it" custom field
// across Jira instances. Checked in priority order; the first custom field
// whose name matches (case-insensitive) wins.
var severityFieldNames = []string{
	"severity",
	"criticality",
	"impact",
	"bug severity",
	"defect severity",
	"issue severity",
}

// resolveCustomFields builds the resolved custom field map by:
//  1. Loading any pre-configured name→ID pairs from JiraConfig.CustomFields
//  2. Handling the legacy SeverityField config
//  3. Loading previously discovered fields from .hero/knowledge/tracker/config.json
//  4. Auto-discovering common severity-like fields and any config names still missing IDs
//     via /rest/api/3/field (single HTTP call, only if needed)
//  5. Persisting newly discovered fields back to the cache
func (j *jira) resolveCustomFields() {
	j.fieldDiscoveryDone = true
	j.resolvedCustom = map[string]string{}

	// 1. Load pre-configured fields (name→ID pairs where ID is known).
	var needDiscovery []string // names that need ID lookup
	if j.jiraCfg != nil {
		for name, id := range j.jiraCfg.CustomFields {
			nameLower := strings.ToLower(name)
			if id != "" {
				j.resolvedCustom[nameLower] = id
			} else {
				needDiscovery = append(needDiscovery, nameLower)
			}
		}

		// 2. Legacy: SeverityField → treat as a pre-configured severity entry.
		if j.jiraCfg.SeverityField != "" {
			if _, exists := j.resolvedCustom["severity"]; !exists {
				j.resolvedCustom["severity"] = j.jiraCfg.SeverityField
			}
		}
	}

	// 3. Load cached discoveries from .hero/knowledge/tracker/config.json.
	var cached *fieldCache
	autoDiscoveryDone := false
	if j.trackerKnowledgeDir != "" {
		cached = loadFieldCache(j.trackerKnowledgeDir)
	}
	if cached != nil {
		// Invalidate cache if it was built for a different project.
		if cached.ProjectKey != "" && cached.ProjectKey != j.projectKey {
			cached = nil
		}
	}
	if cached != nil {
		autoDiscoveryDone = cached.AutoDiscoveryDone
		for name, id := range cached.Fields {
			if _, exists := j.resolvedCustom[name]; !exists {
				j.resolvedCustom[name] = id
			}
		}
	}

	// 4. Determine which names still need discovery.
	// Config names without IDs that weren't in cache:
	var stillNeed []string
	for _, name := range needDiscovery {
		if _, exists := j.resolvedCustom[name]; !exists {
			stillNeed = append(stillNeed, name)
		}
	}
	// Auto-discover severity-like names only if we haven't done it before:
	if !autoDiscoveryDone {
		for _, name := range severityFieldNames {
			if _, exists := j.resolvedCustom[name]; !exists {
				stillNeed = append(stillNeed, name)
			}
		}
	}

	if len(stillNeed) == 0 {
		return // everything resolved from config + cache
	}

	// 5. Single API call to discover remaining field IDs.
	discovered := j.discoverFieldIDs(stillNeed)
	for name, id := range discovered {
		if _, exists := j.resolvedCustom[name]; !exists {
			j.resolvedCustom[name] = id
		}
	}

	// 6. Persist all resolved fields to cache for future runs.
	// Always save after discovery so AutoDiscoveryDone is set even if nothing was found.
	if j.trackerKnowledgeDir != "" {
		saveFieldCache(j.trackerKnowledgeDir, j.resolvedCustom, j.projectKey)
	}
}

// discoverFieldIDs calls /rest/api/3/field and returns name→ID for any of the
// requested names that exist as custom fields. Names are matched case-insensitively.
// Errors are silently ignored (fields just won't be found).
func (j *jira) discoverFieldIDs(wantNames []string) map[string]string {
	if len(wantNames) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/rest/api/3/field", j.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var fields []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return nil
	}

	// Build a lookup set of wanted names (lowercase).
	want := map[string]bool{}
	for _, n := range wantNames {
		want[strings.ToLower(n)] = true
	}

	result := map[string]string{}
	// Collect ALL matching custom field IDs per name (there may be duplicates
	// across projects with the same name but different IDs).
	candidates := map[string][]string{} // name → [id1, id2, ...]
	for _, f := range fields {
		nameLower := strings.ToLower(f.Name)
		if want[nameLower] && strings.HasPrefix(f.ID, "customfield_") {
			candidates[nameLower] = append(candidates[nameLower], f.ID)
		}
	}

	// If any name has multiple candidate IDs, validate against the project by
	// fetching a sample issue with *all fields and checking which IDs exist.
	needsValidation := false
	for _, ids := range candidates {
		if len(ids) > 1 {
			needsValidation = true
			break
		}
	}

	var projectFieldIDs map[string]bool
	if needsValidation && j.projectKey != "" {
		projectFieldIDs = j.sampleIssueFieldIDs()
	}

	for name, ids := range candidates {
		if len(ids) == 1 {
			// Only one candidate — if we have project validation data, check it;
			// otherwise trust it.
			if projectFieldIDs != nil {
				if projectFieldIDs[ids[0]] {
					result[name] = ids[0]
				}
			} else {
				result[name] = ids[0]
			}
		} else {
			// Multiple candidates — pick the one present on this project.
			for _, id := range ids {
				if projectFieldIDs != nil && projectFieldIDs[id] {
					result[name] = id
					break
				}
			}
			// If no project validation available, skip ambiguous fields.
		}
	}
	return result
}

// sampleIssueFieldIDs fetches a single issue from the project with all fields
// and returns the set of field IDs present in the response. Used to validate
// which custom fields actually belong to this project's field scheme.
func (j *jira) sampleIssueFieldIDs() map[string]bool {
	jql := fmt.Sprintf("project = %s ORDER BY created DESC", j.projectKey)
	searchURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&maxResults=1&fields=*all",
		j.baseURL, jql)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result struct {
		Issues []map[string]json.RawMessage `json:"issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Issues) == 0 {
		return nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(result.Issues[0]["fields"], &fields); err != nil {
		return nil
	}

	ids := map[string]bool{}
	for k := range fields {
		if strings.HasPrefix(k, "customfield_") {
			ids[k] = true
		}
	}
	return ids
}

// CreateIssue creates a Jira issue from a spec. Returns the issue key (e.g. "PROJ-42").
func (j *jira) CreateIssue(s *spec.Spec) (string, error) {
	issueType := jiraIssueType(s.Type)

	fields := map[string]interface{}{
		"project": map[string]string{
			"key": j.projectKey,
		},
		"summary":     fmt.Sprintf("[%s] %s", s.Type, s.Title),
		"description": textToADF(IssueBody(s)),
		"issuetype": map[string]string{
			"name": issueType,
		},
		"labels": []string{
			fmt.Sprintf("hero-%s", s.Type),
		},
	}
	// Non-destructive size write on create: if the spec carries a
	// declared size and the mapping resolves it cleanly to a numeric
	// value, set it on the resolved story-points custom field. Jira
	// numeric fields expect a number, not a string — parse before
	// emitting. CreateIssue has nothing to overwrite, so the planner
	// isn't invoked.
	if s.Size != "" {
		if v, err := j.MapSize(s.Size); err == nil && v != "" {
			if n, perr := strconv.ParseFloat(v, 64); perr == nil {
				fields[j.storyPointsField()] = n
			}
		}
	}
	payload := map[string]interface{}{"fields": fields}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling issue: %w", err)
	}

	url := fmt.Sprintf("%s/rest/api/3/issue", j.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("creating issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("jira API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return result.Key, nil
}

// UpdateSize writes the mapped story-points value to a Jira issue.
// Resolves the field name from the configured size mapping (mirrors
// CreateIssue's storyPointsField()/MapSize() flow) and emits a PUT
// against the issue endpoint. The numeric value is sent as a JSON
// number — not a string — matching the create-path shape.
//
// Returns the response body on non-2xx so users see Jira's error
// (e.g. "Field 'customfield_xxx' does not exist") verbatim rather
// than a generic wrap.
func (j *jira) UpdateSize(issueID, localTier string) error {
	v, err := j.MapSize(localTier)
	if err != nil {
		return fmt.Errorf("mapping size %q: %w", localTier, err)
	}
	if v == "" {
		return fmt.Errorf("size %q maps to empty value", localTier)
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("parsing mapped size %q as number: %w", v, err)
	}

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			j.storyPointsField(): n,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling update: %w", err)
	}

	url := fmt.Sprintf("%s/rest/api/3/issue/%s", j.baseURL, issueID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("updating size: %w", err)
	}
	defer resp.Body.Close()

	// Jira returns 204 No Content on successful issue updates.
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira API returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// UpdateStatus updates the Jira issue status. If PushStatusTransitions is configured
// and contains a transition ID for the new status, a Jira workflow transition is
// performed. Otherwise falls back to adding a comment.
func (j *jira) UpdateStatus(issueID string, status spec.Status) error {
	if j.jiraCfg != nil && len(j.jiraCfg.PushStatusTransitions) > 0 {
		if transitionID, ok := j.jiraCfg.PushStatusTransitions[string(status)]; ok && transitionID != "" {
			return j.performTransition(issueID, transitionID, status)
		}
	}
	return j.addStatusComment(issueID, status)
}

// performTransition executes a Jira workflow transition and adds a comment.
func (j *jira) performTransition(issueID, transitionID string, status spec.Status) error {
	payload := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling transition: %w", err)
	}

	url := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", j.baseURL, issueID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("performing transition: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("jira transition API returned %d", resp.StatusCode)
	}

	// Also add a comment for traceability.
	_ = j.addStatusComment(issueID, status)
	return nil
}

func (j *jira) addStatusComment(issueID string, status spec.Status) error {
	comment := map[string]interface{}{
		"body": fmt.Sprintf("*Hero status update:* %s", StatusLabel(status)),
	}
	body, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("marshaling comment: %w", err)
	}

	url := fmt.Sprintf("%s/rest/api/3/issue/%s/comment", j.baseURL, issueID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("jira API returned %d when adding comment", resp.StatusCode)
	}
	return nil
}

// AddComment posts a plain-text comment to a Jira issue.
func (j *jira) AddComment(issueID, body string) error {
	adf := textToADF(body)
	comment := map[string]interface{}{
		"body": adf,
	}
	payload, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("marshaling comment: %w", err)
	}

	url := fmt.Sprintf("%s/rest/api/3/issue/%s/comment", j.baseURL, issueID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("jira comment API returned %d", resp.StatusCode)
	}
	return nil
}

// AttachFile uploads a file as an attachment to a Jira issue.
func (j *jira) AttachFile(issueID, filePath, fileName string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}
	writer.Close()

	url := fmt.Sprintf("%s/rest/api/3/issue/%s/attachments", j.baseURL, issueID)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Attachment API uses multipart, not JSON
	if j.userEmail != "" {
		req.SetBasicAuth(j.userEmail, j.token)
	} else {
		req.Header.Set("Authorization", "Bearer "+j.token)
	}
	req.Header.Set("X-Atlassian-Token", "no-check")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading attachment: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jira attachment API returned %d", resp.StatusCode)
	}
	return nil
}

// GetIssue retrieves issue info from Jira with full field mapping.
func (j *jira) GetIssue(issueID string) (*Issue, error) {
	epicField := j.epicLinkField()
	sprintField := j.sprintField()
	storyPtsField := j.storyPointsField()

	fields := fmt.Sprintf("key,summary,status,priority,assignee,labels,issuetype,description,created,reporter,%s,%s,%s",
		epicField, sprintField, storyPtsField)
	for _, cfID := range j.customFieldIDs() {
		fields += "," + cfID
	}
	url := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=%s", j.baseURL, issueID, fields)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return j.parseIssueRaw(raw)
}

// parseIssueRaw parses a raw Jira issue JSON object into an Issue.
func (j *jira) parseIssueRaw(raw map[string]json.RawMessage) (*Issue, error) {
	var key string
	if err := json.Unmarshal(raw["key"], &key); err != nil {
		return nil, fmt.Errorf("parsing key: %w", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw["fields"], &fields); err != nil {
		return nil, fmt.Errorf("parsing fields: %w", err)
	}

	issue := &Issue{
		ID:  key,
		URL: fmt.Sprintf("%s/browse/%s", j.baseURL, key),
	}

	if v, ok := fields["summary"]; ok {
		_ = json.Unmarshal(v, &issue.Title)
	}

	if v, ok := fields["status"]; ok {
		var st struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(v, &st); err == nil {
			issue.Status = st.Name
		}
	}

	if v, ok := fields["priority"]; ok {
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(v, &p); err == nil {
			name := strings.ToLower(p.Name)
			// Jira returns "None" when no priority is set — treat as empty
			if name != "" && name != "none" {
				issue.Priority = name
			}
		}
	}

	// Parse severity custom field (independent from priority — separate dimension).
	// Populate CustomFields map with all resolved custom field values.
	cf := j.customFields()
	if len(cf) > 0 {
		issue.CustomFields = map[string]string{}
		for name, fieldID := range cf {
			v, ok := fields[fieldID]
			if ok && string(v) != "null" {
				val := parseCustomFieldValue(v)
				if val != "" {
					issue.CustomFields[name] = val
				}
			}
		}
		// Set Severity convenience field: first match from known severity-like names.
		for _, sevName := range severityFieldNames {
			if val, ok := issue.CustomFields[sevName]; ok {
				issue.Severity = val
				break
			}
		}
	}

	if v, ok := fields["assignee"]; ok {
		var a struct {
			DisplayName string `json:"displayName"`
			EmailAddr   string `json:"emailAddress"`
		}
		if err := json.Unmarshal(v, &a); err == nil {
			if a.DisplayName != "" {
				issue.Assignee = a.DisplayName
			} else {
				issue.Assignee = a.EmailAddr
			}
		}
	}

	if v, ok := fields["labels"]; ok {
		var labels []string
		if err := json.Unmarshal(v, &labels); err == nil {
			issue.Labels = labels
		}
	}

	if v, ok := fields["issuetype"]; ok {
		var it struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(v, &it); err == nil {
			issue.IssueType = it.Name
		}
	}

	// Epic link (custom field — Jira Cloud: customfield_10014)
	epicField := j.epicLinkField()
	if v, ok := fields[epicField]; ok {
		var epicKey string
		if err := json.Unmarshal(v, &epicKey); err == nil && epicKey != "" {
			issue.EpicKey = epicKey
		}
	}

	// Reporter
	if v, ok := fields["reporter"]; ok {
		var r struct {
			DisplayName string `json:"displayName"`
			EmailAddr   string `json:"emailAddress"`
		}
		if err := json.Unmarshal(v, &r); err == nil {
			if r.DisplayName != "" {
				issue.Reporter = r.DisplayName
			} else {
				issue.Reporter = r.EmailAddr
			}
		}
	}

	// Created date
	if v, ok := fields["created"]; ok {
		var created string
		if err := json.Unmarshal(v, &created); err == nil {
			issue.CreatedAt = created
		}
	}

	// Description (v3 returns ADF, v2 returns plain text — handle both)
	if v, ok := fields["description"]; ok {
		issue.Description = extractADFText(v)
	}

	// Sprint (custom field — Jira Cloud: customfield_10020). Value is an array.
	sprintField := j.sprintField()
	if v, ok := fields[sprintField]; ok {
		var sprints []struct {
			Name  string `json:"name"`
			State string `json:"state"`
		}
		if err := json.Unmarshal(v, &sprints); err == nil {
			for _, sp := range sprints {
				if sp.State == "active" || sp.State == "future" {
					issue.SprintName = sp.Name
					break
				}
			}
			if issue.SprintName == "" && len(sprints) > 0 {
				issue.SprintName = sprints[len(sprints)-1].Name
			}
		}
	}

	return issue, nil
}

// parseCustomFieldValue extracts a string value from a Jira custom field.
// Custom fields come in several shapes:
//   - plain string: "Critical"
//   - select/option object: {"value": "Critical", "id": "10100"}
//   - cascading select: {"value": "Critical", "child": {"value": "P1"}}
//   - ADF document (rare for selects, but possible for text fields)
//
// Returns lowercase value, or empty string if unparseable/null.
func parseCustomFieldValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	// Try plain string first.
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return strings.ToLower(s)
	}

	// Try object with value/name (Jira select fields).
	var obj struct {
		Value string `json:"value"`
		Name  string `json:"name"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		if obj.Value != "" {
			return strings.ToLower(obj.Value)
		}
		if obj.Name != "" {
			return strings.ToLower(obj.Name)
		}
	}

	// Try ADF (unlikely for severity, but be safe).
	if text := extractADFText(raw); text != "" {
		return strings.ToLower(text)
	}

	return ""
}

// GetEpicIssues fetches all issues belonging to a specific epic.
func (j *jira) GetEpicIssues(epicKey string) ([]Issue, error) {
	epicField := j.epicLinkField()
	jql := fmt.Sprintf(`%s = %s ORDER BY created ASC`, epicField, epicKey)
	epicFields := "key,summary,status,priority,assignee,labels,issuetype"
	for _, cfID := range j.customFieldIDs() {
		epicFields += "," + cfID
	}
	return j.searchIssues(jql, 100, epicFields)
}

// GetTransitions returns available workflow transitions for an issue.
func (j *jira) GetTransitions(issueID string) ([]JiraTransition, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", j.baseURL, issueID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	j.setHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting transitions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Transitions []JiraTransition `json:"transitions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding transitions: %w", err)
	}
	return result.Transitions, nil
}

// JiraTransition represents a Jira workflow transition.
type JiraTransition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		Name string `json:"name"`
	} `json:"to"`
}

func (j *jira) setHeaders(req *http.Request) {
	if j.userEmail != "" {
		// Jira Cloud: basic auth with email + API token
		req.SetBasicAuth(j.userEmail, j.token)
	} else {
		// Jira Server/Data Center: Bearer token (PAT)
		req.Header.Set("Authorization", "Bearer "+j.token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// ListIssues fetches open issues from Jira using JQL. Optionally filters by label.
// defaultSearchLimit is the default max issues returned when no limit is specified.
const defaultSearchLimit = 50

// maxSearchLimit is the safety cap to prevent runaway pagination against huge projects.
// Users can override with --limit but this prevents accidental full-project dumps.
const maxSearchLimit = 500

func (j *jira) ListIssues(label string, limit int) ([]Issue, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	jql := fmt.Sprintf("project = %s AND statusCategory != Done", j.projectKey)
	if label != "" {
		jql += fmt.Sprintf(" AND labels = %q", label)
	}

	listFields := "key,summary,status,priority,assignee,labels,issuetype"
	for _, cfID := range j.customFieldIDs() {
		listFields += "," + cfID
	}
	return j.searchIssues(jql, limit, listFields)
}

// Search fetches issues from Jira using a structured query.
// Supports raw JQL, saved filter IDs, and field-level filters.
func (j *jira) Search(query SearchQuery) ([]Issue, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	fields := "key,summary,status,priority,assignee,labels,issuetype,created,reporter,description"
	for _, cfID := range j.customFieldIDs() {
		fields += "," + cfID
	}

	// If a saved filter ID is provided (and no raw query), use it
	if query.RawQuery == "" && query.FilterID != "" {
		jql := fmt.Sprintf("filter = %s", query.FilterID)
		if query.OrderBy != "" {
			jql += " ORDER BY " + query.OrderBy
		}
		return j.searchIssues(jql, limit, fields)
	}

	// If raw JQL is provided, scope to project unless user already included one
	if query.RawQuery != "" {
		jql := query.RawQuery
		if !jqlContainsProject(jql) && j.projectKey != "" {
			jql = fmt.Sprintf("project = %s AND (%s)", j.projectKey, jql)
		}
		return j.searchIssues(jql, limit, fields)
	}

	// Build JQL from individual field filters
	jql := j.buildJQL(query)
	return j.searchIssues(jql, limit, fields)
}

// jqlContainsProject returns true if the JQL string already includes a project clause.
func jqlContainsProject(jql string) bool {
	lower := strings.ToLower(jql)
	return strings.Contains(lower, "project =") || strings.Contains(lower, "project=") ||
		strings.Contains(lower, "project in")
}

// buildJQL constructs a JQL query string from a SearchQuery's field-level filters.
// It only adds clauses for fields that are set — defaults are applied upstream
// via ImportConfig.EffectiveBaseFilter().
func (j *jira) buildJQL(query SearchQuery) string {
	var clauses []string

	// Always scope to project and exclude resolved issues
	clauses = append(clauses, fmt.Sprintf("project = %s", j.projectKey))
	clauses = append(clauses, "statusCategory != Done")

	if query.IssueType != "" {
		clauses = append(clauses, fmt.Sprintf("issuetype = %q", query.IssueType))
	}

	if query.Assignee != "" {
		switch strings.ToLower(query.Assignee) {
		case "unassigned", "none", "empty":
			clauses = append(clauses, "assignee = EMPTY")
		default:
			clauses = append(clauses, fmt.Sprintf("assignee = %q", query.Assignee))
		}
	}

	for _, label := range query.Labels {
		if label != "" {
			clauses = append(clauses, fmt.Sprintf("labels = %q", label))
		}
	}

	if query.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = %q", query.Status))
	}

	if query.Priority != "" {
		clauses = append(clauses, fmt.Sprintf("priority = %q", query.Priority))
	}

	jql := strings.Join(clauses, " AND ")

	if query.OrderBy != "" {
		jql += " ORDER BY " + query.OrderBy
	} else {
		jql += " ORDER BY created DESC"
	}

	return jql
}

// searchIssues performs a JQL search and returns Issues with pagination.
// Uses the Jira Cloud v3 enhanced search endpoint (/rest/api/3/search/jql)
// which paginates via nextPageToken (not startAt/total).
func (j *jira) searchIssues(jql string, limit int, fields string) ([]Issue, error) {
	if fields == "" {
		fields = "key,summary,status"
	}

	var allIssues []Issue
	nextPageToken := ""
	searchURL := fmt.Sprintf("%s/rest/api/3/search/jql", j.baseURL)

	// Page size: fetch up to 100 per request, but respect the total limit.
	const maxPageSize = 100

	for {
		pageSize := maxPageSize
		if limit > 0 {
			remaining := limit - len(allIssues)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		q := req.URL.Query()
		q.Set("jql", jql)
		q.Set("maxResults", fmt.Sprintf("%d", pageSize))
		q.Set("fields", fields)
		if nextPageToken != "" {
			q.Set("nextPageToken", nextPageToken)
		}
		req.URL.RawQuery = q.Encode()
		j.setHeaders(req)

		resp, err := j.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("searching issues: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("jira API returned %d: %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			Issues        []map[string]json.RawMessage `json:"issues"`
			NextPageToken string                       `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decoding response: %w", err)
		}

		for _, raw := range result.Issues {
			issue, err := j.parseIssueRaw(raw)
			if err != nil {
				continue
			}
			allIssues = append(allIssues, *issue)
		}

		// Trim to limit if we overshot (page may contain more than remaining).
		if limit > 0 && len(allIssues) >= limit {
			allIssues = allIssues[:limit]
			break
		}

		// Stop if no more pages.
		if result.NextPageToken == "" || len(result.Issues) == 0 {
			break
		}
		nextPageToken = result.NextPageToken
	}

	return allIssues, nil
}

// jiraIssueType maps spec types to Jira issue type names.
func jiraIssueType(t spec.Type) string {
	switch t {
	case spec.TypeBug:
		return "Bug"
	case spec.TypeFeature:
		return "Story"
	case spec.TypeInitiative:
		return "Epic"
	default:
		return "Task"
	}
}

// textToADF converts a plain text string to Atlassian Document Format (ADF).
// ADF is required for description fields in Jira Cloud REST API v3.
func textToADF(text string) map[string]interface{} {
	// Split into paragraphs and create ADF paragraph nodes
	paragraphs := strings.Split(text, "\n\n")
	var content []interface{}
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Handle lines within a paragraph
		lines := strings.Split(p, "\n")
		var inlineContent []interface{}
		for i, line := range lines {
			if i > 0 {
				inlineContent = append(inlineContent, map[string]interface{}{"type": "hardBreak"})
			}
			inlineContent = append(inlineContent, map[string]interface{}{
				"type": "text",
				"text": line,
			})
		}
		content = append(content, map[string]interface{}{
			"type":    "paragraph",
			"content": inlineContent,
		})
	}

	if len(content) == 0 {
		content = []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": text},
				},
			},
		}
	}

	return map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

// extractADFText extracts plain text from an ADF (Atlassian Document Format) document.
// Handles the v3 description format which returns ADF instead of plain text.
func extractADFText(v json.RawMessage) string {
	// Try plain string first (v2 format)
	var plainText string
	if err := json.Unmarshal(v, &plainText); err == nil {
		return plainText
	}

	// Try ADF document format (v3)
	var doc struct {
		Type    string `json:"type"`
		Content []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(v, &doc); err != nil {
		return ""
	}

	var parts []string
	for _, block := range doc.Content {
		var blockText []string
		for _, inline := range block.Content {
			if inline.Text != "" {
				blockText = append(blockText, inline.Text)
			}
		}
		if len(blockText) > 0 {
			parts = append(parts, strings.Join(blockText, ""))
		}
	}
	return strings.Join(parts, "\n\n")
}
