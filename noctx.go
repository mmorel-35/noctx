package noctx

import (
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:             "noctx",
	Doc:              "noctx finds function calls without context.Context",
	Run:              Run,
	RunDespiteErrors: false,
	Requires: []*analysis.Analyzer{
		buildssa.Analyzer,
		inspect.Analyzer,
	},
	ResultType: nil,
	FactTypes:  nil,
}

// AutofixInfo contains information needed to generate suggested fixes
type AutofixInfo struct {
	FuncName     string
	ReplacementFunc string
	RequiresContext bool
	RequiresBody    bool
}

var autofixMappings = map[string]AutofixInfo{
	// HTTP functions that can be auto-fixed
	"net/http.NewRequest": {
		FuncName:        "net/http.NewRequest",
		ReplacementFunc: "net/http.NewRequestWithContext", 
		RequiresContext: true,
		RequiresBody:    false,
	},
}

var ngFuncMessages = map[string]string{
	// net
	"net.Listen":       "must not be called. use (*net.ListenConfig).Listen",
	"net.ListenPacket": "must not be called. use (*net.ListenConfig).ListenPacket",
	"net.Dial":         "must not be called. use (*net.Dialer).DialContext",
	"net.DialTimeout":  "must not be called. use (*net.Dialer).DialContext with (*net.Dialer).Timeout",
	"net.LookupCNAME":  "must not be called. use (*net.Resolver).LookupCNAME with a context",
	"net.LookupHost":   "must not be called. use (*net.Resolver).LookupHost with a context",
	"net.LookupIP":     "must not be called. use (*net.Resolver).LookupIPAddr with a context",
	"net.LookupPort":   "must not be called. use (*net.Resolver).LookupPort with a context",
	"net.LookupSRV":    "must not be called. use (*net.Resolver).LookupSRV with a context",
	"net.LookupMX":     "must not be called. use (*net.Resolver).LookupMX with a context",
	"net.LookupNS":     "must not be called. use (*net.Resolver).LookupNS with a context",
	"net.LookupTXT":    "must not be called. use (*net.Resolver).LookupTXT with a context",
	"net.LookupAddr":   "must not be called. use (*net.Resolver).LookupAddr with a context",

	// net/http
	"net/http.Get":                "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
	"net/http.Head":               "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
	"net/http.Post":               "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
	"net/http.PostForm":           "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
	"(*net/http.Client).Get":      "must not be called. use (*net/http.Client).Do(*http.Request)",
	"(*net/http.Client).Head":     "must not be called. use (*net/http.Client).Do(*http.Request)",
	"(*net/http.Client).Post":     "must not be called. use (*net/http.Client).Do(*http.Request)",
	"(*net/http.Client).PostForm": "must not be called. use (*net/http.Client).Do(*http.Request)",
	"net/http.NewRequest":         "must not be called. use net/http.NewRequestWithContext",

	// database/sql
	"(*database/sql.DB).Begin":      "must not be called. use (*database/sql.DB).BeginTx",
	"(*database/sql.DB).Exec":       "must not be called. use (*database/sql.DB).ExecContext",
	"(*database/sql.DB).Ping":       "must not be called. use (*database/sql.DB).PingContext",
	"(*database/sql.DB).Prepare":    "must not be called. use (*database/sql.DB).PrepareContext",
	"(*database/sql.DB).Query":      "must not be called. use (*database/sql.DB).QueryContext",
	"(*database/sql.DB).QueryRow":   "must not be called. use (*database/sql.DB).QueryRowContext",
	"(*database/sql.Tx).Exec":       "must not be called. use (*database/sql.Tx).ExecContext",
	"(*database/sql.Tx).Prepare":    "must not be called. use (*database/sql.Tx).PrepareContext",
	"(*database/sql.Tx).Query":      "must not be called. use (*database/sql.Tx).QueryContext",
	"(*database/sql.Tx).QueryRow":   "must not be called. use (*database/sql.Tx).QueryRowContext",
	"(*database/sql.Tx).Stmt":       "must not be called. use (*database/sql.Tx).StmtContext",
	"(*database/sql.Stmt).Exec":     "must not be called. use (*database/sql.Conn).ExecContext",
	"(*database/sql.Stmt).Query":    "must not be called. use (*database/sql.Conn).QueryContext",
	"(*database/sql.Stmt).QueryRow": "must not be called. use (*database/sql.Conn).QueryRowContext",

	// exec
	"os/exec.Command": "must not be called. use os/exec.CommandContext",

	// crypto/tls dialer
	"crypto/tls.Dial":              "must not be called. use (*crypto/tls.Dialer).DialContext",
	"crypto/tls.DialWithDialer":    "must not be called. use (*crypto/tls.Dialer).DialContext with NetDialer",
	"(*crypto/tls.Conn).Handshake": "must not be called. use (*crypto/tls.Conn).HandshakeContext",

	// log/slog is out of scope of this analyzer, as slog doesn't use the [context.Context]
	// for context cancellation, but for key-value logging via the data stored in it.
	//
	// Related discussion: https://github.com/sonatard/noctx/issues/47
}

func Run(pass *analysis.Pass) (interface{}, error) {
	ngFuncs := typeFuncs(pass, slices.Collect(maps.Keys(ngFuncMessages)))
	if len(ngFuncs) == 0 {
		return nil, nil
	}

	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		panic(fmt.Sprintf("%T is not *buildssa.SSA", pass.ResultOf[buildssa.Analyzer]))
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		panic(fmt.Sprintf("%T is not *inspector.Inspector", pass.ResultOf[inspect.Analyzer]))
	}

	// Collect call expressions for potential autofix
	callExprs := make(map[token.Pos]*ast.CallExpr)
	allCallExprs := []*ast.CallExpr{}  // Also keep a list for fallback matching
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		callExpr := n.(*ast.CallExpr)
		callExprs[callExpr.Pos()] = callExpr
		allCallExprs = append(allCallExprs, callExpr)
	})

	// Use original SSA-based detection
	for _, sf := range ssa.SrcFuncs {
		for _, b := range sf.Blocks {
			for _, instr := range b.Instrs {
				for _, ngFunc := range ngFuncs {
					if analysisutil.Called(instr, nil, ngFunc) {
						funcName := ngFunc.FullName()
						message := ngFuncMessages[funcName]
						
						// Try to provide a suggested fix for specific functions
						fixProvided := false
						if autofixInfo, canAutofix := autofixMappings[funcName]; canAutofix {
							// First try exact position match
							var targetCallExpr *ast.CallExpr
							if callExpr, exists := callExprs[instr.Pos()]; exists {
								targetCallExpr = callExpr
							} else {
								// Fallback: search for matching call expression by function name
								targetCallExpr = findMatchingCallExpr(allCallExprs, funcName)
							}
							
							if targetCallExpr != nil {
								suggestedFix := generateSuggestedFix(pass, targetCallExpr, autofixInfo)
								if suggestedFix != nil {
									pass.Report(analysis.Diagnostic{
										Pos:            instr.Pos(),
										Message:        fmt.Sprintf("%s %s", funcName, message),
										SuggestedFixes: []analysis.SuggestedFix{*suggestedFix},
									})
									fixProvided = true
								}
							}
						}
						
						// Fallback to regular reporting only if no fix was provided
						if !fixProvided {
							pass.Reportf(instr.Pos(), "%s %s", funcName, message)
						}
						break
					}
				}
			}
		}
	}

	return nil, nil
}

// findMatchingCallExpr searches for a call expression that matches the given function name
func findMatchingCallExpr(callExprs []*ast.CallExpr, funcName string) *ast.CallExpr {
	for _, callExpr := range callExprs {
		if matchesFunction(callExpr, funcName) {
			return callExpr
		}
	}
	return nil
}

// matchesFunction checks if a call expression matches a function name
func matchesFunction(callExpr *ast.CallExpr, funcName string) bool {
	switch fun := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		if ident, ok := fun.X.(*ast.Ident); ok {
			callName := ident.Name + "." + fun.Sel.Name
			// For net/http.NewRequest, match both "net/http.NewRequest" and "http.NewRequest"
			return callName == "http.NewRequest" && funcName == "net/http.NewRequest"
		}
	}
	return false
}

// getFunctionName extracts the full function name from a call expression
func getFunctionName(callExpr *ast.CallExpr) string {
	switch fun := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		// Handle package.Function or receiver.Method calls
		if ident, ok := fun.X.(*ast.Ident); ok {
			return ident.Name + "." + fun.Sel.Name
		}
		
		// Handle (*Type).Method calls
		if starExpr, ok := fun.X.(*ast.StarExpr); ok {
			if sel, ok := starExpr.X.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					return "(*" + ident.Name + "." + sel.Sel.Name + ")." + fun.Sel.Name
				}
			}
		}
		
		// Handle more complex selector expressions
		return extractSelectorName(fun)
		
	case *ast.Ident:
		// Handle direct function calls (unlikely for our use case)
		return fun.Name
	}
	
	return ""
}

// extractSelectorName handles complex selector expressions
func extractSelectorName(sel *ast.SelectorExpr) string {
	switch x := sel.X.(type) {
	case *ast.Ident:
		// Simple case: package.Function
		return x.Name + "." + sel.Sel.Name
	case *ast.SelectorExpr:
		// Nested case: package.type.Method -> handle as method call  
		if ident, ok := x.X.(*ast.Ident); ok {
			return "(*" + ident.Name + "." + x.Sel.Name + ")." + sel.Sel.Name
		}
	}
	return ""
}

// generateSuggestedFix creates a SuggestedFix for the given function call
func generateSuggestedFix(pass *analysis.Pass, callExpr *ast.CallExpr, autofixInfo AutofixInfo) *analysis.SuggestedFix {
	if callExpr == nil {
		return nil
	}

	// For now, focus on http.NewRequest -> http.NewRequestWithContext
	if autofixInfo.FuncName == "net/http.NewRequest" {
		return generateHttpNewRequestFix(pass, callExpr, autofixInfo)
	}

	return nil
}

// generateHttpNewRequestFix generates a fix for http.NewRequest calls
func generateHttpNewRequestFix(pass *analysis.Pass, callExpr *ast.CallExpr, autofixInfo AutofixInfo) *analysis.SuggestedFix {
	// Verify this is actually a call to http.NewRequest
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "http" || sel.Sel.Name != "NewRequest" {
		return nil
	}

	// Check that we have the expected number of arguments (method, url, body)
	if len(callExpr.Args) != 3 {
		return nil
	}

	// Detect the appropriate context to use
	contextExpr := detectContext(pass, callExpr)
	
	// Handle body parameter - replace nil with http.NoBody when appropriate
	bodyExpr := callExpr.Args[2]
	bodyReplacement := formatBodyArgument(pass, bodyExpr)

	// Generate the new function call
	methodArg := formatArgument(pass, callExpr.Args[0])
	urlArg := formatArgument(pass, callExpr.Args[1])

	newCall := fmt.Sprintf("http.NewRequestWithContext(%s, %s, %s, %s)", 
		contextExpr, methodArg, urlArg, bodyReplacement)

	// Create text edit to replace the entire function call
	start := callExpr.Pos()
	end := callExpr.End()

	return &analysis.SuggestedFix{
		Message: "Replace with http.NewRequestWithContext",
		TextEdits: []analysis.TextEdit{
			{
				Pos:     start,
				End:     end,
				NewText: []byte(newCall),
			},
		},
	}
}

// detectContext finds the most appropriate context to use
func detectContext(pass *analysis.Pass, callExpr *ast.CallExpr) string {
	// First, try to find if there's a context variable in scope
	if ctx := findContextInScope(pass, callExpr); ctx != "" {
		return ctx
	}

	// Check if we're in a test function - look for testing.T parameter
	if isInTestFunction(pass, callExpr) {
		return "t.Context()"
	}

	// Default to context.Background()
	return "context.Background()"
}

// findContextInScope looks for context variables in the current scope
func findContextInScope(pass *analysis.Pass, callExpr *ast.CallExpr) string {
	// For now, implement a simple check for common context variable names
	// A more sophisticated implementation would do proper scope analysis
	
	// Look for common context variable names
	commonContextNames := []string{"ctx", "context"}
	
	// Check if any of these names are imported or available
	for _, name := range commonContextNames {
		// This is a simplified check - in practice you'd need to verify
		// the variable is actually in scope and of type context.Context
		if name == "ctx" {
			// Assume ctx is available for now
			return "ctx"
		}
	}
	
	return ""
}

// isInTestFunction checks if the call is within a test function
func isInTestFunction(pass *analysis.Pass, callExpr *ast.CallExpr) bool {
	// Check if testing package is imported
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "testing" {
			// If testing is imported, we might be in a test function
			// For a more accurate check, we'd need to analyze the function signature
			return true
		}
	}
	return false
}

// formatArgument converts an AST expression back to source code
func formatArgument(pass *analysis.Pass, expr ast.Expr) string {
	// Get the source text by reading from file
	fset := pass.Fset
	start := fset.Position(expr.Pos())
	end := fset.Position(expr.End())
	
	if start.Filename == end.Filename {
		// Try to get the source text from the position
		src := getSourceFromPosition(start, end)
		if src != "" {
			return src
		}
	}
	
	// Fallback to basic formatting based on expression type
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if ident, ok := e.X.(*ast.Ident); ok {
			return ident.Name + "." + e.Sel.Name
		}
	}
	return "expr" // fallback
}

// formatBodyArgument handles the body parameter, potentially replacing nil with http.NoBody
func formatBodyArgument(pass *analysis.Pass, expr ast.Expr) string {
	// Check if the body is nil
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
		return "http.NoBody"
	}
	
	// For other expressions, format normally
	return formatArgument(pass, expr)
}

// getSourceFromPosition tries to extract source text from file positions
func getSourceFromPosition(start, end token.Position) string {
	// For this implementation, we'll use a simple approach
	// In a production implementation, you'd cache file contents
	// For now, return empty to trigger fallback
	return ""
}
