package chat

// Capability is the JSON snapshot returned by GET /api/chat/capability.
// Empty Interactive / Headless strings mean "no adapter selected for
// this kind"; the UI renders the empty-state CTA in that case.
type Capability struct {
	Adapters      []AdapterInfo `json:"adapters"`
	Interactive   string        `json:"interactive"`
	Headless      string        `json:"headless"`
	UserPreferred string        `json:"user_preferred"`
}

// Resolve picks an interactive and a headless adapter from the
// registry, applying user preference and the "prefer hero-code"
// tiebreak. userPreferred is the adapter TYPE (e.g. "hero-code"),
// NOT a connection id — connection ids change across reconnects.
//
// Selection rules:
//
//   - Interactive: if userPreferred names a connected adapter type
//     that supports interactive, use the first one. Else fall back to
//     PreferHeroCode(interactive).
//   - Headless: PreferHeroCode(headless). IDE bridges that declare
//     headless are filtered out here — the IDE may not be running
//     when a cron fires, so we never route headless to them.
func Resolve(reg *Registry, userPreferred string) Capability {
	all := reg.All()
	cap := Capability{
		Adapters:      all,
		UserPreferred: userPreferred,
	}

	// Interactive — try user preference first.
	if userPreferred != "" {
		for _, info := range all {
			if info.Adapter == userPreferred && supports(info.Kinds, KindInteractive) {
				cap.Interactive = info.ID
				break
			}
		}
	}
	if cap.Interactive == "" {
		// Prefer hero-code; fall back to any interactive-capable adapter.
		for _, info := range all {
			if info.Adapter == "hero-code" && supports(info.Kinds, KindInteractive) {
				cap.Interactive = info.ID
				break
			}
		}
		if cap.Interactive == "" {
			for _, info := range all {
				if supports(info.Kinds, KindInteractive) {
					cap.Interactive = info.ID
					break
				}
			}
		}
	}

	// Headless — only hero-code-class adapters receive headless work.
	// An IDE bridge that declares headless is filtered out: the IDE may
	// not be running when a cron fires.
	for _, info := range all {
		if info.Adapter == "hero-code" && supports(info.Kinds, KindHeadless) {
			cap.Headless = info.ID
			break
		}
	}

	return cap
}
