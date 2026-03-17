package noctx

import (
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"

	"github.com/gostaticanalysis/analysisutil"
	"github.com/sonatard/noctx/internal/fixes"
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
	"net/http.Get":                 "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
	"net/http.Head":                "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
	"net/http.Post":                "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
	"net/http.PostForm":            "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
	"(*net/http.Client).Get":       "must not be called. use (*net/http.Client).Do(*http.Request)",
	"(*net/http.Client).Head":      "must not be called. use (*net/http.Client).Do(*http.Request)",
	"(*net/http.Client).Post":      "must not be called. use (*net/http.Client).Do(*http.Request)",
	"(*net/http.Client).PostForm":  "must not be called. use (*net/http.Client).Do(*http.Request)",
	"net/http.NewRequest":          "must not be called. use net/http.NewRequestWithContext",
	"net/http/httptest.NewRequest": "must not be called. use net/http/httptest.NewRequestWithContext",

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

	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		panic(fmt.Sprintf("%T is not *inspector.Inspector", pass.ResultOf[inspect.Analyzer]))
	}

	// Collect call expressions indexed by Lparen to match SSA instruction positions.
	// SSA call instructions use the Lparen position of the ast.CallExpr.
	callExprs := make(map[token.Pos]*ast.CallExpr)
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		callExprs[ce.Lparen] = ce
	})

	for _, sf := range ssa.SrcFuncs {
		for _, b := range sf.Blocks {
			for _, instr := range b.Instrs {
				for _, ngFunc := range ngFuncs {
					if analysisutil.Called(instr, nil, ngFunc) {
						funcName := ngFunc.FullName()
						msg := fmt.Sprintf("%s %s", funcName, ngFuncMessages[funcName])

						diag := analysis.Diagnostic{
							Pos:     instr.Pos(),
							Message: msg,
						}

						if ce, exists := callExprs[instr.Pos()]; exists {
							if fix := fixes.Generate(pass, funcName, ce); fix != nil {
								diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
							}
						}

						pass.Report(diag)

						break
					}
				}
			}
		}
	}

	return nil, nil
}
