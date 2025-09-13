package noctx_test

import (
	"testing"

	"github.com/sonatard/noctx"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAutofixAnalyzer(t *testing.T) {
	testCases := []struct {
		desc string
	}{
		{desc: "http_autofix"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			analysistest.Run(t, analysistest.TestData(), noctx.Analyzer, test.desc)
		})
	}
}