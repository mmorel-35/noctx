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
	// Use the unified checker for all functions with autofix support
	unifiedChecker := checkers.NewUnifiedChecker()
	if err := unifiedChecker.Check(pass); err != nil {
		return nil, err
	}

	// Fallback: use original logic for functions that may not be fully supported yet
	// This ensures backward compatibility while we transition to the unified approach
	if err := runFallbackChecks(pass, unifiedChecker); err != nil {
		return nil, err
	}

	return nil, nil
}

// runFallbackChecks provides backward compatibility for functions not yet fully supported by the unified checker
func runFallbackChecks(pass *analysis.Pass, unifiedChecker *checkers.UnifiedChecker) error {
	ngFuncs := typeFuncs(pass, slices.Collect(maps.Keys(diagnostics.Messages)))
	if len(ngFuncs) == 0 {
		return nil
	}

	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		return fmt.Errorf("failed to get SSA")
	}

	// Check which functions are supported by the unified checker
	supportedFuncs := unifiedChecker.GetSupportedFunctionNames()

	for _, sf := range ssa.SrcFuncs {
		for _, b := range sf.Blocks {
			for _, instr := range b.Instrs {
				for _, ngFunc := range ngFuncs {
					if analysisutil.Called(instr, nil, ngFunc) {
						funcName := ngFunc.FullName()
						
						// Skip functions that are supported by unified checker
						if _, supported := supportedFuncs[funcName]; supported {
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


