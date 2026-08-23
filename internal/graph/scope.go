package graph

import (
	"strings"

	"github.com/hero-engine/hero/internal/config"
)

// DomainScope captures how a single CLI / MCP / dashboard call wants
// to interact with the domain partition. One value per call, resolved
// once at the entry point and threaded into every graph query the
// call needs.
//
// See domain-scoped-knowledge-graph/spec.md's "Active-domain
// resolution" section for the precedence chain (override → cfg.ResolveDomains
// → engineering fallback) and the per-query stance table.
type DomainScope struct {
	// Active is the resolved active domain. Empty means "no per-
	// domain filter intended" — only the global allow-list types
	// (Mission, Person, …) should have an empty Domain at rest, so
	// in practice Active is always a domain string when filtering.
	Active string

	// Enabled is the default workspace retrieval set: primary followed by
	// enabled extensions. It is empty for an explicit single-domain scope.
	Enabled []string

	// Focused is optional, ephemeral client state used only for ranking.
	// It never changes the committed primary domain or write ownership.
	Focused string

	// AllDomains is true when the caller passed --all-domains (CLI)
	// or domain="*" (MCP). When true, Where returns an empty
	// fragment so the query runs without partition filtering.
	AllDomains bool
}

// ResolveDomain returns the effective scope for one call.
//
// Precedence:
//  1. Explicit override (--domain <name> on CLI; domain field in MCP
//     args; ?domain= query string on dashboard). The literal "*"
//     means AllDomains.
//  2. Workspace config — cfg.ResolveDomains (canonical composition or legacy scalar).
//  3. Hardcoded fallback "engineering". Covers pre-migration
//     workspaces with no domain key.
//
// An empty `override` string means "no override". Pass "*" to widen
// to all domains.
func ResolveDomain(cfg config.Config, override string) DomainScope {
	return ResolveDomainFocused(cfg, override, "")
}

// ResolveDomainFocused is ResolveDomain with an optional enabled-domain focus
// hint. Explicit domain and all-domain overrides retain their existing scope.
func ResolveDomainFocused(cfg config.Config, override, focused string) DomainScope {
	if override == "*" {
		return DomainScope{AllDomains: true}
	}
	if override != "" {
		return DomainScope{Active: override}
	}
	resolved, err := cfg.ResolveDomains()
	if err != nil {
		return DomainScope{Active: "engineering"}
	}
	enabled := make([]string, 0, 2+len(resolved.Extensions))
	enabled = append(enabled, string(resolved.Primary))
	for _, extension := range resolved.Extensions {
		enabled = append(enabled, string(extension))
	}
	enabled = append(enabled, "core")
	if !containsDomain(enabled, focused) {
		focused = ""
	}
	return DomainScope{Active: string(resolved.Primary), Enabled: enabled, Focused: focused}
}

// Where returns a SQL WHERE clause fragment and the arg slice
// suitable for appending into existing queries against the `nodes`
// or `edges` table. Returns ("", nil) when AllDomains is true.
//
// The fragment is bare — caller is responsible for combining with
// `AND` or wrapping in parens. tableAlias may be empty (column
// `domain` is used directly) or a table alias (e.g. "n" → "n.domain").
//
// The fragment uses a placeholder; the returned arg slice carries the
// matching value. Callers should append both to their existing query.
func (d DomainScope) Where(tableAlias string) (string, []any) {
	if d.AllDomains {
		return "", nil
	}
	col := "domain"
	if tableAlias != "" {
		col = tableAlias + ".domain"
	}
	if len(d.Enabled) == 0 {
		return col + " = ?", []any{d.Active}
	}
	placeholders := make([]string, len(d.Enabled))
	args := make([]any, len(d.Enabled))
	for i, domain := range d.Enabled {
		placeholders[i] = "?"
		args[i] = domain
	}
	return col + " IN (" + strings.Join(placeholders, ",") + ")", args
}

// Match reports whether a row with the given Domain should be
// included by this scope. Used by callers that already have the
// row in hand and need to filter post-query (e.g. graph walks
// that loaded a candidate set in one query and now filter by
// domain in Go).
func (d DomainScope) Match(domain string) bool {
	if d.AllDomains {
		return true
	}
	if len(d.Enabled) == 0 {
		return domain == d.Active
	}
	return containsDomain(d.Enabled, domain)
}

// Rank returns a stable lower-is-better domain rank for workspace results:
// focused, primary, then declared extensions. Domains outside the scope rank
// after the enabled stack.
func (d DomainScope) Rank(domain string) int {
	order := make([]string, 0, 2+len(d.Enabled))
	if d.Focused != "" {
		order = append(order, d.Focused)
	}
	if !containsDomain(order, d.Active) {
		order = append(order, d.Active)
	}
	for _, enabled := range d.Enabled {
		if !containsDomain(order, enabled) {
			order = append(order, enabled)
		}
	}
	for i, candidate := range order {
		if domain == candidate {
			return i
		}
	}
	return len(order)
}

func containsDomain(domains []string, want string) bool {
	if want == "" {
		return false
	}
	for _, domain := range domains {
		if domain == want {
			return true
		}
	}
	return false
}
