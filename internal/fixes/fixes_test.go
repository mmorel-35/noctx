package fixes_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
)

func TestContextDetector(t *testing.T) {
	detector := &fixes.ContextDetector{}

	fset := token.NewFileSet()
	expr, err := parser.ParseExpr(`http.Get("url")`)
	if err != nil {
		t.Fatal(err)
	}

	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}

	// Pass with nil Pkg triggers the fallback path.
	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{},
		Pkg:   nil,
	}

	got := detector.DetectContext(pass, callExpr)
	want := "context.Background()"
	if got != want {
		t.Errorf("DetectContext: got %q, want %q", got, want)
	}
}

func TestVariableAssignmentDetector(t *testing.T) {
	detector := &fixes.VariableAssignmentDetector{}

	fset := token.NewFileSet()
	expr, err := parser.ParseExpr(`http.Get("url")`)
	if err != nil {
		t.Fatal(err)
	}

	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}

	pass := &analysis.Pass{
		Fset: fset,
		Pkg:  nil,
	}

	// With nil Pkg, should always return ":=".
	got := detector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	want := ":="
	if got != want {
		t.Errorf("DetectAssignmentOperator: got %q, want %q", got, want)
	}
}

func TestArgumentFormatter(t *testing.T) {
	formatter := &fixes.ArgumentFormatter{}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "string literal", input: `"hello"`, want: `"hello"`},
		{name: "identifier", input: "myVar", want: "myVar"},
		{name: "nil identifier", input: "nil", want: "nil"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			pass := &analysis.Pass{Fset: token.NewFileSet()}
			got := formatter.FormatArgument(pass, expr)
			if got != tc.want {
				t.Errorf("FormatArgument(%q): got %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatBodyArgument(t *testing.T) {
	formatter := &fixes.ArgumentFormatter{}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "nil body", input: "nil", want: "http.NoBody"},
		{name: "other body", input: "myBody", want: "myBody"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			pass := &analysis.Pass{Fset: token.NewFileSet()}
			got := formatter.FormatBodyArgument(pass, expr)
			if got != tc.want {
				t.Errorf("FormatBodyArgument(%q): got %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCreateTextEdit(t *testing.T) {
	expr, err := parser.ParseExpr(`http.Get("url")`)
	if err != nil {
		t.Fatal(err)
	}

	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}

	newCall := `http.NewRequestWithContext(...)`
	edit := fixes.CreateTextEdit(callExpr, newCall)

	if string(edit.NewText) != newCall {
		t.Errorf("CreateTextEdit: NewText = %q, want %q", string(edit.NewText), newCall)
	}
	if edit.Pos != callExpr.Pos() {
		t.Error("CreateTextEdit: Pos mismatch")
	}
	if edit.End != callExpr.End() {
		t.Error("CreateTextEdit: End mismatch")
	}
}

func TestCreateSuggestedFix(t *testing.T) {
	expr, err := parser.ParseExpr(`http.Get("url")`)
	if err != nil {
		t.Fatal(err)
	}

	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}

	message := "Replace with http.NewRequestWithContext"
	newCall := `http.NewRequestWithContext(...)`

	fix := fixes.CreateSuggestedFix(message, callExpr, newCall)

	if fix.Message != message {
		t.Errorf("CreateSuggestedFix: Message = %q, want %q", fix.Message, message)
	}
	if len(fix.TextEdits) != 1 {
		t.Fatalf("CreateSuggestedFix: len(TextEdits) = %d, want 1", len(fix.TextEdits))
	}
	if string(fix.TextEdits[0].NewText) != newCall {
		t.Errorf("CreateSuggestedFix: TextEdits[0].NewText = %q, want %q", string(fix.TextEdits[0].NewText), newCall)
	}
}
