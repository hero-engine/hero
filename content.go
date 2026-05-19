// Package hero provides embedded access to Hero's bundled content files
// (agents, commands, skills). These are compiled into the binary so that
// hero install and hero upgrade work without access to the source tree.
//
// Content is organized by domain under domains/<name>/. The default domain
// is "engineering". Other domains (e.g. "pm", "sales") provide alternative
// agents, commands, and skills for non-engineering workflows.
package hero

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed domains/engineering/agents domains/engineering/commands domains/engineering/skills domains/engineering/spec-types
var engineeringContent embed.FS

//go:embed domains/sales/agents domains/sales/commands domains/sales/skills domains/sales/spec-types
var salesContent embed.FS

//go:embed domains/pm/agents domains/pm/commands domains/pm/skills domains/pm/spec-types
var pmContent embed.FS

//go:embed core/agents core/commands core/skills
var coreContent embed.FS

//go:embed core/vocabularies
var coreVocabularies embed.FS

//go:embed core/methodologies
var coreMethodologies embed.FS

//go:embed core/spec-types
var coreSpecTypes embed.FS

// legacyContent embeds the root-level agents/, commands/, and skills/
// directories. Kept as permanent backward-compat: ContentFS() returns
// this filesystem so callers that pre-date the domain-pack architecture
// (and any project whose hero.json has no "domain" key) keep getting
// content. The root dirs also remain the actively-maintained source
// for the engineering vertical today — domains/engineering/ is a
// scaffolded mirror that is allowed to be incomplete.
//
// B1 decision (2026-05-17, pm-foundation-delivery sprint): leave this
// legacy fallback in place rather than cutting ContentFS() over to
// domains/engineering/. The cutover requires first syncing root →
// domains/engineering/ bit-for-bit, which is out of B1 scope (B1 wires
// PM into the embed surface; it does not migrate engineering content).
// See .hero/planning/features/domain-plugin-architecture/spec.md for
// the full decision record.
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
	if domain == "pm" {
		return fs.Sub(pmContent, "domains/pm")
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

// CoreMethodologiesFS returns a read-only filesystem rooted at
// core/methodologies/ containing the bundled methodology profile YAML
// files (scrum, kanban, shape-up, waterfall, scrumban). Consumed by
// internal/methodology at startup.
func CoreMethodologiesFS() fs.FS {
	sub, err := fs.Sub(coreMethodologies, "core/methodologies")
	if err != nil {
		return coreMethodologies
	}
	return sub
}

// CoreSpecTypesFS returns a read-only filesystem rooted at
// core/spec-types/ containing the canonical work-tracking spec-type
// declarations (initiative, prd, epic, feature, bug, chore, intake,
// release, sprint). Consumed by internal/spectypes at startup.
func CoreSpecTypesFS() fs.FS {
	sub, err := fs.Sub(coreSpecTypes, "core/spec-types")
	if err != nil {
		return coreSpecTypes
	}
	return sub
}

// DomainSpecTypesFS returns a read-only filesystem rooted at
// domains/<domain>/spec-types/ for the active domain. Returns nil if
// the domain has no spec-types extensions. Consumed by
// internal/spectypes to overlay domain-led types (e.g. engineering's
// decision and convention) on top of the core nine.
func DomainSpecTypesFS(domain string) fs.FS {
	if domain == "" {
		domain = "engineering"
	}
	var src embed.FS
	switch domain {
	case "engineering":
		src = engineeringContent
	case "pm":
		src = pmContent
	case "sales":
		src = salesContent
	default:
		return nil
	}
	sub, err := fs.Sub(src, "domains/"+domain+"/spec-types")
	if err != nil {
		return nil
	}
	// Validate the directory actually exists in the embed.
	if _, err := fs.ReadDir(sub, "."); err != nil {
		return nil
	}
	return sub
}

// OverlayFS returns an fs.FS where lookups try top first and fall back
// to bottom. ReadDir merges entries from both, with top's entry winning
// on name collisions. Used by the install pipeline to layer the active
// domain over the universal core: every install renders core +
// active-domain merged, with the domain overriding core on file-level
// path conflicts.
//
// The returned FS implements fs.FS, fs.ReadDirFS, fs.StatFS, and
// fs.ReadFileFS. Either input may be nil; nil is treated as an empty
// FS. The semantics mirror internal/spectypes/loader.go's "core first,
// domain overlays" precedence — diverging only in how the overlay
// happens (FS overlay here vs registry overlay there).
func OverlayFS(top, bottom fs.FS) fs.FS {
	return &overlayFS{top: top, bottom: bottom}
}

type overlayFS struct {
	top    fs.FS
	bottom fs.FS
}

func (o *overlayFS) Open(name string) (fs.File, error) {
	if o.top != nil {
		if f, err := o.top.Open(name); err == nil {
			return f, nil
		}
	}
	if o.bottom != nil {
		return o.bottom.Open(name)
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func (o *overlayFS) Stat(name string) (fs.FileInfo, error) {
	if o.top != nil {
		if fi, err := fs.Stat(o.top, name); err == nil {
			return fi, nil
		}
	}
	if o.bottom != nil {
		return fs.Stat(o.bottom, name)
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

func (o *overlayFS) ReadFile(name string) ([]byte, error) {
	if o.top != nil {
		if data, err := fs.ReadFile(o.top, name); err == nil {
			return data, nil
		}
	}
	if o.bottom != nil {
		return fs.ReadFile(o.bottom, name)
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func (o *overlayFS) ReadDir(name string) ([]fs.DirEntry, error) {
	var topEntries, bottomEntries []fs.DirEntry
	var topErr, bottomErr error

	if o.top != nil {
		topEntries, topErr = fs.ReadDir(o.top, name)
	} else {
		topErr = fs.ErrNotExist
	}
	if o.bottom != nil {
		bottomEntries, bottomErr = fs.ReadDir(o.bottom, name)
	} else {
		bottomErr = fs.ErrNotExist
	}

	// If both sides missing, return an error.
	if topErr != nil && bottomErr != nil {
		// Prefer top's error so callers see the upper-layer reason.
		return nil, topErr
	}

	seen := make(map[string]bool, len(topEntries)+len(bottomEntries))
	out := make([]fs.DirEntry, 0, len(topEntries)+len(bottomEntries))
	for _, e := range topEntries {
		if seen[e.Name()] {
			continue
		}
		seen[e.Name()] = true
		out = append(out, e)
	}
	for _, e := range bottomEntries {
		if seen[e.Name()] {
			continue
		}
		seen[e.Name()] = true
		out = append(out, e)
	}
	// Stable order: sort alphabetically so ReadDir output is deterministic
	// across runs regardless of which side contributed which entries.
	sortDirEntries(out)
	return out, nil
}

// sortDirEntries sorts fs.DirEntry slices by name in ascending order.
// Stable, alphabetical order keeps install diffs deterministic across
// runs regardless of which overlay side contributed which entries.
func sortDirEntries(entries []fs.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
}

// AvailableDomains returns the list of embedded domain names.
// Engineering is the canonical, populated vertical. PM is the second
// populated vertical (intake/PRD/discovery flow). Sales is a scaffold
// today (mission + structure but no real content yet).
func AvailableDomains() []string {
	return []string{"engineering", "sales", "pm"}
}
