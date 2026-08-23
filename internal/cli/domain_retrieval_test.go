package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/domains"
)

func addDomainRetrievalSpec(t *testing.T, env *testEnv, slug, domain string) {
	t.Helper()
	env.addSpec("planning/features/"+slug+"/spec.md", "---\n"+
		"title: "+slug+"\nslug: "+slug+"\ntype: feature\nstatus: planning\n"+
		"domain: "+domain+"\n---\n\n# "+slug+"\n\nshared retrieval phrase\n")
}

func configureRetrievalDomains(t *testing.T, env *testEnv, extensions ...domains.DomainID) {
	t.Helper()
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetDomainComposition(domains.DomainEngineering, extensions); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}
}

func TestListCompositionDefaultsFocusAndHistoricalOverrides(t *testing.T) {
	env := newTestEnv(t)
	addDomainRetrievalSpec(t, env, "eng-work", "engineering")
	addDomainRetrievalSpec(t, env, "pm-work", "pm")
	addDomainRetrievalSpec(t, env, "qa-work", "qa")
	addDomainRetrievalSpec(t, env, "sales-history", "sales")
	configureRetrievalDomains(t, env, domains.DomainPM, domains.DomainQA)

	out, err := runCmd("list", "--format", "text", "--focused-domain", "qa")
	if err != nil {
		t.Fatal(err)
	}
	qa, eng, pm := strings.Index(out, "qa-work"), strings.Index(out, "eng-work"), strings.Index(out, "pm-work")
	if qa < 0 || eng < 0 || pm < 0 || !(qa < eng && eng < pm) {
		t.Fatalf("focused composition order should be qa, engineering, pm:\n%s", out)
	}
	if strings.Contains(out, "sales-history") {
		t.Fatalf("disabled domain appeared in default list:\n%s", out)
	}

	configureRetrievalDomains(t, env, domains.DomainPM)
	out, err = runCmd("list", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "qa-work") {
		t.Fatalf("disabled QA appeared in default list:\n%s", out)
	}
	out, err = runCmd("list", "--format", "text", "--domain", "qa")
	if err != nil || !strings.Contains(out, "qa-work") || strings.Contains(out, "eng-work") {
		t.Fatalf("explicit QA history retrieval failed (err=%v):\n%s", err, out)
	}
	out, err = runCmd("list", "--format", "text", "--all-domains")
	if err != nil || !strings.Contains(out, "qa-work") || !strings.Contains(out, "sales-history") {
		t.Fatalf("all-domain list failed (err=%v):\n%s", err, out)
	}
}

func TestSearchCompositionDefaultsAndFocusRanking(t *testing.T) {
	env := newTestEnv(t)
	addDomainRetrievalSpec(t, env, "eng-search", "engineering")
	addDomainRetrievalSpec(t, env, "qa-search", "qa")
	addDomainRetrievalSpec(t, env, "sales-search", "sales")
	configureRetrievalDomains(t, env, domains.DomainQA)
	env.indexAll()

	out, err := runCmd("search", "shared retrieval phrase", "--focused-domain", "qa")
	if err != nil {
		t.Fatal(err)
	}
	qa, eng := strings.Index(out, "qa-search"), strings.Index(out, "eng-search")
	if qa < 0 || eng < 0 || qa > eng {
		t.Fatalf("focused QA should rank before engineering:\n%s", out)
	}
	if strings.Contains(out, "sales-search") {
		t.Fatalf("disabled sales appeared in default search:\n%s", out)
	}

	configureRetrievalDomains(t, env)
	out, err = runCmd("search", "--list")
	if err != nil || strings.Contains(out, "qa-search") {
		t.Fatalf("disabled QA appeared in default search (err=%v):\n%s", err, out)
	}
	out, err = runCmd("search", "--list", "--domain", "qa")
	if err != nil || !strings.Contains(out, "qa-search") {
		t.Fatalf("explicit QA search failed (err=%v):\n%s", err, out)
	}
	out, err = runCmd("search", "--list", "--domain", "*")
	if err != nil || !strings.Contains(out, "qa-search") || !strings.Contains(out, "sales-search") {
		t.Fatalf("all-domain search failed (err=%v):\n%s", err, out)
	}
}

func TestSearchDomainScopeAppliesBeforeResultCap(t *testing.T) {
	env := newTestEnv(t)
	for i := 0; i < 55; i++ {
		addDomainRetrievalSpec(t, env, fmt.Sprintf("sales-strong-%02d", i), "sales")
	}
	addDomainRetrievalSpec(t, env, "qa-survives-cap", "qa")
	configureRetrievalDomains(t, env, domains.DomainQA)
	env.indexAll()

	// Plain unified search must constrain domains inside the FTS query, before
	// its fixed 20-result cap; post-filtering would return no QA result here.
	out, err := runCmd("search", "shared retrieval phrase")
	if err != nil || !strings.Contains(out, "qa-survives-cap") {
		t.Fatalf("enabled QA was starved by disabled-domain hits (err=%v):\n%s", err, out)
	}
	if strings.Contains(out, "sales-strong") {
		t.Fatalf("disabled sales leaked into default search:\n%s", out)
	}

	// The structured-filter FTS route has the same pre-limit contract, even
	// when QA is disabled and reached through an explicit historical scope.
	configureRetrievalDomains(t, env)
	out, err = runCmd("search", "shared retrieval phrase", "--type", "feature", "--domain", "qa")
	if err != nil || !strings.Contains(out, "qa-survives-cap") {
		t.Fatalf("explicit QA FTS result was starved (err=%v):\n%s", err, out)
	}
	if strings.Contains(out, "sales-strong") {
		t.Fatalf("explicit QA search leaked sales results:\n%s", out)
	}
	out, err = runCmd("search", "--list", "--domain", "qa")
	if err != nil || !strings.Contains(out, "qa-survives-cap") {
		t.Fatalf("explicit QA list result was starved before its 50-result cap (err=%v):\n%s", err, out)
	}
}
