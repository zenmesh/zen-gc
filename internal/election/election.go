/*
Copyright 2026 Zen-Mesh Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

package election provides simple leader election using client-go.
*/

package election

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

type Config struct {
	ElectionID  string
	Namespace   string
	LeaseName   string
	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration
	Enable        bool
}

func RunWithLeaderElection(ctx context.Context, cfg *Config, client kubernetes.Interface, runFn func(context.Context)) error {
	if !cfg.Enable {
		klog.InfoS("Leader election disabled, running directly")
		runFn(ctx)
		return nil
	}

	if cfg.Namespace == "" {
		cfg.Namespace = "default"
	}
	if cfg.ElectionID == "" {
		cfg.ElectionID = "zen-gc-leader-election"
	}
	if cfg.LeaseName == "" {
		cfg.LeaseName = cfg.ElectionID
	}
	if cfg.LeaseDuration == 0 {
		cfg.LeaseDuration = 15 * 1000000000 // 15 seconds
	}
	if cfg.RenewDeadline == 0 {
		cfg.RenewDeadline = 10 * 1000000000 // 10 seconds
	}
	if cfg.RetryPeriod == 0 {
		cfg.RetryPeriod = 5 * 1000000000 // 5 seconds
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.LeaseName,
			Namespace: cfg.Namespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: getPodName(),
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   cfg.LeaseDuration,
		RenewDeadline:   cfg.RenewDeadline,
		RetryPeriod:     cfg.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.InfoS("Started leading", "electionID", cfg.ElectionID)
				runFn(ctx)
			},
			OnStoppedLeading: func() {
				klog.InfoS("Stopped leading, shutting down")
			},
			OnNewLeader: func(identity string) {
				if identity != getPodName() {
					klog.InfoS("New leader elected", "leader", identity)
				}
			},
		},
		WatchDog: leaderelection.NewLeaderHealthzAdaptor(0),
		// Name is required but mainly for logging purposes
		Name: cfg.ElectionID,
	})

	return nil
}

func getPodName() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "unknown"
}

func ShutdownContext(ctx context.Context, name string) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
}