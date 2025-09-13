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

// Functions that have autofix support
var autofixSupportedFuncs = map[string]bool{
	"net/http.NewRequest": true,
	"net/http.Get":        true,
	"net/http.Head":       true,
	"net/http.Post":       true,
	"net.Dial":            true,
}

func Run(pass *analysis.Pass) (interface{}, error) {
	// First, run the specialized checkers for functions with autofix support
	autofixCheckers := []checkers.Checker{
		&checkers.HTTPNewRequest{},
		&checkers.HTTPGet{},
		&checkers.HTTPHead{},
		&checkers.HTTPPost{},
		&checkers.NetDial{},
	}

	for _, checker := range autofixCheckers {
		if err := checker.Check(pass); err != nil {
			return nil, err
		}
	}

	// Then run the original logic for other functions without autofix
	ngFuncs := typeFuncs(pass, slices.Collect(maps.Keys(diagnostics.Messages)))
	if len(ngFuncs) == 0 {
		return nil, nil
	}

	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		panic(fmt.Sprintf("%T is not *buildssa.SSA", pass.ResultOf[buildssa.Analyzer]))
	}

	// Use original SSA-based detection for functions without autofix
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
						
						// Report violation without autofix
						pass.Reportf(instr.Pos(), "%s", diagnostics.FormatDiagnostic(funcName))
						break
					}
				}
			}
		}
	}

	return nil, nil
}


