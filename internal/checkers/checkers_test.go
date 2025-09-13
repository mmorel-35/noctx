package checkers_test

import (
	"testing"

	"github.com/sonatard/noctx/internal/checkers"
)

func TestGetAllCheckers(t *testing.T) {
	allCheckers := checkers.GetAllCheckers()
	
	expectedCount := 4 // HTTP, Net, Exec, TLS
	if len(allCheckers) != expectedCount {
		t.Errorf("Expected %d checkers, got %d", expectedCount, len(allCheckers))
	}
	
	// Verify each checker has a unique name
	names := make(map[checkers.CheckerName]bool)
	for _, checker := range allCheckers {
		name := checker.Name()
		if names[name] {
			t.Errorf("Duplicate checker name: %s", name)
		}
		names[name] = true
	}
	
	// Verify expected checker names exist
	expectedNames := []checkers.CheckerName{
		checkers.HTTPCheckerName,
		checkers.NetCheckerName,
		checkers.ExecCheckerName,
		checkers.TLSCheckerName,
	}
	
	for _, expectedName := range expectedNames {
		if !names[expectedName] {
			t.Errorf("Expected checker name %s not found", expectedName)
		}
	}
}

func TestGetChecker(t *testing.T) {
	// Test getting existing checker
	httpChecker := checkers.GetChecker(checkers.HTTPCheckerName)
	if httpChecker == nil {
		t.Error("Expected HTTPChecker, got nil")
	}
	if httpChecker.Name() != checkers.HTTPCheckerName {
		t.Errorf("Expected name %s, got %s", checkers.HTTPCheckerName, httpChecker.Name())
	}
	
	// Test getting non-existent checker
	invalidChecker := checkers.GetChecker("invalid")
	if invalidChecker != nil {
		t.Error("Expected nil for invalid checker name")
	}
}

func TestCheckerFactories(t *testing.T) {
	// Test that all factories create different instances
	checker1 := checkers.GetChecker(checkers.HTTPCheckerName)
	checker2 := checkers.GetChecker(checkers.HTTPCheckerName)
	
	if checker1 == checker2 {
		t.Error("Factory should create new instances, not return the same one")
	}
}

func TestNewCheckerFunctions(t *testing.T) {
	// Test each individual New function
	httpChecker := checkers.NewHTTPChecker()
	if httpChecker == nil {
		t.Error("NewHTTPChecker returned nil")
	}
	if httpChecker.Name() != checkers.HTTPCheckerName {
		t.Errorf("HTTPChecker name mismatch")
	}
	
	netChecker := checkers.NewNetChecker()
	if netChecker == nil {
		t.Error("NewNetChecker returned nil")
	}
	if netChecker.Name() != checkers.NetCheckerName {
		t.Errorf("NetChecker name mismatch")
	}
	
	execChecker := checkers.NewExecChecker()
	if execChecker == nil {
		t.Error("NewExecChecker returned nil")
	}
	if execChecker.Name() != checkers.ExecCheckerName {
		t.Errorf("ExecChecker name mismatch")
	}
	
	tlsChecker := checkers.NewTLSChecker()
	if tlsChecker == nil {
		t.Error("NewTLSChecker returned nil")
	}
	if tlsChecker.Name() != checkers.TLSCheckerName {
		t.Errorf("TLSChecker name mismatch")
	}
}

func TestRegistryConsistency(t *testing.T) {
	// Test that registry contains all expected checker names
	for name := range checkers.Registry {
		factory := checkers.Registry[name]
		if factory == nil {
			t.Errorf("Registry entry %s has nil factory", name)
			continue
		}
		
		instance := factory()
		if instance == nil {
			t.Errorf("Factory for %s returned nil", name)
			continue
		}
		
		if instance.Name() != name {
			t.Errorf("Checker %s reports name %s", name, instance.Name())
		}
	}
}