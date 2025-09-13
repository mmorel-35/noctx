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
	"net/http.Get": {
		FuncName:        "net/http.Get",
		ReplacementFunc: "net/http.NewRequestWithContext",
		RequiresContext: true,
		RequiresBody:    false,
	},
	"net/http.Head": {
		FuncName:        "net/http.Head", 
		ReplacementFunc: "net/http.NewRequestWithContext",
		RequiresContext: true,
		RequiresBody:    false,
	},
	"net/http.Post": {
		FuncName:        "net/http.Post",
		ReplacementFunc: "net/http.NewRequestWithContext",
		RequiresContext: true,
		RequiresBody:    true,
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
			// For net/http functions, match both "net/http.Function" and "http.Function"
			switch funcName {
			case "net/http.NewRequest":
				return callName == "http.NewRequest"
			case "net/http.Get":
				return callName == "http.Get"
			case "net/http.Head":
				return callName == "http.Head"
			case "net/http.Post":
				return callName == "http.Post"
			}
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

	// Handle different HTTP functions
	switch autofixInfo.FuncName {
	case "net/http.NewRequest":
		return generateHttpNewRequestFix(pass, callExpr, autofixInfo)
	case "net/http.Get", "net/http.Head":
		return generateHttpGetHeadFix(pass, callExpr, autofixInfo)
	case "net/http.Post":
		return generateHttpPostFix(pass, callExpr, autofixInfo)
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

// generateHttpGetHeadFix generates a fix for http.Get and http.Head calls
func generateHttpGetHeadFix(pass *analysis.Pass, callExpr *ast.CallExpr, autofixInfo AutofixInfo) *analysis.SuggestedFix {
	// Verify this is actually a call to http.Get or http.Head
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "http" {
		return nil
	}

	// Check that we have the expected number of arguments (just url)
	if len(callExpr.Args) != 1 {
		return nil
	}

	// Detect the appropriate context to use
	contextExpr := detectContext(pass, callExpr)
	
	// Get the URL argument
	urlArg := formatArgument(pass, callExpr.Args[0])
	
	// Generate the replacement code
	method := ""
	if sel.Sel.Name == "Get" {
		method = "GET"
	} else if sel.Sel.Name == "Head" {
		method = "HEAD"
	}
	
	// Generate a simpler replacement for now
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(%s, %q, %s, http.NoBody)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}()`, contextExpr, method, urlArg)

	// Create text edit to replace the entire function call
	start := callExpr.Pos()
	end := callExpr.End()

	return &analysis.SuggestedFix{
		Message: "Replace with http.NewRequestWithContext and Do",
		TextEdits: []analysis.TextEdit{
			{
				Pos:     start,
				End:     end,
				NewText: []byte(newCall),
			},
		},
	}
}

// generateHttpPostFix generates a fix for http.Post calls
func generateHttpPostFix(pass *analysis.Pass, callExpr *ast.CallExpr, autofixInfo AutofixInfo) *analysis.SuggestedFix {
	// Verify this is actually a call to http.Post
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "http" || sel.Sel.Name != "Post" {
		return nil
	}

	// Check that we have the expected number of arguments (url, contentType, body)
	if len(callExpr.Args) != 3 {
		return nil
	}

	// Detect the appropriate context to use
	contextExpr := detectContext(pass, callExpr)
	
	// Get the arguments
	urlArg := formatArgument(pass, callExpr.Args[0])
	contentTypeArg := formatArgument(pass, callExpr.Args[1])
	bodyArg := formatArgument(pass, callExpr.Args[2])

	// Generate the replacement code
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(%s, "POST", %s, %s)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", %s)
		return http.DefaultClient.Do(req)
	}()`, contextExpr, urlArg, bodyArg, contentTypeArg)

	// Create text edit to replace the entire function call
	start := callExpr.Pos()
	end := callExpr.End()

	return &analysis.SuggestedFix{
		Message: "Replace with http.NewRequestWithContext and Do",
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
	// Simple heuristics for context detection
	// Check if testing package is imported (suggests we might be in tests)
	hasTestingImport := false
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "testing" {
			hasTestingImport = true
			break
		}
	}
	
	// If testing is imported, prefer t.Context() for now
	// In a real implementation, we'd check the actual function signature
	if hasTestingImport {
		return "t.Context()"
	}

	// Check if context package is imported (suggests context variables might be available)
	hasContextImport := false
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "context" {
			hasContextImport = true
			break
		}
	}
	
	// If context is imported and we're in a function that likely has a context parameter,
	// assume ctx is available
	if hasContextImport {
		return "ctx"
	}

	// Default to context.Background()
	return "context.Background()"
}

// findEnclosingFunction finds the function declaration that contains the call expression
func findEnclosingFunction(callExpr *ast.CallExpr) *ast.FuncDecl {
	// In a real implementation, you'd walk up the AST
	// For now, this is a simplified placeholder that returns nil
	// A proper implementation would need to maintain parent pointers or use ast.Inspect
	return nil
}

// isTestFunction checks if a function declaration is a test function
func isTestFunction(funcDecl *ast.FuncDecl) bool {
	if funcDecl == nil || funcDecl.Type.Params == nil {
		return false
	}
	
	// Check if the function has a parameter of type *testing.T
	for _, param := range funcDecl.Type.Params.List {
		if starExpr, ok := param.Type.(*ast.StarExpr); ok {
			if selExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
				if ident, ok := selExpr.X.(*ast.Ident); ok {
					if ident.Name == "testing" && selExpr.Sel.Name == "T" {
						return true
					}
				}
			}
		}
	}
	return false
}

// findContextParameter looks for context.Context parameters in a function
func findContextParameter(pass *analysis.Pass, funcDecl *ast.FuncDecl) string {
	if funcDecl == nil || funcDecl.Type.Params == nil {
		return ""
	}
	
	for _, param := range funcDecl.Type.Params.List {
		if isContextParam(pass, param) && len(param.Names) > 0 {
			return param.Names[0].Name
		}
	}
	return ""
}

// isContextParam checks if a parameter is of type context.Context
func isContextParam(pass *analysis.Pass, param *ast.Field) bool {
	if selExpr, ok := param.Type.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			return ident.Name == "context" && selExpr.Sel.Name == "Context"
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
