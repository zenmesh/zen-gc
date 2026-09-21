/*
Copyright 2026 Zen Mesh

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

package controller

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/zenmesh/zen-gc/internal/logging"
	"github.com/zenmesh/zen-gc/pkg/api/v1alpha1"
	"github.com/zenmesh/zen-gc/pkg/config"
)

// SUPPORT2-032R: standby replicas must be probe-healthy (readiness never
// claims reconciliation; exactly-one is enforced by the lease), while the
// leader keeps the full informer-sync law.
func TestHealthChecker_LeadershipAwareProbes(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	fakeClient := clientfake.NewClientBuilder().WithScheme(scheme).Build()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	reconciler := NewGCPolicyReconcilerWithRESTMapper(
		fakeClient,
		scheme,
		dynamicClient,
		nil,
		NewStatusUpdaterWithConfig(dynamicClient, config.NewControllerConfig()),
		NewEventRecorder(nil),
		config.NewControllerConfig(),
	)

	h := NewHealthChecker(reconciler)
	h.SetMaxTimeSinceLastEvaluation(time.Minute)
	h.UpdateLastEvaluationTime()

	leader := NewLeaderState(15*time.Second, logging.NewLogger("test"))
	h.SetLeaderState(leader)

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)

	// Before any election result the process is a warm standby: healthy.
	if err := h.ReadinessCheck(req); err != nil {
		t.Errorf("pre-election ReadinessCheck: %v", err)
	}
	if err := h.LivenessCheck(req); err != nil {
		t.Errorf("pre-election LivenessCheck: %v", err)
	}

	// Leader: full informer-sync law applies (no informers -> synced/ready).
	leader.SetLeading(true)
	if err := h.ReadinessCheck(req); err != nil {
		t.Errorf("leader ReadinessCheck: %v", err)
	}
	if err := h.LivenessCheck(req); err != nil {
		t.Errorf("leader LivenessCheck: %v", err)
	}

	// Demoted: process-level health again.
	leader.SetLeading(false)
	if err := h.ReadinessCheck(req); err != nil {
		t.Errorf("standby ReadinessCheck: %v", err)
	}
	if err := h.LivenessCheck(req); err != nil {
		t.Errorf("standby LivenessCheck: %v", err)
	}
}

func TestLeaderzCheck(t *testing.T) {
	leader := NewLeaderState(time.Second, logging.NewLogger("test"))
	handler := LeaderzCheck(leader)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/leaderz", http.NoBody))
	if got := rec.Body.String(); got != "standby" {
		t.Errorf("leaderz before election = %q, want standby", got)
	}

	leader.SetLeading(true)
	rec = httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/leaderz", http.NoBody))
	if got := rec.Body.String(); got != "leader" {
		t.Errorf("leaderz while leading = %q, want leader", got)
	}
}

func TestServeStandaloneServesAllReplicas(t *testing.T) {
	// Reserve a free port, release it, and bind the standalone server there.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	stop, err := ServeStandalone(t.Context(), addr, mux)
	if err != nil {
		t.Fatalf("ServeStandalone: %v", err)
	}
	defer stop()

	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get("http://" + addr + "/readyz")
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("/readyz status = %d, want 200", resp.StatusCode)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("standalone server never served /readyz: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
