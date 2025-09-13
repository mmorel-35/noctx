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

// HTTPPost checker for http.Post calls
type HTTPPost struct {
	contextDetector *fixes.ContextDetector
	argFormatter    *fixes.ArgumentFormatter
	assignDetector  *fixes.VariableAssignmentDetector
}

// Check performs the analysis for http.Post calls
func (c *HTTPPost) Check(pass *analysis.Pass) error {
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
	targetFunc := analysisutil.ObjectOf(pass, "net/http", "Post")
	if targetFunc == nil {
		return nil // Function not used
	}

	// Collect call expressions for potential autofix
	callExprs := make(map[token.Pos]*ast.CallExpr)
	allCallExprs := []*ast.CallExpr{}
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		callExpr := n.(*ast.CallExpr)
		if c.isHTTPPostCall(callExpr) {
			callExprs[callExpr.Pos()] = callExpr
			allCallExprs = append(allCallExprs, callExpr)
		}
	})

	// Use SSA-based detection to find violations
	for _, sf := range ssa.SrcFuncs {
		for _, b := range sf.Blocks {
			for _, instr := range b.Instrs {
				if analysisutil.Called(instr, nil, targetFunc.(*types.Func)) {
					funcName := "net/http.Post"
					
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

// isHTTPPostCall checks if a call expression is http.Post
func (c *HTTPPost) isHTTPPostCall(callExpr *ast.CallExpr) bool {
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "http" || sel.Sel.Name != "Post" {
		return false
	}

	return true
}

// findMatchingCallExpr finds a call expression that matches http.Post
func (c *HTTPPost) findMatchingCallExpr(callExprs []*ast.CallExpr) *ast.CallExpr {
	for _, callExpr := range callExprs {
		if c.isHTTPPostCall(callExpr) {
			return callExpr
		}
	}
	return nil
}

// generateSuggestedFix creates a suggested fix for http.Post
func (c *HTTPPost) generateSuggestedFix(pass *analysis.Pass, callExpr *ast.CallExpr) *analysis.SuggestedFix {
	// Verify this is actually a call to http.Post
	if !c.isHTTPPostCall(callExpr) {
		return nil
	}

	// Check that we have the expected number of arguments (url, contentType, body)
	if len(callExpr.Args) != 3 {
		return nil
	}

	// Detect the appropriate context to use
	contextExpr := c.contextDetector.DetectContext(pass, callExpr)
	
	// Get the arguments
	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	contentTypeArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])

	// Detect assignment operator
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")

	// Use http.MethodPost since the original was http.Post (semantic preservation)
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodPost, %s, %s)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", %s)
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, urlArg, bodyArg, contentTypeArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}