//go:build unix

package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/knowledge"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// Per-site cobra-stream tests for the sites migrated by
// `cli-prompt-package-adoption` (B2): skill.go's param gate, promptParam and
// the `skill save` form; handoff.go's promptNextStatus; export.go's conflict
// prompt and its terminal gate.
//
// Every one of these sites is a *preservation* claim — "behaves exactly as it
// did before the migration" — which is the class of claim that rots silently,
// because nothing fails when it stops being true. B1's cold audit found
// exactly one of these unguarded and mutated it without turning the suite red.
// So these tests assert on the two things a mutation cannot fake: the bytes
// written to the command's own output stream, and the state left on disk.
//
// Each site gets the three cases the spec requires:
//
//   - fully-supplied — the value arrives without a prompt, and no prompt is
//     emitted;
//   - non-TTY missing — the value is absent and the stream is not a terminal,
//     so the site takes its existing non-interactive path and never blocks;
//   - TTY missing — the value is absent and the stream IS a terminal, so the
//     site prompts and consumes the answer.
//
// The TTY cases run against a real pseudo-terminal rather than a fake, because
// the predicate under test is term.IsTerminal on an actual file descriptor.
// They are the ones that would fail if any site went back to reading os.Stdin
// instead of cmd.InOrStdin(): a command driven through cobra's streams would
// no longer see the terminal it was handed.

// newPTYStreamCmd returns a cobra command whose INPUT is a real terminal and
// whose output lands in a buffer, plus that buffer. answers is typed into the
// terminal from the master side.
func newPTYStreamCmd(t *testing.T, answers string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	master, slave, _ := openCapturedPTY(t)
	if answers != "" {
		go func() { _, _ = master.WriteString(answers) }()
	}
	cmd := &cobra.Command{}
	cmd.SetIn(slave)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	return cmd, out
}

// openCapturedPTY allocates a terminal and continuously drains its master end,
// returning an accessor for everything written to it so far.
//
// The drain is not optional. A pty's buffer is small and finite: a command
// that renders to the slave with nobody reading the master blocks mid-write
// once it fills. That is a property of the harness, not of the code under
// test — it showed up here as a hang inside the export summary printer, long
// after the prompt under test had already done its job.
func openCapturedPTY(t *testing.T) (master, slave *os.File, captured func() string) {
	t.Helper()
	m, s := openStreamPTY(t)

	var mu sync.Mutex
	var buf bytes.Buffer
	go func() {
		chunk := make([]byte, 4096)
		for {
			n, err := m.Read(chunk)
			if n > 0 {
				mu.Lock()
				buf.Write(chunk[:n])
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	return m, s, func() string {
		mu.Lock()
		defer mu.Unlock()
		return buf.String()
	}
}

// waitForOutput polls captured output for want, because the drain goroutine
// reads asynchronously — the bytes are on the wire when the command returns,
// but not necessarily in the buffer yet.
func waitForOutput(t *testing.T, captured func() string, want string) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(captured(), want) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// --------------------------------------------------------------------------
// skill.go:173 gate + skill.go:320 promptParam — `hero skill run`
// --------------------------------------------------------------------------

// writeParamSkill installs the same one-parameter skill the baseline fixture
// uses, so this test and testdata/prompt_baseline/skill_run_param_prompt.*
// exercise the same site.
func writeParamSkill(t *testing.T, env *testEnv) {
	t.Helper()
	body := "---\ntitle: Adoption Skill\nversion: 1\n---\n\n" +
		"# Adoption Skill\n\n## Parameters\n\n" +
		"- `target` — the thing to act on\n\n" +
		"## Steps\n\n1. Prompt agent: do something with {{target}}\n"
	dir := filepath.Join(env.heroDir, "skills")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "adoption-skill.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

// withSkillRunParams sets the --param flag's backing variable and restores it,
// so cases cannot leak into each other.
func withSkillRunParams(t *testing.T, params []string) {
	t.Helper()
	old := skillRunParams
	skillRunParams = params
	t.Cleanup(func() { skillRunParams = old })
}

// TestSkillRunParamFullySuppliedDoesNotPrompt is the fully-flagged case: with
// --param supplied there is no missing value, so the gate is never even
// consulted and nothing is written to the user.
func TestSkillRunParamFullySuppliedDoesNotPrompt(t *testing.T) {
	env := newTestEnv(t)
	writeParamSkill(t, env)
	withSkillRunParams(t, []string{"target=abc"})

	// A terminal on the input stream, to make the point sharply: even with a
	// terminal available, a supplied value must not produce a prompt.
	cmd, out := newPTYStreamCmd(t, "")
	if err := runSkillRun(cmd, []string{"adoption-skill"}); err != nil {
		t.Fatalf("runSkillRun with --param: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("a fully-supplied invocation wrote %q to the output stream, want nothing — "+
			"prompting is strictly additive", out.String())
	}
}

// TestSkillRunParamNonTTYRefusesAndIgnoresTheStream is the non-TTY case.
//
// The stream deliberately CARRIES a usable answer. Erroring out is not enough:
// the site must not read it. A gate that fell through to the prompt would
// happily consume "abc" here and succeed, which is the silent-default failure
// mode this initiative exists to remove.
func TestSkillRunParamNonTTYRefusesAndIgnoresTheStream(t *testing.T) {
	env := newTestEnv(t)
	writeParamSkill(t, env)
	withSkillRunParams(t, nil)

	cmd, out := newStreamCmd("abc\n")
	err := runSkillRun(cmd, []string{"adoption-skill"})
	if err == nil {
		t.Fatal("runSkillRun succeeded on a non-terminal stream — it consumed the piped answer")
	}
	const want = `missing required parameter "target" (use --param target=<value>)`
	if err.Error() != want {
		t.Errorf("error = %q, want %q (the pre-migration text, byte for byte)", err, want)
	}
	if out.Len() != 0 {
		t.Errorf("prompted into a non-terminal: %q", out.String())
	}
}

// TestSkillRunParamTTYPromptsThroughTheCommandStream is the TTY case, and the
// one that pins the wiring: the terminal is handed to the command via SetIn,
// not to os.Stdin. Reading the answer back proves the gate and the read both
// consult cmd.InOrStdin().
func TestSkillRunParamTTYPromptsThroughTheCommandStream(t *testing.T) {
	env := newTestEnv(t)
	writeParamSkill(t, env)
	withSkillRunParams(t, nil)

	cmd, out := newPTYStreamCmd(t, "from-the-terminal\n")
	if err := runSkillRun(cmd, []string{"adoption-skill"}); err != nil {
		t.Fatalf("runSkillRun with a terminal: %v", err)
	}
	const wantLabel = "  target (the thing to act on): "
	if got := out.String(); got != wantLabel {
		t.Errorf("prompt label = %q, want %q", got, wantLabel)
	}
}

// TestPromptParamLabelShapes pins both label forms. The described and
// undescribed shapes differ, and the baseline records the described one; a
// migration that dropped the parenthetical would still pass a "contains the
// name" assertion.
func TestPromptParamLabelShapes(t *testing.T) {
	tests := []struct {
		name        string
		param       string
		description string
		wantLabel   string
	}{
		{"with description", "target", "the thing to act on", "  target (the thing to act on): "},
		{"without description", "target", "", "  target: "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, out := newStreamCmd("answer\n")
			got, err := promptParam(cmd.InOrStdin(), cmd.OutOrStdout(), tc.param, tc.description)
			if err != nil {
				t.Fatalf("promptParam: %v", err)
			}
			if got != "answer" {
				t.Errorf("promptParam = %q, want %q", got, "answer")
			}
			if out.String() != tc.wantLabel {
				t.Errorf("label = %q, want %q", out.String(), tc.wantLabel)
			}
		})
	}
}

// --------------------------------------------------------------------------
// skill.go:208 — the `skill save` two-field form
// --------------------------------------------------------------------------

// savedSkillTitle returns the `title:` frontmatter value of a saved skill.
//
// Reading the field rather than searching the file for a substring is
// deliberate: the title is interpolated into the frontmatter AND the H1 AND
// the body, so "the file contains the title" is true for almost any bug.
func savedSkillTitle(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved skill %s: %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "title:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "title:"))
		}
	}
	t.Fatalf("saved skill %s has no title: frontmatter line:\n%s", path, data)
	return ""
}

// TestSkillSaveBothFieldsSuppliedWritesTheFile is the fully-supplied case.
// `skill save` has no flag form — the prompts ARE its interface — so "fully
// supplied" means both answers arrive on the stream.
//
// $EDITOR is pointed at `true` so openEditor returns immediately instead of
// launching vi against the test's terminal.
func TestSkillSaveBothFieldsSuppliedWritesTheFile(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("EDITOR", "true")

	cmd, out := newStreamCmd("adopted-skill\nDeliberately Not The Slug\n")
	if err := runSkillSave(cmd, nil); err != nil {
		t.Fatalf("runSkillSave: %v", err)
	}

	path := filepath.Join(env.heroDir, "skills", "adopted-skill.md")
	if got := savedSkillTitle(t, path); got != "Deliberately Not The Slug" {
		t.Errorf("saved title = %q, want %q — the second answer must land in the title field, "+
			"not a slug-derived default", got, "Deliberately Not The Slug")
	}
	// Both labels, in order, on the command's own output stream.
	const wantOut = "Skill name (slug, e.g. my-workflow): Skill title: "
	if out.String() != wantOut {
		t.Errorf("prompt output = %q, want %q", out.String(), wantOut)
	}
}

// TestSkillSaveNonTTYFailsFastAndWritesNothing is the non-TTY missing case.
//
// It pins two things the baseline fixture also records: the exact error, and
// that the name prompt is still printed. `skill save` has never had a TTY
// gate, and adding one here would change what a piped invocation prints —
// this child is allowed no such change.
func TestSkillSaveNonTTYFailsFastAndWritesNothing(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("EDITOR", "true")

	cmd, out := newStreamCmd("")
	err := runSkillSave(cmd, nil)
	if err == nil {
		t.Fatal("runSkillSave succeeded on an empty stream")
	}
	if err.Error() != "reading name: EOF" {
		t.Errorf("error = %q, want %q (byte-identical to the baseline fixture)", err, "reading name: EOF")
	}
	if out.String() != "Skill name (slug, e.g. my-workflow): " {
		t.Errorf("output = %q, want the name label and nothing more", out.String())
	}
	entries, err := os.ReadDir(filepath.Join(env.heroDir, "skills"))
	if err != nil {
		t.Fatalf("readdir skills: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("a failed `skill save` left %d file(s) behind", len(entries))
	}
}

// TestSkillSaveEmptyNameRejected pins the guard that sits between the two
// prompts. Without it, an operator who hits enter gets a file named ".md".
func TestSkillSaveEmptyNameRejected(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("EDITOR", "true")

	cmd, _ := newStreamCmd("\nA Title\n")
	err := runSkillSave(cmd, nil)
	if err == nil || err.Error() != "skill name cannot be empty" {
		t.Fatalf("error = %v, want %q", err, "skill name cannot be empty")
	}
	if _, statErr := os.Stat(filepath.Join(env.heroDir, "skills", ".md")); statErr == nil {
		t.Error("an empty name produced a `.md` file")
	}
}

// TestSkillSaveTTYReadsBothFieldsFromTheTerminal is the TTY case.
func TestSkillSaveTTYReadsBothFieldsFromTheTerminal(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("EDITOR", "true")

	cmd, _ := newPTYStreamCmd(t, "pty-skill\nTyped At A Terminal\n")
	if err := runSkillSave(cmd, nil); err != nil {
		t.Fatalf("runSkillSave with a terminal: %v", err)
	}
	path := filepath.Join(env.heroDir, "skills", "pty-skill.md")
	if got := savedSkillTitle(t, path); got != "Typed At A Terminal" {
		t.Errorf("saved title = %q, want %q", got, "Typed At A Terminal")
	}
}

// --------------------------------------------------------------------------
// end-of-stream parity for the reads that propagate their error
// --------------------------------------------------------------------------
//
// These pin the contract restored in prompt_line.go. The shared package treats
// trailing data with no final newline as a complete answer; every pre-package
// site that DISCARDED its read error is unaffected by that, but the four that
// act on it are not. Left unhandled, `printf 'myname\nMy Title' | hero skill
// save` flipped from a non-zero exit that wrote nothing into a zero exit that
// wrote a file — a failing invocation turning into a succeeding, disk-writing
// one, in a child whose defining constraint is zero behaviour change.
//
// Nothing in the golden fixtures covers unterminated input, which is exactly
// why it needed its own tests rather than a note.

// TestSkillSaveUnterminatedTitleFailsAndWritesNothing is the case with teeth:
// the name arrives cleanly and only the SECOND field is cut short, so the
// command is already past the point where it knows what file to write.
func TestSkillSaveUnterminatedTitleFailsAndWritesNothing(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("EDITOR", "true")

	// No trailing newline after the title — a pipe that ended mid-answer.
	cmd, _ := newStreamCmd("myname\nMy Title")
	err := runSkillSave(cmd, nil)
	if err == nil {
		t.Fatal("runSkillSave accepted an unterminated title — a stream that ends mid-answer " +
			"must not be promoted into a completed form")
	}
	if err.Error() != "reading title: EOF" {
		t.Errorf("error = %q, want %q", err, "reading title: EOF")
	}
	if _, statErr := os.Stat(filepath.Join(env.heroDir, "skills", "myname.md")); statErr == nil {
		t.Error("a failed `skill save` wrote myname.md — the invocation used to exit non-zero " +
			"with nothing on disk")
	}
}

// TestSkillSaveUnterminatedNameFailsAndWritesNothing is the same for the first
// field.
func TestSkillSaveUnterminatedNameFailsAndWritesNothing(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("EDITOR", "true")

	cmd, _ := newStreamCmd("myname")
	err := runSkillSave(cmd, nil)
	if err == nil {
		t.Fatal("runSkillSave accepted an unterminated name")
	}
	if err.Error() != "reading name: EOF" {
		t.Errorf("error = %q, want %q", err, "reading name: EOF")
	}
	entries, readErr := os.ReadDir(filepath.Join(env.heroDir, "skills"))
	if readErr != nil {
		t.Fatalf("readdir skills: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("a failed `skill save` left %d file(s) behind", len(entries))
	}
}

// TestPromptParamUnterminatedInputIsAnError covers the third propagating read.
// Its caller wraps this into `reading param %q`, so swallowing the short read
// would hand a truncated value to the skill runner as though it were typed.
func TestPromptParamUnterminatedInputIsAnError(t *testing.T) {
	cmd, _ := newStreamCmd("abc")
	got, err := promptParam(cmd.InOrStdin(), cmd.OutOrStdout(), "target", "")
	if err == nil {
		t.Fatalf("promptParam returned %q for an unterminated answer, want an error", got)
	}
	if !errors.Is(err, io.EOF) {
		t.Errorf("error = %v, want it to wrap io.EOF (the error bufio.ReadString returned)", err)
	}
	if got != "" {
		t.Errorf("promptParam returned %q alongside an error, want empty", got)
	}
}

// TestPromptLineTerminatedInputIsNotAnError is the positive control. A parity
// shim that reported EOF for everything would satisfy the three tests above
// and break every normal invocation; these prove it discriminates.
func TestPromptLineTerminatedInputIsNotAnError(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"newline terminated", "abc\n", "abc"},
		{"CRLF terminated", "abc\r\n", "abc"},
		{"second line still readable", "abc\ndef\n", "abc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, _ := newStreamCmd(tc.input)
			got, err := promptParam(cmd.InOrStdin(), cmd.OutOrStdout(), "target", "")
			if err != nil {
				t.Fatalf("promptParam(%q): %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("promptParam(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestPromptLineEOFDeliveredWithTheFinalBytes guards the subtlety in the shim.
//
// Some readers hand back their last bytes and io.EOF in the same call. bufio
// returned a nil error for those — it had already found the newline — so a
// shim keyed purely on "did we observe EOF" would fail a perfectly terminated
// line. eofWithFinalRead reproduces that reader.
func TestPromptLineEOFDeliveredWithTheFinalBytes(t *testing.T) {
	got, unterminated, err := promptLine(&eofWithFinalRead{data: []byte("abc\n")}, &bytes.Buffer{}, "")
	if err != nil {
		t.Fatalf("promptLine: %v", err)
	}
	if unterminated {
		t.Error("a newline-terminated line was reported as unterminated because EOF arrived " +
			"with the final byte — bufio returned a nil error here")
	}
	if got != "abc" {
		t.Errorf("promptLine = %q, want %q", got, "abc")
	}
}

// eofWithFinalRead returns io.EOF alongside the last byte rather than on a
// subsequent call, which io.Reader explicitly permits.
type eofWithFinalRead struct {
	data []byte
	pos  int
}

func (r *eofWithFinalRead) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		return n, io.EOF
	}
	return n, nil
}

// --------------------------------------------------------------------------
// handoff.go:326 — promptNextStatus
// --------------------------------------------------------------------------

// handoffMenu is the exact text promptNextStatus renders, including the bare
// "> " prompt.
//
// Comparing the whole block byte for byte is the point. `prompt.Choice` was
// the migration this site was nominally slated for, and Choice would have
// rendered "> " as "[1|delivering|d|2|in-review|review|r]: " instead. That is
// a visible change, and it is recorded verbatim in
// testdata/prompt_baseline/handoff_accept_next_status.*.txt. A "contains
// 'delivering'" assertion would not have noticed.
const handoffMenu = "Pick the next status for this spec:\n  1) delivering (default)\n  2) in-review\n> "

// TestPromptNextStatusAnswersFromTheCommandStream is the fully-supplied case,
// across every alias the site accepts.
func TestPromptNextStatusAnswersFromTheCommandStream(t *testing.T) {
	tests := []struct {
		answer string
		want   spec.Status
	}{
		{"1\n", spec.StatusDelivering},
		{"delivering\n", spec.StatusDelivering},
		{"d\n", spec.StatusDelivering},
		{"\n", spec.StatusDelivering},
		{"2\n", spec.StatusInReview},
		{"in-review\n", spec.StatusInReview},
		{"review\n", spec.StatusInReview},
		{"r\n", spec.StatusInReview},
		// No trailing newline: a scanner treated this as an answer, and so
		// must the migrated read.
		{"2", spec.StatusInReview},
	}
	for _, tc := range tests {
		t.Run(strings.TrimSpace(tc.answer)+"=>"+string(tc.want), func(t *testing.T) {
			cmd, out := newStreamCmd(tc.answer)
			got, err := promptNextStatus(cmd)
			if err != nil {
				t.Fatalf("promptNextStatus(%q): %v", tc.answer, err)
			}
			if got != tc.want {
				t.Errorf("answer %q => %q, want %q", tc.answer, got, tc.want)
			}
			if out.String() != handoffMenu {
				t.Errorf("menu = %q, want %q", out.String(), handoffMenu)
			}
		})
	}
}

// TestPromptNextStatusNonTTYTakesTheDeliveringDefault is the non-TTY missing
// case. End of stream is this site's documented non-interactive behaviour: it
// defaults, with no error and no hang.
func TestPromptNextStatusNonTTYTakesTheDeliveringDefault(t *testing.T) {
	cmd, out := newStreamCmd("")
	got, err := promptNextStatus(cmd)
	if err != nil {
		t.Fatalf("promptNextStatus on a closed stream: %v", err)
	}
	if got != spec.StatusDelivering {
		t.Errorf("closed stream => %q, want %q", got, spec.StatusDelivering)
	}
	if out.String() != handoffMenu {
		t.Errorf("menu = %q, want %q", out.String(), handoffMenu)
	}
}

// TestPromptNextStatusTTYReadsTheTerminalAnswer is the TTY missing case.
func TestPromptNextStatusTTYReadsTheTerminalAnswer(t *testing.T) {
	cmd, out := newPTYStreamCmd(t, "2\n")
	got, err := promptNextStatus(cmd)
	if err != nil {
		t.Fatalf("promptNextStatus with a terminal: %v", err)
	}
	if got != spec.StatusInReview {
		t.Errorf("terminal answer \"2\" => %q, want %q", got, spec.StatusInReview)
	}
	if out.String() != handoffMenu {
		t.Errorf("menu = %q, want %q", out.String(), handoffMenu)
	}
}

// TestPromptNextStatusRejectsUnknownAnswer pins the error text, which
// prompt.Choice would also have replaced.
func TestPromptNextStatusRejectsUnknownAnswer(t *testing.T) {
	cmd, _ := newStreamCmd("3\n")
	got, err := promptNextStatus(cmd)
	if err == nil {
		t.Fatalf("promptNextStatus accepted %q and returned %q", "3", got)
	}
	if err.Error() != `unrecognized choice "3" (1 or 2)` {
		t.Errorf("error = %q, want %q", err, `unrecognized choice "3" (1 or 2)`)
	}
	if got != "" {
		t.Errorf("returned %q alongside an error, want empty", got)
	}
}

// --------------------------------------------------------------------------
// export.go:121 promptConflictStrategy + the terminal gate
// --------------------------------------------------------------------------

// setupExportConflictEnv writes a knowledge note and a destination copy with
// DIFFERENT content, which is what makes the export a conflict.
func setupExportConflictEnv(t *testing.T, env *testEnv) (srcPath, dstPath string) {
	t.Helper()
	srcPath = filepath.Join(env.heroDir, "knowledge", "notes", "adoption.md")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte("source content\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	dstDir := filepath.Join(env.dir, "dest", "notes")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}
	dstPath = filepath.Join(dstDir, "adoption.md")
	if err := os.WriteFile(dstPath, []byte("destination content\n"), 0o644); err != nil {
		t.Fatalf("write destination: %v", err)
	}
	return srcPath, dstPath
}

func withExportConflict(t *testing.T, strategy string) {
	t.Helper()
	old := exportConflict
	exportConflict = strategy
	t.Cleanup(func() { exportConflict = old })
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// TestExportConflictFullyFlaggedDoesNotPrompt is the fully-flagged case:
// --conflict skip resolves every conflict without asking.
func TestExportConflictFullyFlaggedDoesNotPrompt(t *testing.T) {
	env := newTestEnv(t)
	_, dst := setupExportConflictEnv(t, env)
	withExportConflict(t, "skip")

	cmd, out := newStreamCmd("overwrite\n")
	if err := runExportKnowledge(cmd, []string{"dest"}); err != nil {
		t.Fatalf("runExportKnowledge --conflict skip: %v", err)
	}
	if strings.Contains(out.String(), "Conflict: ") {
		t.Errorf("a flag-resolved conflict still prompted: %q", out.String())
	}
	if got := readFileString(t, dst); got != "destination content\n" {
		t.Errorf("destination = %q, want it untouched — `skip` must not consult the stream, "+
			"which offered `overwrite`", got)
	}
}

// TestExportConflictNonTTYRefusesBeforePrompting is the non-TTY case. The gate
// needs BOTH streams to be terminals; here neither is.
func TestExportConflictNonTTYRefusesBeforePrompting(t *testing.T) {
	env := newTestEnv(t)
	_, dst := setupExportConflictEnv(t, env)
	withExportConflict(t, "interactive")

	cmd, out := newStreamCmd("overwrite\n")
	err := runExportKnowledge(cmd, []string{"dest"})
	if err == nil {
		t.Fatal("interactive export succeeded with no terminal — it read the piped answer")
	}
	if err.Error() != "--conflict interactive requires an attached terminal" {
		t.Errorf("error = %q, want the pre-migration text", err)
	}
	if out.Len() != 0 {
		t.Errorf("wrote %q before refusing", out.String())
	}
	if got := readFileString(t, dst); got != "destination content\n" {
		t.Errorf("destination = %q, want it untouched", got)
	}
}

// TestExportConflictTTYResolvesFromTheTerminal is the TTY case.
//
// It asserts the resolved file content rather than the prompt text. Reaching
// the prompt proves the gate opened; the destination changing to the source's
// bytes proves the typed answer was actually read and applied. Prompt text
// alone would not distinguish "asked and read the answer" from "asked and fell
// through to a default".
func TestExportConflictTTYResolvesFromTheTerminal(t *testing.T) {
	tests := []struct {
		name    string
		answer  string
		wantDst string
	}{
		{"overwrite applies the source", "overwrite\n", "source content\n"},
		{"skip keeps the destination", "skip\n", "destination content\n"},
		// An invalid answer re-asks rather than failing — the loop this site
		// has always had, and one prompt.Choice would have replaced with an
		// error.
		{"invalid answer re-asks", "nonsense\noverwrite\n", "source content\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			_, dst := setupExportConflictEnv(t, env)
			withExportConflict(t, "interactive")

			master, slave, captured := openCapturedPTY(t)
			go func() { _, _ = master.WriteString(tc.answer) }()
			cmd := &cobra.Command{}
			// Both streams are the terminal: the gate requires an input
			// terminal to read from AND an output terminal to render on.
			cmd.SetIn(slave)
			cmd.SetOut(slave)

			if err := runExportKnowledge(cmd, []string{"dest"}); err != nil {
				t.Fatalf("runExportKnowledge: %v", err)
			}
			if got := readFileString(t, dst); got != tc.wantDst {
				t.Errorf("destination = %q, want %q", got, tc.wantDst)
			}
			if !waitForOutput(t, captured, "Choose [fail/skip/overwrite/merge]: ") {
				t.Errorf("the conflict prompt was never rendered to the terminal; saw:\n%s", captured())
			}
		})
	}
}

// TestExportConflictGateNeedsBothStreams pins the two-predicate shape of the
// gate. A terminal on input alone is not enough, and neither is one on output:
// the helper this replaced asked one question twice, and collapsing the two
// predicates back into one would go unnoticed without this.
func TestExportConflictGateNeedsBothStreams(t *testing.T) {
	tests := []struct {
		name    string
		ttyIn   bool
		ttyOut  bool
		wantErr bool
	}{
		{"neither stream is a terminal", false, false, true},
		{"only input is a terminal", true, false, true},
		{"only output is a terminal", false, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			_, dst := setupExportConflictEnv(t, env)
			withExportConflict(t, "interactive")

			master, slave, _ := openCapturedPTY(t)
			cmd := &cobra.Command{}
			if tc.ttyIn {
				// A usable answer waits on the terminal. If the gate wrongly
				// opens, the export resolves the conflict and the
				// destination-untouched assertion below fails immediately —
				// rather than the test hanging on an empty terminal, which is
				// a much worse way to learn the same thing.
				go func() { _, _ = master.WriteString("overwrite\n") }()
				cmd.SetIn(slave)
			} else {
				cmd.SetIn(strings.NewReader("overwrite\n"))
			}
			if tc.ttyOut {
				cmd.SetOut(slave)
			} else {
				cmd.SetOut(&bytes.Buffer{})
			}

			err := runExportKnowledge(cmd, []string{"dest"})
			if tc.wantErr && err == nil {
				t.Fatalf("gate opened with ttyIn=%v ttyOut=%v", tc.ttyIn, tc.ttyOut)
			}
			if got := readFileString(t, dst); got != "destination content\n" {
				t.Errorf("destination = %q, want it untouched", got)
			}
		})
	}
}

// TestExportConflictAnswersAreCaseFolded pins the case-insensitivity of the
// conflict answer.
//
// `strings.ToLower` has been in this read since before the migration, so
// `OVERWRITE` and `Skip` have always worked — but nothing covered it, and
// replacing the fold with a bare trim passed the entire suite. An operator who
// types `Overwrite` at a prompt that lists lowercase options is not making a
// mistake, and silently re-asking them is a regression nobody would attribute
// to a refactor.
//
// The closure is driven directly: the TTY gate lives in the caller, so no
// terminal is needed to exercise the read itself.
func TestExportConflictAnswersAreCaseFolded(t *testing.T) {
	tests := []struct {
		answer string
		want   knowledge.ConflictStrategy
	}{
		{"OVERWRITE\n", knowledge.ConflictOverwrite},
		{"Skip\n", knowledge.ConflictSkip},
		{"MeRgE\n", knowledge.ConflictMerge},
		{"FAIL\n", knowledge.ConflictFail},
		{"overwrite\n", knowledge.ConflictOverwrite},
		// Surrounding whitespace was trimmed before the fold, and still is.
		{"  Overwrite  \n", knowledge.ConflictOverwrite},
	}
	for _, tc := range tests {
		t.Run(strings.TrimSpace(tc.answer), func(t *testing.T) {
			var out bytes.Buffer
			ask := promptConflictStrategy(strings.NewReader(tc.answer), &out)
			got, err := ask(knowledge.Conflict{RelPath: "a.md", Reason: "differs"})
			if err != nil {
				t.Fatalf("answer %q: %v", tc.answer, err)
			}
			if got != tc.want {
				t.Errorf("answer %q => %q, want %q", tc.answer, got, tc.want)
			}
			if strings.Contains(out.String(), "Invalid choice") {
				t.Errorf("answer %q was rejected before being accepted: %q", tc.answer, out.String())
			}
		})
	}
}

// TestExportConflictUnterminatedInvalidAnswerStopsAtOnce is the export half of
// the end-of-stream parity fix.
//
// An invalid final answer with no trailing newline used to print one rejection
// and return io.EOF. Treating the short read as an ordinary answer made the
// loop come round once more and emit a second conflict block plus an
// `Invalid choice ""` — output no one asked for, and one that reads like a
// second, phantom conflict.
func TestExportConflictUnterminatedInvalidAnswerStopsAtOnce(t *testing.T) {
	var out bytes.Buffer
	ask := promptConflictStrategy(strings.NewReader("nonsense"), &out)
	got, err := ask(knowledge.Conflict{RelPath: "a.md", Reason: "differs"})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("error = %v (result %q), want io.EOF", err, got)
	}
	if n := strings.Count(out.String(), "Conflict: "); n != 1 {
		t.Errorf("rendered the conflict %d times, want exactly 1:\n%s", n, out.String())
	}
	if n := strings.Count(out.String(), "Invalid choice"); n != 1 {
		t.Errorf("rejected %d answers, want exactly 1 — the user only gave one:\n%s", n, out.String())
	}
	if strings.Contains(out.String(), `Invalid choice ""`) {
		t.Errorf("reported an empty answer the user never gave:\n%s", out.String())
	}
}

// TestExportConflictUnterminatedValidAnswerIsAccepted is the positive control:
// a VALID final answer with no trailing newline was accepted before the
// migration and must still be. This is the case that stops the parity fix from
// being "treat every short read as a failure".
func TestExportConflictUnterminatedValidAnswerIsAccepted(t *testing.T) {
	var out bytes.Buffer
	ask := promptConflictStrategy(strings.NewReader("overwrite"), &out)
	got, err := ask(knowledge.Conflict{RelPath: "a.md", Reason: "differs"})
	if err != nil {
		t.Fatalf("unterminated valid answer: %v", err)
	}
	if got != knowledge.ConflictOverwrite {
		t.Errorf("got %q, want %q", got, knowledge.ConflictOverwrite)
	}
}

// --------------------------------------------------------------------------
// brief.go:837 — audited, kept, and pinned as an OUTPUT-stream predicate
// --------------------------------------------------------------------------

// TestBriefInteractivityFollowsOutputNotInput is the behavioural half of AC-8.
//
// The structural half (TestBriefKeepsDistinctOutputTerminalCheck in
// prompt_policy_test.go) reads the source. This drives the predicate with the
// two streams deliberately CROSSED, which is the only arrangement that can
// tell the two questions apart: if isInteractive() were ever folded into the
// input predicate, both assertions below invert.
func TestBriefInteractivityFollowsOutputNotInput(t *testing.T) {
	_, slave := openStreamPTY(t)
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { pipeR.Close(); pipeW.Close() })

	// Output piped, input a terminal: NOT interactive. `hero brief | less`
	// must not render as though it owned a terminal just because someone is
	// sitting at one.
	if isInteractive(pipeW) {
		t.Error("brief reported interactive with piped stdout — it is reading the input stream")
	}

	// Output a terminal, input piped: interactive. Rendering follows stdout.
	if !isInteractive(slave) {
		t.Error("brief reported non-interactive with stdout on a terminal — it is reading the input stream")
	}
}

// --------------------------------------------------------------------------
// new.go:433 — audited and deliberately left alone
// --------------------------------------------------------------------------

// TestNewInteractiveRemainsOptInAndInjectable records the audit verdict as an
// executable claim.
//
// new.go was audited against this initiative's constraints and found already
// compliant: it prompts only behind an explicit --interactive flag, it reads
// from an injectable package variable rather than a hardcoded os.Stdin, and it
// terminates at end of stream instead of blocking. It is therefore NOT
// migrated — a migration here would be churn against a working, tested path
// and would risk a diff against the baseline for no gain.
//
// This test exists so "we chose not to touch it" is a decision the suite
// holds, rather than an absence someone later reads as an oversight.
func TestNewInteractiveRemainsOptInAndUsesCobraInput(t *testing.T) {
	src, err := os.ReadFile("new.go")
	if err != nil {
		t.Fatalf("read new.go: %v", err)
	}
	body := stripComments(string(src))

	if !strings.Contains(body, "if newInteractive {") {
		t.Error("new.go no longer gates its prompts on the --interactive flag; " +
			"prompting must stay opt-in there")
	}
	if !strings.Contains(body, "cmd.InOrStdin()") {
		t.Error("new.go no longer reads through Cobra's configured input stream")
	}
	if strings.Contains(body, "newStdin") {
		t.Error("new.go retains the replaced package-level input variable")
	}
	if promptCallPattern.MatchString(body) {
		t.Error("new.go now calls the shared prompt package. That may well be right one day, " +
			"but it is a behaviour change against the recorded baseline and belongs to a spec " +
			"that says so — cli-prompt-package-adoption explicitly does not.")
	}
}

// --------------------------------------------------------------------------
// the pair completion metric
// --------------------------------------------------------------------------

// TestNoLegacyStdinReadsRemain asserts the initiative's pair completion metric
// as a check, so it cannot silently regress: a recursive grep over
// internal/cli/ for a bufio Reader or Scanner constructed over the process's
// standard input must return zero hits.
//
// The four forked TTY predicates and every direct standard-input read are
// gone; a contributor who reintroduces one has bypassed the shared package and
// every guarantee that rests on it.
//
// The two search patterns are assembled from fragments below, and this comment
// avoids spelling either of them out, so the guard cannot match itself. That
// is not cosmetic: a self-matching guard fails on arrival and gets deleted or
// weakened by whoever hits it next.
func TestNoLegacyStdinReadsRemain(t *testing.T) {
	patterns := []string{
		"bufio.New" + "Reader(os.Stdin)",
		"bufio.New" + "Scanner(os.Stdin)",
	}

	root := "."
	found := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(data)
		for _, p := range patterns {
			for _, line := range strings.Split(text, "\n") {
				if strings.Contains(line, p) {
					found++
					t.Errorf("%s reads interactive input via %s — "+
						"read through internal/cli/prompt with cmd.InOrStdin() instead:\n\t%s",
						path, p, strings.TrimSpace(line))
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	// Guard the guard: if the walk stops finding files, it would report zero
	// hits forever regardless of the source.
	if _, statErr := os.Stat("skill.go"); statErr != nil {
		t.Fatalf("the completion-metric walk is not covering internal/cli: %v", statErr)
	}
	if found == 0 {
		t.Log("pair completion metric holds: zero legacy os.Stdin reads under internal/cli/")
	}
}
