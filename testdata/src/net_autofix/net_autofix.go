package net_autofix

import (
	"context"
	"net"
)

func testNetDial() {
	// This should be auto-fixed with net.Dial -> (*net.Dialer).DialContext
	conn, _ := net.Dial("tcp", "localhost:8080") // want `net\.Dial must not be called. use \(\*net\.Dialer\)\.DialContext`
	
	_ = conn
}

func testNetDialWithContext(ctx context.Context) {
	// This should be auto-fixed using the existing context parameter
	conn, err := net.Dial("tcp", "localhost:8080") // want `net\.Dial must not be called. use \(\*net\.Dialer\)\.DialContext`
	
	_ = conn
	_ = err
}