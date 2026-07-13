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
// Policy: coexist. When a new daemon starts for a (workspace, client)
// pair that already has a LIVE incumbent, we cannot tell from the pidfile
// alone whether the client dropped the old connection (reconnect → the
// incumbent should die) or whether both connections are live (concurrent
// server → the incumbent is mid-conversation with the agent). Signaling a
// live-and-connected incumbent tears its transport out from under the
// agent — the reported Codex "transport closed" bug. So we do NOT signal
// it: the newcomer claims a distinct per-pid pidfile and both daemons
// coexist. Each still self-reaps when ITS OWN parent/connection dies via
// the parent-liveness watchdog, so this never reintroduces the unbounded
// duplicate leak — it trades "at most one daemon" for "never kill a live
// one". A stale pidfile (holder dead) is still overwritten as free, so
// genuinely-dead/orphaned daemons are reaped exactly as before.

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

// acquireMCPSingleton reaps genuinely-dead/orphaned `hero mcp` daemons for
// a given (workspace, client) pair while never killing a live, connected
// one. If a LIVE incumbent already holds the pidfile at path, this process
// coexists — it claims a distinct per-pid pidfile (see coexistRelease) and
// leaves the incumbent's record and process untouched. A stale pidfile
// (holder dead) is treated as free and overwritten. Returns a release func
// that removes only this process's own pidfile on clean shutdown, so a
// coexisting daemon never deletes the incumbent's file (and vice versa).
func acquireMCPSingleton(path string, self, ppid int) (func(), error) {
	if rec := readMCPPIDRecord(path); rec != nil {
		if rec.PID != self && singletonIsAlive(rec.PID) {
			// Live incumbent for this client+workspace. Killing it is
			// the reported Codex bug (transport torn out mid-session),
			// so coexist instead of supersede: claim a per-pid pidfile
			// and leave the incumbent serving. Each daemon self-reaps
			// when its own parent/connection dies.
			return coexistRelease(path, self, ppid)
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

// coexistRelease claims a per-pid pidfile (path suffixed with ".<pid>") so a
// second live daemon on the same (workspace, parent) does not disturb the
// incumbent's record. Returns a release that removes only our own suffixed
// pidfile, and only if we still own it.
func coexistRelease(path string, self, ppid int) (func(), error) {
	alt := fmt.Sprintf("%s.%d", path, self)
	if err := writeMCPPIDRecord(alt, self, ppid); err != nil {
		return nil, err
	}
	return func() {
		if rec := readMCPPIDRecord(alt); rec != nil && rec.PID == self {
			_ = os.Remove(alt)
		}
	}, nil
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
