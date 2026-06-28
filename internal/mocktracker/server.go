// Package mocktracker is a single-binary, offline HTTP fake that speaks
// the subset of the GitHub, Jira, Linear, and GitLab APIs hero's tracker
// adapters actually call. It is backed by an in-memory SQLite DB seeded
// by sprout-Go, so one Acme fixture backs all four tracker modes at once.
//
// It lives under internal/ (driven by cmd/mock-tracker-server) and is
// never imported by the production hero binary (AC-12).
package mocktracker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Options configures a Server.
type Options struct {
	SeedDir      string // --seed <dir>; empty → embedded default fixture
	SingleMode   string // --single-mode github|jira|linear|gitlab; empty → multi-mode
	RequireToken string // --require-token; empty → any non-empty token accepted
	LogRequests  bool   // --log-requests; one JSON line per request on stderr
}

// Server is the in-memory multi-tracker fake. One RWMutex guards the
// store: admin writes take the write lock, reads take the read lock
// (contention is a non-issue at fixture scale).
type Server struct {
	store *Store
	rl    *rateLimiter
	mu    sync.RWMutex
	opts  Options
}

// NewServer builds and seeds a Server. The DB is created and seeded fresh
// (determinism is the point); reset mid-run via /__admin/reset.
func NewServer(ctx context.Context, opts Options) (*Server, error) {
	store, err := NewStore(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := store.Seed(ctx, opts.SeedDir, false); err != nil {
		store.Close()
		return nil, err
	}
	return &Server{store: store, rl: newRateLimiter(), opts: opts}, nil
}

// Close releases the store.
func (s *Server) Close() error { return s.store.Close() }

// statusRecorder captures the response status for request logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// ServeHTTP authenticates, applies the 429 injector, routes by mode
// prefix, and logs.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sr := &statusRecorder{ResponseWriter: w, status: 200}
	route := s.route(sr, r)
	if s.opts.LogRequests {
		logLine := map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": sr.status,
			"route":  route,
		}
		b, _ := json.Marshal(logLine)
		fmt.Fprintln(os.Stderr, string(b))
	}
}

// route does the actual dispatch and returns a label for logging.
func (s *Server) route(w *statusRecorder, r *http.Request) string {
	// Auth: any non-empty token, unless --require-token is set.
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, "missing or invalid token")
		return "auth"
	}

	// 429 injector — keyed on the full request path.
	if retryAfter, limited := s.rl.throttle(r.URL.Path); limited {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		writeError(w, http.StatusTooManyRequests, "rate limited (injected)")
		return "429-injected"
	}

	path := r.URL.Path
	if strings.HasPrefix(path, "/__admin/") || path == "/__admin" {
		s.handleAdmin(w, r, strings.TrimPrefix(path, "/__admin"))
		return "admin"
	}

	mode, sub := s.resolveMode(path)
	switch mode {
	case "github":
		s.handleGitHub(w, r, sub)
	case "jira":
		s.handleJira(w, r, sub)
	case "linear":
		s.handleLinear(w, r, sub)
	case "gitlab":
		s.handleGitLab(w, r, sub)
	default:
		writeError(w, http.StatusNotFound, "unknown tracker mode")
	}
	return mode
}

// resolveMode determines the tracker mode and the remaining sub-path. In
// single-mode the whole path is the sub-path; otherwise the first
// segment is the mode prefix.
func (s *Server) resolveMode(path string) (mode, sub string) {
	if s.opts.SingleMode != "" {
		return s.opts.SingleMode, path
	}
	trimmed := strings.TrimPrefix(path, "/")
	idx := strings.IndexByte(trimmed, '/')
	if idx < 0 {
		return trimmed, "/"
	}
	return trimmed[:idx], trimmed[idx:]
}

// authOK enforces the token policy. The mock accepts a token in either
// Authorization (GitHub/Linear/Jira) or PRIVATE-TOKEN (GitLab). With
// --require-token the presented token must match.
func (s *Server) authOK(r *http.Request) bool {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	priv := strings.TrimSpace(r.Header.Get("PRIVATE-TOKEN"))
	if s.opts.RequireToken == "" {
		return auth != "" || priv != ""
	}
	want := s.opts.RequireToken
	// Strip common scheme prefixes from Authorization.
	bare := auth
	for _, p := range []string{"Bearer ", "bearer ", "token ", "Token "} {
		bare = strings.TrimPrefix(bare, p)
	}
	return bare == want || priv == want
}

// --- shared response helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

// readLock / writeLock wrap store access so handlers don't each manage
// the mutex.
func (s *Server) read(fn func(*Store)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s.store)
}

func (s *Server) write(fn func(*Store)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.store)
}

// ctx is a convenience for handlers.
func reqCtx(r *http.Request) context.Context { return r.Context() }
