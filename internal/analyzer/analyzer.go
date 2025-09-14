package analyzer

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sonatard/noctx/internal/checkers"
)

// New creates a new noctx analyzer instance
func New() *analysis.Analyzer {
	return &analysis.Analyzer{
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
}

// Run executes the analyzer
func Run(pass *analysis.Pass) (interface{}, error) {
	// Use consolidated checkers for all functions
	allCheckers := checkers.GetAllCheckers()

	for _, checker := range allCheckers {
		if err := checker.Check(pass); err != nil {
			return nil, err
		}
	}

	return nil, nil
}