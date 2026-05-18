package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// markdown_invocations.go — extractor and validator for `hero <command>`
// invocations referenced in markdown surfaces shipped or rendered by this
// repo. The companion test (markdown_drift_test.go) walks every surface
// and asserts each invocation resolves against rootCmd; the CLI surface
// (`hero docs check --invocations`) runs the same scan on demand.
//
// Precedent: internal/cli/hints.go + hints_test.go cover Go-string
// invocations emitted from code. This file is the markdown analog.

// Invocation records a single extracted `hero <args...>` invocation from
// a markdown source.
type Invocation struct {
	// File is the source path (project-relative for on-disk files; a
	// synthetic <rendered:...> path for in-memory rendered content).
	File string
	// Line is the 1-indexed line number where the invocation begins.
	Line int
	// Raw is the literal text starting at "hero " through the captured
	// args — useful for the failure message.
	Raw string
	// Args is the tokenized argument slice (without the leading "hero")
	// suitable for passing to cobra.Command.Traverse.
	Args []string
}

// invocationPattern matches `hero <subcommand>` where <subcommand> is a
// single lowercase token starting with [a-z]. The trailing capture greedily
// grabs the rest of the line so the tokenizer can split it; tokenization
// stops at the first prose-shaped token.
//
// Prefix alternatives (the part before `hero`):
//   - (?m)^[ \t]*       line start with optional indent (covers fenced
//                       code blocks and bare CLI examples)
//   - [\x60(\[<]        an opening delimiter: backtick, paren, bracket,
//                       angle (covers `hero foo`, (hero foo), etc.)
//   - [^a-zA-Z0-9\s] +  any non-word, non-whitespace char followed by
//                       horizontal whitespace (covers list bullets `- `,
//                       table cells `| `, shell prompts `$ `, etc.)
//
// This deliberately rejects prose like "the hero framework" or "via
// hero status" — the char immediately before the whitespace must not
// be a word character.
var invocationPattern = regexp.MustCompile(
	`(?m)(?:^[ \t]*|[\x60(\[<]|[^a-zA-Z0-9\s][ \t]+)hero[ \t]+([a-z][a-z0-9-]*)((?:[ \t]+\S+)*)`,
)

// argTokenRune reports whether a rune is allowed inside a CLI argument
// token. Anything outside this set marks the token as prose and stops
// tokenization for the current invocation.
//
// We allow: a-z A-Z 0-9 . _ / = - + : @ , and the placeholder chars < >
// so that `<spec>`, `<alias>`, `--mode=advisory`, `services/auth`,
// `--files=foo.go`, `2026-05-18`, `host:port` all survive. We reject
// quotes, parens, brackets, backticks, and most punctuation so prose
// like "hero status, but…" tokenizes only to ["status"].
var argTokenRune = func(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}
	switch r {
	case '.', '_', '/', '=', '-', '<', '>', '+', ':', '@', ',':
		return true
	}
	return false
}

// ignoreMarker is the per-line escape hatch that suppresses extraction
// on the same line or the immediately preceding line. Lines describing
// intentionally broken invocations (e.g. inside a bug report's narrative)
// can mark themselves with this comment.
const ignoreMarker = "drift-test:ignore"

// htmlCommentRE strips HTML comments before extraction. Multi-line spans
// are handled by scanning the full content first; we then preserve newline
// count so line numbers reported in Invocation.Line stay accurate.
var htmlCommentRE = regexp.MustCompile(`(?s)<!--.*?-->`)

// ExtractInvocations parses markdown content and returns every
// `hero <command>` invocation it finds. Path is recorded on each result
// for reporting; it is not read from disk.
//
// Rules (see file header for the full design):
//   - First arg token must match [a-z][a-z0-9-]* (filters prose mentions
//     like "the hero framework").
//   - Subsequent args are tokenized on whitespace and stop at the first
//     token containing prose characters (anything outside argTokenRune)
//     or ending in trailing punctuation.
//   - HTML comments are stripped before extraction, preserving newlines
//     so line numbers stay accurate.
//   - A line containing the `drift-test:ignore` marker (or whose
//     immediately preceding line contains it) is skipped.
func ExtractInvocations(path string, content []byte) []Invocation {
	// Scan the ORIGINAL content line-by-line to detect ignore markers
	// (those markers live inside HTML comments, which the stripping pass
	// would erase). Walk the stripped content in parallel for invocation
	// matching, since stripped content preserves newline positions.
	stripped := stripHTMLCommentsPreservingLines(content)

	origLines := splitMarkdownLines(content)
	strippedLines := splitMarkdownLines(stripped)
	if len(origLines) != len(strippedLines) {
		// Defensive: line counts should always match because comment
		// stripping preserves newlines. If they ever diverge, fall back
		// to the stripped count so indexing stays safe.
		if len(strippedLines) < len(origLines) {
			origLines = origLines[:len(strippedLines)]
		} else {
			strippedLines = strippedLines[:len(origLines)]
		}
	}

	var out []Invocation
	prevIgnore := false
	for i, line := range strippedLines {
		lineNum := i + 1
		hasIgnore := strings.Contains(origLines[i], ignoreMarker)
		skip := hasIgnore || prevIgnore
		// Carry the ignore marker forward exactly one line.
		prevIgnore = hasIgnore
		if skip {
			continue
		}

		matches := invocationPattern.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			// m[2:4] is the subcommand capture; m[4:6] is the trailing
			// args span (possibly empty).
			sub := line[m[2]:m[3]]
			tail := ""
			if m[5] > m[4] {
				tail = line[m[4]:m[5]]
			}
			args := tokenizeArgs(sub, tail)
			raw := "hero " + strings.TrimSpace(sub+tail)
			out = append(out, Invocation{
				File: path,
				Line: lineNum,
				Raw:  raw,
				Args: args,
			})
		}
	}
	return out
}

// splitMarkdownLines splits content on '\n' WITHOUT trimming or skipping empties,
// so line numbers (1-indexed) line up exactly with the original content.
func splitMarkdownLines(content []byte) []string {
	if len(content) == 0 {
		return nil
	}
	s := string(content)
	lines := strings.Split(s, "\n")
	// If the file ends in '\n', strings.Split produces a trailing empty
	// element that doesn't correspond to a real line. Drop it.
	if len(lines) > 0 && lines[len(lines)-1] == "" && strings.HasSuffix(s, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// stripHTMLCommentsPreservingLines removes <!-- ... --> spans, replacing
// each removed character that is NOT a newline with a space. This keeps
// the byte offset of every newline (and therefore the line number of
// every subsequent token) identical to the original.
func stripHTMLCommentsPreservingLines(content []byte) []byte {
	return htmlCommentRE.ReplaceAllFunc(content, func(match []byte) []byte {
		out := make([]byte, len(match))
		for i, b := range match {
			if b == '\n' {
				out[i] = '\n'
			} else {
				out[i] = ' '
			}
		}
		return out
	})
}

// tokenizeArgs splits the trailing args span into individual tokens,
// stopping at the first prose-shaped token. The leading subcommand is
// always included.
//
// Each token is first stripped of surrounding markdown noise (trailing
// backticks, closing parens/brackets, sentence punctuation) before
// validation. A token containing prose-only characters after stripping
// terminates tokenization — everything from that token onward is
// treated as prose, not args.
func tokenizeArgs(sub, tail string) []string {
	args := []string{sub}
	for _, raw := range strings.Fields(tail) {
		tok := stripTrailingDelimiters(raw)
		if tok == "" {
			break
		}
		if !isArgToken(tok) {
			break
		}
		args = append(args, tok)
	}
	return args
}

// stripTrailingDelimiters removes closing-delimiter / sentence-tail
// characters that markdown commonly glues onto the last token of an
// invocation (e.g. trailing backtick, period, comma, parens). Stripping
// is repeated until the tail rune is something we want to keep, so
// "list`." becomes "list" in one pass.
//
// A token consisting entirely of "." is preserved — that's the standard
// "current directory" argument used in commands like
// `hero install project . --target opencode`.
func stripTrailingDelimiters(tok string) string {
	if tok == "." || tok == ".." {
		return tok
	}
	for len(tok) > 0 {
		last := tok[len(tok)-1]
		switch last {
		case '`', ')', ']', '"', '\'', '.', ',', ';', ':', '!', '?':
			tok = tok[:len(tok)-1]
			continue
		}
		break
	}
	return tok
}

// isArgToken returns true if tok consists entirely of allowed
// argument-shaped characters. Called after stripTrailingDelimiters so
// the caller has already removed common closing punctuation.
func isArgToken(tok string) bool {
	if tok == "" {
		return false
	}
	for _, r := range tok {
		if !argTokenRune(r) {
			return false
		}
	}
	return true
}

// ValidateInvocation resolves an invocation's args against the cobra
// command tree rooted at root and verifies any --flag tokens exist on
// the resolved command (or its inherited flag set). Returns nil on
// success; otherwise an error describing the resolution failure.
//
// Mirrors the validation in hints_test.go so the two surfaces enforce
// the same bar.
func ValidateInvocation(root *cobra.Command, inv Invocation) error {
	if len(inv.Args) == 0 {
		return fmt.Errorf("empty args")
	}
	// Drop --flag tokens before traversal: Traverse follows positional
	// path tokens, and flags would be interpreted as args to the leaf.
	pathArgs := make([]string, 0, len(inv.Args))
	for _, a := range inv.Args {
		if strings.HasPrefix(a, "--") {
			continue
		}
		pathArgs = append(pathArgs, a)
	}

	cmd, _, err := root.Traverse(pathArgs)
	if err != nil {
		return fmt.Errorf("traverse: %w", err)
	}
	// Traverse returns the root when the first token isn't a known
	// subcommand of root. Detect that explicitly: this is the `hero pull`
	// regression class (a top-level subcommand referenced in docs that
	// was never registered).
	if len(pathArgs) > 0 && cmd == root {
		return fmt.Errorf("unknown subcommand %q", pathArgs[0])
	}

	for _, a := range inv.Args {
		if !strings.HasPrefix(a, "--") {
			continue
		}
		flagName := strings.TrimPrefix(a, "--")
		if eq := strings.IndexByte(flagName, '='); eq >= 0 {
			flagName = flagName[:eq]
		}
		// Cobra injects --help (and -h) lazily during Execute; the flag
		// isn't on Flags() / InheritedFlags() at registration time. Treat
		// it as always-valid to avoid false positives on every documented
		// `hero <foo> --help` example.
		if flagName == "help" {
			continue
		}
		if cmd.Flags().Lookup(flagName) == nil && cmd.InheritedFlags().Lookup(flagName) == nil {
			return fmt.Errorf("flag --%s not found on %s", flagName, cmd.CommandPath())
		}
	}
	return nil
}

// isExcludedDir reports whether a project-relative path falls inside a
// directory that intentionally documents broken invocations (bug specs,
// planning notes). Those surfaces are skipped by default so investigation
// narratives don't have to litter themselves with drift-test:ignore
// markers.
func isExcludedDir(rel string) bool {
	// Use forward-slash form for matching; callers pass filepath.ToSlash'd
	// project-relative paths.
	rel = strings.TrimPrefix(rel, "./")
	for _, pref := range []string{
		".hero/specs/",
		".hero/planning/",
	} {
		if strings.HasPrefix(rel, pref) {
			return true
		}
	}
	return false
}
