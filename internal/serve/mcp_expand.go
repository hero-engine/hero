package serve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/refs"
)

// ensureRefsStore opens the session ref-store if not already open.
// Errors are returned, not logged — callers decide whether the tool
// can continue without it (most opt-in compact paths can't).
func (s *MCPServer) ensureRefsStore() (*refs.Store, error) {
	if s.refsStore != nil {
		return s.refsStore, nil
	}
	store, err := refs.Open(s.heroDir, s.sessionID)
	if err != nil {
		return nil, err
	}
	s.refsStore = store
	return store, nil
}

// registerRef builds an envelope and registers a ref in the store.
// Returns the rendered envelope text. If the store can't be opened,
// returns the body unchanged and a non-nil error so the tool can
// fall back to the legacy shape.
func (s *MCPServer) registerRef(kind refs.Kind, slug, scope string, sourceArgs map[string]any, content, fingerprint, summary string) (string, error) {
	store, err := s.ensureRefsStore()
	if err != nil {
		return "", err
	}
	refID, err := store.Register(kind, slug, scope, sourceArgs, content, fingerprint)
	if err != nil {
		return "", err
	}
	env := envelope{
		RefID:     refID,
		ExpandVia: "hero_expand",
		Kind:      string(kind),
		Slug:      slug,
		Scope:     scope,
		Summary:   summary,
	}
	return renderEnvelopeText(env), nil
}

// toolExpand resolves one or more ref IDs to their full content. It
// accepts either `ref_id` (string) or `ref_ids` (array of strings).
//
// Resolution order for each ref:
//  1. Look up in the session ref-store.
//  2. If found AND content is present AND fingerprint is non-empty,
//     return the cached content.
//  3. Otherwise, invoke the registered resolver for the kind to
//     refetch, persist, and return.
//  4. If no resolver is registered or refetch fails, return a
//     structured error block so the caller can re-fetch via the
//     producing tool.
func (s *MCPServer) toolExpand(args map[string]interface{}) (string, error) {
	ids, err := collectExpandIDs(args)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("ref_id or ref_ids parameter is required")
	}

	store, err := s.ensureRefsStore()
	if err != nil {
		return "", fmt.Errorf("opening ref-store: %w", err)
	}

	results := make([]string, 0, len(ids))
	for _, id := range ids {
		results = append(results, s.expandSingle(store, id))
	}

	if len(results) == 1 {
		return results[0], nil
	}

	// Multiple ids — emit a delimited block per ref.
	var b strings.Builder
	for i, r := range results {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		fmt.Fprintf(&b, "[ref %d/%d: %s]\n", i+1, len(results), ids[i])
		b.WriteString(r)
	}
	return b.String(), nil
}

func (s *MCPServer) expandSingle(store *refs.Store, refID string) string {
	if refID == "" {
		return errorBlock(refID, "empty ref_id", "")
	}
	entry, err := store.Lookup(refID)
	if err != nil {
		return errorBlock(refID, err.Error(), "")
	}
	if entry == nil {
		// Unknown ref — try to give the model a useful retry hint by
		// parsing the kind+slug. Phase 1: caller refetches via the
		// producing tool. Future v2 may include the args needed.
		kind, slug, _, perr := refs.ParseRefID(refID)
		hint := ""
		if perr == nil {
			hint = retryHint(kind, slug)
		}
		return errorBlock(refID, "unknown or expired ref", hint)
	}

	// Cache hit — return cached content if we have any.
	if entry.Content != "" {
		return entry.Content
	}

	// Empty content — invoke resolver to populate.
	content, fingerprint, err := s.refsRegistry.Resolve(entry)
	if err != nil {
		return errorBlock(refID, err.Error(), retryHint(entry.Kind, entry.Slug))
	}
	if uErr := store.UpdateContent(refID, content, fingerprint); uErr != nil {
		// Non-fatal — return the fresh content even if cache update
		// failed.
		_ = uErr
	}
	return content
}

func collectExpandIDs(args map[string]interface{}) ([]string, error) {
	var ids []string
	if raw, ok := args["ref_ids"]; ok {
		switch v := raw.(type) {
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					ids = append(ids, s)
				}
			}
		case []string:
			for _, s := range v {
				if s != "" {
					ids = append(ids, s)
				}
			}
		case string:
			if v != "" {
				if err := json.Unmarshal([]byte(v), &ids); err != nil {
					// Maybe a comma-separated list.
					for _, p := range strings.Split(v, ",") {
						p = strings.TrimSpace(p)
						if p != "" {
							ids = append(ids, p)
						}
					}
				}
			}
		default:
			return nil, fmt.Errorf("ref_ids must be string array or comma-separated string")
		}
	}
	if single, ok := args["ref_id"].(string); ok && single != "" {
		ids = append(ids, single)
	}
	return ids, nil
}

func errorBlock(refID, msg, hint string) string {
	var b strings.Builder
	b.WriteString("[hero expand error]\n")
	if refID != "" {
		fmt.Fprintf(&b, "ref_id: %s\n", refID)
	}
	fmt.Fprintf(&b, "error: %s\n", msg)
	if hint != "" {
		fmt.Fprintf(&b, "rehydrate_via: %s\n", hint)
	}
	b.WriteString("[/hero expand error]")
	return b.String()
}

// retryHint suggests which producing tool can rehydrate a lost ref.
// Used in error blocks so the model can self-recover.
func retryHint(kind refs.Kind, slug string) string {
	switch kind {
	case refs.KindSpec:
		return fmt.Sprintf("hero_read_spec slug=%s", slug)
	case refs.KindConvention, refs.KindDecision, refs.KindRule:
		return fmt.Sprintf("hero_search query=%s type=%s", slug, kind)
	case refs.KindSearch:
		return "hero_search (re-run with original query)"
	case refs.KindContext:
		return "hero_context (re-run with original files)"
	case refs.KindRecap:
		return "hero_recap"
	case refs.KindWhy:
		return "hero_why"
	case refs.KindBlocked:
		return "hero_blocked"
	case refs.KindFeed:
		return "hero_feed"
	}
	return ""
}
