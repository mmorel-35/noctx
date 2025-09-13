package noctx

import (
	"fmt"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sonatard/noctx/internal/checkers"
	"github.com/sonatard/noctx/internal/registry"
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
	autofixCheckers := checkers.GetAllCheckers()

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
	// Get autofix functions from registry
	autofixSupportedFuncs := registry.GetAutofixFunctions()

	// Get all rules (including legacy ones)
	allRules := registry.GetAllRules()
	
	// Extract function names for typeFuncs
	allFuncNames := make([]string, 0, len(allRules))
	for name := range allRules {
		allFuncNames = append(allFuncNames, name)
	}

	ngFuncs := typeFuncs(pass, allFuncNames)
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
						pass.Reportf(instr.Pos(), "%s", registry.FormatDiagnostic(funcName))
						break
					}
				}
			}
		}
	}

	return nil
}


