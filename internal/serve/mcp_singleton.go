package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// The MCP singleton guards against a leak the parent-death watchdog
// structurally cannot cover: a long-running client (e.g. Codex) that
// reconnects or opens multiple sessions accumulates one live `hero mcp`
// daemon per connection, forever. Those daemons are not orphans — their
// parent is still alive — so watchdogGetppid() never changes and the
// watchdog never fires. This adds a live-duplicate guard.
//
// Identity key: (workspace, parent pid). The workspace is s.heroDir; the
// "client" is the parent process (the harness that spawned us). A single
// long-lived client that reconnects keeps the same parent pid, so all of
// its daemons contend for one pidfile and dedup cleanly. Two genuinely
// different clients on the same workspace have different parent pids, get
// separate pidfiles, and never supersede one another.
//
// Policy: supersede. When a new daemon starts for a (workspace, client)
// pair that already has a live incumbent, the incumbent is signaled to
// exit and the newcomer takes over. This makes a reconnect win — the
// client dropped the old connection and opened a new one, so the new
// daemon must be the one left serving. (Refusing instead would leave the
// client's fresh stdio pipe attached to a daemon that immediately exits.)

// Seam vars default to the real runtime behavior; tests override them to
// drive the supersede/stale-holder branches without spawning processes.
// Production code never reassigns these.
var (
	singletonIsAlive = IsProcessAlive
	singletonSignal  = func(pid int) error {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Signal(syscall.SIGTERM)
	}
)

// mcpPIDRecord is the JSON written to the MCP singleton pidfile.
type mcpPIDRecord struct {
	PID       int       `json:"pid"`
	PPID      int       `json:"ppid"`
	StartedAt time.Time `json:"started_at"`
}

// mcpPIDFilePath returns the workspace-scoped MCP pidfile path for a
// given parent pid. It lives directly under the workspace .hero dir,
// alongside other runtime state (mcp-debug.log, jobs.db). Keying the
// filename by parent pid keeps distinct clients from colliding.
func mcpPIDFilePath(heroDir string, ppid int) string {
	return filepath.Join(heroDir, fmt.Sprintf("mcp-%d.pid", ppid))
}

// acquireMCPSingleton ensures at most one live `hero mcp` daemon serves a
// given (workspace, client) pair. If a live incumbent already holds the
// pidfile at path it is signaled to exit (supersede), then this process
// claims the file. A stale pidfile (holder dead) is treated as free and
// overwritten. Returns a release func that removes the pidfile on clean
// shutdown — but only if this process still owns it, so a superseded
// daemon never deletes its successor's file.
func acquireMCPSingleton(path string, self, ppid int) (func(), error) {
	if rec := readMCPPIDRecord(path); rec != nil {
		if rec.PID != self && singletonIsAlive(rec.PID) {
			// Live incumbent for this client+workspace. Supersede it:
			// a reconnect should win. Best-effort — a signal failure
			// must not stop the newcomer from taking over.
			_ = singletonSignal(rec.PID)
		}
		// A dead holder (stale pidfile) or our own pid is treated as
		// free; we simply overwrite the record below.
	}

	if err := writeMCPPIDRecord(path, self, ppid); err != nil {
		return nil, err
	}

	release := func() {
		if rec := readMCPPIDRecord(path); rec != nil && rec.PID == self {
			_ = os.Remove(path)
		}
	}
	return release, nil
}

// readMCPPIDRecord reads and parses the MCP pidfile at path. Returns nil
// when the file is missing or unparseable — callers treat nil as "no
// live incumbent", which is the safe default (worst case: one extra
// daemon, exactly the pre-fix behavior).
func readMCPPIDRecord(path string) *mcpPIDRecord {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var rec mcpPIDRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil
	}
	return &rec
}

// writeMCPPIDRecord writes this process's ownership record to path,
// creating the parent directory if needed.
func writeMCPPIDRecord(path string, self, ppid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating mcp pidfile directory: %w", err)
	}
	rec := mcpPIDRecord{
		PID:       self,
		PPID:      ppid,
		StartedAt: time.Now().UTC(),
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling mcp pidfile: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing mcp pidfile %s: %w", path, err)
	}
	return nil
}
