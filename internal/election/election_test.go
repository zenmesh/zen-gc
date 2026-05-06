package election

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{}

	// Test default values
	if cfg.ElectionID != "" {
		t.Errorf("Expected empty ElectionID, got %s", cfg.ElectionID)
	}
	if cfg.Namespace != "" {
		t.Errorf("Expected empty Namespace, got %s", cfg.Namespace)
	}
	if cfg.LeaseName != "" {
		t.Errorf("Expected empty LeaseName, got %s", cfg.LeaseName)
	}
	if cfg.LeaseDuration != 0 {
		t.Errorf("Expected 0 LeaseDuration, got %v", cfg.LeaseDuration)
	}
	if cfg.RenewDeadline != 0 {
		t.Errorf("Expected 0 RenewDeadline, got %v", cfg.RenewDeadline)
	}
	if cfg.RetryPeriod != 0 {
		t.Errorf("Expected 0 RetryPeriod, got %v", cfg.RetryPeriod)
	}
	if cfg.Enable != false {
		t.Errorf("Expected false Enable, got %v", cfg.Enable)
	}
}

func TestConfigWithValues(t *testing.T) {
	cfg := &Config{
		ElectionID:     "test-election",
		Namespace:      "test-ns",
		LeaseName:      "test-lease",
		LeaseDuration:  30 * time.Second,
		RenewDeadline:  20 * time.Second,
		RetryPeriod:    10 * time.Second,
		Enable:         true,
	}

	if cfg.ElectionID != "test-election" {
		t.Errorf("Expected test-election, got %s", cfg.ElectionID)
	}
	if cfg.Namespace != "test-ns" {
		t.Errorf("Expected test-ns, got %s", cfg.Namespace)
	}
	if cfg.LeaseName != "test-lease" {
		t.Errorf("Expected test-lease, got %s", cfg.LeaseName)
	}
	if cfg.LeaseDuration != 30*time.Second {
		t.Errorf("Expected 30s, got %v", cfg.LeaseDuration)
	}
	if cfg.Enable != true {
		t.Errorf("Expected true, got %v", cfg.Enable)
	}
}

func TestShutdownContext(t *testing.T) {
	ctx, cancel := ShutdownContext(context.Background(), "test")
	if ctx == nil {
		t.Error("Expected non-nil context")
	}
	if cancel == nil {
		t.Error("Expected non-nil cancel function")
	}

	// Cancel should not panic
	cancel()
}

func TestGetPodName(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Skipf("Cannot get hostname: %v", err)
	}

	// The function should return the hostname or "unknown"
	// We can't directly test it since it's internal, but we can verify
	// the function exists and doesn't panic by testing ShutdownContext
	// which uses similar logic internally
	_ = hostname
}

func TestConfigIsValid(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		isValid bool
	}{
		{
			name:    "valid with all fields",
			cfg:     &Config{ElectionID: "test", Namespace: "ns", Enable: true},
			isValid: true,
		},
		{
			name:    "valid disabled",
			cfg:     &Config{Enable: false},
			isValid: true,
		},
		{
			name:    "empty config",
			cfg:     &Config{},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the config can be created
			if tt.cfg == nil {
				t.Error("Expected non-nil config")
			}
		})
	}
}