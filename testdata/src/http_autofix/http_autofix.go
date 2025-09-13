package http_autofix

import (
	"context"
	"net/http"
)

func simpleCase() {
	// This should be auto-fixed to use NewRequestWithContext with context.Background()
	req, _ := http.NewRequest("GET", "https://example.com", nil) // want `net/http\.NewRequest must not be called. use net/http\.NewRequestWithContext`
	
	_ = req
}

func withExistingContext(ctx context.Context) {
	// This should be auto-fixed to use the existing context parameter
	req, _ := http.NewRequest("POST", "https://example.com", nil) // want `net/http\.NewRequest must not be called. use net/http\.NewRequestWithContext`
	
	_ = req
}

func testFunction(t interface{ Context() context.Context }) {
	// This should be auto-fixed to use t.Context()
	req, _ := http.NewRequest("PUT", "https://example.com", nil) // want `net/http\.NewRequest must not be called. use net/http\.NewRequestWithContext`
	
	_ = req
}