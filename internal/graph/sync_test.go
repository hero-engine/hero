package graph

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeServer is a minimal in-memory implementation of the team-server
// graph endpoints. It's used to round-trip Push/Pull without booting
// the real cloud server. The semantics here mirror what cloud/api/
// will eventually implement.
type fakeServer struct {
	nodes []Node
	edges []Edge
}

// orgURL builds the org-scoped base URL that the real CLI passes as
// SyncClient.ServerURL (see internal/cli/sync_graph.go). Push/Pull append
// the /graph/... route onto this — the prefix must already be present so
// the tests exercise the same shape the bug double-nested.
func orgURL(base, org string) string {
	return base + "/api/v1/orgs/" + org
}

func (f *fakeServer) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/orgs/{org}/graph/push", func(w http.ResponseWriter, r *http.Request) {
		var req PushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		// Naive merge: keep latest by ingested_at per (type, key).
		for _, n := range req.Nodes {
			f.nodes = append(f.nodes, n)
		}
		for _, e := range req.Edges {
			f.edges = append(f.edges, e)
		}
		resp := PushResponse{
			Accepted:   len(req.Nodes) + len(req.Edges),
			ServerTime: time.Now().UTC().Format(time.RFC3339),
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("GET /api/v1/orgs/{org}/graph/pull", func(w http.ResponseWriter, r *http.Request) {
		since := r.URL.Query().Get("since")
		var nodesOut []Node
		for _, n := range f.nodes {
			if since != "" && n.IngestedAt <= since {
				continue
			}
			nodesOut = append(nodesOut, n)
		}
		var edgesOut []Edge
		for _, e := range f.edges {
			if since != "" && e.IngestedAt <= since {
				continue
			}
			edgesOut = append(edgesOut, e)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		_ = json.NewEncoder(w).Encode(PullResponse{
			Nodes:      nodesOut,
			Edges:      edgesOut,
			NextCursor: now,
			ServerTime: now,
		})
	})
	return mux
}

func TestPush_SendsTeamScopeNodesAndEdges(t *testing.T) {
	s := openTestStore(t)
	// One team-scope node, one local-scope node — only team should be sent.
	if _, err := s.UpsertNode(&Node{
		Type: "Feature", Key: "shipping", Domain: "engineering",
		Scope: ScopeTeam, ContentHash: "h-team",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpsertNode(&Node{
		Type: "Note", Key: "private", Domain: "engineering",
		Scope: ScopeLocal, ContentHash: "h-local",
	}); err != nil {
		t.Fatal(err)
	}

	srv := &fakeServer{}
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)

	c := NewSyncClient(orgURL(ts.URL, "test-org"), "test-repo", "test-org")
	resp, err := s.Push(c)
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if resp.Accepted != 1 {
		t.Errorf("Accepted = %d, want 1 (team only, local must NOT leave)", resp.Accepted)
	}
	if len(srv.nodes) != 1 || srv.nodes[0].Key != "shipping" {
		t.Errorf("server got %+v, want only the team-scope node", srv.nodes)
	}
}

func TestPush_IsIdempotentViaSyncState(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.UpsertNode(&Node{
		Type: "Feature", Key: "x", Domain: "engineering", Scope: ScopeTeam, ContentHash: "h",
	}); err != nil {
		t.Fatal(err)
	}
	srv := &fakeServer{}
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)

	c := NewSyncClient(orgURL(ts.URL, "test-org"), "test-repo", "test-org")
	if _, err := s.Push(c); err != nil {
		t.Fatalf("first push: %v", err)
	}
	// Wait for monotonic clock; SQLite RFC3339 has second-precision
	time.Sleep(2 * time.Second)
	if _, err := s.Push(c); err != nil {
		t.Fatalf("second push: %v", err)
	}
	if len(srv.nodes) != 1 {
		t.Errorf("second push sent duplicates: server has %d nodes", len(srv.nodes))
	}
}

func TestPullAndApply_RoundTripsWithEdges(t *testing.T) {
	// Source store with two linked features
	src := openTestStore(t)
	a, _ := src.UpsertNode(&Node{Type: "Feature", Key: "a", Domain: "engineering", Scope: ScopeTeam, ContentHash: "h-a"})
	b, _ := src.UpsertNode(&Node{Type: "Feature", Key: "b", Domain: "engineering", Scope: ScopeTeam, ContentHash: "h-b"})
	if _, err := src.UpsertEdge(&Edge{
		FromID: a, ToID: b, Type: "depends_on", Scope: ScopeTeam,
	}); err != nil {
		t.Fatal(err)
	}

	// Push from src to fake server
	srv := &fakeServer{}
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	srcClient := NewSyncClient(orgURL(ts.URL, "test-org"), "test-repo", "test-org")
	if _, err := src.Push(srcClient); err != nil {
		t.Fatalf("src push: %v", err)
	}

	// Pull into a fresh dst store
	dst := openTestStore(t)
	dstClient := NewSyncClient(orgURL(ts.URL, "test-org"), "test-repo", "test-org")
	resp, nodesApplied, edgesApplied, edgesDeferred, err := dst.Pull(dstClient)
	if err != nil {
		t.Fatalf("dst pull: %v", err)
	}
	if nodesApplied != 2 {
		t.Errorf("nodesApplied = %d, want 2", nodesApplied)
	}
	if edgesApplied != 1 {
		t.Errorf("edgesApplied = %d, want 1", edgesApplied)
	}
	if edgesDeferred != 0 {
		t.Errorf("edgesDeferred = %d, want 0", edgesDeferred)
	}
	if resp.NextCursor == "" {
		t.Error("server should return a cursor")
	}

	// Verify dst has both nodes + edge
	stats, _ := dst.Stats()
	if stats.NodesByType["Feature"] != 2 {
		t.Errorf("dst Feature nodes = %d, want 2", stats.NodesByType["Feature"])
	}
	if stats.EdgesByType["depends_on"] != 1 {
		t.Errorf("dst depends_on edges = %d, want 1", stats.EdgesByType["depends_on"])
	}

	// Pulling again with the saved cursor should fetch nothing.
	_, na2, ea2, _, err := dst.Pull(dstClient)
	if err != nil {
		t.Fatalf("second pull: %v", err)
	}
	if na2 != 0 || ea2 != 0 {
		t.Errorf("second pull applied %d nodes, %d edges (want 0,0)", na2, ea2)
	}
}

func TestPush_ServerErrorReturnsErr(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.UpsertNode(&Node{
		Type: "Feature", Key: "x", Domain: "engineering", Scope: ScopeTeam, ContentHash: "h",
	}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", 500)
	}))
	t.Cleanup(ts.Close)
	c := NewSyncClient(orgURL(ts.URL, "test-org"), "test-repo", "test-org")
	_, err := s.Push(c)
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status: %v", err)
	}
}

// TestSyncEndpoints_DoNotDoubleAPIPrefix is the regression guard for the
// graph-sync URL bug (cloud-cli-verify peer handoff from hero-cloud). The
// CLI passes an org-scoped ServerURL (…/api/v1/orgs/<org>); Push/Pull must
// append only /graph/push and /graph/pull, producing a single /api/v1/
// segment — never the doubled …/orgs/<org>/api/v1/graph/push it shipped.
func TestSyncEndpoints_DoNotDoubleAPIPrefix(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.UpsertNode(&Node{
		Type: "Feature", Key: "x", Domain: "engineering", Scope: ScopeTeam, ContentHash: "h",
	}); err != nil {
		t.Fatal(err)
	}

	var paths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		now := time.Now().UTC().Format(time.RFC3339)
		if r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(PushResponse{Accepted: 1, ServerTime: now})
			return
		}
		_ = json.NewEncoder(w).Encode(PullResponse{NextCursor: now, ServerTime: now})
	}))
	t.Cleanup(ts.Close)

	c := NewSyncClient(orgURL(ts.URL, "test-org"), "test-repo", "test-org")
	if _, err := s.Push(c); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if _, _, _, _, err := s.Pull(c); err != nil {
		t.Fatalf("Pull: %v", err)
	}

	want := []string{
		"/api/v1/orgs/test-org/graph/push",
		"/api/v1/orgs/test-org/graph/pull",
	}
	if len(paths) != len(want) {
		t.Fatalf("hit %d endpoints %v, want %v", len(paths), paths, want)
	}
	for i, p := range paths {
		if p != want[i] {
			t.Errorf("endpoint %d = %q, want %q", i, p, want[i])
		}
		if strings.Count(p, "/api/v1/") != 1 {
			t.Errorf("endpoint %q has a doubled /api/v1/ prefix", p)
		}
	}
}

// drainBody is a small helper for diagnostic test bodies.
func drainBody(r io.Reader) string {
	b, _ := io.ReadAll(r)
	return string(b)
}
