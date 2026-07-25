package peering

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
)

type ReceiveOptions struct {
	Type           string
	IdempotencyKey string
	StateRoot      string
}

type ReceiveResult struct {
	MessageID  string                    `json:"message_id"`
	ThreadID   string                    `json:"thread_id"`
	Artifact   *attention.MailArtifact   `json:"artifact"`
	Navigation *mail.NavigationReference `json:"navigation,omitempty"`
	ReplyID    string                    `json:"reply_id"`
	Replayed   bool                      `json:"replayed"`
}

func Receive(projectRoot, messageID string, opts ReceiveOptions) (*ReceiveResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}
	svc, err := projectMailService(projectRoot, opts.StateRoot, cfg)
	if err != nil {
		return nil, err
	}
	item, err := svc.Show(messageID, false)
	if err != nil {
		return nil, err
	}
	if item.Kind != "peer.work_transfer" {
		return nil, fmt.Errorf("message %s is %q, not peer.work_transfer", messageID, item.Kind)
	}
	artifactType := opts.Type
	if artifactType == "" {
		var payload struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal([]byte(item.Body), &payload)
		artifactType = payload.Type
	}
	if artifactType != "feature" && artifactType != "bug" {
		artifactType = "feature"
	}
	key := opts.IdempotencyKey
	if key == "" {
		key = "handoff-receive:" + messageID + ":" + artifactType
	}
	replayed := item.Receipt != nil && item.Receipt.PromotedArtifact != nil
	revision := int64(0)
	if item.Receipt != nil {
		revision = item.Receipt.Revision
	}
	promoted, err := svc.Action(mail.ActionRequest{
		MessageID: messageID, Action: mail.ActionPromote, ArtifactType: artifactType,
		ExpectedRevision: revision, IdempotencyKey: key,
	})
	if err != nil {
		return nil, err
	}
	if promoted.Artifact == nil {
		return nil, errors.New("mail promotion returned no artifact")
	}
	replyBody, _ := json.Marshal(map[string]any{
		"status": "accepted", "artifact": promoted.Artifact, "navigation": promoted.Navigation,
	})
	reply, err := svc.Reply(mail.ReplyRequest{
		MessageID: messageID, Body: string(replyBody), Kind: "peer.work_transfer.response",
		IdempotencyKey: "handoff-receive-reply:" + messageID,
	})
	if err != nil {
		return nil, err
	}
	ingestPromotedSpec(projectRoot, cfg, promoted.Artifact.Slug)

	return &ReceiveResult{
		MessageID: messageID, ThreadID: item.ThreadID, Artifact: promoted.Artifact,
		Navigation: promoted.Navigation, ReplyID: reply.MessageID, Replayed: replayed,
	}, nil
}

// ingestPromotedSpec writes the just-received spec into the graph substrate
// so it is resolvable immediately rather than only after some later command
// happens to reconcile.
//
// The receive path writes a spec file to disk. That lands in the index on
// the next read (the index self-heals), but graph.db has no such self-heal —
// which is why peer-received specs were invisible to `hero why` while
// `hero graph` and `hero search` found them. `hero why` now reconciles on
// read, so this is belt-and-suspenders: it closes the window at the write
// end, and keeps the spec resolvable to any graph reader that does not
// reconcile.
//
// Keyed via gitutil.RepoKey — the same derivation cli's graphRepoKey uses,
// so these nodes land in the partition the readers filter on. Deriving it
// any other way (notably filepath.Base(projectRoot)) writes into a subgraph
// no reader queries.
//
// Only the promoted spec is written; WriteGraph upserts rather than
// replacing the corpus, and resolves relation targets against nodes already
// in the store.
//
// Best-effort by contract: a receive must not fail because the graph could
// not be written. The spec file on disk is the durable truth, and a later
// reconcile picks it up regardless.
func ingestPromotedSpec(projectRoot string, cfg config.Config, slug string) {
	if slug == "" {
		return
	}
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return
	}
	var received []*spec.Spec
	for _, s := range specs {
		if s != nil && s.Slug == slug {
			received = append(received, s)
		}
	}
	if len(received) == 0 {
		return
	}
	store, err := graph.Open(heroDir)
	if err != nil {
		return
	}
	defer store.Close()
	_, _ = spec.WriteGraph(received, gitutil.RepoKey(projectRoot),
		graph.DomainFor(cfg, graph.IntrinsicActive), store)
}
