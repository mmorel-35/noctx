package fixes

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
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
// When the fix uses context.Background() (or any context.* expression),
// a TextEdit to add the "context" import is also included if it is missing.
func Generate(pass *analysis.Pass, funcName string, ce *ast.CallExpr) *analysis.SuggestedFix {
	fn, ok := funcFixes[funcName]
	if !ok {
		return nil
	}
	cd := NewContextDetector(NewGoVersionDetector())
	ctx := cd.DetectContext(pass, ce)
	fix := fn(pass, ce, ctx)
	if fix == nil {
		return nil
	}

	// If the fix uses the "context" package (e.g. context.Background() or an
	// aliased qualifier like ctx.Background()), ensure the import is present.
	// addImportEdit returns nil when the import already exists so it is safe to
	// call unconditionally when the ctx expression is a qualified call.
	ctxQ := resolveImportLocalName(pass, ce, "context")
	if ctxQ != "" && strings.HasPrefix(ctx, ctxQ+".") {
		if edit := addImportEdit(pass, ce, "context"); edit != nil {
			fix.TextEdits = append(fix.TextEdits, *edit)
		}
	}

	return fix
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
// parameter first. If none is found it checks whether a *testing.T/.B
// parameter is available in a Test*/Benchmark* function on Go 1.24+, and
// finally falls back to context.Background().
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

	// 2. In Test*/Benchmark* functions on Go 1.24+, suggest <t>.Context()
	//    using the actual name of the *testing.T / *testing.B parameter.
	vd := cd.versionDetector
	if vd == nil {
		vd = NewGoVersionDetector()
	}
	if cd.hasTestingImport(pass) && vd.IsGo124OrGreater(pass) {
		if tCtx := cd.findTestingContext(pass, ce); tCtx != "" {
			return tCtx
		}
	}

	// 3. Fallback: use <qualifier>.Background() where <qualifier> is whatever
	//    name the file uses to refer to the "context" package (handles aliases
	//    like ctx "context" or dot-imports of "context").
	q := resolveImportLocalName(pass, ce, "context")
	if q == "" {
		return "Background()" // dot-import of "context"
	}
	return q + ".Background()"
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

// findTestingContext looks for a *testing.T or *testing.B parameter in the
// enclosing Test* or Benchmark* function declaration and returns
// "<paramName>.Context()" if found, or "" otherwise.
// Example* functions are excluded because they do not receive a *testing.T.
func (cd *ContextDetector) findTestingContext(pass *analysis.Pass, ce *ast.CallExpr) string {
	fn := findContainingFunc(pass.Files, ce.Pos())
	if fn == nil {
		return ""
	}
	decl, ok := fn.(*ast.FuncDecl)
	if !ok || decl.Name == nil {
		return ""
	}
	name := decl.Name.Name
	if !strings.HasPrefix(name, "Test") && !strings.HasPrefix(name, "Benchmark") {
		return ""
	}
	if decl.Type.Params == nil {
		return ""
	}
	for _, param := range decl.Type.Params.List {
		if isTestingTOrB(pass, param.Type) && len(param.Names) > 0 {
			return param.Names[0].Name + ".Context()"
		}
	}
	return ""
}

// ── VariableAssignmentDetector ────────────────────────────────────────────────

// VariableAssignmentDetector determines whether to use := or = for variable
// assignments in generated fix code.
type VariableAssignmentDetector struct{}

// DetectAssignmentOperator returns ":=" if any of varNames are not yet declared
// in the package-level scope, and "=" if they are all already declared.
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

// findContainingStmt returns the innermost *ast.AssignStmt or *ast.ExprStmt
// that contains pos. It returns nil if no such statement is found.
func findContainingStmt(files []*ast.File, pos token.Pos) ast.Stmt {
	var result ast.Stmt
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
			switch s := n.(type) {
			case *ast.AssignStmt:
				result = s
			case *ast.ExprStmt:
				result = s
			}
			return true
		})
		break
	}
	return result
}

// stmtLHSText returns the source text of the left-hand side of an assignment
// statement as a comma-separated string. For non-assignment statements or nil,
// it returns "_, _".
func stmtLHSText(pass *analysis.Pass, stmt ast.Stmt) string {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) == 0 {
		return "_, _"
	}
	parts := make([]string, len(assign.Lhs))
	for i, expr := range assign.Lhs {
		s := nodeStr(pass.Fset, expr)
		if s == "" {
			s = "_"
		}
		parts[i] = s
	}
	return strings.Join(parts, ", ")
}

// stmtAssignOp returns ":=" for a short-variable declaration or "=" for plain
// assignment or any other statement type (including ExprStmt and nil).
func stmtAssignOp(stmt ast.Stmt) string {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok {
		return "="
	}
	if assign.Tok == token.DEFINE {
		return ":="
	}
	return "="
}

// isDirectCallInStmt reports whether ce is the direct (non-nested) expression
// inside stmt. For an ExprStmt it checks stmt.X == ce. For an AssignStmt it
// checks that the RHS has exactly one element and that element is ce. This is
// used before emitting a multi-statement replacement so that we never corrupt
// code where the call is nested inside a larger expression or assignment.
func isDirectCallInStmt(stmt ast.Stmt, ce *ast.CallExpr) bool {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return s.X == ce
	case *ast.AssignStmt:
		return len(s.Rhs) == 1 && s.Rhs[0] == ce
	}
	return false
}

// buildErrReturn derives an appropriate error-return statement for the
// enclosing function of ce. When the function returns at least one value
// whose last type is error, it emits "return <zeros…>, err". When there are
// no return values (or none end in error) it emits "_ = err" instead.
func buildErrReturn(pass *analysis.Pass, ce *ast.CallExpr) string {
	fn := findContainingFunc(pass.Files, ce.Pos())
	var results *ast.FieldList
	if fn != nil {
		switch f := fn.(type) {
		case *ast.FuncDecl:
			if f.Type != nil {
				results = f.Type.Results
			}
		case *ast.FuncLit:
			if f.Type != nil {
				results = f.Type.Results
			}
		}
	}
	if results == nil || results.NumFields() == 0 {
		return "_ = err"
	}
	// Check whether the last return type is the builtin error interface.
	lastField := results.List[len(results.List)-1]
	if !isBuiltinErrorType(pass, lastField.Type) {
		return "_ = err"
	}
	// Count total return values.
	total := 0
	for _, f := range results.List {
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		total += n
	}
	if total == 1 {
		return "return err"
	}
	// Multiple return values: use nil for all but the last (error).
	parts := make([]string, total)
	for i := range parts {
		parts[i] = "nil"
	}
	parts[total-1] = "err"
	return "return " + strings.Join(parts, ", ")
}

// isBuiltinErrorType reports whether the expression represents the predeclared
// builtin error interface. It compares against the universe scope's error type
// for a reliable match that is unaffected by string formatting.
func isBuiltinErrorType(pass *analysis.Pass, expr ast.Expr) bool {
	if pass.TypesInfo == nil {
		return false
	}
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

// resolveImportLocalName returns the identifier name used to refer to pkgPath
// in the file that contains ce. If the package is imported under an explicit
// alias that alias is returned. For a dot-import (". ") an empty string is
// returned so callers can omit the qualifier. If the package is not imported
// at all the default last path segment is returned (addImportEdit will add
// the import using that name later).
func resolveImportLocalName(pass *analysis.Pass, ce *ast.CallExpr, pkgPath string) string {
	var file *ast.File
	for _, f := range pass.Files {
		if f.Pos() <= ce.Pos() && ce.Pos() <= f.End() {
			file = f
			break
		}
	}
	if file == nil {
		return lastPathSegment(pkgPath)
	}
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) != pkgPath {
			continue
		}
		if imp.Name != nil {
			if imp.Name.Name == "." {
				return "" // dot-import: no qualifier needed
			}
			return imp.Name.Name
		}
		return lastPathSegment(pkgPath)
	}
	// Not yet imported; addImportEdit will add it with the default name.
	return lastPathSegment(pkgPath)
}

// lastPathSegment returns the last "/" separated element of an import path.
func lastPathSegment(pkgPath string) string {
	if i := strings.LastIndex(pkgPath, "/"); i >= 0 {
		return pkgPath[i+1:]
	}
	return pkgPath
}

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

// isContextType reports whether the expression represents the context.Context
// interface. It uses the types.Named package-path check to match the stdlib
// context.Context reliably, even when the analyzed package imports a different
// package also named "context".
func isContextType(pass *analysis.Pass, expr ast.Expr) bool {
	if pass.TypesInfo == nil {
		return false
	}
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

// isTestingTOrB reports whether the expression is of type *testing.T or
// *testing.B.
func isTestingTOrB(pass *analysis.Pass, expr ast.Expr) bool {
	if pass.TypesInfo == nil {
		return false
	}
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "testing" && (obj.Name() == "T" || obj.Name() == "B")
}

// addImportEdit returns a *analysis.TextEdit that inserts an import for
// pkgPath into the file that contains ce, or nil if the import already exists.
//
// For a parenthesised import block `import ( … )`, the new spec is inserted
// before the closing paren with a leading tab (matching Go's formatting
// conventions). For a single-line import or a file with no imports at all, a
// new import statement is appended on the next line. The result is always
// valid Go — goimports can re-sort it afterwards if desired.
func addImportEdit(pass *analysis.Pass, ce *ast.CallExpr, pkgPath string) *analysis.TextEdit {
	// Find the file that contains the call expression.
	var file *ast.File
	for _, f := range pass.Files {
		if f.Pos() <= ce.Pos() && ce.Pos() <= f.End() {
			file = f
			break
		}
	}
	if file == nil {
		return nil
	}

	// Check whether the package is already imported.
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) == pkgPath {
			return nil
		}
	}

	// Try to find an existing import GenDecl to insert into.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		if genDecl.Lparen.IsValid() {
			// Parenthesised import block: insert before the closing paren.
			// The leading tab matches standard gofmt formatting for import specs.
			return &analysis.TextEdit{
				Pos:     genDecl.Rparen,
				End:     genDecl.Rparen,
				NewText: []byte(fmt.Sprintf("\t%q\n", pkgPath)),
			}
		}
		// Single unparenthesised import: genDecl.End() is the position
		// immediately after the closing quote of the path literal, i.e. the
		// position of the newline that terminates the line. Inserting
		// "\nimport …" there places the new statement on the very next line.
		return &analysis.TextEdit{
			Pos:     genDecl.End(),
			End:     genDecl.End(),
			NewText: []byte(fmt.Sprintf("\nimport %q", pkgPath)),
		}
	}

	// No import block at all: insert after the package clause.
	return &analysis.TextEdit{
		Pos:     file.Name.End(),
		End:     file.Name.End(),
		NewText: []byte(fmt.Sprintf("\n\nimport %q", pkgPath)),
	}
}
