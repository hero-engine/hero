package mocktracker

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// handleGitHub serves the subset of the GitHub Issues API that
// internal/tracker/github.go calls. Routes (sub is the path after the
// /github prefix):
//
//	GET  /repos/:owner/:repo/issues
//	GET  /repos/:owner/:repo/issues/:number
//	POST /repos/:owner/:repo/issues
//	PATCH /repos/:owner/:repo/issues/:number
//	POST /repos/:owner/:repo/issues/:number/comments
//	GET  /search/issues?q=...
func (s *Server) handleGitHub(w http.ResponseWriter, r *http.Request, sub string) {
	parts := splitPath(sub)

	// /search/issues
	if len(parts) == 2 && parts[0] == "search" && parts[1] == "issues" {
		s.githubSearch(w, r)
		return
	}

	// /repos/:owner/:repo/issues...  (repos, owner, repo, issues → len 4)
	if len(parts) >= 4 && parts[0] == "repos" && parts[3] == "issues" {
		switch {
		case len(parts) == 4: // collection
			switch r.Method {
			case http.MethodGet:
				s.githubListIssues(w, r)
			case http.MethodPost:
				s.githubCreateIssue(w, r)
			default:
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		case len(parts) == 5: // single issue
			num := parts[4]
			switch r.Method {
			case http.MethodGet:
				s.githubGetIssue(w, r, num)
			case http.MethodPatch:
				s.githubPatchIssue(w, r, num)
			default:
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		case len(parts) == 6 && parts[5] == "comments" && r.Method == http.MethodPost:
			writeJSON(w, http.StatusCreated, map[string]any{"id": 1})
			return
		}
	}

	writeError(w, http.StatusNotFound, "github endpoint not implemented: "+sub)
}

// githubIssue is the wire projection of an IssueRow.
func githubIssue(r IssueRow) map[string]any {
	num, _ := strconv.Atoi(r.IID)
	labels := make([]map[string]string, 0, len(r.Labels))
	for _, l := range r.Labels {
		labels = append(labels, map[string]string{"name": l})
	}
	out := map[string]any{
		"number":   num,
		"title":    r.Title,
		"body":     r.Body,
		"state":    githubState(r.Status),
		"html_url": "https://github.example/issues/" + r.IID,
		"labels":   labels,
	}
	if r.Assignee != "" {
		out["assignee"] = map[string]string{"login": r.Assignee}
	} else {
		out["assignee"] = nil
	}
	return out
}

func githubState(status string) string {
	if strings.EqualFold(status, "closed") {
		return "closed"
	}
	return "open"
}

func (s *Server) githubListIssues(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := IssueFilter{State: q.Get("state")}
	if lbl := q.Get("labels"); lbl != "" {
		f.Labels = strings.Split(lbl, ",")
	}
	if a := q.Get("assignee"); a != "" {
		f.Assignee = a
	}

	var rows []IssueRow
	s.read(func(st *Store) { rows, _ = st.ListIssues(reqCtx(r), f) })

	perPage, page := pageParams(q)
	start, end, hasNext := pageSlice(len(rows), perPage, page)
	if hasNext {
		setLinkHeader(w, r, page, perPage, len(rows))
	}
	out := make([]map[string]any, 0, end-start)
	for _, row := range rows[start:end] {
		out = append(out, githubIssue(row))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) githubGetIssue(w http.ResponseWriter, r *http.Request, num string) {
	var row *IssueRow
	s.read(func(st *Store) { row, _ = st.GetIssueByIID(reqCtx(r), num) })
	if row == nil {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	writeJSON(w, http.StatusOK, githubIssue(*row))
}

func (s *Server) githubCreateIssue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title  string   `json:"title"`
		Body   string   `json:"body"`
		Labels []string `json:"labels"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	var row *IssueRow
	s.write(func(st *Store) { row, _ = st.CreateIssue(reqCtx(r), "GH", body.Title, body.Body, body.Labels) })
	if row == nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, githubIssue(*row))
}

func (s *Server) githubPatchIssue(w http.ResponseWriter, r *http.Request, num string) {
	var body struct {
		Title  *string   `json:"title"`
		Body   *string   `json:"body"`
		State  *string   `json:"state"`
		Labels *[]string `json:"labels"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	var row *IssueRow
	s.write(func(st *Store) {
		row, _ = st.GetIssueByIID(reqCtx(r), num)
		if row == nil {
			return
		}
		fields := map[string]any{}
		if body.Title != nil {
			fields["title"] = *body.Title
		}
		if body.Body != nil {
			fields["body"] = *body.Body
		}
		if body.State != nil {
			fields["status"] = *body.State // open|closed map directly
		}
		st.UpdateIssue(reqCtx(r), row.GlobalID, fields)
		if body.Labels != nil {
			st.SetLabels(reqCtx(r), row.GlobalID, *body.Labels)
		}
		row, _ = st.GetIssueByGlobalID(reqCtx(r), row.GlobalID)
	})
	if row == nil {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	writeJSON(w, http.StatusOK, githubIssue(*row))
}

// githubSearch implements the minimal /search/issues used by the adapter:
// a free-text match over titles, scoped to whatever `repo:` qualifier is
// present (the qualifier is accepted and ignored — single-repo fixture).
func (s *Server) githubSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	// Drop GitHub search qualifiers (repo:foo/bar, is:open, ...).
	var terms []string
	for _, tok := range strings.Fields(q) {
		if strings.Contains(tok, ":") {
			continue
		}
		terms = append(terms, tok)
	}
	f := IssueFilter{State: "all", Search: strings.Join(terms, " ")}

	var rows []IssueRow
	s.read(func(st *Store) { rows, _ = st.ListIssues(reqCtx(r), f) })

	perPage, _ := pageParams(r.URL.Query())
	if len(rows) > perPage {
		rows = rows[:perPage]
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, githubIssue(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"total_count": len(items), "items": items})
}

// splitPath splits a URL sub-path into non-empty segments.
func splitPath(p string) []string {
	var out []string
	for _, seg := range strings.Split(p, "/") {
		if seg != "" {
			out = append(out, seg)
		}
	}
	return out
}
