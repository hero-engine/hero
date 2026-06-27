package cloud

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

// Config holds the parameters the sync daemon needs.
type Config struct {
	CloudURL    string
	Token       string
	OrgID       string
	RepoID      string
	ProjectRoot string
	HeroDir     string
}

// SyncURL returns the spec sync endpoint URL.
func (c Config) SyncURL() string {
	return fmt.Sprintf("%s/api/v1/orgs/%s/repos/%s/sync", c.CloudURL, c.OrgID, c.RepoID)
}

// GraphServerURL returns the org-scoped graph base URL.
func (c Config) GraphServerURL() string {
	return fmt.Sprintf("%s/api/v1/orgs/%s", c.CloudURL, c.OrgID)
}

const (
	debounceInterval = 5 * time.Second
	maxBackoff       = 60 * time.Second
	warnAfter        = 3
)

// Daemon is a background sync daemon that pushes specs and graph data
// to hero-cloud when triggered. It debounces rapid triggers into a
// single sync push per debounce window.
//
// The caller sends triggers via Notify() — typically by subscribing to
// the serve.EventBus and forwarding relevant events. This avoids a
// circular dependency between the cloud and serve packages.
type Daemon struct {
	cfg     Config
	client  *http.Client
	ctx     context.Context
	cancel  context.CancelFunc
	trigger chan struct{}

	mu       sync.Mutex
	failures int
	lastSync time.Time

	wg sync.WaitGroup

	// SyncFunc and GraphFunc are injectable for testing.
	SyncFunc  func(ctx context.Context, client *http.Client, syncURL, heroDir string) (*SyncResult, error)
	GraphFunc func(ctx context.Context, client *http.Client, serverURL, orgID, heroDir, projectRoot string) (*GraphResult, error)
}

// NewAuthenticatedClient returns an HTTP client that injects a Bearer token
// on every request. Shared by both the sync daemon and the presence reporter.
func NewAuthenticatedClient(token string) *http.Client {
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: &bearerTransport{token: token, base: http.DefaultTransport},
	}
}

// NewDaemon creates a sync daemon with the given config.
func NewDaemon(cfg Config) *Daemon {
	ctx, cancel := context.WithCancel(context.Background())

	return &Daemon{
		cfg:    cfg,
		client: NewAuthenticatedClient(cfg.Token),
		ctx:    ctx,
		cancel: cancel,
		trigger: make(chan struct{}, 64),
		SyncFunc:  SyncSpecs,
		GraphFunc: PushGraph,
	}
}

// Start launches the background event loop.
func (d *Daemon) Start() {
	d.wg.Add(1)
	go d.loop()
}

// Stop cancels the daemon and waits for the final sync to complete
// (with a 5-second deadline).
func (d *Daemon) Stop() {
	d.cancel()
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		fmt.Fprintf(os.Stderr, "hero cloud-sync: shutdown deadline exceeded\n")
	}
}

// Notify signals that something changed and a sync should happen
// after the debounce window. Non-blocking — drops if the trigger
// buffer is full.
func (d *Daemon) Notify() {
	select {
	case d.trigger <- struct{}{}:
	default:
	}
}

// TriggerSync forces an immediate sync, bypassing the debounce timer.
func (d *Daemon) TriggerSync() {
	d.Notify()
}

func (d *Daemon) loop() {
	defer d.wg.Done()

	var timer *time.Timer
	var timerC <-chan time.Time
	pending := false

	for {
		select {
		case <-d.ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			if pending {
				d.syncNow()
			}
			return

		case <-d.trigger:
			pending = true
			if timer == nil {
				timer = time.NewTimer(debounceInterval)
				timerC = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(debounceInterval)
			}

		case <-timerC:
			if pending {
				d.syncNow()
				pending = false
			}
			timer = nil
			timerC = nil
		}
	}
}

func (d *Daemon) syncNow() {
	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	specErr := d.doSyncSpecs(ctx)
	graphErr := d.doSyncGraph(ctx)

	d.mu.Lock()

	if specErr != nil || graphErr != nil {
		d.failures++
		failures := d.failures
		if failures == warnAfter {
			fmt.Fprintf(os.Stderr, "hero cloud-sync: %d consecutive failures", failures)
			if specErr != nil {
				fmt.Fprintf(os.Stderr, " (specs: %v)", specErr)
			}
			if graphErr != nil {
				fmt.Fprintf(os.Stderr, " (graph: %v)", graphErr)
			}
			fmt.Fprintln(os.Stderr)
		}
		d.mu.Unlock()
		d.scheduleRetry(failures)
		return
	}

	d.failures = 0
	d.lastSync = time.Now()
	d.mu.Unlock()
}

func (d *Daemon) doSyncSpecs(ctx context.Context) error {
	_, err := d.SyncFunc(ctx, d.client, d.cfg.SyncURL(), d.cfg.HeroDir)
	return err
}

func (d *Daemon) doSyncGraph(ctx context.Context) error {
	_, err := d.GraphFunc(ctx, d.client, d.cfg.GraphServerURL(), d.cfg.OrgID, d.cfg.HeroDir, d.cfg.ProjectRoot)
	return err
}

func (d *Daemon) scheduleRetry(failures int) {
	backoff := time.Duration(1<<min(failures, 6)) * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	jitter := time.Duration(rand.Int63n(int64(500 * time.Millisecond)))
	delay := backoff + jitter

	go func() {
		select {
		case <-time.After(delay):
			d.Notify()
		case <-d.ctx.Done():
		}
	}()
}

// bearerTransport injects Authorization: Bearer <token> on every request.
type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(r2)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
