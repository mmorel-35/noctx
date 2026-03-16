package fixes

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"

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
	return fn(pass, ce, detectContext(pass, ce))
}

// ── helpers ──────────────────────────────────────────────────────────────────

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

// detectContext finds the best context expression to use at the call site.
// It searches the enclosing function or func-literal for a context.Context
// parameter. If none is found, it falls back to context.Background().
func detectContext(pass *analysis.Pass, ce *ast.CallExpr) string {
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
	return "context.Background()"
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
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	return t.String() == "context.Context"
}

// createFix builds an analysis.SuggestedFix that replaces the entire call
// expression with newText.
func createFix(message string, ce *ast.CallExpr, newText string) *analysis.SuggestedFix {
	return &analysis.SuggestedFix{
		Message: message,
		TextEdits: []analysis.TextEdit{
			{
				Pos:     ce.Pos(),
				End:     ce.End(),
				NewText: []byte(newText),
			},
		},
	}
}
