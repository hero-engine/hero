package serve

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestMCPSingleton_CoexistsWithLiveIncumbent is the primary-fix regression:
// when a live daemon already holds the (workspace, client) pidfile, a second
// startup must NOT signal it (killing a live, connected incumbent is the
// reported Codex "transport closed" bug). Instead the newcomer coexists via a
// per-pid pidfile, leaving the incumbent's record and process untouched. This
// is the inverse of the reproduced failure, at the acquire seam.
func TestMCPSingleton_CoexistsWithLiveIncumbent(t *testing.T) {
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
	relNewcomer, err := acquireMCPSingleton(path, newcomer, ppid)
	if err != nil {
		t.Fatalf("newcomer acquire: %v", err)
	}

	// The incumbent must NOT have been signaled — a live session is never
	// torn down.
	if signaled[incumbent] {
		t.Fatalf("live incumbent (pid %d) was signaled — coexist policy violated", incumbent)
	}
	// The incumbent's record is untouched: it still owns the primary pidfile.
	if rec := readMCPPIDRecord(path); rec == nil || rec.PID != incumbent {
		t.Fatalf("incumbent's pidfile was disturbed by coexisting newcomer: %+v", rec)
	}
	// The newcomer owns a distinct per-pid pidfile.
	altPath := fmt.Sprintf("%s.%d", path, newcomer)
	if rec := readMCPPIDRecord(altPath); rec == nil || rec.PID != newcomer {
		t.Fatalf("newcomer did not claim its per-pid pidfile: %+v", rec)
	}

	// The newcomer's release removes only its own suffixed file, never the
	// incumbent's.
	relNewcomer()
	if _, err := os.Stat(altPath); !os.IsNotExist(err) {
		t.Fatalf("newcomer release did not remove its per-pid pidfile: err=%v", err)
	}
	if rec := readMCPPIDRecord(path); rec == nil || rec.PID != incumbent {
		t.Fatalf("newcomer release clobbered incumbent's pidfile: %+v", rec)
	}

	// The incumbent's own release cleanly removes the primary pidfile.
	relIncumbent()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("incumbent release did not remove primary pidfile: err=%v", err)
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

// TestMCPSingleton_ReconnectLeavesNewServerServing models the reconnect the
// fix must not break: the client drops the old connection and opens a new one
// while the old daemon is still winding down. Under the coexist policy the new
// daemon claims a per-pid pidfile and keeps serving (it never refuses); when
// the old daemon exits it removes only the primary pidfile, never the new
// server's file. Both release cleanly with no leak.
func TestMCPSingleton_ReconnectLeavesNewServerServing(t *testing.T) {
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
	// one is still alive/winding down. It coexists rather than killing old.
	aliveSet[newConn] = true
	relNew, err := acquireMCPSingleton(path, newConn, ppid)
	if err != nil {
		t.Fatalf("new connection acquire: %v", err)
	}
	if signaled[oldConn] {
		t.Fatalf("old connection (pid %d) was signaled — coexist policy violated", oldConn)
	}

	// The new server is serving with its own per-pid pidfile.
	altPath := fmt.Sprintf("%s.%d", path, newConn)
	if rec := readMCPPIDRecord(altPath); rec == nil || rec.PID != newConn {
		t.Fatalf("new server did not claim its per-pid pidfile: %+v", rec)
	}

	// The old daemon exits (its release runs). It removes only the primary
	// pidfile and must not touch the new server's file.
	relOld()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("old server release did not remove primary pidfile: err=%v", err)
	}
	if rec := readMCPPIDRecord(altPath); rec == nil || rec.PID != newConn {
		t.Fatalf("old server release clobbered the new server's pidfile: %+v", rec)
	}

	// Clean shutdown of the surviving server releases its own file.
	relNew()
	if _, err := os.Stat(altPath); !os.IsNotExist(err) {
		t.Fatalf("surviving server release did not remove its pidfile: err=%v", err)
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
