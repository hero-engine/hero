package intake

import (
	"fmt"

	"github.com/hero-engine/hero/internal/spec"
)

// Repository is the filesystem authority used by Intake operations.
type Repository struct {
	HeroDir string
}

func (r Repository) Resolve(slug string) (*spec.Spec, error) {
	specs, err := spec.Discover(r.HeroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}
	for _, candidate := range specs {
		if candidate.Slug == slug && candidate.Type == spec.TypeIntake {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("intake %q not found", slug)
}
