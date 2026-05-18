package managed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// region.go — Writer orchestrator: aggregates an ordered list of
// SectionContributor implementations into one managed region per file,
// runs the legacy two-block migration inline, and writes the consolidated
// layout via the marker primitives.
//
// Callers in internal/install and internal/snapshot construct a Writer
// with the contributors they want and call Write. There is no global
// registry: the canonical section order lives at the call site.

// Context is what contributors receive at render time. Kept small on
// purpose — anything load-bearing should be passed in by the
// orchestrator caller, not pulled from globals.
type Context struct {
	// File is the absolute path of the file being rendered (AGENTS.md /
	// CLAUDE.md / NEXT.md).
	File string

	// HeroVersion is the value used for the managed-region start-marker
	// version stamp.
	HeroVersion string

	// ProjectDir is the project root, for resolving relative pointer
	// paths in contributors that emit links.
	ProjectDir string
}

// SectionContributor is one named section inside the single managed
// region. Empty Render output causes the orchestrator to skip the
// section entirely (no heading emitted), so a contributor can no-op
// when it has nothing to contribute.
type SectionContributor interface {
	// SectionID returns a stable identifier used for ordering and
	// debugging. Not currently emitted into the rendered output.
	SectionID() string

	// SectionTitle returns the H2 heading rendered above the section
	// body inside the managed region. Empty title means the contributor
	// owns the entire body (no heading is added) — useful for the
	// install body which already starts with its own H2.
	SectionTitle() string

	// Render returns the markdown body for this section, without any
	// section heading (the orchestrator adds the H2 from SectionTitle).
	// An empty body skips the section.
	Render(ctx Context) (string, error)
}

// Writer is the per-file aggregator. Sections are written in the order
// they appear in the slice — there is no auto-sort. The caller declares
// the canonical order in one place.
//
// File creation rules when the target file doesn't exist:
//
//   - If DefaultH1 is non-empty, the file is created with that line as
//     the first line, a blank line, then the managed region.
//   - If DefaultH1 is empty, the file is created with just the managed
//     region at the top.
//
// On an existing file with no managed region, the region is inserted
// after the file's existing H1 (if any), preserving all content below.
type Writer struct {
	File      string
	Sections  []SectionContributor
	DefaultH1 string
}

// Write renders all sections, runs the legacy two-block migration if
// applicable, and splices the consolidated managed region into the
// file. Returns true if the file was written (or would be in dry-run
// mode), false if the file was already up-to-date.
//
// The write is idempotent: a second Write call with identical
// contributor output is a no-op.
func (w Writer) Write(ctx Context) (changed bool, err error) {
	if w.File == "" {
		return false, fmt.Errorf("managed.Writer.Write: File is required")
	}
	if ctx.File == "" {
		ctx.File = w.File
	}

	body, err := w.renderBody(ctx)
	if err != nil {
		return false, err
	}

	region := RenderManagedRegion(ctx.HeroVersion, body)

	existing := ""
	wasNew := true
	if data, readErr := os.ReadFile(w.File); readErr == nil {
		existing = string(data)
		wasNew = false
	}

	// Inline migration: strip the legacy snapshot pointer block if it
	// exists outside the install marker pair. The contributor will
	// re-emit the pointer inside the consolidated region during this
	// same write.
	if !wasNew {
		existing = stripLegacySnapshotBlock(existing)
	}

	var newContent string
	if existing == "" {
		newContent = renderFreshFile(region, w.DefaultH1)
	} else {
		newContent = InsertManagedRegion(existing, region)
	}

	if !wasNew && newContent == existing {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(w.File), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(w.File, []byte(newContent), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// RenderBody returns the body content (no markers) the Writer would
// render for the given context. Exposed for callers that need to see
// the body without writing (e.g. dry-run paths in internal/install).
func (w Writer) RenderBody(ctx Context) (string, error) {
	return w.renderBody(ctx)
}

func (w Writer) renderBody(ctx Context) (string, error) {
	var sb strings.Builder
	for i, sec := range w.Sections {
		body, err := sec.Render(ctx)
		if err != nil {
			return "", fmt.Errorf("managed: section %s: %w", sec.SectionID(), err)
		}
		body = strings.Trim(body, "\n")
		if body == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		if title := sec.SectionTitle(); title != "" {
			// Emit H2 heading above the section body. Body itself is
			// expected to use H3+ for any internal structure.
			fmt.Fprintf(&sb, "## %s\n\n", title)
		}
		sb.WriteString(body)
		_ = i
	}
	return sb.String(), nil
}

// renderFreshFile produces the content for a brand-new file: optional
// H1, blank line, region.
func renderFreshFile(region, defaultH1 string) string {
	var sb strings.Builder
	if defaultH1 != "" {
		sb.WriteString(defaultH1)
		sb.WriteString("\n\n")
	}
	sb.WriteString(region)
	out := sb.String()
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}

// PlanContent returns what the Writer would write for the given
// context, without touching the filesystem. Returns the existing
// content (for diffing), the would-be new content, and whether the
// target file currently exists. Used by dry-run paths.
func (w Writer) PlanContent(ctx Context) (existing, next string, exists bool, err error) {
	if w.File == "" {
		return "", "", false, fmt.Errorf("managed.Writer.PlanContent: File is required")
	}
	if ctx.File == "" {
		ctx.File = w.File
	}

	body, err := w.renderBody(ctx)
	if err != nil {
		return "", "", false, err
	}
	region := RenderManagedRegion(ctx.HeroVersion, body)

	if data, readErr := os.ReadFile(w.File); readErr == nil {
		existing = string(data)
		exists = true
	}

	migrated := existing
	if exists {
		migrated = stripLegacySnapshotBlock(existing)
	}

	if migrated == "" {
		next = renderFreshFile(region, w.DefaultH1)
	} else {
		next = InsertManagedRegion(migrated, region)
	}
	return existing, next, exists, nil
}
