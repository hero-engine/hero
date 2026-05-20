// Package opsrunner runs an allowlisted set of `hero` CLI verbs as
// subprocesses on behalf of the Project section's Operations panel,
// streaming progress to the browser over SSE.
//
// The package is intentionally small and self-contained:
//
//   - allowlist.go — the fixed verb → args mapping (no shell pass-through).
//   - registry.go  — the in-memory job table + ring buffer used to
//     dedup concurrent starts and let late subscribers catch up.
//   - runner.go    — the public Runner type that spawns subprocesses
//     and writes SSE frames.
//
// Phase 3 of the hero-serve-project-section initiative.
package opsrunner

// Verbs is the canonical verb → `hero` CLI argv mapping. The keys are
// the public verb names that appear in the API path and in the
// rendered template; the values are the argv slices passed to the
// `hero` binary subprocess.
//
// The map is intentionally fixed: there is no shell pass-through, and
// adding a verb requires a code change here. Phase 4 may add a
// `stop-daemon` entry; nothing else is planned.
var Verbs = map[string][]string{
	"re-scan":           {"scan"},
	"re-index":          {"index"},
	"run-check":         {"check"},
	"refresh-queue":     {"queue", "write"},
	"capture-knowledge": {"capture"},
	"snapshot":          {"snapshot"},
	"export":            {"export"},
}

// VerbLabel returns a human-friendly label for the given verb. Unknown
// verbs return the verb itself so the template still renders without
// crashing.
func VerbLabel(verb string) string {
	switch verb {
	case "re-scan":
		return "Re-scan project"
	case "re-index":
		return "Re-index knowledge"
	case "run-check":
		return "Run health check"
	case "refresh-queue":
		return "Refresh queue"
	case "capture-knowledge":
		return "Capture knowledge"
	case "snapshot":
		return "Take snapshot"
	case "export":
		return "Export"
	default:
		return verb
	}
}

// VerbDescription returns a short one-line description used as the
// button tooltip / sub-label. Mirrors the CLI subcommand it dispatches.
func VerbDescription(verb string) string {
	switch verb {
	case "re-scan":
		return "Detect stack, refresh codebase summary."
	case "re-index":
		return "Rebuild the spec / knowledge index."
	case "run-check":
		return "Run workspace health checks."
	case "refresh-queue":
		return "Re-render .hero/QUEUE.md from the live spec set."
	case "capture-knowledge":
		return "Persist session learnings into the knowledge base."
	case "snapshot":
		return "Render the project-shape snapshot."
	case "export":
		return "Export spec set / metrics to disk."
	default:
		return ""
	}
}

// IsAllowed reports whether the given verb is in the fixed allowlist.
func IsAllowed(verb string) bool {
	_, ok := Verbs[verb]
	return ok
}

// AllVerbs returns the verb names in a stable, human-meaningful order
// for rendering. The order matches the Operations section's button
// row: scan → index → check → queue → capture → snapshot → export.
func AllVerbs() []string {
	return []string{
		"re-scan",
		"re-index",
		"run-check",
		"refresh-queue",
		"capture-knowledge",
		"snapshot",
		"export",
	}
}
