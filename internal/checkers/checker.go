package checkers

import (
	"golang.org/x/tools/go/analysis"
)

// Checker defines the interface for function call checkers
type Checker interface {
	// Check performs the analysis and reports violations
	Check(pass *analysis.Pass) error
	
	// Name returns the name of this checker
	Name() string
}

// CheckerFactory creates a new instance of a checker
type CheckerFactory func() Checker

// Registry holds all available checker factories
var Registry = map[string]CheckerFactory{
	"http": func() Checker { return NewHTTPChecker() },
	"net":  func() Checker { return NewNetChecker() },
	"exec": func() Checker { return NewExecChecker() },
	"tls":  func() Checker { return NewTLSChecker() },
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
func GetChecker(name string) Checker {
	if factory, exists := Registry[name]; exists {
		return factory()
	}
	return nil
}