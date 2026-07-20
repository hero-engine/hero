package serve

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// bulkImportRunner runs the canonical tracker import for one project. Keeping
// this boundary injectable makes the daemon lifecycle testable without a live
// tracker or a child process.
type bulkImportRunner func(context.Context, string) (string, error)

// ImportRefresher handles periodic bulk import refresh for a project.
type ImportRefresher struct {
	projectRoot string
	slug        string
	bus         *EventBus
	interval    time.Duration
	cancel      context.CancelFunc
	runImport   bulkImportRunner
}

// StartImportRefresher starts a background goroutine that periodically runs
// the canonical bulk import-and-refresh workflow for a project. Returns nil if
// auto_refresh is disabled.
func StartImportRefresher(projectRoot, slug string, bus *EventBus) *ImportRefresher {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil
	}

	if cfg.Import == nil || !cfg.Import.AutoRefresh {
		return nil
	}

	interval := 30 * time.Minute
	if cfg.Import.RefreshInterval != "" {
		if d, err := time.ParseDuration(cfg.Import.RefreshInterval); err == nil && d > 0 {
			interval = d
		}
	}

	// Don't allow intervals shorter than 5 minutes.
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &ImportRefresher{
		projectRoot: projectRoot,
		slug:        slug,
		bus:         bus,
		interval:    interval,
		cancel:      cancel,
		runImport:   runCanonicalBulkImport,
	}

	go r.run(ctx)
	return r
}

// Stop cancels the refresh loop and any in-flight bulk import process.
func (r *ImportRefresher) Stop() {
	if r != nil && r.cancel != nil {
		r.cancel()
	}
}

func (r *ImportRefresher) run(ctx context.Context) {
	fmt.Fprintf(os.Stderr, "hero serve: bulk auto-refresh enabled for %s (interval: %s)\n", r.slug, r.interval)

	// Run once immediately on startup. Each cycle is synchronous, so refreshes
	// cannot overlap even when a tracker call runs longer than the interval.
	r.refresh(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refresh(ctx)
		}
	}
}

func (r *ImportRefresher) refresh(ctx context.Context) {
	output, err := r.runImport(ctx, r.projectRoot)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		fmt.Fprintf(os.Stderr, "hero serve: bulk auto-refresh failed for %s: %v%s\n",
			r.slug, err, formatImportOutput(output))
		return
	}

	if r.bus != nil {
		r.bus.Publish(Event{
			Type:    EventIndexRebuilt,
			Project: r.slug,
			Message: "bulk tracker auto-refresh completed",
		})
	}
}

// runCanonicalBulkImport delegates to the exact executable running the server.
// This avoids PATH/version drift and, more importantly, keeps query planning,
// per-type limits, discovery, deduplication, and refresh reconciliation on the
// single canonical `hero sync import` implementation. Deep evidence loading is
// intentionally absent from this path.
func runCanonicalBulkImport(ctx context.Context, projectRoot string) (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating running Hero executable: %w", err)
	}

	cmd := newBulkImportCommand(ctx, executable, projectRoot)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func newBulkImportCommand(ctx context.Context, executable, projectRoot string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, executable, "sync", "import", "--refresh", "--no-report")
	cmd.Dir = projectRoot
	return cmd
}

func formatImportOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	const maxLogBytes = 2000
	if len(output) > maxLogBytes {
		output = output[len(output)-maxLogBytes:]
	}
	return "\n" + output
}
