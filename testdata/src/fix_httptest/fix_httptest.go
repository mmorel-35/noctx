package fix_httptest

import (
	"context"
	"net/http"
	"net/http/httptest"
)

func _() {
	ctx := context.Background()
	_ = ctx

	req := httptest.NewRequest("GET", "/api/v1", nil) // want `net/http/httptest\.NewRequest must not be called. use net/http/httptest\.NewRequestWithContext`
	_ = req
}

func withCtx(ctx context.Context) {
	req := httptest.NewRequest("POST", "/submit", nil) // want `net/http/httptest\.NewRequest must not be called. use net/http/httptest\.NewRequestWithContext`
	_ = req
	_ = http.MethodPost
}
