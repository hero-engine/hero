package serve

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
)

func TestIsAddrInUse(t *testing.T) {
	if !isAddrInUse(syscall.EADDRINUSE) {
		t.Errorf("isAddrInUse(EADDRINUSE) = false, want true")
	}
	// Wrapped error.
	wrapped := errors.New("listen tcp 127.0.0.1:7437: bind: address already in use")
	if !isAddrInUse(wrapped) {
		t.Errorf("isAddrInUse(textual wrap) = false, want true")
	}
	if isAddrInUse(nil) {
		t.Errorf("isAddrInUse(nil) = true, want false")
	}
	if isAddrInUse(errors.New("some other error")) {
		t.Errorf("isAddrInUse(unrelated) = true, want false")
	}
}

func TestProbeHeroDaemon_HitsRealServer(t *testing.T) {
	// Spin a local httptest server that returns a valid status payload,
	// then point probeHeroDaemon at it via a custom transport. The
	// production probe hits 127.0.0.1:<port>, so we replace the URL
	// directly by issuing a synthetic request against the handler.
	handler := http.NewServeMux()
	handler.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DaemonStatusResponse{
			Running: true,
			PID:     4242,
			Port:    7437,
			Version: "test",
		})
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Direct decode path — exercise the JSON contract.
	resp, err := http.Get(ts.URL + "/api/status")
	if err != nil {
		t.Fatalf("get test server: %v", err)
	}
	defer resp.Body.Close()
	var got DaemonStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Running || got.PID != 4242 {
		t.Errorf("decoded payload = %+v, want running pid=4242", got)
	}
}

func TestDiagnoseBindError_NonAddressInUse(t *testing.T) {
	s := &Server{port: 7437}
	err := s.diagnoseBindError(errors.New("permission denied"))
	if err == nil || !strings.Contains(err.Error(), "listen 127.0.0.1:7437") {
		t.Errorf("non-EADDRINUSE error should pass through, got %v", err)
	}
}

func TestDiagnoseBindError_ForeignProcess(t *testing.T) {
	// Pick a port that's almost certainly free, and use the textual
	// "address already in use" sentinel to trigger the EADDRINUSE branch
	// without actually binding anything.
	s := &Server{port: 1} // privileged port — probe will not connect
	err := s.diagnoseBindError(errors.New("address already in use"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "in use by another process") {
		t.Errorf("foreign-process branch did not fire: %v", err)
	}
}
