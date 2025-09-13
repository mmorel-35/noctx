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
	// Test that Rules and LegacyRules don't have overlapping keys
	for name := range registry.Rules {
		if _, exists := registry.LegacyRules[name]; exists {
			t.Errorf("Function %s exists in both Rules and LegacyRules", name)
		}
	}
	
	// Test that all Rules have autofix set to true
	for name, rule := range registry.Rules {
		if !rule.HasAutofix {
			t.Errorf("Rule %s should have HasAutofix=true", name)
		}
		if rule.CheckerType == "" {
			t.Errorf("Rule %s should have a CheckerType", name)
		}
	}
	
	// Test that all LegacyRules have autofix set to false
	for name, rule := range registry.LegacyRules {
		if rule.HasAutofix {
			t.Errorf("Legacy rule %s should have HasAutofix=false", name)
		}
	}
}