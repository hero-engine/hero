// Package edition resolves the active Hero edition from the HERO_EDITION
// environment variable and exposes a small set of helpers the shell uses
// to gate routes and top-nav tabs per the deployment-and-rendering
// decision (hero-surface-deployment-and-rendering).
//
// Editions: local | team | cloud | enterprise | ce
//
// Default is "local". Unknown values resolve to "local" with no error so
// a typo never breaks a developer's machine.
package edition

import (
	"net/http"
	"os"
	"strings"
)

// Edition names the active runtime edition.
type Edition string

const (
	Local      Edition = "local"
	Team       Edition = "team"
	Cloud      Edition = "cloud"
	Enterprise Edition = "enterprise"
	CE         Edition = "ce"
)

// All returns every recognized edition in canonical order.
func All() []Edition {
	return []Edition{Local, Team, Cloud, Enterprise, CE}
}

// Resolve reads HERO_EDITION and returns the recognized edition. Unknown
// or empty values resolve to Local. Comparison is case-insensitive.
func Resolve() Edition {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("HERO_EDITION"))) {
	case "team":
		return Team
	case "cloud":
		return Cloud
	case "enterprise":
		return Enterprise
	case "ce":
		return CE
	default:
		return Local
	}
}

// Allowed reports whether the given edition is included in the allowed
// set. An empty allowed slice means "all editions" — used by callers
// (homes, routes) that don't restrict themselves.
func Allowed(active Edition, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	a := strings.ToLower(string(active))
	for _, e := range allowed {
		if strings.ToLower(strings.TrimSpace(e)) == a {
			return true
		}
	}
	return false
}

// Require returns middleware that 404s if the active edition is not in
// the given set. Use it on routes that need a finer gate than a Home's
// own edition filter.
func Require(editions ...string) func(http.Handler) http.Handler {
	active := Resolve()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !Allowed(active, editions) {
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
