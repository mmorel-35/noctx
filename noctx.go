package noctx

import (
	"fmt"
	"maps"
	"slices"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sonatard/noctx/internal/checkers"
	"github.com/sonatard/noctx/internal/diagnostics"
)

// Analyzer defines the noctx analyzer
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

func Run(pass *analysis.Pass) (interface{}, error) {
	// Use consolidated checkers for functions with autofix support
	autofixCheckers := []checkers.Checker{
		&checkers.HTTPChecker{},
		&checkers.NetChecker{},
		&checkers.ExecChecker{},
		&checkers.TLSChecker{},
	}

	for _, checker := range autofixCheckers {
		if err := checker.Check(pass); err != nil {
			return nil, err
		}
	}

	// Fallback: use original logic for functions that may not be fully supported yet
	// This ensures backward compatibility while we add more function support
	if err := runFallbackChecks(pass); err != nil {
		return nil, err
	}

	return nil, nil
}

// runFallbackChecks provides backward compatibility for functions not yet fully supported by the consolidated checkers
func runFallbackChecks(pass *analysis.Pass) error {
	// List of functions that have autofix support (to skip in fallback)
	autofixSupportedFuncs := map[string]bool{
		"net/http.NewRequest": true,
		"net/http.Get":        true,
		"net/http.Head":       true,
		"net/http.Post":       true,
		"net/http.PostForm":   true,
		"net.Dial":            true,
		"net.DialTimeout":     true,
		"net.Listen":          true,
		"net.ListenPacket":    true,
		"net.LookupCNAME":     true,
		"net.LookupHost":      true,
		"net.LookupIP":        true,
		"net.LookupPort":      true,
		"net.LookupSRV":       true,
		"net.LookupMX":        true,
		"net.LookupNS":        true,
		"net.LookupTXT":       true,
		"net.LookupAddr":      true,
		"os/exec.Command":     true,
		"crypto/tls.Dial":     true,
		"crypto/tls.DialWithDialer": true,
	}

	ngFuncs := typeFuncs(pass, slices.Collect(maps.Keys(diagnostics.Messages)))
	if len(ngFuncs) == 0 {
		return nil
	}

	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		return fmt.Errorf("failed to get SSA")
	}

	for _, sf := range ssa.SrcFuncs {
		for _, b := range sf.Blocks {
			for _, instr := range b.Instrs {
				for _, ngFunc := range ngFuncs {
					if analysisutil.Called(instr, nil, ngFunc) {
						funcName := ngFunc.FullName()
						
						// Skip functions that have autofix support (already handled above)
						if autofixSupportedFuncs[funcName] {
							continue
						}
						
						// Report violation without autofix for unsupported functions
						pass.Reportf(instr.Pos(), "%s", diagnostics.FormatDiagnostic(funcName))
						break
					}
				}
			}
		}
	}

	return nil
}


