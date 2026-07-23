// Package focus implements the private, user-global durable Focus store.
package focus

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
)

var (
	ErrNotFound            = errors.New("focus item not found")
	ErrStale               = errors.New("stale focus revision")
	ErrIdempotencyConflict = errors.New("focus idempotency conflict")
)

// Item is the persisted Focus representation. Project is a pointer here so an
// unbound personal intention can remain genuinely project-optional while the
// portable v1 contract retains its already-published value-shaped field.
type Item struct {
	SchemaVersion int                            `json:"schema_version"`
	ID            string                         `json:"id"`
	Project       *attention.ProjectReference    `json:"project,omitempty"`
	Title         string                         `json:"title"`
	Prompt        string                         `json:"prompt"`
	Lifecycle     string                         `json:"lifecycle"`
	Revision      int64                          `json:"revision"`
	CreatedAt     string                         `json:"created_at"`
	UpdatedAt     string                         `json:"updated_at"`
	Origin        *attention.ProvenanceReference `json:"origin,omitempty"`
	OriginKey     string                         `json:"origin_key,omitempty"`
}

type Store struct {
	dir      string
	itemsDir string
	lockPath string
}

func NewStore(stateRoot string) (*Store, error) {
	dir := filepath.Join(stateRoot, attentionstate.FocusDirectory)
	items := filepath.Join(dir, "items")
	if err := os.MkdirAll(items, 0o700); err != nil {
		return nil, fmt.Errorf("create focus items directory: %w", err)
	}
	for _, path := range []string{dir, items} {
		if err := os.Chmod(path, 0o700); err != nil {
			return nil, fmt.Errorf("secure focus directory: %w", err)
		}
	}
	return &Store{dir: dir, itemsDir: items, lockPath: filepath.Join(dir, ".lock")}, nil
}

func (s *Store) Create(item Item) (Item, error) {
	if err := validateID(item.ID); err != nil {
		return Item{}, err
	}
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return Item{}, err
	}
	defer lock.Close()
	if _, err := os.Stat(s.itemPath(item.ID)); err == nil {
		return Item{}, fmt.Errorf("focus item %q already exists", item.ID)
	} else if !os.IsNotExist(err) {
		return Item{}, err
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

// CreateOrGet atomically implements source-key idempotency across processes.
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
		if item.OriginKey != "" && existing.OriginKey == item.OriginKey {
			if replayEqual(existing, item) {
				return existing, false, nil
			}
			return Item{}, false, ErrIdempotencyConflict
		}
	}
	if _, err := os.Stat(s.itemPath(item.ID)); err == nil {
		return Item{}, false, fmt.Errorf("focus item %q already exists", item.ID)
	} else if !os.IsNotExist(err) {
		return Item{}, false, err
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
		return Item{}, fmt.Errorf("read focus item: %w", err)
	}
	var item Item
	if err := json.Unmarshal(b, &item); err != nil {
		return Item{}, fmt.Errorf("decode focus item %q: %w", id, err)
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

func (s *Store) listUnlocked() ([]Item, error) {
	entries, err := os.ReadDir(s.itemsDir)
	if err != nil {
		return nil, fmt.Errorf("list focus items: %w", err)
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
		if items[i].UpdatedAt != items[j].UpdatedAt {
			return items[i].UpdatedAt > items[j].UpdatedAt
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (s *Store) write(item Item) error {
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("encode focus item: %w", err)
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(s.itemsDir, ".focus-*.tmp")
	if err != nil {
		return fmt.Errorf("create focus temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
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
	if err := os.Rename(tmpName, s.itemPath(item.ID)); err != nil {
		return fmt.Errorf("replace focus item: %w", err)
	}
	if err := os.Chmod(s.itemPath(item.ID), 0o600); err != nil {
		return err
	}
	dir, err := os.Open(s.itemsDir)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (s *Store) itemPath(id string) string { return filepath.Join(s.itemsDir, id+".json") }

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

func replayEqual(existing, candidate Item) bool {
	existing.ID, candidate.ID = "", ""
	existing.Revision, candidate.Revision = 0, 0
	existing.CreatedAt, candidate.CreatedAt = "", ""
	existing.UpdatedAt, candidate.UpdatedAt = "", ""
	return existing.SchemaVersion == candidate.SchemaVersion &&
		existing.Title == candidate.Title && existing.Prompt == candidate.Prompt &&
		existing.Lifecycle == candidate.Lifecycle && existing.OriginKey == candidate.OriginKey &&
		equalProject(existing.Project, candidate.Project) && equalOrigin(existing.Origin, candidate.Origin)
}

func equalProject(a, b *attention.ProjectReference) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalOrigin(a, b *attention.ProvenanceReference) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func utcNow(now time.Time) string { return now.UTC().Format(time.RFC3339Nano) }
