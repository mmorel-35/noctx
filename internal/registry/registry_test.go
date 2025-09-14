package registry_test

import (
	"testing"

	"github.com/sonatard/noctx/internal/registry"
)

func TestGetAllRules(t *testing.T) {
	allRules := registry.GetAllRules()
	
	// Should contain both autofix rules and legacy rules
	if len(allRules) == 0 {
		t.Error("Expected non-empty rules map")
	}
	
	// Check for some known rules
	expectedRules := []string{
		"net/http.Get",
		"net/http.NewRequest", 
		"net.Dial",
		"os/exec.Command",
		"crypto/tls.Dial",
		"(*database/sql.DB).Begin", // legacy rule
	}
	
	for _, ruleName := range expectedRules {
		if _, exists := allRules[ruleName]; !exists {
			t.Errorf("Expected rule %s not found", ruleName)
		}
	}
}

func TestGetRulesByChecker(t *testing.T) {
	grouped := registry.GetRulesByChecker()
	
	// Should have rules for all checker types
	expectedCheckers := []string{
		"http",
		"net",
		"exec",
		"tls",
	}
	
	for _, checkerName := range expectedCheckers {
		if rules, exists := grouped[checkerName]; !exists || len(rules) == 0 {
			t.Errorf("Expected rules for checker %s", checkerName)
		}
	}
	
	// Verify HTTP checker has correct number of rules
	httpRules := grouped["http"]
	expectedHTTPCount := 5 // Get, Head, Post, PostForm, NewRequest
	if len(httpRules) != expectedHTTPCount {
		t.Errorf("Expected %d HTTP rules, got %d", expectedHTTPCount, len(httpRules))
	}
}

func TestGetAutofixFunctions(t *testing.T) {
	autofixFuncs := registry.GetAutofixFunctions()
	
	// Should contain autofix-supported functions
	expectedFunctions := []string{
		"net/http.Get",
		"net/http.NewRequest",
		"net.Dial",
		"os/exec.Command",
		"crypto/tls.Dial",
	}
	
	for _, funcName := range expectedFunctions {
		if !autofixFuncs[funcName] {
			t.Errorf("Expected autofix function %s not found", funcName)
		}
	}
	
	// Should NOT contain legacy functions
	legacyFunctions := []string{
		"(*database/sql.DB).Begin",
		"(*net/http.Client).Get",
	}
	
	for _, funcName := range legacyFunctions {
		if autofixFuncs[funcName] {
			t.Errorf("Legacy function %s should not have autofix support", funcName)
		}
	}
}

func TestGetMessage(t *testing.T) {
	tests := []struct {
		funcName     string
		expectedMsg  string
	}{
		{
			funcName:    "net/http.Get",
			expectedMsg: "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		},
		{
			funcName:    "net.Dial",
			expectedMsg: "must not be called. use (*net.Dialer).DialContext",
		},
		{
			funcName:    "unknown.Function",
			expectedMsg: "must not be called without context",
		},
	}
	
	for _, test := range tests {
		t.Run(test.funcName, func(t *testing.T) {
			msg := registry.GetMessage(test.funcName)
			if msg != test.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", test.expectedMsg, msg)
			}
		})
	}
}

func TestFormatDiagnostic(t *testing.T) {
	funcName := "net/http.Get"
	expected := "net/http.Get must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)"
	
	result := registry.FormatDiagnostic(funcName)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestRuleConsistency(t *testing.T) {
	// Test that all functions now have proper configuration
	allRules := registry.GetAllRules()
	
	// Count functions with and without autofix
	autofixCount := 0
	for _, rule := range allRules {
		if rule.HasAutofix && rule.CheckerType != "" {
			autofixCount++
		}
	}
	
	// We should have functions with autofix support
	if autofixCount == 0 {
		t.Error("Expected at least some functions to have autofix support")
	}
	
	// Test that Rules entries have proper configuration when they have autofix
	for name, rule := range registry.Rules {
		if rule.HasAutofix && rule.CheckerType == "" {
			t.Errorf("Rule %s has autofix but no CheckerType", name)
		}
		if !rule.HasAutofix && rule.CheckerType != "" {
			t.Errorf("Rule %s has CheckerType but no autofix", name)
		}
	}
}