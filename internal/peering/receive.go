package peering

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
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
	return &ReceiveResult{
		MessageID: messageID, ThreadID: item.ThreadID, Artifact: promoted.Artifact,
		Navigation: promoted.Navigation, ReplyID: reply.MessageID, Replayed: replayed,
	}, nil
}
