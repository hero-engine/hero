package mocktracker

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleLinear serves the single Linear GraphQL endpoint (POST /graphql).
// It is not a general GraphQL engine: it dispatches on the root field
// present in the query string (the set hero's adapter sends is small and
// stable) and returns {"data": {...}}.
//
// Operations: issueCreate / issueUpdate / commentCreate mutations;
// team{issues} list query; issue(id:) single query (GetIssue / GetFields
// / resolveIssueID). Linear identifiers are the fixture's global_ids.
func (s *Server) handleLinear(w http.ResponseWriter, r *http.Request, sub string) {
	if !strings.HasSuffix(sub, "/graphql") && sub != "/graphql" {
		writeError(w, http.StatusNotFound, "linear endpoint not implemented: "+sub)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad graphql body")
		return
	}
	q := body.Query
	vars := body.Variables

	switch {
	case strings.Contains(q, "issueCreate("):
		s.linearCreate(w, r, vars)
	case strings.Contains(q, "issueUpdate("):
		s.linearUpdate(w, r, vars)
	case strings.Contains(q, "commentCreate("):
		writeJSON(w, http.StatusOK, gqlData(map[string]any{"commentCreate": map[string]any{"success": true}}))
	case strings.Contains(q, "team(") && strings.Contains(q, "issues("):
		s.linearList(w, r, vars)
	case strings.Contains(q, "issue(id:") || strings.Contains(q, "issue(id :"):
		s.linearGetIssue(w, r, vars)
	default:
		writeJSON(w, http.StatusOK, map[string]any{
			"errors": []map[string]string{{"message": "unsupported operation"}},
		})
	}
}

func gqlData(data map[string]any) map[string]any {
	return map[string]any{"data": data}
}

func linearPriorityInt(labels []string) int {
	switch strings.ToLower(priorityLabel(labels)) {
	case "urgent", "critical":
		return 1
	case "high":
		return 2
	case "medium":
		return 3
	case "low":
		return 4
	default:
		return 0
	}
}

func linearNode(r IssueRow) map[string]any {
	labelNodes := make([]map[string]string, 0, len(r.Labels))
	for _, l := range r.Labels {
		labelNodes = append(labelNodes, map[string]string{"name": l})
	}
	node := map[string]any{
		"id":          r.GlobalID,
		"identifier":  r.GlobalID,
		"title":       r.Title,
		"url":         "https://linear.example/issue/" + r.GlobalID,
		"description": r.Body,
		"createdAt":   "2026-06-01T00:00:00.000Z",
		"state":       map[string]any{"name": linearStateName(r.Status)},
		"priority":    linearPriorityInt(r.Labels),
		"creator":     map[string]any{"displayName": "Acme Bot"},
		"labels":      map[string]any{"nodes": labelNodes},
	}
	if r.Weight.Valid {
		node["estimate"] = float64(r.Weight.Int64)
	} else {
		node["estimate"] = nil
	}
	if r.Assignee != "" {
		node["assignee"] = map[string]any{"displayName": r.Assignee}
	} else {
		node["assignee"] = nil
	}
	return node
}

func linearStateName(status string) string {
	switch strings.ToLower(status) {
	case "closed", "done":
		return "Done"
	case "in_review", "in-review":
		return "In Review"
	default:
		return "Todo"
	}
}

func (s *Server) linearGetIssue(w http.ResponseWriter, r *http.Request, vars map[string]interface{}) {
	id, _ := vars["id"].(string)
	var row *IssueRow
	s.read(func(st *Store) { row, _ = st.GetIssueByGlobalID(reqCtx(r), id) })
	if row == nil {
		writeJSON(w, http.StatusOK, gqlData(map[string]any{"issue": nil}))
		return
	}
	writeJSON(w, http.StatusOK, gqlData(map[string]any{"issue": linearNode(*row)}))
}

func (s *Server) linearList(w http.ResponseWriter, r *http.Request, _ map[string]interface{}) {
	var rows []IssueRow
	s.read(func(st *Store) { rows, _ = st.ListIssues(reqCtx(r), IssueFilter{State: "all"}) })
	nodes := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		nodes = append(nodes, linearNode(row))
	}
	writeJSON(w, http.StatusOK, gqlData(map[string]any{
		"team": map[string]any{
			"issues": map[string]any{
				"nodes":    nodes,
				"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
			},
		},
	}))
}

func (s *Server) linearCreate(w http.ResponseWriter, r *http.Request, vars map[string]interface{}) {
	title, _ := vars["title"].(string)
	desc, _ := vars["description"].(string)
	var row *IssueRow
	s.write(func(st *Store) { row, _ = st.CreateIssue(reqCtx(r), "LIN", title, desc, nil) })
	writeJSON(w, http.StatusOK, gqlData(map[string]any{
		"issueCreate": map[string]any{
			"success": true,
			"issue": map[string]any{
				"id":         row.GlobalID,
				"identifier": row.GlobalID,
				"url":        "https://linear.example/issue/" + row.GlobalID,
			},
		},
	}))
}

func (s *Server) linearUpdate(w http.ResponseWriter, r *http.Request, vars map[string]interface{}) {
	id, _ := vars["id"].(string)
	input, _ := vars["input"].(map[string]interface{})
	var found bool
	s.write(func(st *Store) {
		row, _ := st.GetIssueByGlobalID(reqCtx(r), id)
		if row == nil {
			return
		}
		found = true
		fields := map[string]any{}
		if t, ok := input["title"].(string); ok {
			fields["title"] = t
		}
		if d, ok := input["description"].(string); ok {
			fields["body"] = d
		}
		if e, ok := input["estimate"].(float64); ok {
			fields["weight"] = int(e)
		}
		st.UpdateIssue(reqCtx(r), row.GlobalID, fields)
	})
	if !found {
		writeJSON(w, http.StatusOK, gqlData(map[string]any{"issueUpdate": map[string]any{"success": false}}))
		return
	}
	writeJSON(w, http.StatusOK, gqlData(map[string]any{"issueUpdate": map[string]any{"success": true}}))
}
