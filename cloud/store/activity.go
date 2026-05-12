package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ActivityEvent represents an event in the org activity feed.
type ActivityEvent struct {
	ID        string          `json:"id"`
	OrgID     string          `json:"org_id"`
	RepoID    *string         `json:"repo_id,omitempty"`
	UserID    *string         `json:"user_id,omitempty"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// RecordEvent inserts a new activity event.
func (db *DB) RecordEvent(ctx context.Context, orgID string, repoID, userID *string, eventType string, payload json.RawMessage) error {
	if payload == nil {
		payload = json.RawMessage("{}")
	}
	_, err := db.Conn(ctx).Exec(ctx, `
		INSERT INTO activity_events (org_id, repo_id, user_id, event_type, payload)
		VALUES ($1, $2, $3, $4, $5)
	`, orgID, repoID, userID, eventType, payload)
	if err != nil {
		return fmt.Errorf("recording event: %w", err)
	}
	return nil
}

// ListActivity returns recent activity events for an org.
func (db *DB) ListActivity(ctx context.Context, orgID string, limit, offset int) ([]ActivityEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT id, org_id, repo_id, user_id, event_type, payload, created_at
		FROM activity_events
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing activity: %w", err)
	}
	defer rows.Close()

	var events []ActivityEvent
	for rows.Next() {
		var e ActivityEvent
		if err := rows.Scan(&e.ID, &e.OrgID, &e.RepoID, &e.UserID,
			&e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// ListActivityFiltered returns activity events with type and time range filtering.
func (db *DB) ListActivityFiltered(ctx context.Context, orgID string, eventTypes []string, since, until *time.Time, limit, offset int) ([]ActivityEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, org_id, repo_id, user_id, event_type, payload, created_at
		FROM activity_events
		WHERE org_id = $1`
	args := []any{orgID}
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
		return nil, fmt.Errorf("listing filtered activity: %w", err)
	}
	defer rows.Close()

	var events []ActivityEvent
	for rows.Next() {
		var e ActivityEvent
		if err := rows.Scan(&e.ID, &e.OrgID, &e.RepoID, &e.UserID,
			&e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning filtered event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// ActivityHeatmap returns daily event counts for a time range.
func (db *DB) ActivityHeatmap(ctx context.Context, orgID string, since, until time.Time) ([]DayCount, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT date_trunc('day', created_at)::date AS day, COUNT(*)
		FROM activity_events
		WHERE org_id = $1 AND created_at >= $2 AND created_at <= $3
		GROUP BY day
		ORDER BY day
	`, orgID, since, until)
	if err != nil {
		return nil, fmt.Errorf("activity heatmap: %w", err)
	}
	defer rows.Close()

	var days []DayCount
	for rows.Next() {
		var d DayCount
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, err
		}
		days = append(days, d)
	}
	return days, rows.Err()
}

// DayCount is a date with an event count.
type DayCount struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

// DeliveryVelocity returns bucketed delivery counts over time.
func (db *DB) DeliveryVelocity(ctx context.Context, orgID string, since, until time.Time, interval string) ([]VelocityBucket, error) {
	trunc := "week"
	if interval == "day" {
		trunc = "day"
	} else if interval == "month" {
		trunc = "month"
	}

	rows, err := db.Conn(ctx).Query(ctx, fmt.Sprintf(`
		SELECT date_trunc('%s', created_at) AS period,
			COUNT(*) FILTER (WHERE event_type = 'spec.delivered') AS delivered,
			COUNT(*) FILTER (WHERE event_type = 'spec.created') AS created
		FROM activity_events
		WHERE org_id = $1 AND created_at >= $2 AND created_at <= $3
			AND event_type IN ('spec.delivered', 'spec.created')
		GROUP BY period
		ORDER BY period
	`, trunc), orgID, since, until)
	if err != nil {
		return nil, fmt.Errorf("delivery velocity: %w", err)
	}
	defer rows.Close()

	var buckets []VelocityBucket
	for rows.Next() {
		var b VelocityBucket
		if err := rows.Scan(&b.Period, &b.Delivered, &b.Created); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

// VelocityBucket is a time-bucketed delivery count.
type VelocityBucket struct {
	Period    time.Time `json:"period"`
	Delivered int       `json:"delivered"`
	Created   int       `json:"created"`
}
