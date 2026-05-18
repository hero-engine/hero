package cli

import (
	"reflect"
	"testing"
)

// markdown_invocations_test.go — table-driven unit coverage for the
// extractor and validator. The companion markdown_drift_test.go is the
// CI gate that walks the real repo surfaces.

func TestExtractInvocationsExtractionRules(t *testing.T) {
	type want struct {
		line int
		args []string
	}
	cases := []struct {
		name    string
		content string
		want    []want
	}{
		{
			name:    "inline backtick simple",
			content: "Run `hero status` to see active specs.\n",
			want:    []want{{line: 1, args: []string{"status"}}},
		},
		{
			name: "fenced code block",
			content: "```\n" +
				"hero install project . --target opencode\n" +
				"```\n",
			want: []want{{line: 2, args: []string{"install", "project", ".", "--target", "opencode"}}},
		},
		{
			name:    "prose mention not extracted",
			content: "The hero framework is great. Hero is a tool.\n",
			want:    nil,
		},
		{
			name:    "uppercase mention not extracted",
			content: "Hero status is a phrase that should not match.\n",
			want:    nil,
		},
		{
			name:    "trailing punctuation stops args",
			content: "Use `hero status, but check first` carefully.\n",
			want:    []want{{line: 1, args: []string{"status"}}},
		},
		{
			name:    "flag with equals",
			content: "Run `hero peer call alias --mode=advisory`.\n",
			want:    []want{{line: 1, args: []string{"peer", "call", "alias", "--mode=advisory"}}},
		},
		{
			name:    "html comment strips invocation",
			content: "<!-- hero pull should not extract -->\nReal: `hero status`.\n",
			want:    []want{{line: 2, args: []string{"status"}}},
		},
		{
			name: "multi-line html comment strips invocation but keeps line numbers",
			content: "Line 1\n" +
				"<!-- this comment spans\n" +
				"hero pull is inside\n" +
				"-->\n" +
				"Line 5: `hero status`.\n",
			want: []want{{line: 5, args: []string{"status"}}},
		},
		{
			name:    "drift-test:ignore on same line suppresses",
			content: "`hero pull` <!-- drift-test:ignore --> deliberately broken.\n",
			want:    nil,
		},
		{
			name: "drift-test:ignore on preceding line suppresses next line",
			content: "<!-- drift-test:ignore -->\n" +
				"`hero pull` should be skipped.\n" +
				"`hero status` should match.\n",
			want: []want{{line: 3, args: []string{"status"}}},
		},
		{
			name:    "table cell with backticks",
			content: "| Health | `hero check` |\n",
			want:    []want{{line: 1, args: []string{"check"}}},
		},
		{
			name:    "multiple invocations on one line",
			content: "Both `hero status` and `hero check` run.\n",
			want: []want{
				{line: 1, args: []string{"status"}},
				{line: 1, args: []string{"check"}},
			},
		},
		{
			name:    "subcommand with hyphen",
			content: "Try `hero spec-types list`.\n",
			want:    []want{{line: 1, args: []string{"spec-types", "list"}}},
		},
		{
			name:    "placeholder angle brackets in args kept",
			content: "Run `hero handoff <spec> <alias>`.\n",
			want: []want{
				{line: 1, args: []string{"handoff", "<spec>", "<alias>"}},
			},
		},
		{
			name:    "first token uppercase is rejected",
			content: "Try `hero STATUS` — uppercase first token is prose-like.\n",
			want:    nil,
		},
		{
			name:    "no args bare invocation",
			content: "Just `hero` is not a command.\n",
			want:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractInvocations("test.md", []byte(tc.content))
			if len(got) != len(tc.want) {
				t.Fatalf("got %d invocations, want %d: %+v", len(got), len(tc.want), got)
			}
			for i, w := range tc.want {
				if got[i].Line != w.line {
					t.Errorf("[%d] line: got %d, want %d", i, got[i].Line, w.line)
				}
				if !reflect.DeepEqual(got[i].Args, w.args) {
					t.Errorf("[%d] args: got %v, want %v", i, got[i].Args, w.args)
				}
			}
		})
	}
}

func TestValidateInvocationResolvesRealCommands(t *testing.T) {
	// Use the real rootCmd from the package — these must be real commands.
	cases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "status", args: []string{"status"}, wantErr: false},
		{name: "check", args: []string{"check"}, wantErr: false},
		{name: "docs check", args: []string{"docs", "check"}, wantErr: false},
		{name: "unknown root", args: []string{"definitely-not-a-real-command"}, wantErr: true},
		{name: "unknown flag on real command", args: []string{"status", "--definitely-not-a-flag"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInvocation(rootCmd, Invocation{Args: tc.args, Raw: "hero " + joinArgs(tc.args)})
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestIsExcludedDirCoversBugAndPlanningSurfaces(t *testing.T) {
	yes := []string{
		".hero/specs/foo/spec.md",
		".hero/planning/bugs/x/spec.md",
		"./.hero/planning/features/y/spec.md",
	}
	no := []string{
		"commands/design.md",
		"skills/spec-format/spec.md",
		".hero/knowledge/notes/foo/spec.md",
		"README.md",
	}
	for _, p := range yes {
		if !isExcludedDir(p) {
			t.Errorf("expected excluded: %s", p)
		}
	}
	for _, p := range no {
		if isExcludedDir(p) {
			t.Errorf("expected not excluded: %s", p)
		}
	}
}

func joinArgs(a []string) string {
	out := ""
	for i, s := range a {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}
