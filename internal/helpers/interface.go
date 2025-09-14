package helpers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// InterfaceImplementationChecker provides utilities for checking interface implementations
type InterfaceImplementationChecker struct{}

// NewInterfaceImplementationChecker creates a new interface implementation checker
func NewInterfaceImplementationChecker() *InterfaceImplementationChecker {
	return &InterfaceImplementationChecker{}
}

// Implements checks if the given expression implements the specified interface
// This is inspired by testifylint's helpers_interface.go implements function
func (ic *InterfaceImplementationChecker) Implements(pass *analysis.Pass, expr ast.Expr, ifaceObj types.Object) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	
	iface, ok := ifaceObj.Type().Underlying().(*types.Interface)
	if !ok {
		return false
	}
	
	return types.Implements(t, iface)
}

// ImplementsHTTPClient checks if the expression implements the basic HTTP client interface
// This can be used to detect when we need method-based analysis for HTTP client calls
func (ic *InterfaceImplementationChecker) ImplementsHTTPClient(pass *analysis.Pass, expr ast.Expr) bool {
	// For now, we'll do basic type checking
	// This is where we could implement more sophisticated interface checking
	// if we had specific interface requirements
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	
	// Check if it's *http.Client or something that embeds it
	// This is a simplified version - in practice we might need more sophisticated checking
	typeStr := t.String()
	return typeStr == "*net/http.Client" || 
		   typeStr == "net/http.Client"
}

// ImplementsDBInterface checks if the expression implements database interface patterns
func (ic *InterfaceImplementationChecker) ImplementsDBInterface(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	
	typeStr := t.String()
	return typeStr == "*database/sql.DB" || 
		   typeStr == "*database/sql.Conn" ||
		   typeStr == "*database/sql.Tx"
}

// IsMethodCall checks if the call expression is a method call (has a receiver)
func (ic *InterfaceImplementationChecker) IsMethodCall(callExpr *ast.CallExpr) bool {
	if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		// This is a method call if the selector has an expression (receiver)
		return sel.X != nil
	}
	return false
}

// GetMethodReceiver returns the receiver expression for a method call
func (ic *InterfaceImplementationChecker) GetMethodReceiver(callExpr *ast.CallExpr) ast.Expr {
	if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		return sel.X
	}
	return nil
}