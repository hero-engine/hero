package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Audit event types for spec lifecycle tracking.
const (
	EventSpecCreated   = "spec.created"
	EventSpecScored    = "spec.scored"
	EventSpecApproved  = "spec.approved"
	EventSpecDelivered = "spec.delivered"
	EventSpecMerged    = "spec.merged"
	EventSpecCompleted = "spec.completed"
	EventSpecSynced    = "spec.synced"
	EventPRChecked     = "pr.checked"
	EventPRLinked      = "pr.linked"
	EventConventionMatched = "convention.matched"
	EventScopeDrift    = "scope.drift"
)

// AuditEntry is a typed wrapper for recording audit events.
type AuditEntry struct {
	OrgID     string
	RepoID    *string
	UserID    *string
	EventType string
	Payload   map[string]interface{}
}

// RecordAudit records a structured audit event.
func (db *DB) RecordAudit(ctx context.Context, entry AuditEntry) error {
	payload, err := json.Marshal(entry.Payload)
	if err != nil {
		return fmt.Errorf("marshaling audit payload: %w", err)
	}
	return db.RecordEvent(ctx, entry.OrgID, entry.RepoID, entry.UserID, entry.EventType, payload)
}

// ListAuditEvents returns audit events filtered by type, with time range support.
func (db *DB) ListAuditEvents(ctx context.Context, orgID string, eventTypes []string, since, until *time.Time, limit, offset int) ([]ActivityEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, org_id, repo_id, user_id, event_type, payload, created_at
		FROM activity_events
		WHERE org_id = $1`
	args := []interface{}{orgID}
	argN := 2

	if len(eventTypes) > 0 {
		query += fmt.Sprintf(" AND event_type = ANY($%d)", argN)
		args = append(args, eventTypes)
		argN++
	}

	if since != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argN)
		args = append(args, *since)
		argN++
	}

	if until != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argN)
		args = append(args, *until)
		argN++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, limit, offset)

	rows, err := db.Conn(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing audit events: %w", err)
	}
	defer rows.Close()

	var events []ActivityEvent
	for rows.Next() {
		var e ActivityEvent
		if err := rows.Scan(&e.ID, &e.OrgID, &e.RepoID, &e.UserID,
			&e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning audit event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// AuditSummary returns counts of each event type for an org.
func (db *DB) AuditSummary(ctx context.Context, orgID string, since *time.Time) (map[string]int, error) {
	query := `
		SELECT event_type, COUNT(*)
		FROM activity_events
		WHERE org_id = $1`
	args := []interface{}{orgID}

	if since != nil {
		query += " AND created_at >= $2"
		args = append(args, *since)
	}

	query += " GROUP BY event_type ORDER BY event_type"

	rows, err := db.Conn(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("audit summary: %w", err)
	}
	defer rows.Close()

	summary := make(map[string]int)
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("scanning audit summary: %w", err)
		}
		summary[eventType] = count
	}
	return summary, rows.Err()
}

// AnalyticsOverview computes aggregate KPI metrics from audit/activity data.
func (db *DB) AnalyticsOverview(ctx context.Context, orgID string, since, until time.Time) (*AnalyticsOverviewResult, error) {
	result := &AnalyticsOverviewResult{}

	// Specs delivered count
	err := db.Conn(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM activity_events
		WHERE org_id = $1 AND event_type = 'spec.delivered'
			AND created_at >= $2 AND created_at <= $3
	`, orgID, since, until).Scan(&result.SpecsDelivered)
	if err != nil {
		return nil, fmt.Errorf("analytics delivered count: %w", err)
	}

	// Avg time to merge (approved → merged, in hours)
	err = db.Conn(ctx).QueryRow(ctx, `
		WITH approved AS (
			SELECT payload->>'slug' AS slug, MIN(created_at) AS approved_at
			FROM activity_events
			WHERE org_id = $1 AND event_type = 'spec.approved'
				AND created_at >= $2 AND created_at <= $3
			GROUP BY slug
		),
		merged AS (
			SELECT payload->>'slug' AS slug, MIN(created_at) AS merged_at
			FROM activity_events
			WHERE org_id = $1 AND event_type = 'spec.merged'
				AND created_at >= $2 AND created_at <= $3
			GROUP BY slug
		)
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (m.merged_at - a.approved_at)) / 3600), 0)
		FROM approved a
		JOIN merged m ON a.slug = m.slug
		WHERE m.merged_at > a.approved_at
	`, orgID, since, until).Scan(&result.AvgTimeToMergeHours)
	if err != nil {
		return nil, fmt.Errorf("analytics avg time to merge: %w", err)
	}

	// Rework rate: specs with multiple deliveries / total delivered
	err = db.Conn(ctx).QueryRow(ctx, `
		WITH delivery_counts AS (
			SELECT payload->>'slug' AS slug, COUNT(*) AS cnt
			FROM activity_events
			WHERE org_id = $1 AND event_type = 'spec.delivered'
				AND created_at >= $2 AND created_at <= $3
			GROUP BY slug
		)
		SELECT
			CASE WHEN COUNT(*) = 0 THEN 0
			ELSE ROUND(COUNT(*) FILTER (WHERE cnt > 1)::numeric / COUNT(*)::numeric * 100, 1)
			END
		FROM delivery_counts
	`, orgID, since, until).Scan(&result.ReworkRatePct)
	if err != nil {
		return nil, fmt.Errorf("analytics rework rate: %w", err)
	}

	// AI leverage: ratio of agent-delivered specs
	err = db.Conn(ctx).QueryRow(ctx, `
		SELECT
			CASE WHEN COUNT(*) = 0 THEN 0
			ELSE ROUND(COUNT(*) FILTER (WHERE payload->>'claimed_by' IS NOT NULL AND payload->>'claimed_by' != '')::numeric / COUNT(*)::numeric * 100, 1)
			END
		FROM activity_events
		WHERE org_id = $1 AND event_type = 'spec.delivered'
			AND created_at >= $2 AND created_at <= $3
	`, orgID, since, until).Scan(&result.AILeveragePct)
	if err != nil {
		return nil, fmt.Errorf("analytics ai leverage: %w", err)
	}

	return result, nil
}

// AnalyticsOverviewResult holds computed KPI metrics.
type AnalyticsOverviewResult struct {
	SpecsDelivered      int     `json:"specs_delivered"`
	AvgTimeToMergeHours float64 `json:"avg_time_to_merge_hours"`
	ReworkRatePct       float64 `json:"rework_rate_pct"`
	AILeveragePct       float64 `json:"ai_leverage_pct"`
}

// Pattern types for institutional memory.
type Pattern struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"` // info, warning, danger
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
}

// MinePatterns analyzes historical events to find recurring patterns.
func (db *DB) MinePatterns(ctx context.Context, orgID string) ([]Pattern, error) {
	var patterns []Pattern

	// 1. High rework files: files that appear in multiple reworked specs
	reworkFiles, err := db.findReworkFiles(ctx, orgID)
	if err == nil && len(reworkFiles) > 0 {
		patterns = append(patterns, reworkFiles...)
	}

	// 2. Slow-to-merge specs: specs that took abnormally long
	slowSpecs, err := db.findSlowSpecs(ctx, orgID)
	if err == nil && len(slowSpecs) > 0 {
		patterns = append(patterns, slowSpecs...)
	}

	// 3. PR check failure hotspots
	failHotspots, err := db.findPRFailureHotspots(ctx, orgID)
	if err == nil && len(failHotspots) > 0 {
		patterns = append(patterns, failHotspots...)
	}

	return patterns, nil
}

func (db *DB) findReworkFiles(ctx context.Context, orgID string) ([]Pattern, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		WITH reworked_specs AS (
			SELECT payload->>'slug' AS slug
			FROM activity_events
			WHERE org_id = $1 AND event_type = 'spec.delivered'
			GROUP BY payload->>'slug'
			HAVING COUNT(*) > 1
		),
		rework_files AS (
			SELECT unnest(s.files_touched) AS file_path, COUNT(DISTINCT s.slug) AS spec_count
			FROM specs s
			JOIN repos r ON r.id = s.repo_id
			JOIN reworked_specs rs ON rs.slug = s.slug
			WHERE r.org_id = $1
			GROUP BY file_path
			HAVING COUNT(DISTINCT s.slug) >= 2
			ORDER BY spec_count DESC
			LIMIT 10
		)
		SELECT file_path, spec_count FROM rework_files
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var filePath string
		var count int
		if err := rows.Scan(&filePath, &count); err != nil {
			continue
		}
		patterns = append(patterns, Pattern{
			Type:        "rework_hotspot",
			Severity:    "warning",
			Title:       fmt.Sprintf("Rework hotspot: %s", filePath),
			Description: fmt.Sprintf("This file appeared in %d reworked specs. Changes here frequently require revision.", count),
			Evidence:    map[string]interface{}{"file": filePath, "rework_count": count},
		})
	}
	return patterns, rows.Err()
}

func (db *DB) findSlowSpecs(ctx context.Context, orgID string) ([]Pattern, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		WITH lifecycle AS (
			SELECT payload->>'slug' AS slug,
				MIN(CASE WHEN event_type = 'spec.created' THEN created_at END) AS created,
				MIN(CASE WHEN event_type = 'spec.delivered' THEN created_at END) AS delivered
			FROM activity_events
			WHERE org_id = $1
			AND event_type IN ('spec.created', 'spec.delivered')
			GROUP BY payload->>'slug'
			HAVING MIN(CASE WHEN event_type = 'spec.delivered' THEN created_at END) IS NOT NULL
		),
		stats AS (
			SELECT AVG(EXTRACT(EPOCH FROM (delivered - created))) AS avg_sec,
				STDDEV(EXTRACT(EPOCH FROM (delivered - created))) AS stddev_sec
			FROM lifecycle
		)
		SELECT l.slug, EXTRACT(EPOCH FROM (l.delivered - l.created)) / 3600 AS hours
		FROM lifecycle l, stats s
		WHERE EXTRACT(EPOCH FROM (l.delivered - l.created)) > (s.avg_sec + 2 * COALESCE(s.stddev_sec, 0))
		AND s.avg_sec > 0
		ORDER BY hours DESC
		LIMIT 5
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var slug string
		var hours float64
		if err := rows.Scan(&slug, &hours); err != nil {
			continue
		}
		patterns = append(patterns, Pattern{
			Type:        "slow_delivery",
			Severity:    "info",
			Title:       fmt.Sprintf("Slow delivery: %s", slug),
			Description: fmt.Sprintf("Took %.0f hours from creation to delivery (significantly above average).", hours),
			Evidence:    map[string]interface{}{"slug": slug, "hours": hours},
		})
	}
	return patterns, rows.Err()
}

func (db *DB) findPRFailureHotspots(ctx context.Context, orgID string) ([]Pattern, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT payload->>'repo' AS repo,
			COUNT(*) FILTER (WHERE payload->>'result' = 'fail') AS failures,
			COUNT(*) AS total
		FROM activity_events
		WHERE org_id = $1 AND event_type = 'pr.checked'
		GROUP BY payload->>'repo'
		HAVING COUNT(*) FILTER (WHERE payload->>'result' = 'fail') > 2
		ORDER BY failures DESC
		LIMIT 5
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var repo string
		var failures, total int
		if err := rows.Scan(&repo, &failures, &total); err != nil {
			continue
		}
		pct := float64(failures) / float64(total) * 100
		patterns = append(patterns, Pattern{
			Type:        "pr_failure_hotspot",
			Severity:    "danger",
			Title:       fmt.Sprintf("PR check failures in %s", repo),
			Description: fmt.Sprintf("%d of %d PR checks failed (%.0f%%). Specs targeting this repo need extra attention.", failures, total, pct),
			Evidence:    map[string]interface{}{"repo": repo, "failures": failures, "total": total, "failure_pct": pct},
		})
	}
	return patterns, rows.Err()
}

// ConventionSuggestion represents a suggested convention based on observed patterns.
type ConventionSuggestion struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	Rationale   string `json:"rationale"`
	Severity    string `json:"severity"` // info, warning
}

// SuggestConventions analyzes patterns to suggest new conventions.
func (db *DB) SuggestConventions(ctx context.Context, orgID string) ([]ConventionSuggestion, error) {
	var suggestions []ConventionSuggestion

	// Check for files that frequently cause rework → suggest review conventions
	rows, err := db.Conn(ctx).Query(ctx, `
		WITH reworked AS (
			SELECT payload->>'slug' AS slug
			FROM activity_events
			WHERE org_id = $1 AND event_type = 'spec.delivered'
			GROUP BY payload->>'slug'
			HAVING COUNT(*) > 1
		),
		rework_dirs AS (
			SELECT
				regexp_replace(unnest(s.files_touched), '/[^/]+$', '') AS dir,
				COUNT(DISTINCT s.slug) AS spec_count
			FROM specs s
			JOIN repos r ON r.id = s.repo_id
			JOIN reworked rs ON rs.slug = s.slug
			WHERE r.org_id = $1
			GROUP BY dir
			HAVING COUNT(DISTINCT s.slug) >= 3
			ORDER BY spec_count DESC
			LIMIT 5
		)
		SELECT dir, spec_count FROM rework_dirs
	`, orgID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dir string
			var count int
			if err := rows.Scan(&dir, &count); err != nil {
				continue
			}
			suggestions = append(suggestions, ConventionSuggestion{
				Name:        fmt.Sprintf("Review gate for %s", dir),
				Scope:       dir + "/**",
				Description: fmt.Sprintf("Require additional review for changes in %s/", dir),
				Rationale:   fmt.Sprintf("This directory appeared in %d reworked specs. A review gate could catch issues earlier.", count),
				Severity:    "warning",
			})
		}
	}

	// Check for repos with high PR failure rates → suggest pre-submit checks
	rows2, err := db.Conn(ctx).Query(ctx, `
		SELECT payload->>'repo' AS repo,
			COUNT(*) FILTER (WHERE payload->>'result' = 'fail') AS failures,
			COUNT(*) AS total
		FROM activity_events
		WHERE org_id = $1 AND event_type = 'pr.checked'
		GROUP BY payload->>'repo'
		HAVING COUNT(*) FILTER (WHERE payload->>'result' = 'fail')::float / COUNT(*)::float > 0.3
		AND COUNT(*) >= 5
		ORDER BY failures DESC
		LIMIT 3
	`, orgID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var repo string
			var failures, total int
			if err := rows2.Scan(&repo, &failures, &total); err != nil {
				continue
			}
			suggestions = append(suggestions, ConventionSuggestion{
				Name:        fmt.Sprintf("Pre-submit validation for %s", repo),
				Scope:       "*",
				Description: fmt.Sprintf("Add pre-submit spec validation checks for repo %s", repo),
				Rationale:   fmt.Sprintf("%d of %d PR checks failed (%.0f%%). Pre-submit validation would catch issues before PR creation.", failures, total, float64(failures)/float64(total)*100),
				Severity:    "warning",
			})
		}
	}

	return suggestions, nil
}
