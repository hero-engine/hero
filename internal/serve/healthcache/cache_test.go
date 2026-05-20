package healthcache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeOps is a counting OpsDispatcher used to assert refresh-coalescing
// + the "Start dispatched run-check-json" wiring.
type fakeOps struct {
	mu        sync.Mutex
	starts    int32
	delay     time.Duration
	exit      int
	startErr  error
	waitErr   error
	jobIDPfx  string
	onStart   func()
}

func (f *fakeOps) Start(ctx context.Context, slug, projectRoot, verb string) (string, bool, error) {
	if f.startErr != nil {
		return "", false, f.startErr
	}
	atomic.AddInt32(&f.starts, 1)
	if f.onStart != nil {
		f.onStart()
	}
	return f.jobIDPfx + slug + "-" + verb, true, nil
}

func (f *fakeOps) Wait(ctx context.Context, slug, jobID string) (int, error) {
	if f.waitErr != nil {
		return -1, f.waitErr
	}
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return -1, ctx.Err()
		}
	}
	return f.exit, nil
}

// writeHealthArtifact writes a minimal health.json under <root>/.hero/cache/.
// Returns the temp dir as the projectRoot.
func writeHealthArtifact(t *testing.T, rows []HealthRow, captured time.Time) string {
	t.Helper()
	root := t.TempDir()
	cacheDir := filepath.Join(root, ".hero", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"captured_at": captured.UTC(),
		"rows":        rows,
	})
	if err := os.WriteFile(filepath.Join(cacheDir, "health.json"), body, 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	return root
}

func TestCacheHealthMissReturnsFalse(t *testing.T) {
	c := New(5*time.Minute, Options{})
	if _, ok := c.Health("nope"); ok {
		t.Fatal("expected miss")
	}
}

func TestRefreshHealthPopulatesCache(t *testing.T) {
	rows := []HealthRow{{Name: "stale-specs", Status: "pass", Message: "none"}}
	captured := time.Now().UTC().Truncate(time.Second)
	root := writeHealthArtifact(t, rows, captured)

	ops := &fakeOps{exit: 0}
	c := New(2*time.Minute, Options{Ops: ops})

	got, err := c.RefreshHealth(context.Background(), "alpha", root)
	if err != nil {
		t.Fatalf("RefreshHealth: %v", err)
	}
	if got.Slug != "alpha" {
		t.Errorf("slug = %q want alpha", got.Slug)
	}
	if !got.FromDisk {
		t.Error("expected FromDisk = true")
	}
	if !got.Captured.Equal(captured) {
		t.Errorf("captured = %v want %v", got.Captured, captured)
	}
	if len(got.Rows) != 1 || got.Rows[0].Name != "stale-specs" {
		t.Errorf("rows = %+v", got.Rows)
	}
	cached, ok := c.Health("alpha")
	if !ok {
		t.Fatal("expected cache hit after refresh")
	}
	if len(cached.Rows) != 1 {
		t.Errorf("cached rows = %+v", cached.Rows)
	}
}

func TestRefreshHealthCoalescesConcurrentCallers(t *testing.T) {
	rows := []HealthRow{{Name: "x", Status: "pass"}}
	root := writeHealthArtifact(t, rows, time.Now())

	ops := &fakeOps{exit: 0, delay: 100 * time.Millisecond}
	c := New(time.Minute, Options{Ops: ops})

	const N = 10
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			if _, err := c.RefreshHealth(context.Background(), "alpha", root); err != nil {
				t.Errorf("refresh: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&ops.starts); got != 1 {
		t.Fatalf("expected exactly 1 subprocess start across %d concurrent callers, got %d", N, got)
	}
}

func TestRefreshHealthSubsequentRefreshSpawnsNewSubprocess(t *testing.T) {
	rows := []HealthRow{{Name: "x", Status: "pass"}}
	root := writeHealthArtifact(t, rows, time.Now())

	ops := &fakeOps{exit: 0}
	c := New(time.Minute, Options{Ops: ops})

	if _, err := c.RefreshHealth(context.Background(), "alpha", root); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if _, err := c.RefreshHealth(context.Background(), "alpha", root); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if got := atomic.LoadInt32(&ops.starts); got != 2 {
		t.Fatalf("expected 2 starts across two sequential refreshes, got %d", got)
	}
}

func TestRefreshHealthMissingOps(t *testing.T) {
	c := New(time.Minute, Options{})
	_, err := c.RefreshHealth(context.Background(), "x", "/tmp")
	if err == nil {
		t.Fatal("expected error when ops dispatcher nil")
	}
}

func TestRefreshHealthDispatchError(t *testing.T) {
	ops := &fakeOps{startErr: errors.New("nope")}
	c := New(time.Minute, Options{Ops: ops})
	_, err := c.RefreshHealth(context.Background(), "alpha", "/tmp")
	if err == nil {
		t.Fatal("expected dispatch error")
	}
}

func TestRefreshHealthExitNonZeroWithArtifactSucceeds(t *testing.T) {
	// `hero check` exits non-zero whenever issues are found, but still
	// writes the JSON file. The cache should still pick it up.
	rows := []HealthRow{{Name: "stale-specs", Status: "warn", Message: "3 stale specs"}}
	root := writeHealthArtifact(t, rows, time.Now())

	ops := &fakeOps{exit: 1}
	c := New(time.Minute, Options{Ops: ops})

	got, err := c.RefreshHealth(context.Background(), "alpha", root)
	if err != nil {
		t.Fatalf("RefreshHealth with exit=1 + artifact: %v", err)
	}
	if len(got.Rows) != 1 || got.Rows[0].Status != "warn" {
		t.Errorf("rows = %+v", got.Rows)
	}
}

func TestRefreshHealthExitNonZeroAndNoArtifactErrors(t *testing.T) {
	root := t.TempDir() // no .hero/cache/health.json

	ops := &fakeOps{exit: 2}
	c := New(time.Minute, Options{Ops: ops})

	_, err := c.RefreshHealth(context.Background(), "alpha", root)
	if err == nil {
		t.Fatal("expected error when subprocess fails and no artifact written")
	}
}

func TestPeerCacheKeyIsolation(t *testing.T) {
	// alpha and beta both have an aliased peer named "core" — the
	// cache must keep them independent.
	called := make(map[string]int)
	var mu sync.Mutex
	prober := func(_ context.Context, slug, alias, _ string) (PeerResult, error) {
		mu.Lock()
		called[slug+":"+alias]++
		mu.Unlock()
		return PeerResult{Reachable: slug == "alpha"}, nil
	}

	c := New(time.Minute, Options{PeerProber: prober})

	alpha, err := c.ProbePeer(context.Background(), "alpha", "core", "/tmp/alpha")
	if err != nil {
		t.Fatalf("probe alpha: %v", err)
	}
	if !alpha.Reachable {
		t.Error("expected alpha:core reachable")
	}
	beta, err := c.ProbePeer(context.Background(), "beta", "core", "/tmp/beta")
	if err != nil {
		t.Fatalf("probe beta: %v", err)
	}
	if beta.Reachable {
		t.Error("expected beta:core unreachable (independent from alpha)")
	}

	if gotAlpha, _ := c.Peer("alpha", "core"); !gotAlpha.Reachable {
		t.Error("alpha:core lookup wrong after probe")
	}
	if gotBeta, _ := c.Peer("beta", "core"); gotBeta.Reachable {
		t.Error("beta:core lookup wrong after probe")
	}
	if called["alpha:core"] != 1 || called["beta:core"] != 1 {
		t.Errorf("probe call counts = %+v", called)
	}
}

func TestPeerCacheRoundTrip(t *testing.T) {
	root := t.TempDir()
	// Create the peer's .hero/peer-manifest.yaml so the default prober reports reachable.
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hero", "peer-manifest.yaml"), []byte("name: peer\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	c := New(time.Minute, Options{})
	result, err := c.ProbePeer(context.Background(), "alpha", "core", root)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !result.Reachable {
		t.Errorf("expected reachable, got %+v", result)
	}
	got, ok := c.Peer("alpha", "core")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !got.Reachable {
		t.Errorf("cached result not reachable: %+v", got)
	}
}

func TestPeerCacheMissingPath(t *testing.T) {
	c := New(time.Minute, Options{})
	result, err := c.ProbePeer(context.Background(), "alpha", "ghost", "")
	if err != nil {
		t.Fatalf("probe with empty path: %v", err)
	}
	if result.Reachable {
		t.Error("expected unreachable for empty peerPath")
	}
	if result.LastError == "" {
		t.Error("expected LastError set")
	}
}

func TestTTLDefaultsTo5Minutes(t *testing.T) {
	c := New(0, Options{})
	if c.TTL() != 5*time.Minute {
		t.Errorf("TTL = %v want 5m", c.TTL())
	}
}

func TestStaleResultStillReturnedFromCache(t *testing.T) {
	// Stale entries are still returned — the renderer decides whether
	// to show a "stale" marker. The cache itself does not evict on TTL.
	rows := []HealthRow{{Name: "x", Status: "pass"}}
	root := writeHealthArtifact(t, rows, time.Now())

	ops := &fakeOps{exit: 0}
	now := time.Now()
	clk := func() time.Time { return now }
	c := New(time.Second, Options{Ops: ops, Now: clk})

	if _, err := c.RefreshHealth(context.Background(), "alpha", root); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// Advance the clock past the TTL.
	now = now.Add(time.Hour)

	got, ok := c.Health("alpha")
	if !ok {
		t.Fatal("expected hit even when stale")
	}
	// Renderer can compute staleness from Timestamp+TTL.
	age := clk().Sub(got.Timestamp)
	if age <= got.TTL {
		t.Errorf("expected age %v > ttl %v", age, got.TTL)
	}
}
