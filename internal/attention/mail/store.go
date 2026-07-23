// Package mail implements the private user-global Project Mail store.
package mail

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hero-engine/hero/contracts/attention"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
)

var (
	ErrNotFound            = errors.New("mail message not found")
	ErrIdempotencyConflict = errors.New("idempotency_conflict")
)

type Store struct{ root, boxes, outbound, lockPath string }

func NewStore(stateRoot string) (*Store, error) {
	root := filepath.Join(stateRoot, attentionstate.MailDirectory)
	s := &Store{root: root, boxes: filepath.Join(root, "boxes"), outbound: filepath.Join(root, "outbound"), lockPath: filepath.Join(root, ".lock")}
	for _, p := range []string{s.root, s.boxes, s.outbound} {
		if err := secureDir(p); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) Deliver(env attention.MailEnvelope, delivery attention.MailDelivery) (attention.MailDelivery, bool, error) {
	if !validPathID(env.Recipient.PeerID) || !validPathID(env.Sender.PeerID) || !validMailID(env.ID) {
		return attention.MailDelivery{}, false, errors.New("invalid mail storage identifier")
	}
	if err := attention.ValidateMailEnvelope(env); err != nil {
		return attention.MailDelivery{}, false, err
	}
	if delivery.SchemaVersion != attention.SchemaVersion || delivery.MessageID != env.ID || delivery.ThreadID != env.ThreadID || delivery.Sender.PeerID != env.Sender.PeerID || delivery.Recipient.PeerID != env.Recipient.PeerID || delivery.IdempotencyKey != env.IdempotencyKey || delivery.DeliveredAt == "" {
		return attention.MailDelivery{}, false, errors.New("invalid mail delivery record")
	}
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return attention.MailDelivery{}, false, err
	}
	defer lock.Close()
	msgDir := filepath.Join(s.boxes, env.Recipient.PeerID, "messages")
	receiptDir := filepath.Join(s.boxes, env.Recipient.PeerID, "receipts")
	outDir := filepath.Join(s.outbound, env.Sender.PeerID)
	for _, p := range []string{msgDir, receiptDir, outDir} {
		if err := secureDir(p); err != nil {
			return attention.MailDelivery{}, false, err
		}
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return attention.MailDelivery{}, false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var existing attention.MailDelivery
		b, err := os.ReadFile(filepath.Join(outDir, entry.Name()))
		if err != nil {
			return attention.MailDelivery{}, false, err
		}
		if err := json.Unmarshal(b, &existing); err != nil {
			return attention.MailDelivery{}, false, fmt.Errorf("decode outbound delivery %s: %w", entry.Name(), err)
		}
		if !validMailID(existing.MessageID) || !validPathID(existing.Sender.PeerID) || !validPathID(existing.Recipient.PeerID) {
			return attention.MailDelivery{}, false, fmt.Errorf("invalid outbound delivery %s", entry.Name())
		}
		if existing.IdempotencyKey == delivery.IdempotencyKey {
			if existing.Recipient.PeerID != env.Recipient.PeerID || existing.Sender.PeerID != env.Sender.PeerID {
				return attention.MailDelivery{}, false, ErrIdempotencyConflict
			}
			eb, eerr := os.ReadFile(filepath.Join(msgDir, existing.MessageID+".json"))
			if eerr != nil {
				return attention.MailDelivery{}, false, eerr
			}
			var old attention.MailEnvelope
			if json.Unmarshal(eb, &old) != nil {
				return attention.MailDelivery{}, false, ErrIdempotencyConflict
			}
			if old.ThreadID == old.ID {
				old.ThreadID = ""
			}
			if env.ThreadID == env.ID {
				env.ThreadID = ""
			}
			old.ID, env.ID = "", ""
			old.CreatedAt, env.CreatedAt = "", ""
			old.Revision, env.Revision = 0, 0
			ob, _ := json.Marshal(old)
			nb, _ := json.Marshal(env)
			if !bytes.Equal(ob, nb) {
				return attention.MailDelivery{}, false, ErrIdempotencyConflict
			}
			return existing, false, nil
		}
	}
	messagePath := filepath.Join(msgDir, env.ID+".json")
	if b, err := os.ReadFile(messagePath); err == nil {
		candidate, _ := json.MarshalIndent(env, "", "  ")
		candidate = append(candidate, '\n')
		if !bytes.Equal(b, candidate) {
			return attention.MailDelivery{}, false, ErrIdempotencyConflict
		}
		return delivery, false, nil
	} else if !os.IsNotExist(err) {
		return attention.MailDelivery{}, false, err
	}
	if err := atomicCreate(messagePath, env); err != nil {
		return attention.MailDelivery{}, false, err
	}
	if err := atomicCreate(filepath.Join(outDir, env.ID+".json"), delivery); err != nil {
		_ = os.Remove(messagePath)
		return attention.MailDelivery{}, false, err
	}
	return delivery, true, nil
}

func (s *Store) Get(recipient, id string) (attention.MailEnvelope, error) {
	if !validPathID(recipient) || !validMailID(id) {
		return attention.MailEnvelope{}, errors.New("invalid mail storage identifier")
	}
	var env attention.MailEnvelope
	b, err := os.ReadFile(filepath.Join(s.boxes, recipient, "messages", id+".json"))
	if os.IsNotExist(err) {
		return env, ErrNotFound
	}
	if err != nil {
		return env, err
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return env, fmt.Errorf("decode mail message: %w", err)
	}
	if err := attention.ValidateMailEnvelope(env); err != nil {
		return env, err
	}
	return env, nil
}

func (s *Store) List(recipient string) ([]attention.MailEnvelope, error) {
	if !validPathID(recipient) {
		return nil, errors.New("invalid mail storage identifier")
	}
	dir := filepath.Join(s.boxes, recipient, "messages")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []attention.MailEnvelope{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]attention.MailEnvelope, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			v, err := s.Get(recipient, strings.TrimSuffix(e.Name(), ".json"))
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt != result[j].CreatedAt {
			return result[i].CreatedAt > result[j].CreatedAt
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (s *Store) Receipt(recipient, id string) (attention.MailReceipt, error) {
	if !validPathID(recipient) || !validMailID(id) {
		return attention.MailReceipt{}, errors.New("invalid mail storage identifier")
	}
	var r attention.MailReceipt
	b, err := os.ReadFile(filepath.Join(s.boxes, recipient, "receipts", id+".json"))
	if os.IsNotExist(err) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(b, &r)
	if err == nil {
		if invalid := attention.ValidateMailReceipt(r); invalid != nil {
			err = invalid
		}
	}
	return r, err
}

func (s *Store) UpdateReceipt(recipient, id string, revision int64, change func(*attention.MailReceipt)) (attention.MailReceipt, error) {
	if !validPathID(recipient) || !validMailID(id) {
		return attention.MailReceipt{}, errors.New("invalid mail storage identifier")
	}
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return attention.MailReceipt{}, err
	}
	defer lock.Close()
	env, err := s.Get(recipient, id)
	if err != nil {
		return attention.MailReceipt{}, err
	}
	if env.Revision != revision {
		return attention.MailReceipt{}, errors.New("stale envelope revision")
	}
	r, err := s.Receipt(recipient, id)
	if errors.Is(err, ErrNotFound) {
		r = attention.MailReceipt{SchemaVersion: attention.SchemaVersion, ID: "receipt_" + id, EnvelopeID: id, Recipient: env.Recipient, CreatedAt: env.CreatedAt, EnvelopeRevision: revision}
	} else if err != nil {
		return attention.MailReceipt{}, err
	}
	change(&r)
	if invalid := attention.ValidateMailReceipt(r); invalid != nil {
		return r, invalid
	}
	path := filepath.Join(s.boxes, recipient, "receipts", id+".json")
	if err := secureDir(filepath.Dir(path)); err != nil {
		return r, err
	}
	if err := atomicReplace(path, r); err != nil {
		return r, err
	}
	return r, nil
}

func secureDir(p string) error {
	if err := os.MkdirAll(p, 0o700); err != nil {
		return err
	}
	return os.Chmod(p, 0o700)
}
func validPathID(v string) bool {
	if v == "" || v == "." || v == ".." {
		return false
	}
	for _, r := range v {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}
func atomicCreate(path string, v any) error {
	if _, err := os.Stat(path); err == nil {
		return os.ErrExist
	}
	return atomicWrite(path, v, false)
}
func atomicReplace(path string, v any) error { return atomicWrite(path, v, true) }
func atomicWrite(path string, v any, replace bool) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mail-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if !replace {
		if _, err := os.Stat(path); err == nil {
			return os.ErrExist
		}
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
