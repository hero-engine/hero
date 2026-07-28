package codehost

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/filelock"
	herospec "github.com/hero-engine/hero/internal/spec"
)

const (
	mutationJournalVersion  = 1
	mutationJournalMaxAge   = 30 * 24 * time.Hour
	maxMutationJournalBytes = codehostbroker.MaxJournalEntries * (16 << 10)
)

type journalState string

const (
	journalInProgress journalState = "in_progress"
	journalDispatched journalState = "dispatched"
	journalApplied    journalState = "applied"
	journalExternal   journalState = "externally_completed"
	journalNotApplied journalState = "not_applied"
	journalAmbiguous  journalState = "ambiguous"
)

type mutationJournal struct {
	directory  string
	path       string
	lockPath   string
	maxEntries int
	maxAge     time.Duration
	now        func() time.Time
}

type journalDocument struct {
	Version int                      `json:"version"`
	Entries map[string]*journalEntry `json:"entries"`
}

type journalEntry struct {
	KeyDigest        string          `json:"key_digest"`
	PayloadDigest    string          `json:"payload_digest"`
	OperationID      string          `json:"operation_id"`
	Target           mutationTarget  `json:"target"`
	State            journalState    `json:"state"`
	Receipt          *journalReceipt `json:"receipt,omitempty"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
	ReconciledAt     string          `json:"reconciled_at,omitempty"`
	ProviderAttempts int             `json:"provider_attempts"`
	FailureCode      string          `json:"failure_code,omitempty"`
}

// mutationTarget is deliberately content-free: it is sufficient to query the
// provider for the exact effect, but cannot reveal a PR title or body.
type mutationTarget struct {
	ConnectionID string                            `json:"connection_id"`
	Repository   codehostbroker.RepositoryIdentity `json:"repository"`
	Base         codehostbroker.RefIdentity        `json:"base"`
	Head         codehostbroker.RefIdentity        `json:"head"`
}

// journalReceipt retains only provider identity and revisions. The normalized
// PR is read back when a response must be replayed.
type journalReceipt struct {
	Identity codehostbroker.PullRequestIdentity `json:"identity"`
	Base     codehostbroker.RefIdentity         `json:"base"`
	Head     codehostbroker.RefIdentity         `json:"head"`
}

func newMutationJournal(projectRoot string, now func() time.Time) *mutationJournal {
	directory := filepath.Join(projectRoot, ".hero", "cache", "code-host-broker", "v1")
	return &mutationJournal{
		directory:  directory,
		path:       filepath.Join(directory, "mutation-journal.json"),
		lockPath:   filepath.Join(directory, "mutation-journal.lock"),
		maxEntries: codehostbroker.MaxJournalEntries,
		maxAge:     mutationJournalMaxAge,
		now:        now,
	}
}

func (j *mutationJournal) withLock(function func(*journalDocument) error) error {
	if err := os.MkdirAll(j.directory, 0o700); err != nil {
		return fmt.Errorf("create mutation journal directory: %w", err)
	}
	if err := os.Chmod(j.directory, 0o700); err != nil {
		return fmt.Errorf("secure mutation journal directory: %w", err)
	}
	lock, err := filelock.Acquire(j.lockPath, 0o600)
	if err != nil {
		return fmt.Errorf("lock mutation journal: %w", err)
	}
	defer lock.Close()

	document, err := j.load()
	if err != nil {
		return err
	}
	j.prune(document)
	if err := function(document); err != nil {
		return err
	}
	return j.save(document)
}

func (j *mutationJournal) load() (*journalDocument, error) {
	data, err := os.ReadFile(j.path)
	if errors.Is(err, os.ErrNotExist) {
		return &journalDocument{Version: mutationJournalVersion, Entries: map[string]*journalEntry{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read mutation journal: %w", err)
	}
	if len(data) > maxMutationJournalBytes {
		return nil, errors.New("mutation journal exceeds its read bound")
	}
	var document journalDocument
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return nil, errors.New("mutation journal is invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("mutation journal contains trailing data")
	}
	if document.Version != mutationJournalVersion || document.Entries == nil {
		return nil, errors.New("mutation journal version is invalid")
	}
	return &document, nil
}

func (j *mutationJournal) save(document *journalDocument) error {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return errors.New("encode mutation journal")
	}
	data = append(data, '\n')
	if err := herospec.AtomicWriteFile(j.path, data, 0o600); err != nil {
		return fmt.Errorf("write mutation journal: %w", err)
	}
	return os.Chmod(j.path, 0o600)
}

func (j *mutationJournal) prune(document *journalDocument) {
	cutoff := j.now().Add(-j.maxAge)
	type candidate struct {
		key       string
		updatedAt time.Time
	}
	expired := make([]candidate, 0)
	for key, entry := range document.Entries {
		if !terminalJournalState(entry.State) {
			continue
		}
		updatedAt, err := time.Parse(time.RFC3339Nano, entry.UpdatedAt)
		if err != nil || updatedAt.Before(cutoff) {
			expired = append(expired, candidate{key: key, updatedAt: updatedAt})
		}
	}
	sort.Slice(expired, func(left, right int) bool {
		return expired[left].updatedAt.Before(expired[right].updatedAt)
	})
	for _, item := range expired {
		delete(document.Entries, item.key)
	}
}

func terminalJournalState(state journalState) bool {
	switch state {
	case journalApplied, journalExternal, journalNotApplied:
		return true
	default:
		return false
	}
}

func (j *mutationJournal) timestamp() string {
	return j.now().UTC().Format(time.RFC3339Nano)
}
