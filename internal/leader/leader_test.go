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

package leader

import (
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions("test-lease")

	if opts.LeaseName != "test-lease" {
		t.Errorf("Expected LeaseName 'test-lease', got '%s'", opts.LeaseName)
	}

	if opts.Enable {
		t.Error("Expected Enable to be false by default")
	}

	if opts.LeaseDuration != 15*time.Second {
		t.Errorf("Expected LeaseDuration 15s, got %v", opts.LeaseDuration)
	}

	if opts.RenewDeadline != 10*time.Second {
		t.Errorf("Expected RenewDeadline 10s, got %v", opts.RenewDeadline)
	}

	if opts.RetryPeriod != 2*time.Second {
		t.Errorf("Expected RetryPeriod 2s, got %v", opts.RetryPeriod)
	}
}

func TestSetup_Disabled(t *testing.T) {
	opts := Options{
		Enable: false,
	}

	managerOpts := ctrl.Options{}
	setupFunc := Setup(opts)
	setupFunc(&managerOpts)

	if managerOpts.LeaderElection {
		t.Error("Expected LeaderElection to be false when disabled")
	}
}

func TestSetup_Enabled(t *testing.T) {
	opts := Options{
		Enable:        true,
		LeaseName:     "test-lease",
		Namespace:     "test-ns",
		LeaseDuration: 20 * time.Second,
		RenewDeadline: 15 * time.Second,
		RetryPeriod:   3 * time.Second,
	}

	managerOpts := ctrl.Options{}
	setupFunc := Setup(opts)
	setupFunc(&managerOpts)

	if !managerOpts.LeaderElection {
		t.Error("Expected LeaderElection to be true when enabled")
	}

	if managerOpts.LeaderElectionID != "test-lease" {
		t.Errorf("Expected LeaderElectionID 'test-lease', got '%s'", managerOpts.LeaderElectionID)
	}

	if managerOpts.LeaderElectionNamespace != "test-ns" {
		t.Errorf("Expected LeaderElectionNamespace 'test-ns', got '%s'", managerOpts.LeaderElectionNamespace)
	}

	if managerOpts.LeaseDuration == nil {
		t.Error("Expected LeaseDuration to be set")
	} else if *managerOpts.LeaseDuration != 20*time.Second {
		t.Errorf("Expected LeaseDuration 20s, got %v", *managerOpts.LeaseDuration)
	}
}

func TestManagerOptions(t *testing.T) {
	baseOpts := ctrl.Options{
		Scheme: nil,
	}

	leaderOpts := Options{
		Enable:    true,
		LeaseName: "test-lease",
	}

	result := ManagerOptions(&baseOpts, leaderOpts)

	if !result.LeaderElection {
		t.Error("Expected LeaderElection to be enabled")
	}

	if result.LeaderElectionID != "test-lease" {
		t.Errorf("Expected LeaderElectionID 'test-lease', got '%s'", result.LeaderElectionID)
	}
}
