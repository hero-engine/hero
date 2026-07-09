package sync

import "strings"

// MergeBody performs a diff3-style 3-way line merge of a long-text field
// (description/body). Non-conflicting hunks from either side merge cleanly; a
// genuinely conflicting hunk auto-resolves upstream-first — the remote hunk is
// kept and the local hunk is preserved inline under an informational
// `<!-- local edit (unmerged): … -->` marker, so nothing is lost and nothing
// blocks (no git-style <<<<<<< markers a human must resolve).
//
// The algorithm walks the two edit scripts (base→local, base→remote) in
// lockstep over the base lines:
//   - a base region both sides left unchanged → emit it once
//   - a region only one side changed → take that side's version
//   - a region both sides changed identically → take it once
//   - a region both sides changed differently → emit remote, then the local
//     region wrapped in the informational marker
//
// This is deliberately a line-granular merge (not char-level): it's robust and
// deterministic, and the preserve-local guarantee means an imperfect hunk split
// never loses data.
func MergeBody(base, local, remote string) string {
	baseLines := splitLines(base)
	localLines := splitLines(local)
	remoteLines := splitLines(remote)

	// Longest-common-subsequence alignment of base against each side gives, for
	// every base line, whether it survives on that side, and the inserted lines
	// each side placed before it.
	la := alignAgainstBase(baseLines, localLines)
	ra := alignAgainstBase(baseLines, remoteLines)

	var out []string
	// li/ri walk the aligned steps of each side in lockstep over base indices.
	i := 0 // index into baseLines
	for i <= len(baseLines) {
		// Emit insertions that both sides placed before base line i.
		localIns := la.insertBefore[i]
		remoteIns := ra.insertBefore[i]
		out = append(out, reconcileInsertions(localIns, remoteIns)...)

		if i == len(baseLines) {
			break
		}

		// Now the base line itself: kept iff kept on that side.
		keptLocal := la.keptBase[i]
		keptRemote := ra.keptBase[i]
		switch {
		case keptLocal && keptRemote:
			out = append(out, baseLines[i])
		case keptLocal && !keptRemote:
			// remote deleted this base line → honor the deletion.
		case !keptLocal && keptRemote:
			// local deleted this base line → honor the deletion.
		default:
			// both deleted → gone on both sides, drop it.
		}
		i++
	}

	return joinLines(out, base, local, remote)
}

// reconcileInsertions merges two blocks of inserted lines (local-side and
// remote-side) that both land at the same base position.
func reconcileInsertions(local, remote []string) []string {
	if len(local) == 0 && len(remote) == 0 {
		return nil
	}
	if len(local) == 0 {
		return remote
	}
	if len(remote) == 0 {
		return local
	}
	if linesEqual(local, remote) {
		// Both sides inserted the same thing → take it once.
		return remote
	}
	// Genuine conflict: upstream-first, local preserved in the marker.
	out := append([]string(nil), remote...)
	out = append(out, wrapLocalHunk(local)...)
	return out
}

// wrapLocalHunk renders a preserved-local body hunk as an informational,
// non-blocking marker block.
func wrapLocalHunk(local []string) []string {
	block := []string{LocalEditMarkerPrefix + " (unmerged):"}
	block = append(block, local...)
	block = append(block, "-->")
	return block
}

// baseAlignment records, per base line, whether it survives on a side, and the
// lines that side inserted before each base position (index 0..len(base)).
type baseAlignment struct {
	keptBase     []bool     // len == len(base); true if base[i] survives on this side
	insertBefore [][]string // len == len(base)+1; inserts before base line i
}

// alignAgainstBase computes the LCS of base and side, then derives which base
// lines survive and which side lines were inserted before each base position.
func alignAgainstBase(base, side []string) baseAlignment {
	lcs := lcsPairs(base, side)
	al := baseAlignment{
		keptBase:     make([]bool, len(base)),
		insertBefore: make([][]string, len(base)+1),
	}
	bi, si := 0, 0
	for _, p := range lcs {
		// Side lines before this common line, but after the previous common
		// base line, are insertions attributed to base position bi.
		for si < p.sideIdx {
			al.insertBefore[bi] = append(al.insertBefore[bi], side[si])
			si++
		}
		// Base lines skipped before the common line were deleted (kept=false).
		bi = p.baseIdx
		al.keptBase[bi] = true
		bi++
		si = p.sideIdx + 1
	}
	// Trailing side insertions after the last common line.
	for si < len(side) {
		al.insertBefore[len(base)] = append(al.insertBefore[len(base)], side[si])
		si++
	}
	return al
}

type lcsPair struct {
	baseIdx int
	sideIdx int
}

// lcsPairs returns the index pairs of a longest common subsequence of a and b.
func lcsPairs(a, b []string) []lcsPair {
	n, m := len(a), len(b)
	// dp[i][j] = LCS length of a[i:] and b[j:].
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var pairs []lcsPair
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			pairs = append(pairs, lcsPair{baseIdx: i, sideIdx: j})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			i++
		} else {
			j++
		}
	}
	return pairs
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// joinLines re-joins merged lines. If none of the inputs had a trailing
// newline, none is added; the join simply reverses splitLines.
func joinLines(lines []string, _, _, _ string) string {
	return strings.Join(lines, "\n")
}
