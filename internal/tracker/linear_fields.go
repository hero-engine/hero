package tracker

import (
	"fmt"
	"os"
)

// UpdateFields writes canonical fields to a Linear issue via the
// issueUpdate GraphQL mutation. v1 supports the scalar fields Linear
// exposes directly on IssueUpdateInput without ID resolution: title,
// description, and points (→ estimate). priority and labels require
// workflow-state / label-ID lookups and are skipped with a warning
// (forward-compat, per the spec's vocabulary-mapping risk note).
func (l *linear) UpdateFields(issueID string, fields map[string]Value) error {
	if len(fields) == 0 {
		return nil
	}

	input := map[string]interface{}{}
	for name, val := range fields {
		switch name {
		case "title":
			input["title"] = val.Str
		case "description":
			input["description"] = val.Str
		case "points":
			if val.Kind == ValueInt {
				input["estimate"] = float64(val.Int)
			}
		default:
			fmt.Fprintf(os.Stderr, "Warning: linear adapter does not support field %q; skipping\n", name)
		}
	}
	if len(input) == 0 {
		return nil
	}

	internalID, err := l.resolveIssueID(issueID)
	if err != nil {
		return err
	}

	query := `mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
		issueUpdate(id: $id, input: $input) {
			success
		}
	}`
	variables := map[string]interface{}{
		"id":    internalID,
		"input": input,
	}

	result, err := l.graphql(query, variables)
	if err != nil {
		return fmt.Errorf("updating fields: %w", err)
	}
	data, ok := result["issueUpdate"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response shape from Linear")
	}
	if success, _ := data["success"].(bool); !success {
		return fmt.Errorf("linear issueUpdate returned success=false")
	}
	return nil
}

// GetFields fetches the current canonical content-field values from a
// Linear issue. v1 reads title, description, and estimate (→ points) —
// the fields UpdateFields can write — so the diff path round-trips
// cleanly.
func (l *linear) GetFields(issueID string) (map[string]Value, error) {
	query := `query GetIssue($id: String!) {
		issue(id: $id) {
			title
			description
			estimate
		}
	}`
	result, err := l.graphql(query, map[string]interface{}{"id": issueID})
	if err != nil {
		return nil, fmt.Errorf("getting fields: %w", err)
	}
	issueData, ok := result["issue"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}

	out := map[string]Value{}
	if title, _ := issueData["title"].(string); title != "" {
		out["title"] = StringValue(title)
	}
	if desc, _ := issueData["description"].(string); desc != "" {
		out["description"] = StringValue(desc)
	}
	if est, ok := issueData["estimate"].(float64); ok {
		out["points"] = IntValue(int(est))
	}
	return out, nil
}
