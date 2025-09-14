package fixes

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// SuggestedFix contains information for generating autofix suggestions
type SuggestedFix struct {
	Message   string
	TextEdits []analysis.TextEdit
}

// ContextDetector detects the most appropriate context to use in fixes
type ContextDetector struct {
	versionDetector *GoVersionDetector
}

// NewContextDetector creates a new context detector with version detection
func NewContextDetector(versionDetector *GoVersionDetector) *ContextDetector {
	return &ContextDetector{
		versionDetector: versionDetector,
	}
}

// DetectContext finds the most appropriate context expression to use
func (cd *ContextDetector) DetectContext(pass *analysis.Pass, callExpr *ast.CallExpr) string {
	// First, try to find a context variable in the current function scope
	if contextVar := cd.findContextVariable(pass, callExpr); contextVar != "" {
		return contextVar
	}

	// Check if testing package is imported (need to guard against nil package)
	hasTestingImport := false
	if pass.Pkg != nil {
		for _, pkg := range pass.Pkg.Imports() {
			if pkg.Path() == "testing" {
				hasTestingImport = true
				break
			}
		}
	}
	
	// If testing is imported and Go version is 1.24+, use t.Context() for test functions
	if hasTestingImport && cd.versionDetector.IsGo124OrGreater(pass) && cd.isInTestFunction(pass, callExpr) {
		return "t.Context()"
	}

	// Check if context package is imported (need to guard against nil package)
	hasContextImport := false
	if pass.Pkg != nil {
		for _, pkg := range pass.Pkg.Imports() {
			if pkg.Path() == "context" {
				hasContextImport = true
				break
			}
		}
	}
	
	// If context is imported, assume ctx is available as a common pattern
	if hasContextImport {
		return "ctx"
	}

	// Default to context.Background()
	return "context.Background()"
}

// findContextVariable searches for context variables in the current function scope
func (cd *ContextDetector) findContextVariable(pass *analysis.Pass, callExpr *ast.CallExpr) string {
	// Walk up the AST to find the containing function
	var containingFunc *ast.FuncDecl
	
	// Find the function that contains this call expression
	// We need to search through all files in the pass
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok {
				// Check if callExpr is within this function
				if callExpr.Pos() >= funcDecl.Pos() && callExpr.End() <= funcDecl.End() {
					containingFunc = funcDecl
					return false // Found it, stop walking
				}
			}
			return true
		})
		if containingFunc != nil {
			break // Found the function, no need to search other files
		}
	}

	if containingFunc == nil {
		return ""
	}

	// Check function parameters for context
	if containingFunc.Type.Params != nil {
		for _, param := range containingFunc.Type.Params.List {
			if cd.isContextType(pass, param.Type) && len(param.Names) > 0 {
				return param.Names[0].Name
			}
		}
	}

	// TODO: Check local variables declared within the function
	// This would require more sophisticated AST walking

	return ""
}

// isContextType checks if a type expression represents context.Context
func (cd *ContextDetector) isContextType(pass *analysis.Pass, expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		// Check for context.Context
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "context" && t.Sel.Name == "Context"
		}
	case *ast.Ident:
		// Check if it's an aliased context type
		if obj := pass.TypesInfo.ObjectOf(t); obj != nil {
			if named, ok := obj.Type().(*types.Named); ok {
				return named.String() == "context.Context"
			}
		}
	}
	return false
}

// isGoVersion124OrLater checks if the current Go version is 1.24 or later
func (cd *ContextDetector) isGoVersion124OrLater(pass *analysis.Pass) bool {
	// Try to get Go version from module info
	// This is a simplified approach - in production, you might want to
	// parse go.mod file or use build constraints
	
	// For now, we'll be conservative and assume Go 1.24+ is available
	// TODO: Implement proper Go version detection by reading go.mod
	return true // Placeholder - always assume modern Go version for now
}

// isInTestFunction checks if the call expression is within a test function
func (cd *ContextDetector) isInTestFunction(pass *analysis.Pass, callExpr *ast.CallExpr) bool {
	// Find the containing function
	var containingFunc *ast.FuncDecl
	
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok {
				// Check if callExpr is within this function
				if callExpr.Pos() >= funcDecl.Pos() && callExpr.End() <= funcDecl.End() {
					containingFunc = funcDecl
					return false // Found it, stop walking
				}
			}
			return true
		})
		if containingFunc != nil {
			break // Found the function, no need to search other files
		}
	}

	if containingFunc == nil || containingFunc.Name == nil {
		return false
	}

	// Check if function name starts with "Test", "Benchmark", or "Example"
	funcName := containingFunc.Name.Name
	return strings.HasPrefix(funcName, "Test") ||
		strings.HasPrefix(funcName, "Benchmark") ||
		strings.HasPrefix(funcName, "Example")
}

// VariableAssignmentDetector detects whether to use := or = for variable assignments
type VariableAssignmentDetector struct{}

// DetectAssignmentOperator determines whether to use := or = based on context
func (vad *VariableAssignmentDetector) DetectAssignmentOperator(pass *analysis.Pass, callExpr *ast.CallExpr, varNames ...string) string {
	// Check if variables are already declared in the current scope
	// Walk up to find containing function or block
	containingScope := vad.findContainingScope(pass, callExpr)
	if containingScope == nil {
		return ":=" // Default to declaration assignment
	}

	// Check if any of the specified variables are already declared
	allDeclared := true
	for _, varName := range varNames {
		if !vad.isVariableDeclared(pass, containingScope, varName) {
			allDeclared = false
			break
		}
	}

	if allDeclared {
		return "=" // All variables are declared, use assignment
	} else {
		return ":=" // At least one variable needs declaration
	}
}

// findContainingScope finds the scope that contains the call expression
func (vad *VariableAssignmentDetector) findContainingScope(pass *analysis.Pass, callExpr *ast.CallExpr) *types.Scope {
	// This is a simplified implementation
	// In a more robust version, we'd walk the AST to find the exact scope
	return pass.Pkg.Scope()
}

// isVariableDeclared checks if a variable is declared in the given scope
func (vad *VariableAssignmentDetector) isVariableDeclared(pass *analysis.Pass, scope *types.Scope, varName string) bool {
	// Check if variable exists in scope
	obj := scope.Lookup(varName)
	return obj != nil
}

// ArgumentFormatter helps format AST expressions back to source code
type ArgumentFormatter struct{}

// FormatArgument converts an AST expression to source code string
func (af *ArgumentFormatter) FormatArgument(pass *analysis.Pass, expr ast.Expr) string {
	// Get the source text by reading from file position
	fset := pass.Fset
	start := fset.Position(expr.Pos())
	end := fset.Position(expr.End())
	
	if start.Filename == end.Filename {
		// Try to get the source text from the position
		src := af.getSourceFromPosition(start, end)
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

// FormatBodyArgument handles body parameters, potentially replacing nil with http.NoBody
func (af *ArgumentFormatter) FormatBodyArgument(pass *analysis.Pass, expr ast.Expr) string {
	// Check if the body is nil
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
		return "http.NoBody"
	}
	
	// For other expressions, format normally
	return af.FormatArgument(pass, expr)
}

// getSourceFromPosition tries to extract source text from file positions
func (af *ArgumentFormatter) getSourceFromPosition(start, end token.Position) string {
	// For this implementation, we'll use a simple approach
	// In a production implementation, you'd cache file contents
	// For now, return empty to trigger fallback
	return ""
}

// CreateTextEdit creates a text edit for replacing a function call
func CreateTextEdit(callExpr *ast.CallExpr, newCall string) analysis.TextEdit {
	return analysis.TextEdit{
		Pos:     callExpr.Pos(),
		End:     callExpr.End(),
		NewText: []byte(newCall),
	}
}

// CreateSuggestedFix creates a suggested fix with a single text edit
func CreateSuggestedFix(message string, callExpr *ast.CallExpr, newCall string) *analysis.SuggestedFix {
	return &analysis.SuggestedFix{
		Message:   message,
		TextEdits: []analysis.TextEdit{CreateTextEdit(callExpr, newCall)},
	}
}