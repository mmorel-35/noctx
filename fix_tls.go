package noctx

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// ── crypto/tls fixes ──────────────────────────────────────────────────────────

func fixTLSDial(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	q := extractQualifier(ce)
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	config := arg(pass, ce, 2)
	newText := fmt.Sprintf("(&%sDialer{Config: %s}).DialContext(%s, %s, %s)", q, config, ctx, network, address)
	return createFix(fmt.Sprintf("Replace %sDial with (*%sDialer).DialContext", q, q), ce, newText)
}

func fixTLSDialWithDialer(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 4 {
		return nil
	}
	q := extractQualifier(ce)
	dialer := arg(pass, ce, 0)
	network := arg(pass, ce, 1)
	address := arg(pass, ce, 2)
	config := arg(pass, ce, 3)
	newText := fmt.Sprintf("(&%sDialer{NetDialer: %s, Config: %s}).DialContext(%s, %s, %s)", q, dialer, config, ctx, network, address)
	return createFix(fmt.Sprintf("Replace %sDialWithDialer with (*%sDialer).DialContext", q, q), ce, newText)
}
