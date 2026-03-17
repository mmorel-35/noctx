package fix_context_import

import "net"

// This file deliberately does NOT import "context". The suggested fix must
// add the import as part of the TextEdit so the result compiles.

func dial() {
	_, _ = net.Dial("tcp", "localhost:8080") // want `net\.Dial must not be called. use \(\*net\.Dialer\)\.DialContext`
}
