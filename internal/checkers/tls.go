package checkers

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
	"github.com/sonatard/noctx/internal/registry"
)

// TLSChecker handles crypto/tls package functions that need context
type TLSChecker struct {
	contextDetector *fixes.ContextDetector
	argFormatter    *fixes.ArgumentFormatter
	assignDetector  *fixes.VariableAssignmentDetector
}

// NewTLSChecker creates a new TLS checker instance
func NewTLSChecker() *TLSChecker {
	return &TLSChecker{
		contextDetector: &fixes.ContextDetector{},
		argFormatter:    &fixes.ArgumentFormatter{},
		assignDetector:  &fixes.VariableAssignmentDetector{},
	}
}

// Name returns the name of this checker
func (c *TLSChecker) Name() CheckerName {
	return TLSCheckerName
}

// Check performs the analysis for TLS functions
func (c *TLSChecker) Check(pass *analysis.Pass) error {
	// Get TLS rules from registry
	rulesByChecker := registry.GetRulesByChecker()
	tlsRules := rulesByChecker[TLSCheckerName]
	
	if len(tlsRules) == 0 {
		return nil // No TLS rules to process
	}

	functions := []FunctionConfig{
		{"crypto/tls", "Dial", c.generateTLSDialFix},
		{"crypto/tls", "DialWithDialer", c.generateTLSDialWithDialerFix},
	}

	httpChecker := NewHTTPChecker()
	return httpChecker.checkFunctions(pass, functions)
}

func (c *TLSChecker) generateTLSDialFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	configArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "d", "")

	newCall := fmt.Sprintf(`func() (*tls.Conn, error) {
		d %s &tls.Dialer{Config: %s}
		return d.DialContext(%s, %s, %s)
	}()`, assignOp, configArg, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*tls.Dialer).DialContext", callExpr, newCall)
}

func (c *TLSChecker) generateTLSDialWithDialerFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 4 {
		return nil
	}

	dialerArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	configArg := c.argFormatter.FormatArgument(pass, callExpr.Args[3])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "d", "")

	newCall := fmt.Sprintf(`func() (*tls.Conn, error) {
		d %s &tls.Dialer{NetDialer: %s, Config: %s}
		return d.DialContext(%s, %s, %s)
	}()`, assignOp, dialerArg, configArg, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*tls.Dialer).DialContext", callExpr, newCall)
}