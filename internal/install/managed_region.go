package install

import "github.com/hero-engine/hero/internal/managed"

// managed_region.go — thin compatibility shim.
//
// The marker-region primitives (FindManagedRegion, RenderManagedRegion,
// InsertManagedRegion, IsLegacyHeroStub, and the ManagedRegion parse
// struct) now live in internal/managed. This file re-exports them
// under their original names so install-internal call sites and the
// existing internal/install tests keep working unchanged.
//
// New callers should import internal/managed directly.

// ManagedRegion is the parsed view of a Hero-managed region in a file.
// Aliased to managed.Region for source compatibility.
type ManagedRegion = managed.Region

// FindManagedRegion locates the Hero-managed region in content.
// See managed.FindManagedRegion for details.
func FindManagedRegion(content string) ManagedRegion {
	return managed.FindManagedRegion(content)
}

// RenderManagedRegion produces a managed-region block at the given
// version. See managed.RenderManagedRegion for details.
func RenderManagedRegion(version, body string) string {
	return managed.RenderManagedRegion(version, body)
}

// InsertManagedRegion produces a new file content with the managed
// region inserted or updated. See managed.InsertManagedRegion for
// details.
func InsertManagedRegion(existing, region string) string {
	return managed.InsertManagedRegion(existing, region)
}

// IsLegacyHeroStub returns true when content consists entirely of a
// Hero managed region. See managed.IsLegacyHeroStub for details.
func IsLegacyHeroStub(content string) bool {
	return managed.IsLegacyHeroStub(content)
}

// legacyMarker is the pre-versioned single-line Hero marker. Re-exported
// here as a package-internal constant so install-internal test fixtures
// and target_generic.go's legacy stub writer keep building without
// reaching into internal/managed.
const legacyMarker = "<!-- hero:managed -->"
