package snapshot

import (
	"encoding/json"
	"time"
)

// jsonSnapshot is the wire-shape returned by FormatJSON. It mirrors
// the Snapshot struct but strips internal pointers and normalizes
// timestamps to RFC3339 strings.
type jsonSnapshot struct {
	ProjectName          string                `json:"project_name"`
	Mission              string                `json:"mission,omitempty"`
	GeneratedAt          string                `json:"generated_at"`
	Surfaces             []Surface             `json:"surfaces"`
	Assignments          []jsonAssignment      `json:"assignments"`
	Initiatives          []Initiative          `json:"initiatives"`
	RecentlyDone         []jsonRecentItem      `json:"recently_done"`
	NextUp               []NextItem            `json:"next_up"`
	Blockers             []Blocker             `json:"blockers"`
	StaleInFlight        []StaleItem           `json:"stale_in_flight"`
	AgedBugs             []AgedBug             `json:"aged_bugs"`
	UnassignedCount      int                   `json:"unassigned_count"`
	InferredCount        int                   `json:"inferred_count"`
	OverrideAppliedCount int                   `json:"override_applied_count"`
	OverrideEditedAt     string                `json:"override_edited_at,omitempty"`
	GenerationMillis     int64                 `json:"generation_millis"`
	SourceNodes          int                   `json:"source_nodes"`
	HasReleaseSignal     bool                  `json:"has_release_signal"`
}

type jsonAssignment struct {
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	SurfaceID     string `json:"surface_id"`
	ReleaseTarget string `json:"release_target,omitempty"`
	ReleaseSource string `json:"release_source,omitempty"`
	Inferred      bool   `json:"inferred,omitempty"`
}

type jsonRecentItem struct {
	SurfaceID   string `json:"surface_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	CompletedAt string `json:"completed_at"`
	Type        string `json:"type"`
}

func renderJSON(s *Snapshot) ([]byte, error) {
	if s == nil {
		return []byte("{}"), nil
	}
	js := jsonSnapshot{
		ProjectName:          s.ProjectName,
		Mission:              s.Mission,
		GeneratedAt:          s.GeneratedAt.UTC().Format(time.RFC3339),
		Surfaces:             s.Surfaces,
		Initiatives:          s.Initiatives,
		NextUp:               s.NextUp,
		Blockers:             s.Blockers,
		StaleInFlight:        s.StaleInFlight,
		AgedBugs:             s.AgedBugs,
		UnassignedCount:      s.UnassignedCount,
		InferredCount:        s.InferredCount,
		OverrideAppliedCount: s.OverrideAppliedCount,
		GenerationMillis:     s.GenerationMillis,
		SourceNodes:          s.SourceNodes,
		HasReleaseSignal:     s.HasReleaseSignal,
	}
	if !s.OverrideEditedAt.IsZero() {
		js.OverrideEditedAt = s.OverrideEditedAt.UTC().Format(time.RFC3339)
	}
	for _, a := range s.Assignments {
		if a.Spec == nil {
			continue
		}
		js.Assignments = append(js.Assignments, jsonAssignment{
			Slug:          a.Spec.Slug,
			Title:         a.Spec.Title,
			Type:          string(a.Spec.Type),
			Status:        string(a.Spec.Status),
			SurfaceID:     a.SurfaceID,
			ReleaseTarget: a.ReleaseTarget,
			ReleaseSource: a.ReleaseSource,
			Inferred:      a.Inferred,
		})
	}
	for _, r := range s.RecentlyDone {
		js.RecentlyDone = append(js.RecentlyDone, jsonRecentItem{
			SurfaceID:   r.SurfaceID,
			Slug:        r.Slug,
			Title:       r.Title,
			CompletedAt: r.CompletedAt.UTC().Format(time.RFC3339),
			Type:        string(r.Type),
		})
	}
	return json.MarshalIndent(js, "", "  ")
}
