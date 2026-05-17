// Package hero provides embedded access to Hero's bundled content files
// (agents, commands, skills). These are compiled into the binary so that
// hero install and hero upgrade work without access to the source tree.
//
// Content is organized by domain under domains/<name>/. The default domain
// is "engineering". Other domains (e.g. "sales") provide alternative
// agents, commands, and skills for non-engineering workflows.
package hero

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed domains/engineering/agents domains/engineering/commands domains/engineering/skills
var engineeringContent embed.FS

//go:embed domains/sales/agents domains/sales/commands domains/sales/skills domains/sales/spec-types
var salesContent embed.FS

//go:embed core/agents core/commands core/skills
var coreContent embed.FS

//go:embed core/vocabularies
var coreVocabularies embed.FS

// legacyContent embeds the root-level agents/, commands/, and skills/
// directories. Kept as permanent backward-compat: ContentFS() returns
// this filesystem so callers that pre-date the domain-pack architecture
// (and any project whose hero.json has no "domain" key) keep getting
// content. The root dirs also remain the actively-maintained source
// for the engineering vertical today — domains/engineering/ is a
// scaffolded mirror that is allowed to be incomplete. Removing this
// embed is deferred to a follow-up that syncs root → domains/engineering/
// and switches the engineering default over.
//
//go:embed agents commands skills
var legacyContent embed.FS

// ContentFS returns a read-only filesystem for the default (engineering)
// vertical. Returns the legacy root-level agents/commands/skills — see
// the legacyContent comment for why that's still the canonical source.
func ContentFS() fs.FS {
	return legacyContent
}

// DomainFS returns a read-only filesystem for the specified domain.
// The returned FS has agents/, commands/, and skills/ at its root.
//
// For "engineering" (and the empty default), legacyContent is the
// canonical source today; the domains/engineering/ embed is reserved
// for the eventual switchover but isn't authoritative yet.
func DomainFS(domain string) (fs.FS, error) {
	if domain == "" || domain == "engineering" {
		return legacyContent, nil
	}
	if domain == "sales" {
		return fs.Sub(salesContent, "domains/sales")
	}
	return nil, fmt.Errorf("domain %q not found — available domains: %v", domain, AvailableDomains())
}

// CoreFS returns a read-only filesystem for the universal core layer
// (agents/commands/skills that serve any vertical). Verticals layer on
// top of this; consumers that want the full active surface should
// merge CoreFS with the active DomainFS.
func CoreFS() fs.FS {
	sub, err := fs.Sub(coreContent, "core")
	if err != nil {
		return coreContent
	}
	return sub
}

// CoreVocabulariesFS returns a read-only filesystem rooted at
// core/vocabularies/ containing the bundled vocabulary preset YAML
// files (default, agile-scrum, shape-up, kanban, jira, linear).
// Consumed by internal/vocabulary at startup.
func CoreVocabulariesFS() fs.FS {
	sub, err := fs.Sub(coreVocabularies, "core/vocabularies")
	if err != nil {
		return coreVocabularies
	}
	return sub
}

// AvailableDomains returns the list of embedded domain names.
// Engineering is the canonical, populated vertical. Sales is a
// scaffold today (mission + structure but no real content yet).
func AvailableDomains() []string {
	return []string{"engineering", "sales"}
}
