package fixes

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// fixFunc is the signature of a per-function fix generator.
// It returns nil when a fix cannot be constructed (e.g. wrong argument count).
type fixFunc func(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix

// funcFixes maps each flagged function name to its fix generator.
// Entries are grouped by package to mirror noctx.ngFuncMessages.
var funcFixes = map[string]fixFunc{
	// net
	"net.Listen":       fixNetListen,
	"net.ListenPacket": fixNetListenPacket,
	"net.Dial":         fixNetDial,
	"net.DialTimeout":  fixNetDialTimeout,
	"net.LookupCNAME":  netResolverFix("LookupCNAME"),
	"net.LookupHost":   netResolverFix("LookupHost"),
	"net.LookupIP":     fixNetLookupIP,
	"net.LookupPort":   fixNetLookupPort,
	"net.LookupSRV":    fixNetLookupSRV,
	"net.LookupMX":     netResolverFix("LookupMX"),
	"net.LookupNS":     netResolverFix("LookupNS"),
	"net.LookupTXT":    netResolverFix("LookupTXT"),
	"net.LookupAddr":   netResolverFix("LookupAddr"),

	// net/http
	"net/http.Get":        fixHTTPGet,
	"net/http.Head":       fixHTTPHead,
	"net/http.Post":       fixHTTPPost,
	"net/http.PostForm":   fixHTTPPostForm,
	"net/http.NewRequest": fixHTTPNewRequest,

	// net/http/httptest
	"net/http/httptest.NewRequest": fixHTTPTestNewRequest,

	// os/exec
	"os/exec.Command": fixExecCommand,

	// crypto/tls
	"crypto/tls.Dial":           fixTLSDial,
	"crypto/tls.DialWithDialer": fixTLSDialWithDialer,
}

// Generate looks up and calls the fix generator for funcName.
// It returns nil when no fix is available for the function.
func Generate(pass *analysis.Pass, funcName string, ce *ast.CallExpr) *analysis.SuggestedFix {
	fn, ok := funcFixes[funcName]
	if !ok {
		return nil
	}
	cd := NewContextDetector(NewGoVersionDetector())
	return fn(pass, ce, cd.DetectContext(pass, ce))
}

// ── ContextDetector ───────────────────────────────────────────────────────────

// ContextDetector detects the most appropriate context expression to use in fixes.
type ContextDetector struct {
	versionDetector *GoVersionDetector
}

// NewContextDetector creates a new ContextDetector backed by the supplied
// GoVersionDetector.
func NewContextDetector(vd *GoVersionDetector) *ContextDetector {
	return &ContextDetector{versionDetector: vd}
}

// DetectContext finds the best context expression to use at the call site.
// It searches the enclosing function or func-literal for a context.Context
// parameter first. If none is found it checks whether t.Context() is
// appropriate (Go 1.24+ test functions), and finally falls back to
// context.Background().
func (cd *ContextDetector) DetectContext(pass *analysis.Pass, ce *ast.CallExpr) string {
	// 1. Look for a context.Context parameter in the enclosing function/literal.
	fn := findContainingFunc(pass.Files, ce.Pos())
	if fn != nil {
		var params *ast.FieldList
		switch f := fn.(type) {
		case *ast.FuncDecl:
			params = f.Type.Params
		case *ast.FuncLit:
			params = f.Type.Params
		}
		if params != nil {
			for _, param := range params.List {
				if isContextType(pass, param.Type) && len(param.Names) > 0 {
					return param.Names[0].Name
				}
			}
		}
	}

	// 2. In test functions on Go 1.24+, suggest t.Context().
	vd := cd.versionDetector
	if vd == nil {
		vd = NewGoVersionDetector()
	}
	if cd.hasTestingImport(pass) && vd.IsGo124OrGreater(pass) && cd.isInTestFunction(pass, ce) {
		return "t.Context()"
	}

	// 3. Fallback.
	return "context.Background()"
}

// hasTestingImport reports whether the package under analysis imports "testing".
func (cd *ContextDetector) hasTestingImport(pass *analysis.Pass) bool {
	if pass.Pkg == nil {
		return false
	}
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "testing" {
			return true
		}
	}
	return false
}

// isInTestFunction reports whether the call expression is inside a function
// whose name starts with Test, Benchmark, or Example.
func (cd *ContextDetector) isInTestFunction(pass *analysis.Pass, ce *ast.CallExpr) bool {
	fn := findContainingFunc(pass.Files, ce.Pos())
	if fn == nil {
		return false
	}
	decl, ok := fn.(*ast.FuncDecl)
	if !ok || decl.Name == nil {
		return false
	}
	name := decl.Name.Name
	return strings.HasPrefix(name, "Test") ||
		strings.HasPrefix(name, "Benchmark") ||
		strings.HasPrefix(name, "Example")
}

// ── VariableAssignmentDetector ────────────────────────────────────────────────

// VariableAssignmentDetector determines whether to use := or = for variable
// assignments in generated fix code.
type VariableAssignmentDetector struct{}

// DetectAssignmentOperator returns ":=" if any of varNames are not yet declared
// in the enclosing scope, and "=" if they are all already declared.
//
// Note: this implementation uses the package-level scope as a simplified
// approximation. Block-scoped variables declared within functions are not
// visible at package scope, so this will conservatively return ":=" for most
// call sites inside function bodies, which is the safe default.
func (vad *VariableAssignmentDetector) DetectAssignmentOperator(pass *analysis.Pass, _ *ast.CallExpr, varNames ...string) string {
	if pass.Pkg == nil {
		return ":="
	}
	scope := pass.Pkg.Scope()
	for _, name := range varNames {
		if scope.Lookup(name) == nil {
			return ":="
		}
	}
	return "="
}

// ── ArgumentFormatter ─────────────────────────────────────────────────────────

// ArgumentFormatter converts AST expressions back to source-code strings.
type ArgumentFormatter struct{}

// FormatArgument returns the source text of an AST expression.
func (af *ArgumentFormatter) FormatArgument(pass *analysis.Pass, expr ast.Expr) string {
	s := nodeStr(pass.Fset, expr)
	if s != "" {
		return s
	}
	// Fallback for basic node types that format.Node may not handle without a
	// proper FileSet.
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
	return "expr"
}

// FormatBodyArgument is like FormatArgument but replaces a nil body with
// http.NoBody, which is the idiomatic replacement when building an HTTP
// request without a body.
func (af *ArgumentFormatter) FormatBodyArgument(pass *analysis.Pass, expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
		return "http.NoBody"
	}
	return af.FormatArgument(pass, expr)
}

// ── Text-edit helpers (exported) ──────────────────────────────────────────────

// CreateTextEdit creates a TextEdit that replaces the entire call expression
// with newText.
func CreateTextEdit(ce *ast.CallExpr, newText string) analysis.TextEdit {
	return analysis.TextEdit{
		Pos:     ce.Pos(),
		End:     ce.End(),
		NewText: []byte(newText),
	}
}

// CreateSuggestedFix creates a SuggestedFix with a single text edit that
// replaces the entire call expression with newText.
func CreateSuggestedFix(message string, ce *ast.CallExpr, newText string) *analysis.SuggestedFix {
	return &analysis.SuggestedFix{
		Message:   message,
		TextEdits: []analysis.TextEdit{CreateTextEdit(ce, newText)},
	}
}

// ── internal helpers ──────────────────────────────────────────────────────────

// extractQualifier returns the package qualifier used at the call site with a
// trailing dot (e.g. "http.", "h.", "exec."), or an empty string when the
// package was dot-imported (import . "net/http") so callers can always
// concatenate it directly with the identifier name.
//
// Blank imports (import _ "pkg") cannot produce call expressions for exported
// functions, so that case never arises here.
func extractQualifier(ce *ast.CallExpr) string {
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok {
		// Dot-import: function name is a plain *ast.Ident with no qualifier.
		return ""
	}
	if ident, ok := sel.X.(*ast.Ident); ok {
		return ident.Name + "."
	}
	return ""
}

// nodeStr converts an AST node back to its source representation using go/format.
func nodeStr(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}

// arg returns the formatted source text of the i-th argument of a call expression.
func arg(pass *analysis.Pass, ce *ast.CallExpr, i int) string {
	if i >= len(ce.Args) {
		return "?"
	}
	s := nodeStr(pass.Fset, ce.Args[i])
	if s == "" {
		return "?"
	}
	return s
}

// createFix builds a SuggestedFix that replaces the entire call expression
// with newText. It delegates to CreateSuggestedFix.
func createFix(message string, ce *ast.CallExpr, newText string) *analysis.SuggestedFix {
	return CreateSuggestedFix(message, ce, newText)
}

// findContainingFunc returns the innermost *ast.FuncDecl or *ast.FuncLit that
// contains pos.
func findContainingFunc(files []*ast.File, pos token.Pos) ast.Node {
	var result ast.Node
	for _, file := range files {
		if file.Pos() > pos || pos > file.End() {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			if n.Pos() > pos || pos > n.End() {
				return false
			}
			switch fn := n.(type) {
			case *ast.FuncDecl:
				if fn.Body != nil && fn.Body.Pos() <= pos && pos <= fn.Body.End() {
					result = fn
				}
			case *ast.FuncLit:
				if fn.Body != nil && fn.Body.Pos() <= pos && pos <= fn.Body.End() {
					result = fn
				}
			}
			return true
		})
		break
	}
	return result
}

// isContextType reports whether the expression represents the context.Context type.
func isContextType(pass *analysis.Pass, expr ast.Expr) bool {
	if pass.TypesInfo == nil {
		return false
	}
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	return t.String() == "context.Context"
}
