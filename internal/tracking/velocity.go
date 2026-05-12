package tracking

import (
	"sort"
	"time"
)

// AgentVelocity holds per-agent delivery metrics.
type AgentVelocity struct {
	Agent       string  `json:"agent"`
	SpecsDone   int     `json:"specs_done"`
	AvgDays     float64 `json:"avg_days_per_spec"`
	FastestSlug string  `json:"fastest_slug,omitempty"`
	SlowestSlug string  `json:"slowest_slug,omitempty"`
}

// CalcVelocity calculates per-agent velocity from events log.
// since limits the window (zero value = all time).
func CalcVelocity(events []ClaimEvent, since time.Time) []AgentVelocity {
	type specKey struct {
		slug  string
		agent string
	}
	type specTimes struct {
		claimedAt time.Time
		doneAt    time.Time
		done      bool
	}

	// Build map of claim times per (slug, agent)
	claimTimes := make(map[specKey]time.Time)
	completions := make(map[specKey]time.Time)

	for _, evt := range events {
		if !since.IsZero() && evt.At.Before(since) {
			continue
		}
		key := specKey{slug: evt.Slug, agent: evt.Agent}
		switch evt.Event {
		case "claimed":
			// Only record first claim per (slug, agent) pair
			if _, exists := claimTimes[key]; !exists {
				claimTimes[key] = evt.At
			}
		case "completed":
			completions[key] = evt.At
		}
	}

	// Compute per-agent durations
	type agentStats struct {
		totalDays   float64
		count       int
		fastestDays float64
		fastestSlug string
		slowestDays float64
		slowestSlug string
	}

	agentMap := make(map[string]*agentStats)

	for key, doneAt := range completions {
		claimedAt, ok := claimTimes[key]
		if !ok {
			continue
		}
		dur := doneAt.Sub(claimedAt)
		days := dur.Hours() / 24.0
		if days < 0 {
			days = 0
		}

		stats, exists := agentMap[key.agent]
		if !exists {
			stats = &agentStats{
				fastestDays: days,
				fastestSlug: key.slug,
				slowestDays: days,
				slowestSlug: key.slug,
			}
			agentMap[key.agent] = stats
		}

		stats.totalDays += days
		stats.count++

		if days < stats.fastestDays {
			stats.fastestDays = days
			stats.fastestSlug = key.slug
		}
		if days > stats.slowestDays {
			stats.slowestDays = days
			stats.slowestSlug = key.slug
		}
	}

	var results []AgentVelocity
	for agent, stats := range agentMap {
		if stats.count == 0 {
			continue
		}
		avg := stats.totalDays / float64(stats.count)
		results = append(results, AgentVelocity{
			Agent:       agent,
			SpecsDone:   stats.count,
			AvgDays:     avg,
			FastestSlug: stats.fastestSlug,
			SlowestSlug: stats.slowestSlug,
		})
	}

	// Sort by specs done descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].SpecsDone > results[j].SpecsDone
	})

	return results
}
