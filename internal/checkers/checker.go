package checkers

import (
	"golang.org/x/tools/go/analysis"
)

// Checker defines the interface for function call checkers
type Checker interface {
	// Check performs the analysis and reports violations
	Check(pass *analysis.Pass) error
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

// Registry holds all available checkers
var Registry = map[CheckerName]func() Checker{
	HTTPCheckerName: func() Checker { return &HTTPChecker{} },
	NetCheckerName:  func() Checker { return &NetChecker{} },
	ExecCheckerName: func() Checker { return &ExecChecker{} },
	TLSCheckerName:  func() Checker { return &TLSChecker{} },
}

// GetAllCheckers returns instances of all available checkers
func GetAllCheckers() []Checker {
	checkers := make([]Checker, 0, len(Registry))
	for _, factory := range Registry {
		checkers = append(checkers, factory())
	}
	return checkers
}