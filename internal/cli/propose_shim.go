package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/hero-engine/hero/internal/propose"
	"github.com/spf13/cobra"
)

// proposeShimCmd is the `hero agent propose-shim` subcommand. It runs
// an agent subprocess, tails its stdout for HERO-PROPOSAL: NDJSON
// lines, validates each envelope, and POSTs it to the hero daemon's
// inline-propose ingest endpoint. Non-proposal stdout passes through
// unchanged so the user still sees agent chatter.
//
// See docs/contracts/inline-propose-v1.md for the wire contract.
var proposeShimCmd = &cobra.Command{
	Use:   "propose-shim [-- <command> [args...]]",
	Short: "Tail an agent's stdout for HERO-PROPOSAL: NDJSON and forward to the daemon",
	Long: `propose-shim wraps an agent invocation and converts inline-propose
output into REST calls against the local hero daemon.

If a wrapped command is given after ` + "`--`" + `, the shim runs it as a child
process and tails its stdout. With no wrapped command, the shim reads
from its own stdin (useful for piping and testing).

The shim exports HERO_SESSION_ID into the child's environment so the
agent inherits the session identifier the daemon uses to scope
proposals.`,
	RunE: runProposeShim,
}

var (
	proposeShimDaemonURL string
	proposeShimProject   string
	proposeShimSessionID string
)

func init() {
	proposeShimCmd.Flags().StringVar(&proposeShimDaemonURL, "daemon", "http://127.0.0.1:7437", "hero daemon URL")
	proposeShimCmd.Flags().StringVar(&proposeShimProject, "project", "", "project slug (defaults to current workspace name)")
	proposeShimCmd.Flags().StringVar(&proposeShimSessionID, "session", "", "session id (defaults to $HERO_SESSION_ID, then a fresh value)")
	agentCmd.AddCommand(proposeShimCmd)
}

func runProposeShim(cmd *cobra.Command, args []string) error {
	project := proposeShimProject
	if project == "" {
		project = defaultProposeProject()
	}
	sessionID := proposeShimSessionID
	if sessionID == "" {
		sessionID = os.Getenv("HERO_SESSION_ID")
	}
	if sessionID == "" {
		// Mint a transient session id. Not cryptographically
		// significant — it just needs to be unique within the daemon
		// process for the lifetime of this shim run.
		sessionID = fmt.Sprintf("shim-%d", os.Getpid())
	}

	shimCfg := propose.ShimConfig{
		DaemonURL:   proposeShimDaemonURL,
		Project:     project,
		SessionID:   sessionID,
		PassThrough: os.Stdout,
		ErrorLog:    os.Stderr,
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	if len(args) == 0 {
		return propose.ScanAndPost(ctx, os.Stdin, shimCfg)
	}

	return runWrappedAgent(ctx, args, sessionID, shimCfg)
}

func runWrappedAgent(ctx context.Context, args []string, sessionID string, shimCfg propose.ShimConfig) error {
	child := exec.CommandContext(ctx, args[0], args[1:]...)

	// Propagate environment + the session id the daemon expects.
	env := append([]string{}, os.Environ()...)
	env = append(env, "HERO_SESSION_ID="+sessionID)
	env = append(env, "HERO_INLINE_PROPOSE=1")
	child.Env = env

	stdout, err := child.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	child.Stderr = os.Stderr
	child.Stdin = os.Stdin

	if err := child.Start(); err != nil {
		return fmt.Errorf("start agent: %w", err)
	}

	var scanErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanErr = propose.ScanAndPost(ctx, stdout, shimCfg)
		// Drain anything left.
		if scanErr == io.EOF {
			scanErr = nil
		}
	}()

	waitErr := child.Wait()
	wg.Wait()

	if scanErr != nil {
		fmt.Fprintf(os.Stderr, "hero agent propose-shim: scanner error: %v\n", scanErr)
	}

	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	return waitErr
}

func defaultProposeProject() string {
	root := findProjectRoot()
	if root == "" {
		return ""
	}
	// Match the slug `hero serve` uses when registering a single-project
	// context: the project root's directory base name.
	return filepath.Base(root)
}
