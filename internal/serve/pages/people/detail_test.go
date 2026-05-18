package people

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProfileDetail_RendersComingSoonStub(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/people/profiles/bwheeler")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	for _, want := range []string{
		"bwheeler",
		"people-profile-",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("response missing %q", want)
		}
	}
}

// TestSlugRoute_DoesNotShadowSiblings — the bare `/people/profiles`
// list and the slug-detail `/people/profiles/{user}` route are
// separate registrations; verify both still hit their own handlers.
func TestSlugRoute_DoesNotShadowSiblings(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	cases := []struct {
		path     string
		mustHave string
	}{
		{"/people/profiles", "people-profiles-stub"},        // list-level stub
		{"/people/activity", "people-activity-stub"},
		{"/people/handoffs", "people-handoffs-stub"},
		{"/people/roi", "people-methodology"},
	}
	for _, tc := range cases {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Errorf("GET %s: %v", tc.path, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(body), tc.mustHave) {
			t.Errorf("GET %s: missing %q", tc.path, tc.mustHave)
		}
	}
}

func TestChatInput_RendersOnPeopleHome(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/people")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `data-chat-input-variant="inline"`) {
		t.Errorf("/people missing inline chat-input")
	}
}
