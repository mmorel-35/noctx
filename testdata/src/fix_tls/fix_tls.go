package fix_tls

import (
	"context"
	"crypto/tls"
	"net"
)

func _() {
	ctx := context.Background()
	_ = ctx

	netDialer := &net.Dialer{}
	tlsConfig := &tls.Config{}

	_, _ = tls.Dial("tcp", "localhost:443", tlsConfig)                      // want `crypto/tls\.Dial must not be called. use \(\*crypto/tls\.Dialer\)\.DialContext`
	_, _ = tls.DialWithDialer(netDialer, "tcp", "localhost:443", tlsConfig) // want `crypto/tls\.DialWithDialer must not be called. use \(\*crypto/tls\.Dialer\)\.DialContext with NetDialer`
}
