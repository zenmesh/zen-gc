package config

import (
	"os"
	"testing"
	"time"
)

func TestRequireEnv(t *testing.T) {
	os.Unsetenv("TEST_VAR")
	_, err := RequireEnv("TEST_VAR")
	if err == nil {
		t.Error("Expected error for missing env var")
	}

	os.Setenv("TEST_VAR", "value")
	val, err := RequireEnv("TEST_VAR")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != "value" {
		t.Errorf("Expected 'value', got '%s'", val)
	}
	os.Unsetenv("TEST_VAR")
}

func TestRequireEnvWithDefault(t *testing.T) {
	os.Unsetenv("TEST_DEFAULT")
	val := RequireEnvWithDefault("TEST_DEFAULT", "default")
	if val != "default" {
		t.Errorf("Expected 'default', got '%s'", val)
	}

	os.Setenv("TEST_DEFAULT", "actual")
	val = RequireEnvWithDefault("TEST_DEFAULT", "default")
	if val != "actual" {
		t.Errorf("Expected 'actual', got '%s'", val)
	}
	os.Unsetenv("TEST_DEFAULT")
}

func TestRequireEnvInt(t *testing.T) {
	os.Unsetenv("TEST_INT")
	_, err := RequireEnvInt("TEST_INT")
	if err == nil {
		t.Error("Expected error for missing env var")
	}

	os.Setenv("TEST_INT", "42")
	val, err := RequireEnvInt("TEST_INT")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
	os.Unsetenv("TEST_INT")

	os.Setenv("TEST_INT", "invalid")
	_, err = RequireEnvInt("TEST_INT")
	if err == nil {
		t.Error("Expected error for invalid int")
	}
	os.Unsetenv("TEST_INT")
}

func TestRequireEnvIntWithDefault(t *testing.T) {
	os.Unsetenv("TEST_INT_DFT")
	val := RequireEnvIntWithDefault("TEST_INT_DFT", 100)
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	os.Setenv("TEST_INT_DFT", "50")
	val = RequireEnvIntWithDefault("TEST_INT_DFT", 100)
	if val != 50 {
		t.Errorf("Expected 50, got %d", val)
	}
	os.Unsetenv("TEST_INT_DFT")
}

func TestRequireEnvBool(t *testing.T) {
	os.Unsetenv("TEST_BOOL")
	_, err := RequireEnvBool("TEST_BOOL")
	if err == nil {
		t.Error("Expected error for missing env var")
	}

	os.Setenv("TEST_BOOL", "true")
	val, err := RequireEnvBool("TEST_BOOL")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != true {
		t.Errorf("Expected true, got %v", val)
	}
	os.Unsetenv("TEST_BOOL")

	os.Setenv("TEST_BOOL", "false")
	val, err = RequireEnvBool("TEST_BOOL")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != false {
		t.Errorf("Expected false, got %v", val)
	}
	os.Unsetenv("TEST_BOOL")
}

func TestRequireEnvBoolWithDefault(t *testing.T) {
	os.Unsetenv("TEST_BOOL_DFT")
	val := RequireEnvBoolWithDefault("TEST_BOOL_DFT", true)
	if val != true {
		t.Errorf("Expected true, got %v", val)
	}

	os.Setenv("TEST_BOOL_DFT", "false")
	val = RequireEnvBoolWithDefault("TEST_BOOL_DFT", true)
	if val != false {
		t.Errorf("Expected false, got %v", val)
	}
	os.Unsetenv("TEST_BOOL_DFT")
}

func TestRequireEnvOneOf(t *testing.T) {
	os.Unsetenv("TEST_ONEOF")
	allowed := []string{"a", "b", "c"}
	_, err := RequireEnvOneOf("TEST_ONEOF", allowed)
	if err == nil {
		t.Error("Expected error for missing env var")
	}

	os.Setenv("TEST_ONEOF", "invalid")
	_, err = RequireEnvOneOf("TEST_ONEOF", allowed)
	if err == nil {
		t.Error("Expected error for invalid value")
	}
	os.Unsetenv("TEST_ONEOF")

	os.Setenv("TEST_ONEOF", "a")
	val, err := RequireEnvOneOf("TEST_ONEOF", allowed)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != "a" {
		t.Errorf("Expected 'a', got '%s'", val)
	}
	os.Unsetenv("TEST_ONEOF")
}

func TestRequireEnvSecret(t *testing.T) {
	os.Unsetenv("TEST_SECRET")
	_, err := RequireEnvSecret("TEST_SECRET", 8)
	if err == nil {
		t.Error("Expected error for missing env var")
	}

	os.Setenv("TEST_SECRET", "short")
	_, err = RequireEnvSecret("TEST_SECRET", 8)
	if err == nil {
		t.Error("Expected error for too short secret")
	}
	os.Unsetenv("TEST_SECRET")

	os.Setenv("TEST_SECRET", "long-enough-secret")
	val, err := RequireEnvSecret("TEST_SECRET", 8)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != "long-enough-secret" {
		t.Errorf("Unexpected value: %s", val)
	}
	os.Unsetenv("TEST_SECRET")
}

func TestRequireAtLeastOne(t *testing.T) {
	os.Unsetenv("TEST_KEY1")
	os.Unsetenv("TEST_KEY2")

	err := RequireAtLeastOne([]string{"TEST_KEY1", "TEST_KEY2"})
	if err == nil {
		t.Error("Expected error when all keys missing")
	}

	os.Setenv("TEST_KEY1", "value")
	err = RequireAtLeastOne([]string{"TEST_KEY1", "TEST_KEY2"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	os.Unsetenv("TEST_KEY1")
	os.Unsetenv("TEST_KEY2")
}

func TestValidateProduction(t *testing.T) {
	// This test just ensures the function doesn't panic
	// In production environment (when KUBERNETES_SERVICE_HOST is set), it should pass
	_ = ValidateProduction()
}

func TestRequireEnvDuration(t *testing.T) {
	os.Unsetenv("TEST_DURATION")
	_, err := RequireEnvDuration("TEST_DURATION")
	if err == nil {
		t.Error("Expected error for missing env var")
	}

	os.Setenv("TEST_DURATION", "30s")
	val, err := RequireEnvDuration("TEST_DURATION")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != 30*time.Second {
		t.Errorf("Expected 30s, got %v", val)
	}
	os.Unsetenv("TEST_DURATION")

	os.Setenv("TEST_DURATION", "invalid")
	_, err = RequireEnvDuration("TEST_DURATION")
	if err == nil {
		t.Error("Expected error for invalid duration")
	}
	os.Unsetenv("TEST_DURATION")
}

func TestRequireEnvDurationWithDefault(t *testing.T) {
	os.Unsetenv("TEST_DURATION_DFT")
	val := RequireEnvDurationWithDefault("TEST_DURATION_DFT", 60*time.Second)
	if val != 60*time.Second {
		t.Errorf("Expected 60s, got %v", val)
	}

	os.Setenv("TEST_DURATION_DFT", "30s")
	val = RequireEnvDurationWithDefault("TEST_DURATION_DFT", 60*time.Second)
	if val != 30*time.Second {
		t.Errorf("Expected 30s, got %v", val)
	}
	os.Unsetenv("TEST_DURATION_DFT")
}

func TestRequireEnvCSV(t *testing.T) {
	os.Unsetenv("TEST_CSV")
	_, err := RequireEnvCSV("TEST_CSV")
	if err == nil {
		t.Error("Expected error for missing env var")
	}

	os.Setenv("TEST_CSV", "a,b,c")
	val, err := RequireEnvCSV("TEST_CSV")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(val) != 3 || val[0] != "a" || val[1] != "b" || val[2] != "c" {
		t.Errorf("Unexpected value: %v", val)
	}
	os.Unsetenv("TEST_CSV")
}

func TestRequireEnvCSVWithDefault(t *testing.T) {
	os.Unsetenv("TEST_CSV_DFT")
	val := RequireEnvCSVWithDefault("TEST_CSV_DFT", []string{"x", "y"})
	if len(val) != 2 || val[0] != "x" || val[1] != "y" {
		t.Errorf("Expected default, got %v", val)
	}

	os.Setenv("TEST_CSV_DFT", "a,b")
	val = RequireEnvCSVWithDefault("TEST_CSV_DFT", []string{"x", "y"})
	if len(val) != 2 || val[0] != "a" || val[1] != "b" {
		t.Errorf("Expected a,b, got %v", val)
	}
	os.Unsetenv("TEST_CSV_DFT")
}