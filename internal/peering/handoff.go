package peering

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

type HandoffOptions struct {
	PeerAlias      string
	Title          string
	Type           string
	Reason         string
	IdempotencyKey string
	At             time.Time
	StateRoot      string
}

type HandoffResult struct {
	PeerSlug         string `json:"peer_slug,omitempty"`
	PeerPath         string `json:"peer_path,omitempty"`
	PeerID           string `json:"peer_id"`
	OriginPeerID     string `json:"origin_peer_id"`
	PeerType         string `json:"peer_type,omitempty"`
	PeerAliasDisplay string `json:"peer_alias_display"`
	MessageID        string `json:"message_id"`
	ThreadID         string `json:"thread_id"`
	Status           string `json:"status"`
}

func Handoff(projectRoot, originSlug string, opts HandoffOptions) (*HandoffResult, error) {
	if originSlug == "" {
		return nil, errors.New("origin slug required")
	}
	if opts.PeerAlias == "" {
		return nil, errors.New("peer alias required")
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
	specs, err := spec.Discover(cfg.HeroDir(projectRoot))
	if err != nil {
		return nil, fmt.Errorf("discover local specs: %w", err)
	}
	var origin *spec.Spec
	for _, candidate := range specs {
		if candidate.Slug == originSlug {
			origin = candidate
			break
		}
	}
	if origin == nil {
		return nil, fmt.Errorf("no spec with slug %q in this workspace", originSlug)
	}
	title := opts.Title
	if title == "" {
		title = origin.Title
	}
	targetType := opts.Type
	if targetType == "" {
		targetType = string(origin.Type)
	}
	summary := boundedSpecSummary(origin)
	body, err := json.Marshal(map[string]any{
		"origin_slug": originSlug, "title": title, "type": targetType,
		"reason": opts.Reason, "source_peer_id": cfg.PeerID,
		"source_commit": readGitHEAD(projectRoot), "summary": summary,
	})
	if err != nil {
		return nil, err
	}
	key := opts.IdempotencyKey
	if key == "" {
		key = fmt.Sprintf("peer-work-transfer:%s:%s:%s", cfg.PeerID, originSlug, opts.PeerAlias)
	}
	svc, err := projectMailService(projectRoot, opts.StateRoot, cfg)
	if err != nil {
		return nil, err
	}
	delivery, err := svc.Send(mail.SendRequest{
		RecipientAlias: opts.PeerAlias,
		Subject:        "Work transfer: " + title,
		Body:           string(body),
		Kind:           "peer.work_transfer",
		IdempotencyKey: key,
	})
	if err != nil {
		return nil, err
	}
	return &HandoffResult{
		PeerID: manifest.Repo.PeerID, OriginPeerID: cfg.PeerID,
		PeerType: targetType, PeerAliasDisplay: opts.PeerAlias,
		MessageID: delivery.MessageID, ThreadID: delivery.ThreadID, Status: "queued",
	}, nil
}

func boundedSpecSummary(origin *spec.Spec) string {
	data, err := os.ReadFile(origin.Path)
	if err != nil {
		return origin.Title
	}
	text := string(data)
	if len(text) > 12_000 {
		text = text[:12_000]
	}
	return text
}

func readPeerIDFromJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var shape struct {
		PeerID string `json:"peer_id"`
	}
	if json.Unmarshal(data, &shape) != nil {
		return ""
	}
	return shape.PeerID
}

func readGitHEAD(dir string) string {
	gitDir := filepath.Join(dir, ".git")
	if data, err := os.ReadFile(gitDir); err == nil && strings.HasPrefix(string(data), "gitdir: ") {
		gitDir = strings.TrimSpace(strings.TrimPrefix(string(data), "gitdir: "))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(dir, gitDir)
		}
	}
	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(head))
	if strings.HasPrefix(value, "ref: ") {
		ref := strings.TrimPrefix(value, "ref: ")
		data, err := os.ReadFile(filepath.Join(gitDir, filepath.FromSlash(ref)))
		if err != nil {
			return ""
		}
		value = strings.TrimSpace(string(data))
	}
	if len(value) > 7 {
		return value[:7]
	}
	return value
}

func workspaceDisplayName(cfg config.Config, root string) string {
	if cfg.Peering != nil && cfg.Peering.Display != "" {
		return cfg.Peering.Display
	}
	return filepath.Base(root)
}
