package cloud

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/gitutil"
)

type GraphResult struct {
	Accepted  int
	Conflicts int
}

// PushGraph opens the local graph store and pushes pending deltas to the
// cloud server. It is the library equivalent of `hero sync graph push`.
func PushGraph(ctx context.Context, httpClient *http.Client, serverURL, orgID, heroDir, projectRoot string) (*GraphResult, error) {
	store, err := graph.Open(heroDir)
	if err != nil {
		return nil, fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(projectRoot)

	client := graph.NewSyncClient(serverURL, repoKey, orgID)
	client.HTTP = httpClient

	resp, err := store.Push(client)
	if err != nil {
		return nil, fmt.Errorf("push: %w", err)
	}

	return &GraphResult{
		Accepted:  resp.Accepted,
		Conflicts: len(resp.Conflicts),
	}, nil
}
