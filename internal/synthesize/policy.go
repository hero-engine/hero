package synthesize

// Mode is the synthesis-autonomy setting (knowledge.explainer_synthesis).
type Mode string

const (
	ModeAuto   Mode = "auto"
	ModeReview Mode = "review"
	ModeOff    Mode = "off"

	// autoConfidenceBar is the confidence at or above which a candidate may
	// auto-synthesize in `auto` mode. Below it, candidates always route to
	// review — detection false positives must never be written silently.
	autoConfidenceBar = 0.9
)

// NormalizeMode maps a config string to a Mode, defaulting to review for
// empty or unrecognized values (conservative: propose, don't auto-write).
func NormalizeMode(s string) Mode {
	switch Mode(s) {
	case ModeAuto:
		return ModeAuto
	case ModeOff:
		return ModeOff
	default:
		return ModeReview
	}
}

// Action returns the action for a candidate under a mode: "auto", "review",
// or "skip". Sub-threshold confidence forces review even in auto mode.
func Action(confidence float64, mode Mode) string {
	switch mode {
	case ModeOff:
		return "skip"
	case ModeAuto:
		if confidence >= autoConfidenceBar {
			return "auto"
		}
		return "review"
	default:
		return "review"
	}
}
