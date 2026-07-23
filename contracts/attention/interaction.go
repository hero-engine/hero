package attention

type InteractionEffect string

const (
	EffectRead          InteractionEffect = "read"
	EffectAdvisoryWrite InteractionEffect = "advisory_write"
	EffectStateWrite    InteractionEffect = "state_write"
	EffectExternalWrite InteractionEffect = "external_write"
	EffectCommitment    InteractionEffect = "commitment"
)

type ConsentRequirement string

const (
	ConsentNone               ConsentRequirement = "none"
	ConsentExplicitUser       ConsentRequirement = "explicit_user"
	ConsentExplicitAcceptance ConsentRequirement = "explicit_acceptance"
)

type InteractionDisposition string

const (
	DispositionDispatch        InteractionDisposition = "dispatch"
	DispositionSuggest         InteractionDisposition = "suggest"
	DispositionClarify         InteractionDisposition = "clarify"
	DispositionIgnoreUntrusted InteractionDisposition = "ignore_untrusted"
)

type InteractionSource string

const (
	SourceUser        InteractionSource = "user"
	SourceModel       InteractionSource = "model"
	SourceMailContent InteractionSource = "mail_content"
)

const (
	OperationAttentionSnapshot = "attention.snapshot"
	OperationMailList          = "mail.list"
	OperationMailShow          = "mail.show"
	OperationMailSend          = "mail.send"
	OperationMailReply         = "mail.reply"
	OperationMailMarkRead      = "mail.mark_read"
	OperationMailAcknowledge   = "mail.acknowledge"
	OperationMailDismiss       = "mail.dismiss"
	OperationMailPromote       = "mail.promote"
	OperationMailAddToToday    = "mail.add_to_today"
	OperationFocusCreate       = "focus.create"
	OperationFocusSuggest      = "focus.suggest"
	OperationFocusSuggestions  = "focus.suggestions"
	OperationFocusLaunch       = "focus.launch"
	OperationFocusMoveInbox    = "focus.move_inbox"
	OperationFocusMoveLater    = "focus.move_later"
	OperationFocusComplete     = "focus.complete"
	OperationSuggestionDoNext  = "suggestion.do_next"
	OperationSuggestionToday   = "suggestion.today"
	OperationSuggestionLater   = "suggestion.later"
	OperationSuggestionDismiss = "suggestion.dismiss"
)

type OperationPolicy struct {
	ID                   string             `json:"id"`
	ToolName             string             `json:"tool_name,omitempty"`
	ActionID             string             `json:"action_id,omitempty"`
	Effect               InteractionEffect  `json:"effect"`
	Consent              ConsentRequirement `json:"consent"`
	RequiresUniqueTarget bool               `json:"requires_unique_target"`
	ReplaySafe           bool               `json:"replay_safe"`
	OpenWorld            bool               `json:"open_world"`
}

type ResolutionFacts struct {
	CandidateCount int `json:"candidate_count,omitempty"`
}

type InteractionCase struct {
	ID                string                 `json:"id"`
	Source            InteractionSource      `json:"source"`
	Utterance         string                 `json:"utterance"`
	Resolution        ResolutionFacts        `json:"resolution,omitempty"`
	Disposition       InteractionDisposition `json:"disposition"`
	ExpectedOperation string                 `json:"expected_operation,omitempty"`
	ExpectedEffect    InteractionEffect      `json:"expected_effect,omitempty"`
	ExpectedConsent   ConsentRequirement     `json:"expected_consent,omitempty"`
}

type InteractionPolicyFixture struct {
	SchemaVersion int               `json:"schema_version"`
	Operations    []OperationPolicy `json:"operations"`
	Cases         []InteractionCase `json:"cases"`
}

var operationPolicies = []OperationPolicy{
	{ID: OperationAttentionSnapshot, ToolName: "hero_attention_snapshot", Effect: EffectRead, Consent: ConsentNone, ReplaySafe: true},
	{ID: OperationMailList, ToolName: "hero_mail_list", Effect: EffectRead, Consent: ConsentNone, ReplaySafe: true},
	{ID: OperationMailShow, ToolName: "hero_mail_show", Effect: EffectRead, Consent: ConsentNone, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationMailSend, ToolName: "hero_mail_send", Effect: EffectExternalWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true, OpenWorld: true},
	{ID: OperationMailReply, ToolName: "hero_mail_reply", Effect: EffectExternalWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true, OpenWorld: true},
	{ID: OperationMailMarkRead, ToolName: "hero_attention_action", ActionID: "mark_read", Effect: EffectStateWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationMailAcknowledge, ToolName: "hero_attention_action", ActionID: "acknowledge", Effect: EffectStateWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationMailDismiss, ToolName: "hero_attention_action", ActionID: "dismiss", Effect: EffectStateWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationMailPromote, ToolName: "hero_attention_action", ActionID: "promote", Effect: EffectCommitment, Consent: ConsentExplicitAcceptance, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationMailAddToToday, ToolName: "hero_attention_action", ActionID: "add_to_today", Effect: EffectCommitment, Consent: ConsentExplicitAcceptance, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationFocusCreate, ToolName: "hero_focus_create", Effect: EffectCommitment, Consent: ConsentExplicitUser, ReplaySafe: true},
	{ID: OperationFocusSuggest, ToolName: "hero_focus_suggest", Effect: EffectAdvisoryWrite, Consent: ConsentNone, ReplaySafe: true},
	{ID: OperationFocusSuggestions, ToolName: "hero_focus_suggestions", Effect: EffectRead, Consent: ConsentNone, ReplaySafe: true},
	{ID: OperationFocusLaunch, ToolName: "hero_attention_action", ActionID: "launch", Effect: EffectExternalWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationFocusMoveInbox, ToolName: "hero_attention_action", ActionID: "move_inbox", Effect: EffectStateWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationFocusMoveLater, ToolName: "hero_attention_action", ActionID: "move_later", Effect: EffectStateWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationFocusComplete, ToolName: "hero_attention_action", ActionID: "complete", Effect: EffectStateWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationSuggestionDoNext, ToolName: "hero_attention_action", ActionID: "do_next", Effect: EffectCommitment, Consent: ConsentExplicitAcceptance, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationSuggestionToday, ToolName: "hero_attention_action", ActionID: "today", Effect: EffectCommitment, Consent: ConsentExplicitAcceptance, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationSuggestionLater, ToolName: "hero_attention_action", ActionID: "later", Effect: EffectCommitment, Consent: ConsentExplicitAcceptance, RequiresUniqueTarget: true, ReplaySafe: true},
	{ID: OperationSuggestionDismiss, ToolName: "hero_attention_action", ActionID: "dismiss", Effect: EffectStateWrite, Consent: ConsentExplicitUser, RequiresUniqueTarget: true, ReplaySafe: true},
}

func OperationPolicies() []OperationPolicy {
	out := make([]OperationPolicy, len(operationPolicies))
	copy(out, operationPolicies)
	return out
}

func OperationPolicyByID(id string) (OperationPolicy, bool) {
	for _, policy := range operationPolicies {
		if policy.ID == id {
			return policy, true
		}
	}
	return OperationPolicy{}, false
}

func AnnotateActionDescriptor(descriptor ActionDescriptor, operationID string) (ActionDescriptor, bool) {
	policy, ok := OperationPolicyByID(operationID)
	if !ok {
		return descriptor, false
	}
	descriptor.OperationID = policy.ID
	descriptor.Effect = string(policy.Effect)
	descriptor.Consent = string(policy.Consent)
	return descriptor, true
}
