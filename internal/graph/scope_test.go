package graph

import (
	"reflect"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/domains"
)

func TestResolveDomain_PrecedenceChain(t *testing.T) {
	cases := []struct {
		name     string
		cfg      config.Config
		override string
		want     DomainScope
	}{
		{"override pm beats cfg engineering", config.Config{Domain: "engineering"}, "pm", DomainScope{Active: "pm"}},
		{"override * widens to all", config.Config{Domain: "engineering"}, "*", DomainScope{AllDomains: true}},
		{"no override falls back to cfg", config.Config{Domain: "pm"}, "", DomainScope{Active: "pm", Enabled: []string{"pm", "core"}}},
		{"empty cfg falls back to engineering", config.Config{}, "", DomainScope{Active: "engineering", Enabled: []string{"engineering", "core"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveDomain(tc.cfg, tc.override)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ResolveDomain(%+v, %q) = %+v, want %+v", tc.cfg, tc.override, got, tc.want)
			}
		})
	}
}

func TestDomainScope_Where(t *testing.T) {
	cases := []struct {
		name     string
		scope    DomainScope
		alias    string
		wantFrag string
		wantArgs []any
	}{
		{"all-domains returns empty", DomainScope{AllDomains: true}, "n", "", nil},
		{"active no alias", DomainScope{Active: "engineering"}, "", "domain = ?", []any{"engineering"}},
		{"active with alias", DomainScope{Active: "pm"}, "n", "n.domain = ?", []any{"pm"}},
		{"enabled stack", DomainScope{Active: "engineering", Enabled: []string{"engineering", "pm", "qa"}}, "n", "n.domain IN (?,?,?)", []any{"engineering", "pm", "qa"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frag, args := tc.scope.Where(tc.alias)
			if frag != tc.wantFrag {
				t.Errorf("frag = %q, want %q", frag, tc.wantFrag)
			}
			if !reflect.DeepEqual(args, tc.wantArgs) {
				t.Errorf("args = %v, want %v", args, tc.wantArgs)
			}
		})
	}
}

func TestDomainScope_Match(t *testing.T) {
	cases := []struct {
		name   string
		scope  DomainScope
		domain string
		want   bool
	}{
		{"all-domains matches anything", DomainScope{AllDomains: true}, "pm", true},
		{"all-domains matches empty", DomainScope{AllDomains: true}, "", true},
		{"engineering active matches engineering", DomainScope{Active: "engineering"}, "engineering", true},
		{"engineering active rejects pm", DomainScope{Active: "engineering"}, "pm", false},
		{"engineering active rejects empty", DomainScope{Active: "engineering"}, "", false},
		{"enabled stack includes qa", DomainScope{Active: "engineering", Enabled: []string{"engineering", "pm", "qa"}}, "qa", true},
		{"enabled stack rejects sales", DomainScope{Active: "engineering", Enabled: []string{"engineering", "pm", "qa"}}, "sales", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.scope.Match(tc.domain); got != tc.want {
				t.Errorf("Match(%q) = %v, want %v", tc.domain, got, tc.want)
			}
		})
	}
}

func TestResolveDomainFocusedRanking(t *testing.T) {
	cfg := config.Config{Domains: &domains.Composition{
		Primary: domains.DomainEngineering, Extensions: []domains.DomainID{domains.DomainPM, domains.DomainQA},
	}}
	scope := ResolveDomainFocused(cfg, "", "qa")
	if got := []int{scope.Rank("qa"), scope.Rank("engineering"), scope.Rank("pm"), scope.Rank("sales")}; !reflect.DeepEqual(got, []int{0, 1, 2, 4}) {
		t.Fatalf("ranks = %v", got)
	}
	explicit := ResolveDomainFocused(cfg, "pm", "qa")
	if explicit.Active != "pm" || len(explicit.Enabled) != 0 || !explicit.Match("pm") || explicit.Match("qa") {
		t.Fatalf("explicit scope = %#v", explicit)
	}
}
