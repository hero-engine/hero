// Package managed owns the single Hero-managed region inside user-owned
// files (AGENTS.md, CLAUDE.md, NEXT.md, and future harness instruction
// files).
//
// Two concerns live here:
//
//  1. The low-level marker primitives (FindManagedRegion,
//     RenderManagedRegion, InsertManagedRegion, IsLegacyHeroStub) that
//     wrap content with the versioned `<!-- hero:managed-start v=X -->`
//     / `<!-- hero:managed-end -->` markers and splice the region into
//     a file while preserving user content outside the markers
//     byte-for-byte.
//
//  2. The Region orchestrator that aggregates an ordered list of
//     SectionContributor implementations into a single managed region
//     per file. Callers in internal/install and internal/snapshot
//     compose a Region with the contributors they need and call
//     Region.Write — there is no global registry.
//
// Region.Write also runs an inline one-shot migration that detects the
// legacy two-block layout (install marker pair + separate snapshot
// pointer marker pair) and consolidates it into the new single-block
// form. The migration only mutates byte ranges inside known marker
// pairs; user content outside any marker pair is never touched.
//
// Dependency direction: this package depends on neither internal/install
// nor internal/snapshot. Those packages depend on this one.
package managed
