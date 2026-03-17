package fix_http

import (
	"context"
	"net/http"
)

func withContext(ctx context.Context) {
	http.NewRequest("GET", "https://example.com", nil) // want `net/http\.NewRequest must not be called. use net/http\.NewRequestWithContext`
}
