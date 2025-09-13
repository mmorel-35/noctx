package checkers

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
	"github.com/sonatard/noctx/internal/registry"
)

// NetChecker handles all net package functions that need context
type NetChecker struct {
	contextDetector *fixes.ContextDetector
	argFormatter    *fixes.ArgumentFormatter
	assignDetector  *fixes.VariableAssignmentDetector
}

// NewNetChecker creates a new Net checker instance
func NewNetChecker() *NetChecker {
	return &NetChecker{
		contextDetector: &fixes.ContextDetector{},
		argFormatter:    &fixes.ArgumentFormatter{},
		assignDetector:  &fixes.VariableAssignmentDetector{},
	}
}

// Name returns the name of this checker
func (c *NetChecker) Name() CheckerName {
	return NetCheckerName
}

// Check performs the analysis for all net functions
func (c *NetChecker) Check(pass *analysis.Pass) error {
	// Get Net rules from registry
	rulesByChecker := registry.GetRulesByChecker()
	netRules := rulesByChecker[NetCheckerName]
	
	if len(netRules) == 0 {
		return nil // No Net rules to process
	}

	// Define all net functions we handle
	functions := []FunctionConfig{
		{"net", "Dial", c.generateNetDialFix},
		{"net", "DialTimeout", c.generateNetDialTimeoutFix},
		{"net", "Listen", c.generateNetListenFix},
		{"net", "ListenPacket", c.generateNetListenPacketFix},
		{"net", "LookupCNAME", c.generateNetLookupFix},
		{"net", "LookupHost", c.generateNetLookupFix},
		{"net", "LookupIP", c.generateNetLookupFix},
		{"net", "LookupPort", c.generateNetLookupFix},
		{"net", "LookupSRV", c.generateNetLookupFix},
		{"net", "LookupMX", c.generateNetLookupFix},
		{"net", "LookupNS", c.generateNetLookupFix},
		{"net", "LookupTXT", c.generateNetLookupFix},
		{"net", "LookupAddr", c.generateNetLookupFix},
	}

	httpChecker := NewHTTPChecker()
	return httpChecker.checkFunctions(pass, functions)
}

// Net Fix Generators

func (c *NetChecker) generateNetDialFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "dialer", "")

	newCall := fmt.Sprintf(`func() (net.Conn, error) {
		dialer %s &net.Dialer{}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext", callExpr, newCall)
}

func (c *NetChecker) generateNetDialTimeoutFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	timeoutArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "dialer", "")

	newCall := fmt.Sprintf(`func() (net.Conn, error) {
		dialer %s &net.Dialer{Timeout: %s}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, timeoutArg, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext", callExpr, newCall)
}

func (c *NetChecker) generateNetListenFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "lc", "")

	newCall := fmt.Sprintf(`func() (net.Listener, error) {
		lc %s &net.ListenConfig{}
		return lc.Listen(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.ListenConfig).Listen", callExpr, newCall)
}

func (c *NetChecker) generateNetListenPacketFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "lc", "")

	newCall := fmt.Sprintf(`func() (net.PacketConn, error) {
		lc %s &net.ListenConfig{}
		return lc.ListenPacket(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.ListenConfig).ListenPacket", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) == 0 {
		return nil
	}

	// Get function name from the call expression
	funcName := ""
	if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		funcName = sel.Sel.Name
	}

	args := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		args[i] = c.argFormatter.FormatArgument(pass, arg)
	}
	
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "r", "")
	
	// Map function names to their context equivalents
	contextFunction := funcName
	if funcName == "LookupIP" {
		contextFunction = "LookupIPAddr"
	}

	allArgs := append([]string{contextExpr}, args...)
	argList := strings.Join(allArgs, ", ")

	newCall := fmt.Sprintf(`func() (interface{}, error) {
		r %s &net.Resolver{}
		return r.%s(%s)
	}()`, assignOp, contextFunction, argList)

	return fixes.CreateSuggestedFix(fmt.Sprintf("Replace with (*net.Resolver).%s", contextFunction), callExpr, newCall)
}