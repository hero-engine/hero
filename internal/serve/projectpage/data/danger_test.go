package data

import "testing"

func TestLoadDanger_HiddenWhenNotRegistered(t *testing.T) {
	d := LoadDanger(DangerInputs{Slug: "foo", Registered: false})
	if d.Visible {
		t.Errorf("Visible should be false when not registered; got %+v", d)
	}
	if len(d.Verbs) != 0 {
		t.Errorf("expected no verbs when hidden; got %+v", d.Verbs)
	}
}

func TestLoadDanger_HiddenWhenSlugEmpty(t *testing.T) {
	d := LoadDanger(DangerInputs{Slug: "", Registered: true})
	if d.Visible {
		t.Errorf("Visible should be false with empty slug; got %+v", d)
	}
}

func TestLoadDanger_DeregisterPresent(t *testing.T) {
	d := LoadDanger(DangerInputs{Slug: "foo", Registered: true})
	if !d.Visible {
		t.Fatalf("expected Visible=true")
	}
	if len(d.Verbs) != 1 {
		t.Fatalf("expected 1 verb, got %d", len(d.Verbs))
	}
	if d.Verbs[0].Verb != "deregister" {
		t.Errorf("Verb = %q, want deregister", d.Verbs[0].Verb)
	}
	if d.Verbs[0].Endpoint != "/api/foo/registry/remove" {
		t.Errorf("Endpoint = %q, want /api/foo/registry/remove", d.Verbs[0].Endpoint)
	}
}
