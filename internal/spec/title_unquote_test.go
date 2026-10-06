package spec

import "testing"

// A quoted YAML title must read as its value, not with its quotes
// (hero-harness reported hero_list returning `"Title"` verbatim).
func TestParseTitleUnquotesYAMLScalars(t *testing.T) {
	for raw, want := range map[string]string{
		`"Fix: cart total rounding"`:           "Fix: cart total rounding",
		`'It''s a ''quoted'' title'`:           "It's a 'quoted' title",
		`"Escaped \"inner\" quotes"`:           `Escaped "inner" quotes`,
		`Plain title with "inner" quotes`:      `Plain title with "inner" quotes`,
		`"unterminated`:                        `"unterminated`,
		`"Ten writers omit Repo, so '' stays"`: "Ten writers omit Repo, so '' stays",
	} {
		s := parseInit(t, "---\ntitle: "+raw+"\ntype: bug\nstatus: planning\nslug: x\n---\n# x\n")
		if s.Title != want {
			t.Errorf("title %s parsed as %q, want %q", raw, s.Title, want)
		}
	}
}
