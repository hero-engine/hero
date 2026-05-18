package snapshot

import (
	"fmt"
	"strings"
)

// Diff returns a plain text unified-style diff between two archive
// bodies. The implementation is intentionally simple — line-by-line
// "=", "-", "+" markers — to avoid pulling a diff library dependency.
//
// Callers pass the trimmed body (without frontmatter) for both sides.
// Pass "live" snapshot bytes for one side when comparing to the live
// SNAPSHOT.md.
func Diff(left, right string) string {
	if left == right {
		return ""
	}
	a := strings.Split(left, "\n")
	b := strings.Split(right, "\n")
	matrix := lcsLengths(a, b)
	var out strings.Builder
	emitDiff(&out, a, b, matrix, len(a), len(b))
	return out.String()
}

func lcsLengths(a, b []string) [][]int {
	m := len(a)
	n := len(b)
	matrix := make([][]int, m+1)
	for i := range matrix {
		matrix[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				matrix[i][j] = matrix[i-1][j-1] + 1
			} else if matrix[i-1][j] >= matrix[i][j-1] {
				matrix[i][j] = matrix[i-1][j]
			} else {
				matrix[i][j] = matrix[i][j-1]
			}
		}
	}
	return matrix
}

func emitDiff(out *strings.Builder, a, b []string, matrix [][]int, i, j int) {
	if i > 0 && j > 0 && a[i-1] == b[j-1] {
		emitDiff(out, a, b, matrix, i-1, j-1)
		fmt.Fprintf(out, "  %s\n", a[i-1])
		return
	}
	if j > 0 && (i == 0 || matrix[i][j-1] >= matrix[i-1][j]) {
		emitDiff(out, a, b, matrix, i, j-1)
		fmt.Fprintf(out, "+ %s\n", b[j-1])
		return
	}
	if i > 0 && (j == 0 || matrix[i][j-1] < matrix[i-1][j]) {
		emitDiff(out, a, b, matrix, i-1, j)
		fmt.Fprintf(out, "- %s\n", a[i-1])
		return
	}
}
