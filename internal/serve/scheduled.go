package serve

import (
	"context"
	"log"
	"os/exec"
	"time"
)

// ScheduledTasks runs periodic maintenance on the team server.
type ScheduledTasks struct {
	projectRoot string
	heroDir     string
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewScheduledTasks creates the scheduled task runner.
func NewScheduledTasks(projectRoot, heroDir string) *ScheduledTasks {
	ctx, cancel := context.WithCancel(context.Background())
	return &ScheduledTasks{
		projectRoot: projectRoot,
		heroDir:     heroDir,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the scheduled task loops.
func (st *ScheduledTasks) Start() {
	go st.reconcileLoop()
	go st.suggestionsLoop()
	go st.indexLoop()
	log.Printf("hero team: scheduled tasks started (reconcile: hourly, suggestions: weekly, index: 15min)")
}

// Stop cancels all scheduled tasks.
func (st *ScheduledTasks) Stop() {
	st.cancel()
}

func (st *ScheduledTasks) reconcileLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-st.ctx.Done():
			return
		case <-ticker.C:
			st.runCommand("check", "--reconcile")
		}
	}
}

func (st *ScheduledTasks) suggestionsLoop() {
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	// Also run once at startup after a delay
	go func() {
		select {
		case <-st.ctx.Done():
			return
		case <-time.After(5 * time.Minute):
			st.runCommand("suggestions", "--top", "10")
		}
	}()

	for {
		select {
		case <-st.ctx.Done():
			return
		case <-ticker.C:
			st.runCommand("suggestions", "--top", "10")
		}
	}
}

func (st *ScheduledTasks) indexLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-st.ctx.Done():
			return
		case <-ticker.C:
			st.runCommand("index")
		}
	}
}

func (st *ScheduledTasks) runCommand(args ...string) {
	cmd := exec.Command("hero", args...)
	cmd.Dir = st.projectRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("hero team scheduled: hero %v: %v\n%s", args, err, string(out))
	}
}
