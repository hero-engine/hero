package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/hero-engine/hero/cloud/middleware"
	"github.com/hero-engine/hero/cloud/store"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// orgSession verifies the caller is a member of the org named in the
// request URL path and binds a per-request DB connection with
// app.org_id set so RLS policies apply to every store call below.
//
// On failure it writes the HTTP error and returns ok=false.
// The caller MUST `defer release()` to return the connection to the pool.
func orgSession(db *store.DB, w http.ResponseWriter, r *http.Request) (orgID, userID string, ctx context.Context, release func(), ok bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return "", "", r.Context(), func() {}, false
	}
	orgID = r.PathValue("org_id")
	if orgID == "" {
		writeError(w, http.StatusBadRequest, "org_id required")
		return "", "", r.Context(), func() {}, false
	}
	role, err := db.GetMemberRole(r.Context(), orgID, claims.UserID)
	if err != nil || role == "" {
		writeError(w, http.StatusForbidden, "not a member of this org")
		return "", "", r.Context(), func() {}, false
	}
	ctx, release, err = db.WithOrg(r.Context(), orgID)
	if err != nil {
		log.Printf("acquire org session: %v", err)
		writeError(w, http.StatusInternalServerError, "session error")
		return "", "", r.Context(), func() {}, false
	}
	return orgID, claims.UserID, ctx, release, true
}

// withOrg wraps a handler so that every db call using r.Context() inside
// the handler runs against an RLS-bound connection for the URL's org_id.
// Use this on routes under /api/v1/orgs/{org_id}/... where every DB
// access should be tenant-scoped.
//
// userIDKey lets a handler read claims.UserID from the bound ctx without
// re-extracting from the JWT.
type userIDKeyType struct{}

var userIDKey = userIDKeyType{}

func UserIDFromContext(ctx context.Context) string {
	s, _ := ctx.Value(userIDKey).(string)
	return s
}

func withOrg(db *store.DB, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, userID, ctx, release, ok := orgSession(db, w, r)
		if !ok {
			return
		}
		defer release()
		ctx = context.WithValue(ctx, userIDKey, userID)
		h.ServeHTTP(w, r.WithContext(ctx))
	}
}
