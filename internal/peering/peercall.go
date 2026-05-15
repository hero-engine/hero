package peering

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/spec"
)

// Default budget caps for sync peer calls. Locked at Phase 2: advisory
// is cheap probe-shaped work, spec-out is full /design under a subagent.
// Both are documented in the cross-repo-peering spec.
const (
	DefaultAdvisoryTurns   = 20
	DefaultAdvisoryTokens  = 50_000
	DefaultSpecOutTurns    = 50
	DefaultSpecOutTokens   = 150_000
	DefaultCallTimeout     = 10 * time.Minute
)

// Default subagent command. Overridable via hero.json
// peering.subagent.command.
const DefaultSubagentCommand = "claude"

// resultFenceRE matches the structured result block emitted by the
// peer subagent. Tolerates leading/trailing whitespace; capture group 1
// is the YAML body.
var resultFenceRE = regexp.MustCompile(`(?s)<peer-call-result>\s*(.*?)\s*</peer-call-result>`)

// CallOptions configures a single sync peer call.
type CallOptions struct {
	// PeerAlias is the local alias of the target peer. Required.
	PeerAlias string

	// Mode is advisory | spec-out (v1) or full (deferred to v2).
	Mode contractpeering.PeerCallMode

	// Prompt is the user-supplied prompt for the peer subagent.
	Prompt string

	// Budget caps consumption. Zero values fall back to the per-mode
	// defaults declared in DefaultAdvisory* / DefaultSpecOut* above.
	Budget contractpeering.BudgetSpec

	// RelatedSpec is the originator-side slug that anchors trail
	// entries. Optional — without it the call is a one-off probe.
	RelatedSpec string

	// Reason is the free-form rationale captured at call time.
	Reason string

	// At fixes the wall-clock time of the call. Zero → time.Now().
	// Exposed for tests.
	At time.Time

	// DryRun, when true, builds and prints the envelope to Stdout
	// (when non-nil) without spawning the subagent. Returns a synthetic
	// PeerCallResult with mode-appropriate empty fields.
	DryRun bool

	// Stdout is where the dry-run envelope is written. nil → os.Stdout.
	// Used only when DryRun is true. Set to io.Discard to suppress.
	Stdout io.Writer

	// Now is a test seam for time.Now. nil → time.Now.
	Now func() time.Time
}

// CallResult bundles the peer's structured result with the metadata
// hero records on the originator side.
type CallResult struct {
	// Result is the parsed envelope returned by the peer subagent.
	Result contractpeering.PeerCallResult

	// CallID is the ULID-like ID for this call (echoed in Result).
	CallID string

	// PeerID is the canonical id of the target peer.
	PeerID string

	// PeerPath is the absolute filesystem path of the peer workspace.
	PeerPath string

	// EnvelopeJSON is the prompt envelope that was piped to the
	// subagent. Captured for audit / dry-run inspection.
	EnvelopeJSON string

	// Stdout is the full raw stdout from the subagent. Captured for
	// debug — the structured Result is what callers should consume.
	Stdout string
}

// Call performs a sync peer call: prepare envelope, spawn subagent in
// the peer's workspace, parse the structured result, persist trail
// entries and events on the originator side.
//
// Returns an error without writing any state when:
//   - peer is not configured / unreachable
//   - peer has no peer_id minted yet
//   - subagent command is missing or fails
//   - structured result block is missing or malformed
//
// On success, the call:
//   - records a trail entry on the related spec (if any)
//   - emits peer.call.invoked and peer.call.completed events
//   - for spec-out: moves related spec status to awaiting_peer
//
// NEVER writes on the peer side. Advisory mode is no-write by design;
// spec-out mode writes the new spec but only via the subagent itself
// running the peer's /design flow.
func Call(projectRoot string, opts CallOptions) (*CallResult, error) {
	if opts.PeerAlias == "" {
		return nil, errors.New("peer alias required")
	}
	if opts.Mode == "" {
		return nil, errors.New("mode required (advisory | spec-out)")
	}
	if opts.Mode == contractpeering.PeerCallFull {
		return nil, errors.New("mode=full is v2 — not implemented")
	}
	if opts.Mode != contractpeering.PeerCallAdvisory && opts.Mode != contractpeering.PeerCallSpecOut {
		return nil, fmt.Errorf("unknown mode %q (advisory | spec-out)", opts.Mode)
	}
	if strings.TrimSpace(opts.Prompt) == "" {
		return nil, errors.New("prompt required")
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.PeerID == "" {
		return nil, fmt.Errorf("workspace has no peer_id — run `hero` once to mint")
	}

	peerPath, err := cfg.ResolveRepoPath(projectRoot, opts.PeerAlias)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(peerPath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("peer %q path %q not a directory", opts.PeerAlias, peerPath)
	}
	peerHeroDir := filepath.Join(peerPath, cfg.Folder)
	if _, err := os.Stat(peerHeroDir); err != nil {
		return nil, fmt.Errorf("peer %q has no %s directory — run `hero init` in %s", opts.PeerAlias, cfg.Folder, peerPath)
	}
	peerCfgPath := filepath.Join(peerHeroDir, "hero.json")
	peerPeerID := readPeerIDFromJSON(peerCfgPath)
	if peerPeerID == "" {
		return nil, fmt.Errorf("peer %q has no peer_id in %s — run `hero` once in %s",
			opts.PeerAlias, filepath.Join(cfg.Folder, "hero.json"), peerPath)
	}

	// Apply mode-default budgets.
	budget := applyBudgetDefaults(opts.Mode, opts.Budget)

	now := opts.Now
	if now == nil {
		now = time.Now
	}
	at := opts.At
	if at.IsZero() {
		at = now().UTC()
	}
	callID := newCallID()
	atCommit := readGitHEAD(projectRoot)

	req := contractpeering.PeerCallRequest{
		ContractsVersion: contractpeering.PeeringContractsVersion,
		CallID:           callID,
		OriginPeerID:     cfg.PeerID,
		TargetPeerID:     peerPeerID,
		Mode:             opts.Mode,
		Prompt:           opts.Prompt,
		Budget:           budget,
		RelatedSpec:      opts.RelatedSpec,
		Reason:           opts.Reason,
		At:               at,
		AtCommit:         atCommit,
	}

	envelope := renderEnvelope(req, opts.PeerAlias, workspaceDisplayName(cfg, projectRoot))

	heroDir := cfg.HeroDir(projectRoot)
	// Emit invoked event up front so a crashed call still leaves a
	// trail in the events log.
	_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
		Timestamp: at,
		Type:      string(contractpeering.EventCallInvoked),
		Agent:     "hero",
		Slug:      opts.RelatedSpec,
		Message: fmt.Sprintf("peer call %s mode=%s target=%s call_id=%s",
			opts.Mode, opts.Mode, opts.PeerAlias, callID),
	})

	out := &CallResult{
		CallID:       callID,
		PeerID:       peerPeerID,
		PeerPath:     peerPath,
		EnvelopeJSON: envelope,
	}

	if opts.DryRun {
		var w io.Writer = opts.Stdout
		if w == nil {
			w = os.Stdout
		}
		fmt.Fprintln(w, "--- peer-call envelope (dry-run, not dispatched) ---")
		fmt.Fprintln(w, envelope)
		fmt.Fprintln(w, "--- end envelope ---")
		out.Result = contractpeering.PeerCallResult{
			ContractsVersion: contractpeering.PeeringContractsVersion,
			CallID:           callID,
			Mode:             opts.Mode,
			Kind:             contractpeering.ResultFindings,
			Findings:         "(dry-run — subagent not dispatched)",
			At:               now().UTC(),
		}
		return out, nil
	}

	subCfg := resolveSubagentConfig(cfg)
	stdout, runErr := runSubagent(peerPath, subCfg, envelope, budget)
	out.Stdout = stdout
	if runErr != nil {
		// Emit a completed event with the error captured.
		_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
			Timestamp: now().UTC(),
			Type:      string(contractpeering.EventCallCompleted),
			Agent:     "hero",
			Slug:      opts.RelatedSpec,
			Message:   fmt.Sprintf("peer call failed: %v (call_id %s)", runErr, callID),
		})
		return out, fmt.Errorf("subagent: %w", runErr)
	}

	result, parseErr := parseResultBlock(stdout)
	if parseErr != nil {
		_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
			Timestamp: now().UTC(),
			Type:      string(contractpeering.EventCallCompleted),
			Agent:     "hero",
			Slug:      opts.RelatedSpec,
			Message:   fmt.Sprintf("peer call result unparseable: %v (call_id %s)", parseErr, callID),
		})
		return out, fmt.Errorf("parse subagent result: %w", parseErr)
	}
	// Stamp through any fields the subagent didn't fill.
	if result.CallID == "" {
		result.CallID = callID
	}
	if result.Mode == "" {
		result.Mode = opts.Mode
	}
	if result.ContractsVersion == 0 {
		result.ContractsVersion = contractpeering.PeeringContractsVersion
	}
	if result.At.IsZero() {
		result.At = now().UTC()
	}
	out.Result = result

	// Persist trail + status on the originator side.
	if err := recordOriginatorSide(projectRoot, cfg, opts, req, result, peerPeerID, atCommit); err != nil {
		// We have a successful peer call but couldn't update local
		// state. Surface to the caller; the events.log still has the
		// invoked entry.
		_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
			Timestamp: now().UTC(),
			Type:      string(contractpeering.EventCallCompleted),
			Agent:     "hero",
			Slug:      opts.RelatedSpec,
			Message:   fmt.Sprintf("peer call returned but local persist failed: %v (call_id %s)", err, callID),
		})
		return out, fmt.Errorf("record originator side: %w", err)
	}

	_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
		Timestamp: result.At,
		Type:      string(contractpeering.EventCallCompleted),
		Agent:     "hero",
		Slug:      opts.RelatedSpec,
		Message: fmt.Sprintf("peer call ok mode=%s target=%s kind=%s call_id=%s",
			opts.Mode, opts.PeerAlias, result.Kind, callID),
	})

	return out, nil
}

// applyBudgetDefaults fills in per-mode defaults when the caller left
// budget fields zero. Caller-supplied non-zero values are preserved.
func applyBudgetDefaults(mode contractpeering.PeerCallMode, b contractpeering.BudgetSpec) contractpeering.BudgetSpec {
	switch mode {
	case contractpeering.PeerCallAdvisory:
		if b.Turns == 0 {
			b.Turns = DefaultAdvisoryTurns
		}
		if b.Tokens == 0 {
			b.Tokens = DefaultAdvisoryTokens
		}
	case contractpeering.PeerCallSpecOut:
		if b.Turns == 0 {
			b.Turns = DefaultSpecOutTurns
		}
		if b.Tokens == 0 {
			b.Tokens = DefaultSpecOutTokens
		}
	}
	return b
}

// resolveSubagentConfig returns the effective subagent invocation
// settings — config block overrides defaults; missing fields use the
// per-field defaults.
func resolveSubagentConfig(cfg config.Config) config.SubagentConfig {
	sc := config.SubagentConfig{
		Command:        DefaultSubagentCommand,
		Args:           []string{"-p"},
		EnvPassthrough: []string{"ANTHROPIC_API_KEY", "PATH", "HOME", "USER", "LANG", "LC_ALL", "TERM"},
	}
	if cfg.Peering != nil && cfg.Peering.Subagent != nil {
		if cfg.Peering.Subagent.Command != "" {
			sc.Command = cfg.Peering.Subagent.Command
		}
		if cfg.Peering.Subagent.Args != nil {
			sc.Args = append([]string{}, cfg.Peering.Subagent.Args...)
		}
		if cfg.Peering.Subagent.EnvPassthrough != nil {
			sc.EnvPassthrough = append([]string{}, cfg.Peering.Subagent.EnvPassthrough...)
		}
	}
	return sc
}

// runSubagent exec's the configured LLM CLI with cwd set to the peer
// workspace, pipes the envelope on stdin, captures stdout. Honors
// DefaultCallTimeout. Returns (stdout, err).
func runSubagent(peerPath string, sub config.SubagentConfig, envelope string, budget contractpeering.BudgetSpec) (string, error) {
	if sub.Command == "" {
		return "", errors.New("subagent command is empty — set peering.subagent.command in hero.json")
	}
	if _, err := exec.LookPath(sub.Command); err != nil {
		return "", fmt.Errorf("subagent command %q not found on PATH: %w", sub.Command, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultCallTimeout)
	defer cancel()

	// Build args. We append no flags of our own: the budget is
	// advisory on the subagent side and is documented in the envelope.
	cmd := exec.CommandContext(ctx, sub.Command, sub.Args...)
	cmd.Dir = peerPath
	cmd.Stdin = strings.NewReader(envelope)

	// Restrict env to the passthrough set, plus a HERO_PEER_CALL marker
	// so the subagent can detect it's running under hero peer call.
	env := []string{
		"HERO_PEER_CALL=1",
		fmt.Sprintf("HERO_PEER_CALL_BUDGET_TURNS=%d", budget.Turns),
		fmt.Sprintf("HERO_PEER_CALL_BUDGET_TOKENS=%d", budget.Tokens),
	}
	for _, k := range sub.EnvPassthrough {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Surface stderr in the error for diagnosability — the
		// subagent's last words are usually the most useful clue.
		errTail := strings.TrimSpace(stderr.String())
		if len(errTail) > 2000 {
			errTail = "..." + errTail[len(errTail)-2000:]
		}
		if ctx.Err() == context.DeadlineExceeded {
			return stdout.String(), fmt.Errorf("subagent timed out after %s: %s", DefaultCallTimeout, errTail)
		}
		return stdout.String(), fmt.Errorf("subagent exited with error: %w (stderr: %s)", err, errTail)
	}
	return stdout.String(), nil
}

// parseResultBlock locates the <peer-call-result>...</peer-call-result>
// fence in subagent stdout and YAML-unmarshals the body into a
// PeerCallResult. Returns a clear error when the fence is missing or
// the body fails to parse — neither case should write any state.
func parseResultBlock(stdout string) (contractpeering.PeerCallResult, error) {
	m := resultFenceRE.FindStringSubmatch(stdout)
	if len(m) < 2 {
		preview := stdout
		if len(preview) > 800 {
			preview = preview[:800] + "..."
		}
		return contractpeering.PeerCallResult{}, fmt.Errorf("no <peer-call-result> fence in subagent output (first 800 bytes: %s)", preview)
	}
	body := strings.TrimSpace(m[1])
	var out contractpeering.PeerCallResult
	if err := yaml.Unmarshal([]byte(body), &out); err != nil {
		return contractpeering.PeerCallResult{}, fmt.Errorf("unmarshal result block: %w", err)
	}
	return out, nil
}

// recordOriginatorSide writes the trail entry on the related spec
// (when set) and flips status to awaiting_peer for spec-out mode.
// Advisory mode appends a trail entry but does NOT change spec status.
func recordOriginatorSide(
	projectRoot string,
	cfg config.Config,
	opts CallOptions,
	req contractpeering.PeerCallRequest,
	result contractpeering.PeerCallResult,
	peerPeerID string,
	atCommit string,
) error {
	if opts.RelatedSpec == "" {
		return nil
	}
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discover specs: %w", err)
	}
	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == opts.RelatedSpec {
			target = s
			break
		}
	}
	if target == nil {
		return fmt.Errorf("related spec %q not found in this workspace", opts.RelatedSpec)
	}

	// Translate call mode to trail mode. Advisory → advisory, spec-out
	// → spec-out. Full is rejected upstream.
	tmode := contractpeering.ModeAdvisory
	if req.Mode == contractpeering.PeerCallSpecOut {
		tmode = contractpeering.ModeSpecOut
	}

	peerSpec := ""
	if result.SpecSlug != "" {
		peerSpec = opts.PeerAlias + "/" + result.SpecSlug
	}

	entry := contractpeering.TrailEntry{
		At:               result.At,
		Direction:        contractpeering.DirectionOut,
		PeerAliasDisplay: opts.PeerAlias,
		PeerID:           peerPeerID,
		Mode:             tmode,
		OriginatingSpec:  opts.RelatedSpec,
		PeerSpec:         peerSpec,
		PeerStatus:       result.PeerStatus,
		AtCommit:         atCommit,
		ResultRef:        result.CallID,
		Reason:           opts.Reason,
	}

	data, err := os.ReadFile(target.Path)
	if err != nil {
		return fmt.Errorf("read related spec: %w", err)
	}
	updated := string(data)

	// Spec-out → flip status to awaiting_peer.
	if req.Mode == contractpeering.PeerCallSpecOut {
		updated = spec.SetFrontmatterField(updated, "status", string(spec.StatusAwaitingPeer))
	}

	updated = AppendTrailToContent(updated, entry)
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write related spec: %w", err)
	}
	return nil
}

// renderEnvelope produces the structured prompt the subagent receives
// on stdin. It contains:
//   - the contractual request (YAML, for machine parsing)
//   - mode-specific instructions (plain prose, for the model)
//
// The subagent's job is to honor the instructions and emit a
// `<peer-call-result>...</peer-call-result>` block to stdout.
func renderEnvelope(req contractpeering.PeerCallRequest, peerAlias, originDisplay string) string {
	var b strings.Builder
	b.WriteString("# Hero peer call\n\n")
	b.WriteString("You are running as a subagent invoked by `hero peer call` from a sibling\n")
	b.WriteString("Hero workspace. Your cwd is **this peer workspace**. Load this workspace's\n")
	b.WriteString("Hero context (conventions, decisions, code knowledge) — use `hero context`,\n")
	b.WriteString("`hero search`, and the local files as needed.\n\n")
	b.WriteString("## Caller\n\n")
	fmt.Fprintf(&b, "- Origin workspace: %s (peer_id %s)\n", originDisplay, req.OriginPeerID)
	fmt.Fprintf(&b, "- Target alias on caller's side: %s\n", peerAlias)
	fmt.Fprintf(&b, "- Call ID: %s\n", req.CallID)
	fmt.Fprintf(&b, "- Mode: %s\n", req.Mode)
	if req.RelatedSpec != "" {
		fmt.Fprintf(&b, "- Caller's related spec: %s\n", req.RelatedSpec)
	}
	if req.Reason != "" {
		fmt.Fprintf(&b, "- Caller's reason: %s\n", req.Reason)
	}
	fmt.Fprintf(&b, "- Budget: %d turns, %d tokens (advisory cap — respect it)\n\n", req.Budget.Turns, req.Budget.Tokens)

	b.WriteString("## Mode-specific instructions\n\n")
	switch req.Mode {
	case contractpeering.PeerCallAdvisory:
		b.WriteString("**Advisory mode.** Investigate the prompt against this workspace's\n")
		b.WriteString("loaded context. Return findings. **Do not write any spec, code,\n")
		b.WriteString("or non-event-log state.** No `hero design`, no `hero deliver`, no\n")
		b.WriteString("edits to disk. Pure read-only investigation.\n\n")
	case contractpeering.PeerCallSpecOut:
		b.WriteString("**Spec-out mode.** Run this workspace's `/design` flow on the prompt.\n")
		b.WriteString("Produce a spec at `.hero/planning/<type>/<slug>/spec.md` with status\n")
		b.WriteString("`planning` and a `received_from:` block referencing the caller's\n")
		b.WriteString("peer_id and related_spec. Bake this workspace's conventions into the\n")
		b.WriteString("design. The spec, not the findings, is the deliverable.\n\n")
	}

	b.WriteString("## Prompt\n\n")
	b.WriteString(req.Prompt)
	if !strings.HasSuffix(req.Prompt, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("## Return format\n\n")
	b.WriteString("Emit a single block to stdout containing your structured result,\n")
	b.WriteString("fenced exactly like this (YAML body):\n\n")
	b.WriteString("```\n")
	b.WriteString("<peer-call-result>\n")
	switch req.Mode {
	case contractpeering.PeerCallAdvisory:
		b.WriteString("kind: findings\n")
		b.WriteString("findings: |\n")
		b.WriteString("  <your investigation summary + supporting details>\n")
	case contractpeering.PeerCallSpecOut:
		b.WriteString("kind: spec-ref\n")
		b.WriteString("spec_slug: <the slug you wrote>\n")
		b.WriteString("peer_status: planning\n")
	}
	b.WriteString("budget_consumed:\n")
	b.WriteString("  turns: <actual>\n")
	b.WriteString("  tokens: <actual>\n")
	b.WriteString("</peer-call-result>\n")
	b.WriteString("```\n\n")
	b.WriteString("The fence must appear verbatim — Hero parses it with a regex.\n")

	// Also embed the machine-readable request as a final reference
	// block in case the subagent prefers structured input.
	b.WriteString("\n## Request envelope (raw)\n\n")
	b.WriteString("```yaml\n")
	if data, err := yaml.Marshal(req); err == nil {
		b.Write(data)
	}
	b.WriteString("```\n")

	return b.String()
}

// newCallID returns a 26-char ULID-like identifier. We don't depend on
// a ULID library here — hex-of-time + random is sufficient for a join
// key. Lexicographically sortable by call time.
func newCallID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	ts := time.Now().UTC().UnixNano()
	return fmt.Sprintf("%016x%s", ts, hex.EncodeToString(buf[:]))
}
