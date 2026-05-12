package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RegisterJobsAPI adds team server job endpoints to a mux.
func RegisterJobsAPI(mux *http.ServeMux, jq *JobQueue, authMiddleware func(http.Handler) http.Handler) {
	wrap := func(h http.HandlerFunc) http.Handler {
		if authMiddleware != nil {
			return authMiddleware(h)
		}
		return h
	}

	mux.Handle("/api/jobs", wrap(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleListJobs(w, r, jq)
		case "POST":
			handleSubmitJob(w, r, jq)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.Handle("/api/jobs/", wrap(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/jobs/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "job ID required", http.StatusBadRequest)
			return
		}

		jobID := parts[0]
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}

		switch {
		case action == "approve" && r.Method == "POST":
			handleApproveJob(w, r, jq, jobID)
		case action == "reject" && r.Method == "POST":
			handleRejectJob(w, r, jq, jobID)
		case action == "cancel" && r.Method == "POST":
			handleCancelJob(w, r, jq, jobID)
		case action == "" && r.Method == "GET":
			handleGetJob(w, r, jq, jobID)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))

	mux.Handle("/api/team/status", wrap(func(w http.ResponseWriter, r *http.Request) {
		handleTeamStatus(w, r, jq)
	}))

	mux.Handle("/api/team/usage", wrap(func(w http.ResponseWriter, r *http.Request) {
		handleTeamUsage(w, r, jq)
	}))
}

func handleListJobs(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	status := r.URL.Query().Get("status")
	jobs, err := jq.List(status, 50)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if jobs == nil {
		jobs = []*Job{}
	}
	jsonResponse(w, jobs)
}

func handleSubmitJob(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	var req struct {
		Command string  `json:"command"`
		Args    string  `json:"args"`
		Provider string `json:"provider"`
		Model   string  `json:"model"`
		Budget  float64 `json:"budget"`
		MaxTurns int    `json:"max_turns"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Command == "" {
		jsonError(w, "command is required", http.StatusBadRequest)
		return
	}

	job := &Job{
		ID:       fmt.Sprintf("%d", timeNowUnixNano()),
		Command:  req.Command,
		Args:     req.Args,
		Provider: req.Provider,
		Model:    req.Model,
		Budget:   req.Budget,
		MaxTurns: req.MaxTurns,
		Status:   JobQueued,
	}
	if job.Provider == "" {
		job.Provider = "anthropic"
	}
	if job.MaxTurns == 0 {
		job.MaxTurns = 100
	}

	// Extract user from auth context if available
	if user := r.Header.Get("X-Hero-User"); user != "" {
		job.SubmittedBy = user
	}

	if err := jq.Submit(job); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, job)
}

func handleGetJob(w http.ResponseWriter, r *http.Request, jq *JobQueue, id string) {
	job, err := jq.Get(id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonResponse(w, job)
}

func handleApproveJob(w http.ResponseWriter, r *http.Request, jq *JobQueue, id string) {
	if err := jq.Approve(id); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(w, map[string]string{"status": "approved", "id": id})
}

func handleRejectJob(w http.ResponseWriter, r *http.Request, jq *JobQueue, id string) {
	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := jq.Reject(id, req.Reason); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(w, map[string]string{"status": "rejected", "id": id})
}

func handleCancelJob(w http.ResponseWriter, r *http.Request, jq *JobQueue, id string) {
	if err := jq.Cancel(id); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(w, map[string]string{"status": "cancelled", "id": id})
}

func handleTeamStatus(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	sessions, _ := jq.ActiveSessions()
	running, _ := jq.List("running", 50)
	queued, _ := jq.List("queued", 50)
	awaiting, _ := jq.List("awaiting_approval", 50)

	if sessions == nil {
		sessions = []map[string]string{}
	}

	jsonResponse(w, map[string]interface{}{
		"sessions":          sessions,
		"running_jobs":      running,
		"queued_jobs":       queued,
		"awaiting_approval": awaiting,
	})
}

func handleTeamUsage(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	since := r.URL.Query().Get("since")
	var sinceTime time.Time
	if since != "" {
		sinceTime, _ = time.Parse("2006-01-02", since)
	}
	if sinceTime.IsZero() {
		sinceTime = time.Now().AddDate(0, 0, -7)
	}

	summaries, err := jq.UsageSummary(sinceTime)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if summaries == nil {
		summaries = []map[string]interface{}{}
	}
	jsonResponse(w, summaries)
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// timeNowUnixNano is a var for testability
var timeNowUnixNano = func() int64 { return time.Now().UnixNano() }
