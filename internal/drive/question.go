package drive

import (
	"fmt"
	"strings"
)

// Pause-question markers delimit the block Drive owns in the handoff file,
// so it can be rewritten in place without clobbering the rest of NEXT.md.
const (
	QuestionStart = "<!-- drive-pause -->"
	QuestionEnd   = "<!-- /drive-pause -->"
)

// ComposeQuestion renders the pause as a precise, self-contained question —
// not a bare status line. It carries what stopped the run, what's done, and
// how to resume, so a fresh session (or another machine) can act on it cold.
func ComposeQuestion(initSlug string, res CheckResult) string {
	var b strings.Builder
	b.WriteString(QuestionStart + "\n")
	b.WriteString("## Drive paused — needs you\n\n")
	fmt.Fprintf(&b, "**Initiative:** %s\n", initSlug)
	if res.Pause != nil {
		fmt.Fprintf(&b, "**Stopped at:** %s  (category: %s)\n\n", res.NextSpec, res.Pause.Category)
		fmt.Fprintf(&b, "**Decision needed:** %s\n", res.Pause.Reason)
	}
	if len(res.Completed) > 0 {
		fmt.Fprintf(&b, "**Done so far:** %s\n", strings.Join(res.Completed, ", "))
	}
	if len(res.Remaining) > 0 {
		fmt.Fprintf(&b, "**Remaining:** %s\n", strings.Join(res.Remaining, ", "))
	}
	fmt.Fprintf(&b, "\n**To resume:** `hero goal %s --answer \"<your call>\"`, then re-run `/drive %s` (or it resumes on the next turn if still armed).\n", initSlug, initSlug)
	b.WriteString(QuestionEnd + "\n")
	return b.String()
}

// MergeQuestion replaces an existing drive-pause block in prior handoff
// content with the fresh block (idempotent), or appends it when absent.
func MergeQuestion(prior, block string) string {
	start := strings.Index(prior, QuestionStart)
	end := strings.Index(prior, QuestionEnd)
	if start >= 0 && end > start {
		end += len(QuestionEnd)
		merged := strings.TrimRight(prior[:start], "\n") + "\n\n" + block
		if rest := strings.TrimLeft(prior[end:], "\n"); rest != "" {
			merged += "\n" + rest
		}
		return merged
	}
	if strings.TrimSpace(prior) == "" {
		return block
	}
	return strings.TrimRight(prior, "\n") + "\n\n" + block
}

// StripQuestion removes the drive-pause block from handoff content (used when
// the run resumes or completes so the stale question doesn't linger).
func StripQuestion(prior string) string {
	start := strings.Index(prior, QuestionStart)
	end := strings.Index(prior, QuestionEnd)
	if start < 0 || end <= start {
		return prior
	}
	end += len(QuestionEnd)
	return strings.TrimRight(prior[:start], "\n") + strings.TrimLeft(prior[end:], "\n")
}
