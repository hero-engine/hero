package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/hero-engine/hero/internal/propose"
)

// proposalStores holds one in-memory proposal store per project.
// Per-project isolation keeps multi-project daemons honest (one
// project cannot see another's pending proposals).
type proposalStores struct {
	mu     sync.Mutex
	stores map[string]*propose.Store // project slug → store
}

func newProposalStores() *proposalStores {
	return &proposalStores{stores: make(map[string]*propose.Store)}
}

func (p *proposalStores) get(project string) *propose.Store {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stores == nil {
		p.stores = make(map[string]*propose.Store)
	}
	st, ok := p.stores[project]
	if !ok {
		st = propose.NewStore()
		p.stores[project] = st
	}
	return st
}

// routeProposals dispatches /sessions/{session_id}/proposals[/...]
// requests. The caller has already trimmed the prefix to leave
// `extra` as either "" (list), "ingest", "bulk/<action>", or
// "<proposal_id>/<action>".
func (a *API) routeProposals(w http.ResponseWriter, r *http.Request, pc *ProjectContext, sessionID, extra string) {
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	store := a.proposalStore(pc.Slug)

	switch {
	case extra == "" || extra == "/":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		a.handleProposalsList(w, r, store, sessionID)
		return

	case extra == "ingest":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		a.handleProposalIngest(w, r, store, pc.Slug, sessionID)
		return

	case strings.HasPrefix(extra, "bulk/"):
		action := strings.TrimPrefix(extra, "bulk/")
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		a.handleProposalBulk(w, r, store, pc.Slug, sessionID, action)
		return
	}

	// {proposal_id}/{action}
	parts := strings.SplitN(extra, "/", 2)
	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "unknown proposals endpoint")
		return
	}
	proposalID, action := parts[0], parts[1]
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	a.handleProposalAction(w, r, store, pc.Slug, sessionID, proposalID, action)
}

func (a *API) proposalStore(project string) *propose.Store {
	if a.server == nil {
		// Fallback for tests that hold a bare API. Use a singleton via the API.
		if a.proposals == nil {
			a.proposals = newProposalStores()
		}
		return a.proposals.get(project)
	}
	if a.proposals == nil {
		a.proposals = newProposalStores()
	}
	return a.proposals.get(project)
}

func (a *API) handleProposalsList(w http.ResponseWriter, r *http.Request, store *propose.Store, sessionID string) {
	specSlug := r.URL.Query().Get("spec_slug")
	batchID := r.URL.Query().Get("batch_id")
	agent := r.URL.Query().Get("agent")

	list := store.List(sessionID, specSlug, batchID, agent)
	if list == nil {
		list = []*propose.Envelope{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"proposals": list,
		"count":     len(list),
	})
}

func (a *API) handleProposalIngest(w http.ResponseWriter, r *http.Request, store *propose.Store, project, sessionID string) {
	defer r.Body.Close()
	var env propose.Envelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("json decode: %v", err))
		return
	}
	// The daemon trusts the agent's session_id but also accepts the URL value
	// if the body left it blank. They must match if both are set.
	if env.SessionID == "" {
		env.SessionID = sessionID
	} else if env.SessionID != sessionID {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("session_id mismatch: url=%q body=%q", sessionID, env.SessionID))
		return
	}

	res, err := store.Ingest(&env)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	a.bus.Publish(Event{
		Type:      EventProposalEmitted,
		Project:   project,
		SessionID: sessionID,
		Slug:      env.Target.SpecSlug,
		Payload:   res.Envelope,
	})

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"proposal_id": res.Envelope.ProposalID,
		"replaced_id": res.ReplacedID,
	})
}

type actionBody struct {
	By         string `json:"by,omitempty"`
	Reason     string `json:"reason,omitempty"`
	EditedBody string `json:"edited_body,omitempty"`
	BatchID    string `json:"batch_id,omitempty"`
}

func decodeAction(r *http.Request) actionBody {
	var body actionBody
	if r.Body != nil {
		defer r.Body.Close()
		// Empty body is fine.
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	return body
}

func actionToEventType(action propose.LifecycleAction) EventType {
	switch action {
	case propose.ActionAccepted:
		return EventProposalAccepted
	case propose.ActionEdited:
		return EventProposalEdited
	case propose.ActionRejected:
		return EventProposalRejected
	case propose.ActionDismissed:
		return EventProposalDismissed
	}
	return ""
}

func (a *API) handleProposalAction(w http.ResponseWriter, r *http.Request, store *propose.Store, project, sessionID, proposalID, action string) {
	body := decodeAction(r)

	var lifecycle propose.LifecycleAction
	switch action {
	case "accept":
		lifecycle = propose.ActionAccepted
	case "edit-accept":
		if body.EditedBody == "" {
			writeError(w, http.StatusBadRequest, "edited_body is required for edit-accept")
			return
		}
		lifecycle = propose.ActionEdited
	case "reject":
		lifecycle = propose.ActionRejected
	case "dismiss":
		lifecycle = propose.ActionDismissed
	default:
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown proposal action %q", action))
		return
	}

	rec, summary, err := store.Close(sessionID, proposalID, lifecycle, body.By, body.Reason, body.EditedBody)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	a.bus.Publish(Event{
		Type:      actionToEventType(lifecycle),
		Project:   project,
		SessionID: sessionID,
		Slug:      rec.SpecSlug,
		Payload:   rec,
	})

	if summary != nil {
		fmt.Fprintln(os.Stderr, summary.LogLine())
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"proposal_id": proposalID,
		"action":      action,
	})
}

func (a *API) handleProposalBulk(w http.ResponseWriter, r *http.Request, store *propose.Store, project, sessionID, action string) {
	body := decodeAction(r)
	if body.BatchID == "" {
		writeError(w, http.StatusBadRequest, "batch_id is required")
		return
	}

	var lifecycle propose.LifecycleAction
	switch action {
	case "accept":
		lifecycle = propose.ActionAccepted
	case "reject":
		lifecycle = propose.ActionRejected
	case "dismiss":
		lifecycle = propose.ActionDismissed
	case "edit-accept":
		writeError(w, http.StatusBadRequest, "bulk edit-accept is not supported; edit is per-proposal")
		return
	default:
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown bulk action %q", action))
		return
	}

	records, summary, err := store.CloseBatch(sessionID, body.BatchID, lifecycle, body.By)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	applied := make([]string, 0, len(records))
	evType := actionToEventType(lifecycle)
	for _, rec := range records {
		applied = append(applied, rec.ProposalID)
		a.bus.Publish(Event{
			Type:      evType,
			Project:   project,
			SessionID: sessionID,
			Slug:      rec.SpecSlug,
			Payload:   rec,
		})
	}

	if summary != nil {
		fmt.Fprintln(os.Stderr, summary.LogLine())
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"applied": applied,
		"count":   len(applied),
	})
}
