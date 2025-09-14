package checkers

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
	"github.com/sonatard/noctx/internal/helpers"
	"github.com/sonatard/noctx/internal/registry"
)

// ExecChecker detects exec functions that should use context
type ExecChecker struct {
	ctx *helpers.CheckerContext
}

// NewExecChecker creates a new exec function checker
func NewExecChecker() *ExecChecker {
	return &ExecChecker{
		ctx: helpers.NewCheckerContext(),
	}
}

// Name returns the name of this checker
func (c *ExecChecker) Name() string {
	return "exec"
}

// Check performs the analysis for exec functions
func (c *ExecChecker) Check(pass *analysis.Pass) error {
	// Get exec rules from registry
	rulesByChecker := registry.GetRulesByChecker()
	
	execRules, exists := rulesByChecker["exec"]
	if !exists {
		return nil
	}

	// Convert rules to function configs
	execFunctions := make([]helpers.FunctionConfig, 0, len(execRules))
	for _, rule := range execRules {
		if rule.HasAutofix {
			execFunctions = append(execFunctions, helpers.FunctionConfig{
				PackagePath: rule.PackagePath,
				FuncName:    rule.FuncName,
				Handler:     c.getExecHandler(rule.FullName),
			})
		}
	}

	// Check functions with autofix support
	if len(execFunctions) > 0 {
		if err := helpers.CheckFunctionsWithAutofix(pass, execFunctions, c.ctx); err != nil {
			return err
		}
	}

	return nil
}

// getExecHandler returns the appropriate fix handler for an exec function
func (c *ExecChecker) getExecHandler(funcName string) func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	switch funcName {
	case "os/exec.Command":
		return c.generateExecCommandFix
	default:
		return nil
	}
}

func (c *ExecChecker) generateExecCommandFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) < 1 {
		return nil
	}

	// Format all arguments
	var args []string
	for _, arg := range callExpr.Args {
		args = append(args, c.ctx.ArgFormatter.FormatArgument(pass, arg))
	}

	var newCall string
	if len(args) == 1 {
		newCall = fmt.Sprintf("exec.CommandContext(%s, %s)", contextExpr, args[0])
	} else {
		argsStr := ""
		for i, arg := range args {
			if i > 0 {
				argsStr += ", " + arg
			}
		}
		newCall = fmt.Sprintf("exec.CommandContext(%s, %s%s)", contextExpr, args[0], argsStr)
	}

	return fixes.CreateSuggestedFix("Replace with exec.CommandContext", callExpr, newCall)
}