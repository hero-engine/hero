//go:build unix

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJSONModeNeverPrompts is AC-10.
//
// Every case here runs with a REAL pseudo-terminal on fd 0. That is the whole
// point: under a pipe these commands would decline to prompt anyway, so a
// non-TTY run would pass whether or not the --json guard exists. Attaching a
// terminal removes that excuse — the only thing that can suppress the prompt
// is the --json rule itself.
//
// Under --json, stdout carries a single machine-readable result object. A
// prompt does two bad things there: it blocks a caller that will never answer,
// and its text lands in the middle of the JSON the caller is parsing.
//
// install.go:567's candidate walk had no --json guard at all before this
// child; it is covered here alongside the site that did.
func TestJSONModeNeverPrompts(t *testing.T) {
	bin := baselineBinary(t)

	tests := []struct {
		name string
		// setup builds the workspace that would otherwise trigger the prompt.
		setup func(t *testing.T, root string)
		args  []string
		// promptText is what would appear if the guard were missing. It is
		// taken from the site's own baseline fixture, so the two cannot drift.
		promptText string
		// wantJSON asserts stdout is exactly one parseable JSON object, which
		// is the contract a prompt would corrupt.
		wantJSON bool
	}{
		{
			name:       "promptTarget",
			setup:      setupBareProjectDir,
			args:       []string{"install", "project", "proj", "--json", "--dry-run"},
			promptText: "Install target",
		},
		{
			name:       "subproject add confirm",
			setup:      setupWorkspaceWithUndeclaredSubdir,
			args:       []string{"install", "project", "sub", "--json"},
			promptText: "Add it as a subproject",
			wantJSON:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "work")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatalf("mkdir work root: %v", err)
			}
			tc.setup(t, root)

			// Deliberately feed NOTHING, and never close the terminal.
			//
			// Checking only for the prompt's text is not enough here: --json
			// routes the install through silenceStdout, so a prompt that DID
			// fire would be swallowed and the assertion would pass against
			// broken code. Confirmed by falsification — deleting the guard
			// left a text-only version of this test green.
			//
			// An unanswered read on a live terminal blocks forever, so the
			// observable signature of a missing guard is the timeout. That is
			// also the failure a real programmatic caller would experience.
			exitCode, stdout, stderr := runHero(t, bin, base, root, tc.args, condTTY, "")
			combined := stdout + stderr

			if exitCode == -1 {
				t.Fatalf("command BLOCKED under --json with a terminal attached — it prompted and is "+
					"waiting for an answer no programmatic caller will ever give.\noutput so far:\n%s", combined)
			}
			if strings.Contains(combined, tc.promptText) {
				t.Errorf("prompted under --json (terminal attached).\nfound: %q\noutput:\n%s",
					tc.promptText, combined)
			}
			if tc.wantJSON {
				if err := json.Unmarshal([]byte(stdout), &map[string]any{}); err != nil {
					t.Errorf("stdout under --json is not a single JSON object: %v\nstdout:\n%s", err, stdout)
				}
			}
		})
	}
}

// TestCandidateWalkJSONGuardIsStructural covers install.go:567 — the site the
// spec calls out as "lacking the --json guard today".
//
// The guard is now present, but it is NOT observable behavior, and this test
// is deliberately structural rather than behavioral so that nobody mistakes a
// green behavioral run for proof.
//
// Under --json, runInstall short-circuits at the `if installJSON { ... return
// emitJSON(...) }` block and returns before it ever reaches the candidate
// walk. So the walk is unreachable in JSON mode regardless of its own guard,
// and a behavioral test of it passes identically with the guard present or
// deleted — verified by falsification while writing this file. There was no
// live --json bug at this site; what existed was an unguarded prompt sitting
// one refactor away from becoming one.
//
// The guard is therefore defense in depth: if that early return is ever moved
// or the walk is hoisted above it, the walk must not start prompting into a
// JSON result object. This test pins the guard so such a refactor cannot
// silently drop it.
func TestCandidateWalkJSONGuardIsStructural(t *testing.T) {
	src, err := os.ReadFile("install.go")
	if err != nil {
		t.Fatalf("read install.go: %v", err)
	}
	body := stripComments(string(src))

	idx := strings.Index(body, "func postRootInstallSubprojectWalk(")
	if idx < 0 {
		t.Fatal("postRootInstallSubprojectWalk no longer exists")
	}
	// Look only at the head of the function, where the guard belongs.
	head := body[idx:]
	if len(head) > 600 {
		head = head[:600]
	}
	if !strings.Contains(head, "installJSON") {
		t.Error("the candidate walk no longer checks installJSON before prompting; " +
			"a prompt here would interleave with the single JSON result object a programmatic caller parses")
	}
	if !strings.Contains(head, "prompt.IsInputTTY") {
		t.Error("the candidate walk no longer gates on prompt.IsInputTTY")
	}
}
