package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

// DaemonStatusResponse is the JSON shape returned by /api/status and
// parsed by the CLI status / stop commands. Field names match the
// endpoint response exactly.
type DaemonStatusResponse struct {
	Running       bool             `json:"running"`
	PID           int              `json:"pid"`
	Port          int              `json:"port"`
	StartedAt     time.Time        `json:"started_at"`
	UptimeSeconds int64            `json:"uptime_seconds"`
	Version       string           `json:"version"`
	ProjectCount  int              `json:"project_count"`
	Projects      []DaemonStatusPC `json:"projects"`
}

// DaemonStatusPC is one project entry inside DaemonStatusResponse.
type DaemonStatusPC struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// isAddrInUse reports whether err originated from EADDRINUSE. Probes
// both the wrapped syscall and the textual "address already in use"
// fallback for platforms that don't surface the errno cleanly.
func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	// net.Listen wraps the error; the wrapped chain occasionally hides
	// the syscall. Fall back to a string sniff for cross-platform
	// safety.
	return strings.Contains(strings.ToLower(err.Error()), "address already in use")
}

// probeHeroDaemon attempts to GET /api/status from a daemon on the
// given port. Returns the parsed status on success, nil on any failure
// — the caller treats nil as "not a hero daemon (or unreachable)".
func probeHeroDaemon(port int) *DaemonStatusResponse {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/status", port)
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var out DaemonStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	if !out.Running || out.PID == 0 {
		return nil
	}
	return &out
}

// PortListenerHeld reports whether a TCP listener can be opened on the
// given port. Returns true when the port is held by some other
// process. The probe is best-effort and racy by nature — callers use
// it as a signal, not a guarantee.
func PortListenerHeld(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return isAddrInUse(err)
	}
	ln.Close()
	return false
}

// IsProcessAlive returns true when the OS reports a process with the
// given PID is still running. Sends signal 0 (no-op) and treats
// "process not found" as dead. Handles both syscall.ESRCH (Linux) and
// the Go-wrapped os.ErrProcessDone / "process already finished"
// (Darwin) — Go does not surface ESRCH consistently across platforms.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
		return false
	}
	// Darwin wraps ESRCH into the string "process already finished"
	// without preserving the errno chain — match it by text.
	if strings.Contains(err.Error(), "already finished") {
		return false
	}
	// Any other error (EPERM, etc.) means the process exists but we
	// lack permission to signal it — treat as alive.
	return true
}
