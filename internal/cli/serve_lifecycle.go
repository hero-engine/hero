package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/hero-engine/hero/internal/serve"
	"github.com/spf13/cobra"
)

var (
	serveStopPort   int
	serveStatusPort int
	serveStatusJSON bool
	serveForce      bool
)

// stopTimeout is the SIGTERM grace period before escalating to SIGKILL.
const stopTimeout = 5 * time.Second

var serveStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running hero daemon",
	Long: `Stops a running hero daemon by reading its PID file, sending SIGTERM,
waiting up to 5 seconds, and escalating to SIGKILL if needed.

Idempotent: prints a friendly message and exits 0 when no daemon is
running or when the PID file is stale.`,
	RunE: runServeStop,
}

var serveStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the running hero daemon's status",
	Long: `Prints the daemon's PID, port, uptime, version, and the list of served
projects.

Exits 1 when no daemon is running. Works from any directory.`,
	RunE: runServeStatus,
}

func init() {
	serveStopCmd.Flags().IntVar(&serveStopPort, "port", 0, "target daemon port (default 7437)")
	serveStatusCmd.Flags().IntVar(&serveStatusPort, "port", 0, "target daemon port (default 7437)")
	serveStatusCmd.Flags().BoolVar(&serveStatusJSON, "json", false, "emit raw /api/status JSON for scripting")
	serveCmd.AddCommand(serveStopCmd)
	serveCmd.AddCommand(serveStatusCmd)
}

func runServeStop(cmd *cobra.Command, args []string) error {
	return stopDaemon(serveStopPort, os.Stdout)
}

// stopDaemon is shared by `hero serve stop` and `hero serve --force`.
// out is where user-facing messages go (stdout for stop, stderr when
// invoked from --force so it doesn't pollute scripting output).
func stopDaemon(port int, out *os.File) error {
	resolvedPort := port
	if resolvedPort == 0 {
		resolvedPort = serve.DefaultPort
	}

	info, pidPath, err := serve.ReadPIDFile(resolvedPort)
	if err != nil {
		return fmt.Errorf("reading pid file: %w", err)
	}
	if info == nil {
		fmt.Fprintln(out, "no hero daemon is running")
		return nil
	}

	// Confirm the PID is actually a hero daemon by probing /api/status.
	// On a fresh stop this also catches PID-reuse where another process
	// happens to hold the recorded PID.
	if !serve.IsProcessAlive(info.PID) {
		if rmErr := os.Remove(pidPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return fmt.Errorf("removing stale pid file: %w", rmErr)
		}
		fmt.Fprintf(out, "cleaned up stale pid file at %s\n", pidPath)
		return nil
	}

	// Send SIGTERM.
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", info.PID, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to pid %d: %w", info.PID, err)
	}

	// Wait up to stopTimeout for the daemon to free the port and exit.
	deadline := time.Now().Add(stopTimeout)
	for time.Now().Before(deadline) {
		if !serve.IsProcessAlive(info.PID) && !serve.PortListenerHeld(info.Port) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Escalate if still alive.
	if serve.IsProcessAlive(info.PID) {
		fmt.Fprintf(out, "pid %d did not exit after %s, sending SIGKILL\n", info.PID, stopTimeout)
		if err := proc.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("sending SIGKILL to pid %d: %w", info.PID, err)
		}
		// Brief settle window for the kernel to release the port.
		for i := 0; i < 10; i++ {
			if !serve.IsProcessAlive(info.PID) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Best-effort PID file cleanup (server may have already removed it
	// during its own shutdown path).
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing pid file: %w", err)
	}
	fmt.Fprintf(out, "stopped hero daemon (pid %d, port %d)\n", info.PID, info.Port)
	return nil
}

func runServeStatus(cmd *cobra.Command, args []string) error {
	resolvedPort := serveStatusPort
	if resolvedPort == 0 {
		resolvedPort = serve.DefaultPort
	}

	info, pidPath, err := serve.ReadPIDFile(resolvedPort)
	if err != nil {
		return fmt.Errorf("reading pid file: %w", err)
	}
	if info == nil {
		fmt.Fprintln(os.Stderr, "no hero daemon is running")
		os.Exit(1)
		return nil
	}

	// Probe /api/status for live info.
	url := fmt.Sprintf("http://127.0.0.1:%d/api/status", info.Port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, httpErr := client.Get(url)
	if httpErr != nil || resp.StatusCode != http.StatusOK {
		// HTTP failed — the daemon is wedged or the PID file is stale.
		if resp != nil {
			resp.Body.Close()
		}
		if !serve.IsProcessAlive(info.PID) {
			fmt.Fprintf(os.Stderr, "pid file at %s is stale (pid %d is dead)\n", pidPath, info.PID)
			fmt.Fprintln(os.Stderr, "run `hero serve stop` to clean it up.")
			os.Exit(1)
			return nil
		}
		fmt.Fprintf(os.Stderr, "hero daemon at pid %d is not responding on :%d\n", info.PID, info.Port)
		fmt.Fprintln(os.Stderr, "run `hero serve --force` to replace it.")
		os.Exit(1)
		return nil
	}
	defer resp.Body.Close()

	var status serve.DaemonStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("parsing /api/status: %w", err)
	}

	if serveStatusJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	fmt.Printf("hero daemon: running\n")
	fmt.Printf("  pid:       %d\n", status.PID)
	fmt.Printf("  port:      %d\n", status.Port)
	fmt.Printf("  version:   %s\n", status.Version)
	fmt.Printf("  uptime:    %s\n", formatUptime(status.UptimeSeconds))
	fmt.Printf("  projects:  %d\n", status.ProjectCount)
	for _, p := range status.Projects {
		fmt.Printf("    - %-20s  %s\n", p.Slug, p.Path)
	}
	return nil
}

// formatUptime renders an integer-seconds duration in a human-friendly
// "Xh Ym Zs" / "Xm Ys" / "Xs" shape.
func formatUptime(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
}
