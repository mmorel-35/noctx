package checkers

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/sonatard/noctx/internal/diagnostics"
	"github.com/sonatard/noctx/internal/fixes"
)

// FunctionInfo holds metadata about a function that needs context
type FunctionInfo struct {
	Package        string
	Function       string
	MethodReceiver string // For methods like (*net/http.Client).Get
	AutofixPattern string // The pattern to use for autofix
}

// UnifiedChecker handles all functions that need context-aware fixes
type UnifiedChecker struct {
	contextDetector *fixes.ContextDetector
	argFormatter    *fixes.ArgumentFormatter
	assignDetector  *fixes.VariableAssignmentDetector
	
	// Supported functions with their autofix patterns
	supportedFunctions map[string]FunctionInfo
}

// NewUnifiedChecker creates a new unified checker with all supported functions
func NewUnifiedChecker() *UnifiedChecker {
	return &UnifiedChecker{
		contextDetector:    &fixes.ContextDetector{},
		argFormatter:       &fixes.ArgumentFormatter{},
		assignDetector:     &fixes.VariableAssignmentDetector{},
		supportedFunctions: getSupportedFunctions(),
	}
}

// Check performs the analysis for all supported functions
func (c *UnifiedChecker) Check(pass *analysis.Pass) error {
	// Get required analyzers
	ssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		return fmt.Errorf("failed to get SSA")
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return fmt.Errorf("failed to get inspector")
	}

	// Collect all call expressions for potential autofix
	callExprs := make(map[token.Pos]*ast.CallExpr)
	allCallExprs := []*ast.CallExpr{}
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		callExpr := n.(*ast.CallExpr)
		callExprs[callExpr.Pos()] = callExpr
		allCallExprs = append(allCallExprs, callExpr)
	})

	// Process each supported function
	for funcName, funcInfo := range c.supportedFunctions {
		targetFunc := c.getTargetFunction(pass, funcInfo)
		if targetFunc == nil {
			continue // Function not used
		}

		// Use SSA-based detection to find violations
		for _, sf := range ssa.SrcFuncs {
			for _, b := range sf.Blocks {
				for _, instr := range b.Instrs {
					if analysisutil.Called(instr, nil, targetFunc) {
						// Try to find matching call expression for autofix
						var targetCallExpr *ast.CallExpr
						if callExpr, exists := callExprs[instr.Pos()]; exists {
							targetCallExpr = callExpr
						} else {
							// Fallback: search for matching call expression
							targetCallExpr = c.findMatchingCallExpr(allCallExprs, funcInfo)
						}
						
						if targetCallExpr != nil {
							suggestedFix := c.generateSuggestedFix(pass, targetCallExpr, funcInfo)
							if suggestedFix != nil {
								pass.Report(analysis.Diagnostic{
									Pos:            instr.Pos(),
									Message:        diagnostics.FormatDiagnostic(funcName),
									SuggestedFixes: []analysis.SuggestedFix{*suggestedFix},
								})
								continue
							}
						}
						
						// Fallback to regular reporting
						pass.Reportf(instr.Pos(), "%s", diagnostics.FormatDiagnostic(funcName))
					}
				}
			}
		}
	}

	return nil
}

// getTargetFunction resolves the target function based on function info
func (c *UnifiedChecker) getTargetFunction(pass *analysis.Pass, funcInfo FunctionInfo) *types.Func {
	if funcInfo.MethodReceiver != "" {
		// Handle method calls like (*net/http.Client).Get
		return c.resolveMethod(pass, funcInfo)
	}
	
	// Handle package functions like net/http.Get
	obj := analysisutil.ObjectOf(pass, funcInfo.Package, funcInfo.Function)
	if obj == nil {
		return nil
	}
	
	if fn, ok := obj.(*types.Func); ok {
		return fn
	}
	
	return nil
}

// resolveMethod resolves a method based on its receiver type
func (c *UnifiedChecker) resolveMethod(pass *analysis.Pass, funcInfo FunctionInfo) *types.Func {
	// Parse receiver type like "*net/http.Client" or "*database/sql.DB"
	receiver := funcInfo.MethodReceiver
	if !strings.HasPrefix(receiver, "*") {
		return nil
	}
	
	// Remove the * prefix
	receiver = receiver[1:]
	
	// Split package and type name
	parts := strings.Split(receiver, ".")
	if len(parts) != 2 {
		return nil
	}
	
	packagePath := parts[0]
	typeName := parts[1]
	
	// Get the type
	typ := analysisutil.TypeOf(pass, packagePath, "*"+typeName)
	if typ == nil {
		return nil
	}
	
	// Get the method
	method := analysisutil.MethodOf(typ, funcInfo.Function)
	return method
}

// findMatchingCallExpr finds a call expression that matches the function info
func (c *UnifiedChecker) findMatchingCallExpr(callExprs []*ast.CallExpr, funcInfo FunctionInfo) *ast.CallExpr {
	for _, callExpr := range callExprs {
		if c.matchesFunction(callExpr, funcInfo) {
			return callExpr
		}
	}
	return nil
}

// matchesFunction checks if a call expression matches the function info
func (c *UnifiedChecker) matchesFunction(callExpr *ast.CallExpr, funcInfo FunctionInfo) bool {
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if funcInfo.MethodReceiver != "" {
		// Handle method calls like client.Get() where client is *http.Client
		// We'll do a simpler check here - just match the method name
		// The SSA analysis will do the precise type checking
		return sel.Sel.Name == funcInfo.Function
	}

	// Handle package function calls
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	expectedPkg := funcInfo.Package
	if strings.Contains(expectedPkg, "/") {
		// For qualified packages like "net/http", use the last part
		parts := strings.Split(expectedPkg, "/")
		expectedPkg = parts[len(parts)-1]
	}

	return ident.Name == expectedPkg && sel.Sel.Name == funcInfo.Function
}

// generateSuggestedFix creates a suggested fix based on the function info and pattern
func (c *UnifiedChecker) generateSuggestedFix(pass *analysis.Pass, callExpr *ast.CallExpr, funcInfo FunctionInfo) *analysis.SuggestedFix {
	// Verify this matches the expected function
	if !c.matchesFunction(callExpr, funcInfo) {
		return nil
	}

	// Detect the appropriate context to use
	contextExpr := c.contextDetector.DetectContext(pass, callExpr)
	
	// Generate fix based on pattern
	switch funcInfo.AutofixPattern {
	case "http-get-pattern":
		return c.generateHTTPGetPattern(pass, callExpr, contextExpr, "http.MethodGet")
	case "http-head-pattern":
		return c.generateHTTPGetPattern(pass, callExpr, contextExpr, "http.MethodHead")
	case "http-post-pattern":
		return c.generateHTTPPostPattern(pass, callExpr, contextExpr)
	case "http-postform-pattern":
		return c.generateHTTPPostFormPattern(pass, callExpr, contextExpr)
	case "http-newrequest-pattern":
		return c.generateHTTPNewRequestPattern(pass, callExpr, contextExpr)
	case "net-dial-pattern":
		return c.generateNetDialPattern(pass, callExpr, contextExpr)
	case "net-dialtimeout-pattern":
		return c.generateNetDialTimeoutPattern(pass, callExpr, contextExpr)
	case "net-listen-pattern":
		return c.generateNetListenPattern(pass, callExpr, contextExpr)
	case "net-listenpacket-pattern":
		return c.generateNetListenPacketPattern(pass, callExpr, contextExpr)
	case "net-lookup-pattern":
		return c.generateNetLookupPattern(pass, callExpr, contextExpr, funcInfo.Function)
	case "exec-command-pattern":
		return c.generateExecCommandPattern(pass, callExpr, contextExpr)
	case "sql-db-pattern":
		return c.generateSQLDBPattern(pass, callExpr, contextExpr, funcInfo.Function)
	case "sql-tx-pattern":
		return c.generateSQLTxPattern(pass, callExpr, contextExpr, funcInfo.Function)
	case "sql-stmt-pattern":
		return c.generateSQLStmtPattern(pass, callExpr, contextExpr, funcInfo.Function)
	case "tls-dial-pattern":
		return c.generateTLSDialPattern(pass, callExpr, contextExpr)
	case "tls-dialwithdialer-pattern":
		return c.generateTLSDialWithDialerPattern(pass, callExpr, contextExpr)
	case "tls-handshake-pattern":
		return c.generateTLSHandshakePattern(pass, callExpr, contextExpr)
	case "http-client-get-pattern":
		return c.generateHTTPClientGetPattern(pass, callExpr, contextExpr)
	case "http-client-head-pattern":
		return c.generateHTTPClientHeadPattern(pass, callExpr, contextExpr)
	case "http-client-post-pattern":
		return c.generateHTTPClientPostPattern(pass, callExpr, contextExpr)
	case "http-client-postform-pattern":
		return c.generateHTTPClientPostFormPattern(pass, callExpr, contextExpr)
	default:
		return nil
	}
}

// GetSupportedFunctionNames returns the set of function names supported by this checker
func (c *UnifiedChecker) GetSupportedFunctionNames() map[string]bool {
	result := make(map[string]bool)
	for funcName := range c.supportedFunctions {
		result[funcName] = true
	}
	return result
}

// getSupportedFunctions returns the map of all supported functions with their autofix patterns
func getSupportedFunctions() map[string]FunctionInfo {
	return map[string]FunctionInfo{
		// HTTP package functions
		"net/http.Get": {
			Package:        "net/http",
			Function:       "Get",
			AutofixPattern: "http-get-pattern",
		},
		"net/http.Head": {
			Package:        "net/http", 
			Function:       "Head",
			AutofixPattern: "http-head-pattern",
		},
		"net/http.Post": {
			Package:        "net/http",
			Function:       "Post", 
			AutofixPattern: "http-post-pattern",
		},
		"net/http.PostForm": {
			Package:        "net/http",
			Function:       "PostForm",
			AutofixPattern: "http-postform-pattern",
		},
		"net/http.NewRequest": {
			Package:        "net/http",
			Function:       "NewRequest",
			AutofixPattern: "http-newrequest-pattern",
		},
		
		// Net package functions
		"net.Dial": {
			Package:        "net",
			Function:       "Dial",
			AutofixPattern: "net-dial-pattern",
		},
		"net.DialTimeout": {
			Package:        "net",
			Function:       "DialTimeout",
			AutofixPattern: "net-dialtimeout-pattern",
		},
		"net.Listen": {
			Package:        "net",
			Function:       "Listen",
			AutofixPattern: "net-listen-pattern",
		},
		"net.ListenPacket": {
			Package:        "net",
			Function:       "ListenPacket",
			AutofixPattern: "net-listenpacket-pattern",
		},
		"net.LookupCNAME": {
			Package:        "net",
			Function:       "LookupCNAME",
			AutofixPattern: "net-lookup-pattern",
		},
		"net.LookupHost": {
			Package:        "net",
			Function:       "LookupHost",
			AutofixPattern: "net-lookup-pattern",
		},
		"net.LookupIP": {
			Package:        "net",
			Function:       "LookupIP",
			AutofixPattern: "net-lookup-pattern",
		},
		"net.LookupPort": {
			Package:        "net",
			Function:       "LookupPort",
			AutofixPattern: "net-lookup-pattern",
		},
		"net.LookupSRV": {
			Package:        "net",
			Function:       "LookupSRV",
			AutofixPattern: "net-lookup-pattern",
		},
		"net.LookupMX": {
			Package:        "net",
			Function:       "LookupMX",
			AutofixPattern: "net-lookup-pattern",
		},
		"net.LookupNS": {
			Package:        "net",
			Function:       "LookupNS",
			AutofixPattern: "net-lookup-pattern",
		},
		"net.LookupTXT": {
			Package:        "net",
			Function:       "LookupTXT",
			AutofixPattern: "net-lookup-pattern",
		},
		"net.LookupAddr": {
			Package:        "net",
			Function:       "LookupAddr",
			AutofixPattern: "net-lookup-pattern",
		},
		
		// Exec package functions
		"os/exec.Command": {
			Package:        "os/exec",
			Function:       "Command",
			AutofixPattern: "exec-command-pattern",
		},
		
		// Crypto/TLS package functions
		"crypto/tls.Dial": {
			Package:        "crypto/tls",
			Function:       "Dial",
			AutofixPattern: "tls-dial-pattern",
		},
		"crypto/tls.DialWithDialer": {
			Package:        "crypto/tls",
			Function:       "DialWithDialer",
			AutofixPattern: "tls-dialwithdialer-pattern",
		},
		"(*crypto/tls.Conn).Handshake": {
			Package:        "crypto/tls",
			Function:       "Handshake",
			MethodReceiver: "*crypto/tls.Conn",
			AutofixPattern: "tls-handshake-pattern",
		},
		
		// HTTP Client methods
		"(*net/http.Client).Get": {
			Package:        "net/http",
			Function:       "Get",
			MethodReceiver: "*net/http.Client",
			AutofixPattern: "http-client-get-pattern",
		},
		"(*net/http.Client).Head": {
			Package:        "net/http",
			Function:       "Head",
			MethodReceiver: "*net/http.Client",
			AutofixPattern: "http-client-head-pattern",
		},
		"(*net/http.Client).Post": {
			Package:        "net/http",
			Function:       "Post",
			MethodReceiver: "*net/http.Client",
			AutofixPattern: "http-client-post-pattern",
		},
		"(*net/http.Client).PostForm": {
			Package:        "net/http",
			Function:       "PostForm",
			MethodReceiver: "*net/http.Client",
			AutofixPattern: "http-client-postform-pattern",
		},
		
		// Database SQL DB methods
		"(*database/sql.DB).Begin": {
			Package:        "database/sql",
			Function:       "Begin",
			MethodReceiver: "*database/sql.DB",
			AutofixPattern: "sql-db-pattern",
		},
		"(*database/sql.DB).Exec": {
			Package:        "database/sql",
			Function:       "Exec",
			MethodReceiver: "*database/sql.DB",
			AutofixPattern: "sql-db-pattern",
		},
		"(*database/sql.DB).Ping": {
			Package:        "database/sql",
			Function:       "Ping",
			MethodReceiver: "*database/sql.DB",
			AutofixPattern: "sql-db-pattern",
		},
		"(*database/sql.DB).Prepare": {
			Package:        "database/sql",
			Function:       "Prepare",
			MethodReceiver: "*database/sql.DB",
			AutofixPattern: "sql-db-pattern",
		},
		"(*database/sql.DB).Query": {
			Package:        "database/sql",
			Function:       "Query",
			MethodReceiver: "*database/sql.DB",
			AutofixPattern: "sql-db-pattern",
		},
		"(*database/sql.DB).QueryRow": {
			Package:        "database/sql",
			Function:       "QueryRow",
			MethodReceiver: "*database/sql.DB",
			AutofixPattern: "sql-db-pattern",
		},
		
		// Database SQL Tx methods
		"(*database/sql.Tx).Exec": {
			Package:        "database/sql",
			Function:       "Exec",
			MethodReceiver: "*database/sql.Tx",
			AutofixPattern: "sql-tx-pattern",
		},
		"(*database/sql.Tx).Prepare": {
			Package:        "database/sql",
			Function:       "Prepare",
			MethodReceiver: "*database/sql.Tx",
			AutofixPattern: "sql-tx-pattern",
		},
		"(*database/sql.Tx).Query": {
			Package:        "database/sql",
			Function:       "Query",
			MethodReceiver: "*database/sql.Tx",
			AutofixPattern: "sql-tx-pattern",
		},
		"(*database/sql.Tx).QueryRow": {
			Package:        "database/sql",
			Function:       "QueryRow",
			MethodReceiver: "*database/sql.Tx",
			AutofixPattern: "sql-tx-pattern",
		},
		"(*database/sql.Tx).Stmt": {
			Package:        "database/sql",
			Function:       "Stmt",
			MethodReceiver: "*database/sql.Tx",
			AutofixPattern: "sql-tx-pattern",
		},
		
		// Database SQL Stmt methods
		"(*database/sql.Stmt).Exec": {
			Package:        "database/sql",
			Function:       "Exec",
			MethodReceiver: "*database/sql.Stmt",
			AutofixPattern: "sql-stmt-pattern",
		},
		"(*database/sql.Stmt).Query": {
			Package:        "database/sql",
			Function:       "Query",
			MethodReceiver: "*database/sql.Stmt",
			AutofixPattern: "sql-stmt-pattern",
		},
		"(*database/sql.Stmt).QueryRow": {
			Package:        "database/sql",
			Function:       "QueryRow",
			MethodReceiver: "*database/sql.Stmt",
			AutofixPattern: "sql-stmt-pattern",
		},
	}
}