package checkers

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
	"github.com/sonatard/noctx/internal/helpers"
	"github.com/sonatard/noctx/internal/registry"
)

// TLSChecker detects TLS functions that should use context
type TLSChecker struct {
	ctx *helpers.CheckerContext
}

// NewTLSChecker creates a new TLS function checker
func NewTLSChecker() *TLSChecker {
	return &TLSChecker{
		ctx: helpers.NewCheckerContext(),
	}
}

// Name returns the name of this checker
func (c *TLSChecker) Name() string {
	return "tls"
}

// Check performs the analysis for TLS functions
func (c *TLSChecker) Check(pass *analysis.Pass) error {
	// Get TLS rules from registry
	rulesByChecker := registry.GetRulesByChecker()
	
	tlsRules, exists := rulesByChecker["tls"]
	if !exists {
		return nil
	}

	// Convert rules to function configs
	tlsFunctions := make([]helpers.FunctionConfig, 0, len(tlsRules))
	for _, rule := range tlsRules {
		if rule.HasAutofix {
			tlsFunctions = append(tlsFunctions, helpers.FunctionConfig{
				PackagePath: rule.PackagePath,
				FuncName:    rule.FuncName,
				Handler:     c.getTLSHandler(rule.FullName),
			})
		}
	}

	// Check functions with autofix support
	if len(tlsFunctions) > 0 {
		if err := helpers.CheckFunctionsWithAutofix(pass, tlsFunctions, c.ctx); err != nil {
			return err
		}
	}

	// Check method functions that are detection-only
	var methodFunctions []string
	for _, rule := range tlsRules {
		if !rule.HasAutofix {
			methodFunctions = append(methodFunctions, rule.FullName)
		}
	}
	if len(methodFunctions) > 0 {
		if err := helpers.CheckMethodFunctionsWithoutAutofix(pass, methodFunctions); err != nil {
			return err
		}
	}

	return nil
}

// getTLSHandler returns the appropriate fix handler for a TLS function
func (c *TLSChecker) getTLSHandler(funcName string) func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	switch funcName {
	case "crypto/tls.Dial":
		return c.generateTLSDialFix
	case "crypto/tls.DialWithDialer":
		return c.generateTLSDialWithDialerFix
	default:
		return nil
	}
}

func (c *TLSChecker) generateTLSDialFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addrArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	configArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	
	newCall := fmt.Sprintf("(&tls.Dialer{}).DialContext(%s, %s, %s, %s)", contextExpr, networkArg, addrArg, configArg)
	return fixes.CreateSuggestedFix("Replace with (*tls.Dialer).DialContext", callExpr, newCall)
}

func (c *TLSChecker) generateTLSDialWithDialerFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 4 {
		return nil
	}

	dialerArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	addrArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	configArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[3])
	
	newCall := fmt.Sprintf("(&tls.Dialer{NetDialer: %s}).DialContext(%s, %s, %s, %s)", dialerArg, contextExpr, networkArg, addrArg, configArg)
	return fixes.CreateSuggestedFix("Replace with (*tls.Dialer).DialContext with custom dialer", callExpr, newCall)
}