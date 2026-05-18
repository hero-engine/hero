package edition

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		env  string
		want Edition
	}{
		{"", Local},
		{"local", Local},
		{"LOCAL", Local},
		{" Team ", Team},
		{"cloud", Cloud},
		{"enterprise", Enterprise},
		{"CE", CE},
		{"garbage", Local}, // unknown → local
	}
	for _, c := range cases {
		t.Run(c.env, func(t *testing.T) {
			t.Setenv("HERO_EDITION", c.env)
			got := Resolve()
			if got != c.want {
				t.Fatalf("Resolve(env=%q) = %q, want %q", c.env, got, c.want)
			}
		})
	}
}

func TestAllowed(t *testing.T) {
	cases := []struct {
		name    string
		active  Edition
		allowed []string
		want    bool
	}{
		{"empty allowed = all", Local, nil, true},
		{"empty slice = all", Cloud, []string{}, true},
		{"match", Team, []string{"team", "cloud"}, true},
		{"case insensitive", Team, []string{"TEAM"}, true},
		{"trimmed", Cloud, []string{" cloud "}, true},
		{"miss", Local, []string{"team", "cloud"}, false},
		{"enterprise allowed", Enterprise, []string{"cloud", "enterprise"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Allowed(c.active, c.allowed); got != c.want {
				t.Fatalf("Allowed(%q, %v) = %v, want %v", c.active, c.allowed, got, c.want)
			}
		})
	}
}

func TestRequire(t *testing.T) {
	t.Setenv("HERO_EDITION", "team")
	mw := Require("team", "cloud")
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rec.Code)
		}
	})

	t.Run("blocked", func(t *testing.T) {
		t.Setenv("HERO_EDITION", "local")
		mw := Require("team", "cloud")
		h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("want 404, got %d", rec.Code)
		}
	})
}

func TestAllSlice(t *testing.T) {
	all := All()
	want := []Edition{Local, Team, Cloud, Enterprise, CE}
	if len(all) != len(want) {
		t.Fatalf("len = %d, want %d", len(all), len(want))
	}
	for i, e := range all {
		if e != want[i] {
			t.Fatalf("All()[%d] = %q, want %q", i, e, want[i])
		}
	}
}
