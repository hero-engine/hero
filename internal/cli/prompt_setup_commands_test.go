//go:build unix

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func setupCombineStreams(stdout, stderr string) string {
	return stdout + "\n" + stderr
}

// Adoption tests for the PROMPT-class setup commands: `hero connect`'s provider
// picker, the install/uninstall target pickers, `hero admin repos add`,
// `hero admin users add`, and `hero trust`.
//
// Every command gets the four cases the spec requires — fully supplied, missing
// with no terminal, missing with a terminal, and (where the flag exists) under
// --json.
//
// They run through the shipped binary rather than in-process cobra for the
// same three reasons the baseline does: exit codes are part of the contract,
// most of these sites print with fmt.Print rather than to cmd.OutOrStdout(),
// and the predicate under test is term.IsTerminal on a real fd 0. The
// in-process alternative cannot observe any of the three.
//
// Assertions land on state wherever state exists — the repos map in hero.json,
// the user row in the local job queue — because a claim about what a command
// DID is not provable from what it PRINTED. Two earlier tests on this
// initiative were fooled exactly that way.

// ---------------------------------------------------------------------------
// hero connect — provider picker
// ---------------------------------------------------------------------------

// TestConnectProviderPickerAsksAtATerminal is AC-1 for connect.
//
// The assertion is not "a picker appeared". It is that the ANSWER selected the
// provider: only github's guided path prints the GitHub banner and asks for a
// repository in owner/repo form. A picker that rendered and then ignored its
// answer would still print the prompt and fail this.
func TestConnectProviderPickerAsksAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	// provider, then the role default, then the repository. The token prompt
	// after it goes to /dev/tty, which the test child does not have, so the
	// command ends in the documented refusal — well past the point this test
	// is asserting about.
	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"connect"}, condTTY, "github\n\nowner/repo\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("connect blocked with a terminal attached:\n%s", combined)
	}
	if !strings.Contains(combined, "Provider [") {
		t.Errorf("no provider picker at a terminal:\n%s", combined)
	}
	if !strings.Contains(combined, "Connecting to GitHub Issues...") {
		t.Errorf("the picked provider did not drive the guided path — github's banner never appeared:\n%s", combined)
	}
	if !strings.Contains(combined, "Repository (owner/repo): ") {
		t.Errorf("connect did not reach github's own field collection:\n%s", combined)
	}
}

// TestConnectProviderPickerOffersEveryProvider is AC-9's visible half: the
// rendered option list is the provider set, not a subset someone typed out.
func TestConnectProviderPickerOffersEveryProvider(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	_, stdout, stderr := runHero(t, bin, base, root,
		[]string{"connect"}, condTTY, "\n")
	combined := setupCombineStreams(stdout, stderr)

	want := "Provider [" + strings.Join(connectProviderOptions(), "|") + "]: "
	if !strings.Contains(combined, want) {
		t.Errorf("provider picker does not offer the full provider set.\nwant substring: %q\ngot:\n%s", want, combined)
	}
	// The empty answer fed above is not a default: connect has no provider it
	// could sensibly assume, so it reports the usage error instead.
	if strings.Contains(combined, "Connecting to") {
		t.Errorf("an empty answer selected a provider anyway:\n%s", combined)
	}
	if !strings.Contains(errorLine(combined), "usage: hero connect <type>") {
		t.Errorf("error line = %q, want the usage error for an unanswered picker", errorLine(combined))
	}
}

// TestConnectProviderOptionsComeFromTheProviderMap is AC-9's structural half.
//
// Risk 6 in the spec: a hardcoded picker list drifts from the registry the
// writer validates against, and nothing catches it until a user picks a
// provider connect then rejects. This pins both directions — the options are
// exactly the connectProviders keys, and every one of them is a provider
// internal/config also knows about, which is what writeConnection checks.
func TestConnectProviderOptionsComeFromTheProviderMap(t *testing.T) {
	want := make([]string, 0, len(connectProviders))
	for name := range connectProviders {
		want = append(want, name)
	}
	sort.Strings(want)

	got := connectProviderOptions()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("picker options = %v, want the connectProviders keys %v", got, want)
	}
	if len(got) == 0 {
		t.Fatal("picker offers nothing")
	}
	for _, provider := range got {
		// connectRoleOptions is non-empty only for a provider
		// config.ValidateProviderRole recognizes — the same check
		// writeConnection makes before persisting anything.
		if len(connectRoleOptions(provider)) == 0 {
			t.Errorf("picker offers %q, which the connect writer's provider registry does not recognize", provider)
		}
	}
}

// TestConnectProviderPickerStaysSilentWithoutATerminal is AC-3 for connect.
func TestConnectProviderPickerStaysSilentWithoutATerminal(t *testing.T) {
	const wantErr = "usage: hero connect <type>  (github, jira, linear, gitlab, confluence)"

	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newSanctionedWorkspace(t)

			exit, stdout, stderr := runHero(t, bin, base, root, []string{"connect"}, cond, "github\n")
			combined := setupCombineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("connect blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if strings.Contains(combined, "Provider [") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
			if !strings.Contains(errorLine(combined), wantErr) {
				t.Errorf("error line = %q, want the pre-existing usage error %q", errorLine(combined), wantErr)
			}
			// The picker must not have been answered off the stream either:
			// "github\n" was on it, and github's banner would prove it was read.
			if strings.Contains(combined, "Connecting to GitHub Issues...") {
				t.Errorf("connect took the provider off a non-terminal stream:\n%s", combined)
			}
		})
	}
}

// TestConnectProviderPickerRefusesUnderJSON is AC-7 for connect.
//
// It runs with a REAL terminal on fd 0. Under a pipe the picker would decline
// anyway, so the only thing that can suppress it here is the --json rule.
// Nothing is fed to the terminal: a picker that fired would block, and the
// timeout is the observable signature a programmatic caller would hit.
func TestConnectProviderPickerRefusesUnderJSON(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root, []string{"connect", "--json"}, condTTY, "")
	combined := setupCombineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("connect BLOCKED under --json with a terminal attached — it prompted:\n%s", combined)
	}
	if strings.Contains(combined, "Provider [") {
		t.Errorf("prompted under --json:\n%s", combined)
	}
	if exit == 0 {
		t.Errorf("exit = 0, want non-zero:\n%s", combined)
	}
}

// TestConnectProviderSuppliedDoesNotPrompt is AC-2 for connect: naming the
// provider must not add a question that did not exist before.
func TestConnectProviderSuppliedDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	_, stdout, stderr := runHero(t, bin, base, root,
		[]string{"connect", "github", "--integration-id", "gh-x", "--project", "owner/repo",
			"--local-only", "--no-verify", "--token-stdin"}, condPipe, "tkn\n")
	combined := setupCombineStreams(stdout, stderr)

	if strings.Contains(combined, "Provider [") {
		t.Errorf("a fully-specified connect emitted a provider picker:\n%s", combined)
	}
	if !strings.Contains(combined, "Connected integration gh-x (github)") {
		t.Errorf("the fully-specified connect did not complete:\n%s", combined)
	}
}

// TestConnectProviderPickerRejectsAnUnknownAnswer keeps a typo from being
// constructed into a bogus provider, the way promptTarget's unvalidated
// install.Target(input) once was.
func TestConnectProviderPickerRejectsAnUnknownAnswer(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root, []string{"connect"}, condTTY, "bitbucket\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0 for an unknown provider:\n%s", combined)
	}
	if !strings.Contains(combined, `invalid choice "bitbucket"`) {
		t.Errorf("the typo was not rejected at the picker:\n%s", combined)
	}
	if strings.Contains(combined, "Connecting to") {
		t.Errorf("connect proceeded with an unknown provider:\n%s", combined)
	}
}

// ---------------------------------------------------------------------------
// hero uninstall — target picker
// ---------------------------------------------------------------------------

// promptOptionsPattern extracts the option list prompt.Choice renders, e.g.
// the `a|b|c` from "Install target (default: opencode) [a|b|c]: ".
var promptOptionsPattern = regexp.MustCompile(`target[^\[]*\[([a-z|-]+)\]: `)

func optionsFromPrompt(t *testing.T, output string) []string {
	t.Helper()
	m := promptOptionsPattern.FindStringSubmatch(output)
	if m == nil {
		t.Fatalf("no target picker found in output:\n%s", output)
	}
	return strings.Split(m[1], "|")
}

// TestUninstallTargetPickerOffersAllSixTargets is AC-4 for uninstall.
func TestUninstallTargetPickerOffersAllSixTargets(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root, []string{"uninstall"}, condTTY, "claude\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("uninstall blocked at a terminal:\n%s", combined)
	}
	want := []string{"opencode", "cursor", "claude", "copilot", "codex", "generic"}
	got := optionsFromPrompt(t, combined)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("uninstall picker offers %v, want all six targets %v", got, want)
	}
	// And the answer must actually select: uninstalling claude from a
	// workspace with nothing installed reports that, and reports it for claude.
	if !strings.Contains(combined, "Nothing to remove.") {
		t.Errorf("the picked target did not drive the uninstall:\n%s", combined)
	}
}

// TestInstallAndUninstallPickersEnumerateIdenticalTargets is AC-5.
//
// Both lists are read out of the two commands' real rendered prompts rather
// than compared in source, because "the two commands offer the same set" is a
// claim about what a user sees. `hero uninstall` accepting four targets while
// `hero install` wrote six is the defect the harness-changes-cover-all-targets
// tripwire exists for, and it survived precisely because nothing compared the
// two surfaces.
func TestInstallAndUninstallPickersEnumerateIdenticalTargets(t *testing.T) {
	bin := baselineBinary(t)

	instBase, instRoot := newBareInstallTarget(t)
	_, instOut, instErr := runHero(t, bin, instBase, instRoot,
		[]string{"install", "project", "proj", "--dry-run"}, condTTY, "claude\n")
	installOptions := optionsFromPrompt(t, instOut+instErr)

	uninstBase, uninstRoot := newSanctionedWorkspace(t)
	_, uninstOut, uninstErr := runHero(t, bin, uninstBase, uninstRoot,
		[]string{"uninstall"}, condTTY, "claude\n")
	uninstallOptions := optionsFromPrompt(t, uninstOut+uninstErr)

	if strings.Join(installOptions, "|") != strings.Join(uninstallOptions, "|") {
		t.Errorf("install picker offers %v but uninstall picker offers %v — the two must be identical",
			installOptions, uninstallOptions)
	}
	if len(installOptions) != 6 {
		t.Errorf("the shared target list has %d entries, want 6: %v", len(installOptions), installOptions)
	}
}

// TestUninstallWithoutTargetStaysSilentWithoutATerminal is AC-3 for uninstall.
func TestUninstallWithoutTargetStaysSilentWithoutATerminal(t *testing.T) {
	const wantErr = "--target is required (opencode|cursor|claude|copilot|codex|generic)"

	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newSanctionedWorkspace(t)

			exit, stdout, stderr := runHero(t, bin, base, root, []string{"uninstall"}, cond, "claude\n")
			combined := setupCombineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("uninstall blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if strings.Contains(combined, "Uninstall target") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
			if errorLine(combined) != "Error: "+wantErr {
				t.Errorf("error line = %q, want the pre-existing %q", errorLine(combined), "Error: "+wantErr)
			}
		})
	}
}

// TestUninstallEmptyAnswerDoesNotPickATarget covers the one place this picker
// deliberately differs from install's.
//
// install takes opencode on an empty answer because installing is additive.
// Uninstall deletes files, so pressing enter must report the missing --target
// rather than stripping a harness the user never named.
func TestUninstallEmptyAnswerDoesNotPickATarget(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root, []string{"uninstall"}, condTTY, "\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("uninstall blocked at a terminal:\n%s", combined)
	}
	if exit == 0 {
		t.Errorf("exit = 0: an empty answer selected a target to uninstall\n%s", combined)
	}
	if strings.Contains(combined, "Nothing to remove.") || strings.Contains(combined, "Removed ") {
		t.Errorf("an empty answer ran an uninstall anyway:\n%s", combined)
	}
	if !strings.Contains(errorLine(combined), "--target is required") {
		t.Errorf("error line = %q, want the missing --target error", errorLine(combined))
	}
}

// TestUninstallWithTargetSuppliedDoesNotPrompt is AC-2 for uninstall.
func TestUninstallWithTargetSuppliedDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"uninstall", "--target", "claude"}, condTTY, "")
	combined := setupCombineStreams(stdout, stderr)

	if exit != 0 {
		t.Errorf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Uninstall target") {
		t.Errorf("a fully-specified uninstall emitted a picker:\n%s", combined)
	}
}

// ---------------------------------------------------------------------------
// hero admin repos add
// ---------------------------------------------------------------------------

// configuredRepos reads the repos map out of the workspace's hero.json.
//
// State, not log text: "Added peer-a → ../peer-a" is printed before the save
// is verified, so asserting on that line would pass against a build that never
// wrote the file.
func configuredRepos(t *testing.T, root string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, ".hero", "hero.json"))
	if err != nil {
		t.Fatalf("read hero.json: %v", err)
	}
	var shape struct {
		Repos map[string]any `json:"repos"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		t.Fatalf("parse hero.json: %v", err)
	}
	return shape.Repos
}

// TestReposAddAsksForBothValuesAtATerminal is AC-1 for repos add.
func TestReposAddAsksForBothValuesAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "repos", "add"}, condTTY, "peer-a\n../peer-a\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("repos add blocked at a terminal:\n%s", combined)
	}
	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	for _, label := range []string{"Repo alias: ", "Path to the repo: "} {
		if !strings.Contains(combined, label) {
			t.Errorf("missing prompt %q:\n%s", label, combined)
		}
	}
	repos := configuredRepos(t, root)
	if got := repos["peer-a"]; got != "../peer-a" {
		t.Errorf("hero.json repos[peer-a] = %v, want ../peer-a (the prompted answers were not persisted)", got)
	}
}

// TestReposAddAsksOnlyForTheMissingValue proves the prompt is additive at the
// level of the individual argument, not the whole invocation.
func TestReposAddAsksOnlyForTheMissingValue(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "repos", "add", "peer-b"}, condTTY, "../peer-b\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Repo alias: ") {
		t.Errorf("asked for an alias that was supplied on the command line:\n%s", combined)
	}
	if !strings.Contains(combined, "Path to the repo: ") {
		t.Errorf("did not ask for the missing path:\n%s", combined)
	}
	if got := configuredRepos(t, root)["peer-b"]; got != "../peer-b" {
		t.Errorf("hero.json repos[peer-b] = %v, want ../peer-b", got)
	}
}

// TestReposAddEmptyAnswerIsRejected pins the empty-answer branch: an alias is
// still required, and pressing enter must not register a repo under "".
func TestReposAddEmptyAnswerIsRejected(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "repos", "add"}, condTTY, "\n../peer-a\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0: an empty alias was accepted\n%s", combined)
	}
	if !strings.Contains(errorLine(combined), "alias is required") {
		t.Errorf("error line = %q, want \"alias is required\"", errorLine(combined))
	}
	if len(configuredRepos(t, root)) != 0 {
		t.Errorf("an empty alias was registered: %v", configuredRepos(t, root))
	}
}

// TestReposAddWithoutATerminalKeepsItsArgumentError is AC-3 for repos add.
//
// The error is cobra's own argument-count message, unchanged, and it must
// still arrive without the command having written anything.
func TestReposAddWithoutATerminalKeepsItsArgumentError(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no arguments", []string{"admin", "repos", "add"}, "Error: accepts 2 arg(s), received 0"},
		{"one argument", []string{"admin", "repos", "add", "peer-a"}, "Error: accepts 2 arg(s), received 1"},
	}
	for _, tc := range cases {
		for _, cond := range []string{condPipe, condClosed} {
			t.Run(tc.name+"/"+cond, func(t *testing.T) {
				bin := baselineBinary(t)
				base, root := newSanctionedWorkspace(t)

				exit, stdout, stderr := runHero(t, bin, base, root, tc.args, cond, "peer-a\n../peer-a\n")
				combined := setupCombineStreams(stdout, stderr)

				if exit == -1 {
					t.Fatalf("repos add blocked on a non-terminal — hard constraint 3:\n%s", combined)
				}
				if exit == 0 {
					t.Errorf("exit = 0, want non-zero:\n%s", combined)
				}
				if errorLine(combined) != tc.wantErr {
					t.Errorf("error line = %q, want the pre-existing %q", errorLine(combined), tc.wantErr)
				}
				if strings.Contains(combined, "Repo alias: ") || strings.Contains(combined, "Path to the repo: ") {
					t.Errorf("prompted on a non-terminal:\n%s", combined)
				}
				if len(configuredRepos(t, root)) != 0 {
					t.Errorf("a rejected repos add wrote to hero.json: %v", configuredRepos(t, root))
				}
			})
		}
	}
}

// TestReposAddWithBothArgumentsDoesNotPrompt is AC-2 for repos add.
func TestReposAddWithBothArgumentsDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "repos", "add", "peer-c", "../peer-c"}, condTTY, "")
	combined := setupCombineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Repo alias: ") || strings.Contains(combined, "Path to the repo: ") {
		t.Errorf("a fully-specified repos add prompted:\n%s", combined)
	}
	if got := configuredRepos(t, root)["peer-c"]; got != "../peer-c" {
		t.Errorf("hero.json repos[peer-c] = %v, want ../peer-c", got)
	}
}

// TestReposAddRejectsExtraArgumentsEvenAtATerminal pins the half of the
// relaxation that is NOT relaxed. A prompt can supply a missing argument; it
// can do nothing about a surplus one, and silently dropping it is how a typo
// becomes a wrong registration.
func TestReposAddRejectsExtraArgumentsEvenAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "repos", "add", "peer-d", "../peer-d", "extra"}, condTTY, "")
	combined := setupCombineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0: three arguments were accepted\n%s", combined)
	}
	if errorLine(combined) != "Error: accepts 2 arg(s), received 3" {
		t.Errorf("error line = %q, want cobra's argument-count error", errorLine(combined))
	}
	if len(configuredRepos(t, root)) != 0 {
		t.Errorf("an over-specified repos add wrote to hero.json: %v", configuredRepos(t, root))
	}
}

// ---------------------------------------------------------------------------
// hero admin users add
// ---------------------------------------------------------------------------

// listedUsers runs `hero admin users` and returns its output, which is the
// only read-back the local job queue exposes.
func listedUsers(t *testing.T, bin, base, root string) string {
	t.Helper()
	_, stdout, stderr := runHero(t, bin, base, root, []string{"admin", "users"}, condPipe, "")
	return setupCombineStreams(stdout, stderr)
}

// TestUsersAddAsksForTheUsernameAtATerminal is AC-1 for users add.
//
// --password is supplied so the test asserts about the username prompt rather
// than about prompt.Secret, which reads /dev/tty and is unavailable to a test
// child by design. The refusal path is asserted separately below.
func TestUsersAddAsksForTheUsernameAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "users", "add", "--password", "hunter2", "--email", "alice@example.com"},
		condTTY, "alice\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("users add blocked at a terminal:\n%s", combined)
	}
	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if !strings.Contains(combined, "Username: ") {
		t.Errorf("no username prompt at a terminal:\n%s", combined)
	}
	if listed := listedUsers(t, bin, base, root); !strings.Contains(listed, "alice") {
		t.Errorf("the prompted username was not persisted; `hero admin users` shows:\n%s", listed)
	}
}

// TestUsersAddEmptyAnswerIsRejected pins the empty-answer branch. Without it
// the queue would be asked to create a user with an empty username.
func TestUsersAddEmptyAnswerIsRejected(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "users", "add", "--password", "hunter2"}, condTTY, "\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0: an empty username was accepted\n%s", combined)
	}
	if !strings.Contains(errorLine(combined), "username is required") {
		t.Errorf("error line = %q, want \"username is required\"", errorLine(combined))
	}
	if listed := listedUsers(t, bin, base, root); !strings.Contains(listed, "No users.") {
		t.Errorf("a user was created from an empty answer:\n%s", listed)
	}
}

// TestUsersAddWithoutATerminalKeepsItsArgumentError is AC-3 for users add.
func TestUsersAddWithoutATerminalKeepsItsArgumentError(t *testing.T) {
	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newSanctionedWorkspace(t)

			exit, stdout, stderr := runHero(t, bin, base, root,
				[]string{"admin", "users", "add", "--password", "hunter2"}, cond, "alice\n")
			combined := setupCombineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("users add blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if errorLine(combined) != "Error: accepts 1 arg(s), received 0" {
				t.Errorf("error line = %q, want the pre-existing argument error", errorLine(combined))
			}
			if strings.Contains(combined, "Username: ") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
			if listed := listedUsers(t, bin, base, root); strings.Contains(listed, "alice") {
				t.Errorf("the username was taken off a non-terminal stream and a user was created:\n%s", listed)
			}
		})
	}
}

// TestUsersAddRefusesToReadThePasswordFromAStream is AC-6.
//
// The username is supplied so the invocation reaches the password read, and
// the pipe carries a password. The command must refuse rather than take it,
// and no user may exist afterwards.
func TestUsersAddRefusesToReadThePasswordFromAStream(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "users", "add", "bob"}, condPipe, "hunter2\nhunter2\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0: a password was accepted from a pipe\n%s", combined)
	}
	if !mentionsTerminalRefusal(combined) {
		t.Errorf("the refusal does not explain that a terminal is required:\n%s", combined)
	}
	if !strings.Contains(combined, "--password") {
		t.Errorf("the refusal does not name the non-interactive alternative:\n%s", combined)
	}
	if listed := listedUsers(t, bin, base, root); strings.Contains(listed, "bob") {
		t.Errorf("the user was created despite the refused password:\n%s", listed)
	}
}

// TestUsersAddWithUsernameSuppliedDoesNotPrompt is AC-2 for users add.
func TestUsersAddWithUsernameSuppliedDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "users", "add", "carol", "--password", "hunter2"}, condTTY, "")
	combined := setupCombineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Username: ") {
		t.Errorf("a fully-specified users add prompted for the username:\n%s", combined)
	}
}

// ---------------------------------------------------------------------------
// hero trust
// ---------------------------------------------------------------------------

// TestTrustAsksForTheTargetAtATerminal is AC-1 for trust.
func TestTrustAsksForTheTargetAtATerminal(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root, []string{"trust"}, condTTY, "codex\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == -1 {
		t.Fatalf("trust blocked at a terminal:\n%s", combined)
	}
	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if !strings.Contains(combined, "Trust target [codex|claude]: ") {
		t.Errorf("no trust target picker at a terminal:\n%s", combined)
	}
	// The answer must select: only codex prints the Codex hint.
	if !strings.Contains(combined, "Codex permissions: optional one-time setup") {
		t.Errorf("the picked target did not drive the command:\n%s", combined)
	}
}

// TestTrustClaudeFromThePickerAppliesTheAllowlist proves the picked answer
// reaches the branch that changes state, not just the one that prints.
func TestTrustClaudeFromThePickerAppliesTheAllowlist(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root, []string{"trust"}, condTTY, "claude\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	settings, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("claude settings were never written from the picked target: %v\n%s", err, combined)
	}
	if !strings.Contains(string(settings), "Bash(hero:*)") {
		t.Errorf("settings do not carry the hero allowlist entry:\n%s", settings)
	}
}

// TestTrustEmptyAnswerIsRejected pins the empty-answer branch.
//
// The two targets do different things — codex prints instructions, claude
// edits a settings file — so an empty answer must not fall back to either.
// This case was found by falsification: an earlier round mutated the branch to
// default to codex and the whole suite stayed green.
func TestTrustEmptyAnswerIsRejected(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root, []string{"trust"}, condTTY, "\n")
	combined := setupCombineStreams(stdout, stderr)

	if exit == 0 {
		t.Errorf("exit = 0: an empty answer selected a trust target\n%s", combined)
	}
	if strings.Contains(combined, "Codex permissions") {
		t.Errorf("an empty answer fell back to codex:\n%s", combined)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "settings.json")); err == nil {
		t.Errorf("an empty answer fell back to claude and wrote settings.json")
	}
	if !strings.Contains(errorLine(combined), "trust target is required") {
		t.Errorf("error line = %q, want \"trust target is required\"", errorLine(combined))
	}
}

// TestTrustWithoutATerminalKeepsItsArgumentError is AC-3 for trust.
func TestTrustWithoutATerminalKeepsItsArgumentError(t *testing.T) {
	for _, cond := range []string{condPipe, condClosed} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newSanctionedWorkspace(t)

			exit, stdout, stderr := runHero(t, bin, base, root, []string{"trust"}, cond, "codex\n")
			combined := setupCombineStreams(stdout, stderr)

			if exit == -1 {
				t.Fatalf("trust blocked on a non-terminal — hard constraint 3:\n%s", combined)
			}
			if exit == 0 {
				t.Errorf("exit = 0, want non-zero:\n%s", combined)
			}
			if errorLine(combined) != "Error: accepts between 1 and 2 arg(s), received 0" {
				t.Errorf("error line = %q, want the pre-existing argument error", errorLine(combined))
			}
			if strings.Contains(combined, "Trust target") {
				t.Errorf("prompted on a non-terminal:\n%s", combined)
			}
			if strings.Contains(combined, "Codex permissions") {
				t.Errorf("trust took its target off a non-terminal stream:\n%s", combined)
			}
		})
	}
}

// TestTrustWithTargetSuppliedDoesNotPrompt is AC-2 for trust.
func TestTrustWithTargetSuppliedDoesNotPrompt(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exit, stdout, stderr := runHero(t, bin, base, root, []string{"trust", "codex"}, condTTY, "")
	combined := setupCombineStreams(stdout, stderr)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", exit, combined)
	}
	if strings.Contains(combined, "Trust target") {
		t.Errorf("a fully-specified trust prompted:\n%s", combined)
	}
}

// ---------------------------------------------------------------------------
// the shared argument gate
// ---------------------------------------------------------------------------

// TestSetupCommandsCallThePrimitivesDirectly is AC-8, and the standing guard
// on the initiative's scope cap.
//
// The scoped successor rejected a generic promptfield abstraction. Connect
// owns its private collector; every other setup command has flat fields and
// calls the prompt primitives directly.
func TestSetupCommandsCallThePrimitivesDirectly(t *testing.T) {
	adopters := []string{"install.go", "uninstall.go", "repos.go", "users.go", "trust.go"}
	primitive := regexp.MustCompile(`prompt\.(Prompt|Choice|Secret|Confirm)\(`)

	for _, name := range append(adopters, "prompt_args.go") {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		body := stripComments(string(src))
		for _, banned := range []string{"collectFields", "promptField", "fieldReader"} {
			if strings.Contains(body, banned) {
				t.Errorf("%s references %s — these commands have flat fields and must call the "+
					"prompt primitives directly; the generic descriptor is out of scope.", name, banned)
			}
		}
	}
	for _, name := range adopters {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !primitive.MatchString(stripComments(string(src))) {
			t.Errorf("%s no longer calls a prompt primitive directly", name)
		}
	}
}

// TestPromptableArgsRelaxesOnlyAShortfallAtATerminal is the unit-level pin on
// prompt_args.go.
//
// The three subprocess suites above each prove their own command's wiring.
// This proves the rule those wirings share, including the case none of them
// can reach cheaply: a surplus argument must still fail at a terminal.
func TestPromptableArgsRelaxesOnlyAShortfallAtATerminal(t *testing.T) {
	_, slave, _ := openCapturedPTY(t)

	terminal := &cobra.Command{}
	terminal.SetIn(slave)
	pipe := &cobra.Command{}
	pipe.SetIn(strings.NewReader(""))

	rule := promptableArgs(2, cobra.ExactArgs(2))

	cases := []struct {
		name    string
		cmd     *cobra.Command
		args    []string
		wantErr bool
	}{
		{"shortfall at a terminal is relaxed", terminal, []string{}, false},
		{"partial shortfall at a terminal is relaxed", terminal, []string{"a"}, false},
		{"exact count at a terminal passes", terminal, []string{"a", "b"}, false},
		{"surplus at a terminal still fails", terminal, []string{"a", "b", "c"}, true},
		{"shortfall on a pipe still fails", pipe, []string{}, true},
		{"partial shortfall on a pipe still fails", pipe, []string{"a"}, true},
		{"exact count on a pipe passes", pipe, []string{"a", "b"}, false},
		{"surplus on a pipe fails", pipe, []string{"a", "b", "c"}, true},
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
