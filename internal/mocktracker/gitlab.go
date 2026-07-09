package mocktracker

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// handleGitLab serves the subset of the GitLab REST v4 API that
// internal/tracker/gitlab.go (and its sprint loader) call. Routes (sub
// is the path after the /gitlab prefix):
//
//	GET  /api/v4/projects/:id/issues                  (Link pagination)
//	GET/POST/PUT /api/v4/projects/:id/issues/:iid
//	POST /api/v4/projects/:id/issues/:iid/notes
//	GET  /api/v4/projects/:id/milestones
//	GET  /api/v4/projects/:id/iterations
//	GET  /api/v4/groups/:id/epics
//
// GitLab project IDs carry %2F-encoded slashes ("group%2Fproj"), so this
// routes on the ESCAPED path to keep the project as one segment.
func (s *Server) handleGitLab(w http.ResponseWriter, r *http.Request, _ string) {
	sub := s.gitlabSub(r)
	parts := splitPath(sub)

	// /api/v4/groups/:id/{epics,iterations}
	if len(parts) >= 5 && parts[0] == "api" && parts[1] == "v4" && parts[2] == "groups" && r.Method == http.MethodGet {
		switch parts[4] {
		case "epics":
			s.gitlabListEpics(w, r)
			return
		case "iterations":
			s.gitlabListIterations(w, r)
			return
		}
	}

	// /api/v4/projects/:id/...
	if len(parts) >= 5 && parts[0] == "api" && parts[1] == "v4" && parts[2] == "projects" {
		resource := parts[4]
		switch resource {
		case "issues":
			switch {
			case len(parts) == 5:
				switch r.Method {
				case http.MethodGet:
					s.gitlabListIssues(w, r)
				case http.MethodPost:
					s.gitlabCreateIssue(w, r)
				default:
					writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				}
				return
			case len(parts) == 6:
				iid := parts[5]
				switch r.Method {
				case http.MethodGet:
					s.gitlabGetIssue(w, r, iid)
				case http.MethodPut:
					s.gitlabPutIssue(w, r, iid)
				default:
					writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				}
				return
			case len(parts) == 7 && parts[6] == "notes" && r.Method == http.MethodPost:
				writeJSON(w, http.StatusCreated, map[string]any{"id": 1})
				return
			}
		case "milestones":
			if r.Method == http.MethodGet {
				s.gitlabListMilestones(w, r)
				return
			}
		case "iterations":
			if r.Method == http.MethodGet {
				s.gitlabListIterations(w, r)
				return
			}
		}
	}

	writeError(w, http.StatusNotFound, "gitlab endpoint not implemented: "+sub)
}

// gitlabSub returns the request's escaped path with the /gitlab mode
// prefix stripped (no-op in single-mode).
func (s *Server) gitlabSub(r *http.Request) string {
	p := r.URL.EscapedPath()
	if s.opts.SingleMode == "gitlab" {
		return p
	}
	return strings.TrimPrefix(p, "/gitlab")
}

type containerMaps struct {
	epics map[string]EpicRow
	miles map[string]MilestoneRow
	iters map[string]IterationRow
}

func (s *Server) buildContainerMaps(r *http.Request, st *Store) containerMaps {
	cm := containerMaps{
		epics: map[string]EpicRow{},
		miles: map[string]MilestoneRow{},
		iters: map[string]IterationRow{},
	}
	epics, _ := st.ListEpics(reqCtx(r))
	for _, e := range epics {
		cm.epics[e.GlobalID] = e
	}
	miles, _ := st.ListMilestones(reqCtx(r))
	for _, m := range miles {
		cm.miles[m.GlobalID] = m
	}
	iters, _ := st.ListIterations(reqCtx(r))
	for _, it := range iters {
		cm.iters[it.GlobalID] = it
	}
	return cm
}

func gitlabState(status string) string {
	if strings.EqualFold(status, "closed") {
		return "closed"
	}
	return "opened"
}

// gitlabIssueType projects the neutral type onto GitLab's issue_type.
func gitlabIssueType(t string) string {
	switch strings.ToLower(t) {
	case "bug":
		return "incident"
	case "task":
		return "task"
	default:
		return "issue"
	}
}

func gitlabIssue(r IssueRow, cm containerMaps) map[string]any {
	iid, _ := strconv.Atoi(r.IID)
	out := map[string]any{
		"id":          iid + 100000, // global id, distinct from iid
		"iid":         iid,
		"title":       r.Title,
		"description": r.Body,
		"state":       gitlabState(r.Status),
		"web_url":     "https://gitlab.example/-/issues/" + r.IID,
		"labels":      r.Labels,
		"issue_type":  gitlabIssueType(r.Type),
	}
	if r.Weight.Valid {
		out["weight"] = r.Weight.Int64
	}
	if r.Assignee != "" {
		out["assignee"] = map[string]string{"username": r.Assignee}
	}
	if e, ok := cm.epics[r.EpicID]; ok {
		out["epic"] = map[string]any{"id": iidIntOf(e.GlobalID) + 200000, "iid": iidIntOf(e.GlobalID), "title": e.Title}
	}
	if m, ok := cm.miles[r.MilestoneID]; ok {
		out["milestone"] = map[string]any{"id": iidIntOf(m.GlobalID), "title": m.Title}
	}
	if it, ok := cm.iters[r.IterationID]; ok {
		out["iteration"] = map[string]any{"id": iidIntOf(it.GlobalID), "title": it.Name}
	}
	return out
}

func iidIntOf(globalID string) int {
	n, _ := strconv.Atoi(iidOf(globalID))
	return n
}

func (s *Server) gitlabListIssues(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := IssueFilter{State: q.Get("state")}
	if lbl := q.Get("labels"); lbl != "" {
		f.Labels = strings.Split(lbl, ",")
	}
	if a := q.Get("assignee_username"); a != "" {
		f.Assignee = a
	}
	if q.Get("assignee_id") == "None" {
		f.Assignee = "none"
	}
	if it := q.Get("iteration_id"); it != "" {
		// map iteration iid back to a global_id by scanning containers
		f.IterationID = s.iterationGlobalByIID(r, it)
	}
	if sr := q.Get("search"); sr != "" {
		f.Search = sr
	}

	var rows []IssueRow
	var cm containerMaps
	s.read(func(st *Store) {
		rows, _ = st.ListIssues(reqCtx(r), f)
		cm = s.buildContainerMaps(r, st)
	})

	perPage, page := pageParams(q)
	start, end, hasNext := pageSlice(len(rows), perPage, page)
	if hasNext {
		setLinkHeader(w, r, page, perPage, len(rows))
	}
	out := make([]map[string]any, 0, end-start)
	for _, row := range rows[start:end] {
		out = append(out, gitlabIssue(row, cm))
	}
	writeJSON(w, http.StatusOK, out)
}

// iterationGlobalByIID resolves an iteration IID query param back to its
// global_id so issue filtering can match the stored iteration_id.
func (s *Server) iterationGlobalByIID(r *http.Request, iid string) string {
	var found string
	s.read(func(st *Store) {
		iters, _ := st.ListIterations(reqCtx(r))
		for _, it := range iters {
			if iidOf(it.GlobalID) == iid {
				found = it.GlobalID
				return
			}
		}
	})
	return found
}

func (s *Server) gitlabGetIssue(w http.ResponseWriter, r *http.Request, iid string) {
	var row *IssueRow
	var cm containerMaps
	s.read(func(st *Store) {
		row, _ = st.GetIssueByIID(reqCtx(r), iid)
		cm = s.buildContainerMaps(r, st)
	})
	if row == nil {
		writeError(w, http.StatusNotFound, "404 Issue Not Found")
		return
	}
	writeJSON(w, http.StatusOK, gitlabIssue(*row, cm))
}

func (s *Server) gitlabCreateIssue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Labels      string `json:"labels"` // comma-separated
	}
	json.NewDecoder(r.Body).Decode(&body)
	var labels []string
	if body.Labels != "" {
		labels = strings.Split(body.Labels, ",")
	}
	var row *IssueRow
	var cm containerMaps
	s.write(func(st *Store) {
		row, _ = st.CreateIssue(reqCtx(r), "GL", body.Title, body.Description, labels)
		cm = s.buildContainerMaps(r, st)
	})
	if row == nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, gitlabIssue(*row, cm))
}

func (s *Server) gitlabPutIssue(w http.ResponseWriter, r *http.Request, iid string) {
	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Labels      *string `json:"labels"` // comma-separated (full set)
		Weight      *int    `json:"weight"`
		StateEvent  *string `json:"state_event"` // close|reopen
	}
	json.NewDecoder(r.Body).Decode(&body)

	var row *IssueRow
	var cm containerMaps
	s.write(func(st *Store) {
		row, _ = st.GetIssueByIID(reqCtx(r), iid)
		if row == nil {
			return
		}
		fields := map[string]any{}
		if body.Title != nil {
			fields["title"] = *body.Title
		}
		if body.Description != nil {
			fields["body"] = *body.Description
		}
		if body.Weight != nil {
			fields["weight"] = *body.Weight
		}
		if body.StateEvent != nil {
			switch *body.StateEvent {
			case "close":
				fields["status"] = "closed"
			case "reopen":
				fields["status"] = "open"
			}
		}
		st.UpdateIssue(reqCtx(r), row.GlobalID, fields)
		if body.Labels != nil {
			st.SetLabels(reqCtx(r), row.GlobalID, strings.Split(*body.Labels, ","))
		}
		row, _ = st.GetIssueByGlobalID(reqCtx(r), row.GlobalID)
		cm = s.buildContainerMaps(r, st)
	})
	if row == nil {
		writeError(w, http.StatusNotFound, "404 Issue Not Found")
		return
	}
	writeJSON(w, http.StatusOK, gitlabIssue(*row, cm))
}

func (s *Server) gitlabListEpics(w http.ResponseWriter, r *http.Request) {
	var epics []EpicRow
	s.read(func(st *Store) { epics, _ = st.ListEpics(reqCtx(r)) })
	out := make([]map[string]any, 0, len(epics))
	for _, e := range epics {
		m := map[string]any{
			"id":      iidIntOf(e.GlobalID) + 200000,
			"iid":     iidIntOf(e.GlobalID),
			"title":   e.Title,
			"state":   "opened",
			"web_url": "https://gitlab.example/-/epics/" + iidOf(e.GlobalID),
		}
		if e.ParentID != "" {
			m["parent_id"] = iidIntOf(e.ParentID) + 200000
		}
		out = append(out, m)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) gitlabListMilestones(w http.ResponseWriter, r *http.Request) {
	var miles []MilestoneRow
	s.read(func(st *Store) { miles, _ = st.ListMilestones(reqCtx(r)) })
	out := make([]map[string]any, 0, len(miles))
	for _, m := range miles {
		out = append(out, map[string]any{
			"id":       iidIntOf(m.GlobalID),
			"title":    m.Title,
			"due_date": m.Due,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) gitlabListIterations(w http.ResponseWriter, r *http.Request) {
	var iters []IterationRow
	s.read(func(st *Store) { iters, _ = st.ListIterations(reqCtx(r)) })
	out := make([]map[string]any, 0, len(iters))
	for _, it := range iters {
		out = append(out, map[string]any{
			"id":         iidIntOf(it.GlobalID),
			"iid":        iidIntOf(it.GlobalID),
			"title":      it.Name,
			"state":      2, // current
			"start_date": it.Start,
			"due_date":   it.End,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
