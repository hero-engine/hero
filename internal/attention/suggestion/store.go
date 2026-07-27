// Package suggestion implements private deferred-work proposals. Suggestions
// are advisory records, deliberately separate from committed Focus items.
package suggestion

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/filelock"
)

var (
	ErrNotFound            = errors.New("suggestion not found")
	ErrStale               = errors.New("stale suggestion revision")
	ErrIdempotencyConflict = errors.New("suggestion idempotency conflict")
)

const (
	StatePending   = "pending"
	StateAccepted  = "accepted"
	StateDismissed = "dismissed"
	StateExpired   = "expired"
)

type Item struct {
	SchemaVersion  int                            `json:"schema_version"`
	ID             string                         `json:"id"`
	Kind           string                         `json:"kind"`
	Title          string                         `json:"title"`
	Reason         string                         `json:"reason"`
	Prompt         string                         `json:"prompt"`
	Project        *attention.ProjectReference    `json:"project,omitempty"`
	Provenance     *attention.ProvenanceReference `json:"provenance,omitempty"`
	IdempotencyKey string                         `json:"idempotency_key"`
	CreatedAt      string                         `json:"created_at"`
	UpdatedAt      string                         `json:"updated_at"`
	ExpiresAt      string                         `json:"expires_at"`
	RetainUntil    string                         `json:"retain_until"`
	Revision       int64                          `json:"revision"`
	State          string                         `json:"state"`
	Decision       string                         `json:"decision,omitempty"`
	ActionKey      string                         `json:"action_idempotency_key,omitempty"`
	FocusID        string                         `json:"focus_id,omitempty"`
	FocusRevision  int64                          `json:"focus_revision,omitempty"`
}

type Store struct {
	dir      string
	lockPath string
	// beforeWrite is a test seam used to prove recovery after Focus creation.
	beforeWrite func(Item) error
}

func NewStore(stateRoot string) (*Store, error) {
	dir := filepath.Join(stateRoot, attentionstate.FocusDirectory, "suggestions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create suggestion directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, fmt.Errorf("secure suggestion directory: %w", err)
	}
	return &Store{dir: dir, lockPath: filepath.Join(dir, ".lock")}, nil
}

func (s *Store) CreateOrGet(item Item) (Item, bool, error) {
	if err := validateID(item.ID); err != nil {
		return Item{}, false, err
	}
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return Item{}, false, err
	}
	defer lock.Close()
	items, err := s.listUnlocked()
	if err != nil {
		return Item{}, false, err
	}
	for _, existing := range items {
		if existing.IdempotencyKey == item.IdempotencyKey {
			if proposalEqual(existing, item) {
				return existing, false, nil
			}
			return Item{}, false, ErrIdempotencyConflict
		}
	}
	item.Revision, err = revisionFor(item)
	if err != nil {
		return Item{}, false, err
	}
	if err := s.write(item); err != nil {
		return Item{}, false, err
	}
	return item, true, nil
}

func (s *Store) Get(id string) (Item, error) {
	if err := validateID(id); err != nil {
		return Item{}, err
	}
	b, err := os.ReadFile(s.itemPath(id))
	if os.IsNotExist(err) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, fmt.Errorf("read suggestion: %w", err)
	}
	var item Item
	if err := json.Unmarshal(b, &item); err != nil {
		return Item{}, fmt.Errorf("decode suggestion %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) List() ([]Item, error) { return s.listUnlocked() }

func (s *Store) Replace(id string, expected int64, change func(*Item) (bool, error)) (Item, error) {
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return Item{}, err
	}
	defer lock.Close()
	item, err := s.Get(id)
	if err != nil {
		return Item{}, err
	}
	if item.Revision != expected {
		return Item{}, ErrStale
	}
	changed, err := change(&item)
	if err != nil {
		return Item{}, err
	}
	if !changed {
		return item, nil
	}
	item.Revision, err = revisionFor(item)
	if err != nil {
		return Item{}, err
	}
	if err := s.write(item); err != nil {
		return Item{}, err
	}
	return item, nil
}

// Cleanup expires pending proposals after seven days and removes all proposal
// records after their 30-day audit window. It runs only during store access.
func (s *Store) Cleanup(now time.Time) error {
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return err
	}
	defer lock.Close()
	items, err := s.listUnlocked()
	if err != nil {
		return err
	}
	for _, item := range items {
		retain, err := time.Parse(time.RFC3339Nano, item.RetainUntil)
		if err != nil {
			return fmt.Errorf("parse retain_until for %s: %w", item.ID, err)
		}
		if !now.Before(retain) {
			if err := os.Remove(s.itemPath(item.ID)); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		if item.State != StatePending {
			continue
		}
		expires, err := time.Parse(time.RFC3339Nano, item.ExpiresAt)
		if err != nil {
			return fmt.Errorf("parse expires_at for %s: %w", item.ID, err)
		}
		if !now.Before(expires) {
			item.State = StateExpired
			item.UpdatedAt = utcNow(now)
			item.Revision, err = revisionFor(item)
			if err != nil {
				return err
			}
			if err := s.write(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) listUnlocked() ([]Item, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("list suggestions: %w", err)
	}
	items := make([]Item, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		item, err := s.Get(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt != items[j].CreatedAt {
			return items[i].CreatedAt > items[j].CreatedAt
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (s *Store) write(item Item) error {
	if s.beforeWrite != nil {
		if err := s.beforeWrite(item); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(s.dir, ".suggestion-*.tmp")
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
	if err := os.Rename(name, s.itemPath(item.ID)); err != nil {
		return err
	}
	return os.Chmod(s.itemPath(item.ID), 0o600)
}

func (s *Store) itemPath(id string) string { return filepath.Join(s.dir, id+".json") }

func revisionFor(item Item) (int64, error) {
	item.Revision = 0
	b, err := json.Marshal(item)
	if err != nil {
		return 0, err
	}
	sum := sha256.Sum256(b)
	revision := int64(binary.BigEndian.Uint64(sum[:8]) & (1<<63 - 1))
	if revision == 0 {
		revision = 1
	}
	return revision, nil
}

func proposalEqual(a, b Item) bool {
	return a.Kind == b.Kind && a.Title == b.Title && a.Reason == b.Reason &&
		a.Prompt == b.Prompt && equalProject(a.Project, b.Project) &&
		equalProvenance(a.Provenance, b.Provenance)
}

func equalProject(a, b *attention.ProjectReference) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalProvenance(a, b *attention.ProvenanceReference) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func validateID(id string) error {
	if !strings.HasPrefix(id, "suggestion_") || len(id) == len("suggestion_") {
		return errors.New("invalid suggestion id")
	}
	for _, r := range id[len("suggestion_"):] {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return errors.New("invalid suggestion id")
		}
	}
	return nil
}

func utcNow(now time.Time) string { return now.UTC().Format(time.RFC3339Nano) }

type fileLock struct{ lock *filelock.Lock }

func acquireLock(path string) (*fileLock, error) {
	lock, err := filelock.Acquire(path, 0o600)
	if err != nil {
		return nil, fmt.Errorf("lock suggestion store: %w", err)
	}
	return &fileLock{lock: lock}, nil
}

func (l *fileLock) Close() error {
	if err := l.lock.Close(); err != nil {
		return fmt.Errorf("close suggestion store lock: %w", err)
	}
	return nil
}
