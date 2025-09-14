package checkers_test

import (
	"testing"

	"github.com/sonatard/noctx/internal/checkers"
)

func TestGetAllCheckers(t *testing.T) {
	allCheckers := checkers.GetAllCheckers()
	
	expectedCount := 1 // Unified checker
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
		"unified",
	}
	
	for _, expectedName := range expectedNames {
		if !names[expectedName] {
			t.Errorf("Expected checker name %s not found", expectedName)
		}
	}
}

func TestGetChecker(t *testing.T) {
	// Test getting existing checker
	unifiedChecker := checkers.GetChecker("unified")
	if unifiedChecker == nil {
		t.Error("Expected UnifiedChecker, got nil")
	}
	if unifiedChecker.Name() != "unified" {
		t.Errorf("Expected name %s, got %s", "unified", unifiedChecker.Name())
	}
	
	// Test getting non-existent checker
	invalidChecker := checkers.GetChecker("invalid")
	if invalidChecker != nil {
		t.Error("Expected nil for invalid checker name")
	}
}

func TestCheckerFactories(t *testing.T) {
	// Test that all factories create different instances
	checker1 := checkers.GetChecker("unified")
	checker2 := checkers.GetChecker("unified")
	
	if checker1 == checker2 {
		t.Error("Factory should create new instances, not return the same one")
	}
}

func TestNewCheckerFunctions(t *testing.T) {
	// Test the unified checker function
	unifiedChecker := checkers.NewUnifiedChecker()
	if unifiedChecker == nil {
		t.Error("NewUnifiedChecker returned nil")
	}
	if unifiedChecker.Name() != "unified" {
		t.Errorf("UnifiedChecker name mismatch")
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