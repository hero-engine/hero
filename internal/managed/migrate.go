package managed

import "strings"

// migrate.go — one-shot legacy-layout consolidation.
//
// Pre-consolidation, two writers each wrote a separate marker pair into
// AGENTS.md / CLAUDE.md / NEXT.md: the install pair
// (`<!-- hero:managed-start v=X -->` ... `<!-- hero:managed-end -->`)
// and the snapshot pointer pair
// (`<!-- >>> hero snapshot pointer (managed) >>> -->` ...
// `<!-- <<< hero snapshot pointer (managed) <<< -->`).
//
// On the first run of Writer.Write against a legacy file, the snapshot
// pointer block is stripped (markers and enclosed content). The pointer
// contributor then re-emits its content inside the consolidated install
// region during the same write.
//
// All operations are pure string ops over file content. No I/O.
// User content outside both marker pairs is never touched.

const (
	legacySnapshotStart = "<!-- >>> hero snapshot pointer (managed) >>> -->"
	legacySnapshotEnd   = "<!-- <<< hero snapshot pointer (managed) <<< -->"
)

// detectLegacySnapshotBlock returns the byte offsets of the legacy
// snapshot pointer block (markers inclusive) in content, or
// found=false if no complete block is present. A start marker without
// a matching end marker counts as not-found (the file is malformed
// but conservative migration leaves it alone).
func detectLegacySnapshotBlock(content string) (start, end int, found bool) {
	startIdx := strings.Index(content, legacySnapshotStart)
	if startIdx < 0 {
		return 0, 0, false
	}
	afterStart := startIdx + len(legacySnapshotStart)
	rel := strings.Index(content[afterStart:], legacySnapshotEnd)
	if rel < 0 {
		// Half-open block; leave it alone so the user notices on the
		// next read and can hand-clean.
		return 0, 0, false
	}
	endAbs := afterStart + rel + len(legacySnapshotEnd)
	return startIdx, endAbs, true
}

// stripLegacySnapshotBlock returns content with the legacy snapshot
// pointer block removed (markers and enclosed content). Seam
// whitespace is normalized: at most one blank line where the block
// used to be. If no legacy block is present, content is returned
// unchanged.
//
// Idempotent: a second call against the stripped result is a no-op.
func stripLegacySnapshotBlock(content string) string {
	start, end, found := detectLegacySnapshotBlock(content)
	if !found {
		return content
	}

	prefix := content[:start]
	suffix := content[end:]

	// Normalize seam: collapse any run of newlines on the boundaries
	// into a single blank line (two newlines) when both sides have
	// content, or trim leading newlines from suffix when prefix is
	// empty.
	prefix = strings.TrimRight(prefix, "\n")
	suffix = strings.TrimLeft(suffix, "\n")

	var out string
	switch {
	case prefix == "" && suffix == "":
		out = ""
	case prefix == "":
		out = suffix
	case suffix == "":
		out = prefix + "\n"
	default:
		out = prefix + "\n\n" + suffix
	}
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}
