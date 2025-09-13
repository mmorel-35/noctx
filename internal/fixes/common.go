package fixes

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// SuggestedFix contains information for generating autofix suggestions
type SuggestedFix struct {
	Message   string
	TextEdits []analysis.TextEdit
}

// ContextDetector detects the most appropriate context to use in fixes
type ContextDetector struct{}

// DetectContext finds the most appropriate context expression to use
func (cd *ContextDetector) DetectContext(pass *analysis.Pass, callExpr *ast.CallExpr) string {
	// Check if testing package is imported
	hasTestingImport := false
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "testing" {
			hasTestingImport = true
			break
		}
	}
	
	// If testing is imported, use t.Context() for test functions
	if hasTestingImport {
		return "t.Context()"
	}

	// Check if context package is imported
	hasContextImport := false
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "context" {
			hasContextImport = true
			break
		}
	}
	
	// If context is imported, assume ctx is available
	// TODO: Improve this by actually analyzing function parameters
	if hasContextImport {
		return "ctx"
	}

	// Default to context.Background()
	return "context.Background()"
}

// VariableAssignmentDetector detects whether to use := or = for variable assignments
type VariableAssignmentDetector struct{}

// DetectAssignmentOperator determines whether to use := or = based on context
func (vad *VariableAssignmentDetector) DetectAssignmentOperator(pass *analysis.Pass, callExpr *ast.CallExpr, varNames ...string) string {
	// TODO: Implement logic to detect if variables are already declared
	// For now, default to := (declaration assignment)
	return ":="
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