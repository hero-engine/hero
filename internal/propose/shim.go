package propose

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ShimConfig drives ScanAndPost, the stdout-tailing logic used by the
// `hero agent propose-shim` command.
type ShimConfig struct {
	// DaemonURL is the base URL of the hero daemon, e.g. http://127.0.0.1:7437.
	DaemonURL string
	// Project is the project slug the proposals route through.
	Project string
	// SessionID is the session the agent runs under. The shim
	// propagates this as the env-var HERO_SESSION_ID to the agent and
	// uses it to build the ingest URL.
	SessionID string
	// HTTPClient is optional; defaults to http.DefaultClient.
	HTTPClient *http.Client
	// PassThrough is where non-proposal stdout lines are forwarded
	// (typically os.Stdout). If nil, lines are dropped.
	PassThrough io.Writer
	// ErrorLog is where invalid envelopes and POST failures are
	// reported (typically os.Stderr). If nil, errors are dropped.
	ErrorLog io.Writer
}

// ScanAndPost reads stdout-style lines from r line-by-line, strips
// the HERO-PROPOSAL: prefix from any matching lines, validates them
// as envelopes, and POSTs each one to the daemon. Non-prefixed lines
// pass through to cfg.PassThrough. The function returns when r is
// exhausted or ctx is cancelled.
func ScanAndPost(ctx context.Context, r io.Reader, cfg ShimConfig) error {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}

	scanner := bufio.NewScanner(r)
	// Allow long proposal lines — proposed content blocks can be sizable.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, HeroProposalPrefix) {
			if cfg.PassThrough != nil {
				fmt.Fprintln(cfg.PassThrough, line)
			}
			continue
		}
		payload := strings.TrimPrefix(line, HeroProposalPrefix)
		env, err := ParseEnvelope([]byte(payload))
		if err != nil {
			logErr(cfg.ErrorLog, fmt.Sprintf("hero agent propose-shim: invalid envelope: %v", err))
			continue
		}
		// Backfill session_id if the agent didn't set it.
		if env.SessionID == "" {
			env.SessionID = cfg.SessionID
		}
		if err := postEnvelope(ctx, cfg, env); err != nil {
			logErr(cfg.ErrorLog, fmt.Sprintf("hero agent propose-shim: post failed: %v", err))
		}
	}
	return scanner.Err()
}

func postEnvelope(ctx context.Context, cfg ShimConfig, env *Envelope) error {
	url := fmt.Sprintf("%s/api/%s/sessions/%s/proposals/ingest",
		strings.TrimRight(cfg.DaemonURL, "/"), cfg.Project, env.SessionID)
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func logErr(w io.Writer, msg string) {
	if w == nil {
		return
	}
	fmt.Fprintln(w, msg)
}
