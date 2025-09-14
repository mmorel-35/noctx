package helpers

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/sonatard/noctx/internal/fixes"
	"github.com/sonatard/noctx/internal/registry"
)

// FunctionConfig defines how to handle a specific function
type FunctionConfig struct {
	PackagePath string
	FuncName    string
	Handler     func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix
}

// CheckerContext provides shared context for checkers
type CheckerContext struct {
	ContextDetector *fixes.ContextDetector
	ArgFormatter    *fixes.ArgumentFormatter
	AssignDetector  *fixes.VariableAssignmentDetector
}

// NewCheckerContext creates a new shared checker context
func NewCheckerContext() *CheckerContext {
	return &CheckerContext{
		ContextDetector: &fixes.ContextDetector{},
		ArgFormatter:    &fixes.ArgumentFormatter{},
		AssignDetector:  &fixes.VariableAssignmentDetector{},
	}
}

// CheckFunctionsWithAutofix checks a list of functions and provides autofix when possible
func CheckFunctionsWithAutofix(pass *analysis.Pass, functions []FunctionConfig, ctx *CheckerContext) error {
	// Get required analyzers
	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		return fmt.Errorf("failed to get SSA")
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return fmt.Errorf("failed to get inspector")
	}

	// Collect call expressions for potential autofix
	callExprs := make(map[token.Pos]*ast.CallExpr)
	allCallExprs := []*ast.CallExpr{}
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		callExpr := n.(*ast.CallExpr)
		callExprs[callExpr.Pos()] = callExpr
		allCallExprs = append(allCallExprs, callExpr)
	})

	// Process each function
	for _, funcConfig := range functions {
		if err := checkSingleFunction(pass, funcConfig, ssa, callExprs, allCallExprs, ctx); err != nil {
			return err
		}
	}

	return nil
}

// checkSingleFunction checks a single function for violations and applies autofix
func checkSingleFunction(pass *analysis.Pass, funcConfig FunctionConfig, ssa *buildssa.SSA, callExprs map[token.Pos]*ast.CallExpr, allCallExprs []*ast.CallExpr, ctx *CheckerContext) error {
	targetFunc := analysisutil.ObjectOf(pass, funcConfig.PackagePath, funcConfig.FuncName)
	if targetFunc == nil {
		return nil // Function not used
	}

	// Use SSA-based detection to find violations
	for _, sf := range ssa.SrcFuncs {
		for _, b := range sf.Blocks {
			for _, instr := range b.Instrs {
				if analysisutil.Called(instr, nil, targetFunc.(*types.Func)) {
					funcName := fmt.Sprintf("%s.%s", funcConfig.PackagePath, funcConfig.FuncName)
					
					// Try to find matching call expression for autofix
					var targetCallExpr *ast.CallExpr
					if callExpr, exists := callExprs[instr.Pos()]; exists {
						targetCallExpr = callExpr
					} else {
						// Fallback: search for matching call expression
						targetCallExpr = findMatchingCallExpr(allCallExprs, funcConfig)
					}
					
					if targetCallExpr != nil && funcConfig.Handler != nil {
						contextExpr := ctx.ContextDetector.DetectContext(pass, targetCallExpr)
						suggestedFix := funcConfig.Handler(pass, targetCallExpr, contextExpr)
						if suggestedFix != nil {
							pass.Report(analysis.Diagnostic{
								Pos:            instr.Pos(),
								Message:        registry.FormatDiagnostic(funcName),
								SuggestedFixes: []analysis.SuggestedFix{*suggestedFix},
							})
							continue
						}
					}
					
					// Fallback to regular reporting
					pass.Reportf(instr.Pos(), "%s", registry.FormatDiagnostic(funcName))
				}
			}
		}
	}

	return nil
}

// findMatchingCallExpr finds a call expression that matches the function config
func findMatchingCallExpr(callExprs []*ast.CallExpr, funcConfig FunctionConfig) *ast.CallExpr {
	expectedPkg := funcConfig.PackagePath
	if strings.Contains(expectedPkg, "/") {
		parts := strings.Split(expectedPkg, "/")
		expectedPkg = parts[len(parts)-1]
	}

	for _, callExpr := range callExprs {
		if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == expectedPkg && sel.Sel.Name == funcConfig.FuncName {
					return callExpr
				}
			}
		}
	}
	return nil
}

// CheckMethodFunctionsWithoutAutofix checks method functions that don't have autofix support yet
func CheckMethodFunctionsWithoutAutofix(pass *analysis.Pass, functions []string) error {
	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		return fmt.Errorf("failed to get SSA")
	}

	// Get type functions for method checking
	ngFuncs := GetTypeFuncs(pass, functions)
	if len(ngFuncs) == 0 {
		return nil
	}

	for _, sf := range ssa.SrcFuncs {
		for _, b := range sf.Blocks {
			for _, instr := range b.Instrs {
				for _, ngFunc := range ngFuncs {
					if analysisutil.Called(instr, nil, ngFunc) {
						funcName := ngFunc.FullName()
						pass.Reportf(instr.Pos(), "%s", registry.FormatDiagnostic(funcName))
						break
					}
				}
			}
		}
	}

	return nil
}

// GetTypeFuncs returns type functions for the given function names
func GetTypeFuncs(pass *analysis.Pass, funcNames []string) []*types.Func {
	fs := make([]*types.Func, 0, len(funcNames))

	for _, fn := range funcNames {
		f, err := getTypeFunc(pass, fn)
		if err != nil {
			continue
		}

		fs = append(fs, f)
	}

	return fs
}

// getTypeFunc gets a type function by name
func getTypeFunc(pass *analysis.Pass, funcName string) (*types.Func, error) {
	nameParts := strings.Split(strings.TrimSpace(funcName), ".")

	switch len(nameParts) {
	case 2:
		// package function: pkgname.Func
		f, ok := analysisutil.ObjectOf(pass, nameParts[0], nameParts[1]).(*types.Func)
		if !ok || f == nil {
			return nil, fmt.Errorf("function not found")
		}

		return f, nil
	case 3:
		// method: (*pkgname.Type).Method
		pkgName := strings.TrimLeft(nameParts[0], "(")
		typeName := strings.TrimRight(nameParts[1], ")")

		if pkgName != "" && pkgName[0] == '*' {
			pkgName = pkgName[1:]
			typeName = "*" + typeName
		}

		typ := analysisutil.TypeOf(pass, pkgName, typeName)
		if typ == nil {
			return nil, fmt.Errorf("type not found")
		}

		m := analysisutil.MethodOf(typ, nameParts[2])
		if m == nil {
			return nil, fmt.Errorf("method not found")
		}

		return m, nil
	}

	return nil, fmt.Errorf("invalid function name format")
}