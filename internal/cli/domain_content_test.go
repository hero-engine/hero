package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/domains"
)

func TestDomainContentListsAndLoadsEnabledDeferredContentLocally(t *testing.T) {
	env := newTestEnv(t)
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetDomainComposition(domains.DomainEngineering, []domains.DomainID{domains.DomainPM}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}

	listed, err := runCmd("domain", "content")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pm.agent.metrics-analyst", "pm.skill.metrics-design", "local=true"} {
		if !strings.Contains(listed, want) {
			t.Errorf("content list missing %q: %s", want, listed)
		}
	}
	if strings.Contains(listed, "qa.agent.decision-table-author") {
		t.Fatalf("disabled QA content was listed: %s", listed)
	}

	loaded, err := runCmd("domain", "content", "pm.agent.metrics-analyst")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(loaded, "name: metrics-analyst") {
		t.Fatalf("unexpected deferred content: %s", loaded)
	}
	if _, err := os.Stat(filepath.Join(env.dir, ".codex", "agents", "metrics-analyst.toml")); !os.IsNotExist(err) {
		t.Fatalf("read-only content load wrote an installed agent: %v", err)
	}
}

func TestDomainContentRejectsDisabledOwner(t *testing.T) {
	newTestEnv(t)
	_, err := runCmd("domain", "content", "qa.agent.decision-table-author")
	if err == nil || !strings.Contains(err.Error(), "disabled pack \"qa\"") {
		t.Fatalf("error = %v, want disabled-owner rejection", err)
	}
}
