package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/serve/pages/now"
)

// fakeSub implements Subscriber with an injectable channel — tests
// drive events without needing the real serve.EventBus.
type fakeSub struct {
	mu    sync.Mutex
	subs  map[uint64]chan Event
	nextI uint64
}

func newFakeSub() *fakeSub { return &fakeSub{subs: map[uint64]chan Event{}} }

func (f *fakeSub) Subscribe(bufSize int) (uint64, <-chan Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextI
	f.nextI++
	ch := make(chan Event, bufSize)
	f.subs[id] = ch
	return id, ch
}

func (f *fakeSub) Unsubscribe(id uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if ch, ok := f.subs[id]; ok {
		delete(f.subs, id)
		close(ch)
	}
}

func (f *fakeSub) publish(ev Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ch := range f.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func TestNowHandler_FragmentEndpoints(t *testing.T) {
	h := NewNowHandler(now.Deps{}, nil)
	mux := http.NewServeMux()
	h.Mount(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, section := range []string{"inbox", "plate", "agents", "changes"} {
		resp, err := http.Get(srv.URL + "/api/now/" + section)
		if err != nil {
			t.Errorf("GET /api/now/%s: %v", section, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("/api/now/%s status = %d, want 200", section, resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Errorf("/api/now/%s Content-Type = %q, want text/html*", section, ct)
		}
		resp.Body.Close()
	}
}

func TestNowHandler_SSEHeaders(t *testing.T) {
	h := NewNowHandler(now.Deps{}, nil)
	mux := http.NewServeMux()
	h.Mount(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/now/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
}

func TestSectionForEventType(t *testing.T) {
	cases := map[string]string{
		"proposal_emitted":  "inbox",
		"proposal_accepted": "inbox",
		"peer.handoff.sent": "inbox",
		"spec.created":      "plate",
		"spec.modified":     "plate",
		"spec.deleted":      "plate",
		"index.rebuilt":     "changes",
		"health.check":      "",
		"":                  "",
	}
	for in, want := range cases {
		if got := sectionForEventType(in); got != want {
			t.Errorf("sectionForEventType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNowHandler_SSEDebouncesAndPublishes(t *testing.T) {
	sub := newFakeSub()
	h := NewNowHandler(now.Deps{}, sub)
	h.debounceDur = 20 * time.Millisecond
	mux := http.NewServeMux()
	h.Mount(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/now/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	defer resp.Body.Close()

	// Give the handler a moment to subscribe.
	time.Sleep(20 * time.Millisecond)

	// Burst three events for the same section.
	for i := 0; i < 3; i++ {
		sub.publish(Event{Type: "proposal_emitted"})
	}

	// Read in a goroutine; if the context expires the body is closed
	// and Read returns an error, releasing the goroutine.
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		var read []byte
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				read = append(read, buf[:n]...)
				if strings.Contains(string(read), "event: inbox") {
					done <- string(read)
					return
				}
			}
			if err != nil {
				done <- string(read)
				return
			}
		}
	}()

	var got string
	select {
	case got = <-done:
	case <-time.After(400 * time.Millisecond):
		got = "<read-timeout>"
	}
	if !strings.Contains(got, "event: connected") {
		t.Errorf("missing connected frame in stream: %q", got)
	}
	if !strings.Contains(got, "event: inbox") {
		t.Errorf("expected debounced 'event: inbox' frame in stream: %q", got)
	}
}
