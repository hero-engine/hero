package serve

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestMCPSingleton_SupersedesLiveIncumbent verifies the core dedup: when a
// live daemon already holds the (workspace, client) pidfile, a second startup
// signals it to exit and takes over the file, leaving exactly one owner. This
// is the leak the orphan watchdog cannot reap — a still-connected duplicate
// under a live parent.
func TestMCPSingleton_SupersedesLiveIncumbent(t *testing.T) {
	restore := stubSingletonSeams(t)
	defer restore()

	path := filepath.Join(t.TempDir(), "mcp-1000.pid")
	const incumbent, newcomer, ppid = 111, 222, 1000

	// Incumbent claims the lock first.
	aliveSet[incumbent] = true
	relIncumbent, err := acquireMCPSingleton(path, incumbent, ppid)
	if err != nil {
		t.Fatalf("incumbent acquire: %v", err)
	}
	if rec := readMCPPIDRecord(path); rec == nil || rec.PID != incumbent {
		t.Fatalf("incumbent did not claim pidfile: %+v", rec)
	}

	// Newcomer starts for the same client+workspace; incumbent is alive.
	aliveSet[newcomer] = true
	if _, err := acquireMCPSingleton(path, newcomer, ppid); err != nil {
		t.Fatalf("newcomer acquire: %v", err)
	}

	// The incumbent must have received the exit signal...
	if !signaled[incumbent] {
		t.Fatalf("incumbent (pid %d) was not signaled to exit", incumbent)
	}
	// ...and exactly one owner remains: the newcomer.
	rec := readMCPPIDRecord(path)
	if rec == nil || rec.PID != newcomer {
		t.Fatalf("newcomer did not take over pidfile: %+v", rec)
	}

	// The superseded incumbent's release must NOT delete the successor's
	// file — it no longer owns it.
	relIncumbent()
	if rec := readMCPPIDRecord(path); rec == nil || rec.PID != newcomer {
		t.Fatalf("superseded incumbent release clobbered successor: %+v", rec)
	}
}

// TestMCPSingleton_StalePidfileTreatedAsFree verifies a pidfile whose holder
// is dead is treated as free: the newcomer takes over and rewrites the file,
// and never signals the dead holder.
func TestMCPSingleton_StalePidfileTreatedAsFree(t *testing.T) {
	restore := stubSingletonSeams(t)
	defer restore()

	path := filepath.Join(t.TempDir(), "mcp-1000.pid")
	const deadHolder, newcomer, ppid = 111, 222, 1000

	// Seed a stale pidfile: a record whose PID is not alive.
	if err := writeMCPPIDRecord(path, deadHolder, ppid); err != nil {
		t.Fatalf("seed stale pidfile: %v", err)
	}
	// deadHolder intentionally absent from aliveSet → IsAlive == false.
	aliveSet[newcomer] = true

	if _, err := acquireMCPSingleton(path, newcomer, ppid); err != nil {
		t.Fatalf("newcomer acquire over stale pidfile: %v", err)
	}

	if signaled[deadHolder] {
		t.Fatalf("dead holder (pid %d) should never be signaled", deadHolder)
	}
	rec := readMCPPIDRecord(path)
	if rec == nil || rec.PID != newcomer {
		t.Fatalf("newcomer did not take over stale pidfile: %+v", rec)
	}
}

// TestMCPSingleton_ReconnectEndsWithNewServer models the reconnect the fix
// must not break: the client drops the old connection and opens a new one.
// The new daemon must end up owning the lock (serving), not refuse forever.
func TestMCPSingleton_ReconnectEndsWithNewServer(t *testing.T) {
	restore := stubSingletonSeams(t)
	defer restore()

	path := filepath.Join(t.TempDir(), "mcp-1000.pid")
	const oldConn, newConn, ppid = 111, 222, 1000

	aliveSet[oldConn] = true
	relOld, err := acquireMCPSingleton(path, oldConn, ppid)
	if err != nil {
		t.Fatalf("old connection acquire: %v", err)
	}

	// Reconnect: client spawns a fresh daemon (same parent) while the old
	// one is still winding down.
	aliveSet[newConn] = true
	relNew, err := acquireMCPSingleton(path, newConn, ppid)
	if err != nil {
		t.Fatalf("new connection acquire: %v", err)
	}

	// The old daemon exits (its release runs). It must not remove the new
	// owner's pidfile.
	relOld()

	rec := readMCPPIDRecord(path)
	if rec == nil || rec.PID != newConn {
		t.Fatalf("reconnect did not leave the new server owning the lock: %+v", rec)
	}

	// Clean shutdown of the surviving server releases the lock.
	relNew()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("surviving server release did not remove pidfile: err=%v", err)
	}
}

// TestMCPSingleton_DistinctClientsDoNotCollide verifies two genuinely
// different clients (different parent pids) on the same workspace get
// separate pidfiles and never supersede each other.
func TestMCPSingleton_DistinctClientsDoNotCollide(t *testing.T) {
	restore := stubSingletonSeams(t)
	defer restore()

	heroDir := t.TempDir()
	const pidA, ppidA = 111, 1000
	const pidB, ppidB = 222, 2000

	aliveSet[pidA] = true
	aliveSet[pidB] = true

	if _, err := acquireMCPSingleton(mcpPIDFilePath(heroDir, ppidA), pidA, ppidA); err != nil {
		t.Fatalf("client A acquire: %v", err)
	}
	if _, err := acquireMCPSingleton(mcpPIDFilePath(heroDir, ppidB), pidB, ppidB); err != nil {
		t.Fatalf("client B acquire: %v", err)
	}

	if signaled[pidA] || signaled[pidB] {
		t.Fatalf("distinct clients must not signal each other: A=%v B=%v", signaled[pidA], signaled[pidB])
	}
	if rec := readMCPPIDRecord(mcpPIDFilePath(heroDir, ppidA)); rec == nil || rec.PID != pidA {
		t.Fatalf("client A lock disturbed: %+v", rec)
	}
	if rec := readMCPPIDRecord(mcpPIDFilePath(heroDir, ppidB)); rec == nil || rec.PID != pidB {
		t.Fatalf("client B lock disturbed: %+v", rec)
	}
}

// TestMCPSingleton_OrphanWatchdogStillFires is the explicit regression guard
// that the live-duplicate dedup did not break orphan reaping. It engages the
// singleton lock (the new mechanism) and then drives the orphan watchdog's
// reparent-decision branch, asserting the watchdog still calls watchdogExit(0)
// on a ppid change. The two guards are orthogonal: dedup reaps live duplicates
// under a live parent; the watchdog reaps orphans whose parent died. This test
// proves adding the former leaves the latter intact. It must NOT run in
// parallel — it mutates the watchdog seam vars.
func TestMCPSingleton_OrphanWatchdogStillFires(t *testing.T) {
	restoreSingleton := stubSingletonSeams(t)
	defer restoreSingleton()

	// Engage the singleton so its state is live during the watchdog run.
	path := filepath.Join(t.TempDir(), "mcp-1000.pid")
	aliveSet[111] = true
	if _, err := acquireMCPSingleton(path, 111, 1000); err != nil {
		t.Fatalf("acquire singleton: %v", err)
	}

	origExit := watchdogExit
	origGetppid := watchdogGetppid
	origInterval := parentWatchdogInterval
	defer func() {
		watchdogExit = origExit
		watchdogGetppid = origGetppid
		parentWatchdogInterval = origInterval
	}()

	var calls int64
	watchdogGetppid = func() int {
		if atomic.AddInt64(&calls, 1) == 1 {
			return 1000 // startPpid
		}
		return 1 // reparented → parent died
	}
	exited := make(chan int, 1)
	watchdogExit = func(code int) {
		exited <- code
		select {} // os.Exit never returns
	}
	parentWatchdogInterval = time.Millisecond

	done := make(chan struct{})
	defer close(done)
	startParentWatchdog(done)

	select {
	case code := <-exited:
		if code != 0 {
			t.Fatalf("watchdog exited with code %d, want 0", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("orphan watchdog did not fire on ppid change with dedup engaged")
	}
}

// aliveSet and signaled back the stubbed singleton seams. aliveSet maps pid →
// alive; signaled records which pids received the supersede signal.
var (
	aliveSet = map[int]bool{}
	signaled = map[int]bool{}
)

// stubSingletonSeams swaps singletonIsAlive/singletonSignal for map-backed
// fakes and resets the maps. Returns a restore func. Tests using it must not
// run in parallel — they mutate package-level seam vars.
func stubSingletonSeams(t *testing.T) func() {
	t.Helper()
	origAlive := singletonIsAlive
	origSignal := singletonSignal
	aliveSet = map[int]bool{}
	signaled = map[int]bool{}
	singletonIsAlive = func(pid int) bool { return aliveSet[pid] }
	singletonSignal = func(pid int) error {
		signaled[pid] = true
		return nil
	}
	return func() {
		singletonIsAlive = origAlive
		singletonSignal = origSignal
	}
}
