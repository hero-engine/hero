package mocktracker

import (
	"encoding/json"
	"net/http"
)

// handleAdmin serves the /__admin control plane (sub is the path after
// /__admin):
//
//	POST /__admin/inject-429   {"path_glob","retry_after_seconds","count"}
//	POST /__admin/mutate       {"id","field","value"}   (out-of-band drift)
//	POST /__admin/rotate-ids   {"id","new_id"}          (rotate IIDs)
//	POST /__admin/reset        re-seed + clear 429 rules
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request, sub string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "admin endpoints are POST")
		return
	}
	switch sub {
	case "/inject-429":
		s.adminInject429(w, r)
	case "/mutate":
		s.adminMutate(w, r)
	case "/rotate-ids":
		s.adminRotateIDs(w, r)
	case "/reset":
		s.adminReset(w, r)
	default:
		writeError(w, http.StatusNotFound, "unknown admin endpoint: "+sub)
	}
}

func (s *Server) adminInject429(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PathGlob          string `json:"path_glob"`
		RetryAfterSeconds int    `json:"retry_after_seconds"`
		Count             int    `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PathGlob == "" {
		writeError(w, http.StatusBadRequest, "path_glob is required")
		return
	}
	s.rl.inject(body.PathGlob, body.RetryAfterSeconds, body.Count)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) adminMutate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID    string `json:"id"`
		Field string `json:"field"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == "" || body.Field == "" {
		writeError(w, http.StatusBadRequest, "id and field are required")
		return
	}
	var err error
	s.write(func(st *Store) { err = st.Mutate(reqCtx(r), body.ID, body.Field, body.Value) })
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) adminRotateIDs(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID    string `json:"id"`     // global id (stable)
		NewID string `json:"new_id"` // optional new IID; empty → derived perturbation
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	var err error
	s.write(func(st *Store) { err = st.RotateIID(reqCtx(r), body.ID, body.NewID) })
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) adminReset(w http.ResponseWriter, r *http.Request) {
	s.rl.reset()
	var err error
	s.write(func(st *Store) { _, err = st.Seed(reqCtx(r), s.opts.SeedDir, true) })
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
