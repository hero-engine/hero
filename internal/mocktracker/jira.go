package mocktracker

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// handleJira serves the subset of the Jira Cloud v3 API that
// internal/tracker/jira.go calls. Routes (sub is the path after the
// /jira prefix):
//
//	GET  /rest/api/3/search/jql?jql=&maxResults=&fields=
//	GET  /rest/api/3/issue/:key
//	POST /rest/api/3/issue
//	PUT  /rest/api/3/issue/:key
//	POST /rest/api/3/issue/:key/transitions
//	POST /rest/api/3/issue/:key/comment
//	GET  /rest/api/3/field
//
// Jira issues are key-based; the fixture's global_ids (ACME-NNN) are the
// keys directly. JQL is parsed best-effort for the clauses hero emits
// (project / status / assignee); ordering is normalized to global_id.
func (s *Server) handleJira(w http.ResponseWriter, r *http.Request, sub string) {
	parts := splitPath(sub)
	// /rest/api/3/...
	if len(parts) < 4 || parts[0] != "rest" || parts[1] != "api" || parts[2] != "3" {
		writeError(w, http.StatusNotFound, "jira endpoint not implemented: "+sub)
		return
	}
	tail := parts[3:]

	switch {
	case len(tail) == 2 && tail[0] == "search" && tail[1] == "jql" && r.Method == http.MethodGet:
		s.jiraSearch(w, r)
	case len(tail) == 1 && tail[0] == "field" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, []map[string]string{}) // no custom fields in the mock
	case len(tail) == 1 && tail[0] == "issue" && r.Method == http.MethodPost:
		s.jiraCreateIssue(w, r)
	case len(tail) == 2 && tail[0] == "issue":
		key := tail[1]
		switch r.Method {
		case http.MethodGet:
			s.jiraGetIssue(w, r, key)
		case http.MethodPut:
			s.jiraPutIssue(w, r, key)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case len(tail) == 3 && tail[0] == "issue" && tail[2] == "transitions" && r.Method == http.MethodPost:
		writeJSON(w, http.StatusNoContent, nil)
	case len(tail) == 3 && tail[0] == "issue" && tail[2] == "comment" && r.Method == http.MethodPost:
		writeJSON(w, http.StatusCreated, map[string]any{"id": "1"})
	default:
		writeError(w, http.StatusNotFound, "jira endpoint not implemented: "+sub)
	}
}

func jiraIssueType(t string) string {
	switch strings.ToLower(t) {
	case "bug":
		return "Bug"
	case "epic", "initiative":
		return "Epic"
	default:
		return "Story"
	}
}

// jiraFields builds the Jira `fields` object for an issue. description is
// returned as plain text (the adapter's adfToText handles both ADF and a
// bare string).
func jiraFields(r IssueRow) map[string]any {
	f := map[string]any{
		"summary":     r.Title,
		"description": r.Body,
		"status":      map[string]any{"name": jiraStatusName(r.Status)},
		"issuetype":   map[string]any{"name": jiraIssueType(r.Type)},
		"labels":      r.Labels,
		"created":     "2026-06-01T00:00:00.000+0000",
	}
	if p := priorityLabel(r.Labels); p != "" {
		f["priority"] = map[string]any{"name": p}
	}
	if r.Assignee != "" {
		f["assignee"] = map[string]any{"displayName": r.Assignee, "emailAddress": r.Assignee + "@acme.test"}
	}
	return f
}

func jiraStatusName(status string) string {
	switch strings.ToLower(status) {
	case "closed", "done":
		return "Done"
	case "in_review", "in-review":
		return "In Review"
	default:
		return "Open"
	}
}

// priorityLabel extracts the level from a priority::<level> scoped label.
func priorityLabel(labels []string) string {
	for _, l := range labels {
		if strings.HasPrefix(strings.ToLower(l), "priority::") {
			return strings.TrimPrefix(l, "priority::")
		}
	}
	return ""
}

func jiraIssueJSON(r IssueRow) map[string]any {
	return map[string]any{
		"id":     iidOf(r.GlobalID),
		"key":    r.GlobalID,
		"fields": jiraFields(r),
	}
}

func (s *Server) jiraSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := parseJQL(q.Get("jql"))

	var rows []IssueRow
	s.read(func(st *Store) { rows, _ = st.ListIssues(reqCtx(r), f) })

	maxResults := 50
	if v := q.Get("maxResults"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxResults = n
		}
	}
	offset := 0
	if tok := q.Get("nextPageToken"); tok != "" {
		offset, _ = strconv.Atoi(tok)
	}

	end := offset + maxResults
	if end > len(rows) {
		end = len(rows)
	}
	start := offset
	if start > len(rows) {
		start = len(rows)
	}
	window := rows[start:end]
	issues := make([]map[string]any, 0, len(window))
	for _, row := range window {
		issues = append(issues, jiraIssueJSON(row))
	}
	resp := map[string]any{"issues": issues}
	if end < len(rows) {
		resp["nextPageToken"] = strconv.Itoa(end)
	}
	writeJSON(w, http.StatusOK, resp)
}

// parseJQL extracts the clauses hero emits into an IssueFilter,
// best-effort. statusCategory != Done → open; status = X / assignee = X
// are honored; project and ORDER BY are accepted and ignored.
func parseJQL(jql string) IssueFilter {
	f := IssueFilter{State: "all"}
	lower := strings.ToLower(jql)
	if strings.Contains(lower, "statuscategory != done") || strings.Contains(lower, "statuscategory != \"done\"") {
		f.State = "open"
	}
	if strings.Contains(lower, "status = done") || strings.Contains(lower, "status = closed") {
		f.State = "closed"
	}
	return f
}

func (s *Server) jiraGetIssue(w http.ResponseWriter, r *http.Request, key string) {
	var row *IssueRow
	s.read(func(st *Store) { row, _ = st.GetIssueByGlobalID(reqCtx(r), key) })
	if row == nil {
		writeError(w, http.StatusNotFound, "Issue does not exist")
		return
	}
	writeJSON(w, http.StatusOK, jiraIssueJSON(*row))
}

func (s *Server) jiraCreateIssue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	var row *IssueRow
	s.write(func(st *Store) { row, _ = st.CreateIssue(reqCtx(r), "ACME", body.Fields.Summary, "", nil) })
	writeJSON(w, http.StatusCreated, map[string]any{"id": iidOf(row.GlobalID), "key": row.GlobalID})
}

func (s *Server) jiraPutIssue(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad body")
		return
	}
	var found bool
	s.write(func(st *Store) {
		row, _ := st.GetIssueByGlobalID(reqCtx(r), key)
		if row == nil {
			return
		}
		found = true
		fields := map[string]any{}
		if v, ok := body.Fields["summary"]; ok {
			var sm string
			json.Unmarshal(v, &sm)
			fields["title"] = sm
		}
		if v, ok := body.Fields["description"]; ok {
			fields["body"] = adfPlainText(v)
		}
		st.UpdateIssue(reqCtx(r), row.GlobalID, fields)
		if v, ok := body.Fields["labels"]; ok {
			var labels []string
			json.Unmarshal(v, &labels)
			// preserve priority label rotation handled client-side; replace set
			st.SetLabels(reqCtx(r), row.GlobalID, labels)
		}
		if v, ok := body.Fields["priority"]; ok {
			var p struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(v, &p) == nil && p.Name != "" {
				rotatePriorityLabel(reqCtx(r), st, row.GlobalID, p.Name)
			}
		}
	})
	if !found {
		writeError(w, http.StatusNotFound, "Issue does not exist")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// rotatePriorityLabel replaces the issue's priority::* label with the new
// level (Jira priority is a scalar field, but the fixture models priority
// as a scoped label so all four modes share one column-free convention).
func rotatePriorityLabel(ctx context.Context, st *Store, globalID, level string) {
	labels, _ := st.labels(ctx, globalID)
	var kept []string
	for _, l := range labels {
		if !strings.HasPrefix(strings.ToLower(l), "priority::") {
			kept = append(kept, l)
		}
	}
	kept = append(kept, "priority::"+strings.ToLower(level))
	st.SetLabels(ctx, globalID, kept)
}

// adfPlainText extracts text from an ADF document or a bare JSON string.
func adfPlainText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var doc struct {
		Content []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	var parts []string
	for _, block := range doc.Content {
		var line []string
		for _, run := range block.Content {
			line = append(line, run.Text)
		}
		parts = append(parts, strings.Join(line, ""))
	}
	return strings.Join(parts, "\n")
}
