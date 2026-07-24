package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

type mcpFocusResolver struct {
	ref  *attention.ProjectReference
	path string
	err  error
}

func (r mcpFocusResolver) ResolveReference(ref *attention.ProjectReference) focus.ResolvedProject {
	if ref != nil && r.ref != nil && ref.PeerID == r.ref.PeerID {
		return focus.ResolvedProject{Reference: ref, Availability: focus.ProjectAvailable, Path: r.path}
	}
	return focus.ResolvedProject{Reference: ref, Availability: focus.ProjectMissing}
}
func (r mcpFocusResolver) ResolveInput(value string) (*attention.ProjectReference, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.ref, nil
}
func (r mcpFocusResolver) ResolveCurrent() (*attention.ProjectReference, error) { return r.ref, nil }

func TestMCPFocusSuggestionToolsAreStructuredAndConsentBounded(t *testing.T) {
	stateRoot := t.TempDir()
	ref := &attention.ProjectReference{PeerID: "peer-demo", RegistrySlug: "demo", DisplayName: "Demo"}
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	srv.attentionStateRoot = stateRoot
	srv.attentionResolver = mcpFocusResolver{ref: ref, path: filepath.Join(t.TempDir(), "demo")}

	out, err := srv.toolFocusSuggest(map[string]interface{}{
		"title": "Cache issue", "reason": "Separate useful work", "prompt": "Investigate the cache.\n", "project": "demo",
		"source_kind": "run", "source_id": "run-1", "idempotency_key": "run-1:cache",
	})
	if err != nil {
		t.Fatal(err)
	}
	var created suggestion.Presented
	if err := json.Unmarshal([]byte(out), &created); err != nil || created.State != suggestion.StatePending || len(created.Actions) != 4 {
		t.Fatalf("suggest = %q, %v", out, err)
	}
	focusStore, _ := focus.NewStore(stateRoot)
	if items, _ := focusStore.List(); len(items) != 0 {
		t.Fatalf("suggest created Focus: %#v", items)
	}

	out, err = srv.toolFocusSuggestions(map[string]interface{}{"pending": true})
	if err != nil || !json.Valid([]byte(out)) || !containsJSONID(out, created.ID) {
		t.Fatalf("list = %q, %v", out, err)
	}

	out, err = srv.toolFocusSuggestionAction(map[string]interface{}{"suggestion_id": created.ID, "action": "do_next", "revision": fmt.Sprintf("%d", created.Revision), "idempotency_key": "accept-1"})
	if err != nil {
		t.Fatal(err)
	}
	var result suggestion.ActionResult
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.Focus == nil || result.Launch == nil || result.Launch.Prompt != created.Prompt {
		t.Fatalf("action = %q, %v", out, err)
	}

	errorJSON, err := srv.toolFocusSuggestionAction(map[string]interface{}{"suggestion_id": created.ID, "action": "today", "revision": fmt.Sprintf("%d", created.Revision), "idempotency_key": "another"})
	if err != nil || !json.Valid([]byte(errorJSON)) || !containsJSONID(errorJSON, "stale") {
		t.Fatalf("structured error = %q, %v", errorJSON, err)
	}
}

func TestMCPFocusCreateRequiresUserAndReplaysExactlyOnce(t *testing.T) {
	stateRoot := t.TempDir()
	ref := &attention.ProjectReference{PeerID: "peer-demo", RegistrySlug: "demo", DisplayName: "Demo"}
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	srv.attentionStateRoot = stateRoot
	srv.attentionResolver = mcpFocusResolver{ref: ref, path: filepath.Join(t.TempDir(), "demo")}

	args := map[string]interface{}{
		"schema_version": float64(1), "intent_source": "user", "title": "Remember this",
		"prompt": "Resume this exact prompt.\n", "lifecycle": attention.FocusToday,
		"project": "demo", "project_peer_id": "peer-demo", "source_id": "session_1", "idempotency_key": "focus_1",
	}
	out, err := srv.toolFocusCreate(args)
	if err != nil {
		t.Fatal(err)
	}
	var result attention.ActionResult
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.Error != nil {
		t.Fatalf("create = %s, %v", out, err)
	}
	var created focus.Item
	if err := json.Unmarshal(result.Source, &created); err != nil || created.Lifecycle != attention.FocusToday || created.Prompt != args["prompt"] || created.Project == nil || created.Project.PeerID != "peer-demo" {
		t.Fatalf("focus = %#v, %v", created, err)
	}
	replay, _ := srv.toolFocusCreate(args)
	var replayResult attention.ActionResult
	_ = json.Unmarshal([]byte(replay), &replayResult)
	var replayed focus.Item
	_ = json.Unmarshal(replayResult.Source, &replayed)
	if replayed.ID != created.ID || replayed.Revision != created.Revision {
		t.Fatalf("replay duplicated: %#v %#v", created, replayed)
	}
	args["prompt"] = "different"
	conflict, _ := srv.toolFocusCreate(args)
	var conflictResult attention.ActionResult
	_ = json.Unmarshal([]byte(conflict), &conflictResult)
	if conflictResult.Error == nil || conflictResult.Error.Code != attention.ErrorIdempotencyConflict {
		t.Fatalf("conflict = %s", conflict)
	}

	args["idempotency_key"] = "focus_model"
	args["intent_source"] = "model"
	denied, _ := srv.toolFocusCreate(args)
	var deniedResult attention.ActionResult
	_ = json.Unmarshal([]byte(denied), &deniedResult)
	if deniedResult.Error == nil || deniedResult.Error.Code != attention.ErrorPermission {
		t.Fatalf("model create = %s", denied)
	}
	store, _ := focus.NewStore(stateRoot)
	items, _ := store.List()
	if len(items) != 1 {
		t.Fatalf("denied create mutated Focus: %#v", items)
	}
	args["intent_source"] = "user"
	args["idempotency_key"] = "focus_wrong_project"
	args["project_peer_id"] = "peer-other"
	mismatch, _ := srv.toolFocusCreate(args)
	var mismatchResult attention.ActionResult
	_ = json.Unmarshal([]byte(mismatch), &mismatchResult)
	if mismatchResult.Error == nil || mismatchResult.Error.Code != attention.ErrorValidation || mismatchResult.Error.Field != "project_peer_id" {
		t.Fatalf("project mismatch = %s", mismatch)
	}
	items, _ = store.List()
	if len(items) != 1 {
		t.Fatalf("project mismatch mutated Focus: %#v", items)
	}

	unboundArgs := map[string]interface{}{
		"schema_version": float64(1), "intent_source": "user", "title": "Unbound",
		"prompt": "Work anywhere.", "lifecycle": attention.FocusInbox,
		"source_id": "session_2", "idempotency_key": "focus_2",
	}
	unboundOut, _ := srv.toolFocusCreate(unboundArgs)
	var unboundResult attention.ActionResult
	_ = json.Unmarshal([]byte(unboundOut), &unboundResult)
	var unbound focus.Item
	_ = json.Unmarshal(unboundResult.Source, &unbound)
	if unbound.Project != nil {
		t.Fatalf("unbound request inferred project: %#v", unbound)
	}

	maxKeyArgs := map[string]interface{}{
		"schema_version": float64(1), "intent_source": "user", "title": "Max key",
		"prompt": "Keep the entire external key valid.", "lifecycle": attention.FocusLater,
		"source_id": "session_3", "idempotency_key": strings.Repeat("k", 512),
	}
	maxKeyOut, _ := srv.toolFocusCreate(maxKeyArgs)
	var maxKeyResult attention.ActionResult
	_ = json.Unmarshal([]byte(maxKeyOut), &maxKeyResult)
	var maxKeyItem focus.Item
	_ = json.Unmarshal(maxKeyResult.Source, &maxKeyItem)
	if maxKeyResult.Error != nil || maxKeyItem.ID == "" || len(maxKeyItem.OriginKey) > 512 {
		t.Fatalf("maximum key = %s", maxKeyOut)
	}

	unavailableServer := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	unavailableServer.attentionStateRoot = t.TempDir()
	unavailableServer.attentionResolver = mcpFocusResolver{err: errors.New("registry read failed")}
	unavailableOut, _ := unavailableServer.toolFocusCreate(map[string]interface{}{
		"schema_version": float64(1), "intent_source": "user", "title": "Unavailable",
		"prompt": "Do not create.", "lifecycle": attention.FocusInbox, "project": "broken",
		"project_peer_id": "peer-broken", "source_id": "session_4", "idempotency_key": "focus_unavailable",
	})
	var unavailableResult attention.ActionResult
	_ = json.Unmarshal([]byte(unavailableOut), &unavailableResult)
	if unavailableResult.Error == nil || unavailableResult.Error.Code != attention.ErrorUnavailable {
		t.Fatalf("unavailable project authority = %s", unavailableOut)
	}
}

func containsJSONID(value, fragment string) bool {
	var decoded any
	if json.Unmarshal([]byte(value), &decoded) != nil {
		return false
	}
	b, _ := json.Marshal(decoded)
	for i := 0; i+len(fragment) <= len(b); i++ {
		if string(b[i:i+len(fragment)]) == fragment {
			return true
		}
	}
	return false
}

func TestMCPFocusSuggestionDefinitionsAndDispatch(t *testing.T) {
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	for _, name := range []string{"hero_focus_create", "hero_focus_suggest", "hero_focus_suggestions", "hero_focus_suggestion_action"} {
		if _, ok := srv.toolHandlers()[name]; !ok {
			t.Errorf("missing handler %s", name)
		}
		found := false
		for _, def := range srv.toolDefinitions() {
			if def.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing definition %s", name)
		}
	}
}
