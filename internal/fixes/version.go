package fixes

import (
	"go/build"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// GoVersionDetector provides Go version detection capabilities
type GoVersionDetector struct {
	skipGoVersionDetection bool
}

// NewGoVersionDetector creates a new Go version detector
func NewGoVersionDetector() *GoVersionDetector {
	return &GoVersionDetector{
		skipGoVersionDetection: false,
	}
}

// IsGo124OrGreater checks if the current package supports Go 1.24 or greater
// This is used to determine if t.Context() should be suggested in test functions
func (g *GoVersionDetector) IsGo124OrGreater(pass *analysis.Pass) bool {
	if g.skipGoVersionDetection {
		return true
	}

	// Prior to go1.22, versions.FileVersion returns only the toolchain version,
	// which is of no use to us, so disable this check on earlier versions.
	if !slices.Contains(build.Default.ReleaseTags, "go1.22") {
		return false
	}

	pkgVersion := pass.Pkg.GoVersion()
	if pkgVersion == "" {
		// Empty means Go devel - assume it supports t.Context()
		return true
	}

	raw := strings.TrimPrefix(pkgVersion, "go")

	// Handle prerelease versions (go1.24rc1)
	idx := strings.IndexFunc(raw, func(r rune) bool {
		return (r < '0' || r > '9') && r != '.'
	})

	if idx != -1 {
		raw = raw[:idx]
	}

	vParts := strings.Split(raw, ".")
	if len(vParts) < 2 {
		return false
	}

	// Convert major.minor to integer (1.24 -> 124)
	v, err := strconv.Atoi(strings.Join(vParts[:2], ""))
	if err != nil {
		// Default to older version if we can't parse
		return false
	}

	return v >= 124
}

// SetSkipGoVersionDetection allows skipping Go version detection for testing
func (g *GoVersionDetector) SetSkipGoVersionDetection(skip bool) {
	g.skipGoVersionDetection = skip
}