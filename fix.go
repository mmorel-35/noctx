package noctx

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

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
// It searches the enclosing function/func-literal for a context.Context parameter.
// If none is found, it falls back to context.Background().
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
// contains pos. Walking the AST in pre-order and updating on each match ensures
// the last update is the deepest (innermost) containing function.
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

// createFix builds an analysis.SuggestedFix that replaces the entire call expression
// with newText.
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

// generateFix dispatches to the appropriate fix generator for funcName.
// It returns nil when no fix is available for the given function.
func generateFix(pass *analysis.Pass, funcName string, ce *ast.CallExpr) *analysis.SuggestedFix {
	ctx := detectContext(pass, ce)
	switch funcName {
	case "net/http.NewRequest":
		return fixHTTPNewRequest(pass, ce, ctx)
	case "net/http/httptest.NewRequest":
		return fixHTTPTestNewRequest(pass, ce, ctx)
	case "net/http.Get":
		return fixHTTPGet(pass, ce, ctx)
	case "net/http.Head":
		return fixHTTPHead(pass, ce, ctx)
	case "net/http.Post":
		return fixHTTPPost(pass, ce, ctx)
	case "net/http.PostForm":
		return fixHTTPPostForm(pass, ce, ctx)
	case "os/exec.Command":
		return fixExecCommand(pass, ce, ctx)
	case "net.Dial":
		return fixNetDial(pass, ce, ctx)
	case "net.DialTimeout":
		return fixNetDialTimeout(pass, ce, ctx)
	case "net.Listen":
		return fixNetListen(pass, ce, ctx)
	case "net.ListenPacket":
		return fixNetListenPacket(pass, ce, ctx)
	case "net.LookupCNAME":
		return fixNetLookup1(pass, ce, ctx, "LookupCNAME")
	case "net.LookupHost":
		return fixNetLookup1(pass, ce, ctx, "LookupHost")
	case "net.LookupIP":
		return fixNetLookupIP(pass, ce, ctx)
	case "net.LookupMX":
		return fixNetLookup1(pass, ce, ctx, "LookupMX")
	case "net.LookupNS":
		return fixNetLookup1(pass, ce, ctx, "LookupNS")
	case "net.LookupTXT":
		return fixNetLookup1(pass, ce, ctx, "LookupTXT")
	case "net.LookupAddr":
		return fixNetLookup1(pass, ce, ctx, "LookupAddr")
	case "net.LookupPort":
		return fixNetLookupPort(pass, ce, ctx)
	case "net.LookupSRV":
		return fixNetLookupSRV(pass, ce, ctx)
	case "crypto/tls.Dial":
		return fixTLSDial(pass, ce, ctx)
	case "crypto/tls.DialWithDialer":
		return fixTLSDialWithDialer(pass, ce, ctx)
	}
	return nil
}

func fixHTTPNewRequest(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	method := arg(pass, ce, 0)
	url := arg(pass, ce, 1)
	body := arg(pass, ce, 2)
	newText := fmt.Sprintf("http.NewRequestWithContext(%s, %s, %s, %s)", ctx, method, url, body)
	return createFix("Replace http.NewRequest with http.NewRequestWithContext", ce, newText)
}

func fixHTTPTestNewRequest(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	method := arg(pass, ce, 0)
	target := arg(pass, ce, 1)
	body := arg(pass, ce, 2)
	newText := fmt.Sprintf("httptest.NewRequestWithContext(%s, %s, %s, %s)", ctx, method, target, body)
	return createFix("Replace httptest.NewRequest with httptest.NewRequestWithContext", ce, newText)
}

func fixHTTPGet(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 1 {
		return nil
	}
	url := arg(pass, ce, 0)
	newText := fmt.Sprintf(`func() (*http.Response, error) {
	req, err := http.NewRequestWithContext(%s, http.MethodGet, %s, http.NoBody)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}()`, ctx, url)
	return createFix("Replace http.Get with http.NewRequestWithContext", ce, newText)
}

func fixHTTPHead(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 1 {
		return nil
	}
	url := arg(pass, ce, 0)
	newText := fmt.Sprintf(`func() (*http.Response, error) {
	req, err := http.NewRequestWithContext(%s, http.MethodHead, %s, http.NoBody)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}()`, ctx, url)
	return createFix("Replace http.Head with http.NewRequestWithContext", ce, newText)
}

func fixHTTPPost(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	url := arg(pass, ce, 0)
	contentType := arg(pass, ce, 1)
	body := arg(pass, ce, 2)
	newText := fmt.Sprintf(`func() (*http.Response, error) {
	req, err := http.NewRequestWithContext(%s, http.MethodPost, %s, %s)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", %s)
	return http.DefaultClient.Do(req)
}()`, ctx, url, body, contentType)
	return createFix("Replace http.Post with http.NewRequestWithContext", ce, newText)
}

func fixHTTPPostForm(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	url := arg(pass, ce, 0)
	data := arg(pass, ce, 1)
	newText := fmt.Sprintf(`func() (*http.Response, error) {
	req, err := http.NewRequestWithContext(%s, http.MethodPost, %s, strings.NewReader(%s.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return http.DefaultClient.Do(req)
}()`, ctx, url, data)
	return createFix("Replace http.PostForm with http.NewRequestWithContext", ce, newText)
}

func fixExecCommand(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) < 1 {
		return nil
	}
	args := make([]string, len(ce.Args))
	for i := range ce.Args {
		args[i] = arg(pass, ce, i)
	}
	newText := fmt.Sprintf("exec.CommandContext(%s, %s)", ctx, strings.Join(args, ", "))
	return createFix("Replace exec.Command with exec.CommandContext", ce, newText)
}

func fixNetDial(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	newText := fmt.Sprintf("(&net.Dialer{}).DialContext(%s, %s, %s)", ctx, network, address)
	return createFix("Replace net.Dial with (*net.Dialer).DialContext", ce, newText)
}

func fixNetDialTimeout(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	timeout := arg(pass, ce, 2)
	newText := fmt.Sprintf("(&net.Dialer{Timeout: %s}).DialContext(%s, %s, %s)", timeout, ctx, network, address)
	return createFix("Replace net.DialTimeout with (*net.Dialer).DialContext", ce, newText)
}

func fixNetListen(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	newText := fmt.Sprintf("(&net.ListenConfig{}).Listen(%s, %s, %s)", ctx, network, address)
	return createFix("Replace net.Listen with (*net.ListenConfig).Listen", ce, newText)
}

func fixNetListenPacket(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	newText := fmt.Sprintf("(&net.ListenConfig{}).ListenPacket(%s, %s, %s)", ctx, network, address)
	return createFix("Replace net.ListenPacket with (*net.ListenConfig).ListenPacket", ce, newText)
}

func fixNetLookup1(pass *analysis.Pass, ce *ast.CallExpr, ctx string, method string) *analysis.SuggestedFix {
	if len(ce.Args) != 1 {
		return nil
	}
	lookupArg := arg(pass, ce, 0)
	newText := fmt.Sprintf("(&net.Resolver{}).%s(%s, %s)", method, ctx, lookupArg)
	return createFix(fmt.Sprintf("Replace net.%s with (*net.Resolver).%s", method, method), ce, newText)
}

func fixNetLookupIP(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 1 {
		return nil
	}
	host := arg(pass, ce, 0)
	newText := fmt.Sprintf("(&net.Resolver{}).LookupIPAddr(%s, %s)", ctx, host)
	return createFix("Replace net.LookupIP with (*net.Resolver).LookupIPAddr", ce, newText)
}

func fixNetLookupPort(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	network := arg(pass, ce, 0)
	service := arg(pass, ce, 1)
	newText := fmt.Sprintf("(&net.Resolver{}).LookupPort(%s, %s, %s)", ctx, network, service)
	return createFix("Replace net.LookupPort with (*net.Resolver).LookupPort", ce, newText)
}

func fixNetLookupSRV(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	service := arg(pass, ce, 0)
	proto := arg(pass, ce, 1)
	name := arg(pass, ce, 2)
	newText := fmt.Sprintf("(&net.Resolver{}).LookupSRV(%s, %s, %s, %s)", ctx, service, proto, name)
	return createFix("Replace net.LookupSRV with (*net.Resolver).LookupSRV", ce, newText)
}

func fixTLSDial(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	network := arg(pass, ce, 0)
	address := arg(pass, ce, 1)
	config := arg(pass, ce, 2)
	newText := fmt.Sprintf("(&tls.Dialer{Config: %s}).DialContext(%s, %s, %s)", config, ctx, network, address)
	return createFix("Replace tls.Dial with (*tls.Dialer).DialContext", ce, newText)
}

func fixTLSDialWithDialer(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 4 {
		return nil
	}
	dialer := arg(pass, ce, 0)
	network := arg(pass, ce, 1)
	address := arg(pass, ce, 2)
	config := arg(pass, ce, 3)
	newText := fmt.Sprintf("(&tls.Dialer{NetDialer: %s, Config: %s}).DialContext(%s, %s, %s)", dialer, config, ctx, network, address)
	return createFix("Replace tls.DialWithDialer with (*tls.Dialer).DialContext", ce, newText)
}
