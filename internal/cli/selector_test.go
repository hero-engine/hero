//go:build unix

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// Adoption tests for the SELECTOR-class commands: `hero spec score`,
// `hero spec verify`, `hero spec move`, `hero supersede`, `hero size`, the five
// name-taking `hero skill` subcommands, `hero handoff`, and
// `hero handoff accept`.
//
// Every command gets the five cases the spec's Validation section requires:
// supplied (identical, no picker), missing + non-TTY (existing error, non-zero,
// no hang), missing + TTY (a picker over the expected corpus), empty corpus +
// TTY (existing error, not an empty picker), and — where the flag exists —
// --json (no picker).
//
// They run through the shipped binary rather than in-process cobra, for the
// three reasons the baseline does: exit codes are part of the contract, several
// of these sites print with fmt.Print rather than to cmd.OutOrStdout(), and the
// predicate under test is term.IsTerminal on a real fd 0.
//
// Where the command changes state, that is what is asserted — the frontmatter
// supersede writes, the `subproject:` field `spec move` rewrites, the trail
// entry `handoff` appends, the file `skill rm` deletes. Where it does not
// (score, verify, size are all reads), the discriminator is that the PICKED
// candidate drove the command and the unpicked one did not: every fixture below
// offers at least two candidates and answers with the one that is not first, so
// a picker that rendered and then ignored its answer fails.

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

// pickerSpec is one spec written into a picker fixture.
type pickerSpec struct {
	slug   string
	status string
	size   string
	// modAge backdates the file's mtime. `hero list` ranks by recency, so
	// distinct ages give the corpus a deterministic order that is neither
	// alphabetical nor write order — which is what makes the ordering
	// assertion meaningful.
	modAge time.Duration
}

// writePickerSpecs writes each spec into the workspace's planning/features
// directory and backdates its mtime.
func writePickerSpecs(t *testing.T, root string, specs ...pickerSpec) {
	t.Helper()
	dir := filepath.Join(root, ".hero", "planning", "features")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir planning/features: %v", err)
	}
	now := time.Now()
	for _, s := range specs {
		status := s.status
		if status == "" {
			status = "planning"
		}
		body := "---\ntitle: " + s.slug + "\nslug: " + s.slug +
			"\ntype: feature\nstatus: " + status + "\ncreated: 2026-01-01\n"
		if s.size != "" {
			body += "size: " + s.size + "\n"
		}
		body += "---\n\n# " + s.slug + "\n\n## Context\n\nPicker fixture.\n"

		path := filepath.Join(dir, s.slug+".md")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write spec %s: %v", s.slug, err)
		}
		mod := now.Add(-s.modAge)
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatalf("chtimes %s: %v", s.slug, err)
		}
	}
}

// newPickerWorkspace builds a workspace holding the given specs.
func newPickerWorkspace(t *testing.T, specs ...pickerSpec) (base, root string) {
	t.Helper()
	base, root = newSanctionedWorkspace(t)
	writePickerSpecs(t, root, specs...)
	return base, root
}

// twoSpecs is the standard two-candidate corpus. `pick-recent` is newer, so it
// ranks FIRST; tests answer with `pick-older`, which cannot be reached by a
// picker that ignores its answer and takes the head of the list.
func twoSpecs() []pickerSpec {
	return []pickerSpec{
		{slug: "pick-recent", size: "small", modAge: time.Hour},
		{slug: "pick-older", size: "large", modAge: 48 * time.Hour},
	}
}

// pickerOptions extracts the option list prompt.Choice rendered for label.
//
// prompt.Choice writes `<label> [a|b|c]: `, so the options are recoverable
// verbatim. Returning them as a slice lets the ordering assertion compare
// against `hero list` rather than against a hardcoded expectation.
func pickerOptions(t *testing.T, combined, label string) []string {
	t.Helper()
	head := label + " ["
	i := strings.Index(combined, head)
	if i < 0 {
		return nil
	}
	rest := combined[i+len(head):]
	j := strings.Index(rest, "]: ")
	if j < 0 {
		return nil
	}
	inner := rest[:j]
	if inner == "" {
		return []string{}
	}
	return strings.Split(inner, "|")
}

func combineStreams(stdout, stderr string) string {
	return stdout + stderr
}

// frontmatterField reads one frontmatter value straight off disk.
func frontmatterField(t *testing.T, path, field string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(field) + `:\s*(.*)$`)
	m := re.FindStringSubmatch(string(data))
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func specPath(root, slug string) string {
	return filepath.Join(root, ".hero", "planning", "features", slug+".md")
}

// ---------------------------------------------------------------------------
// hero spec score
// ---------------------------------------------------------------------------

// TestScorePicksASpecAtATerminal is AC-1 for score.
func TestScorePicksASpecAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "pick-older\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("spec score blocked at a terminal:\n%s", combined)
	}
	if !strings.Contains(combined, "Spec [") {
		t.Fatalf("no spec picker at a terminal:\n%s", combined)
	}
	// The answer must select. score reports the slug it scored, and the
	// answered slug is deliberately NOT the first candidate.
	if !strings.Contains(combined, "Spec:  pick-older") {
		t.Errorf("the picked spec did not drive the score:\n%s", combined)
	}
	if strings.Contains(combined, "Spec:  pick-recent") {
		t.Errorf("score ran against the head of the list rather than the answer:\n%s", combined)
	}
}

// TestScoreWithSlugSuppliedDoesNotPrompt is AC-2 for score.
func TestScoreWithSlugSuppliedDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score", "pick-recent"}, condTTY, "")
	combined := combineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("a fully-specified score prompted:\n%s", combined)
	}
}

// TestScoreWithoutATerminalKeepsItsArgumentError is AC-3 for score.
func TestScoreWithoutATerminalKeepsItsArgumentError(t *testing.T) {
	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newPickerWorkspace(t, twoSpecs()...)

			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"spec", "score"}, cond, "pick-older\n")
			combined := combineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("spec score blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if errorLine(combined) != "Error: accepts 1 arg(s), received 0" {
				t.Errorf("error line = %q, want the pre-existing argument error", errorLine(combined))
			}
			if strings.Contains(combined, "Spec [") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
			if strings.Contains(combined, "pick-older") && strings.Contains(combined, "Score:") {
				t.Errorf("score took its slug off a non-terminal stream:\n%s", combined)
			}
		})
	}
}

func TestScoreUnderJSONDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score", "--json"}, condTTY, "")
	combined := combineStreams(stdout, stderr)
	if exit == -1 {
		t.Fatalf("spec score blocked under --json:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") || errorLine(combined) != "Error: accepts 1 arg(s), received 0" {
		t.Errorf("score did not retain its JSON argument path:\n%s", combined)
	}
}

// TestScoreWithAnEmptyCorpusKeepsItsArgumentError is AC-4.
//
// Zero candidates is the same condition as "no value available": the command
// must report what it always reported rather than rendering `Spec []: `, which
// asks a question with no answer and then fails on the empty reply.
func TestScoreWithAnEmptyCorpusKeepsItsArgumentError(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t) // no specs at all

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("spec score blocked on an empty corpus:\n%s", combined)
	}
	if exit == 0 {
		t.Errorf("exit = 0, want non-zero:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("rendered a picker over an empty corpus:\n%s", combined)
	}
	if !strings.Contains(combined, "accepts 1 arg(s), received 0") {
		t.Errorf("empty corpus did not report the command's existing error:\n%s", combined)
	}
}

// TestScoreEmptyAnswerIsRejected pins the empty-answer branch.
//
// prompt.Choice returns ("", nil) for an empty answer — the caller owns its own
// default, and a SELECTOR has none. Without this the picker would fall through
// to resolveSpec("") and report `spec "" not found`.
func TestScoreEmptyAnswerIsRejected(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "\n")
	combined := combineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0: an empty answer selected a spec\n%s", combined)
	}
	if strings.Contains(combined, "Score:") {
		t.Errorf("an empty answer scored something anyway:\n%s", combined)
	}
	if !strings.Contains(combined, ErrSelectorCancelled.Error()) {
		t.Errorf("an empty answer did not report selector cancellation:\n%s", combined)
	}
}

func TestPickerRendersOneCandidate(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, pickerSpec{slug: "only-spec", modAge: time.Hour})

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "only-spec\n")
	combined := combineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if got := pickerOptions(t, combined, "Spec"); strings.Join(got, ",") != "only-spec" {
		t.Errorf("one-item picker options = %v, want [only-spec]", got)
	}
}

// ---------------------------------------------------------------------------
// corpus policy: ordering, the cap
// ---------------------------------------------------------------------------

// TestSpecPickerOrderingMatchesHeroList is AC-5.
//
// The comparison is against `hero list`'s ACTUAL output in the same workspace,
// not against an expectation written by hand. A hand-written expectation only
// proves the picker matches whatever the test author believed `hero list` does;
// this proves it matches what `hero list` does.
//
// The fixture's mtimes make the recency order neither alphabetical nor write
// order, so a picker that sorted by slug — or that skipped the ranking entirely
// and used spec.Discover's directory order — fails here.
func TestSpecPickerOrderingMatchesHeroList(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t,
		pickerSpec{slug: "delta-pick", modAge: 3 * time.Hour},
		pickerSpec{slug: "alpha-pick", modAge: 1 * time.Hour},
		pickerSpec{slug: "charlie-pick", modAge: 4 * time.Hour},
		pickerSpec{slug: "bravo-pick", modAge: 2 * time.Hour},
		// Closed work is in the workspace but not in `hero list`'s default
		// output, so a picker that dropped ExcludeClosedDefault would offer a
		// completed spec the listing never showed. Its mtime puts it in the
		// middle, so the mismatch is an extra element, not a trailing one.
		pickerSpec{slug: "echo-pick", status: "completed", modAge: 150 * time.Minute},
	)

	exit, listOut, listErr := runHero(t, bin, base, root,
		[]string{"list", "--format", "json"}, condPipe, "")
	if exit != 0 {
		t.Fatalf("hero list exit = %d:\n%s", exit, combineStreams(listOut, listErr))
	}
	var rows []struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal([]byte(listOut), &rows); err != nil {
		t.Fatalf("hero list --format json is not an array: %v\n%s", err, listOut)
	}
	want := make([]string, 0, len(rows))
	for _, r := range rows {
		want = append(want, r.Slug)
	}
	if len(want) != 4 {
		t.Fatalf("fixture did not produce four listed specs, got %v", want)
	}

	_, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "\n")
	got := pickerOptions(t, combineStreams(stdout, stderr), "Spec")

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("picker order = %v\nhero list order = %v\nA picker that orders differently from the "+
			"listing the user just read is how someone picks the wrong item.", got, want)
	}
	// Guard the guard: an alphabetical picker would coincidentally pass if the
	// fixture happened to be alphabetical. It must not be.
	sorted := append([]string(nil), want...)
	for i := 1; i < len(sorted); i++ {
		if sorted[i-1] > sorted[i] {
			return // not alphabetical — the assertion above is meaningful
		}
	}
	t.Fatal("the fixture's recency order is alphabetical, so this test cannot tell the two apart")
}

func TestPickerFiltersAnOversizedCorpus(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	over := make([]pickerSpec, 0, pickerMax+1)
	for i := 0; i <= pickerMax; i++ {
		over = append(over, pickerSpec{
			slug:   fmt.Sprintf("bulk-spec-%02d", i),
			modAge: time.Duration(i) * time.Hour,
		})
	}
	writePickerSpecs(t, root, over...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "bulk-spec-00\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("spec score blocked on an oversized corpus:\n%s", combined)
	}
	if exit != 0 {
		t.Errorf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("an exact filter should select immediately, not render a choice:\n%s", combined)
	}
	if !strings.Contains(combined, "Filter Spec:") {
		t.Errorf("did not ask for a filter:\n%s", combined)
	}
	if !strings.Contains(combined, "Spec:  bulk-spec-00") {
		t.Errorf("the exact filter did not select its candidate:\n%s", combined)
	}
}

func TestPickerFiltersTwoHundredFiftyCandidatesToABoundedChoice(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)
	bulk := make([]pickerSpec, 0, 250)
	for i := 0; i < 248; i++ {
		bulk = append(bulk, pickerSpec{slug: fmt.Sprintf("bulk-%03d", i), modAge: time.Duration(i) * time.Hour})
	}
	bulk = append(bulk,
		pickerSpec{slug: "needle-first", modAge: 249 * time.Hour},
		pickerSpec{slug: "needle-final", modAge: 250 * time.Hour},
	)
	writePickerSpecs(t, root, bulk...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "needle\nneedle-final\n")
	combined := combineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if !strings.Contains(combined, "2 spec candidate(s) match") {
		t.Errorf("filter did not report its bounded result:\n%s", combined)
	}
	if got := pickerOptions(t, combined, "Spec"); len(got) != 2 || got[0] != "needle-first" || got[1] != "needle-final" {
		t.Errorf("filtered choice = %v, want stable recency order [needle-first needle-final]", got)
	}
	if !strings.Contains(combined, "Spec:  needle-final") {
		t.Errorf("candidate beyond the first 25 was not selected:\n%s", combined)
	}
}

func TestPickerRetriesNoMatchThenSelects(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)
	bulk := make([]pickerSpec, 0, 26)
	for i := 0; i < 26; i++ {
		bulk = append(bulk, pickerSpec{slug: fmt.Sprintf("bulk-spec-%02d", i), modAge: time.Duration(i) * time.Hour})
	}
	writePickerSpecs(t, root, bulk...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "missing\nbulk-spec-00\n")
	combined := combineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if !strings.Contains(combined, `No spec candidates match "missing". Try again.`) {
		t.Errorf("missing filter did not give a retry message:\n%s", combined)
	}
	if !strings.Contains(combined, "Spec:  bulk-spec-00") {
		t.Errorf("retry did not select the exact candidate:\n%s", combined)
	}
}

func TestPickerCancellationAndInvalidFinalChoiceDoNotScore(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)
	bulk := make([]pickerSpec, 0, 26)
	for i := 0; i < 24; i++ {
		bulk = append(bulk, pickerSpec{slug: fmt.Sprintf("bulk-%02d", i), modAge: time.Duration(i) * time.Hour})
	}
	bulk = append(bulk, pickerSpec{slug: "target-one", modAge: 25 * time.Hour}, pickerSpec{slug: "target-two", modAge: 26 * time.Hour})
	writePickerSpecs(t, root, bulk...)

	for name, tc := range map[string]struct {
		stdin string
		want  string
	}{
		"cancel":  {stdin: "\n", want: ErrSelectorCancelled.Error()},
		"invalid": {stdin: "target\nnot-a-choice\n", want: "invalid choice"},
	} {
		t.Run(name, func(t *testing.T) {
			exit, stdout, stderr := runHero(t, bin, base, root, []string{"spec", "score"}, condTTY, tc.stdin)
			combined := combineStreams(stdout, stderr)
			if exit == 0 || strings.Contains(combined, "Score:") {
				t.Fatalf("selector %s ran score unexpectedly:\n%s", name, combined)
			}
			if !strings.Contains(combined, tc.want) {
				t.Errorf("selector %s error = %q, want %q", name, combined, tc.want)
			}
		})
	}
}

func TestSelectorFilteringIsStableAndScopeAware(t *testing.T) {
	if got := filterCandidates([]string{"Zulu", "alpha", "ALPHA-two", "beta"}, "alp"); strings.Join(got, ",") != "alpha,ALPHA-two" {
		t.Errorf("case-insensitive stable filter = %v", got)
	}

	base, root := newSanctionedWorkspace(t)
	subs := &install.SubprojectsManifest{}
	subs.AddSubproject(install.Subproject{Path: "services/api", Scope: "services/api"})
	if err := install.SaveSubprojects(filepath.Join(root, ".hero"), subs); err != nil {
		t.Fatalf("save subprojects: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "services", "api"), 0o755); err != nil {
		t.Fatalf("mkdir active scope: %v", err)
	}
	t.Chdir(filepath.Join(root, "services", "api"))
	now := time.Now()
	got := specSlugCandidates([]*spec.Spec{
		{Slug: "root", Type: spec.TypeFeature, Status: spec.StatusPlanning, ModifiedAt: now},
		{Slug: "scoped", Type: spec.TypeFeature, Status: spec.StatusPlanning, Subproject: "services/api", ModifiedAt: now.Add(-time.Hour)},
	})
	if strings.Join(got, ",") != "scoped" {
		t.Errorf("active scope candidates = %v, want [scoped]", got)
	}
	_ = base
}

// TestPickerRendersExactlyAtTheCap is the other side of the boundary: pickerMax
// candidates still render. Without it the cap could be off by one in the
// direction that silently disables every picker.
func TestPickerRendersExactlyAtTheCap(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	atCap := make([]pickerSpec, 0, pickerMax)
	for i := 0; i < pickerMax; i++ {
		atCap = append(atCap, pickerSpec{
			slug:   fmt.Sprintf("bulk-spec-%02d", i),
			modAge: time.Duration(i) * time.Hour,
		})
	}
	writePickerSpecs(t, root, atCap...)

	_, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "score"}, condTTY, "bulk-spec-07\n")
	combined := combineStreams(stdout, stderr)

	if got := pickerOptions(t, combined, "Spec"); len(got) != pickerMax {
		t.Errorf("picker rendered %d options at the cap of %d:\n%s", len(got), pickerMax, combined)
	}
	if !strings.Contains(combined, "Spec:  bulk-spec-07") {
		t.Errorf("the answer at the cap did not select:\n%s", combined)
	}
}

// TestSelectorCorpusResolutionIsLocal is AC-6.
//
// Structural, deliberately. A behavioral "it did not use the network" assertion
// is unfalsifiable without sandboxing the process, whereas the property that
// actually matters is that no resolver in selector.go reaches for a client. The
// team-server and tracker clients are the only two ways CLI code makes a call.
func TestSelectorCorpusResolutionIsLocal(t *testing.T) {
	src, err := os.ReadFile("selector.go")
	if err != nil {
		t.Fatalf("read selector.go: %v", err)
	}
	body := stripComments(string(src))
	for _, banned := range []string{"http.", "teamclient", "serve.NewClient", "tracker."} {
		if strings.Contains(body, banned) {
			t.Errorf("selector.go references %s — every corpus in this child resolves from local "+
				"state; whether a SELECTOR may make a network call is unresolved in the initiative.", banned)
		}
	}
	// And the resolvers it does use must be the local ones.
	for _, want := range []string{"loadAllSpecs()", "skills.Discover("} {
		if !strings.Contains(body, want) {
			t.Errorf("selector.go no longer resolves its corpus via %s", want)
		}
	}
}

// ---------------------------------------------------------------------------
// hero spec verify
// ---------------------------------------------------------------------------

// TestVerifyPicksASpecAtATerminal is AC-1 for verify.
//
// verify makes no state change on a planning spec — it reports the lifecycle
// refusal and stops — so the discriminator is that the refusal names the
// ANSWERED spec and not the head of the list.
func TestVerifyPicksASpecAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "verify"}, condTTY, "pick-older\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("spec verify blocked at a terminal:\n%s", combined)
	}
	if !strings.Contains(combined, "Spec [") {
		t.Fatalf("no spec picker at a terminal:\n%s", combined)
	}
	if !strings.Contains(combined, `"pick-older"`) {
		t.Errorf("the picked spec did not drive verify:\n%s", combined)
	}
	if strings.Contains(combined, `"pick-recent"`) {
		t.Errorf("verify ran against the head of the list rather than the answer:\n%s", combined)
	}
}

// TestVerifyWithoutATerminalKeepsItsArgumentError is AC-3 for verify.
func TestVerifyWithoutATerminalKeepsItsArgumentError(t *testing.T) {
	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newPickerWorkspace(t, twoSpecs()...)

			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"spec", "verify"}, cond, "pick-older\n")
			combined := combineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("spec verify blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if errorLine(combined) != "Error: accepts 1 arg(s), received 0" {
				t.Errorf("error line = %q, want the pre-existing argument error", errorLine(combined))
			}
			if strings.Contains(combined, "Spec [") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
		})
	}
}

// TestVerifyUnderJSONDoesNotPrompt is AC-9 for verify.
//
// The terminal is real and nothing is ever typed into it. If the --json refusal
// were missing the command would render a picker and block forever, which is
// exactly what a programmatic caller would experience.
func TestVerifyUnderJSONDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "verify", "--json"}, condTTY, "")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("spec verify BLOCKED under --json with a terminal attached — it prompted and is "+
			"waiting for an answer no programmatic caller will give:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("prompted under --json:\n%s", combined)
	}
	if errorLine(combined) != "Error: accepts 1 arg(s), received 0" {
		t.Errorf("error line = %q, want the pre-existing argument error under --json", errorLine(combined))
	}
}

// ---------------------------------------------------------------------------
// hero spec move
// ---------------------------------------------------------------------------

// TestSpecMovePicksTheSourceSlug is AC-1 for spec move, asserted on disk.
func TestSpecMovePicksTheSourceSlug(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "move", "--to-scope", ""}, condTTY, "pick-older\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("spec move blocked at a terminal:\n%s", combined)
	}
	if !strings.Contains(combined, "Spec [") {
		t.Fatalf("no spec picker at a terminal:\n%s", combined)
	}
	// Re-rooting a spec that is already at root is a no-op the command reports
	// by name; that name is the proof the answer selected.
	if !strings.Contains(combined, `Spec "pick-older" is already in scope`) {
		t.Errorf("the picked spec did not drive the move:\n%s", combined)
	}
	if strings.Contains(combined, `Spec "pick-recent"`) {
		t.Errorf("move ran against the head of the list rather than the answer:\n%s", combined)
	}
}

// TestSpecMoveReportsTheMissingFlagBeforeAsking pins the ordering choice.
//
// The picker sits after the --to-scope validation: asking which spec to move
// and only then rejecting the destination wastes the answer. `hero spec move`
// with neither value therefore still reports the missing flag first, exactly as
// it does today.
func TestSpecMoveReportsTheMissingFlagBeforeAsking(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"spec", "move"}, condTTY, "pick-older\n")
	combined := combineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0, want non-zero:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("asked which spec to move before checking --to-scope:\n%s", combined)
	}
	if !strings.Contains(combined, "--to-scope is required") {
		t.Errorf("error does not name the missing flag:\n%s", combined)
	}
}

// TestSpecMoveWithoutATerminalKeepsItsArgumentError is AC-3 for spec move.
func TestSpecMoveWithoutATerminalKeepsItsArgumentError(t *testing.T) {
	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newPickerWorkspace(t, twoSpecs()...)

			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"spec", "move", "--to-scope", ""}, cond, "pick-older\n")
			combined := combineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("spec move blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if errorLine(combined) != "Error: accepts 1 arg(s), received 0" {
				t.Errorf("error line = %q, want the pre-existing argument error", errorLine(combined))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// hero supersede — two corpus inputs
// ---------------------------------------------------------------------------

// TestSupersedeAsksOnlyForTheOmittedHalf is AC-7.
//
// `hero supersede <old>` with --by omitted must ask for --by and NOTHING else.
// Prompting for a value the user already typed is a behavior change beyond the
// additive guarantee, and it is the specific risk the spec calls out.
//
// The assertion is on disk: superseded_by lands on the supplied old spec, and
// the inverse supersedes: relation lands on the answered replacement.
func TestSupersedeAsksOnlyForTheOmittedHalf(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"supersede", "pick-older"}, condTTY, "pick-recent\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("supersede blocked at a terminal:\n%s", combined)
	}
	if strings.Contains(combined, "Spec to supersede [") {
		t.Errorf("prompted for the old slug even though it was supplied:\n%s", combined)
	}
	if !strings.Contains(combined, "Replaced by [") {
		t.Fatalf("no picker for the omitted --by:\n%s", combined)
	}
	if got := frontmatterField(t, specPath(root, "pick-older"), "superseded_by"); got != "pick-recent" {
		t.Errorf("superseded_by = %q, want the answered replacement (exit %d)\n%s", got, exit, combined)
	}
}

// TestSupersedeAsksForBothWhenBothAreOmitted covers the bare invocation.
func TestSupersedeAsksForBothWhenBothAreOmitted(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"supersede"}, condTTY, "pick-older\npick-recent\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("supersede blocked at a terminal:\n%s", combined)
	}
	if !strings.Contains(combined, "Spec to supersede [") {
		t.Errorf("no picker for the old slug:\n%s", combined)
	}
	if !strings.Contains(combined, "Replaced by [") {
		t.Errorf("no picker for --by:\n%s", combined)
	}
	if got := frontmatterField(t, specPath(root, "pick-older"), "superseded_by"); got != "pick-recent" {
		t.Errorf("superseded_by = %q, want the answered pair (exit %d)\n%s", got, exit, combined)
	}
}

// TestSupersedeByPickerExcludesTheOldSlug pins the one narrowing the --by
// picker applies.
//
// supersede refuses `a --by a` outright, so offering the already-chosen old
// slug as a replacement would put a guaranteed-invalid option in the list.
func TestSupersedeByPickerExcludesTheOldSlug(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	_, stdout, stderr := runHero(t, bin, base, root,
		[]string{"supersede", "pick-older"}, condTTY, "\n")
	combined := combineStreams(stdout, stderr)

	options := pickerOptions(t, combined, "Replaced by")
	if len(options) == 0 {
		t.Fatalf("no --by picker rendered:\n%s", combined)
	}
	for _, o := range options {
		if o == "pick-older" {
			t.Errorf("the --by picker offers the spec being superseded: %v", options)
		}
	}
	if len(options) != 1 || options[0] != "pick-recent" {
		t.Errorf("--by options = %v, want just the other spec", options)
	}
}

// TestSupersedeWithoutATerminalKeepsItsErrors is AC-3 for supersede, both
// halves.
func TestSupersedeWithoutATerminalKeepsItsErrors(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no old slug", []string{"supersede"}, "supply the old spec slug (the one being replaced)"},
		{"no --by", []string{"supersede", "pick-older"}, "--by <new-slug> is required (the replacement spec)"},
	}
	for _, tc := range cases {
		for _, cond := range []string{condPipe, condClosed} {
			t.Run(tc.name+"/"+cond, func(t *testing.T) {
				bin := baselineBinary(t)
				base, root := newPickerWorkspace(t, twoSpecs()...)

				exit, stdout, stderr := runHero(t, bin, base, root, tc.args, cond, "pick-recent\n")
				combined := combineStreams(stdout, stderr)

				if exit == -1 {
					t.Fatalf("supersede blocked on a non-terminal — hard constraint 3:\n%s", combined)
				}
				if exit == 0 {
					t.Errorf("exit = 0, want non-zero:\n%s", combined)
				}
				if !strings.Contains(combined, tc.wantErr) {
					t.Errorf("output does not carry the pre-existing error %q:\n%s", tc.wantErr, combined)
				}
				if strings.Contains(combined, "[pick-") {
					t.Errorf("prompted on a non-terminal:\n%s", combined)
				}
				if got := frontmatterField(t, specPath(root, "pick-older"), "superseded_by"); got != "" {
					t.Errorf("supersede took a value off a non-terminal stream and wrote %q", got)
				}
			})
		}
	}
}

// TestSupersedeWithBothSuppliedDoesNotPrompt is AC-2 for supersede.
func TestSupersedeWithBothSuppliedDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"supersede", "pick-older", "--by", "pick-recent"}, condTTY, "")
	combined := combineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Spec to supersede [") || strings.Contains(combined, "Replaced by [") {
		t.Errorf("a fully-specified supersede prompted:\n%s", combined)
	}
}

// TestSupersedeScanTakesNoPositionalsStill guards the mode dispatch.
//
// supersede has no arity rule to relax, so its terminal gate lives in RunE
// after the --scan/--list/--unset dispatch. A gate placed before it would start
// asking `hero supersede --scan` which spec to supersede.
func TestSupersedeScanTakesNoPositionalsStill(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"supersede", "--scan"}, condTTY, "")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("supersede --scan blocked at a terminal — it is asking something:\n%s", combined)
	}
	if strings.Contains(combined, "Spec to supersede [") {
		t.Errorf("--scan prompted for a spec:\n%s", combined)
	}
}

// ---------------------------------------------------------------------------
// hero size
// ---------------------------------------------------------------------------

// TestSizePicksASpecAtATerminal is AC-1 for size.
//
// The two fixture specs carry DIFFERENT declared sizes, so the tier `hero size`
// prints identifies which spec it read. A picker that rendered and ignored its
// answer would print the other tier.
func TestSizePicksASpecAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"size"}, condTTY, "pick-older\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("size blocked at a terminal:\n%s", combined)
	}
	if !strings.Contains(combined, "Spec [") {
		t.Fatalf("no spec picker at a terminal:\n%s", combined)
	}
	if !strings.Contains(stdout, "large") {
		t.Errorf("size did not print the answered spec's tier (want large):\n%s", combined)
	}
	if strings.Contains(stdout, "small") {
		t.Errorf("size read the head of the list rather than the answer:\n%s", combined)
	}
}

// TestSizeWithoutATerminalKeepsItsUsageError is AC-3 for size.
//
// `hero size` sets SilenceErrors, so cmd/hero/main.go prints the bare error
// with no "Error: " prefix — errorLine does not apply here.
func TestSizeWithoutATerminalKeepsItsUsageError(t *testing.T) {
	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newPickerWorkspace(t, twoSpecs()...)

			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"size"}, cond, "pick-older\n")
			combined := combineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("size blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if !strings.Contains(combined, "usage: hero size <slug> [tier]") {
				t.Errorf("output does not carry the pre-existing usage error:\n%s", combined)
			}
			if strings.Contains(combined, "Spec [") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
		})
	}
}

// TestSizeCheckStillTakesNoPositionals guards size's mode dispatch.
//
// --check is why size cannot use an Args wrapper: zero positionals is a legal,
// complete invocation there. A gate placed before the mode dispatch would ask
// `hero size --check` which spec to report on.
func TestSizeCheckStillTakesNoPositionals(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"size", "--check"}, condTTY, "")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("size --check blocked at a terminal — it is asking something:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("--check prompted for a spec:\n%s", combined)
	}
	if !strings.Contains(combined, "tracker:") {
		t.Errorf("--check did not run its scan:\n%s", combined)
	}
}

// TestSizeWithSlugSuppliedDoesNotPrompt is AC-2 for size.
func TestSizeWithSlugSuppliedDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"size", "pick-recent"}, condTTY, "")
	combined := combineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("a fully-specified size prompted:\n%s", combined)
	}
	if !strings.Contains(stdout, "small") {
		t.Errorf("size did not print the supplied spec's tier:\n%s", combined)
	}
}

// ---------------------------------------------------------------------------
// hero skill — five commands, one corpus
// ---------------------------------------------------------------------------

func writeSkills(t *testing.T, root string, names ...string) {
	t.Helper()
	dir := filepath.Join(root, ".hero", "skills")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	for _, n := range names {
		body := "---\ntitle: " + n + "\nversion: 1\n---\n\n# " + n +
			"\n\n## Steps\n\n1. Prompt agent: marker-for-" + n + "\n"
		if err := os.WriteFile(filepath.Join(dir, n+".md"), []byte(body), 0o644); err != nil {
			t.Fatalf("write skill %s: %v", n, err)
		}
	}
}

// TestSkillShowPicksAnInstalledSkill is AC-1 for the skill corpus.
//
// `skill show` prints the file, and each fixture skill carries a unique marker,
// so the printed body identifies which one was opened.
func TestSkillShowPicksAnInstalledSkill(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)
	writeSkills(t, root, "alpha-skill", "zulu-skill")

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"skill", "show"}, condTTY, "zulu-skill\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("skill show blocked at a terminal:\n%s", combined)
	}
	if got := pickerOptions(t, combined, "Skill"); strings.Join(got, ",") != "alpha-skill,zulu-skill" {
		t.Errorf("skill picker options = %v, want the installed skills in `hero skill list` order", got)
	}
	if !strings.Contains(combined, "marker-for-zulu-skill") {
		t.Errorf("the picked skill was not the one shown:\n%s", combined)
	}
	if strings.Contains(combined, "marker-for-alpha-skill") {
		t.Errorf("skill show opened the head of the list rather than the answer:\n%s", combined)
	}
}

// TestSkillRmPicksAnInstalledSkill asserts the picker on the one skill command
// that changes state.
func TestSkillRmPicksAnInstalledSkill(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)
	writeSkills(t, root, "alpha-skill", "zulu-skill")

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"skill", "rm"}, condTTY, "zulu-skill\n")
	combined := combineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if _, err := os.Stat(filepath.Join(root, ".hero", "skills", "zulu-skill.md")); err == nil {
		t.Errorf("the answered skill was not removed:\n%s", combined)
	}
	if _, err := os.Stat(filepath.Join(root, ".hero", "skills", "alpha-skill.md")); err != nil {
		t.Errorf("skill rm removed the head of the list rather than the answer: %v", err)
	}
}

// TestSkillCommandsWithoutATerminalKeepTheirArgumentError is AC-3 across all
// five name-taking skill subcommands.
//
// All five, not a representative one: each carries its own Args wiring, and
// swapping any single one back to a plain cobra rule must turn exactly its own
// case red.
func TestSkillCommandsWithoutATerminalKeepTheirArgumentError(t *testing.T) {
	for _, verb := range []string{"show", "run", "edit", "rm", "log"} {
		t.Run(verb, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newSanctionedWorkspace(t)
			writeSkills(t, root, "alpha-skill", "zulu-skill")

			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"skill", verb}, condPipe, "zulu-skill\n")
			combined := combineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("skill %s blocked on a non-terminal — hard constraint 3:\n%s", verb, combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if errorLine(combined) != "Error: accepts 1 arg(s), received 0" {
				t.Errorf("error line = %q, want the pre-existing argument error", errorLine(combined))
			}
			if strings.Contains(combined, "Skill [") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
		})
	}
}

// TestSkillCommandsPromptAtATerminal is AC-1 across all five, for the same
// per-command reason.
func TestSkillCommandsPromptAtATerminal(t *testing.T) {
	for _, verb := range []string{"show", "run", "edit", "rm", "log"} {
		t.Run(verb, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newSanctionedWorkspace(t)
			writeSkills(t, root, "alpha-skill", "zulu-skill")

			// An empty answer, so nothing destructive runs and every verb
			// stops at the same place.
			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"skill", verb}, condTTY, "\n")
			combined := combineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("skill %s blocked at a terminal:\n%s", verb, combined)
			}
			if got := pickerOptions(t, combined, "Skill"); strings.Join(got, ",") != "alpha-skill,zulu-skill" {
				t.Errorf("skill %s picker options = %v, want the installed skills", verb, got)
			}
		})
	}
}

// TestSkillPickerRefusesAnEmptyCorpus is AC-4 for the skill corpus.
func TestSkillPickerRefusesAnEmptyCorpus(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t) // workspace has an empty skills dir

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"skill", "show"}, condTTY, "\n")
	combined := combineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0, want non-zero:\n%s", combined)
	}
	if strings.Contains(combined, "Skill [") {
		t.Errorf("rendered a picker over an empty skill corpus:\n%s", combined)
	}
	if !strings.Contains(combined, "accepts 1 arg(s), received 0") {
		t.Errorf("empty corpus did not report the command's existing error:\n%s", combined)
	}
}

// TestSkillRunWithNameSuppliedStillPromptsForParams pins the seam with the
// `skill run` parameter loop.
//
// The name picker lands in front of a site that already prompts. They must
// compose: a supplied name skips the name picker and still collects params.
func TestSkillRunWithNameSuppliedStillPromptsForParams(t *testing.T) {
	bin := baselineBinary(t)
	base := t.TempDir()
	root := filepath.Join(base, "work")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir work root: %v", err)
	}
	setupSkillWithParam(t, root)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"skill", "run", "baseline-skill"}, condTTY, "abc\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("skill run blocked at a terminal:\n%s", combined)
	}
	if strings.Contains(combined, "Skill [") {
		t.Errorf("a supplied skill name still triggered the name picker:\n%s", combined)
	}
	if !strings.Contains(combined, "target (the thing to act on): ") {
		t.Errorf("the parameter loop no longer runs behind the name picker:\n%s", combined)
	}
}

// ---------------------------------------------------------------------------
// hero handoff / hero handoff accept
// ---------------------------------------------------------------------------

// registerPeer adds a sibling hero workspace and wires it into hero.json.
//
// The sibling gets a real peer manifest via `hero peer manifest`, because
// peering.Handoff refuses to queue a transfer to a workspace that has not
// published one. Without it the command fails after the pickers have already
// done their job, which would leave the state assertion unreachable.
func registerPeer(t *testing.T, bin, base, root, alias string) {
	t.Helper()
	peerRoot := filepath.Join(base, "peers", alias)
	if err := os.MkdirAll(peerRoot, 0o755); err != nil {
		t.Fatalf("mkdir peer %s: %v", alias, err)
	}
	writeWorkspace(t, peerRoot)
	if exit, out, errOut := runHero(t, bin, base, peerRoot,
		[]string{"peer", "manifest"}, condPipe, ""); exit != 0 {
		t.Fatalf("hero peer manifest in %s exit = %d:\n%s", alias, exit, combineStreams(out, errOut))
	}

	path := filepath.Join(root, ".hero", "hero.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read hero.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse hero.json: %v", err)
	}
	// config.Config models repos as alias -> path, not as an object.
	repos, _ := cfg["repos"].(map[string]any)
	if repos == nil {
		repos = map[string]any{}
	}
	repos[alias] = peerRoot
	cfg["repos"] = repos
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal hero.json: %v", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatalf("write hero.json: %v", err)
	}
}

// TestHandoffPicksSpecAndPeer is AC-1 for handoff's two corpora.
//
// Two peers are registered and the answer is the second one, so a peer picker
// that ignored its answer would queue the transfer to the wrong repo. The proof
// is the trail entry handoff appends to the spec file — on disk, and read back
// through `hero handoff status`.
func TestHandoffPicksSpecAndPeer(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)
	registerPeer(t, bin, base, root, "alpha-peer")
	registerPeer(t, bin, base, root, "zulu-peer")

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"handoff"}, condTTY, "pick-older\nzulu-peer\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("handoff blocked at a terminal:\n%s", combined)
	}
	if !strings.Contains(combined, "Spec [") {
		t.Errorf("no spec picker at a terminal:\n%s", combined)
	}
	if got := pickerOptions(t, combined, "Peer"); strings.Join(got, ",") != "alpha-peer,zulu-peer" {
		t.Errorf("peer picker options = %v, want the registered peers in the order the "+
			"\"configured peers\" error lists them", got)
	}
	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}

	// State: `hero handoff` queues a durable Mail message, and the message
	// records BOTH answers — the recipient it was addressed to and the origin
	// slug it carries. Asserted off the message on disk rather than off the
	// success line, which merely echoes the alias variable back.
	queued := queuedMail(t, base)
	if len(queued) != 1 {
		t.Fatalf("want exactly one queued transfer, got %d:\n%s", len(queued), combined)
	}
	msg := queued[0]
	if msg.Recipient.RegistrySlug != "zulu-peer" {
		t.Errorf("the transfer was addressed to %q — the peer picker's answer did not select",
			msg.Recipient.RegistrySlug)
	}
	if !strings.Contains(msg.Body, `"origin_slug":"pick-older"`) {
		t.Errorf("the transfer carries the wrong spec — the spec picker's answer did not select:\n%s", msg.Body)
	}
}

// mailMessage is the slice of a queued Mail record these tests assert on.
type mailMessage struct {
	Recipient struct {
		RegistrySlug string `json:"registry_slug"`
	} `json:"recipient"`
	Body string `json:"body"`
	Kind string `json:"kind"`
}

// queuedMail returns every message sitting in the per-user Mail store the test
// child writes to.
//
// The store is at $HOME/.local/state/hero/mail, and runHero points HOME at
// <base>/home, so it is isolated per test.
func queuedMail(t *testing.T, base string) []mailMessage {
	t.Helper()
	boxes := filepath.Join(base, "home", ".local", "state", "hero", "mail", "boxes")
	var out []mailMessage
	err := filepath.WalkDir(boxes, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var m mailMessage
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		out = append(out, m)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", boxes, err)
	}
	return out
}

// TestHandoffAsksOnlyForTheOmittedHalf is the two-input rule applied to
// handoff: `hero handoff <slug>` supplies the spec and omits the peer, so only
// the peer picker may fire.
func TestHandoffAsksOnlyForTheOmittedHalf(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)
	registerPeer(t, bin, base, root, "alpha-peer")
	registerPeer(t, bin, base, root, "zulu-peer")

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"handoff", "pick-recent"}, condTTY, "zulu-peer\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("handoff blocked at a terminal:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("prompted for the spec even though it was supplied:\n%s", combined)
	}
	if !strings.Contains(combined, "Peer [") {
		t.Fatalf("no peer picker for the omitted half:\n%s", combined)
	}
	queued := queuedMail(t, base)
	if len(queued) != 1 {
		t.Fatalf("want exactly one queued transfer, got %d:\n%s", len(queued), combined)
	}
	if queued[0].Recipient.RegistrySlug != "zulu-peer" {
		t.Errorf("addressed to %q, want the answered peer", queued[0].Recipient.RegistrySlug)
	}
	if !strings.Contains(queued[0].Body, `"origin_slug":"pick-recent"`) {
		t.Errorf("the transfer does not carry the SUPPLIED spec:\n%s", queued[0].Body)
	}
}

// TestHandoffWithNoPeersRegisteredDoesNotSpendTheAnswer is the cold audit's
// finding.
//
// A workspace with no registered peers cannot complete a handoff whichever spec
// is chosen. Asking which one anyway throws the answer away AND buries the only
// message that says how to fix it — `hero repos add` — behind a bare
// argument-count error. This is the ordering rule `spec move` already follows
// by validating --to-scope before it asks.
func TestHandoffWithNoPeersRegisteredDoesNotSpendTheAnswer(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...) // specs, but no peers

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"handoff"}, condTTY, "pick-older\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("handoff blocked at a terminal:\n%s", combined)
	}
	if exit == 0 {
		t.Errorf("exit = 0, want non-zero:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("asked which spec to hand off with nowhere to send it — the answer is discarded:\n%s", combined)
	}
	if !strings.Contains(combined, "hero repos add") {
		t.Errorf("the failure does not name how to register a peer — this is the guidance the bare "+
			"argument error hides:\n%s", combined)
	}
	// And nothing was queued.
	if q := queuedMail(t, base); len(q) != 0 {
		t.Errorf("a transfer was queued with no peer registered: %d message(s)", len(q))
	}
}

// TestHandoffWithASuppliedAliasKeepsItsUnregisteredError pins the other half of
// the shared constructor: the supplied-alias wording is unchanged.
func TestHandoffWithASuppliedAliasKeepsItsUnregisteredError(t *testing.T) {
	for _, tc := range []struct {
		name    string
		peers   []string
		wantErr string
	}{
		{"no peers registered", nil, `peer "ghost" is not configured — register one with `},
		{"some peers registered", []string{"alpha-peer"}, `peer "ghost" is not configured — configured peers: alpha-peer`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newPickerWorkspace(t, twoSpecs()...)
			for _, p := range tc.peers {
				registerPeer(t, bin, base, root, p)
			}

			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"handoff", "pick-older", "ghost"}, condPipe, "")
			combined := combineStreams(stdout, stderr)

			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if !strings.Contains(combined, tc.wantErr) {
				t.Errorf("output does not carry the pre-existing error %q:\n%s", tc.wantErr, combined)
			}
		})
	}
}

// TestSelectorCommandsTreatAnEmptyArgumentAsSupplied is the argument-supplied
// guarantee at its edge.
//
// An omitted positional is missing; a positional supplied as "" is a value the
// command already rejects with its own message. Testing emptiness instead of
// supplied-ness would make `hero spec score ""` — an invocation that carries an
// argument — open a picker, which is a prompt on a fully-specified call.
func TestSelectorCommandsTreatAnEmptyArgumentAsSupplied(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		label   string
		wantErr string
	}{
		{"spec score", []string{"spec", "score", ""}, "Spec", `spec "" not found`},
		{"spec verify", []string{"spec", "verify", ""}, "Spec", `spec "" not found`},
		{"skill show", []string{"skill", "show", ""}, "Skill", `skill "" not found`},
		{"supersede", []string{"supersede", "", "--by", "pick-recent"}, "Spec to supersede", `spec "" not found`},
		{"handoff", []string{"handoff", "pick-older", ""}, "Peer", `peer "" is not configured`},
		{"handoff accept", []string{"handoff", "accept", ""}, "Spec", `no spec with slug ""`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newPickerWorkspace(t, twoSpecs()...)
			writeSkills(t, root, "alpha-skill")
			registerPeer(t, bin, base, root, "alpha-peer")

			exit, stdout, stderr := runHero(t, bin, base, root, tc.args, condTTY, "\n")
			combined := combineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("%s blocked at a terminal:\n%s", tc.name, combined)
			}
			if strings.Contains(combined, tc.label+" [") {
				t.Errorf("prompted for a positional the invocation already supplied (as \"\"):\n%s", combined)
			}
			if !strings.Contains(combined, tc.wantErr) {
				t.Errorf("output does not carry the pre-existing error %q:\n%s", tc.wantErr, combined)
			}
		})
	}
}

// TestHandoffWithoutATerminalKeepsItsArgumentError is AC-3 for handoff.
func TestHandoffWithoutATerminalKeepsItsArgumentError(t *testing.T) {
	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newPickerWorkspace(t, twoSpecs()...)
			registerPeer(t, bin, base, root, "alpha-peer")

			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"handoff"}, cond, "pick-older\nalpha-peer\n")
			combined := combineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("handoff blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if errorLine(combined) != "Error: accepts 2 arg(s), received 0" {
				t.Errorf("error line = %q, want the pre-existing argument error", errorLine(combined))
			}
			if strings.Contains(combined, "Spec [") || strings.Contains(combined, "Peer [") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
		})
	}
}

// TestHandoffSurplusArgumentStillFailsAtATerminal pins the half of the shared
// gate no picker can repair.
//
// Too many arguments is not a shortfall: no prompt can fix an extra argument,
// and quietly ignoring it is how a typo becomes a wrong write.
func TestHandoffSurplusArgumentStillFailsAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)
	registerPeer(t, bin, base, root, "alpha-peer")

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"handoff", "pick-older", "alpha-peer", "extra"}, condTTY, "")
	combined := combineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0: a surplus argument was accepted at a terminal\n%s", combined)
	}
	if errorLine(combined) != "Error: accepts 2 arg(s), received 3" {
		t.Errorf("error line = %q, want cobra's argument-count error", errorLine(combined))
	}
}

// TestHandoffUnderJSONDoesNotPrompt is AC-9 for handoff.
func TestHandoffUnderJSONDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)
	registerPeer(t, bin, base, root, "alpha-peer")

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"handoff", "--json"}, condTTY, "")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("handoff BLOCKED under --json with a terminal attached:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") || strings.Contains(combined, "Peer [") {
		t.Errorf("prompted under --json:\n%s", combined)
	}
	if errorLine(combined) != "Error: accepts 2 arg(s), received 0" {
		t.Errorf("error line = %q, want the pre-existing argument error under --json", errorLine(combined))
	}
}

// TestHandoffAcceptOffersOnlyHandedBackSpecs is AC-8.
//
// accept refuses anything that is not handed_back, so the general spec corpus
// would put choices in the picker that the command is about to reject. The
// fixture deliberately carries three specs, only one of them handed_back.
func TestHandoffAcceptOffersOnlyHandedBackSpecs(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t,
		pickerSpec{slug: "still-planning", modAge: time.Hour},
		pickerSpec{slug: "came-back", status: "handed_back", modAge: 2 * time.Hour},
		pickerSpec{slug: "also-planning", modAge: 3 * time.Hour},
	)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"handoff", "accept"}, condTTY, "came-back\n\n")
	combined := combineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("handoff accept blocked at a terminal:\n%s", combined)
	}
	got := pickerOptions(t, combined, "Spec")
	if strings.Join(got, ",") != "came-back" {
		t.Errorf("accept picker options = %v, want only the handed_back spec", got)
	}
	// State: the answered spec's status moved off handed_back.
	if s := frontmatterField(t, specPath(root, "came-back"), "status"); s != "delivering" {
		t.Errorf("status = %q, want delivering — the picked spec was not accepted (exit %d)\n%s", s, exit, combined)
	}
}

// TestHandoffAcceptWithNoPendingHandoffsKeepsItsArgumentError is AC-4 for the
// narrowed corpus.
//
// A workspace full of specs, none handed_back, is an EMPTY corpus for this
// command. It must report the existing error rather than offering the specs it
// cannot accept.
func TestHandoffAcceptWithNoPendingHandoffsKeepsItsArgumentError(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t, twoSpecs()...)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"handoff", "accept"}, condTTY, "\n")
	combined := combineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0, want non-zero:\n%s", combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("offered a picker when no spec has a pending handoff:\n%s", combined)
	}
	if !strings.Contains(combined, "accepts 1 arg(s), received 0") {
		t.Errorf("did not report the command's existing error:\n%s", combined)
	}
}

// TestHandoffAcceptWithSlugSuppliedDoesNotPrompt is AC-2 for accept, and pins
// that the name picker did not displace promptNextStatus.
func TestHandoffAcceptWithSlugSuppliedDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newPickerWorkspace(t,
		pickerSpec{slug: "came-back", status: "handed_back", modAge: time.Hour},
	)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"handoff", "accept", "came-back"}, condTTY, "\n")
	combined := combineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Spec [") {
		t.Errorf("a fully-specified accept prompted for the slug:\n%s", combined)
	}
	if !strings.Contains(combined, "Pick the next status for this spec:") {
		t.Errorf("the pre-existing next-status prompt no longer runs:\n%s", combined)
	}
}

// ---------------------------------------------------------------------------
// the shared gate, as a unit
// ---------------------------------------------------------------------------

// TestSelectorArgsRefusesToRelaxUnderJSON is the unit-level pin on the one rule
// selectorArgs adds to promptableArgs.
//
// The subprocess suites prove it for verify and handoff. This proves the rule
// itself, including the combination they cannot reach cheaply: a shortfall at a
// terminal WITH --json set.
func TestSelectorArgsRefusesToRelaxUnderJSON(t *testing.T) {
	_, slave, _ := openCapturedPTY(t)

	newCmd := func(jsonOn bool) *cobra.Command {
		c := &cobra.Command{}
		c.SetIn(slave)
		var flag bool
		c.Flags().BoolVar(&flag, "json", false, "")
		if jsonOn {
			if err := c.Flags().Set("json", "true"); err != nil {
				t.Fatalf("set --json: %v", err)
			}
		}
		return c
	}
	noFlag := &cobra.Command{}
	noFlag.SetIn(slave)

	rule := selectorOneArg.rule()

	cases := []struct {
		name    string
		cmd     *cobra.Command
		args    []string
		wantErr bool
	}{
		{"shortfall at a terminal is relaxed", newCmd(false), []string{}, false},
		{"shortfall at a terminal under --json is NOT relaxed", newCmd(true), []string{}, true},
		{"exact count under --json passes", newCmd(true), []string{"a"}, false},
		{"surplus at a terminal still fails", newCmd(false), []string{"a", "b"}, true},
		{"a command with no --json flag is relaxed", noFlag, []string{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := rule(tc.cmd, tc.args)
			if tc.wantErr && err == nil {
				t.Errorf("args %v accepted, want rejection", tc.args)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("args %v rejected: %v", tc.args, err)
			}
		})
	}
}

// TestSelectorArgsMissingMatchesItsOwnRule is the pin on the reconciliation
// selectorArgs exists for.
//
// promptableArgs takes `need` and `strict` as independent parameters with
// nothing checking that they agree. selectorArgs derives both from one arity,
// and this proves it: the error RunE falls back to is exactly the error the
// Args rule would have produced on a pipe.
func TestSelectorArgsMissingMatchesItsOwnRule(t *testing.T) {
	for _, sel := range []selectorArgs{selectorOneArg, selectorTwoArgs} {
		pipe := &cobra.Command{}
		pipe.SetIn(strings.NewReader(""))

		fromRule := sel.rule()(pipe, nil)
		fromMissing := sel.missing(pipe, nil)
		if fromRule == nil || fromMissing == nil {
			t.Fatalf("arity %d: expected both to reject an empty argument list", sel.need)
		}
		if fromRule.Error() != fromMissing.Error() {
			t.Errorf("arity %d: Args rule says %q but the RunE fallback says %q — a command whose "+
				"two statements of arity disagree misbehaves exactly in the empty-corpus case",
				sel.need, fromRule, fromMissing)
		}
		if !strings.Contains(fromMissing.Error(), fmt.Sprintf("accepts %d arg(s)", sel.need)) {
			t.Errorf("arity %d: fallback error %q does not state the command's arity", sel.need, fromMissing)
		}
	}
}

// TestSelectorCommandsCallThePrimitivesDirectly is the standing guard on the
// initiative's scope cap, extended to this child's files.
//
// The scoped successor rejected a generic promptfield descriptor. A selector
// picks one value and must call prompt.Choice directly.
func TestSelectorCommandsCallThePrimitivesDirectly(t *testing.T) {
	adopters := []string{"selector.go", "score.go", "verify.go", "spec_move.go", "supersede.go", "size.go", "handoff.go"}

	for _, name := range adopters {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		body := stripComments(string(src))
		for _, banned := range []string{"collectFields", "promptField", "fieldReader"} {
			if strings.Contains(body, banned) {
				t.Errorf("%s references %s — a SELECTOR picks one value and must not introduce "+
					"a generic field descriptor.", name, banned)
			}
		}
	}
	if !regexp.MustCompile(`prompt\.Choice\(`).MatchString(stripComments(readFileOrFail(t, "selector.go"))) {
		t.Error("selector.go no longer calls prompt.Choice directly")
	}
}

// TestGenericFieldDescriptorRemainsAbsent pins the successor architecture:
// connect owns collectConnectFields, skill prompts directly, and no reusable
// promptfield.go surface exists.
func TestGenericFieldDescriptorRemainsAbsent(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var consumers []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") || name == "promptfield.go" {
			continue
		}
		if strings.Contains(stripComments(readFileOrFail(t, name)), "collectFields(") {
			consumers = append(consumers, name)
		}
	}
	if len(consumers) != 0 {
		t.Errorf("generic collectFields consumers = %v, want none; connect owns collectConnectFields and skill prompts directly", consumers)
	}
	if _, err := os.Stat("promptfield.go"); !os.IsNotExist(err) {
		t.Errorf("promptfield.go must remain absent; stat error = %v", err)
	}
}

func readFileOrFail(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(src)
}
