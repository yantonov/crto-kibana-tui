package opensearch

import (
	"context"

	"github.com/yantonov/crtokt/src/config"
	"github.com/yantonov/crtokt/src/models"
)

// Searcher is the interface the TUI uses to authenticate and run searches.
// Using this interface instead of *Client allows callers to be tested with
// stub implementations.
type Searcher interface {
	Login(ctx context.Context, kibanaURL, username, password string) error
	IsAuthenticated() bool
	SearchAll(ctx context.Context, cfg config.Provider, filter models.Filter) models.CombinedResult
}

// Compile-time assertion that *Client satisfies Searcher.
var _ Searcher = (*Client)(nil)
