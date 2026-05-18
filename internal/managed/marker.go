package managed

import (
	"fmt"
	"regexp"
	"strings"
)

// marker.go — versioned managed-region primitive used to inject
// Hero-managed content into otherwise user-owned files (AGENTS.md
// primarily, plus CLAUDE.md and NEXT.md, and per-harness config files
// like opencode.json/.aider.conf.yml).
//
// Region format:
//
//	<!-- hero:managed-start v=<version> -->
//	<arbitrary Hero-generated content>
//	<!-- hero:managed-end -->
//
// Hero owns everything between the markers (inclusive). User owns
// everything outside. Re-running the orchestrator regenerates the
// region in place, preserving outside content byte-for-byte.
//
// Detection is stateless. The functions here never read remembered
// state — always re-derive from file content. This is what makes
// user-authored content inviolate across upgrades.

const (
	managedStartPrefix = "<!-- hero:managed-start"
	managedEndMarker   = "<!-- hero:managed-end -->"

	// Legacy single-line marker from pre-P1 installs. We recognize it
	// for migration so existing CLAUDE.md / AGENTS.md content can be
	// converted to the versioned form automatically.
	legacyMarker = "<!-- hero:managed -->"
)

// versionedStartRe matches `<!-- hero:managed-start v=<version> -->`
// and captures the version string. Tolerates whitespace inside the
// comment.
var versionedStartRe = regexp.MustCompile(`<!--\s*hero:managed-start\s+v=([^\s>]+)\s*-->`)

// Region describes the position and content of a Hero-managed region
// inside a file. (Distinct from the higher-level Region orchestrator
// in region.go — this is the parsed view of an existing file; that is
// the writer.)
type Region struct {
	// Present is false if the file has no managed region at all.
	Present bool

	// Legacy is true when the file uses the old single-marker style
	// (`<!-- hero:managed -->` ... `<!-- hero:managed -->`) rather than
	// the versioned start/end pair. Legacy regions are upgraded to the
	// versioned form on the next install/upgrade.
	Legacy bool

	// Version is the version stamp from the start marker (empty for
	// legacy regions). Used to detect "managed region is at an older
	// Hero version, needs regeneration."
	Version string

	// StartIdx is the byte offset of the first character of the start
	// marker.
	StartIdx int

	// EndIdx is the byte offset just past the last character of the end
	// marker. So content[StartIdx:EndIdx] is the full region including
	// markers; content[:StartIdx] and content[EndIdx:] are the user-owned
	// prefix and suffix respectively.
	EndIdx int

	// Body is the content between the markers (exclusive of markers,
	// trimmed of leading/trailing newlines). This is what callers
	// compare against newly-generated content to detect user edits
	// inside the region.
	Body string
}

// FindManagedRegion locates the Hero-managed region in content. Returns
// Present=false when no markers are found at all. Recognizes both the
// versioned form and the legacy single-marker form.
func FindManagedRegion(content string) Region {
	// Versioned form first (preferred).
	if m := versionedStartRe.FindStringSubmatchIndex(content); m != nil {
		startIdx := m[0]
		afterStart := m[1] // index just past the start marker
		version := content[m[2]:m[3]]

		endIdx := strings.Index(content[afterStart:], managedEndMarker)
		if endIdx >= 0 {
			endAbs := afterStart + endIdx + len(managedEndMarker)
			body := strings.Trim(content[afterStart:afterStart+endIdx], "\n")
			return Region{
				Present:  true,
				Version:  version,
				StartIdx: startIdx,
				EndIdx:   endAbs,
				Body:     body,
			}
		}
		// Start marker without end marker — degenerate but recoverable:
		// treat as "everything from start to EOF" so the next regenerate
		// rewrites it cleanly.
		body := strings.Trim(content[afterStart:], "\n")
		return Region{
			Present:  true,
			Version:  version,
			StartIdx: startIdx,
			EndIdx:   len(content),
			Body:     body,
		}
	}

	// Legacy form: first occurrence of the bare marker, search for the
	// matching closing marker.
	startIdx := strings.Index(content, legacyMarker)
	if startIdx < 0 {
		return Region{Present: false}
	}
	afterStart := startIdx + len(legacyMarker)
	rel := strings.Index(content[afterStart:], legacyMarker)
	if rel < 0 {
		// Single legacy marker — historically Hero sometimes wrote just
		// one at the top. Treat the whole rest of file as the region so
		// it gets regenerated cleanly into the versioned form.
		body := strings.Trim(content[afterStart:], "\n")
		return Region{
			Present:  true,
			Legacy:   true,
			StartIdx: startIdx,
			EndIdx:   len(content),
			Body:     body,
		}
	}
	endAbs := afterStart + rel + len(legacyMarker)
	body := strings.Trim(content[afterStart:afterStart+rel], "\n")
	return Region{
		Present:  true,
		Legacy:   true,
		StartIdx: startIdx,
		EndIdx:   endAbs,
		Body:     body,
	}
}

// RenderManagedRegion produces a managed-region block — start marker,
// body, end marker — at the given version. The body is the caller's
// Hero-generated content (no markers in it; this function adds them).
func RenderManagedRegion(version, body string) string {
	if version == "" {
		version = "dev"
	}
	body = strings.Trim(body, "\n")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<!-- hero:managed-start v=%s -->\n", version))
	if body != "" {
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	sb.WriteString(managedEndMarker)
	sb.WriteString("\n")
	return sb.String()
}

// InsertManagedRegion produces a new file content with the managed
// region inserted or updated.
//
// The three cases:
//
//  1. file has no managed region: insert `region` at the top of the
//     file (after the H1 title line if one exists), preserving all
//     existing content below.
//  2. file has a managed region: replace just the region (everything
//     between and including the markers) with the new region. Outside
//     content is byte-for-byte preserved.
//  3. file is empty: result is just the region.
//
// `region` is expected to be the output of RenderManagedRegion.
func InsertManagedRegion(existing, region string) string {
	region = strings.TrimRight(region, "\n") + "\n"

	if existing == "" {
		return region
	}

	mr := FindManagedRegion(existing)
	if mr.Present {
		// Replace the region in place; preserve everything outside.
		prefix := existing[:mr.StartIdx]
		suffix := existing[mr.EndIdx:]
		// Normalize seam whitespace: ensure exactly one newline between
		// prefix and region and between region and suffix.
		prefix = strings.TrimRight(prefix, "\n")
		suffix = strings.TrimLeft(suffix, "\n")

		var sb strings.Builder
		if prefix != "" {
			sb.WriteString(prefix)
			sb.WriteString("\n\n")
		}
		sb.WriteString(region)
		if suffix != "" {
			sb.WriteString("\n")
			sb.WriteString(suffix)
		}
		out := sb.String()
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		return out
	}

	// No managed region. Insert at the top, but after the first H1 if
	// the file starts with one — model sees the project title, then
	// Hero's managed content, then the rest.
	lines := strings.SplitN(existing, "\n", 2)
	if len(lines) >= 1 && strings.HasPrefix(lines[0], "# ") {
		// Preserve the H1 line and any blank line immediately after it.
		head := lines[0]
		rest := ""
		if len(lines) == 2 {
			rest = lines[1]
		}
		rest = strings.TrimLeft(rest, "\n")
		var sb strings.Builder
		sb.WriteString(head)
		sb.WriteString("\n\n")
		sb.WriteString(region)
		if rest != "" {
			sb.WriteString("\n")
			sb.WriteString(rest)
		}
		out := sb.String()
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		return out
	}

	// No H1: region goes at the very top.
	var sb strings.Builder
	sb.WriteString(region)
	sb.WriteString("\n")
	sb.WriteString(strings.TrimLeft(existing, "\n"))
	out := sb.String()
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}

// IsLegacyHeroStub returns true when content consists entirely of a
// Hero managed region (legacy or versioned) with no user content
// outside it. Used to detect "this CLAUDE.md is a Hero stub from a
// prior install and can safely be replaced with a symlink/shim" — vs
// the user-authored case where any meaningful content exists outside
// Hero markers.
//
// Whitespace and blank lines outside the region don't count as user
// content.
func IsLegacyHeroStub(content string) bool {
	mr := FindManagedRegion(content)
	if !mr.Present {
		return false
	}
	prefix := strings.TrimSpace(content[:mr.StartIdx])
	suffix := strings.TrimSpace(content[mr.EndIdx:])
	return prefix == "" && suffix == ""
}
