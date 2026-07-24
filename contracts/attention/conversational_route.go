package attention

import (
	"fmt"
	"strings"
)

type ConversationalRouteCategory string

const (
	RouteCategoryFamily        ConversationalRouteCategory = "route_family"
	RouteCategoryAmbiguity     ConversationalRouteCategory = "ambiguity"
	RouteCategoryUntrustedMail ConversationalRouteCategory = "untrusted_mail"
	RouteCategoryResilience    ConversationalRouteCategory = "resilience"
)

type ConversationalDispatchSurface string

const (
	DispatchSurfaceMCPTool          ConversationalDispatchSurface = "mcp_tool"
	DispatchSurfaceAdvertisedAction ConversationalDispatchSurface = "advertised_action"
	DispatchSurfaceCLIWorkflow      ConversationalDispatchSurface = "cli_workflow"
)

type ConversationalRetryExpectation string

const (
	RetryNotApplicable      ConversationalRetryExpectation = "not_applicable"
	RetrySameKeyNoDuplicate ConversationalRetryExpectation = "same_key_no_duplicate"
	RetryRefreshThenRetry   ConversationalRetryExpectation = "refresh_then_retry"
	RetryDoNotRetry         ConversationalRetryExpectation = "do_not_retry"
)

const (
	OperationPeerAdvisory = "peer.advisory"
	OperationPeerSpecOut  = "peer.spec_out"
	OperationPeerHandoff  = "peer.handoff"
)

type ConversationalResolution struct {
	CandidateCount int      `json:"candidate_count,omitempty"`
	ResolvedFacts  []string `json:"resolved_facts,omitempty"`
	MissingFacts   []string `json:"missing_facts,omitempty"`
}

type ConversationalRouteCase struct {
	ID                         string                         `json:"id"`
	Category                   ConversationalRouteCategory    `json:"category"`
	Phrase                     string                         `json:"phrase"`
	Source                     InteractionSource              `json:"source"`
	TrustClass                 string                         `json:"trust_class"`
	Resolution                 ConversationalResolution       `json:"resolution,omitempty"`
	ExpectedDisposition        InteractionDisposition         `json:"expected_disposition"`
	ExpectedOperation          string                         `json:"expected_operation,omitempty"`
	ExpectedSurface            ConversationalDispatchSurface  `json:"expected_surface,omitempty"`
	ExpectedTool               string                         `json:"expected_tool,omitempty"`
	ExpectedAction             string                         `json:"expected_action,omitempty"`
	ExpectedCommand            string                         `json:"expected_command,omitempty"`
	ExpectedEffect             InteractionEffect              `json:"expected_effect,omitempty"`
	ExpectedConsent            ConsentRequirement             `json:"expected_consent,omitempty"`
	ExpectedMutationCount      int                            `json:"expected_mutation_count"`
	RetryExpectation           ConversationalRetryExpectation `json:"retry_expectation"`
	RetryExpectedMutationCount int                            `json:"retry_expected_mutation_count"`
	ExpectedClarificationField string                         `json:"expected_clarification_field,omitempty"`
	ExpectedErrorCode          string                         `json:"expected_error_code,omitempty"`
}

type ConversationalRouteFixture struct {
	SchemaVersion int                       `json:"schema_version"`
	Cases         []ConversationalRouteCase `json:"cases"`
}

type commandRoutePolicy struct {
	Command string
	Effect  InteractionEffect
	Consent ConsentRequirement
}

var commandRoutePolicies = map[string]commandRoutePolicy{
	OperationPeerAdvisory: {Command: "hero peer call --mode=advisory", Effect: EffectExternalWrite, Consent: ConsentExplicitUser},
	OperationPeerSpecOut:  {Command: "hero peer call --mode=spec-out", Effect: EffectExternalWrite, Consent: ConsentExplicitUser},
	OperationPeerHandoff:  {Command: "hero handoff", Effect: EffectCommitment, Consent: ConsentExplicitUser},
}

func ValidateConversationalRouteFixture(v ConversationalRouteFixture) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if len(v.Cases) < 26 {
		return invalid("cases", "must contain at least 26 conversational conformance cases")
	}

	seen := make(map[string]bool, len(v.Cases))
	categoryCounts := make(map[ConversationalRouteCategory]int)
	for i, routeCase := range v.Cases {
		field := fmt.Sprintf("cases[%d]", i)
		if strings.TrimSpace(routeCase.ID) == "" || seen[routeCase.ID] {
			return invalid(field+".id", "is required and must be unique")
		}
		seen[routeCase.ID] = true
		if strings.TrimSpace(routeCase.Phrase) == "" {
			return invalid(field+".phrase", "is required")
		}
		if !validConversationalCategory(routeCase.Category) {
			return invalid(field+".category", "is not a stable conversational route category")
		}
		categoryCounts[routeCase.Category]++
		if !validInteractionSource(routeCase.Source) {
			return invalid(field+".source", "is not a stable v1 interaction source")
		}
		if routeCase.TrustClass != "trusted_user" && routeCase.TrustClass != "model_originated" && routeCase.TrustClass != "untrusted_mail" {
			return invalid(field+".trust_class", "must be trusted_user, model_originated, or untrusted_mail")
		}
		if !validInteractionDisposition(routeCase.ExpectedDisposition) {
			return invalid(field+".expected_disposition", "is not a stable v1 disposition")
		}
		if routeCase.ExpectedMutationCount < 0 || routeCase.RetryExpectedMutationCount < 0 {
			return invalid(field+".expected_mutation_count", "mutation counts must not be negative")
		}
		if !validRetryExpectation(routeCase.RetryExpectation) {
			return invalid(field+".retry_expectation", "is not a stable retry expectation")
		}
		if err := validateResolutionFacts(field+".resolution", routeCase.Resolution); err != nil {
			return err
		}
		if err := validateConversationalRouteCase(field, routeCase); err != nil {
			return err
		}
	}

	for category, floor := range map[ConversationalRouteCategory]int{
		RouteCategoryFamily:        13,
		RouteCategoryAmbiguity:     6,
		RouteCategoryUntrustedMail: 3,
		RouteCategoryResilience:    4,
	} {
		if categoryCounts[category] < floor {
			return invalid("cases", fmt.Sprintf("category %s must contain at least %d cases", category, floor))
		}
	}
	return nil
}

func validateConversationalRouteCase(field string, routeCase ConversationalRouteCase) *ContractError {
	if routeCase.Source == SourceMailContent {
		if routeCase.TrustClass != "untrusted_mail" || routeCase.ExpectedDisposition != DispositionIgnoreUntrusted {
			return invalid(field+".expected_disposition", "Mail content must remain untrusted and dispatch nothing")
		}
	}
	if routeCase.Source == SourceModel && routeCase.TrustClass != "model_originated" {
		return invalid(field+".trust_class", "model-originated routes must declare model_originated trust")
	}

	switch routeCase.ExpectedDisposition {
	case DispositionClarify, DispositionIgnoreUntrusted:
		if routeCase.ExpectedOperation != "" || routeCase.ExpectedSurface != "" || routeCase.ExpectedTool != "" ||
			routeCase.ExpectedAction != "" || routeCase.ExpectedCommand != "" || routeCase.ExpectedEffect != "" ||
			routeCase.ExpectedConsent != "" {
			return invalid(field+".expected_operation", "non-dispatch routes must not name an executable operation")
		}
		if routeCase.ExpectedMutationCount != 0 || routeCase.RetryExpectedMutationCount != 0 {
			return invalid(field+".expected_mutation_count", "non-dispatch routes must mutate zero times")
		}
		if routeCase.ExpectedDisposition == DispositionClarify && strings.TrimSpace(routeCase.ExpectedClarificationField) == "" {
			return invalid(field+".expected_clarification_field", "clarification routes must name the one missing field")
		}
		if routeCase.RetryExpectation != RetryDoNotRetry {
			return invalid(field+".retry_expectation", "non-dispatch routes must not retry")
		}
		return nil
	case DispositionDispatch, DispositionSuggest:
	default:
		return invalid(field+".expected_disposition", "unsupported conversational disposition")
	}

	if strings.TrimSpace(routeCase.ExpectedClarificationField) != "" {
		return invalid(field+".expected_clarification_field", "dispatch routes must not ask for clarification")
	}

	if policy, ok := OperationPolicyByID(routeCase.ExpectedOperation); ok {
		if routeCase.ExpectedEffect != policy.Effect || routeCase.ExpectedConsent != policy.Consent {
			return invalid(field, "effect and consent must match the canonical operation policy")
		}
		if routeCase.ExpectedTool != policy.ToolName || routeCase.ExpectedAction != policy.ActionID || routeCase.ExpectedCommand != "" {
			return invalid(field, "tool and action must match the canonical operation policy")
		}
		expectedSurface := DispatchSurfaceMCPTool
		if policy.ActionID != "" {
			expectedSurface = DispatchSurfaceAdvertisedAction
		}
		if routeCase.ExpectedSurface != expectedSurface {
			return invalid(field+".expected_surface", "does not match the canonical operation surface")
		}
		if policy.RequiresUniqueTarget && routeCase.Resolution.CandidateCount != 1 {
			return invalid(field+".resolution.candidate_count", "must be exactly one for the expected operation")
		}
		if routeCase.Source == SourceModel && policy.ID != OperationFocusSuggest {
			return invalid(field+".expected_operation", "model-originated deferred work must use Focus suggestion")
		}
	} else if policy, ok := commandRoutePolicies[routeCase.ExpectedOperation]; ok {
		if routeCase.ExpectedSurface != DispatchSurfaceCLIWorkflow || routeCase.ExpectedCommand != policy.Command ||
			routeCase.ExpectedTool != "" || routeCase.ExpectedAction != "" {
			return invalid(field, "CLI workflow dispatch must match the canonical typed peering route")
		}
		if routeCase.ExpectedEffect != policy.Effect || routeCase.ExpectedConsent != policy.Consent {
			return invalid(field, "effect and consent must match the typed peering route")
		}
		if routeCase.Resolution.CandidateCount != 1 {
			return invalid(field+".resolution.candidate_count", "typed peering requires one resolved peer")
		}
	} else {
		return invalid(field+".expected_operation", "must reference a canonical Attention or peering operation")
	}

	if routeCase.ExpectedErrorCode != "" && !validErrorCode(routeCase.ExpectedErrorCode) {
		return invalid(field+".expected_error_code", "is not a stable v1 error code")
	}
	if routeCase.ExpectedErrorCode == ErrorStale && routeCase.RetryExpectation != RetryRefreshThenRetry {
		return invalid(field+".retry_expectation", "stale dispatch must refresh before retry")
	}
	if routeCase.ExpectedErrorCode != "" && routeCase.ExpectedMutationCount != 0 {
		return invalid(field+".expected_mutation_count", "failed dispatch must mutate zero times")
	}
	if routeCase.ExpectedEffect == EffectRead && routeCase.ExpectedMutationCount != 0 {
		return invalid(field+".expected_mutation_count", "read dispatch must mutate zero times")
	}
	if routeCase.ExpectedErrorCode == "" && routeCase.ExpectedEffect != EffectRead {
		replayedWrite := routeCase.Category == RouteCategoryResilience &&
			routeCase.RetryExpectation == RetrySameKeyNoDuplicate &&
			containsFact(routeCase.Resolution.ResolvedFacts, "idempotency_key")
		if replayedWrite {
			if routeCase.ExpectedMutationCount != 0 {
				return invalid(field+".expected_mutation_count", "same-key replay must create no duplicate mutation")
			}
		} else if routeCase.ExpectedMutationCount != 1 {
			return invalid(field+".expected_mutation_count", "successful write dispatch must mutate exactly once")
		}
	}
	switch routeCase.RetryExpectation {
	case RetryRefreshThenRetry:
		if routeCase.RetryExpectedMutationCount != 1 {
			return invalid(field+".retry_expected_mutation_count", "a refreshed authorized retry must mutate exactly once")
		}
	default:
		if routeCase.RetryExpectedMutationCount != 0 {
			return invalid(field+".retry_expected_mutation_count", "this retry policy must create no mutation")
		}
	}
	return nil
}

func validateResolutionFacts(field string, resolution ConversationalResolution) *ContractError {
	if resolution.CandidateCount < 0 {
		return invalid(field+".candidate_count", "must not be negative")
	}
	seen := make(map[string]bool)
	for _, fact := range resolution.ResolvedFacts {
		if strings.TrimSpace(fact) == "" || seen[fact] {
			return invalid(field+".resolved_facts", "must contain unique non-empty names")
		}
		seen[fact] = true
	}
	for _, fact := range resolution.MissingFacts {
		if strings.TrimSpace(fact) == "" || seen[fact] {
			return invalid(field+".missing_facts", "must be unique and not already resolved")
		}
		seen[fact] = true
	}
	return nil
}

func validConversationalCategory(v ConversationalRouteCategory) bool {
	return v == RouteCategoryFamily || v == RouteCategoryAmbiguity || v == RouteCategoryUntrustedMail || v == RouteCategoryResilience
}

func validRetryExpectation(v ConversationalRetryExpectation) bool {
	return v == RetryNotApplicable || v == RetrySameKeyNoDuplicate || v == RetryRefreshThenRetry || v == RetryDoNotRetry
}

func containsFact(facts []string, want string) bool {
	for _, fact := range facts {
		if fact == want {
			return true
		}
	}
	return false
}
