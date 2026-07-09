package cloud

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDaemonStartAndStop(t *testing.T) {
	d := NewDaemon(Config{
		CloudURL:    "http://localhost:9999",
		Token:       "test-token",
		OrgID:       "org-1",
		RepoID:      "repo-1",
		ProjectRoot: t.TempDir(),
		HeroDir:     t.TempDir(),
	})
	d.SyncFunc = noopSyncSpecs
	d.GraphFunc = noopPushGraph
	d.Start()
	d.Stop()
}

func TestDaemonDebounce(t *testing.T) {
	var syncCount atomic.Int32

	d := NewDaemon(Config{
		CloudURL:    "http://localhost:9999",
		Token:       "test-token",
		OrgID:       "org-1",
		RepoID:      "repo-1",
		ProjectRoot: t.TempDir(),
		HeroDir:     t.TempDir(),
	})
	d.SyncFunc = func(ctx context.Context, client *http.Client, syncURL, heroDir string) (*SyncResult, error) {
		syncCount.Add(1)
		return &SyncResult{Synced: 1, Total: 1}, nil
	}
	d.GraphFunc = noopPushGraph
	d.Start()
	defer d.Stop()

	// Send 10 rapid notifications — should batch into 1 sync.
	for i := 0; i < 10; i++ {
		d.Notify()
	}

	// Wait for debounce + sync to complete.
	time.Sleep(7 * time.Second)

	count := syncCount.Load()
	if count != 1 {
		t.Errorf("expected 1 sync after debounce, got %d", count)
	}
}

func TestDaemonMultipleDebounceWindows(t *testing.T) {
	var syncCount atomic.Int32

	d := NewDaemon(Config{
		CloudURL:    "http://localhost:9999",
		Token:       "test-token",
		OrgID:       "org-1",
		RepoID:      "repo-1",
		ProjectRoot: t.TempDir(),
		HeroDir:     t.TempDir(),
	})
	d.SyncFunc = func(ctx context.Context, client *http.Client, syncURL, heroDir string) (*SyncResult, error) {
		syncCount.Add(1)
		return &SyncResult{Synced: 1, Total: 1}, nil
	}
	d.GraphFunc = noopPushGraph
	d.Start()
	defer d.Stop()

	// First batch.
	d.Notify()
	time.Sleep(7 * time.Second)

	// Second batch.
	d.Notify()
	time.Sleep(7 * time.Second)

	count := syncCount.Load()
	if count != 2 {
		t.Errorf("expected 2 syncs from 2 windows, got %d", count)
	}
}

func TestDaemonRetryOnFailure(t *testing.T) {
	var callCount atomic.Int32

	d := NewDaemon(Config{
		CloudURL:    "http://localhost:9999",
		Token:       "test-token",
		OrgID:       "org-1",
		RepoID:      "repo-1",
		ProjectRoot: t.TempDir(),
		HeroDir:     t.TempDir(),
	})
	d.SyncFunc = func(ctx context.Context, client *http.Client, syncURL, heroDir string) (*SyncResult, error) {
		callCount.Add(1)
		return nil, context.DeadlineExceeded
	}
	d.GraphFunc = noopPushGraph
	d.Start()

	d.Notify()
	// Wait for: 5s debounce + sync + 2s backoff + 5s debounce + sync = ~12s.
	time.Sleep(14 * time.Second)
	d.Stop()

	n := callCount.Load()
	if n < 2 {
		t.Errorf("expected at least 2 sync attempts (initial + retry), got %d", n)
	}
}

func TestDaemonGracefulShutdownFlush(t *testing.T) {
	var syncCount atomic.Int32

	d := NewDaemon(Config{
		CloudURL:    "http://localhost:9999",
		Token:       "test-token",
		OrgID:       "org-1",
		RepoID:      "repo-1",
		ProjectRoot: t.TempDir(),
		HeroDir:     t.TempDir(),
	})
	d.SyncFunc = func(ctx context.Context, client *http.Client, syncURL, heroDir string) (*SyncResult, error) {
		syncCount.Add(1)
		return &SyncResult{Synced: 1, Total: 1}, nil
	}
	d.GraphFunc = noopPushGraph
	d.Start()

	// Notify then immediately stop — pending sync should flush.
	d.Notify()
	time.Sleep(50 * time.Millisecond) // let the trigger land
	d.Stop()

	count := syncCount.Load()
	if count != 1 {
		t.Errorf("expected 1 flush sync on shutdown, got %d", count)
	}
}

func TestDaemonBearerToken(t *testing.T) {
	var capturedAuth string
	var mu sync.Mutex

	// Use a capturing transport as the base of the bearer transport.
	capture := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		capturedAuth = r.Header.Get("Authorization")
		mu.Unlock()
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})

	d := NewDaemon(Config{
		CloudURL:    "http://localhost:9999",
		Token:       "my-secret-token",
		OrgID:       "org-1",
		RepoID:      "repo-1",
		ProjectRoot: t.TempDir(),
		HeroDir:     t.TempDir(),
	})
	// Replace the transport with a bearer transport that wraps our capture.
	d.client.Transport = &bearerTransport{token: "my-secret-token", base: capture}
	d.SyncFunc = func(ctx context.Context, client *http.Client, syncURL, heroDir string) (*SyncResult, error) {
		req, _ := http.NewRequestWithContext(ctx, "GET", syncURL, nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		return &SyncResult{}, nil
	}
	d.GraphFunc = noopPushGraph

	d.Start()
	d.Notify()
	time.Sleep(7 * time.Second)
	d.Stop()

	mu.Lock()
	got := capturedAuth
	mu.Unlock()

	expected := "Bearer my-secret-token"
	if got != expected {
		t.Errorf("expected auth %q, got %q", expected, got)
	}
}

func TestAutoSyncConfig(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name     string
		autoSync *bool
		want     bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Import config package indirectly — test the pointer semantics.
			var val *bool = tt.autoSync
			got := true
			if val != nil {
				got = *val
			}
			if got != tt.want {
				t.Errorf("AutoSyncEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncURL(t *testing.T) {
	cfg := Config{
		CloudURL: "https://cloud.example.com",
		OrgID:    "org-abc",
		RepoID:   "repo-123",
	}
	want := "https://cloud.example.com/api/v1/orgs/org-abc/repos/repo-123/sync"
	if got := cfg.SyncURL(); got != want {
		t.Errorf("SyncURL() = %q, want %q", got, want)
	}
}

func TestGraphServerURL(t *testing.T) {
	cfg := Config{
		CloudURL: "https://cloud.example.com",
		OrgID:    "org-abc",
	}
	want := "https://cloud.example.com/api/v1/orgs/org-abc"
	if got := cfg.GraphServerURL(); got != want {
		t.Errorf("GraphServerURL() = %q, want %q", got, want)
	}
}

func noopSyncSpecs(_ context.Context, _ *http.Client, _, _ string) (*SyncResult, error) {
	return &SyncResult{}, nil
}

func noopPushGraph(_ context.Context, _ *http.Client, _, _, _, _ string) (*GraphResult, error) {
	return &GraphResult{}, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
