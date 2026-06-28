package mocktracker

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	sprout "github.com/bdwheeler/sprout/go"
)

// defaultSeed is the tiny embedded Acme fixture (2 epics, ~6 issues, 1
// milestone, 1 iteration) so `go test ./internal/mocktracker/...` is
// fully self-contained — no dependency on hero-code's full dataset.
//
//go:embed fixtures/seed/*.yaml fixtures/seed/seeds.list
var defaultSeed embed.FS

// Seed applies Sprout seed YAML to the store's DB. When dir is empty the
// embedded default fixture is used; otherwise dir is a real directory
// (hero-code's full Acme dataset via --seed). force re-applies even when
// the checksum matches (sprout's idempotent skip is bypassed), used for
// the mid-run reset. After applying, id_alias is rebuilt.
//
// Format and validation are owned by Sprout (seed-json-schema): a
// malformed seed surfaces sprout's validation error verbatim.
func (s *Store) Seed(ctx context.Context, dir string, force bool) (sprout.Results, error) {
	opts := []sprout.Option{sprout.WithDialect(sprout.SQLiteDialect{})}
	if dir == "" {
		opts = append(opts, sprout.WithFS(defaultSeed, "fixtures/seed"))
	} else {
		opts = append(opts, sprout.WithDir(dir))
	}
	if force {
		opts = append(opts, sprout.WithForce())
	}
	res, err := sprout.Apply(ctx, s.db, opts...)
	if err != nil {
		return res, fmt.Errorf("seed: %w", err)
	}
	if err := s.RebuildAliases(ctx); err != nil {
		return res, fmt.Errorf("rebuild id aliases: %w", err)
	}
	return res, nil
}

// seedFromFS applies an arbitrary fs.FS rooted at base. Used by tests to
// drive a clean apply / idempotent re-apply / invalid seed without
// touching disk.
func (s *Store) seedFromFS(ctx context.Context, fsys fs.FS, base string, force bool) (sprout.Results, error) {
	opts := []sprout.Option{
		sprout.WithFS(fsys, base),
		sprout.WithDialect(sprout.SQLiteDialect{}),
	}
	if force {
		opts = append(opts, sprout.WithForce())
	}
	return sprout.Apply(ctx, s.db, opts...)
}
