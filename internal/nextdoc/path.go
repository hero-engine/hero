package nextdoc

import (
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
)

// UserSlug names the per-user handoff file in team mode: the configured
// default agent (without its "human/" prefix), else the git user name.
func UserSlug(cfg config.Config) string {
	if cfg.Tracking != nil && cfg.Tracking.DefaultAgent != "" {
		return strings.TrimPrefix(cfg.Tracking.DefaultAgent, "human/")
	}
	return gitutil.UserName()
}

// HandoffPath is the handoff briefing `hero next` shows: the per-user
// .hero/next/<user>.md in team mode, else the shared .hero/NEXT.md.
func HandoffPath(heroDir string, cfg config.Config) string {
	if cfg.NextMode() == "team" {
		return filepath.Join(heroDir, "next", UserSlug(cfg)+".md")
	}
	return filepath.Join(heroDir, "NEXT.md")
}
