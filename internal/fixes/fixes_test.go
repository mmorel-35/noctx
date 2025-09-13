package fixes_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sonatard/noctx/internal/fixes"
)

func TestContextDetector(t *testing.T) {
	detector := &fixes.ContextDetector{}
	
	// Test case 1: Simple fallback when no package is available
	fset := token.NewFileSet()
	expr, err := parser.ParseExpr("http.Get(\"url\")")
	if err != nil {
		t.Fatal(err)
	}
	
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}
	
	// Create minimal pass for testing - Pkg is nil
	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{},
		Pkg:   nil, // This will trigger the fallback path
	}
	
	// Test context detection - should fallback to context.Background()
	contextExpr := detector.DetectContext(pass, callExpr)
	expectedFallback := "context.Background()"
	if contextExpr != expectedFallback {
		t.Errorf("Expected fallback context %s, got %s", expectedFallback, contextExpr)
	}
}

func TestVariableAssignmentDetector(t *testing.T) {
	detector := &fixes.VariableAssignmentDetector{}
	
	// Create a mock analysis.Pass and call expression for testing
	fset := token.NewFileSet()
	
	expr, err := parser.ParseExpr("http.Get(\"url\")")
	if err != nil {
		t.Fatal(err)
	}
	
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}
	
	// Create minimal pass for testing
	pass := &analysis.Pass{
		Fset: fset,
		Pkg:  nil,
	}
	
	// Test assignment operator detection
	assignOp := detector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	
	// Should default to := when variables are not found
	expected := ":="
	if assignOp != expected {
		t.Errorf("Expected assignment operator %s, got %s", expected, assignOp)
	}
}

func TestArgumentFormatter(t *testing.T) {
	formatter := &fixes.ArgumentFormatter{}
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "string literal",
			input:    `"hello"`,
			expected: `"hello"`,
		},
		{
			name:     "identifier",
			input:    "myVar",
			expected: "myVar",
		},
		{
			name:     "nil identifier",
			input:    "nil",
			expected: "nil",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(test.input)
			if err != nil {
				t.Fatal(err)
			}
			
			// Create minimal pass for testing
			pass := &analysis.Pass{
				Fset: token.NewFileSet(),
			}
			
			result := formatter.FormatArgument(pass, expr)
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestFormatBodyArgument(t *testing.T) {
	formatter := &fixes.ArgumentFormatter{}
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "nil body",
			input:    "nil",
			expected: "http.NoBody",
		},
		{
			name:     "other body",
			input:    "myBody",
			expected: "myBody",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(test.input)
			if err != nil {
				t.Fatal(err)
			}
			
			// Create minimal pass for testing
			pass := &analysis.Pass{
				Fset: token.NewFileSet(),
			}
			
			result := formatter.FormatBodyArgument(pass, expr)
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestCreateTextEdit(t *testing.T) {
	expr, err := parser.ParseExpr("http.Get(\"url\")")
	if err != nil {
		t.Fatal(err)
	}
	
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}
	
	newCall := "http.NewRequestWithContext(...)"
	edit := fixes.CreateTextEdit(callExpr, newCall)
	
	if string(edit.NewText) != newCall {
		t.Errorf("Expected NewText %s, got %s", newCall, string(edit.NewText))
	}
	
	if edit.Pos != callExpr.Pos() {
		t.Error("TextEdit position mismatch")
	}
	
	if edit.End != callExpr.End() {
		t.Error("TextEdit end position mismatch")
	}
}

func TestCreateSuggestedFix(t *testing.T) {
	expr, err := parser.ParseExpr("http.Get(\"url\")")
	if err != nil {
		t.Fatal(err)
	}
	
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}
	
	message := "Replace with http.NewRequestWithContext"
	newCall := "http.NewRequestWithContext(...)"
	
	fix := fixes.CreateSuggestedFix(message, callExpr, newCall)
	
	if fix.Message != message {
		t.Errorf("Expected message %s, got %s", message, fix.Message)
	}
	
	if len(fix.TextEdits) != 1 {
		t.Errorf("Expected 1 text edit, got %d", len(fix.TextEdits))
	}
	
	if string(fix.TextEdits[0].NewText) != newCall {
		t.Errorf("Expected text edit content %s, got %s", newCall, string(fix.TextEdits[0].NewText))
	}
}

// Integration test with analysistest
func TestFixesIntegration(t *testing.T) {
	// This would require a full setup with testdata
	// For now, just verify the basic structure works
	testdata := analysistest.TestData()
	if testdata == "" {
		t.Skip("No testdata available")
	}
}