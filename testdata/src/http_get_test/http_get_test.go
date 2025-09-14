package http_get_test

import (
	"net/http"
)

func testHttpGet() {
	// This should be auto-fixed
	resp, _ := http.Get("https://example.com") // want `net/http\.Get must not be called. use net/http\.NewRequestWithContext and \(\*net/http.Client\)\.Do\(\*http.Request\)`
	
	_ = resp
}

func testHttpHead() {
	// This should be auto-fixed
	resp, _ := http.Head("https://example.com") // want `net/http\.Head must not be called. use net/http\.NewRequestWithContext and \(\*net/http.Client\)\.Do\(\*http.Request\)`
	
	_ = resp
}