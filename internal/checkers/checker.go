package checkers

import (
	"golang.org/x/tools/go/analysis"
)

// Checker defines the interface for function call checkers
type Checker interface {
	// Check performs the analysis and reports violations
	Check(pass *analysis.Pass) error
	
	// Name returns the name of this checker
	Name() CheckerName
}

// CheckerName represents the type of checker
type CheckerName string

const (
	// Consolidated checkers
	HTTPCheckerName CheckerName = "http"
	NetCheckerName  CheckerName = "net"
	ExecCheckerName CheckerName = "exec"
	TLSCheckerName  CheckerName = "tls"
)

// CheckerFactory creates a new instance of a checker
type CheckerFactory func() Checker

// Registry holds all available checker factories
var Registry = map[CheckerName]CheckerFactory{
	HTTPCheckerName: func() Checker { return NewHTTPChecker() },
	NetCheckerName:  func() Checker { return NewNetChecker() },
	ExecCheckerName: func() Checker { return NewExecChecker() },
	TLSCheckerName:  func() Checker { return NewTLSChecker() },
}

// GetAllCheckers returns instances of all available checkers
func GetAllCheckers() []Checker {
	checkers := make([]Checker, 0, len(Registry))
	for _, factory := range Registry {
		checkers = append(checkers, factory())
	}
	return checkers
}

// GetChecker returns a new instance of the specified checker
func GetChecker(name CheckerName) Checker {
	if factory, exists := Registry[name]; exists {
		return factory()
	}
	return nil
}