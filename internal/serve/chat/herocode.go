package chat

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// HeroCodeAdapter probes a configured hero-code endpoint and, when
// reachable, registers itself as the canonical Hero adapter.
//
// The wire surface on hero-code's side is still being built (per the
// cross-repo peering trail on hero-chat-and-model). The client-side
// probe here is complete: it verifies reachability and, when the
// remote responds, the registered adapter shows up in capability
// JSON. Stream returns a "not yet wired" error until hero-code's
// server side ships, matching mcpClientAdapter's behavior.
type HeroCodeAdapter struct {
	endpoint string
	version  string
}

// TryConnectHeroCode probes endpoint and returns an adapter on
// success. endpoint may be:
//
//   - "unix:///path/to/socket"  — verified with net.Dial("unix", ...)
//   - "http://host:port"        — verified with a 2s GET against /
//   - "https://host:port"       — same
//
// Other schemes return an error.
func TryConnectHeroCode(endpoint string) (*HeroCodeAdapter, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint required")
	}
	switch {
	case strings.HasPrefix(endpoint, "unix://"):
		path := strings.TrimPrefix(endpoint, "unix://")
		conn, err := net.DialTimeout("unix", path, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("dial %s: %w", endpoint, err)
		}
		conn.Close()
	case strings.HasPrefix(endpoint, "http://"), strings.HasPrefix(endpoint, "https://"):
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(endpoint)
		if err != nil {
			return nil, fmt.Errorf("get %s: %w", endpoint, err)
		}
		resp.Body.Close()
	default:
		return nil, fmt.Errorf("unsupported endpoint scheme: %s", endpoint)
	}
	return &HeroCodeAdapter{endpoint: endpoint, version: "unknown"}, nil
}

func (a *HeroCodeAdapter) Name() string    { return "hero-code" }
func (a *HeroCodeAdapter) Version() string { return a.version }
func (a *HeroCodeAdapter) Kinds() []Kind   { return []Kind{KindInteractive, KindHeadless} }
func (a *HeroCodeAdapter) Close() error    { return nil }

// Stream is a placeholder. The hero-code server side that implements
// the chat protocol over the configured endpoint is in flight; until
// then, dispatched turns surface as a clear adapter_not_wired error.
func (a *HeroCodeAdapter) Stream(ctx context.Context, req DispatchRequest) (<-chan Event, error) {
	out := make(chan Event, 2)
	go func() {
		defer close(out)
		out <- ErrorEvent(
			"adapter_not_wired",
			fmt.Sprintf("hero-code at %s registered but the dispatch wire is not yet implemented", a.endpoint),
			"",
		)
		out <- DoneEvent(0, nil)
	}()
	return out, nil
}
