package config

import (
	"os"
	"testing"
)

func TestValidator_RequireString(t *testing.T) {
	v := NewValidator()

	// Test missing env var
	os.Unsetenv("TEST_REQUIRED")
	val := v.RequireString("TEST_REQUIRED")
	if val != "" {
		t.Errorf("Expected empty string, got %s", val)
	}
	if !v.HasErrors() {
		t.Error("Expected validation error")
	}

	// Test present env var
	os.Setenv("TEST_REQUIRED", "test-value")
	v2 := NewValidator()
	val = v2.RequireString("TEST_REQUIRED")
	if val != "test-value" {
		t.Errorf("Expected 'test-value', got %s", val)
	}
	if v2.HasErrors() {
		t.Error("Expected no validation errors")
	}
	os.Unsetenv("TEST_REQUIRED")
}

func TestValidator_OptionalString(t *testing.T) {
	v := NewValidator()

	// Test missing env var with default
	os.Unsetenv("TEST_OPTIONAL")
	val := v.OptionalString("TEST_OPTIONAL", "default-value")
	if val != "default-value" {
		t.Errorf("Expected 'default-value', got %s", val)
	}

	// Test present env var
	os.Setenv("TEST_OPTIONAL", "actual-value")
	val = v.OptionalString("TEST_OPTIONAL", "default-value")
	if val != "actual-value" {
		t.Errorf("Expected 'actual-value', got %s", val)
	}
	os.Unsetenv("TEST_OPTIONAL")
}

func TestValidator_RequireInt(t *testing.T) {
	v := NewValidator()

	// Test missing env var
	os.Unsetenv("TEST_INT")
	val := v.RequireInt("TEST_INT")
	if val != 0 {
		t.Errorf("Expected 0, got %d", val)
	}
	if !v.HasErrors() {
		t.Error("Expected validation error")
	}

	// Test invalid int
	os.Setenv("TEST_INT", "not-a-number")
	v2 := NewValidator()
	val = v2.RequireInt("TEST_INT")
	if val != 0 {
		t.Errorf("Expected 0, got %d", val)
	}
	if !v2.HasErrors() {
		t.Error("Expected validation error")
	}

	// Test valid int
	os.Setenv("TEST_INT", "42")
	v3 := NewValidator()
	val = v3.RequireInt("TEST_INT")
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
	if v3.HasErrors() {
		t.Error("Expected no validation errors")
	}
	os.Unsetenv("TEST_INT")
}

func TestValidator_RequireURL(t *testing.T) {
	v := NewValidator()

	// Test missing env var
	os.Unsetenv("TEST_URL")
	val := v.RequireURL("TEST_URL")
	if val != "" {
		t.Errorf("Expected empty string, got %s", val)
	}
	if !v.HasErrors() {
		t.Error("Expected validation error")
	}

	// Test invalid URL
	os.Setenv("TEST_URL", "not-a-url")
	v2 := NewValidator()
	val = v2.RequireURL("TEST_URL")
	if val != "" {
		t.Errorf("Expected empty string, got %s", val)
	}
	if !v2.HasErrors() {
		t.Error("Expected validation error")
	}

	// Test valid URL
	os.Setenv("TEST_URL", "https://example.com")
	v3 := NewValidator()
	val = v3.RequireURL("TEST_URL")
	if val != "https://example.com" {
		t.Errorf("Expected 'https://example.com', got %s", val)
	}
	if v3.HasErrors() {
		t.Error("Expected no validation errors")
	}
	os.Unsetenv("TEST_URL")
}

func TestValidator_RequireCSV(t *testing.T) {
	v := NewValidator()

	// Test missing env var
	os.Unsetenv("TEST_CSV")
	val := v.RequireCSV("TEST_CSV")
	if val != nil {
		t.Errorf("Expected nil, got %v", val)
	}
	if !v.HasErrors() {
		t.Error("Expected validation error")
	}

	// Test valid CSV
	os.Setenv("TEST_CSV", "a,b,c")
	v2 := NewValidator()
	val = v2.RequireCSV("TEST_CSV")
	if len(val) != 3 {
		t.Errorf("Expected 3 items, got %d", len(val))
	}
	if v2.HasErrors() {
		t.Error("Expected no validation errors")
	}
	os.Unsetenv("TEST_CSV")
}

func TestValidator_Validate(t *testing.T) {
	v := NewValidator()

	// No errors
	err := v.Validate()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// With errors
	os.Unsetenv("TEST_REQUIRED")
	v.RequireString("TEST_REQUIRED")
	err = v.Validate()
	if err == nil {
		t.Error("Expected validation error")
	}
	os.Unsetenv("TEST_REQUIRED")
}
