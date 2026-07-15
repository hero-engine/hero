package install

import "github.com/hero-engine/hero/internal/managed"

// operational_guidance.go — the shared, domain-agnostic "Hero Binary & MCP
// Surface" section.
//
// The version/schema confusion this guidance defends against is domain- and
// harness-agnostic: any agent in any pack that shells a stale `hero` and
// reads a newer graph can confabulate a migration story. It concerns the
// hero binary and MCP surface, nothing pack-specific, so it is authored once
// here and rendered into every domain's managed region via defaultSections —
// rather than living inside a single pack body (its original, engineering-
// only home in generateEngineeringAgentsMdBody). Modeled on
// internal/snapshot/pointers.go NewPointerSection: the orchestrator owns the
// H2 heading (emitted from SectionTitle), the section owns only the body.

// operationalGuidanceSectionID is the stable SectionContributor identifier
// for the shared operational guidance. Stable across releases so the
// managed-region differ tracks the section.
const operationalGuidanceSectionID = "install:hero-operational-guidance"

// heroOperationalGuidance is the verbatim routing paragraph (originally
// shipped in v0.25.0 inside the engineering pack body): prefer the in-process
// MCP surface over a bare shelled-out `hero`, and on any schema/version
// mismatch run `hero doctor` — NOT `hero upgrade` — instead of inventing a
// migration narrative.
const heroOperationalGuidance = "**Prefer Hero's MCP tools over shelling out to a bare `hero` in a terminal.** A GUI-launched harness can resolve a *different or stale* `hero` binary on its PATH than your login shell does; the MCP surface is the in-process Hero you're already connected to, so it can't drift out from under you. When you must use the CLI and hit a schema/version mismatch or a confusing `hero` version error, **run `hero doctor` and act on its output** — it reports which binary is actually on PATH, its schema, the graph's schema, and the real remediation. Do NOT invent a schema-migration narrative, and do NOT run `hero upgrade` to \"fix schema\": `hero upgrade` updates workspace files, not the binary, so it cannot fix a wrong-binary-on-PATH situation.\n\nTracker connections use stable IDs under `integrations.connections`. Shared non-secret settings belong in `.hero/hero.json`; personal `auth.token` belongs at the same path in `.hero/hero.local.json`. Use `hero connect --list` to inspect readiness and `hero sync import` to import tracker issues. Never put credentials in argv or committed config; automation uses `--token-stdin`."

// operationalGuidanceSection adapts heroOperationalGuidance to
// managed.SectionContributor. Render returns only the paragraph body; the
// orchestrator prepends the H2 heading from SectionTitle (matching the
// snapshot pointer section's heading discipline).
type operationalGuidanceSection struct{}

// newHeroOperationalGuidanceSection returns the shared, domain-agnostic
// operational-guidance section contributor. It takes no arguments — the
// content is identical for every domain, target, and file.
func newHeroOperationalGuidanceSection() managed.SectionContributor {
	return operationalGuidanceSection{}
}

func (operationalGuidanceSection) SectionID() string    { return operationalGuidanceSectionID }
func (operationalGuidanceSection) SectionTitle() string { return "Hero Binary & MCP Surface" }

func (operationalGuidanceSection) Render(_ managed.Context) (string, error) {
	return heroOperationalGuidance, nil
}
