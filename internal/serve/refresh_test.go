package serve

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewBulkImportCommandUsesCanonicalBulkRefresh(t *testing.T) {
	projectRoot := t.TempDir()
	cmd := newBulkImportCommand(context.Background(), "/opt/hero/current", projectRoot)

	wantArgs := []string{"/opt/hero/current", "sync", "import", "--refresh", "--no-report"}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Fatalf("command args = %#v, want %#v", cmd.Args, wantArgs)
	}
	if cmd.Dir != projectRoot {
		t.Fatalf("command dir = %q, want %q", cmd.Dir, projectRoot)
	}
}

func TestImportRefresherRefreshRunsOneBulkImportAndPublishes(t *testing.T) {
	bus := NewEventBus()
	id, events := bus.Subscribe(1)
	defer bus.Unsubscribe(id)

	var calls int
	r := &ImportRefresher{
		projectRoot: "/workspace/project",
		slug:        "project",
		bus:         bus,
		runImport: func(_ context.Context, root string) (string, error) {
			calls++
			if root != "/workspace/project" {
				t.Fatalf("project root = %q", root)
			}
			return "Imported: 2, Skipped: 8", nil
		},
	}

	r.refresh(context.Background())

	if calls != 1 {
		t.Fatalf("bulk import calls = %d, want 1", calls)
	}
	select {
	case event := <-events:
		if event.Type != EventIndexRebuilt || event.Project != "project" {
			t.Fatalf("event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected index rebuilt event after successful bulk import")
	}
}

func TestImportRefresherRefreshFailureDoesNotPublish(t *testing.T) {
	bus := NewEventBus()
	id, events := bus.Subscribe(1)
	defer bus.Unsubscribe(id)

	r := &ImportRefresher{
		projectRoot: t.TempDir(),
		slug:        "project",
		bus:         bus,
		runImport: func(context.Context, string) (string, error) {
			return "jira unavailable", errors.New("exit status 1")
		},
	}

	r.refresh(context.Background())

	select {
	case event := <-events:
		t.Fatalf("unexpected success event after failed import: %+v", event)
	default:
	}
}

func TestImportRefresherRunStopsAndCancelsInFlightImport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	stopped := make(chan struct{})
	r := &ImportRefresher{
		projectRoot: t.TempDir(),
		slug:        "project",
		interval:    time.Hour,
		runImport: func(ctx context.Context, _ string) (string, error) {
			close(started)
			<-ctx.Done()
			close(stopped)
			return "", ctx.Err()
		},
	}

	done := make(chan struct{})
	go func() {
		r.run(ctx)
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("bulk import did not start")
	}
	cancel()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("in-flight bulk import was not cancelled")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("refresh loop did not stop")
	}
}

func TestServerAutoRefreshHasNoPerTicketTrackerPath(t *testing.T) {
	path := filepath.Join("refresh.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	source := string(content)
	for _, forbidden := range []string{
		"internal/tracker",
		"GetIssue(",
		"spec.Discover(",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("server auto-refresh must not contain private per-ticket sync path %q", forbidden)
		}
	}
}

func TestFormatImportOutputBoundsDaemonLogs(t *testing.T) {
	if got := formatImportOutput("  "); got != "" {
		t.Fatalf("empty output formatted as %q", got)
	}
	long := strings.Repeat("x", 2100)
	got := formatImportOutput(long)
	if len(got) != 2001 || !strings.HasPrefix(got, "\n") {
		t.Fatalf("bounded output length = %d, want 2001", len(got))
	}
}
