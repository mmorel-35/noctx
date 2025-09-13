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

// Check performs the analysis for exec functions
func (c *ExecChecker) Check(pass *analysis.Pass) error {
	if c.contextDetector == nil {
		c.contextDetector = &fixes.ContextDetector{}
	}
	if c.argFormatter == nil {
		c.argFormatter = &fixes.ArgumentFormatter{}
	}
	if c.assignDetector == nil {
		c.assignDetector = &fixes.VariableAssignmentDetector{}
	}

	functions := []FunctionConfig{
		{"os/exec", "Command", c.generateExecCommandFix},
	}

	httpChecker := &HTTPChecker{c.contextDetector, c.argFormatter, c.assignDetector}
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

// TLSChecker handles crypto/tls package functions that need context
type TLSChecker struct {
	contextDetector *fixes.ContextDetector
	argFormatter    *fixes.ArgumentFormatter
	assignDetector  *fixes.VariableAssignmentDetector
}

// Check performs the analysis for TLS functions
func (c *TLSChecker) Check(pass *analysis.Pass) error {
	if c.contextDetector == nil {
		c.contextDetector = &fixes.ContextDetector{}
	}
	if c.argFormatter == nil {
		c.argFormatter = &fixes.ArgumentFormatter{}
	}
	if c.assignDetector == nil {
		c.assignDetector = &fixes.VariableAssignmentDetector{}
	}

	functions := []FunctionConfig{
		{"crypto/tls", "Dial", c.generateTLSDialFix},
		{"crypto/tls", "DialWithDialer", c.generateTLSDialWithDialerFix},
	}

	httpChecker := &HTTPChecker{c.contextDetector, c.argFormatter, c.assignDetector}
	return httpChecker.checkFunctions(pass, functions)
}

func (c *TLSChecker) generateTLSDialFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	configArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "d", "")

	newCall := fmt.Sprintf(`func() (*tls.Conn, error) {
		d %s &tls.Dialer{Config: %s}
		return d.DialContext(%s, %s, %s)
	}()`, assignOp, configArg, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*tls.Dialer).DialContext", callExpr, newCall)
}

func (c *TLSChecker) generateTLSDialWithDialerFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 4 {
		return nil
	}

	dialerArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	configArg := c.argFormatter.FormatArgument(pass, callExpr.Args[3])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "d", "")

	newCall := fmt.Sprintf(`func() (*tls.Conn, error) {
		d %s &tls.Dialer{NetDialer: %s, Config: %s}
		return d.DialContext(%s, %s, %s)
	}()`, assignOp, dialerArg, configArg, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*tls.Dialer).DialContext", callExpr, newCall)
}