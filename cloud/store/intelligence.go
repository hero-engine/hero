package store

import (
	"context"
	"fmt"
	"time"
)

// StackProfile represents an org's anonymized tech stack for similarity matching.
type StackProfile struct {
	OrgID       string    `json:"org_id"`
	Languages   []string  `json:"languages"`
	Frameworks  []string  `json:"frameworks"`
	SpecCount   int       `json:"spec_count"`
	MemberCount int       `json:"member_count"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GlobalPattern is an anonymized pattern observed across multiple orgs.
type GlobalPattern struct {
	ID          string                 `json:"id"`
	PatternType string                `json:"pattern_type"`
	Category    string                 `json:"category"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Frequency   int                    `json:"frequency"`
	Languages   []string               `json:"languages"`
	Frameworks  []string               `json:"frameworks"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// GlobalConvention is a convention template derived from popular conventions across orgs.
type GlobalConvention struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Template    string   `json:"template"`
	Scope       []string `json:"scope"`
	Languages   []string `json:"languages"`
	Frameworks  []string `json:"frameworks"`
	Adoption    int      `json:"adoption"`
}

// Insight is a recommendation for an org based on cross-org intelligence.
type Insight struct {
	Type        string `json:"type"` // convention, pattern, warning
	Title       string `json:"title"`
	Description string `json:"description"`
	Confidence  int    `json:"confidence"` // 0-100
	Source      string `json:"source"`     // e.g. "85 similar projects"
}

// UpsertStackProfile creates or updates an org's tech stack profile.
func (db *DB) UpsertStackProfile(ctx context.Context, p *StackProfile) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO stack_profiles (org_id, languages, frameworks, spec_count, member_count, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (org_id)
		DO UPDATE SET
			languages = EXCLUDED.languages,
			frameworks = EXCLUDED.frameworks,
			spec_count = EXCLUDED.spec_count,
			member_count = EXCLUDED.member_count,
			updated_at = now()
	`, p.OrgID, p.Languages, p.Frameworks, p.SpecCount, p.MemberCount)
	if err != nil {
		return fmt.Errorf("upserting stack profile: %w", err)
	}
	return nil
}

// GetStackProfile returns an org's stack profile.
func (db *DB) GetStackProfile(ctx context.Context, orgID string) (*StackProfile, error) {
	var p StackProfile
	err := db.pool.QueryRow(ctx, `
		SELECT org_id, languages, frameworks, spec_count, member_count, updated_at
		FROM stack_profiles WHERE org_id = $1
	`, orgID).Scan(&p.OrgID, &p.Languages, &p.Frameworks, &p.SpecCount, &p.MemberCount, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting stack profile: %w", err)
	}
	return &p, nil
}

// UpsertGlobalPattern inserts or increments frequency of a global pattern.
func (db *DB) UpsertGlobalPattern(ctx context.Context, p *GlobalPattern) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO global_patterns (pattern_type, category, title, description, frequency, languages, frameworks, metadata)
		VALUES ($1, $2, $3, $4, 1, $5, $6, $7)
		ON CONFLICT (pattern_type, title)
		DO UPDATE SET
			frequency = global_patterns.frequency + 1,
			description = CASE WHEN length(EXCLUDED.description) > length(global_patterns.description)
				THEN EXCLUDED.description ELSE global_patterns.description END,
			languages = (SELECT array_agg(DISTINCT v) FROM unnest(global_patterns.languages || EXCLUDED.languages) v),
			frameworks = (SELECT array_agg(DISTINCT v) FROM unnest(global_patterns.frameworks || EXCLUDED.frameworks) v),
			updated_at = now()
	`, p.PatternType, p.Category, p.Title, p.Description, p.Languages, p.Frameworks, p.Metadata)
	if err != nil {
		return fmt.Errorf("upserting global pattern: %w", err)
	}
	return nil
}

// UpsertGlobalConvention inserts or increments adoption of a global convention.
func (db *DB) UpsertGlobalConvention(ctx context.Context, c *GlobalConvention) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO global_conventions (slug, title, category, description, template, scope, languages, frameworks, adoption)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1)
		ON CONFLICT (slug)
		DO UPDATE SET
			adoption = global_conventions.adoption + 1,
			languages = (SELECT array_agg(DISTINCT v) FROM unnest(global_conventions.languages || EXCLUDED.languages) v),
			frameworks = (SELECT array_agg(DISTINCT v) FROM unnest(global_conventions.frameworks || EXCLUDED.frameworks) v),
			updated_at = now()
	`, c.Slug, c.Title, c.Category, c.Description, c.Template, c.Scope, c.Languages, c.Frameworks)
	if err != nil {
		return fmt.Errorf("upserting global convention: %w", err)
	}
	return nil
}

// GetInsights returns cross-org recommendations for an org based on its stack profile.
func (db *DB) GetInsights(ctx context.Context, orgID string) ([]Insight, error) {
	// Check opt-in
	var optedIn bool
	err := db.pool.QueryRow(ctx, `
		SELECT COALESCE(opted_in, false) FROM intelligence_opt_in WHERE org_id = $1
	`, orgID).Scan(&optedIn)
	if err != nil || !optedIn {
		return nil, nil // Not opted in, return empty
	}

	// Get org stack profile
	profile, err := db.GetStackProfile(ctx, orgID)
	if err != nil {
		return nil, nil // No profile, can't match
	}

	var insights []Insight

	// 1. Find conventions popular among similar stacks that this org hasn't adopted
	convRows, err := db.pool.Query(ctx, `
		SELECT gc.title, gc.description, gc.category, gc.adoption, gc.languages
		FROM global_conventions gc
		WHERE gc.languages && $1
		AND gc.adoption >= 3
		AND NOT EXISTS (
			SELECT 1 FROM conventions c
			JOIN repos r ON r.id = c.repo_id
			WHERE r.org_id = $2
			AND lower(c.title) = lower(gc.title)
		)
		ORDER BY gc.adoption DESC
		LIMIT 5
	`, profile.Languages, orgID)
	if err == nil {
		defer convRows.Close()
		for convRows.Next() {
			var title, desc, category string
			var adoption int
			var langs []string
			if err := convRows.Scan(&title, &desc, &category, &adoption, &langs); err != nil {
				continue
			}
			confidence := min(adoption*10, 95)
			insights = append(insights, Insight{
				Type:        "convention",
				Title:       fmt.Sprintf("Consider adding: %s", title),
				Description: desc,
				Confidence:  confidence,
				Source:      fmt.Sprintf("adopted by %d similar projects", adoption),
			})
		}
	}

	// 2. Find patterns common in similar stacks
	patRows, err := db.pool.Query(ctx, `
		SELECT gp.title, gp.description, gp.pattern_type, gp.frequency
		FROM global_patterns gp
		WHERE gp.languages && $1
		AND gp.frequency >= 3
		ORDER BY gp.frequency DESC
		LIMIT 5
	`, profile.Languages)
	if err == nil {
		defer patRows.Close()
		for patRows.Next() {
			var title, desc, patType string
			var freq int
			if err := patRows.Scan(&title, &desc, &patType, &freq); err != nil {
				continue
			}
			confidence := min(freq*8, 90)
			insights = append(insights, Insight{
				Type:        "pattern",
				Title:       title,
				Description: desc,
				Confidence:  confidence,
				Source:      fmt.Sprintf("observed in %d projects", freq),
			})
		}
	}

	return insights, nil
}

// SetIntelligenceOptIn sets an org's opt-in status for cross-org intelligence.
func (db *DB) SetIntelligenceOptIn(ctx context.Context, orgID string, optIn bool) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO intelligence_opt_in (org_id, opted_in, opted_at, updated_at)
		VALUES ($1, $2, CASE WHEN $2 THEN now() ELSE NULL END, now())
		ON CONFLICT (org_id)
		DO UPDATE SET
			opted_in = EXCLUDED.opted_in,
			opted_at = CASE WHEN EXCLUDED.opted_in THEN now() ELSE intelligence_opt_in.opted_at END,
			updated_at = now()
	`, orgID, optIn)
	if err != nil {
		return fmt.Errorf("setting opt-in: %w", err)
	}
	return nil
}

// GetIntelligenceOptIn returns whether an org has opted into cross-org intelligence.
func (db *DB) GetIntelligenceOptIn(ctx context.Context, orgID string) (bool, error) {
	var optedIn bool
	err := db.pool.QueryRow(ctx, `
		SELECT COALESCE(opted_in, false) FROM intelligence_opt_in WHERE org_id = $1
	`, orgID).Scan(&optedIn)
	if err != nil {
		return false, nil // Not found = not opted in
	}
	return optedIn, nil
}

// AggregateOrgPatterns rolls up an org's patterns into the global pool (anonymized).
// This should be called periodically (e.g., daily cron).
func (db *DB) AggregateOrgPatterns(ctx context.Context, orgID string) error {
	// Check opt-in
	optedIn, err := db.GetIntelligenceOptIn(ctx, orgID)
	if err != nil || !optedIn {
		return nil
	}

	// Get org profile for language/framework tagging
	profile, err := db.GetStackProfile(ctx, orgID)
	if err != nil {
		return nil
	}

	// Mine org patterns and contribute anonymized versions
	patterns, err := db.MinePatterns(ctx, orgID)
	if err != nil {
		return fmt.Errorf("mining patterns for aggregation: %w", err)
	}

	for _, p := range patterns {
		gp := &GlobalPattern{
			PatternType: p.Type,
			Category:    p.Severity,
			Title:       p.Title,
			Description: p.Description,
			Languages:   profile.Languages,
			Frameworks:  profile.Frameworks,
		}
		// Anonymize: strip org-specific file paths, repo names
		// Title patterns like "Rework hotspot: src/api/handler.go" → "Rework hotspot: API handler files"
		// For now, use the pattern type as the title for anonymization
		switch p.Type {
		case "rework_hotspot":
			gp.Title = "Rework hotspot in modified files"
			gp.Description = "Certain files frequently require rework after initial delivery."
		case "slow_delivery":
			gp.Title = "Slow spec delivery pattern"
			gp.Description = "Some specs take significantly longer than average to deliver."
		case "pr_failure_hotspot":
			gp.Title = "PR check failure pattern"
			gp.Description = "Certain repositories have higher-than-average PR check failure rates."
		}
		if err := db.UpsertGlobalPattern(ctx, gp); err != nil {
			continue // Best effort
		}
	}

	// Contribute conventions (anonymized)
	convRows, err := db.pool.Query(ctx, `
		SELECT c.slug, c.title, c.status, c.scope, c.content
		FROM conventions c
		JOIN repos r ON r.id = c.repo_id
		WHERE r.org_id = $1 AND c.status = 'active'
	`, orgID)
	if err == nil {
		defer convRows.Close()
		for convRows.Next() {
			var slug, title, status, content string
			var scope []string
			if err := convRows.Scan(&slug, &title, &status, &scope, &content); err != nil {
				continue
			}
			gc := &GlobalConvention{
				Slug:       slug,
				Title:      title,
				Category:   categorizeConvention(title, content),
				Description: truncate(content, 200),
				Template:   content,
				Scope:      scope,
				Languages:  profile.Languages,
				Frameworks: profile.Frameworks,
			}
			if err := db.UpsertGlobalConvention(ctx, gc); err != nil {
				continue
			}
		}
	}

	return nil
}

// RunGlobalAggregation aggregates patterns from all opted-in orgs.
func (db *DB) RunGlobalAggregation(ctx context.Context) error {
	rows, err := db.pool.Query(ctx, `
		SELECT org_id FROM intelligence_opt_in WHERE opted_in = true
	`)
	if err != nil {
		return fmt.Errorf("listing opted-in orgs: %w", err)
	}
	defer rows.Close()

	var orgIDs []string
	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			continue
		}
		orgIDs = append(orgIDs, orgID)
	}

	for _, orgID := range orgIDs {
		_ = db.AggregateOrgPatterns(ctx, orgID)
	}

	return nil
}

// categorizeConvention infers a category from convention title/content.
func categorizeConvention(title, content string) string {
	lower := fmt.Sprintf("%s %s", title, content)
	switch {
	case contains(lower, "test", "testing", "coverage"):
		return "testing"
	case contains(lower, "review", "approve", "pr"):
		return "code-review"
	case contains(lower, "security", "auth", "token", "secret"):
		return "security"
	case contains(lower, "style", "format", "lint"):
		return "code-style"
	case contains(lower, "doc", "readme", "comment"):
		return "documentation"
	case contains(lower, "deploy", "release", "ci", "cd"):
		return "deployment"
	case contains(lower, "error", "log", "monitor"):
		return "observability"
	default:
		return "general"
	}
}

func contains(s string, terms ...string) bool {
	for _, t := range terms {
		if len(t) > 0 && len(s) >= len(t) {
			for i := 0; i <= len(s)-len(t); i++ {
				if s[i:i+len(t)] == t {
					return true
				}
			}
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
