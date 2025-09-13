package noctx_test

import (
	"testing"

	"github.com/sonatard/noctx"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestHttpGetSuggestedFixes(t *testing.T) {
	testdir := analysistest.TestData()
	
	// Test that suggested fixes are generated for http.Get and http.Head
	results := analysistest.Run(t, testdir, noctx.Analyzer, "http_get_test")
	
	// Check that at least one diagnostic has a suggested fix
	hasFixedDiagnostic := false
	for _, result := range results {
		for _, diagnostic := range result.Diagnostics {
			t.Logf("Diagnostic: %s (fixes: %d)", diagnostic.Message, len(diagnostic.SuggestedFixes))
			if len(diagnostic.SuggestedFixes) > 0 {
				hasFixedDiagnostic = true
				// Verify the fix content
				for _, fix := range diagnostic.SuggestedFixes {
					t.Logf("Found suggested fix: %s", fix.Message)
					for _, edit := range fix.TextEdits {
						t.Logf("  TextEdit: %s", string(edit.NewText))
					}
				}
			}
		}
	}
	
	if !hasFixedDiagnostic {
		t.Error("Expected at least one diagnostic with a suggested fix")
	}
}