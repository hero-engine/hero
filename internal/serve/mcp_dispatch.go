package serve

import (
	"context"
	"encoding/json"
	"fmt"
)

// toolHandler is the signature every Hero MCP tool implementation
// satisfies. The dispatcher looks up the handler by tool name and
// delegates the call.
type toolHandler func(args map[string]interface{}) (string, error)

// toolHandlers returns the name → handler mapping. Adding a new tool
// is one line here plus one entry in toolDefinitions(). The map is
// rebuilt per call (cheap; runs only on tools/call) so handlers
// stay closures over the live MCPServer.
//
// Categories follow the same split as the mcp_tools_*.go files:
//   - read   (no state change; returns data)
//   - mutate (writes state)
//   - analyze (computes a derived view)
//
// Some tools straddle (toolActive, toolCode) — placed where their
// primary action sits.
func (s *MCPServer) toolHandlers() map[string]toolHandler {
	handlers := map[string]toolHandler{
		// read
		"hero_context":              s.toolContext,
		"hero_search":               s.toolSearch,
		"hero_status":               s.toolStatus,
		"hero_check":                s.toolCheck,
		"hero_nudge":                s.toolNudge,
		"hero_list":                 s.toolList,
		"hero_queue":                s.toolQueue,
		"hero_goal":                 s.toolGoal,
		"hero_kickoff":              s.toolKickoff,
		"hero_knowledge":            s.toolKnowledge,
		"hero_read_spec":            s.toolReadSpec,
		"hero_ask":                  s.toolAsk,
		"hero_anchor":               s.toolAnchor,
		"hero_pulse":                s.toolPulse,
		"hero_plan":                 s.toolPlan,
		"hero_contract":             s.toolContract,
		"hero_impact":               s.toolImpact,
		"hero_recap":                s.toolRecap,
		"hero_feed":                 s.toolFeed,
		"hero_active":               s.toolActive,
		"hero_coverage":             s.toolCoverage,
		"hero_ci":                   s.toolCI,
		"hero_why":                  s.toolWhy,
		"hero_blocked":              s.toolBlocked,
		"hero_expand":               s.toolExpand,
		"hero_snapshot":             s.toolSnapshot,
		"hero_focus_suggestions":    s.toolFocusSuggestions,
		"hero_mail_list":            s.toolMailList,
		"hero_mail_show":            s.toolMailShow,
		"hero_mail_thread_list":     s.toolMailThreadList,
		"hero_mail_thread_show":     s.toolMailThreadShow,
		"hero_mail_thread_contract": s.toolMailThreadContract,
		"hero_attention_contract":   s.toolAttentionContract,

		// mutate
		"hero_tracker_get_issue":       s.toolTrackerGetIssue,
		"hero_tracker_search":          s.toolTrackerSearch,
		"hero_tracker_request":         s.toolTrackerRequest,
		"hero_tracker_cli":             s.toolTrackerCLI,
		"hero_tracker_load_evidence":   s.toolTrackerLoadEvidence,
		"hero_event":                   s.toolEvent,
		"hero_claim":                   s.toolClaim,
		"hero_skill_run":               s.toolSkillRun,
		"hero_test_generate":           s.toolTestGenerate,
		"hero_demo_record":             s.toolDemoRecord,
		"hero_code":                    s.toolCode,
		"hero_enrich":                  s.toolEnrich,
		"hero_synthesize":              s.toolSynthesize,
		"hero_focus_suggest":           s.toolFocusSuggest,
		"hero_focus_create":            s.toolFocusCreate,
		"hero_focus_suggestion_action": s.toolFocusSuggestionAction,
		"hero_mail_send":               s.toolMailSend,
		"hero_mail_reply":              s.toolMailReply,
		"hero_mail_action":             s.toolMailAction,
		"hero_mail_thread_action":      s.toolMailThreadAction,

		// analyze
		"hero_drift":         s.toolDrift,
		"hero_diagnose":      s.toolDiagnose,
		"hero_score":         s.toolScore,
		"hero_error_pattern": s.toolErrorPattern,
		"hero_velocity":      s.toolVelocity,
		"hero_verify":        s.toolVerify,
		"hero_conflicts":     s.toolConflicts,
		"hero_sequence":      s.toolSequence,
		"hero_warnings":      s.toolWarnings,
		"hero_insights":      s.toolInsights,
	}
	for _, operation := range codeHostMCPOperations() {
		operation := operation
		handlers[codeHostMCPToolName(operation)] = func(args map[string]interface{}) (string, error) {
			return s.toolCodeHost(operation, args)
		}
	}
	handlers["hero_attention_snapshot"] = s.toolAttentionSnapshot
	handlers["hero_attention_action"] = s.toolAttentionAction
	return handlers
}

// toolSafetyClass is the read / mutate / analyze grouping that toolHandlers()
// is already sectioned by, lifted from source-file comments into data so
// tools/list can derive MCP annotations from it instead of hand-judging every
// tool. It is kept adjacent to toolHandlers() and in the same section order.
//
// This map only needs the "regular" tools — the Attention/Mail/Focus and
// code-host families already carry policy-derived annotations set at their
// definition sites, and finalizeToolMetadata leaves those untouched. The
// metadata drift guard asserts every advertised tool ends with annotations, so
// a handler added here without a class below fails loudly.
//
// The grouping mirrors the existing toolHandlers() curation, EXCEPT that the
// handler-section split is by a tool's PRIMARY action and several "read"/
// "analyze" tools also write on some inputs (an action= parameter, archive:true,
// etc.). Safety class must be "can this ever write," so every such conditional
// writer is listed under mutate below with the write path noted. A cold audit
// found three writers still misclassed as read (hero_active, hero_contract,
// hero_snapshot) after a first pass fixed two (hero_plan, hero_error_pattern);
// TestConditionalWritersAreNotReadOnly now pins all five so a regrouping that
// flips one back to read fails loudly. This map is the safety source of truth.
type toolSafetyClass int

const (
	safetyRead toolSafetyClass = iota
	safetyMutate
	safetyAnalyze
)

func toolSafetyClasses() map[string]toolSafetyClass {
	return map[string]toolSafetyClass{
		// read
		"hero_context":   safetyRead,
		"hero_search":    safetyRead,
		"hero_status":    safetyRead,
		"hero_check":     safetyRead,
		"hero_nudge":     safetyRead,
		"hero_list":      safetyRead,
		"hero_queue":     safetyRead,
		"hero_goal":      safetyRead,
		"hero_kickoff":   safetyRead,
		"hero_knowledge": safetyRead,
		"hero_read_spec": safetyRead,
		"hero_ask":       safetyRead,
		"hero_anchor":    safetyRead,
		"hero_pulse":     safetyRead,
		"hero_impact":    safetyRead,
		"hero_recap":     safetyRead,
		"hero_coverage":  safetyRead,
		"hero_ci":        safetyRead,
		"hero_why":       safetyRead,
		"hero_blocked":   safetyRead,
		"hero_expand":    safetyRead,
		"hero_feed":      safetyRead,

		// mutate
		"hero_tracker_get_issue":     safetyMutate,
		"hero_tracker_search":        safetyMutate,
		"hero_tracker_request":       safetyMutate,
		"hero_tracker_cli":           safetyMutate,
		"hero_tracker_load_evidence": safetyMutate,
		"hero_event":                 safetyMutate,
		"hero_claim":                 safetyMutate,
		"hero_skill_run":             safetyMutate,
		"hero_test_generate":         safetyMutate,
		"hero_demo_record":           safetyMutate,
		"hero_code":                  safetyMutate,
		"hero_enrich":                safetyMutate,
		"hero_synthesize":            safetyMutate,
		// Conditional writers: these sit in the read/analyze handler sections
		// of toolHandlers() because their PRIMARY action returns data, but each
		// writes to disk on some inputs. Safety class is "can this ever write,"
		// not "what does it usually do" — ReadOnlyHint=true would tell a client
		// it is safe to call unprompted, and a conforming harness could then
		// auto-invoke a state-writer. The handler-section grouping is by primary
		// action and is NOT a safety oracle; every conditional writer must be
		// listed here explicitly. (Covered by TestConditionalWritersAreNotReadOnly.)
		"hero_plan":          safetyMutate, // action save → os.WriteFile(planPath)
		"hero_error_pattern": safetyMutate, // → errpattern.SavePattern
		"hero_contract":      safetyMutate, // action link → contract.Link writes tracker_id into the spec
		"hero_active":        safetyMutate, // action register/unregister/prune → active.* writes .hero state
		"hero_snapshot":      safetyMutate, // archive:true → snapshot archive os.WriteFile

		// analyze
		"hero_drift":     safetyAnalyze,
		"hero_diagnose":  safetyAnalyze,
		"hero_score":     safetyAnalyze,
		"hero_velocity":  safetyAnalyze,
		"hero_verify":    safetyAnalyze,
		"hero_conflicts": safetyAnalyze,
		"hero_sequence":  safetyAnalyze,
		"hero_warnings":  safetyAnalyze,
		"hero_insights":  safetyAnalyze,
	}
}

// annotationsForSafety derives MCP annotations from a tool's safety class.
// read/analyze → read-only and idempotent; mutate → not read-only. No regular
// mutating Hero tool unconditionally deletes or overwrites workspace state, so
// DestructiveHint is false across the set; a harness reads OpenWorldHint on the
// policy-annotated tracker/code-host families for genuinely open-world reach.
func annotationsForSafety(class toolSafetyClass) *ToolAnnotations {
	switch class {
	case safetyMutate:
		return &ToolAnnotations{ReadOnlyHint: boolPointer(false), DestructiveHint: boolPointer(false)}
	default: // safetyRead, safetyAnalyze
		return &ToolAnnotations{ReadOnlyHint: boolPointer(true), DestructiveHint: boolPointer(false), IdempotentHint: boolPointer(true)}
	}
}

// handleToolsCall validates the call, looks up the handler in the
// dispatch table, and emits the result via finishToolCall. Adding a
// tool no longer requires editing this function — see toolHandlers().
func (s *MCPServer) handleToolsCall(req *JSONRPCRequest) {
	s.handleToolsCallContext(req, s.ctx)
}

func (s *MCPServer) handleToolsCallContext(req *JSONRPCRequest, ctx context.Context) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, ErrCodeInvalidParams, "Invalid tool call parameters")
		return
	}

	// Reject calls to tools that have been filtered out for this
	// profile (e.g. read-only profiles can't invoke mutating tools).
	if s.filter != nil {
		allowed := s.filter.FilterTools(s.toolDefinitions(), s.profile)
		found := false
		for _, t := range allowed {
			if t.Name == params.Name {
				found = true
				break
			}
		}
		if !found {
			s.sendError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("Tool not available: %s", params.Name))
			return
		}
	}

	if operation, ok := codeHostOperationForMCPTool(params.Name); ok {
		result, toolErr := s.toolCodeHostContext(ctx, operation, params.Arguments)
		s.finishToolCall(req.ID, result, toolErr)
		return
	}

	handler, ok := s.toolHandlers()[params.Name]
	if !ok {
		s.sendError(req.ID, ErrCodeInvalidParams, fmt.Sprintf("Unknown tool: %s", params.Name))
		return
	}

	result, toolErr := handler(params.Arguments)
	s.finishToolCall(req.ID, result, toolErr)
}
