package graph

import "github.com/hero-engine/hero/internal/config"

// DomainScope captures how a single CLI / MCP / dashboard call wants
// to interact with the domain partition. One value per call, resolved
// once at the entry point and threaded into every graph query the
// call needs.
//
// See domain-scoped-knowledge-graph/spec.md's "Active-domain
// resolution" section for the precedence chain (override → cfg.Domain
// → engineering fallback) and the per-query stance table.
type DomainScope struct {
	// Active is the resolved active domain. Empty means "no per-
	// domain filter intended" — only the global allow-list types
	// (Mission, Person, …) should have an empty Domain at rest, so
	// in practice Active is always a domain string when filtering.
	Active string

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
//  2. Workspace config — cfg.Domain (from hero.json).
//  3. Hardcoded fallback "engineering". Covers pre-migration
//     workspaces with no domain key.
//
// An empty `override` string means "no override". Pass "*" to widen
// to all domains.
func ResolveDomain(cfg config.Config, override string) DomainScope {
	if override == "*" {
		return DomainScope{AllDomains: true}
	}
	if override != "" {
		return DomainScope{Active: override}
	}
	if cfg.Domain != "" {
		return DomainScope{Active: cfg.Domain}
	}
	return DomainScope{Active: "engineering"}
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
	return col + " = ?", []any{d.Active}
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
	return domain == d.Active
}
