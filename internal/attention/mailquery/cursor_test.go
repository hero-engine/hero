package mailquery

import "testing"

func TestCursorRoundTripAndFilterBinding(t *testing.T) {
	unread := true
	want := messageCursor{
		Filters:  cursorFilters{ProjectPeerID: "peer_a", ThreadID: "mail_thread", UnreadOnly: &unread},
		Revision: "revision", ActivityAt: "2026-08-17T10:00:00Z", PeerID: "peer_a", MessageID: "mail_1",
	}
	encoded, err := encodeCursor(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != cursorVersion || got.Revision != want.Revision || got.MessageID != want.MessageID || !sameFilters(got.Filters, want.Filters) {
		t.Fatalf("decoded cursor = %#v", got)
	}

	read := false
	for _, filters := range []cursorFilters{
		{ProjectPeerID: "peer_b", ThreadID: "mail_thread", UnreadOnly: &unread},
		{ProjectPeerID: "peer_a", ThreadID: "mail_other", UnreadOnly: &unread},
		{ProjectPeerID: "peer_a", ThreadID: "mail_thread", UnreadOnly: &read},
		{ProjectPeerID: "peer_a", ThreadID: "mail_thread"},
	} {
		if sameFilters(got.Filters, filters) {
			t.Fatalf("cursor unexpectedly matched filters %#v", filters)
		}
	}
}

func TestCursorRejectsMalformedAndIncompleteValues(t *testing.T) {
	for _, value := range []string{"not-base64!", "e30", "eyJ2ZXJzaW9uIjoyfQ"} {
		if _, err := decodeCursor(value); err == nil {
			t.Fatalf("decodeCursor(%q) succeeded", value)
		}
	}
}
