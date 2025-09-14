package checkers

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
	"github.com/sonatard/noctx/internal/helpers"
	"github.com/sonatard/noctx/internal/registry"
)

// UnifiedChecker handles all functions with autofix support using a simplified approach
type UnifiedChecker struct {
	ctx *helpers.CheckerContext
}

// NewUnifiedChecker creates a new unified checker
func NewUnifiedChecker() *UnifiedChecker {
	return &UnifiedChecker{
		ctx: helpers.NewCheckerContext(),
	}
}

// Name returns the name of this checker
func (c *UnifiedChecker) Name() CheckerName {
	return "unified"
}

// Check performs the analysis for all supported functions
func (c *UnifiedChecker) Check(pass *analysis.Pass) error {
	// Get all functions that have autofix support
	rulesByChecker := registry.GetRulesByChecker()
	
	// Check HTTP functions
	if httpRules, exists := rulesByChecker["http"]; exists {
		httpFunctions := make([]helpers.FunctionConfig, 0, len(httpRules))
		for _, rule := range httpRules {
			httpFunctions = append(httpFunctions, helpers.FunctionConfig{
				PackagePath: rule.PackagePath,
				FuncName:    rule.FuncName,
				Handler:     c.getHTTPHandler(rule.FullName),
			})
		}
		if err := helpers.CheckFunctionsWithAutofix(pass, httpFunctions, c.ctx); err != nil {
			return err
		}
	}

	// Check Net functions
	if netRules, exists := rulesByChecker["net"]; exists {
		netFunctions := make([]helpers.FunctionConfig, 0, len(netRules))
		for _, rule := range netRules {
			netFunctions = append(netFunctions, helpers.FunctionConfig{
				PackagePath: rule.PackagePath,
				FuncName:    rule.FuncName,
				Handler:     c.getNetHandler(rule.FullName),
			})
		}
		if err := helpers.CheckFunctionsWithAutofix(pass, netFunctions, c.ctx); err != nil {
			return err
		}
	}

	// Check Exec functions
	if execRules, exists := rulesByChecker["exec"]; exists {
		execFunctions := make([]helpers.FunctionConfig, 0, len(execRules))
		for _, rule := range execRules {
			execFunctions = append(execFunctions, helpers.FunctionConfig{
				PackagePath: rule.PackagePath,
				FuncName:    rule.FuncName,
				Handler:     c.getExecHandler(rule.FullName),
			})
		}
		if err := helpers.CheckFunctionsWithAutofix(pass, execFunctions, c.ctx); err != nil {
			return err
		}
	}

	// Check TLS functions
	if tlsRules, exists := rulesByChecker["tls"]; exists {
		tlsFunctions := make([]helpers.FunctionConfig, 0, len(tlsRules))
		for _, rule := range tlsRules {
			tlsFunctions = append(tlsFunctions, helpers.FunctionConfig{
				PackagePath: rule.PackagePath,
				FuncName:    rule.FuncName,
				Handler:     c.getTLSHandler(rule.FullName),
			})
		}
		if err := helpers.CheckFunctionsWithAutofix(pass, tlsFunctions, c.ctx); err != nil {
			return err
		}
	}

	// Check method functions without autofix
	methodFunctions := []string{}
	for name, rule := range registry.Rules {
		if !rule.HasAutofix {
			methodFunctions = append(methodFunctions, name)
		}
	}
	if len(methodFunctions) > 0 {
		if err := helpers.CheckMethodFunctionsWithoutAutofix(pass, methodFunctions); err != nil {
			return err
		}
	}

	return nil
}

// HTTP Handler Functions
func (c *UnifiedChecker) getHTTPHandler(funcName string) func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
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

func (c *UnifiedChecker) generateHTTPGetFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
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

func (c *UnifiedChecker) generateHTTPHeadFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
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

func (c *UnifiedChecker) generateHTTPPostFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	urlArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	contentTypeArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.ctx.ArgFormatter.FormatBodyArgument(pass, callExpr.Args[2])
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

func (c *UnifiedChecker) generateHTTPPostFormFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
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

func (c *UnifiedChecker) generateHTTPNewRequestFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	methodArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	urlArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.ctx.ArgFormatter.FormatBodyArgument(pass, callExpr.Args[2])

	newCall := fmt.Sprintf("http.NewRequestWithContext(%s, %s, %s, %s)", contextExpr, methodArg, urlArg, bodyArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext", callExpr, newCall)
}

// Net Handler Functions
func (c *UnifiedChecker) getNetHandler(funcName string) func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	switch funcName {
	case "net.Dial":
		return c.generateNetDialFix
	case "net.DialTimeout":
		return c.generateNetDialTimeoutFix
	case "net.Listen":
		return c.generateNetListenFix
	case "net.ListenPacket":
		return c.generateNetListenPacketFix
	default:
		// All Lookup functions use the same pattern
		if len(funcName) > 10 && funcName[:10] == "net.Lookup" {
			return c.generateNetLookupFix
		}
		return nil
	}
}

func (c *UnifiedChecker) generateNetDialFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "dialer", "")

	newCall := fmt.Sprintf(`func() (net.Conn, error) {
		dialer %s &net.Dialer{}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext", callExpr, newCall)
}

func (c *UnifiedChecker) generateNetDialTimeoutFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	timeoutArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "dialer", "")

	newCall := fmt.Sprintf(`func() (net.Conn, error) {
		dialer %s &net.Dialer{Timeout: %s}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, timeoutArg, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext", callExpr, newCall)
}

func (c *UnifiedChecker) generateNetListenFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "lc", "")

	newCall := fmt.Sprintf(`func() (net.Listener, error) {
		lc %s &net.ListenConfig{}
		return lc.Listen(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.ListenConfig).Listen", callExpr, newCall)
}

func (c *UnifiedChecker) generateNetListenPacketFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "lc", "")

	newCall := fmt.Sprintf(`func() (net.PacketConn, error) {
		lc %s &net.ListenConfig{}
		return lc.ListenPacket(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.ListenConfig).ListenPacket", callExpr, newCall)
}

func (c *UnifiedChecker) generateNetLookupFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) < 1 {
		return nil
	}

	// Get the function name from the call expression
	funcName := ""
	if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		funcName = sel.Sel.Name
	}

	// Format all arguments
	args := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		args[i] = c.ctx.ArgFormatter.FormatArgument(pass, arg)
	}

	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "resolver", "")

	// Build the new call with resolver
	argList := contextExpr
	for _, arg := range args {
		argList += ", " + arg
	}

	newCall := fmt.Sprintf(`func() (interface{}, error) {
		resolver %s &net.Resolver{}
		return resolver.%s(%s)
	}()`, assignOp, funcName, argList)

	return fixes.CreateSuggestedFix("Replace with (*net.Resolver)."+funcName, callExpr, newCall)
}

// Exec Handler Functions
func (c *UnifiedChecker) getExecHandler(funcName string) func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if funcName == "os/exec.Command" {
		return c.generateExecCommandFix
	}
	return nil
}

func (c *UnifiedChecker) generateExecCommandFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) < 1 {
		return nil
	}

	// Format all arguments
	args := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		args[i] = c.ctx.ArgFormatter.FormatArgument(pass, arg)
	}

	// Build argument list
	argList := contextExpr
	for _, arg := range args {
		argList += ", " + arg
	}

	newCall := fmt.Sprintf("exec.CommandContext(%s)", argList)

	return fixes.CreateSuggestedFix("Replace with exec.CommandContext", callExpr, newCall)
}

// TLS Handler Functions
func (c *UnifiedChecker) getTLSHandler(funcName string) func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	switch funcName {
	case "crypto/tls.Dial":
		return c.generateTLSDialFix
	case "crypto/tls.DialWithDialer":
		return c.generateTLSDialWithDialerFix
	default:
		return nil
	}
}

func (c *UnifiedChecker) generateTLSDialFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addrArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	configArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "dialer", "")

	newCall := fmt.Sprintf(`func() (*tls.Conn, error) {
		dialer %s &tls.Dialer{Config: %s}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, configArg, contextExpr, networkArg, addrArg)

	return fixes.CreateSuggestedFix("Replace with (*tls.Dialer).DialContext", callExpr, newCall)
}

func (c *UnifiedChecker) generateTLSDialWithDialerFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 4 {
		return nil
	}

	netDialerArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	addrArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	configArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[3])
	assignOp := c.ctx.AssignDetector.DetectAssignmentOperator(pass, callExpr, "dialer", "")

	newCall := fmt.Sprintf(`func() (*tls.Conn, error) {
		dialer %s &tls.Dialer{NetDialer: %s, Config: %s}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, netDialerArg, configArg, contextExpr, networkArg, addrArg)

	return fixes.CreateSuggestedFix("Replace with (*tls.Dialer).DialContext", callExpr, newCall)
}