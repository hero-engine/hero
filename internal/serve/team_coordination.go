package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RegisterTeamCoordinationAPI adds claim enforcement and feed endpoints.
func RegisterTeamCoordinationAPI(mux *http.ServeMux, jq *JobQueue, authMiddleware func(http.Handler) http.Handler) {
	wrap := func(h http.HandlerFunc) http.Handler {
		if authMiddleware != nil {
			return authMiddleware(h)
		}
		return h
	}

	// Server-enforced claims
	mux.Handle("/api/claims", wrap(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleListClaims(w, r, jq)
		case "POST":
			handleCreateClaim(w, r, jq)
		case "DELETE":
			handleReleaseClaim(w, r, jq)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Centralized event feed
	mux.Handle("/api/feed", wrap(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleListFeed(w, r, jq)
		case "POST":
			handlePostFeed(w, r, jq)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
}

// --- Claims ---

func handleListClaims(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	rows, err := jq.db.Query(`
		SELECT slug, user_id, claimed_at FROM claims
		ORDER BY claimed_at DESC`)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var claims []map[string]string
	for rows.Next() {
		var slug, userID, claimedAt string
		rows.Scan(&slug, &userID, &claimedAt)
		claims = append(claims, map[string]string{
			"slug": slug, "user_id": userID, "claimed_at": claimedAt,
		})
	}
	if claims == nil {
		claims = []map[string]string{}
	}
	jsonResponse(w, claims)
}

func handleCreateClaim(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	var req struct {
		Slug   string `json:"slug"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		req.UserID = r.Header.Get("X-Hero-User")
	}

	// Check for existing claim
	var existing string
	err := jq.db.QueryRow("SELECT user_id FROM claims WHERE slug = ?", req.Slug).Scan(&existing)
	if err == nil {
		if existing == req.UserID {
			jsonResponse(w, map[string]string{"status": "already_claimed", "slug": req.Slug, "user_id": existing})
			return
		}
		jsonError(w, fmt.Sprintf("spec %s is already claimed by %s", req.Slug, existing), http.StatusConflict)
		return
	}

	_, err = jq.db.Exec(`INSERT INTO claims (slug, user_id, claimed_at) VALUES (?, ?, ?)`,
		req.Slug, req.UserID, time.Now().Format(time.RFC3339))
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, map[string]string{"status": "claimed", "slug": req.Slug, "user_id": req.UserID})
}

func handleReleaseClaim(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	var req struct {
		Slug string `json:"slug"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	result, err := jq.db.Exec("DELETE FROM claims WHERE slug = ?", req.Slug)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonError(w, "claim not found", http.StatusNotFound)
		return
	}
	jsonResponse(w, map[string]string{"status": "released", "slug": req.Slug})
}

// --- Feed ---

func handleListFeed(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	limit := 50
	rows, err := jq.db.Query(`
		SELECT id, user_id, event_type, spec_slug, message, created_at
		FROM feed ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []map[string]string
	for rows.Next() {
		var id, userID, eventType, specSlug, message, createdAt string
		rows.Scan(&id, &userID, &eventType, &specSlug, &message, &createdAt)
		events = append(events, map[string]string{
			"id": id, "user_id": userID, "event_type": eventType,
			"spec_slug": specSlug, "message": message, "created_at": createdAt,
		})
	}
	if events == nil {
		events = []map[string]string{}
	}
	jsonResponse(w, events)
}

func handlePostFeed(w http.ResponseWriter, r *http.Request, jq *JobQueue) {
	var req struct {
		EventType string `json:"event_type"`
		SpecSlug  string `json:"spec_slug"`
		Message   string `json:"message"`
		UserID    string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.EventType == "" {
		jsonError(w, "event_type is required", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		req.UserID = r.Header.Get("X-Hero-User")
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	_, err := jq.db.Exec(`
		INSERT INTO feed (id, user_id, event_type, spec_slug, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, req.UserID, req.EventType, req.SpecSlug, req.Message,
		time.Now().Format(time.RFC3339))
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, map[string]string{"id": id, "status": "posted"})
}
