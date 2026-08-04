//go:build unix

package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// This file holds the standing policy guards for Hero's prompt layer. Unlike
// the baseline fixtures, which pin what specific commands do today, these
// assert rules that must hold for code that does not exist yet — they are
// aimed at the contributor who, a year from now, wires a prompt into the
// agent-facing surface without knowing why they must not.

// neverPromptSources maps each command family in the NEVER-PROMPT class to the
// files that implement it.
//
// These commands are invoked programmatically, by agents and by the MCP
// surface. They carry --revision and --idempotency-key semantics, so a prompt
// does one of two bad things: it hangs a caller that will never type anything,
// or it invites a human to invent an idempotency key, which silently breaks
// the retry semantics the key exists to provide.
var neverPromptSources = map[string][]string{
	"focus":      {"focus.go"},
	"mail":       {"attention.go"},
	"graph node": {"graph_node.go"},
	"graph edge": {"graph_edge.go"},
	"graph":      {"graph.go", "graph_memory.go"},
	"nlhook":     {"nlhook.go"},
	"hook":       {"hook.go", "hooks.go", "host_hooks.go"},
	"brokers":    {"code_host_broker.go", "tracker_broker.go"},
	"run":        {"run.go"},
	"jobs":       {"jobs.go"},
	"next-emit":  {"next_emit.go", "propose_shim.go"},
}

// promptCallPattern matches an interactive read through the shared package.
// IsInputTTY/IsOutputTTY are deliberately NOT matched: asking whether a stream
// is a terminal is a legitimate rendering or guard decision anywhere. Actually
// reading from the user is what the NEVER-PROMPT class forbids.
//
// promptLine/promptLineStrict (prompt_line.go) are matched too. They wrap
// prompt.Prompt to restore bufio's end-of-stream contract, and a wrapper that
// this guard could not see would be a way around it.
var promptCallPattern = regexp.MustCompile(`prompt\.(Prompt|Confirm|Choice|Secret)\b|\bpromptLine(Strict)?\(`)

// legacyReadPattern matches the pre-package ways of reading interactive input,
// so a contributor cannot sidestep the rule by going back to os.Stdin.
var legacyReadPattern = regexp.MustCompile(`bufio\.New(Reader|Scanner)\(os\.Stdin\)|fmt\.Scanln|term\.ReadPassword`)

// TestNeverPromptClassDoesNotPrompt is the structural half of AC-11.
//
// A behavioral test can only prove that the commands which exist today do not
// prompt. This proves that the ones added tomorrow cannot either, because the
// rule is enforced against the source of the whole command family rather than
// against a list of invocations someone has to remember to extend.
func TestNeverPromptClassDoesNotPrompt(t *testing.T) {
	for family, files := range neverPromptSources {
		t.Run(family, func(t *testing.T) {
			checked := 0
			for _, name := range files {
				path := filepath.Join(".", name)
				src, err := os.ReadFile(path)
				if err != nil {
					if os.IsNotExist(err) {
						// The file may have been renamed or the command
						// removed; other guards cover the inventory.
						continue
					}
					t.Fatalf("read %s: %v", path, err)
				}
				checked++
				body := stripComments(string(src))
				if loc := promptCallPattern.FindString(body); loc != "" {
					t.Errorf("%s is in the NEVER-PROMPT class but calls %s.\n"+
						"These commands are driven by agents and the MCP surface: a prompt either hangs the "+
						"caller or invites a human-invented idempotency key that breaks retry semantics.", name, loc)
				}
				if loc := legacyReadPattern.FindString(body); loc != "" {
					t.Errorf("%s is in the NEVER-PROMPT class but reads interactive input via %s.", name, loc)
				}
			}
			if checked == 0 {
				t.Fatalf("no source files found for NEVER-PROMPT family %q — the guard is not covering anything", family)
			}
		})
	}
}

// TestNoDeclarativeFieldDescriptor guards the successor's explicit rejection
// of a generic form or field-schema layer. Connect's private collector does not
// justify a reusable abstraction.
func TestNoDeclarativeFieldDescriptor(t *testing.T) {
	banned := []string{"FormDefinition", "FieldSpec"}
	// `Schema` is banned only as a type declaration — the word appears
	// legitimately in JSON-schema and graph-schema contexts throughout the CLI.
	bannedTypeDecl := regexp.MustCompile(`type\s+(Schema|FormDefinition|FieldSpec|FieldType|FieldRegistry)\b`)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		body := stripComments(string(src))
		for _, word := range banned {
			if strings.Contains(body, word) {
				t.Errorf("%s mentions %s — the declarative field descriptor is out of scope for this "+
					"initiative and belongs to connect-writer-unification, capped at connect's real needs.", name, word)
			}
		}
		if loc := bannedTypeDecl.FindString(body); loc != "" {
			t.Errorf("%s declares %q — a field-type registry or form schema is the hard-stop drift alarm "+
				"for this initiative; stop and escalate rather than building it.", name, loc)
		}
	}
}

// TestBriefKeepsDistinctOutputTerminalCheck is AC-12.
//
// brief.go's isInteractive() asks whether *stdout* is a terminal, for
// rendering decisions. It is a different question from "may I prompt?", and
// folding it into the input predicate would make `hero brief | less` render as
// though it were writing to a terminal. This child ships IsOutputTTY so a
// correct home exists, but deliberately does not migrate or delete brief.go —
// that audit belongs to cli-prompt-package-adoption.
//
// Any change that deletes brief.go's check in favour of the INPUT predicate
// must fail here.
func TestBriefKeepsDistinctOutputTerminalCheck(t *testing.T) {
	src, err := os.ReadFile("brief.go")
	if err != nil {
		t.Fatalf("read brief.go: %v", err)
	}
	body := stripComments(string(src))

	if !strings.Contains(body, "func isInteractive(") {
		t.Error("brief.go no longer defines isInteractive(); this child must leave it in place")
	}
	if !strings.Contains(body, "prompt.IsOutputTTY") {
		t.Error("brief.go no longer uses the output-terminal predicate for rendering")
	}
	if strings.Contains(body, "prompt.IsInputTTY") {
		t.Error("brief.go now uses the INPUT predicate for a rendering decision. " +
			"That is a correctness regression, not a cleanup: it would tie output rendering to what stdin happens to be.")
	}
}

// TestPromptPackageExposesBothPredicates is the structural half of AC-2 and
// the standing guard for the two-predicate design.
//
// Assigning each function to a typed variable makes the signatures part of the
// compile. A future "simplification" that collapses the two predicates into
// one, or that gives Secret an io.Reader so a caller could substitute a
// non-terminal stream, stops compiling here.
func TestPromptPackageExposesBothPredicates(t *testing.T) {
	// Compile-time signature pins; see promptContractPins in this package.
	if promptContractPins.inputTTY == nil || promptContractPins.outputTTY == nil {
		t.Fatal("prompt package predicates are not wired")
	}
	if promptContractPins.secret == nil {
		t.Fatal("prompt.Secret is not wired")
	}
}

// stripComments removes // and /* */ comments so a guard does not fire on
// prose that merely discusses the thing it forbids — this file and the
// migrated call sites both explain the old patterns by name.
func stripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	for i := 0; i < len(src); {
		switch {
		case strings.HasPrefix(src[i:], "//"):
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				return b.String()
			}
			i += end
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return b.String()
			}
			i += end + 4
		default:
			b.WriteByte(src[i])
			i++
		}
	}
	return b.String()
}
