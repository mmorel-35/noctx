package checkers

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/sonatard/noctx/internal/fixes"
)

// HTTPChecker handles all HTTP-related functions that need context
type HTTPChecker struct {
	contextDetector *fixes.ContextDetector
	argFormatter    *fixes.ArgumentFormatter
	assignDetector  *fixes.VariableAssignmentDetector
}

// NewHTTPChecker creates a new HTTP checker instance
func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		contextDetector: &fixes.ContextDetector{},
		argFormatter:    &fixes.ArgumentFormatter{},
		assignDetector:  &fixes.VariableAssignmentDetector{},
	}
}

// Name returns the name of this checker
func (c *HTTPChecker) Name() CheckerName {
	return HTTPCheckerName
}

// FunctionConfig defines how to handle a specific function
type FunctionConfig struct {
	PackagePath string
	FuncName    string
	Handler     func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix
}

// Check performs the analysis for all HTTP functions
func (c *HTTPChecker) Check(pass *analysis.Pass) error {
	// Define all HTTP functions we handle
	functions := []FunctionConfig{
		{"net/http", "Get", c.generateHTTPGetFix},
		{"net/http", "Head", c.generateHTTPHeadFix},
		{"net/http", "Post", c.generateHTTPPostFix},
		{"net/http", "PostForm", c.generateHTTPPostFormFix},
		{"net/http", "NewRequest", c.generateHTTPNewRequestFix},
	}

	return c.checkFunctions(pass, functions)
}

// getHTTPMessage returns the diagnostic message for HTTP functions
func (c *HTTPChecker) getHTTPMessage(funcName string) string {
	messages := map[string]string{
		"net/http.Get":        "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		"net/http.Head":       "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		"net/http.Post":       "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		"net/http.PostForm":   "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		"net/http.NewRequest": "must not be called. use net/http.NewRequestWithContext",
	}
	if msg, exists := messages[funcName]; exists {
		return funcName + " " + msg
	}
	return funcName + " must not be called without context"
}

// checkFunctions is the shared logic for checking multiple functions
func (c *HTTPChecker) checkFunctions(pass *analysis.Pass, functions []FunctionConfig) error {
	// Get required analyzers
	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		return fmt.Errorf("failed to get SSA")
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return fmt.Errorf("failed to get inspector")
	}

	// Collect call expressions for potential autofix
	callExprs := make(map[token.Pos]*ast.CallExpr)
	allCallExprs := []*ast.CallExpr{}
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		callExpr := n.(*ast.CallExpr)
		callExprs[callExpr.Pos()] = callExpr
		allCallExprs = append(allCallExprs, callExpr)
	})

	// Process each function
	for _, funcConfig := range functions {
		targetFunc := analysisutil.ObjectOf(pass, funcConfig.PackagePath, funcConfig.FuncName)
		if targetFunc == nil {
			continue // Function not used
		}

		// Use SSA-based detection to find violations
		for _, sf := range ssa.SrcFuncs {
			for _, b := range sf.Blocks {
				for _, instr := range b.Instrs {
					if analysisutil.Called(instr, nil, targetFunc.(*types.Func)) {
						funcName := fmt.Sprintf("%s.%s", funcConfig.PackagePath, funcConfig.FuncName)
						
						// Try to find matching call expression for autofix
						var targetCallExpr *ast.CallExpr
						if callExpr, exists := callExprs[instr.Pos()]; exists {
							targetCallExpr = callExpr
						} else {
							// Fallback: search for matching call expression
							targetCallExpr = c.findMatchingCallExpr(allCallExprs, funcConfig)
						}
						
						if targetCallExpr != nil {
							contextExpr := c.contextDetector.DetectContext(pass, targetCallExpr)
							suggestedFix := funcConfig.Handler(pass, targetCallExpr, contextExpr)
							if suggestedFix != nil {
								pass.Report(analysis.Diagnostic{
									Pos:            instr.Pos(),
									Message:        c.getHTTPMessage(funcName),
									SuggestedFixes: []analysis.SuggestedFix{*suggestedFix},
								})
								continue
							}
						}
						
						// Fallback to regular reporting
						pass.Reportf(instr.Pos(), "%s", c.getHTTPMessage(funcName))
					}
				}
			}
		}
	}

	return nil
}

// findMatchingCallExpr finds a call expression that matches the function config
func (c *HTTPChecker) findMatchingCallExpr(callExprs []*ast.CallExpr, funcConfig FunctionConfig) *ast.CallExpr {
	expectedPkg := funcConfig.PackagePath
	if strings.Contains(expectedPkg, "/") {
		parts := strings.Split(expectedPkg, "/")
		expectedPkg = parts[len(parts)-1]
	}

	for _, callExpr := range callExprs {
		if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == expectedPkg && sel.Sel.Name == funcConfig.FuncName {
					return callExpr
				}
			}
		}
	}
	return nil
}

// HTTP Fix Generators

func (c *HTTPChecker) generateHTTPGetFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodGet, %s, http.NoBody)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, urlArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}

func (c *HTTPChecker) generateHTTPHeadFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodHead, %s, http.NoBody)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, urlArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}

func (c *HTTPChecker) generateHTTPPostFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	contentTypeArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
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

func (c *HTTPChecker) generateHTTPPostFormFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	dataArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodPost, %s, strings.NewReader(%s.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, urlArg, dataArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}

func (c *HTTPChecker) generateHTTPNewRequestFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	methodArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	
	// Replace nil body with http.NoBody for better performance
	if bodyArg == "nil" {
		bodyArg = "http.NoBody"
	}
	
	newCall := fmt.Sprintf("http.NewRequestWithContext(%s, %s, %s, %s)", contextExpr, methodArg, urlArg, bodyArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext", callExpr, newCall)
}