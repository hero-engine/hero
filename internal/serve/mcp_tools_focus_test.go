package serve

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

type mcpFocusResolver struct {
	ref  *attention.ProjectReference
	path string
}

func (r mcpFocusResolver) ResolveReference(ref *attention.ProjectReference) focus.ResolvedProject {
	if ref != nil && r.ref != nil && ref.PeerID == r.ref.PeerID {
		return focus.ResolvedProject{Reference: ref, Availability: focus.ProjectAvailable, Path: r.path}
	}
	return focus.ResolvedProject{Reference: ref, Availability: focus.ProjectMissing}
}
func (r mcpFocusResolver) ResolveInput(value string) (*attention.ProjectReference, error) {
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
	for _, name := range []string{"hero_focus_suggest", "hero_focus_suggestions", "hero_focus_suggestion_action"} {
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
