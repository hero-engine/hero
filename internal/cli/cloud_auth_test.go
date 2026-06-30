package cli

import (
	"strconv"
	"testing"
)

func TestCallbackListenerOSAssigned(t *testing.T) {
	ln, port, err := callbackListener("")
	if err != nil {
		t.Fatalf("callbackListener: %v", err)
	}
	defer ln.Close()
	if port <= 0 {
		t.Fatalf("expected an OS-assigned port > 0, got %d", port)
	}
}

func TestCallbackListenerConcurrentNoCollision(t *testing.T) {
	ln1, port1, err := callbackListener("")
	if err != nil {
		t.Fatalf("first listener: %v", err)
	}
	defer ln1.Close()

	ln2, port2, err := callbackListener("")
	if err != nil {
		t.Fatalf("second listener (should not collide on fixed port): %v", err)
	}
	defer ln2.Close()

	if port1 == port2 {
		t.Fatalf("two OS-assigned listeners got the same port %d", port1)
	}
}

func TestCallbackListenerExplicitFlagPort(t *testing.T) {
	// Grab a free port, release it, then ask for it explicitly.
	probe, p, err := callbackListener("")
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	probe.Close()

	ln, got, err := callbackListener(strconv.Itoa(p))
	if err != nil {
		t.Fatalf("explicit port bind: %v", err)
	}
	defer ln.Close()
	if got != p {
		t.Fatalf("explicit port = %d, want %d", got, p)
	}
}

func TestCallbackListenerEnvPort(t *testing.T) {
	probe, p, err := callbackListener("")
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	probe.Close()

	t.Setenv("HERO_CALLBACK_PORT", strconv.Itoa(p))
	ln, got, err := callbackListener("")
	if err != nil {
		t.Fatalf("env port bind: %v", err)
	}
	defer ln.Close()
	if got != p {
		t.Fatalf("env port = %d, want %d", got, p)
	}
}

func TestCallbackListenerFlagBeatsEnv(t *testing.T) {
	probeF, pf, _ := callbackListener("")
	probeF.Close()
	probeE, pe, _ := callbackListener("")
	probeE.Close()
	if pf == pe {
		// extremely unlikely, but guard the assertion
		t.Skip("probe ports collided")
	}

	t.Setenv("HERO_CALLBACK_PORT", strconv.Itoa(pe))
	ln, got, err := callbackListener(strconv.Itoa(pf))
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	defer ln.Close()
	if got != pf {
		t.Fatalf("flag should win: got %d, want %d", got, pf)
	}
}
