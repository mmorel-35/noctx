package checkers

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/sonatard/noctx/internal/diagnostics"
	"github.com/sonatard/noctx/internal/fixes"
)

// NetDial checker for net.Dial calls
type NetDial struct {
	contextDetector *fixes.ContextDetector
	argFormatter    *fixes.ArgumentFormatter
	assignDetector  *fixes.VariableAssignmentDetector
}

// Check performs the analysis for net.Dial calls
func (c *NetDial) Check(pass *analysis.Pass) error {
	if c.contextDetector == nil {
		c.contextDetector = &fixes.ContextDetector{}
	}
	if c.argFormatter == nil {
		c.argFormatter = &fixes.ArgumentFormatter{}
	}
	if c.assignDetector == nil {
		c.assignDetector = &fixes.VariableAssignmentDetector{}
	}

	// Get required analyzers
	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		return fmt.Errorf("failed to get SSA")
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return fmt.Errorf("failed to get inspector")
	}

	// Find the target function
	targetFunc := analysisutil.ObjectOf(pass, "net", "Dial")
	if targetFunc == nil {
		return nil // Function not used
	}

	// Collect call expressions for potential autofix
	callExprs := make(map[token.Pos]*ast.CallExpr)
	allCallExprs := []*ast.CallExpr{}
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		callExpr := n.(*ast.CallExpr)
		if c.isNetDialCall(callExpr) {
			callExprs[callExpr.Pos()] = callExpr
			allCallExprs = append(allCallExprs, callExpr)
		}
	})

	// Use SSA-based detection to find violations
	for _, sf := range ssa.SrcFuncs {
		for _, b := range sf.Blocks {
			for _, instr := range b.Instrs {
				if analysisutil.Called(instr, nil, targetFunc.(*types.Func)) {
					funcName := "net.Dial"
					
					// Try to find matching call expression for autofix
					var targetCallExpr *ast.CallExpr
					if callExpr, exists := callExprs[instr.Pos()]; exists {
						targetCallExpr = callExpr
					} else {
						// Fallback: search for matching call expression
						targetCallExpr = c.findMatchingCallExpr(allCallExprs)
					}
					
					if targetCallExpr != nil {
						suggestedFix := c.generateSuggestedFix(pass, targetCallExpr)
						if suggestedFix != nil {
							pass.Report(analysis.Diagnostic{
								Pos:            instr.Pos(),
								Message:        diagnostics.FormatDiagnostic(funcName),
								SuggestedFixes: []analysis.SuggestedFix{*suggestedFix},
							})
							continue
						}
					}
					
					// Fallback to regular reporting
					pass.Reportf(instr.Pos(), "%s", diagnostics.FormatDiagnostic(funcName))
				}
			}
		}
	}

	return nil
}

// isNetDialCall checks if a call expression is net.Dial
func (c *NetDial) isNetDialCall(callExpr *ast.CallExpr) bool {
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "net" || sel.Sel.Name != "Dial" {
		return false
	}

	return true
}

// findMatchingCallExpr finds a call expression that matches net.Dial
func (c *NetDial) findMatchingCallExpr(callExprs []*ast.CallExpr) *ast.CallExpr {
	for _, callExpr := range callExprs {
		if c.isNetDialCall(callExpr) {
			return callExpr
		}
	}
	return nil
}

// generateSuggestedFix creates a suggested fix for net.Dial
func (c *NetDial) generateSuggestedFix(pass *analysis.Pass, callExpr *ast.CallExpr) *analysis.SuggestedFix {
	// Verify this is actually a call to net.Dial
	if !c.isNetDialCall(callExpr) {
		return nil
	}

	// Check that we have the expected number of arguments (network, address)
	if len(callExpr.Args) != 2 {
		return nil
	}

	// Detect the appropriate context to use
	contextExpr := c.contextDetector.DetectContext(pass, callExpr)
	
	// Get the arguments
	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])

	// Detect assignment operator
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "conn", "err")

	// Generate the replacement using (*net.Dialer).DialContext
	newCall := fmt.Sprintf(`func() (net.Conn, error) {
		dialer %s &net.Dialer{}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext", callExpr, newCall)
}