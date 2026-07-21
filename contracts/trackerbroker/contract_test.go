package trackerbroker

import (
	"encoding/json"
	"testing"
)

func TestConsumerFixture(t *testing.T) {
	b, err := ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version   string                     `json:"version"`
		Responses map[string]json.RawMessage `json:"responses"`
	}
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != Version {
		t.Fatalf("version = %q", fixture.Version)
	}
	for _, name := range []string{"get_issue", "search", "request", "cli", "error", "truncated"} {
		var response Response
		if err := json.Unmarshal(fixture.Responses[name], &response); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if response.Version != Version {
			t.Fatalf("%s version = %q", name, response.Version)
		}
	}
}
