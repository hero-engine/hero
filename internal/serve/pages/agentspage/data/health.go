package data

// HealthInputs is the per-request input bundle for the agents-health
// page. v1 surfaces empty metrics until agent-contribution-tracking
// wires real aggregation; the page renders cleanly off zeros.
type HealthInputs struct {
	ProjectRoot string
	HeroDir     string
	Range       string // "24h" | "7d" | "30d" | "all"
}

// Health is the /agents/health payload.
type Health struct {
	Range       string
	Leaderboard []HealthRow
}

// HealthRow is one row in the per-agent leaderboard.
type HealthRow struct {
	Agent           string
	Runs            int
	WinRate         string
	InterruptRate   string
	CostPerMerged   string
	P95Turns        int
	RecentFailures  int
}

// LoadHealth returns the health payload. Range falls back to "7d" when
// empty.
func LoadHealth(in HealthInputs) Health {
	rng := in.Range
	if rng == "" {
		rng = "7d"
	}
	return Health{Range: rng, Leaderboard: []HealthRow{}}
}
