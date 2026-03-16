package noctx

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// ── os/exec fixes ─────────────────────────────────────────────────────────────

func fixExecCommand(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) < 1 {
		return nil
	}
	q := extractQualifier(ce)
	args := make([]string, len(ce.Args))
	for i := range ce.Args {
		args[i] = arg(pass, ce, i)
	}
	newText := q + "CommandContext(" + ctx
	for _, a := range args {
		newText += ", " + a
	}
	newText += ")"
	return createFix(fmt.Sprintf("Replace %sCommand with %sCommandContext", q, q), ce, newText)
}
