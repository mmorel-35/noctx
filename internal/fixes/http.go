package fixes

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

func fixHTTPNewRequest(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	q := extractQualifier(ce)
	method := arg(pass, ce, 0)
	url := arg(pass, ce, 1)
	body := arg(pass, ce, 2)
	newText := fmt.Sprintf("%sNewRequestWithContext(%s, %s, %s, %s)", q, ctx, method, url, body)
	return createFix(fmt.Sprintf("Replace %sNewRequest with %sNewRequestWithContext", q, q), ce, newText)
}

func fixHTTPTestNewRequest(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	q := extractQualifier(ce)
	method := arg(pass, ce, 0)
	target := arg(pass, ce, 1)
	body := arg(pass, ce, 2)
	newText := fmt.Sprintf("%sNewRequestWithContext(%s, %s, %s, %s)", q, ctx, method, target, body)
	return createFix(fmt.Sprintf("Replace %sNewRequest with %sNewRequestWithContext", q, q), ce, newText)
}

func fixHTTPGet(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 1 {
		return nil
	}
	stmt := findContainingStmt(pass.Files, ce.Pos())
	if stmt == nil || !isDirectCallInStmt(stmt, ce) {
		return nil
	}
	q := extractQualifier(ce)
	url := arg(pass, ce, 0)
	lhs := stmtLHSText(pass, stmt)
	op := stmtAssignOp(stmt)
	errReturn := buildErrReturn(pass, ce)
	newText := fmt.Sprintf(
		"req, err := %sNewRequestWithContext(%s, %sMethodGet, %s, %sNoBody)\n\tif err != nil {\n\t\t%s\n\t}\n\t%s %s %sDefaultClient.Do(req)",
		q, ctx, q, url, q, errReturn, lhs, op, q)
	return &analysis.SuggestedFix{
		Message: fmt.Sprintf("Replace %sGet with %sNewRequestWithContext", q, q),
		TextEdits: []analysis.TextEdit{
			{Pos: stmt.Pos(), End: stmt.End(), NewText: []byte(newText)},
		},
	}
}

func fixHTTPHead(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 1 {
		return nil
	}
	stmt := findContainingStmt(pass.Files, ce.Pos())
	if stmt == nil || !isDirectCallInStmt(stmt, ce) {
		return nil
	}
	q := extractQualifier(ce)
	url := arg(pass, ce, 0)
	lhs := stmtLHSText(pass, stmt)
	op := stmtAssignOp(stmt)
	errReturn := buildErrReturn(pass, ce)
	newText := fmt.Sprintf(
		"req, err := %sNewRequestWithContext(%s, %sMethodHead, %s, %sNoBody)\n\tif err != nil {\n\t\t%s\n\t}\n\t%s %s %sDefaultClient.Do(req)",
		q, ctx, q, url, q, errReturn, lhs, op, q)
	return &analysis.SuggestedFix{
		Message: fmt.Sprintf("Replace %sHead with %sNewRequestWithContext", q, q),
		TextEdits: []analysis.TextEdit{
			{Pos: stmt.Pos(), End: stmt.End(), NewText: []byte(newText)},
		},
	}
}

func fixHTTPPost(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	stmt := findContainingStmt(pass.Files, ce.Pos())
	if stmt == nil || !isDirectCallInStmt(stmt, ce) {
		return nil
	}
	q := extractQualifier(ce)
	url := arg(pass, ce, 0)
	contentType := arg(pass, ce, 1)
	body := arg(pass, ce, 2)
	lhs := stmtLHSText(pass, stmt)
	op := stmtAssignOp(stmt)
	errReturn := buildErrReturn(pass, ce)
	newText := fmt.Sprintf(
		"req, err := %sNewRequestWithContext(%s, %sMethodPost, %s, %s)\n\tif err != nil {\n\t\t%s\n\t}\n\treq.Header.Set(\"Content-Type\", %s)\n\t%s %s %sDefaultClient.Do(req)",
		q, ctx, q, url, body, errReturn, contentType, lhs, op, q)
	return &analysis.SuggestedFix{
		Message: fmt.Sprintf("Replace %sPost with %sNewRequestWithContext", q, q),
		TextEdits: []analysis.TextEdit{
			{Pos: stmt.Pos(), End: stmt.End(), NewText: []byte(newText)},
		},
	}
}

func fixHTTPPostForm(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	stmt := findContainingStmt(pass.Files, ce.Pos())
	if stmt == nil || !isDirectCallInStmt(stmt, ce) {
		return nil
	}
	q := extractQualifier(ce)
	url := arg(pass, ce, 0)
	data := arg(pass, ce, 1)
	lhs := stmtLHSText(pass, stmt)
	op := stmtAssignOp(stmt)
	errReturn := buildErrReturn(pass, ce)
	newText := fmt.Sprintf(
		"req, err := %sNewRequestWithContext(%s, %sMethodPost, %s, strings.NewReader(%s.Encode()))\n\tif err != nil {\n\t\t%s\n\t}\n\treq.Header.Set(\"Content-Type\", \"application/x-www-form-urlencoded\")\n\t%s %s %sDefaultClient.Do(req)",
		q, ctx, q, url, data, errReturn, lhs, op, q)
	fix := &analysis.SuggestedFix{
		Message: fmt.Sprintf("Replace %sPostForm with %sNewRequestWithContext", q, q),
		TextEdits: []analysis.TextEdit{
			{Pos: stmt.Pos(), End: stmt.End(), NewText: []byte(newText)},
		},
	}
	// The fix uses strings.NewReader; ensure "strings" is imported.
	if edit := addImportEdit(pass, ce, "strings"); edit != nil {
		fix.TextEdits = append(fix.TextEdits, *edit)
	}
	return fix
}
