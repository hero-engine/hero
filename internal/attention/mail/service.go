package mail

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/peering"
)

type SendRequest struct{ RecipientAlias, Subject, Body, Kind, MessageID, IdempotencyKey string }
type ReplyRequest struct{ MessageID, Subject, Body, Kind, IdempotencyKey string }
type ListedMessage struct {
	attention.MailEnvelope
	Receipt *attention.MailReceipt `json:"receipt,omitempty"`
}

type Service struct {
	store *Store
	root  string
	cfg   config.Config
	now   func() time.Time
	newID func() (string, error)
}

func NewService(store *Store, projectRoot string, cfg config.Config) *Service {
	return &Service{store: store, root: projectRoot, cfg: cfg, now: time.Now, newID: randomMailID}
}

func (s *Service) Send(req SendRequest) (attention.MailDelivery, error) {
	if strings.TrimSpace(req.RecipientAlias) == "" {
		return attention.MailDelivery{}, errors.New("recipient alias is required")
	}
	path, err := s.cfg.ResolveRepoPath(s.root, req.RecipientAlias)
	if err != nil {
		return attention.MailDelivery{}, err
	}
	if _, err := filepath.Abs(path); err != nil {
		return attention.MailDelivery{}, fmt.Errorf("resolve recipient path: %w", err)
	}
	manifest, err := peering.ReadPeerManifest(path, s.cfg.Folder)
	if err != nil {
		return attention.MailDelivery{}, err
	}
	if strings.TrimSpace(s.cfg.PeerID) == "" {
		return attention.MailDelivery{}, errors.New("sender peer_id is required")
	}
	if strings.TrimSpace(manifest.Repo.PeerID) == "" {
		return attention.MailDelivery{}, errors.New("recipient peer manifest has no peer_id")
	}
	if !validPathID(manifest.Repo.PeerID) || !validPathID(s.cfg.PeerID) {
		return attention.MailDelivery{}, errors.New("peer_id contains forbidden path characters")
	}
	senderName := filepath.Base(s.root)
	if s.cfg.Peering != nil && s.cfg.Peering.Display != "" {
		senderName = s.cfg.Peering.Display
	}
	recipientName := manifest.Repo.Display
	if recipientName == "" {
		recipientName = manifest.Repo.Name
	}
	if recipientName == "" {
		recipientName = req.RecipientAlias
	}
	return s.deliver(req, attention.ProjectReference{PeerID: s.cfg.PeerID, DisplayName: senderName}, attention.ProjectReference{PeerID: manifest.Repo.PeerID, RegistrySlug: req.RecipientAlias, DisplayName: recipientName}, "", "")
}

func (s *Service) Reply(req ReplyRequest) (attention.MailDelivery, error) {
	original, err := s.store.Get(s.cfg.PeerID, req.MessageID)
	if err != nil {
		return attention.MailDelivery{}, err
	}
	if original.ThreadID == "" {
		original.ThreadID = original.ID
	}
	if original.InReplyTo != "" {
		target, err := s.store.Get(s.cfg.PeerID, original.InReplyTo)
		if err != nil {
			return attention.MailDelivery{}, errors.New("reply target is missing")
		}
		targetThread := target.ThreadID
		if targetThread == "" {
			targetThread = target.ID
		}
		if targetThread != original.ThreadID {
			return attention.MailDelivery{}, errors.New("reply target belongs to a different thread")
		}
	}
	alias := original.Sender.RegistrySlug
	if alias == "" {
		for candidate, meta := range s.cfg.RepoMeta {
			if meta.PeerID == original.Sender.PeerID {
				alias = candidate
				break
			}
		}
	}
	if alias == "" {
		for candidate := range s.cfg.Repos {
			p, e := s.cfg.ResolveRepoPath(s.root, candidate)
			if e != nil {
				continue
			}
			m, e := peering.ReadPeerManifest(p, s.cfg.Folder)
			if e == nil && m.Repo.PeerID == original.Sender.PeerID {
				alias = candidate
				break
			}
		}
	}
	if alias == "" {
		return attention.MailDelivery{}, errors.New("original sender is not a configured peer")
	}
	path, err := s.cfg.ResolveRepoPath(s.root, alias)
	if err != nil {
		return attention.MailDelivery{}, err
	}
	manifest, err := peering.ReadPeerManifest(path, s.cfg.Folder)
	if err != nil {
		return attention.MailDelivery{}, err
	}
	if manifest.Repo.PeerID != original.Sender.PeerID {
		return attention.MailDelivery{}, errors.New("original sender peer_id no longer matches configured peer")
	}
	subject := req.Subject
	if subject == "" {
		subject = "Re: " + original.Subject
	}
	kind := req.Kind
	if kind == "" {
		kind = attention.MailKindResponse
	}
	senderName := filepath.Base(s.root)
	if s.cfg.Peering != nil && s.cfg.Peering.Display != "" {
		senderName = s.cfg.Peering.Display
	}
	return s.deliver(SendRequest{Subject: subject, Body: req.Body, Kind: kind, IdempotencyKey: req.IdempotencyKey}, attention.ProjectReference{PeerID: s.cfg.PeerID, DisplayName: senderName}, original.Sender, original.ThreadID, original.ID)
}

func (s *Service) deliver(req SendRequest, sender, recipient attention.ProjectReference, threadID, inReplyTo string) (attention.MailDelivery, error) {
	if strings.TrimSpace(req.Subject) == "" {
		return attention.MailDelivery{}, errors.New("subject is required")
	}
	if !utf8.ValidString(req.Subject) || utf8.RuneCountInString(req.Subject) > attention.MaxSubjectCharacters {
		return attention.MailDelivery{}, errors.New("subject must be valid UTF-8 and at most 200 characters")
	}
	if !utf8.ValidString(req.Body) || len(req.Body) > attention.MaxBodyBytes {
		return attention.MailDelivery{}, errors.New("body must be valid UTF-8 and at most 65536 bytes")
	}
	if req.Kind == "" {
		req.Kind = attention.MailKindNotice
	}
	if !utf8.ValidString(req.Kind) || len(req.Kind) > 64 {
		return attention.MailDelivery{}, errors.New("kind must be valid UTF-8 and at most 64 bytes")
	}
	id := req.MessageID
	var err error
	if id == "" {
		id, err = s.newID()
		if err != nil {
			return attention.MailDelivery{}, err
		}
	}
	if !validMailID(id) {
		return attention.MailDelivery{}, errors.New("invalid mail message ID")
	}
	key := req.IdempotencyKey
	if key == "" {
		key = id
	}
	if !utf8.ValidString(key) || len(key) > 512 {
		return attention.MailDelivery{}, errors.New("idempotency key must be valid UTF-8 and at most 512 bytes")
	}
	if threadID == "" {
		threadID = id
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	env := attention.MailEnvelope{SchemaVersion: attention.SchemaVersion, ID: id, Recipient: recipient, Sender: sender, Subject: req.Subject, Body: req.Body, Kind: req.Kind, ThreadID: threadID, InReplyTo: inReplyTo, IdempotencyKey: key, CreatedAt: now}
	env.Revision = envelopeRevision(env)
	if cerr := attention.ValidateMailEnvelope(env); cerr != nil {
		return attention.MailDelivery{}, cerr
	}
	d := attention.MailDelivery{SchemaVersion: attention.SchemaVersion, MessageID: id, ThreadID: threadID, Sender: sender, Recipient: recipient, IdempotencyKey: key, DeliveredAt: now}
	result, _, err := s.store.Deliver(env, d)
	return result, err
}

func (s *Service) Inbox(project string, unread bool) ([]ListedMessage, error) {
	recipient := s.cfg.PeerID
	if project != "" && project != recipient {
		if _, ok := s.cfg.Repos[project]; ok {
			path, err := s.cfg.ResolveRepoPath(s.root, project)
			if err != nil {
				return nil, err
			}
			manifest, err := peering.ReadPeerManifest(path, s.cfg.Folder)
			if err != nil {
				return nil, err
			}
			recipient = manifest.Repo.PeerID
		} else {
			recipient = project
		}
	}
	if !validPathID(recipient) {
		return nil, errors.New("invalid project peer ID")
	}
	items, err := s.store.List(recipient)
	if err != nil {
		return nil, err
	}
	out := make([]ListedMessage, 0, len(items))
	for _, env := range items {
		var rp *attention.MailReceipt
		if r, e := s.store.Receipt(recipient, env.ID); e == nil {
			rp = &r
		} else if !errors.Is(e, ErrNotFound) {
			return nil, e
		}
		if unread && rp != nil && rp.ReadAt != "" {
			continue
		}
		out = append(out, ListedMessage{MailEnvelope: env, Receipt: rp})
	}
	return out, nil
}

func (s *Service) Show(id string, markRead bool) (ListedMessage, error) {
	env, err := s.store.Get(s.cfg.PeerID, id)
	if err != nil {
		return ListedMessage{}, err
	}
	var rp *attention.MailReceipt
	if markRead {
		r, e := s.store.UpdateReceipt(s.cfg.PeerID, id, env.Revision, func(r *attention.MailReceipt) {
			if r.ReadAt == "" {
				r.ReadAt = s.now().UTC().Format(time.RFC3339Nano)
			}
			if r.Kind == "" {
				r.Kind = attention.ReceiptRead
			}
		})
		if e != nil {
			return ListedMessage{}, e
		}
		rp = &r
	} else if r, e := s.store.Receipt(s.cfg.PeerID, id); e == nil {
		rp = &r
	} else if !errors.Is(e, ErrNotFound) {
		return ListedMessage{}, e
	}
	return ListedMessage{MailEnvelope: env, Receipt: rp}, nil
}
func (s *Service) Ack(id, note string) (attention.MailReceipt, error) {
	if !utf8.ValidString(note) || utf8.RuneCountInString(note) > 500 {
		return attention.MailReceipt{}, errors.New("acknowledgement note must be valid UTF-8 and at most 500 characters")
	}
	env, err := s.store.Get(s.cfg.PeerID, id)
	if err != nil {
		return attention.MailReceipt{}, err
	}
	return s.store.UpdateReceipt(s.cfg.PeerID, id, env.Revision, func(r *attention.MailReceipt) {
		if r.AcknowledgedAt == "" {
			r.AcknowledgedAt = s.now().UTC().Format(time.RFC3339Nano)
			r.AcknowledgementNote = note
		}
		r.Kind = attention.ReceiptAcknowledged
	})
}

func validMailID(id string) bool {
	if !strings.HasPrefix(id, "mail_") || len(id) == 5 {
		return false
	}
	for _, r := range id[5:] {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}
func randomMailID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "mail_" + hex.EncodeToString(b), nil
}
func envelopeRevision(v attention.MailEnvelope) int64 {
	v.Revision = 0
	b, _ := jsonMarshal(v)
	sum := sha256.Sum256(b)
	n := int64(binary.BigEndian.Uint64(sum[:8]) & (1<<63 - 1))
	if n == 0 {
		n = 1
	}
	return n
}
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
