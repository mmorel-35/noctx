package noctx

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// ── net fixes ────────────────────────────────────────────────────────────────

func fixNetDial(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	q := extractQualifier(ce)
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	newText := fmt.Sprintf("(&%sDialer{}).DialContext(%s, %s, %s)", q, ctx, network, address)
	return createFix(fmt.Sprintf("Replace %sDial with (*%sDialer).DialContext", q, q), ce, newText)
}

func fixNetDialTimeout(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	q := extractQualifier(ce)
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	timeout := arg(pass, ce, 2)
	newText := fmt.Sprintf("(&%sDialer{Timeout: %s}).DialContext(%s, %s, %s)", q, timeout, ctx, network, address)
	return createFix(fmt.Sprintf("Replace %sDialTimeout with (*%sDialer).DialContext", q, q), ce, newText)
}

func fixNetListen(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	q := extractQualifier(ce)
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	newText := fmt.Sprintf("(&%sListenConfig{}).Listen(%s, %s, %s)", q, ctx, network, address)
	return createFix(fmt.Sprintf("Replace %sListen with (*%sListenConfig).Listen", q, q), ce, newText)
}

func fixNetListenPacket(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	q := extractQualifier(ce)
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	newText := fmt.Sprintf("(&%sListenConfig{}).ListenPacket(%s, %s, %s)", q, ctx, network, address)
	return createFix(fmt.Sprintf("Replace %sListenPacket with (*%sListenConfig).ListenPacket", q, q), ce, newText)
}

// netResolverFix returns a fixFunc (a closure) for a single-argument
// net.Lookup* function that maps to (*net.Resolver).method(ctx, arg).
// It is a factory so the method name can be captured without a separate
// wrapper function per Lookup variant.
func netResolverFix(method string) fixFunc {
	return func(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
		if len(ce.Args) != 1 {
			return nil
		}
		q := extractQualifier(ce)
		lookupArg := arg(pass, ce, 0)
		newText := fmt.Sprintf("(&%sResolver{}).%s(%s, %s)", q, method, ctx, lookupArg)
		return createFix(fmt.Sprintf("Replace %s%s with (*%sResolver).%s", q, method, q, method), ce, newText)
	}
}

func fixNetLookupIP(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 1 {
		return nil
	}
	q := extractQualifier(ce)
	host := arg(pass, ce, 0)
	newText := fmt.Sprintf("(&%sResolver{}).LookupIPAddr(%s, %s)", q, ctx, host)
	return createFix(fmt.Sprintf("Replace %sLookupIP with (*%sResolver).LookupIPAddr", q, q), ce, newText)
}

func fixNetLookupPort(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	q := extractQualifier(ce)
	network := arg(pass, ce, 0)
	service := arg(pass, ce, 1)
	newText := fmt.Sprintf("(&%sResolver{}).LookupPort(%s, %s, %s)", q, ctx, network, service)
	return createFix(fmt.Sprintf("Replace %sLookupPort with (*%sResolver).LookupPort", q, q), ce, newText)
}

func fixNetLookupSRV(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	q := extractQualifier(ce)
	service := arg(pass, ce, 0)
	proto := arg(pass, ce, 1)
	name := arg(pass, ce, 2)
	newText := fmt.Sprintf("(&%sResolver{}).LookupSRV(%s, %s, %s, %s)", q, ctx, service, proto, name)
	return createFix(fmt.Sprintf("Replace %sLookupSRV with (*%sResolver).LookupSRV", q, q), ce, newText)
}
