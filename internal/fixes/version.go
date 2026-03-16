package fixes

import (
	"go/build"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// GoVersionDetector provides Go version detection capabilities.
type GoVersionDetector struct {
	skipGoVersionDetection bool
}

// NewGoVersionDetector creates a new GoVersionDetector.
func NewGoVersionDetector() *GoVersionDetector {
	return &GoVersionDetector{}
}

// IsGo124OrGreater reports whether the package under analysis requires Go 1.24
// or later. This is used to decide whether t.Context() should be suggested
// inside test functions.
func (g *GoVersionDetector) IsGo124OrGreater(pass *analysis.Pass) bool {
	if g.skipGoVersionDetection {
		return true
	}

	// Prior to go1.22, pass.Pkg.GoVersion() only reflects the toolchain version,
	// not the module's declared minimum version, so the result would be
	// unreliable. Disable the t.Context() suggestion on toolchains older than
	// go1.22 to avoid false positives.
	if !slices.Contains(build.Default.ReleaseTags, "go1.22") {
		return false
	}

	if pass.Pkg == nil {
		return false
	}

	pkgVersion := pass.Pkg.GoVersion()
	if pkgVersion == "" {
		// Empty version string can mean a development build or a malformed module
		// configuration; assume modern Go to be conservative about t.Context().
		return true
	}

	raw := strings.TrimPrefix(pkgVersion, "go")

	// Strip pre-release suffixes (e.g. "go1.24rc1" → "1.24").
	if idx := strings.IndexFunc(raw, func(r rune) bool {
		return (r < '0' || r > '9') && r != '.'
	}); idx != -1 {
		raw = raw[:idx]
	}

	parts := strings.Split(raw, ".")
	if len(parts) < 2 {
		return false
	}

	// Convert "1.24" → 124 for a simple integer comparison.
	v, err := strconv.Atoi(strings.Join(parts[:2], ""))
	if err != nil {
		return false
	}

	return v >= 124
}

// SetSkipGoVersionDetection disables Go version detection (useful in tests).
func (g *GoVersionDetector) SetSkipGoVersionDetection(skip bool) {
	g.skipGoVersionDetection = skip
}
