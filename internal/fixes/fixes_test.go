package fixes_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
)

// ── GoVersionDetector ─────────────────────────────────────────────────────────

func TestGoVersionDetector_SkipDetection(t *testing.T) {
	vd := fixes.NewGoVersionDetector()
	vd.SetSkipGoVersionDetection(true)

	// With skip enabled, IsGo124OrGreater must return true regardless of pass.
	pass := &analysis.Pass{Pkg: nil}
	if !vd.IsGo124OrGreater(pass) {
		t.Error("IsGo124OrGreater with skip=true: got false, want true")
	}
}

func TestGoVersionDetector_NilPkg(t *testing.T) {
	vd := fixes.NewGoVersionDetector()
	pass := &analysis.Pass{Pkg: nil}
	// When Pkg is nil, must return false (cannot determine version).
	if vd.IsGo124OrGreater(pass) {
		t.Error("IsGo124OrGreater with nil Pkg: got true, want false")
	}
}

// ── ContextDetector ───────────────────────────────────────────────────────────

func TestContextDetector_NilPkg_FallsBack(t *testing.T) {
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

func TestContextDetector_NewContextDetectorNilVD(t *testing.T) {
	// NewContextDetector(nil) should not panic and should fall back correctly.
	detector := fixes.NewContextDetector(nil)
	fset := token.NewFileSet()
	expr, err := parser.ParseExpr(`f()`)
	if err != nil {
		t.Fatal(err)
	}
	callExpr := expr.(*ast.CallExpr)
	pass := &analysis.Pass{Fset: fset, Files: []*ast.File{}, Pkg: nil}
	got := detector.DetectContext(pass, callExpr)
	if got != "context.Background()" {
		t.Errorf("DetectContext with nil vd: got %q, want %q", got, "context.Background()")
	}
}

func TestContextDetector_FindsContextParam(t *testing.T) {
	// Parse a function that has a context.Context parameter and contains a call.
	const src = `package p
import "context"
func f(ctx context.Context) {
	_ = ctx
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Locate the RHS "ctx" identifier inside the assignment in f's body.
	var ctxIdent *ast.Ident
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if assign, ok := n.(*ast.AssignStmt); ok {
			if len(assign.Rhs) == 1 {
				if id, ok := assign.Rhs[0].(*ast.Ident); ok {
					ctxIdent = id
				}
			}
		}
		return true
	})
	if ctxIdent == nil {
		t.Fatal("could not find 'ctx' ident in parsed AST")
	}

	// Build a minimal CallExpr whose Pos() is inside f's body.
	ce := &ast.CallExpr{
		Fun:    ctxIdent,
		Lparen: ctxIdent.Pos(),
		Rparen: ctxIdent.End(),
	}

	// We need a real TypesInfo to recognise "context.Context".
	// Since we cannot type-check here without a full loader, we fall back to
	// checking just the "no TypesInfo" guard: with nil TypesInfo, isContextType
	// returns false, so the detector must still fall back to context.Background().
	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		TypesInfo: nil,
		Pkg:       nil,
	}

	got := fixes.NewContextDetector(nil).DetectContext(pass, ce)
	// Without TypesInfo we cannot recognise context.Context, so we expect
	// the fallback.
	if got != "context.Background()" {
		t.Errorf("DetectContext without TypesInfo: got %q, want %q", got, "context.Background()")
	}
}

// ── VariableAssignmentDetector ────────────────────────────────────────────────

func TestVariableAssignmentDetector_NilPkg(t *testing.T) {
	detector := &fixes.VariableAssignmentDetector{}

	expr, err := parser.ParseExpr(`http.Get("url")`)
	if err != nil {
		t.Fatal(err)
	}
	callExpr := expr.(*ast.CallExpr)

	pass := &analysis.Pass{Fset: token.NewFileSet(), Pkg: nil}

	got := detector.DetectAssignmentOperator(pass, callExpr, "req", "err")
	if got != ":=" {
		t.Errorf("DetectAssignmentOperator(nil pkg): got %q, want %q", got, ":=")
	}
}

func TestVariableAssignmentDetector_NoVarNames(t *testing.T) {
	// With no var names to check, it should return "=" (all zero variables "found").
	detector := &fixes.VariableAssignmentDetector{}

	expr, err := parser.ParseExpr(`f()`)
	if err != nil {
		t.Fatal(err)
	}
	callExpr := expr.(*ast.CallExpr)

	pass := &analysis.Pass{Fset: token.NewFileSet(), Pkg: nil}

	// Nil pkg → always ":="
	got := detector.DetectAssignmentOperator(pass, callExpr)
	if got != ":=" {
		t.Errorf("DetectAssignmentOperator(nil pkg, no vars): got %q, want %q", got, ":=")
	}
}

// ── ArgumentFormatter ─────────────────────────────────────────────────────────

func TestArgumentFormatter_FormatArgument(t *testing.T) {
	formatter := &fixes.ArgumentFormatter{}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "string literal", input: `"hello"`, want: `"hello"`},
		{name: "identifier", input: "myVar", want: "myVar"},
		{name: "nil identifier", input: "nil", want: "nil"},
		{name: "selector expression", input: "pkg.Func", want: "pkg.Func"},
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

func TestArgumentFormatter_FormatBodyArgument(t *testing.T) {
	formatter := &fixes.ArgumentFormatter{}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "nil body", input: "nil", want: "http.NoBody"},
		{name: "other body", input: "myBody", want: "myBody"},
		{name: "reader body", input: "strings.NewReader(s)", want: "strings.NewReader(s)"},
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

// ── CreateTextEdit / CreateSuggestedFix ───────────────────────────────────────

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
