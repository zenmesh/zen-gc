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
	"fmt"
	"time"

	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

// LeadershipMode defines how leader election is configured.
type LeadershipMode string

const (
	// BuiltIn uses Kubernetes Lease API directly (controller-runtime).
	// This is the default and recommended mode for controller HA.
	BuiltIn LeadershipMode = "builtin"

	// ZenLeadManaged uses a Lease provisioned and owned by zen-lead (via LeaderGroup CRD).
	// Requires zen-lead to be deployed with CRD module enabled.
	ZenLeadManaged LeadershipMode = "zenlead"

	// Disabled disables leader election (only allowed when replicas=1).
	// Use with extreme caution - unsafe if replicas > 1.
	Disabled LeadershipMode = "disabled"
)

// LeaderElectionConfig configures leader election for a component.
type LeaderElectionConfig struct {
	// Mode determines how leader election is configured.
	// Default: BuiltIn
	Mode LeadershipMode

	// ElectionID is the name of the Lease resource used for leader election.
	// Required for BuiltIn and ZenLeadManaged modes.
	// Format: "<component-name>-leader-election"
	ElectionID string

	// Namespace is the namespace where the Lease resource is created.
	// Required for BuiltIn and ZenLeadManaged modes.
	// Typically obtained via RequirePodNamespace().
	Namespace string

	// LeaseName is the name of the LeaderGroup CRD (only for ZenLeadManaged mode).
	// Required when Mode == ZenLeadManaged.
	// zen-lead will create a Lease with deterministic name derived from this.
	LeaseName string

	// LeaseDuration is how long a leader holds the lease before it expires.
	// Optional: uses controller-runtime defaults if not set.
	LeaseDuration *time.Duration

	// RenewDeadline is the time to renew the lease before losing leadership.
	// Optional: uses controller-runtime defaults if not set.
	RenewDeadline *time.Duration

	// RetryPeriod is how often to retry acquiring leadership.
	// Optional: uses controller-runtime defaults if not set.
	RetryPeriod *time.Duration
}

// ControllerRuntimeDefaults applies recommended REST client defaults for controller-runtime.
// This ensures consistent QPS/Burst settings, user-agent, and timeouts across all components.
func ControllerRuntimeDefaults(cfg *rest.Config) {
	if cfg.QPS == 0 {
		cfg.QPS = 50 // Default is 20, increase for faster reconciliation
	}
	if cfg.Burst == 0 {
		cfg.Burst = 100 // Default is 30, increase for burst handling
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "zen-sdk/zenlead"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
}

// PrepareManagerOptions configures controller-runtime Manager options with leader election.
// This is the single function all components MUST use for leader election configuration.
//
// Parameters:
//   - base: Base ctrl.Options to extend
//   - le: Leader election configuration
//
// Returns:
//   - Configured ctrl.Options with leader election settings
//   - error if configuration is invalid
func PrepareManagerOptions(base *ctrl.Options, le *LeaderElectionConfig) (ctrl.Options, error) {
	opts := *base

	// Validate configuration
	if err := validateConfig(le); err != nil {
		return opts, fmt.Errorf("invalid leader election config: %w", err)
	}

	// Configure based on mode
	switch le.Mode {
	case BuiltIn:
		opts.LeaderElection = true
		opts.LeaderElectionID = le.ElectionID
		opts.LeaderElectionNamespace = le.Namespace
		opts.LeaderElectionReleaseOnCancel = true

	case ZenLeadManaged:
		// For zen-lead managed mode, derive ElectionID from LeaseName deterministically
		// This ensures consistency: zen-lead creates Lease with this name
		opts.LeaderElection = true
		opts.LeaderElectionID = deriveElectionIDFromLeaseName(le.LeaseName)
		opts.LeaderElectionNamespace = le.Namespace
		opts.LeaderElectionReleaseOnCancel = true

	case Disabled:
		opts.LeaderElection = false
		// No other settings needed
	}

	// Apply optional timing overrides
	if le.LeaseDuration != nil {
		opts.LeaseDuration = le.LeaseDuration
	}
	if le.RenewDeadline != nil {
		opts.RenewDeadline = le.RenewDeadline
	}
	if le.RetryPeriod != nil {
		opts.RetryPeriod = le.RetryPeriod
	}

	return opts, nil
}

// EnforceSafeHA validates that HA configuration is safe.
// This MUST be called at component startup to prevent unsafe configurations.
//
// Parameters:
//   - replicaCount: Number of component replicas
//   - leaderElectionEnabled: Whether leader election is enabled
//
// Returns:
//   - error if configuration is unsafe (replicas > 1 without leader election)
func EnforceSafeHA(replicaCount int, leaderElectionEnabled bool) error {
	if replicaCount > 1 && !leaderElectionEnabled {
		return fmt.Errorf(
			"unsafe HA configuration: replicas=%d but leader election is disabled. "+
				"This will cause split-brain scenarios. "+
				"Either set replicas=1 or enable leader election (mode: builtin or zenlead)",
			replicaCount,
		)
	}
	return nil
}

// validateConfig validates the leader election configuration.
func validateConfig(le *LeaderElectionConfig) error {
	switch le.Mode {
	case BuiltIn:
		if le.ElectionID == "" {
			return fmt.Errorf("ElectionID is required for BuiltIn mode")
		}
		if le.Namespace == "" {
			return fmt.Errorf("Namespace is required for BuiltIn mode")
		}

	case ZenLeadManaged:
		if le.LeaseName == "" {
			return fmt.Errorf("LeaseName is required for ZenLeadManaged mode")
		}
		if le.Namespace == "" {
			return fmt.Errorf("Namespace is required for ZenLeadManaged mode")
		}

	case Disabled:
		// No validation needed for disabled mode

	default:
		return fmt.Errorf("invalid LeadershipMode: %q (must be builtin, zenlead, or disabled)", le.Mode)
	}

	return nil
}

// deriveElectionIDFromLeaseName derives a deterministic ElectionID from a LeaseName.
// This ensures zen-lead and components use the same Lease name.
// Format: "<lease-name>-lease"
func deriveElectionIDFromLeaseName(leaseName string) string {
	return fmt.Sprintf("%s-lease", leaseName)
}
