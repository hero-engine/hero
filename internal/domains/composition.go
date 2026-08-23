package domains

import (
	"fmt"
	"slices"
)

var bundledPacks = []Pack{
	{ID: DomainEngineering, Roles: []ActivationRole{RolePrimary}, Bundled: true},
	{ID: DomainSales, Roles: []ActivationRole{RolePrimary}, Bundled: true},
	{ID: DomainPM, Roles: []ActivationRole{RolePrimary, RoleExtension}, Bundled: true},
	{ID: DomainQA, Roles: []ActivationRole{RolePrimary, RoleExtension}, Bundled: true},
}

// AvailablePacks returns a detached snapshot of the embedded pack catalog.
func AvailablePacks() []Pack {
	out := make([]Pack, len(bundledPacks))
	for i, pack := range bundledPacks {
		out[i] = pack
		out[i].Roles = append([]ActivationRole(nil), pack.Roles...)
	}
	return out
}

// ResolveComposition validates and normalizes workspace domain configuration.
// A nil declared composition uses the legacy scalar domain, or engineering
// when neither shape is configured. When both shapes exist, matching primary
// values are accepted for rolling compatibility; conflicting values fail.
func ResolveComposition(declared *Composition, legacy DomainID) (ResolvedComposition, error) {
	if declared == nil {
		primary := legacy
		if primary == "" {
			primary = DomainEngineering
		}
		if err := validateRole(primary, RolePrimary); err != nil {
			return ResolvedComposition{}, err
		}
		return ResolvedComposition{Primary: primary}, nil
	}

	if declared.Primary == "" {
		return ResolvedComposition{}, fmt.Errorf("domains.primary is required")
	}
	if legacy != "" && legacy != declared.Primary {
		return ResolvedComposition{}, fmt.Errorf(
			"conflicting domain configuration: legacy domain %q does not match domains.primary %q",
			legacy, declared.Primary,
		)
	}
	if err := validateRole(declared.Primary, RolePrimary); err != nil {
		return ResolvedComposition{}, err
	}
	if len(declared.Extensions) > 0 && declared.Primary == DomainSales {
		return ResolvedComposition{}, fmt.Errorf(
			"domain %q does not support capability-pack extensions", declared.Primary,
		)
	}

	resolved := ResolvedComposition{Primary: declared.Primary}
	seen := make(map[DomainID]struct{}, len(declared.Extensions))
	for _, extension := range declared.Extensions {
		if extension == "" {
			return ResolvedComposition{}, fmt.Errorf("domains.extensions contains an empty domain")
		}
		if extension == declared.Primary {
			return ResolvedComposition{}, fmt.Errorf(
				"domain %q cannot be both primary and an extension", extension,
			)
		}
		if _, duplicate := seen[extension]; duplicate {
			continue
		}
		if err := validateRole(extension, RoleExtension); err != nil {
			return ResolvedComposition{}, err
		}
		seen[extension] = struct{}{}
		resolved.Extensions = append(resolved.Extensions, extension)
	}
	return resolved, nil
}

func validateRole(id DomainID, role ActivationRole) error {
	for _, pack := range bundledPacks {
		if pack.ID != id {
			continue
		}
		if slices.Contains(pack.Roles, role) {
			return nil
		}
		return fmt.Errorf("domain %q does not support the %s role", id, role)
	}
	return fmt.Errorf("domain %q is not available", id)
}
