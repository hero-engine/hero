package spec

import (
	"os/exec"
	"strings"
)

// KnownSigners returns the identities allowed to sign off ledger rows: every
// git commit author's name, email, and email user in projectRoot's history,
// plus the configured hero.json ledger.signers entries (with the email user
// of any email entry). `hero spec verify` Gate 1 and the read contract's
// verify state both resolve sign-offs against this one set.
func KnownSigners(projectRoot string, configured []string) map[string]bool {
	known := map[string]bool{}
	add := func(id string) {
		id = NormalizeSigner(id)
		if id == "" {
			return
		}
		known[id] = true
		if user, _, ok := strings.Cut(id, "@"); ok && user != "" {
			known[user] = true
		}
	}
	for _, id := range configured {
		add(id)
	}
	if out, err := exec.Command("git", "-C", projectRoot, "log", "--format=%an%n%ae").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			add(line)
		}
	}
	return known
}
