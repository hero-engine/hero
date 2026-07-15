package spec

import "testing"

// TestParseSections_FencedHeadingsAreNotSections is the core regression: a
// spec that documents Hero's own format with a fenced markdown example must
// keep the example inside the section that contains it, and must not forge a
// section named after the quoted heading.
func TestParseSections_FencedHeadingsAreNotSections(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Approach

We render the ledger like this:

`+"```"+`markdown
## Unchecked
- [ ] example
`+"```"+`

Tail line that must survive.

## Changes

- real changes
`)

	if _, ok := s.Sections["unchecked"]; ok {
		t.Errorf("forged phantom section %q from a fenced heading; sections = %v", "unchecked", keysOf(s.Sections))
	}

	approach := s.Sections["approach"]
	if !contains(approach, "Tail line that must survive") {
		t.Errorf("approach truncated at the fence; got %q", approach)
	}
	if !contains(approach, "## Unchecked") {
		t.Errorf("approach lost the fenced example body; got %q", approach)
	}
	if !contains(s.Sections["changes"], "real changes") {
		t.Errorf("changes = %q, want the real section to still parse", s.Sections["changes"])
	}
}

// TestParseSections_FencedHeadingDoesNotOverwriteRealSection covers the
// silent-content-loss case: because Sections is a map, a phantom that collides
// with a real section name overwrites it with no error. The fenced example is
// deliberately placed *after* the real `## Changes` so the phantom is the last
// write — the order in which content is actually lost.
func TestParseSections_FencedHeadingDoesNotOverwriteRealSection(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Changes

- real changes

## Approach

Quoting the changes heading:

`+"```"+`markdown
## Changes
- fake changes from the example
`+"```"+`
`)

	got := s.Sections["changes"]
	if !contains(got, "real changes") {
		t.Errorf("changes = %q, want the real section content preserved", got)
	}
	if contains(got, "fake changes from the example") {
		t.Errorf("changes = %q, want the fenced example not to overwrite the real section", got)
	}
}

// TestParseSections_IndentedFence guards the report's note that heading
// detection trims leading space before matching, so an indented example block
// must still be recognized as a fence.
func TestParseSections_IndentedFence(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Approach

- A list item with a nested example:

  `+"```"+`markdown
  ## Nested
  `+"```"+`

  Still inside approach.
`)

	if _, ok := s.Sections["nested"]; ok {
		t.Errorf("indented fence did not suppress heading detection; sections = %v", keysOf(s.Sections))
	}
	if !contains(s.Sections["approach"], "Still inside approach") {
		t.Errorf("approach = %q, want content after the indented fence", s.Sections["approach"])
	}
}

// TestParseSections_NestedFenceLengths verifies the tracker keys on the
// opening run's length: a ````-delimited block quoting a ``` block must not
// close on the inner fence.
func TestParseSections_NestedFenceLengths(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Approach

`+"````"+`markdown
`+"```"+`go
x := 1
`+"```"+`
## Inner
`+"````"+`

Tail of approach.

## Changes

- real
`)

	if _, ok := s.Sections["inner"]; ok {
		t.Errorf("inner fence closed the outer block early; sections = %v", keysOf(s.Sections))
	}
	if !contains(s.Sections["approach"], "Tail of approach") {
		t.Errorf("approach = %q, want content after the nested fence", s.Sections["approach"])
	}
}

// TestParseSections_TildeFence covers the other CommonMark fence character.
func TestParseSections_TildeFence(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Approach

~~~markdown
## Tilde
~~~

Tail of approach.
`)

	if _, ok := s.Sections["tilde"]; ok {
		t.Errorf("tilde fence not tracked; sections = %v", keysOf(s.Sections))
	}
	if !contains(s.Sections["approach"], "Tail of approach") {
		t.Errorf("approach = %q, want content after the tilde fence", s.Sections["approach"])
	}
}

// TestParseSections_UnclosedFence pins the behavior for a malformed spec: an
// unclosed fence swallows the rest of the document rather than silently
// resuming section detection. This matches how markdown renderers treat it.
func TestParseSections_UnclosedFence(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Approach

`+"```"+`markdown
## Changes
- never closed
`)

	if _, ok := s.Sections["changes"]; ok {
		t.Errorf("unclosed fence should not yield a changes section; sections = %v", keysOf(s.Sections))
	}
}

// TestParseSections_RealHeadingsStillParse is the guard against the fix
// over-reaching: ordinary specs with no fences must be unaffected.
func TestParseSections_RealHeadingsStillParse(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Approach

approach body

## Changes

- a change

## Acceptance criteria

- **AC-1:** THE SYSTEM SHALL work
`)

	for _, name := range []string{"approach", "changes", "acceptance criteria"} {
		if _, ok := s.Sections[name]; !ok {
			t.Errorf("missing section %q; sections = %v", name, keysOf(s.Sections))
		}
	}
	if s.Sections["approach"] != "approach body" {
		t.Errorf("approach = %q, want %q", s.Sections["approach"], "approach body")
	}
}

// TestParseACBlock_FencedACIsNotAnEntry covers the same flaw in the
// acceptance-criteria scanner: a quoted `AC-N:` example must not become an
// addressable criterion.
func TestParseACBlock_FencedACIsNotAnEntry(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Acceptance criteria

- **AC-1:** THE SYSTEM SHALL parse real criteria

The format looks like:

`+"```"+`markdown
- **AC-99:** THE SYSTEM SHALL be an example only
`+"```"+`
`)

	acs := s.ParseAcceptanceCriteria()
	for _, ac := range acs {
		if ac.ID == "AC-99" {
			t.Errorf("fenced example became an addressable criterion: %#v", acs)
		}
	}
	if len(acs) != 1 || acs[0].ID != "AC-1" {
		t.Errorf("got %#v, want exactly AC-1", acs)
	}
}

func TestFenceTracker(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		inCode []bool // expected inCode() after each line
	}{
		{
			name:   "simple backtick block",
			lines:  []string{"```", "code", "```", "after"},
			inCode: []bool{true, true, false, false},
		},
		{
			name:   "info string opens",
			lines:  []string{"```markdown", "## x", "```"},
			inCode: []bool{true, true, false},
		},
		{
			name:   "closing fence may not carry an info string",
			lines:  []string{"```go", "x", "```go", "still in", "```"},
			inCode: []bool{true, true, true, true, false},
		},
		{
			name:   "longer fence closes a shorter opening",
			lines:  []string{"```", "x", "````"},
			inCode: []bool{true, true, false},
		},
		{
			name:   "shorter fence does not close a longer opening",
			lines:  []string{"````", "```", "still in", "````"},
			inCode: []bool{true, true, true, false},
		},
		{
			name:   "tilde and backtick do not cross-close",
			lines:  []string{"~~~", "```", "still in", "~~~"},
			inCode: []bool{true, true, true, false},
		},
		{
			name:   "fewer than three markers is not a fence",
			lines:  []string{"``", "not code"},
			inCode: []bool{false, false},
		},
		{
			name:   "inline code with backtick in info does not open",
			lines:  []string{"``a ` b``", "not code"},
			inCode: []bool{false, false},
		},
		{
			name:   "indented fence still opens",
			lines:  []string{"  ```markdown", "  ## x", "  ```"},
			inCode: []bool{true, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fenceTracker
			for i, line := range tt.lines {
				f.mark(line)
				if got := f.inCode(); got != tt.inCode[i] {
					t.Errorf("after line %d (%q): inCode() = %v, want %v", i, line, got, tt.inCode[i])
				}
			}
		})
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
