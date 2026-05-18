package serve

import (
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
// Some tools straddle (toolActive, toolCode) — placed where their
// primary action sits.
func (s *MCPServer) toolHandlers() map[string]toolHandler {
	return map[string]toolHandler{
		// read
		"hero_context":   s.toolContext,
		"hero_search":    s.toolSearch,
		"hero_status":    s.toolStatus,
		"hero_check":     s.toolCheck,
		"hero_nudge":     s.toolNudge,
		"hero_list":      s.toolList,
		"hero_queue":     s.toolQueue,
		"hero_kickoff":   s.toolKickoff,
		"hero_knowledge": s.toolKnowledge,
		"hero_read_spec": s.toolReadSpec,
		"hero_ask":       s.toolAsk,
		"hero_anchor":    s.toolAnchor,
		"hero_pulse":     s.toolPulse,
		"hero_plan":      s.toolPlan,
		"hero_contract":  s.toolContract,
		"hero_impact":    s.toolImpact,
		"hero_recap":     s.toolRecap,
		"hero_feed":      s.toolFeed,
		"hero_active":    s.toolActive,
		"hero_coverage":  s.toolCoverage,
		"hero_ci":        s.toolCI,
		"hero_why":       s.toolWhy,
		"hero_blocked":   s.toolBlocked,
		"hero_expand":    s.toolExpand,
		"hero_snapshot":  s.toolSnapshot,

		// mutate
		"hero_event":         s.toolEvent,
		"hero_claim":         s.toolClaim,
		"hero_skill_run":     s.toolSkillRun,
		"hero_test_generate": s.toolTestGenerate,
		"hero_demo_record":   s.toolDemoRecord,
		"hero_code":          s.toolCode,
		"hero_enrich":        s.toolEnrich,

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
}

// handleToolsCall validates the call, looks up the handler in the
// dispatch table, and emits the result via finishToolCall. Adding a
// tool no longer requires editing this function — see toolHandlers().
func (s *MCPServer) handleToolsCall(req *JSONRPCRequest) {
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

	handler, ok := s.toolHandlers()[params.Name]
	if !ok {
		s.sendError(req.ID, ErrCodeInvalidParams, fmt.Sprintf("Unknown tool: %s", params.Name))
		return
	}

	result, toolErr := handler(params.Arguments)
	s.finishToolCall(req.ID, result, toolErr)
}
