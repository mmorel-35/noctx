package noctx

import (
	"fmt"

	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// ── net/http and net/http/httptest fixes ──────────────────────────────────────

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
	q := extractQualifier(ce)
	url := arg(pass, ce, 0)
	newText := fmt.Sprintf(`func() (*%sResponse, error) {
	req, err := %sNewRequestWithContext(%s, %sMethodGet, %s, %sNoBody)
	if err != nil {
		return nil, err
	}
	return %sDefaultClient.Do(req)
}()`, q, q, ctx, q, url, q, q)
	return createFix(fmt.Sprintf("Replace %sGet with %sNewRequestWithContext", q, q), ce, newText)
}

func fixHTTPHead(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 1 {
		return nil
	}
	q := extractQualifier(ce)
	url := arg(pass, ce, 0)
	newText := fmt.Sprintf(`func() (*%sResponse, error) {
	req, err := %sNewRequestWithContext(%s, %sMethodHead, %s, %sNoBody)
	if err != nil {
		return nil, err
	}
	return %sDefaultClient.Do(req)
}()`, q, q, ctx, q, url, q, q)
	return createFix(fmt.Sprintf("Replace %sHead with %sNewRequestWithContext", q, q), ce, newText)
}

func fixHTTPPost(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 3 {
		return nil
	}
	q := extractQualifier(ce)
	url := arg(pass, ce, 0)
	contentType := arg(pass, ce, 1)
	body := arg(pass, ce, 2)
	newText := fmt.Sprintf(`func() (*%sResponse, error) {
	req, err := %sNewRequestWithContext(%s, %sMethodPost, %s, %s)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", %s)
	return %sDefaultClient.Do(req)
}()`, q, q, ctx, q, url, body, contentType, q)
	return createFix(fmt.Sprintf("Replace %sPost with %sNewRequestWithContext", q, q), ce, newText)
}

func fixHTTPPostForm(pass *analysis.Pass, ce *ast.CallExpr, ctx string) *analysis.SuggestedFix {
	if len(ce.Args) != 2 {
		return nil
	}
	q := extractQualifier(ce)
	url := arg(pass, ce, 0)
	data := arg(pass, ce, 1)
	newText := fmt.Sprintf(`func() (*%sResponse, error) {
	req, err := %sNewRequestWithContext(%s, %sMethodPost, %s, strings.NewReader(%s.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return %sDefaultClient.Do(req)
}()`, q, q, ctx, q, url, data, q)
	return createFix(fmt.Sprintf("Replace %sPostForm with %sNewRequestWithContext", q, q), ce, newText)
}
