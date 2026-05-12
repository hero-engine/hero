package sessions

import (
	"fmt"
	"strings"
)

// Distill analyzes a session log and returns text suggestions for new knowledge entries.
func Distill(heroDir, sessionID string) (string, error) {
	events, err := ReadEvents(heroDir, sessionID)
	if err != nil {
		return "", fmt.Errorf("reading events: %w", err)
	}

	if len(events) == 0 {
		return fmt.Sprintf("No new patterns observed in session %s.", sessionID), nil
	}

	// Count how many times each knowledge slug was retrieved via context_retrieved
	slugCounts := make(map[string]int)
	for _, evt := range events {
		evtType, _ := evt["event"].(string)
		if evtType == "context_retrieved" {
			if slug, ok := evt["knowledge"].(string); ok && slug != "" {
				slugCounts[slug]++
			}
		}
	}

	// Collect ask_answered events: questions not obviously in knowledge
	type askEntry struct {
		question string
		answer   string
	}
	var asks []askEntry
	for _, evt := range events {
		evtType, _ := evt["event"].(string)
		if evtType == "ask_answered" {
			q, _ := evt["question"].(string)
			a, _ := evt["answer"].(string)
			if q != "" {
				asks = append(asks, askEntry{question: q, answer: a})
			}
		}
	}

	var suggestions []string

	// Suggest frequently-retrieved knowledge as a rule
	for slug, count := range slugCounts {
		if count >= 3 {
			suggestions = append(suggestions,
				fmt.Sprintf("Knowledge entry %q was retrieved %d times — consider promoting to a rule", slug, count))
		}
	}

	// Suggest adding ask/answer pairs to knowledge
	for _, ask := range asks {
		suggestions = append(suggestions,
			fmt.Sprintf("Q: %q — consider adding answer to knowledge base", ask.question))
	}

	if len(suggestions) == 0 {
		return fmt.Sprintf("No new patterns observed in session %s.", sessionID), nil
	}

	var sb strings.Builder
	sb.WriteString("Observed patterns not in knowledge base:\n\n")
	for _, s := range suggestions {
		fmt.Fprintf(&sb, "  ? %s\n", s)
	}

	return sb.String(), nil
}
