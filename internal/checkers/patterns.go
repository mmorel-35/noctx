package checkers

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
)

// HTTP Pattern Generators

func (c *UnifiedChecker) generateHTTPGetPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr, method string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, %s, %s, http.NoBody)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}()`, assignOp, contextExpr, method, urlArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and Do", callExpr, newCall)
}

func (c *UnifiedChecker) generateHTTPPostPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	contentTypeArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
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

func (c *UnifiedChecker) generateHTTPPostFormPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	dataArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
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

func (c *UnifiedChecker) generateHTTPNewRequestPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	methodArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	
	// Replace nil body with http.NoBody for better performance
	if bodyArg == "nil" {
		bodyArg = "http.NoBody"
	}
	
	newCall := fmt.Sprintf("http.NewRequestWithContext(%s, %s, %s, %s)", contextExpr, methodArg, urlArg, bodyArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext", callExpr, newCall)
}

// Net Pattern Generators

func (c *UnifiedChecker) generateNetDialPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "dialer", "")

	newCall := fmt.Sprintf(`func() (net.Conn, error) {
		dialer %s &net.Dialer{}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext", callExpr, newCall)
}

func (c *UnifiedChecker) generateNetDialTimeoutPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	timeoutArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "dialer", "")

	newCall := fmt.Sprintf(`func() (net.Conn, error) {
		dialer %s &net.Dialer{Timeout: %s}
		return dialer.DialContext(%s, %s, %s)
	}()`, assignOp, timeoutArg, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext", callExpr, newCall)
}

func (c *UnifiedChecker) generateNetListenPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "lc", "")

	newCall := fmt.Sprintf(`func() (net.Listener, error) {
		lc %s &net.ListenConfig{}
		return lc.Listen(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.ListenConfig).Listen", callExpr, newCall)
}

func (c *UnifiedChecker) generateNetListenPacketPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "lc", "")

	newCall := fmt.Sprintf(`func() (net.PacketConn, error) {
		lc %s &net.ListenConfig{}
		return lc.ListenPacket(%s, %s, %s)
	}()`, assignOp, contextExpr, networkArg, addressArg)

	return fixes.CreateSuggestedFix("Replace with (*net.ListenConfig).ListenPacket", callExpr, newCall)
}

func (c *UnifiedChecker) generateNetLookupPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr, function string) *analysis.SuggestedFix {
	if len(callExpr.Args) == 0 {
		return nil
	}

	args := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		args[i] = c.argFormatter.FormatArgument(pass, arg)
	}
	
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "r", "")
	
	// Map function names to their context equivalents
	contextFunction := function
	if function == "LookupIP" {
		contextFunction = "LookupIPAddr"
	}

	allArgs := append([]string{contextExpr}, args...)
	argList := strings.Join(allArgs, ", ")

	newCall := fmt.Sprintf(`func() (interface{}, error) {
		r %s &net.Resolver{}
		return r.%s(%s)
	}()`, assignOp, contextFunction, argList)

	return fixes.CreateSuggestedFix(fmt.Sprintf("Replace with (*net.Resolver).%s", contextFunction), callExpr, newCall)
}

// Exec Pattern Generator

func (c *UnifiedChecker) generateExecCommandPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
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

// SQL Pattern Generators

func (c *UnifiedChecker) generateSQLDBPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr, function string) *analysis.SuggestedFix {
	args := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		args[i] = c.argFormatter.FormatArgument(pass, arg)
	}
	
	// Map function names to their context equivalents
	contextFunction := function + "Context"
	if function == "Begin" {
		contextFunction = "BeginTx"
		// BeginTx requires additional options parameter
		allArgs := append([]string{contextExpr, "nil"}, args...)
		argList := strings.Join(allArgs, ", ")
		newCall := fmt.Sprintf("db.%s(%s)", contextFunction, argList)
		return fixes.CreateSuggestedFix(fmt.Sprintf("Replace with db.%s", contextFunction), callExpr, newCall)
	}
	
	allArgs := append([]string{contextExpr}, args...)
	argList := strings.Join(allArgs, ", ")

	newCall := fmt.Sprintf("db.%s(%s)", contextFunction, argList)

	return fixes.CreateSuggestedFix(fmt.Sprintf("Replace with db.%s", contextFunction), callExpr, newCall)
}

func (c *UnifiedChecker) generateSQLTxPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr, function string) *analysis.SuggestedFix {
	args := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		args[i] = c.argFormatter.FormatArgument(pass, arg)
	}
	
	contextFunction := function + "Context"
	allArgs := append([]string{contextExpr}, args...)
	argList := strings.Join(allArgs, ", ")

	newCall := fmt.Sprintf("tx.%s(%s)", contextFunction, argList)

	return fixes.CreateSuggestedFix(fmt.Sprintf("Replace with tx.%s", contextFunction), callExpr, newCall)
}

func (c *UnifiedChecker) generateSQLStmtPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr, function string) *analysis.SuggestedFix {
	args := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		args[i] = c.argFormatter.FormatArgument(pass, arg)
	}
	
	contextFunction := function + "Context"
	allArgs := append([]string{contextExpr}, args...)
	argList := strings.Join(allArgs, ", ")

	newCall := fmt.Sprintf("stmt.%s(%s)", contextFunction, argList)

	return fixes.CreateSuggestedFix(fmt.Sprintf("Replace with stmt.%s", contextFunction), callExpr, newCall)
}

// TLS Pattern Generators

func (c *UnifiedChecker) generateTLSDialPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
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

func (c *UnifiedChecker) generateTLSDialWithDialerPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
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

func (c *UnifiedChecker) generateTLSHandshakePattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 0 {
		return nil
	}

	newCall := fmt.Sprintf("conn.HandshakeContext(%s)", contextExpr)

	return fixes.CreateSuggestedFix("Replace with conn.HandshakeContext", callExpr, newCall)
}

// HTTP Client Pattern Generators

func (c *UnifiedChecker) generateHTTPClientGetPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")

	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodGet, %s, http.NoBody)
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	}()`, assignOp, contextExpr, urlArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and client.Do", callExpr, newCall)
}

func (c *UnifiedChecker) generateHTTPClientHeadPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")

	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodHead, %s, http.NoBody)
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	}()`, assignOp, contextExpr, urlArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and client.Do", callExpr, newCall)
}

func (c *UnifiedChecker) generateHTTPClientPostPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	contentTypeArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	bodyArg := c.argFormatter.FormatArgument(pass, callExpr.Args[2])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")

	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodPost, %s, %s)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", %s)
		return client.Do(req)
	}()`, assignOp, contextExpr, urlArg, bodyArg, contentTypeArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and client.Do", callExpr, newCall)
}

func (c *UnifiedChecker) generateHTTPClientPostFormPattern(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	urlArg := c.argFormatter.FormatArgument(pass, callExpr.Args[0])
	dataArg := c.argFormatter.FormatArgument(pass, callExpr.Args[1])
	assignOp := c.assignDetector.DetectAssignmentOperator(pass, callExpr, "req", "err")

	newCall := fmt.Sprintf(`func() (*http.Response, error) {
		req, err %s http.NewRequestWithContext(%s, http.MethodPost, %s, strings.NewReader(%s.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return client.Do(req)
	}()`, assignOp, contextExpr, urlArg, dataArg)

	return fixes.CreateSuggestedFix("Replace with http.NewRequestWithContext and client.Do", callExpr, newCall)
}