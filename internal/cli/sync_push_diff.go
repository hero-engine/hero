package cli

import (
	"github.com/hero-engine/hero/internal/tracker"
)

// fieldClass is the push classification for a canonical field. Mirrors
// internal/spectypes.Classification but kept local to the push command
// so the diff is a pure function over a small, explicit table (the
// "Tracker round-trip" table in the spec) rather than dragging the full
// registry-load chain into the hot path.
type fieldClass string

const (
	classContent  fieldClass = "content"   // local → tracker; pushable
	classOrgState fieldClass = "org-state" // tracker-owned; never pushed
)

// ClassifiedField names a canonical hero-side field, its push class,
// and a value-parse hint ("string" | "int" | "strings") used to coerce
// a raw frontmatter scalar into a tracker.Value.
type ClassifiedField struct {
	Name  string
	Class fieldClass
	Hint  string
}

// pushFields is the canonical content-field table for field-level push,
// taken directly from the spec's "Tracker round-trip" matrix. Only
// content-classified fields are diffed and pushed; org-state fields are
// tracker-authoritative and refused on the --field path. status is
// handled by the existing status-push path (sync spec / sync jira), so
// it is intentionally absent here — this command owns the non-status
// content fields.
var pushFields = []ClassifiedField{
	{Name: "title", Class: classContent, Hint: "string"},
	{Name: "description", Class: classContent, Hint: "string"},
	{Name: "points", Class: classContent, Hint: "int"},
	{Name: "priority", Class: classContent, Hint: "string"},
	{Name: "labels", Class: classContent, Hint: "strings"},
}

// orgStateFields are tracker-authoritative; a --field naming one of
// these is refused with a non-zero exit (AC: org-state refusal).
var orgStateFields = map[string]bool{
	"tracker_id":         true,
	"created":            true,
	"tracker_updated_at": true,
	"reporter":           true,
	"jira_status":        true,
	"github_status":      true,
	"linear_status":      true,
	"jira_updated_at":    true,
	"github_updated_at":  true,
	"gitlab_updated_at":  true,
	"linear_updated_at":  true,
}

// classifyField returns the ClassifiedField for a canonical name, or
// false if the name is neither a known content field nor a known
// org-state field. Unknown names are treated as skippable (forward
// compat) by callers.
func classifyField(name string) (ClassifiedField, bool) {
	for _, f := range pushFields {
		if f.Name == name {
			return f, true
		}
	}
	if orgStateFields[name] {
		return ClassifiedField{Name: name, Class: classOrgState}, true
	}
	return ClassifiedField{}, false
}

// Diff computes the patch to push: for each content-classified field,
// the local value is included when it differs from the remote value.
// org-state fields are never included. A field present locally but
// absent remotely (added) and a field whose value changed both
// register; a field absent locally (no local intent) is left untouched
// — push is local→tracker and never clears a tracker field the spec
// doesn't mention. Unchanged fields are skipped, which is what makes a
// second invocation a no-op (idempotency).
func Diff(local, remote map[string]tracker.Value, fields []ClassifiedField) map[string]tracker.Value {
	patch := map[string]tracker.Value{}
	for _, f := range fields {
		if f.Class != classContent {
			continue
		}
		lv, hasLocal := local[f.Name]
		if !hasLocal {
			continue
		}
		rv, hasRemote := remote[f.Name]
		if !hasRemote || !lv.Equal(rv) {
			patch[f.Name] = lv
		}
	}
	return patch
}
