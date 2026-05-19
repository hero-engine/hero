package propose

import (
	"fmt"
	"sync"
	"time"
)

// LifecycleAction is the action that closed a proposal's life.
type LifecycleAction string

const (
	ActionAccepted  LifecycleAction = "accepted"
	ActionEdited    LifecycleAction = "edited"
	ActionRejected  LifecycleAction = "rejected"
	ActionDismissed LifecycleAction = "dismissed"
)

// LifecycleRecord is the payload emitted on the SSE stream when a
// proposal closes. It is also what the dashboard renders in the chat
// log scroll.
type LifecycleRecord struct {
	ProposalID string          `json:"proposal_id"`
	BatchID    string          `json:"batch_id"`
	SessionID  string          `json:"session_id"`
	SpecSlug   string          `json:"spec_slug"`
	Agent      string          `json:"agent"`
	Anchor     Anchor          `json:"anchor"`
	Action     LifecycleAction `json:"action"`
	By         string          `json:"by,omitempty"`
	Reason     string          `json:"reason,omitempty"`
	EditedBody string          `json:"edited_body,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// BatchSummary is the rolled-up batch result the daemon logs once
// every proposal in a batch has closed. It is also the payload of
// optional batch-close SSE events (not part of v1.0 — left for a
// future minor bump).
type BatchSummary struct {
	BatchID   string `json:"batch_id"`
	SessionID string `json:"session_id"`
	Agent     string `json:"agent"`
	Total     int    `json:"total"`
	Accepted  int    `json:"accepted"`
	Edited    int    `json:"edited"`
	Rejected  int    `json:"rejected"`
	Dismissed int    `json:"dismissed"`
}

// LogLine renders the canonical lifecycle-summary string. Format is
// pinned by docs/contracts/inline-propose-v1.md §5.
func (s BatchSummary) LogLine() string {
	noun := "proposal"
	if s.Total != 1 {
		noun = "proposals"
	}
	return fmt.Sprintf("%s drafted %d %s → %d accepted, %d edited, %d rejected, %d dismissed",
		s.Agent, s.Total, noun, s.Accepted, s.Edited, s.Rejected, s.Dismissed)
}

// Store is the in-memory proposal store. Transient by design
// (Decision 1) — proposals live for the life of the daemon process,
// keyed by session.
type Store struct {
	mu sync.RWMutex

	// pending proposals keyed by proposal_id, then grouped by session.
	pending map[string]map[string]*Envelope // session_id → proposal_id → envelope

	// anchor index for per-anchor replacement lookups (Decision 2).
	anchorIdx map[anchorKey]string // anchorKey → proposal_id

	// batch tracking — total emitted + closure counts per batch.
	// Used to render the lifecycle summary when a batch closes.
	batches map[string]*batchState // batch_id → state
}

type batchState struct {
	sessionID string
	agent     string
	emitted   int
	accepted  int
	edited    int
	rejected  int
	dismissed int
	open      int // emitted - (accepted+edited+rejected+dismissed)
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{
		pending:   make(map[string]map[string]*Envelope),
		anchorIdx: make(map[anchorKey]string),
		batches:   make(map[string]*batchState),
	}
}

// IngestResult captures what happened during Ingest. When ReplacedID
// is non-empty, the new envelope replaced an existing pending
// proposal at the same anchor (same agent) — see Decision 2.
type IngestResult struct {
	Envelope   *Envelope
	ReplacedID string
}

// Ingest stores an envelope, applying per-anchor replacement scoped
// to the same agent. Returns the stored envelope (with EmittedAt
// stamped if it was missing) and the proposal_id of any prior
// envelope that was replaced.
func (s *Store) Ingest(e *Envelope) (*IngestResult, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if e.EmittedAt.IsZero() {
		e.EmittedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	res := &IngestResult{Envelope: e}

	// Per-anchor replacement (Decision 2)
	key := e.anchorKey()
	if priorID, ok := s.anchorIdx[key]; ok {
		// Remove prior envelope from pending; do not emit a lifecycle
		// event for the replaced proposal — replacement is silent so
		// the dashboard sees only the latest proposal at the anchor.
		if sess, ok := s.pending[e.SessionID]; ok {
			if prior, ok := sess[priorID]; ok {
				delete(sess, priorID)
				// Decrement the prior envelope's batch counts (it never
				// actually closed — it was replaced) so the batch
				// summary reflects what the user actually sees.
				if bs, ok := s.batches[prior.BatchID]; ok {
					bs.emitted--
					bs.open--
					if bs.emitted <= 0 {
						delete(s.batches, prior.BatchID)
					}
				}
			}
		}
		res.ReplacedID = priorID
	}

	// Index the new envelope.
	if _, ok := s.pending[e.SessionID]; !ok {
		s.pending[e.SessionID] = make(map[string]*Envelope)
	}
	s.pending[e.SessionID][e.ProposalID] = e
	s.anchorIdx[key] = e.ProposalID

	bs, ok := s.batches[e.BatchID]
	if !ok {
		bs = &batchState{sessionID: e.SessionID, agent: e.Agent}
		s.batches[e.BatchID] = bs
	}
	bs.emitted++
	bs.open++

	return res, nil
}

// Get returns the pending envelope for a (session, proposal_id) pair.
func (s *Store) Get(sessionID, proposalID string) (*Envelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.pending[sessionID]
	if !ok {
		return nil, false
	}
	e, ok := sess[proposalID]
	return e, ok
}

// List returns pending envelopes in a session, optionally filtered by
// spec_slug, batch_id, and agent. Empty filters match all.
func (s *Store) List(sessionID, specSlug, batchID, agent string) []*Envelope {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.pending[sessionID]
	if !ok {
		return nil
	}
	out := make([]*Envelope, 0, len(sess))
	for _, e := range sess {
		if specSlug != "" && e.Target.SpecSlug != specSlug {
			continue
		}
		if batchID != "" && e.BatchID != batchID {
			continue
		}
		if agent != "" && e.Agent != agent {
			continue
		}
		out = append(out, e)
	}
	return out
}

// Close removes a pending proposal and records its action. Returns
// the LifecycleRecord that should be emitted on the SSE stream, plus
// an optional BatchSummary if this close drained the batch.
func (s *Store) Close(sessionID, proposalID string, action LifecycleAction, by, reason, editedBody string) (*LifecycleRecord, *BatchSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.pending[sessionID]
	if !ok {
		return nil, nil, fmt.Errorf("session %q has no pending proposals", sessionID)
	}
	e, ok := sess[proposalID]
	if !ok {
		return nil, nil, fmt.Errorf("proposal %q not found in session %q", proposalID, sessionID)
	}

	delete(sess, proposalID)
	delete(s.anchorIdx, e.anchorKey())

	rec := &LifecycleRecord{
		ProposalID: e.ProposalID,
		BatchID:    e.BatchID,
		SessionID:  e.SessionID,
		SpecSlug:   e.Target.SpecSlug,
		Agent:      e.Agent,
		Anchor:     e.Target.Anchor,
		Action:     action,
		By:         by,
		Reason:     reason,
		EditedBody: editedBody,
		Timestamp:  time.Now().UTC(),
	}

	var summary *BatchSummary
	if bs, ok := s.batches[e.BatchID]; ok {
		switch action {
		case ActionAccepted:
			bs.accepted++
		case ActionEdited:
			bs.edited++
		case ActionRejected:
			bs.rejected++
		case ActionDismissed:
			bs.dismissed++
		}
		bs.open--
		if bs.open <= 0 {
			summary = &BatchSummary{
				BatchID:   e.BatchID,
				SessionID: bs.sessionID,
				Agent:     bs.agent,
				Total:     bs.emitted,
				Accepted:  bs.accepted,
				Edited:    bs.edited,
				Rejected:  bs.rejected,
				Dismissed: bs.dismissed,
			}
			delete(s.batches, e.BatchID)
		}
	}

	return rec, summary, nil
}

// CloseBatch applies the given action to every pending proposal in a
// (session, batch) pair. Returns the lifecycle records emitted and an
// optional batch summary if the batch drained.
func (s *Store) CloseBatch(sessionID, batchID string, action LifecycleAction, by string) ([]*LifecycleRecord, *BatchSummary, error) {
	switch action {
	case ActionAccepted, ActionRejected, ActionDismissed:
	default:
		return nil, nil, fmt.Errorf("action %q not supported for bulk close (edit is per-proposal)", action)
	}

	s.mu.Lock()
	// Collect IDs to close so we don't hold the lock through external work.
	var ids []string
	if sess, ok := s.pending[sessionID]; ok {
		for id, e := range sess {
			if e.BatchID == batchID {
				ids = append(ids, id)
			}
		}
	}
	s.mu.Unlock()

	if len(ids) == 0 {
		return nil, nil, fmt.Errorf("no pending proposals for batch %q in session %q", batchID, sessionID)
	}

	var records []*LifecycleRecord
	var summary *BatchSummary
	for _, id := range ids {
		rec, sum, err := s.Close(sessionID, id, action, by, "", "")
		if err != nil {
			return records, summary, err
		}
		records = append(records, rec)
		if sum != nil {
			summary = sum
		}
	}
	return records, summary, nil
}

// PendingCount returns the number of pending proposals in a session.
// Useful for tests and metrics.
func (s *Store) PendingCount(sessionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pending[sessionID])
}

// SnapshotAll returns every pending envelope across every session in
// the store. Used by the Now dashboard's inbox to surface all open
// proposals without needing a separate session enumerator. Order is
// stable within a session but not across sessions (map iteration).
func (s *Store) SnapshotAll() []*Envelope {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Envelope
	for _, sess := range s.pending {
		for _, e := range sess {
			out = append(out, e)
		}
	}
	return out
}
