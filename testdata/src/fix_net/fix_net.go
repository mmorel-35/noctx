package fix_net

import (
	"context"
	"net"
	"time"
)

func _() {
	ctx := context.Background()
	_ = ctx

	_, _ = net.Dial("tcp", "localhost:8080")                   // want `net\.Dial must not be called. use \(\*net\.Dialer\)\.DialContext`
	_, _ = net.DialTimeout("tcp", "localhost:8080", 10*time.Second) // want `net\.DialTimeout must not be called. use \(\*net\.Dialer\)\.DialContext with \(\*net\.Dialer\)\.Timeout`
	_, _ = net.Listen("tcp", ":8080")                          // want `net\.Listen must not be called. use \(\*net\.ListenConfig\)\.Listen`
	_, _ = net.ListenPacket("udp", ":8080")                    // want `net\.ListenPacket must not be called. use \(\*net\.ListenConfig\)\.ListenPacket`
	_, _ = net.LookupCNAME("example.com")                     // want `net\.LookupCNAME must not be called. use \(\*net\.Resolver\)\.LookupCNAME with a context`
	_, _ = net.LookupHost("example.com")                      // want `net\.LookupHost must not be called. use \(\*net\.Resolver\)\.LookupHost with a context`
	_, _ = net.LookupIP("example.com")                        // want `net\.LookupIP must not be called. use \(\*net\.Resolver\)\.LookupIPAddr with a context`
	_, _ = net.LookupPort("tcp", "http")                      // want `net\.LookupPort must not be called. use \(\*net\.Resolver\)\.LookupPort with a context`
	_, _, _ = net.LookupSRV("http", "tcp", "example.com")    // want `net\.LookupSRV must not be called. use \(\*net\.Resolver\)\.LookupSRV with a context`
	_, _ = net.LookupMX("example.com")                        // want `net\.LookupMX must not be called. use \(\*net\.Resolver\)\.LookupMX with a context`
	_, _ = net.LookupNS("example.com")                        // want `net\.LookupNS must not be called. use \(\*net\.Resolver\)\.LookupNS with a context`
	_, _ = net.LookupTXT("example.com")                       // want `net\.LookupTXT must not be called. use \(\*net\.Resolver\)\.LookupTXT with a context`
	_, _ = net.LookupAddr("8.8.8.8")                          // want `net\.LookupAddr must not be called. use \(\*net\.Resolver\)\.LookupAddr with a context`
}
