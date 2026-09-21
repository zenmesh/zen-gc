/*
Copyright 2025 Zen Mesh

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

// Package main implements the GC controller command-line application.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/zenmesh/zen-gc/internal/election"
	sdklog "github.com/zenmesh/zen-gc/internal/logging"
	"github.com/zenmesh/zen-gc/pkg/api/v1alpha1"
	"github.com/zenmesh/zen-gc/pkg/config"
	"github.com/zenmesh/zen-gc/pkg/controller"
	gcwebhook "github.com/zenmesh/zen-gc/pkg/webhook"
)

// ErrWebhookTLSCertificatesMissing indicates that webhook TLS certificates are missing.
var ErrWebhookTLSCertificatesMissing = errors.New("webhook TLS certificates not found")

const (
	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second

	// DefaultBatchSize is the default batch size for deletions.
	DefaultBatchSize = 50

	// DefaultMaxConcurrentEvaluations is the default maximum number of concurrent policy evaluations.
	DefaultMaxConcurrentEvaluations = 5
)

var (
	// Version information (set via build flags).
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
	logger    *sdklog.Logger
	setupLog  *sdklog.Logger
)

var (
	metricsAddr              = flag.String("metrics-addr", ":8080", "The address the metric endpoint binds to")
	healthProbeAddr          = flag.String("health-probe-addr", ":8081", "The address the standalone health server binds to (serves /healthz, /readyz, /startup, /leaderz on leader AND standby)")
	webhookAddr              = flag.String("webhook-addr", ":9443", "The address the webhook endpoint binds to")
	webhookCertFile          = flag.String("webhook-cert-file", "/etc/webhook/certs/tls.crt", "Path to TLS certificate file")
	webhookKeyFile           = flag.String("webhook-key-file", "/etc/webhook/certs/tls.key", "Path to TLS private key file")
	leaderElection           = flag.Bool("leader-election", true, "Enable leader election (recommended for multi-replica)")
	leaderElectionID         = flag.String("leader-election-id", "gc-controller-leader-election", "The ID for leader election")
	leaderElectionNamespace  = flag.String("leader-election-namespace", "default", "The namespace for leader election lock")
	enableWebhook            = flag.Bool("enable-webhook", true, "Enable validating webhook server")
	insecureWebhook          = flag.Bool("insecure-webhook", false, "Allow webhook to start without TLS (testing only)")
	gcInterval               = flag.Duration("gc-interval", 1*time.Minute, "Interval between GC evaluation runs")
	maxDeletionsPerSecond    = flag.Int("max-deletions-per-second", 10, "Default maximum deletions per second")
	batchSize                = flag.Int("batch-size", DefaultBatchSize, "Default batch size for deletions")
	maxConcurrentEvaluations = flag.Int("max-concurrent-evaluations", DefaultMaxConcurrentEvaluations, "Maximum number of policies to evaluate concurrently")
)

func main() {
	flag.Parse()
	os.Exit(runMain())
}

func runMain() int {
	// Initialize the zen-gc logger (configures the controller-runtime logger automatically)
	logger = sdklog.NewLogger("zen-gc")
	setupLog = logger.WithComponent("setup")
	setupLog.Debug("GC Controller starting", sdklog.String("version", version), sdklog.String("commit", commit), sdklog.String("buildDate", buildDate))

	// OpenTelemetry tracing initialization can be added here when a tracing backend is configured
	// For now, continue without tracing

	// Get config using controller-runtime (handles kubeconfig flag automatically)
	restCfg := ctrl.GetConfigOrDie()

	// Create dynamic client (still needed for resource informers)
	dynamicClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		setupLog.Error(err, "Error building dynamic client", sdklog.ErrorCode("CLIENT_ERROR"))
		return 1
	}

	// Create Kubernetes client for events
	kubeClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		setupLog.Error(err, "Error building Kubernetes client", sdklog.ErrorCode("CLIENT_ERROR"))
		return 1
	}

	// Create scheme and add GarbageCollectionPolicy types
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "Error adding scheme", sdklog.ErrorCode("SCHEME_ERROR"))
		return 1
	}

	// Load controller configuration
	controllerConfig := config.NewControllerConfig()
	if err := controllerConfig.LoadFromEnv(); err != nil {
		setupLog.Error(err, "Error loading configuration from environment", sdklog.ErrorCode("CONFIG_LOAD_ERROR"))
		return 1
	}
	controllerConfig.WithGCInterval(*gcInterval)
	controllerConfig.WithMaxDeletionsPerSecond(*maxDeletionsPerSecond)
	controllerConfig.WithBatchSize(*batchSize)
	controllerConfig.WithMaxConcurrentEvaluations(*maxConcurrentEvaluations)

	setupLog.Info("Controller configuration",
		sdklog.String("gcInterval", controllerConfig.GCInterval.String()),
		sdklog.Int("maxDeletionsPerSecond", controllerConfig.MaxDeletionsPerSecond),
		sdklog.Int("batchSize", controllerConfig.BatchSize),
		sdklog.Int("maxConcurrentEvaluations", controllerConfig.MaxConcurrentEvaluations))

	// Create status updater with configuration
	statusUpdater := controller.NewStatusUpdaterWithConfig(dynamicClient, controllerConfig)

	// Create event recorder
	eventRecorder := controller.NewEventRecorder(kubeClient)

	// Set up graceful shutdown context
	ctx, cancel := election.ShutdownContext(context.Background(), "zen-gc")
	defer cancel()

	// SUPPORT2-032R: the reconciler and health checker are created for ALL
	// replicas (leader + standby), and a standalone always-on health server
	// serves /healthz, /readyz, /startup and /leaderz for every replica.
	// Readiness reflects "healthy leader OR healthy standby" — never lease
	// ownership — while exactly-one reconciliation remains enforced by
	// client-go leader election.
	crClient, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "Error building controller-runtime client", sdklog.ErrorCode("CLIENT_ERROR"))
		return 1
	}
	restMapper, err := apiutil.NewDynamicRESTMapper(restCfg, http.DefaultClient)
	if err != nil {
		setupLog.Error(err, "Error building REST mapper", sdklog.ErrorCode("CLIENT_ERROR"))
		return 1
	}
	leaderState := controller.NewLeaderState(15*time.Second, logger)
	reconciler := controller.NewGCPolicyReconcilerWithRESTMapper(
		crClient,
		scheme,
		dynamicClient,
		restMapper,
		statusUpdater,
		eventRecorder,
		controllerConfig,
	)
	healthChecker := controller.NewHealthChecker(reconciler)
	healthChecker.SetLeaderState(leaderState)

	// Standalone always-on health server (leader + standby).
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := healthChecker.LivenessCheck(r); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	healthMux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := healthChecker.ReadinessCheck(r); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	healthMux.HandleFunc("/startup", func(w http.ResponseWriter, r *http.Request) {
		if err := healthChecker.StartupCheck(r); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	healthMux.Handle("/leaderz", controller.LeaderzCheck(leaderState))
	stopHealth, err := controller.ServeStandalone(ctx, *healthProbeAddr, healthMux)
	if err != nil {
		setupLog.Error(err, "Error starting standalone health server", sdklog.ErrorCode("HEALTH_SERVER_ERROR"))
		return 1
	}
	defer stopHealth()

	// Setup controller-runtime manager
	baseOpts := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: *metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    9443,
			CertDir: "", // We'll handle webhook separately for now
		}),
		// SUPPORT2-032R: health endpoints are served by the standalone
		// always-on server for ALL replicas (leader + standby). The manager
		// does not bind its own probe port (avoids double-bind).
		HealthProbeBindAddress: "0",
	}

	// Configure manager options (no leader election - we use client-go leader election)
	mgrOpts := baseOpts

	// Run with leader election using client-go
	leConfig := &election.Config{
		ElectionID: *leaderElectionID,
		Namespace:  *leaderElectionNamespace,
		Enable:     *leaderElection,
	}

	if *leaderElection {
		setupLog.Info("Leader election enabled",
			sdklog.String("electionID", *leaderElectionID),
			sdklog.String("namespace", *leaderElectionNamespace))
	} else {
		setupLog.Warn("Leader election disabled - only safe for single replica")
	}

	// Leadership transitions feed the health semantics (leader readiness is
	// the full law; standby readiness is process-level, SUPPORT2-032R).
	leConfig.OnLeadingChange = leaderState.SetLeading

	// SUPPORT2-032R: admission is stateless and the webhook Service routes
	// to every ready replica, so every replica serves the webhook — not
	// only the leader (a leader-only webhook 502s whenever the apiserver's
	// dial lands on the standby under failurePolicy=Fail).
	startWebhookServer(ctx)

	err = election.RunWithLeaderElection(ctx, leConfig, kubeClient, func(runCtx context.Context) {
		runController(runCtx, restCfg, &mgrOpts, reconciler, healthChecker, scheme, dynamicClient, statusUpdater, eventRecorder, controllerConfig)
	})
	if err != nil {
		setupLog.Error(err, "Leader election failed", sdklog.ErrorCode("LEADER_ELECTION_ERROR"))
		return 1
	}
	return 0
}

// startWebhookServer starts the admission webhook server on this replica
// (invoked for ALL replicas, before leader election). TLS is required
// unless --insecure-webhook is explicitly set (testing only).
func startWebhookServer(ctx context.Context) {
	if !*enableWebhook {
		return
	}
	webhookServer, err := gcwebhook.NewWebhookServer(*webhookAddr, *webhookCertFile, *webhookKeyFile)
	if err != nil {
		setupLog.Error(err, "Error creating webhook server", sdklog.ErrorCode("WEBHOOK_CREATE_ERROR"))
		os.Exit(1)
	}

	certExists := false
	keyExists := false
	if _, err := os.Stat(*webhookCertFile); err == nil {
		certExists = true
	}
	if _, err := os.Stat(*webhookKeyFile); err == nil {
		keyExists = true
	}

	if !certExists || !keyExists {
		if !*insecureWebhook {
			setupLog.Error(fmt.Errorf("%w (cert: %s, key: %s). TLS is required for production. Use --insecure-webhook flag only for testing", ErrWebhookTLSCertificatesMissing, *webhookCertFile, *webhookKeyFile), "TLS certificates missing", sdklog.ErrorCode("TLS_CERT_MISSING"))
			os.Exit(1)
		}
		setupLog.Warn("Webhook starting without TLS (insecure mode) - NOT RECOMMENDED FOR PRODUCTION", sdklog.Component("webhook"))
		go func() {
			if err := webhookServer.Start(ctx); err != nil {
				setupLog.Error(err, "Error starting webhook server", sdklog.ErrorCode("WEBHOOK_START_ERROR"))
			}
		}()
		return
	}

	go func() {
		if err := webhookServer.StartTLS(ctx, *webhookCertFile, *webhookKeyFile); err != nil {
			setupLog.Error(err, "Error starting webhook server", sdklog.ErrorCode("WEBHOOK_START_ERROR"))
		}
	}()
	setupLog.Info("Webhook server starting with TLS", sdklog.String("address", *webhookAddr), sdklog.Component("webhook"))
}

// runController runs the controller manager and all components (leader only
// — invoked by the election callback). The reconciler and health checker are
// created in runMain so health endpoints exist on standby replicas too
// (SUPPORT2-032R).
func runController(ctx context.Context, restCfg *rest.Config, mgrOpts *ctrl.Options, reconciler *controller.GCPolicyReconciler, healthChecker *controller.HealthChecker, scheme *runtime.Scheme, dynamicClient dynamic.Interface, statusUpdater *controller.StatusUpdater, eventRecorder *controller.EventRecorder, controllerConfig *config.ControllerConfig) {
	setupLog := logger.WithComponent("controller")

	mgr, err := ctrl.NewManager(restCfg, *mgrOpts)
	if err != nil {
		setupLog.Error(err, "Error creating controller manager", sdklog.ErrorCode("MANAGER_CREATE_ERROR"))
		os.Exit(1)
	}

	// Rebind the reconciler to the manager's cached client and scheme so
	// policy watch/list flows through the shared cache as before. The
	// reconciler keeps the runMain RESTMapper (discovery-backed GVR
	// resolution for irregular CRDs).
	reconciler.Client = mgr.GetClient()
	reconciler.Scheme = mgr.GetScheme()

	// Setup reconciler with manager
	if err := reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Error setting up reconciler", sdklog.ErrorCode("RECONCILER_SETUP_ERROR"))
		os.Exit(1)
	}

	// Register the same checks on the manager as before; the manager's own
	// probe port is disabled ("0") — the standalone server owns :8081 for
	// leader AND standby (SUPPORT2-032R).

	// Add enhanced liveness check (verifies active processing)
	if err := mgr.AddHealthzCheck("healthz", healthChecker.LivenessCheck); err != nil {
		setupLog.Error(err, "Error adding health check", sdklog.ErrorCode("HEALTH_CHECK_ERROR"))
		os.Exit(1)
	}

	// Add enhanced readiness check (verifies informer sync status)
	if err := mgr.AddReadyzCheck("readyz", healthChecker.ReadinessCheck); err != nil {
		setupLog.Error(err, "Error adding readiness check", sdklog.ErrorCode("READY_CHECK_ERROR"))
		os.Exit(1)
	}

	// Add startup check (simple initialization check)
	if err := mgr.AddHealthzCheck("startup", healthChecker.StartupCheck); err != nil {
		setupLog.Error(err, "Error adding startup check", sdklog.ErrorCode("STARTUP_CHECK_ERROR"))
		os.Exit(1)
	}

	// Graceful shutdown is handled by election context
	// The election.RunWithLeaderElection provides the context that cancels on SIGINT/SIGTERM

	// SUPPORT2-032R: the admission webhook is started in runMain for ALL
	// replicas (leader and standby) — see startWebhookServer. Inside the
	// leader-only callback the webhook Service routed admission dials to
	// the standby too, which serves nothing on :9443, so roughly half of
	// all admissions failed with 502 under failurePolicy=Fail.

	// Start the manager (this blocks until context is canceled)
	// mgr.Start() errors are typically non-fatal (e.g., context canceled on shutdown)
	// We don't call os.Exit here to allow graceful shutdown via defer cancel()
	setupLog.Info("Starting GC controller manager", sdklog.Operation("start"))
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "Error starting manager", sdklog.ErrorCode("MANAGER_START_ERROR"))
		// Don't call os.Exit here - let the defer cancel() run for cleanup
		return
	}

	setupLog.Info("GC controller shutdown complete", sdklog.Operation("shutdown"))
}
