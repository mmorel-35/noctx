package fix_http_methods

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

func _() {
	ctx := context.Background()
	_ = ctx
	_ = strings.NewReader // needed by the PostForm suggested fix

	_, _ = http.Get("https://example.com")                              // want `net/http\.Get must not be called. use net/http\.NewRequestWithContext and \(\*net/http.Client\)\.Do\(\*http.Request\)`
	_, _ = http.Head("https://example.com")                             // want `net/http\.Head must not be called. use net/http\.NewRequestWithContext and \(\*net/http.Client\)\.Do\(\*http.Request\)`
	_, _ = http.Post("https://example.com", "text/plain", nil)          // want `net/http\.Post must not be called. use net/http\.NewRequestWithContext and \(\*net/http.Client\)\.Do\(\*http.Request\)`
	_, _ = http.PostForm("https://example.com", url.Values{"k": {"v"}}) // want `net/http\.PostForm must not be called. use net/http\.NewRequestWithContext and \(\*net/http.Client\)\.Do\(\*http.Request\)`
}
