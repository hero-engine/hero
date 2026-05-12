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

//go:embed agents commands skills
var legacyContent embed.FS

// ContentFS returns a read-only filesystem for the default (engineering) domain.
// For backward compatibility, this returns the root-level agents/commands/skills.
func ContentFS() fs.FS {
	return legacyContent
}

// DomainFS returns a read-only filesystem for the specified domain.
// The returned FS has agents/, commands/, and skills/ at its root.
func DomainFS(domain string) (fs.FS, error) {
	if domain == "" || domain == "engineering" {
		sub, err := fs.Sub(engineeringContent, "domains/engineering")
		if err != nil {
			return legacyContent, nil
		}
		return sub, nil
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

// AvailableDomains returns the list of embedded domain names.
// Engineering is the canonical, populated vertical. Sales is a
// scaffold today (mission + structure but no real content yet).
func AvailableDomains() []string {
	return []string{"engineering", "sales"}
}
