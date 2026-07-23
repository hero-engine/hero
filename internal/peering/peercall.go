package peering

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/attention/mail"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/config"
)

const (
	DefaultAdvisoryTurns  = 20
	DefaultAdvisoryTokens = 50_000
	DefaultSpecOutTurns   = 50
	DefaultSpecOutTokens  = 150_000
)

type CallOptions struct {
	PeerAlias      string
	Mode           contractpeering.PeerCallMode
	Prompt         string
	Budget         contractpeering.BudgetSpec
	RelatedSpec    string
	Reason         string
	IdempotencyKey string
	Wait           time.Duration
	PollInterval   time.Duration
	At             time.Time
	DryRun         bool
	Stdout         io.Writer
	Now            func() time.Time
	StateRoot      string
}

type CallResult struct {
	Result                  contractpeering.PeerCallResult `json:"legacy_result,omitempty"`
	CallID                  string                         `json:"call_id"`
	PeerID                  string                         `json:"peer_id"`
	PeerPath                string                         `json:"peer_path,omitempty"`
	EnvelopeJSON            string                         `json:"envelope_json,omitempty"`
	MessageID               string                         `json:"message_id"`
	ThreadID                string                         `json:"thread_id"`
	Status                  string                         `json:"status"`
	Response                *mail.ListedMessage            `json:"response,omitempty"`
	DeprecatedConfigIgnored bool                           `json:"deprecated_config_ignored,omitempty"`
}

func Call(projectRoot string, opts CallOptions) (*CallResult, error) {
	if opts.PeerAlias == "" {
		return nil, errors.New("peer alias required")
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
	peerPath, err := cfg.ResolveRepoPath(projectRoot, opts.PeerAlias)
	if err != nil {
		return nil, err
	}
	manifest, err := ReadPeerManifest(peerPath, cfg.Folder)
	if err != nil {
		return nil, err
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	at := opts.At
	if at.IsZero() {
		at = now().UTC()
	}
	budget := applyBudgetDefaults(opts.Mode, opts.Budget)
	key := opts.IdempotencyKey
	if key == "" {
		key = stablePeerCallKey(cfg.PeerID, opts.PeerAlias, string(opts.Mode), opts.Prompt)
	}
	kind := "peer.advisory"
	if opts.Mode == contractpeering.PeerCallSpecOut {
		kind = "peer.spec_out"
	}
	body, err := json.Marshal(map[string]any{
		"mode": opts.Mode, "prompt": opts.Prompt, "reason": opts.Reason,
		"related_spec": opts.RelatedSpec, "budget_hint": budget,
		"origin_commit":         readGitHEAD(projectRoot),
		"peer_manifest_version": manifest.ContractsVersion,
	})
	if err != nil {
		return nil, err
	}
	out := &CallResult{
		CallID: key, PeerID: manifest.Repo.PeerID, PeerPath: peerPath,
		Status: "queued", DeprecatedConfigIgnored: cfg.Peering != nil && cfg.Peering.Subagent != nil,
	}
	if opts.DryRun {
		preview := map[string]any{"recipient": opts.PeerAlias, "kind": kind, "subject": fmt.Sprintf("Peer %s request", opts.Mode), "body": string(body), "idempotency_key": key}
		encoded, _ := json.MarshalIndent(preview, "", "  ")
		out.EnvelopeJSON = string(encoded)
		w := opts.Stdout
		if w == nil {
			w = os.Stdout
		}
		_, _ = fmt.Fprintln(w, out.EnvelopeJSON)
		return out, nil
	}
	svc, err := projectMailService(projectRoot, opts.StateRoot, cfg)
	if err != nil {
		return nil, err
	}
	delivery, err := svc.Send(mail.SendRequest{
		RecipientAlias: opts.PeerAlias,
		Subject:        fmt.Sprintf("Peer %s request", opts.Mode),
		Body:           string(body),
		Kind:           kind,
		IdempotencyKey: key,
	})
	if err != nil {
		return nil, err
	}
	out.CallID = delivery.MessageID
	out.MessageID = delivery.MessageID
	out.ThreadID = delivery.ThreadID
	if opts.Wait <= 0 {
		return out, nil
	}
	interval := opts.PollInterval
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	deadline := time.Now().Add(opts.Wait)
	for {
		items, err := svc.Inbox("", false)
		if err != nil {
			return nil, err
		}
		for i := range items {
			if items[i].ThreadID == delivery.ThreadID && items[i].InReplyTo != "" && items[i].Sender.PeerID == manifest.Repo.PeerID {
				out.Response = &items[i]
				out.Status = "responded"
				return out, nil
			}
		}
		if !time.Now().Before(deadline) {
			out.Status = "pending"
			return out, nil
		}
		time.Sleep(interval)
	}
}

func stablePeerCallKey(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("peer-call:%x", hash.Sum(nil))
}

func projectMailService(projectRoot, stateRoot string, cfg config.Config) (*mail.Service, error) {
	var err error
	if stateRoot == "" {
		stateRoot, err = attentionstate.Ensure(attentionstate.Options{ProjectRoot: projectRoot})
	} else {
		stateRoot, err = attentionstate.Ensure(attentionstate.Options{Root: stateRoot})
	}
	if err != nil {
		return nil, err
	}
	store, err := mail.NewStore(stateRoot)
	if err != nil {
		return nil, err
	}
	return mail.NewService(store, projectRoot, cfg), nil
}

func applyBudgetDefaults(mode contractpeering.PeerCallMode, b contractpeering.BudgetSpec) contractpeering.BudgetSpec {
	if b.Turns == 0 {
		if mode == contractpeering.PeerCallSpecOut {
			b.Turns = DefaultSpecOutTurns
		} else {
			b.Turns = DefaultAdvisoryTurns
		}
	}
	if b.Tokens == 0 {
		if mode == contractpeering.PeerCallSpecOut {
			b.Tokens = DefaultSpecOutTokens
		} else {
			b.Tokens = DefaultAdvisoryTokens
		}
	}
	return b
}

func deprecatedSubagentWarning(cfg config.Config) string {
	if cfg.Peering != nil && cfg.Peering.Subagent != nil {
		return "peering.subagent is deprecated and ignored; Hero core does not execute model CLIs"
	}
	return ""
}

func peerCallStatePath(projectRoot, id string) string {
	return filepath.Join(projectRoot, ".hero", "peer-calls", id+".md")
}
