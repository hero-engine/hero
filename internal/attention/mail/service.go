package mail

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/intake"
	"gopkg.in/yaml.v3"
)

type SendRequest struct {
	RecipientAlias        string
	ExpectedRecipientPeer string
	Subject               string
	Body                  string
	Kind                  string
	MessageID             string
	IdempotencyKey        string
	Provenance            []attention.ProvenanceReference
}
type ReplyRequest struct {
	MessageID      string
	ExpectedThread string
	Subject        string
	Body           string
	Kind           string
	IdempotencyKey string
	Provenance     []attention.ProvenanceReference
}
type ReplyResult struct {
	Delivery attention.MailDelivery `json:"delivery"`
	Thread   mailthread.ThreadView  `json:"thread"`
}
type ListedMessage struct {
	attention.MailEnvelope
	Receipt *attention.MailReceipt `json:"receipt,omitempty"`
}

type UnreadSummary struct {
	Count int                 `json:"count"`
	Items []UnreadSummaryItem `json:"items"`
}

type UnreadSummaryItem struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
}

type Service struct {
	store     *Store
	root      string
	cfg       config.Config
	now       func() time.Time
	newID     func() (string, error)
	intake    *intake.Service
	focus     *focus.Service
	failAfter func(string) error
}

func NewService(store *Store, projectRoot string, cfg config.Config) *Service {
	service := &Service{store: store, root: projectRoot, cfg: cfg, now: time.Now, newID: randomMailID}
	service.intake = intake.NewService(cfg.HeroDir(projectRoot))
	if focusStore, err := focus.NewStore(store.stateRoot); err == nil {
		if resolver, resolverErr := focus.LoadRegistryResolver(projectRoot); resolverErr == nil {
			service.focus = focus.NewService(focusStore, resolver)
		}
	}
	return service
}

func (s *Service) SetFailureInjector(inject func(step string) error) {
	s.failAfter = inject
}

func (s *Service) Send(req SendRequest) (attention.MailDelivery, error) {
	if strings.TrimSpace(req.RecipientAlias) == "" {
		return attention.MailDelivery{}, errors.New("recipient alias is required")
	}
	path, err := s.cfg.ResolveRepoPath(s.root, req.RecipientAlias)
	if err != nil {
		return attention.MailDelivery{}, fmt.Errorf("%w: %v", ErrRecipientMissing, err)
	}
	if _, err := filepath.Abs(path); err != nil {
		return attention.MailDelivery{}, fmt.Errorf("%w: resolve recipient path: %v", ErrUnavailable, err)
	}
	manifest, err := readPeerManifest(path, s.cfg.Folder)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return attention.MailDelivery{}, fmt.Errorf("%w: %v", ErrRecipientMissing, err)
		}
		return attention.MailDelivery{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if strings.TrimSpace(s.cfg.PeerID) == "" {
		return attention.MailDelivery{}, fmt.Errorf("%w: sender peer_id is required", ErrUnavailable)
	}
	if strings.TrimSpace(manifest.Repo.PeerID) == "" {
		return attention.MailDelivery{}, fmt.Errorf("%w: recipient peer manifest has no peer_id", ErrUnavailable)
	}
	if req.ExpectedRecipientPeer != "" && manifest.Repo.PeerID != req.ExpectedRecipientPeer {
		return attention.MailDelivery{}, ErrRecipientMismatch
	}
	if !validPathID(manifest.Repo.PeerID) || !validPathID(s.cfg.PeerID) {
		return attention.MailDelivery{}, fmt.Errorf("%w: peer_id contains forbidden path characters", ErrUnavailable)
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
	result, err := s.ReplyAndReconcile(req)
	return result.Delivery, err
}

func (s *Service) ReplyAndReconcile(req ReplyRequest) (ReplyResult, error) {
	original, err := s.store.Get(s.cfg.PeerID, req.MessageID)
	if err != nil {
		return ReplyResult{}, err
	}
	threadID := threadIdentity(original)
	messageIDs, err := s.threadMessageIDs(threadID)
	if err != nil {
		return ReplyResult{}, err
	}
	delivery, err := s.replyDelivery(req)
	if err != nil {
		return ReplyResult{}, err
	}
	if err := s.markMessagesRead(messageIDs, "reply:"+delivery.IdempotencyKey); err != nil {
		return ReplyResult{Delivery: delivery}, err
	}
	view, _, err := s.store.ReconcileThread(s.cfg.PeerID, threadID, s.now())
	if err != nil {
		return ReplyResult{Delivery: delivery}, err
	}
	event := mailthread.Event{
		SchemaVersion: mailthread.SchemaVersion,
		Identity:      mailthread.Identity{ProjectPeerID: s.cfg.PeerID, ThreadID: threadID},
		Kind:          mailthread.EventReplySucceeded, EventID: "reply:" + delivery.IdempotencyKey,
		ExpectedRevision: view.State.Revision, OccurredAt: delivery.DeliveredAt,
		Source: "mail.reply", SourceID: delivery.MessageID,
	}
	eventResult, err := s.applyStoreEventLatest(event)
	if err != nil {
		return ReplyResult{Delivery: delivery}, err
	}
	return ReplyResult{Delivery: delivery, Thread: eventResult.Thread}, nil
}

func (s *Service) replyDelivery(req ReplyRequest) (attention.MailDelivery, error) {
	original, err := s.store.Get(s.cfg.PeerID, req.MessageID)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return attention.MailDelivery{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		return attention.MailDelivery{}, err
	}
	if original.ThreadID == "" {
		original.ThreadID = original.ID
	}
	if req.ExpectedThread != "" && original.ThreadID != req.ExpectedThread {
		return attention.MailDelivery{}, ErrThreadMismatch
	}
	if original.InReplyTo != "" {
		target, err := s.store.Get(s.cfg.PeerID, original.InReplyTo)
		if errors.Is(err, ErrNotFound) {
			// A received reply points at the prior outbound envelope, which is
			// authoritative in the remote recipient's mailbox.
			target, err = s.store.Get(original.Sender.PeerID, original.InReplyTo)
		}
		if err != nil {
			return attention.MailDelivery{}, fmt.Errorf("%w: reply target is missing", ErrUnavailable)
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
			m, e := readPeerManifest(p, s.cfg.Folder)
			if e == nil && m.Repo.PeerID == original.Sender.PeerID {
				alias = candidate
				break
			}
		}
	}
	if alias == "" {
		return attention.MailDelivery{}, fmt.Errorf("%w: original sender is not a configured peer", ErrRecipientMissing)
	}
	path, err := s.cfg.ResolveRepoPath(s.root, alias)
	if err != nil {
		return attention.MailDelivery{}, fmt.Errorf("%w: %v", ErrRecipientMissing, err)
	}
	manifest, err := readPeerManifest(path, s.cfg.Folder)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return attention.MailDelivery{}, fmt.Errorf("%w: %v", ErrRecipientMissing, err)
		}
		return attention.MailDelivery{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
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
	return s.deliver(SendRequest{
		Subject: subject, Body: req.Body, Kind: kind, IdempotencyKey: req.IdempotencyKey, Provenance: req.Provenance,
	}, attention.ProjectReference{PeerID: s.cfg.PeerID, DisplayName: senderName}, original.Sender, original.ThreadID, original.ID)
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
	env := attention.MailEnvelope{
		SchemaVersion: attention.SchemaVersion, ID: id, Recipient: recipient, Sender: sender,
		Subject: req.Subject, Body: req.Body, Kind: req.Kind, ThreadID: threadID, InReplyTo: inReplyTo,
		IdempotencyKey: key, CreatedAt: now, Provenance: req.Provenance,
	}
	env.Revision = envelopeRevision(env)
	if cerr := attention.ValidateMailEnvelope(env); cerr != nil {
		return attention.MailDelivery{}, cerr
	}
	d := attention.MailDelivery{SchemaVersion: attention.SchemaVersion, MessageID: id, ThreadID: threadID, Sender: sender, Recipient: recipient, IdempotencyKey: key, DeliveredAt: now}
	result, _, err := s.store.Deliver(env, d)
	if err == nil {
		view, _, threadErr := s.store.Thread(recipient.PeerID, result.ThreadID)
		if threadErr != nil {
			return result, fmt.Errorf("%w: initialize mail thread: %v", ErrUnavailable, threadErr)
		}
		event := mailthread.Event{
			SchemaVersion: mailthread.SchemaVersion,
			Identity:      mailthread.Identity{ProjectPeerID: recipient.PeerID, ThreadID: result.ThreadID},
			Kind:          mailthread.EventInboundActivity, EventID: "delivery:" + result.MessageID,
			ExpectedRevision: view.State.Revision, OccurredAt: result.DeliveredAt,
			Source: "mail.delivery", SourceID: result.MessageID, MessageID: result.MessageID,
		}
		inbound, eventErr := s.applyStoreEventLatest(event)
		if eventErr != nil {
			return result, fmt.Errorf("%w: reconcile inbound mail thread: %v", ErrUnavailable, eventErr)
		}
		if inReplyTo != "" {
			if original, originalErr := s.store.Get(sender.PeerID, inReplyTo); originalErr == nil {
				kind := mailthread.EventKind("")
				outcome := ""
				source := ""
				switch original.Kind {
				case "peer.advisory":
					kind, outcome, source = mailthread.EventAdvisoryTerminal, mailthread.OutcomeAnswered, "peer.advisory"
				case "peer.spec_out":
					kind, outcome, source = mailthread.EventSpecOutTerminal, mailthread.OutcomeCompleted, "peer.spec_out"
				}
				if kind != "" {
					terminal := mailthread.Event{
						SchemaVersion: mailthread.SchemaVersion,
						Identity:      mailthread.Identity{ProjectPeerID: recipient.PeerID, ThreadID: result.ThreadID},
						Kind:          kind, EventID: "terminal:" + result.MessageID,
						ExpectedRevision: inbound.Thread.State.Revision, OccurredAt: result.DeliveredAt,
						Source: source, SourceID: original.ID, Outcome: outcome,
					}
					if _, terminalErr := s.applyStoreEventLatest(terminal); terminalErr != nil {
						return result, fmt.Errorf("%w: reconcile typed peer response: %v", ErrUnavailable, terminalErr)
					}
				}
			}
		}
		return result, nil
	}
	if errors.Is(err, ErrIdempotencyConflict) {
		return result, err
	}
	var contractErr *attention.ContractError
	if errors.As(err, &contractErr) {
		return result, err
	}
	return result, fmt.Errorf("%w: %v", ErrUnavailable, err)
}

func (s *Service) Inbox(project string, unread bool) ([]ListedMessage, error) {
	recipient := s.cfg.PeerID
	if project != "" && project != recipient {
		if _, ok := s.cfg.Repos[project]; ok {
			path, err := s.cfg.ResolveRepoPath(s.root, project)
			if err != nil {
				return nil, err
			}
			manifest, err := readPeerManifest(path, s.cfg.Folder)
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
		if _, _, threadErr := s.store.ReconcileThread(recipient, threadIdentity(env), s.now()); threadErr != nil {
			return nil, threadErr
		}
		var rp *attention.MailReceipt
		if r, e := s.store.Receipt(recipient, env.ID); e == nil {
			rp = &r
		} else if !errors.Is(e, ErrNotFound) {
			return nil, e
		}
		if rp != nil && rp.DismissedAt != "" {
			continue
		}
		if unread && rp != nil && rp.ReadAt != "" {
			continue
		}
		out = append(out, ListedMessage{MailEnvelope: env, Receipt: rp})
	}
	return out, nil
}

func readPeerManifest(projectRoot, folder string) (*contractpeering.PeerManifest, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, folder, "peer-manifest.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read peer manifest: %w", err)
	}
	var manifest contractpeering.PeerManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode peer manifest: %w", err)
	}
	return &manifest, nil
}

func (s *Service) UnreadSummary(limit int) (UnreadSummary, error) {
	items, err := s.Inbox("", true)
	if err != nil {
		return UnreadSummary{}, err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt != items[j].CreatedAt {
			return items[i].CreatedAt < items[j].CreatedAt
		}
		return items[i].ID < items[j].ID
	})
	summary := UnreadSummary{Count: len(items)}
	if limit < 0 {
		limit = 0
	}
	for i, item := range items {
		if i >= limit {
			break
		}
		summary.Items = append(summary.Items, UnreadSummaryItem{ID: item.ID, Subject: item.Subject})
	}
	return summary, nil
}

func (s *Service) Show(id string, markRead bool) (ListedMessage, error) {
	env, err := s.store.Get(s.cfg.PeerID, id)
	if err != nil {
		return ListedMessage{}, err
	}
	if _, _, err := s.store.ReconcileThread(s.cfg.PeerID, threadIdentity(env), s.now()); err != nil {
		return ListedMessage{}, err
	}
	var rp *attention.MailReceipt
	if markRead {
		current, e := currentReceipt(s.store, s.cfg.PeerID, id)
		if e != nil {
			return ListedMessage{}, e
		}
		result, e := s.Action(ActionRequest{MessageID: id, Action: ActionRead, ExpectedRevision: current.Revision, IdempotencyKey: legacyActionKey("show-read", id)})
		if e != nil {
			return ListedMessage{}, e
		}
		rp = &result.Receipt
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
	current, err := currentReceipt(s.store, s.cfg.PeerID, id)
	if err != nil {
		return attention.MailReceipt{}, err
	}
	result, err := s.Action(ActionRequest{MessageID: id, Action: ActionAcknowledge, ExpectedRevision: current.Revision, IdempotencyKey: legacyActionKey("ack", id, note), Note: note})
	return result.Receipt, err
}

func legacyActionKey(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("legacy:%x", hash.Sum(nil))
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
