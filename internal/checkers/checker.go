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
	// HTTP function checkers
	HTTPNewRequestChecker CheckerName = "http-new-request"
	HTTPGetChecker        CheckerName = "http-get"
	HTTPHeadChecker       CheckerName = "http-head"
	HTTPPostChecker       CheckerName = "http-post"
	
	// Network function checkers
	NetDialChecker CheckerName = "net-dial"
)

// Registry holds all available checkers
var Registry = map[CheckerName]func() Checker{
	HTTPNewRequestChecker: func() Checker { return &HTTPNewRequest{} },
	HTTPGetChecker:        func() Checker { return &HTTPGet{} },
	HTTPHeadChecker:       func() Checker { return &HTTPHead{} },
	HTTPPostChecker:       func() Checker { return &HTTPPost{} },
	NetDialChecker:        func() Checker { return &NetDial{} },
}

// GetAllCheckers returns instances of all available checkers
func GetAllCheckers() []Checker {
	checkers := make([]Checker, 0, len(Registry))
	for _, factory := range Registry {
		checkers = append(checkers, factory())
	}
	return checkers
}