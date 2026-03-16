package fix_alias

import (
	"context"
	e "os/exec"
	h "net/http"
)

// Aliases: the fix must use the alias (e., h.) rather than the canonical package name.

func withAlias(ctx context.Context) {
	e.Command("ls", "-l")                             // want `os/exec.Command must not be called. use os/exec.CommandContext`
	h.NewRequest("GET", "https://example.com", nil)   // want `net/http\.NewRequest must not be called. use net/http\.NewRequestWithContext`
}
