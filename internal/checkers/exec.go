package checkers

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
)

// ExecChecker handles os/exec package functions that need context
type ExecChecker struct {
	contextDetector *fixes.ContextDetector
	argFormatter    *fixes.ArgumentFormatter
	assignDetector  *fixes.VariableAssignmentDetector
}

// NewExecChecker creates a new Exec checker instance
func NewExecChecker() *ExecChecker {
	return &ExecChecker{
		contextDetector: &fixes.ContextDetector{},
		argFormatter:    &fixes.ArgumentFormatter{},
		assignDetector:  &fixes.VariableAssignmentDetector{},
	}
}

// Name returns the name of this checker
func (c *ExecChecker) Name() CheckerName {
	return ExecCheckerName
}

// Check performs the analysis for exec functions
func (c *ExecChecker) Check(pass *analysis.Pass) error {
	functions := []FunctionConfig{
		{"os/exec", "Command", c.generateExecCommandFix},
	}

	httpChecker := NewHTTPChecker()
	return httpChecker.checkFunctions(pass, functions)
}

func (c *ExecChecker) generateExecCommandFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) == 0 {
		return nil
	}

	args := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		args[i] = c.argFormatter.FormatArgument(pass, arg)
	}
	
	allArgs := append([]string{contextExpr}, args...)
	argList := strings.Join(allArgs, ", ")

	newCall := fmt.Sprintf("exec.CommandContext(%s)", argList)

	return fixes.CreateSuggestedFix("Replace with exec.CommandContext", callExpr, newCall)
}