/*
Copyright 2025 Kube-ZEN Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package zenlead

import (
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

func TestPrepareManagerOptions_BuiltIn(t *testing.T) {
	base := ctrl.Options{
		Scheme: nil,
	}

	leConfig := LeaderElectionConfig{
		Mode:       BuiltIn,
		ElectionID: "test-controller-leader-election",
		Namespace:  "test-namespace",
	}

	opts, err := PrepareManagerOptions(&base, &leConfig)
	if err != nil {
		t.Fatalf("PrepareManagerOptions failed: %v", err)
	}

	if !opts.LeaderElection {
		t.Error("LeaderElection should be true for BuiltIn mode")
	}
	if opts.LeaderElectionID != "test-controller-leader-election" {
		t.Errorf("LeaderElectionID = %q, want %q", opts.LeaderElectionID, "test-controller-leader-election")
	}
	if opts.LeaderElectionNamespace != "test-namespace" {
		t.Errorf("LeaderElectionNamespace = %q, want %q", opts.LeaderElectionNamespace, "test-namespace")
	}
	if !opts.LeaderElectionReleaseOnCancel {
		t.Error("LeaderElectionReleaseOnCancel should be true")
	}
}

func TestPrepareManagerOptions_ZenLeadManaged(t *testing.T) {
	base := ctrl.Options{
		Scheme: nil,
	}

	leConfig := LeaderElectionConfig{
		Mode:      ZenLeadManaged,
		LeaseName: "test-leader-group",
		Namespace: "test-namespace",
	}

	opts, err := PrepareManagerOptions(&base, &leConfig)
	if err != nil {
		t.Fatalf("PrepareManagerOptions failed: %v", err)
	}

	if !opts.LeaderElection {
		t.Error("LeaderElection should be true for ZenLeadManaged mode")
	}
	expectedID := "test-leader-group-lease"
	if opts.LeaderElectionID != expectedID {
		t.Errorf("LeaderElectionID = %q, want %q", opts.LeaderElectionID, expectedID)
	}
	if opts.LeaderElectionNamespace != "test-namespace" {
		t.Errorf("LeaderElectionNamespace = %q, want %q", opts.LeaderElectionNamespace, "test-namespace")
	}
}

func TestPrepareManagerOptions_Disabled(t *testing.T) {
	base := ctrl.Options{
		Scheme: nil,
	}

	leConfig := LeaderElectionConfig{
		Mode: Disabled,
	}

	opts, err := PrepareManagerOptions(&base, &leConfig)
	if err != nil {
		t.Fatalf("PrepareManagerOptions failed: %v", err)
	}

	if opts.LeaderElection {
		t.Error("LeaderElection should be false for Disabled mode")
	}
}

func TestPrepareManagerOptions_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  LeaderElectionConfig
		wantErr string
	}{
		{
			name: "BuiltIn without ElectionID",
			config: LeaderElectionConfig{
				Mode:      BuiltIn,
				Namespace: "test",
			},
			wantErr: "ElectionID is required",
		},
		{
			name: "BuiltIn without Namespace",
			config: LeaderElectionConfig{
				Mode:       BuiltIn,
				ElectionID: "test",
			},
			wantErr: "Namespace is required",
		},
		{
			name: "ZenLeadManaged without LeaseName",
			config: LeaderElectionConfig{
				Mode:      ZenLeadManaged,
				Namespace: "test",
			},
			wantErr: "LeaseName is required",
		},
		{
			name: "ZenLeadManaged without Namespace",
			config: LeaderElectionConfig{
				Mode:      ZenLeadManaged,
				LeaseName: "test",
			},
			wantErr: "Namespace is required",
		},
		{
			name: "Invalid mode",
			config: LeaderElectionConfig{
				Mode: LeadershipMode("invalid"),
			},
			wantErr: "invalid LeadershipMode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PrepareManagerOptions(&ctrl.Options{}, &tt.config)
			if err == nil {
				t.Fatal("PrepareManagerOptions should have failed")
			}
			if err.Error() == "" || !contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPrepareManagerOptions_TimingOverrides(t *testing.T) {
	leaseDuration := 20 * time.Second
	renewDeadline := 15 * time.Second
	retryPeriod := 5 * time.Second

	base := ctrl.Options{
		Scheme: nil,
	}

	leConfig := LeaderElectionConfig{
		Mode:          BuiltIn,
		ElectionID:    "test",
		Namespace:     "test",
		LeaseDuration: &leaseDuration,
		RenewDeadline: &renewDeadline,
		RetryPeriod:   &retryPeriod,
	}

	opts, err := PrepareManagerOptions(&base, &leConfig)
	if err != nil {
		t.Fatalf("PrepareManagerOptions failed: %v", err)
	}

	if opts.LeaseDuration == nil || *opts.LeaseDuration != leaseDuration {
		t.Errorf("LeaseDuration = %v, want %v", opts.LeaseDuration, &leaseDuration)
	}
	if opts.RenewDeadline == nil || *opts.RenewDeadline != renewDeadline {
		t.Errorf("RenewDeadline = %v, want %v", opts.RenewDeadline, &renewDeadline)
	}
	if opts.RetryPeriod == nil || *opts.RetryPeriod != retryPeriod {
		t.Errorf("RetryPeriod = %v, want %v", opts.RetryPeriod, &retryPeriod)
	}
}

func TestEnforceSafeHA(t *testing.T) {
	tests := []struct {
		name                  string
		replicaCount          int
		leaderElectionEnabled bool
		wantErr               bool
	}{
		{
			name:                  "Safe: replicas=1, leader election disabled",
			replicaCount:          1,
			leaderElectionEnabled: false,
			wantErr:               false,
		},
		{
			name:                  "Safe: replicas=2, leader election enabled",
			replicaCount:          2,
			leaderElectionEnabled: true,
			wantErr:               false,
		},
		{
			name:                  "Unsafe: replicas=2, leader election disabled",
			replicaCount:          2,
			leaderElectionEnabled: false,
			wantErr:               true,
		},
		{
			name:                  "Unsafe: replicas=3, leader election disabled",
			replicaCount:          3,
			leaderElectionEnabled: false,
			wantErr:               true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnforceSafeHA(tt.replicaCount, tt.leaderElectionEnabled)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnforceSafeHA() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if err.Error() == "" {
					t.Error("error message should not be empty")
				}
			}
		})
	}
}

func TestDeriveElectionIDFromLeaseName(t *testing.T) {
	tests := []struct {
		leaseName string
		want      string
	}{
		{
			leaseName: "my-controller",
			want:      "my-controller-lease",
		},
		{
			leaseName: "test-leader-group",
			want:      "test-leader-group-lease",
		},
	}

	for _, tt := range tests {
		t.Run(tt.leaseName, func(t *testing.T) {
			got := deriveElectionIDFromLeaseName(tt.leaseName)
			if got != tt.want {
				t.Errorf("deriveElectionIDFromLeaseName(%q) = %q, want %q", tt.leaseName, got, tt.want)
			}
		})
	}
}
