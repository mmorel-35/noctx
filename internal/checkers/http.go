package checkers

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
	"github.com/sonatard/noctx/internal/helpers"
	"github.com/sonatard/noctx/internal/registry"
)

// HTTPChecker detects HTTP functions that should use context
type HTTPChecker struct {
	ctx *helpers.CheckerContext
}

// NewHTTPChecker creates a new HTTP function checker
func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		ctx: helpers.NewCheckerContext(),
	}
}

// Name returns the name of this checker
func (c *HTTPChecker) Name() string {
	return "http"
}

// Check performs the analysis for HTTP functions
func (c *HTTPChecker) Check(pass *analysis.Pass) error {
	// Get HTTP rules from registry
	rulesByChecker := registry.GetRulesByChecker()
	
	httpRules, exists := rulesByChecker["http"]
	if !exists {
		return nil
	}

	// Convert rules to function configs
	httpFunctions := make([]helpers.FunctionConfig, 0, len(httpRules))
	for _, rule := range httpRules {
		if rule.HasAutofix {
			httpFunctions = append(httpFunctions, helpers.FunctionConfig{
				PackagePath: rule.PackagePath,
				FuncName:    rule.FuncName,
				Handler:     c.getHTTPHandler(rule.FullName),
			})
		}
	}

	// Check functions with autofix support
	if len(httpFunctions) > 0 {
		if err := helpers.CheckFunctionsWithAutofix(pass, httpFunctions, c.ctx); err != nil {
			return err
		}
	}

	// Check method functions that are detection-only
	var methodFunctions []string
	for _, rule := range httpRules {
		if !rule.HasAutofix {
			methodFunctions = append(methodFunctions, rule.FullName)
		}
	}
	if len(methodFunctions) > 0 {
		if err := helpers.CheckMethodFunctionsWithoutAutofix(pass, methodFunctions); err != nil {
			return err
		}
	}

	return nil
}

// getHTTPHandler returns the appropriate fix handler for an HTTP function
func (c *HTTPChecker) getHTTPHandler(funcName string) func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	switch funcName {
	case "net/http.Get":
		return c.generateHTTPGetFix
	case "net/http.Head":
		return c.generateHTTPHeadFix
	case "net/http.Post":
		return c.generateHTTPPostFix
	case "net/http.PostForm":
		return c.generateHTTPPostFormFix
	case "net/http.NewRequest":
		return c.generateHTTPNewRequestFix
	default:
		return nil
	}
}

func (c *HTTPChecker) generateHTTPGetFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}

	urlArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodGet, %s, http.NoBody)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, urlArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}

func (c *HTTPChecker) generateHTTPHeadFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}

	urlArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodHead, %s, http.NoBody)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, urlArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}

func (c *HTTPChecker) generateHTTPPostFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	urlArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	contentTypeArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodPost, %s, %s)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", %s)
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, urlArg, bodyArg, contentTypeArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}

func (c *HTTPChecker) generateHTTPPostFormFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	urlArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	dataArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodPost, %s, strings.NewReader(%s.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, urlArg, dataArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}

func (c *HTTPChecker) generateHTTPNewRequestFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	methodArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	urlArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	
	newCall := fmt.Sprintf("http.NewRequestWithContext(%s, %s, %s, %s)", contextExpr, methodArg, urlArg, bodyArg)
	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext", callExpr, newCall)
}