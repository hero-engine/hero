package sessions

import (
	"fmt"
	"strings"
	"time"
)

// Replay renders a human-readable summary of a session log.
func Replay(heroDir, sessionID string) (string, error) {
	sess, err := Load(heroDir, sessionID)
	if err != nil {
		return "", fmt.Errorf("loading session: %w", err)
	}

	events, err := ReadEvents(heroDir, sessionID)
	if err != nil {
		return "", fmt.Errorf("reading events: %w", err)
	}

	var sb strings.Builder

	// Header line
	dateStr := sess.Start.Format("2006-01-02")
	startStr := sess.Start.Format("15:04")
	endStr := "ongoing"
	durationStr := ""
	if sess.End != nil {
		endStr = sess.End.Format("15:04")
		dur := sess.End.Sub(sess.Start)
		mins := int(dur.Minutes())
		durationStr = fmt.Sprintf(", %d min", mins)
	}

	name := sess.Name
	if name == "" {
		name = sess.ID
	}
	fmt.Fprintf(&sb, "Session: %s — %s (%s %s → %s%s)\n", sess.ID, name, dateStr, startStr, endStr, durationStr)
	if sess.Agent != "" {
		fmt.Fprintf(&sb, "Agent: %s\n", sess.Agent)
	}

	if len(events) == 0 {
		sb.WriteString("\nNo events logged.\n")
		return sb.String(), nil
	}

	// Group events by type
	type contextEntry struct {
		file      string
		knowledge string
	}
	var contextEntries []contextEntry
	type knowledgeEntry struct {
		slug    string
		excerpt string
	}
	var knowledgeEntries []knowledgeEntry
	type specEntry struct {
		slug      string
		claimedAt string
		doneAt    string
		durMins   int
	}
	var specEntries []specEntry
	totalHeroCalls := 0

	specMap := make(map[string]*specEntry)

	for _, evt := range events {
		evtType, _ := evt["event"].(string)
		switch evtType {
		case "context_retrieved":
			file, _ := evt["file"].(string)
			knowledge, _ := evt["knowledge"].(string)
			contextEntries = append(contextEntries, contextEntry{file: file, knowledge: knowledge})
		case "knowledge_applied":
			slug, _ := evt["slug"].(string)
			excerpt, _ := evt["excerpt"].(string)
			knowledgeEntries = append(knowledgeEntries, knowledgeEntry{slug: slug, excerpt: excerpt})
		case "hero_call":
			totalHeroCalls++
		case "spec_claimed":
			slug, _ := evt["slug"].(string)
			t, _ := evt["t"].(string)
			if _, ok := specMap[slug]; !ok {
				specMap[slug] = &specEntry{slug: slug}
			}
			specMap[slug].claimedAt = t
		case "spec_completed":
			slug, _ := evt["slug"].(string)
			t, _ := evt["t"].(string)
			if _, ok := specMap[slug]; !ok {
				specMap[slug] = &specEntry{slug: slug}
			}
			specMap[slug].doneAt = t
			// Calculate duration
			entry := specMap[slug]
			if entry.claimedAt != "" {
				claimedTime, err1 := time.Parse(time.RFC3339, entry.claimedAt)
				doneTime, err2 := time.Parse(time.RFC3339, t)
				if err1 == nil && err2 == nil {
					entry.durMins = int(doneTime.Sub(claimedTime).Minutes())
				}
			}
		}
	}

	// Collect spec entries in order
	for _, se := range specMap {
		specEntries = append(specEntries, *se)
	}

	sb.WriteString("\n")

	if len(contextEntries) > 0 {
		fmt.Fprintf(&sb, "Context retrieved (%d calls):\n", len(contextEntries))
		for _, ce := range contextEntries {
			fmt.Fprintf(&sb, "  - %s\n", ce.file)
			if ce.knowledge != "" {
				fmt.Fprintf(&sb, "    → %s\n", ce.knowledge)
			}
		}
		sb.WriteString("\n")
	}

	if len(knowledgeEntries) > 0 {
		sb.WriteString("Knowledge applied:\n")
		for _, ke := range knowledgeEntries {
			fmt.Fprintf(&sb, "  - %s: %s\n", ke.slug, ke.excerpt)
		}
		sb.WriteString("\n")
	}

	if len(specEntries) > 0 {
		sb.WriteString("Specs:\n")
		for _, se := range specEntries {
			if se.doneAt != "" {
				fmt.Fprintf(&sb, "  - %s: claimed → completed (%d min)\n", se.slug, se.durMins)
			} else if se.claimedAt != "" {
				fmt.Fprintf(&sb, "  - %s: claimed (in progress)\n", se.slug)
			} else {
				fmt.Fprintf(&sb, "  - %s\n", se.slug)
			}
		}
		sb.WriteString("\n")
	}

	if totalHeroCalls > 0 {
		fmt.Fprintf(&sb, "%d total Hero calls.\n", totalHeroCalls)
	} else if sess.HeroCalls > 0 {
		fmt.Fprintf(&sb, "%d total Hero calls.\n", sess.HeroCalls)
	}

	return sb.String(), nil
}
