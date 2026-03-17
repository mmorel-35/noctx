package fixes

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

func fixExecCommand(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) < 1 {
		return nil
	}
	q := extractQualifier(ce)
	newText := q + "CommandContext(" + ctx
	for i := range ce.Args {
		newText += ", " + arg(pass, ce, i)
	}
	newText += ")"
	return createFix(fmt.Sprintf("Replace %sCommand with %sCommandContext", q, q), ce, newText)
}
