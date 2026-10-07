package serve

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func drainFixture(t *testing.T, h http.Handler) (*Server, string) {
	t.Helper()
	s := &Server{}
	reqCtx, cancel := context.WithCancel(context.Background())
	s.requestCancel = cancel
	s.httpServer = &http.Server{Handler: s.countInFlight(h), BaseContext: func(net.Listener) context.Context { return reqCtx }}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go s.httpServer.Serve(ln)
	return s, ln.Addr().String()
}

func drain(t *testing.T, s *Server) (time.Duration, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	err := s.drainHTTP(ctx)
	return time.Since(start), err
}

// followup-serve-shutdown AC-1: a connection that never sends a request
// (browser preconnect) no longer holds shutdown for the 5s deadline.
func TestDrainClosesIdleNewConnectionsPromptly(t *testing.T) {
	s, addr := drainFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)
	took, err := drain(t, s)
	if err != nil || took > time.Second {
		t.Fatalf("drain took %s, err %v; want prompt, nil", took, err)
	}
}

// followup-serve-shutdown AC-2: an in-flight request still completes.
func TestDrainLetsInFlightRequestsFinish(t *testing.T) {
	s, addr := drainFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		io.WriteString(w, "done")
	}))
	got := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/")
		if err != nil {
			got <- "error: " + err.Error()
			return
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		got <- string(b)
	}()
	time.Sleep(50 * time.Millisecond)
	if _, err := drain(t, s); err != nil {
		t.Fatalf("drain err %v", err)
	}
	if body := <-got; body != "done" {
		t.Fatalf("in-flight request cut off: %q", body)
	}
}

// followup-serve-shutdown AC-3: a streaming handler that waits on its
// request context ends after the short grace, well before the deadline,
// and shutdown reports no error.
func TestDrainEndsStreamingHandlersAfterGrace(t *testing.T) {
	s, addr := drainFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	took, err := drain(t, s)
	if err != nil || took > 2*time.Second {
		t.Fatalf("drain took %s, err %v; want ~1s grace, nil", took, err)
	}
}
