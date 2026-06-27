package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const heartbeatInterval = 30 * time.Second

// PresenceReporter registers a CLI session with hero-cloud on start,
// sends periodic heartbeats, and unregisters on stop.
type PresenceReporter struct {
	cfg    Config
	client *http.Client

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu        sync.Mutex
	sessionID string
}

// NewPresenceReporter creates a presence reporter using the same config
// as the CloudSyncDaemon.
func NewPresenceReporter(cfg Config, client *http.Client) *PresenceReporter {
	ctx, cancel := context.WithCancel(context.Background())
	return &PresenceReporter{
		cfg:    cfg,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}
}

type registerRequest struct {
	RepoID   string `json:"repo_id"`
	SpecSlug string `json:"spec_slug"`
	Command  string `json:"command"`
}

type registerResponse struct {
	ID string `json:"id"`
}

// Start registers the session and begins the heartbeat loop.
func (p *PresenceReporter) Start(specSlug, command string) {
	id, err := p.register(specSlug, command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero presence: register failed: %v\n", err)
		return
	}

	p.mu.Lock()
	p.sessionID = id
	p.mu.Unlock()

	p.wg.Add(1)
	go p.heartbeatLoop()
}

// Stop unregisters the session and waits for the heartbeat loop to exit.
func (p *PresenceReporter) Stop() {
	p.cancel()
	p.wg.Wait()

	p.mu.Lock()
	id := p.sessionID
	p.mu.Unlock()

	if id != "" {
		p.unregister(id)
	}
}

func (p *PresenceReporter) register(specSlug, command string) (string, error) {
	body, _ := json.Marshal(registerRequest{
		RepoID:   p.cfg.RepoID,
		SpecSlug: specSlug,
		Command:  command,
	})

	url := fmt.Sprintf("%s/api/v1/orgs/%s/sessions", p.cfg.CloudURL, p.cfg.OrgID)
	req, err := http.NewRequestWithContext(p.ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result registerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func (p *PresenceReporter) heartbeatLoop() {
	defer p.wg.Done()
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			id := p.sessionID
			p.mu.Unlock()
			if id != "" {
				p.heartbeat(id)
			}
		}
	}
}

func (p *PresenceReporter) heartbeat(sessionID string) {
	url := fmt.Sprintf("%s/api/v1/orgs/%s/sessions/%s/heartbeat", p.cfg.CloudURL, p.cfg.OrgID, sessionID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (p *PresenceReporter) unregister(sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1/orgs/%s/sessions/%s", p.cfg.CloudURL, p.cfg.OrgID, sessionID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
