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

package lifecycle

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zenmesh/zen-gc/internal/logging"
)

// DefaultShutdownTimeout is the default timeout for graceful shutdown
const DefaultShutdownTimeout = 30 * time.Second

// ShutdownContext creates a context that cancels on SIGINT/SIGTERM.
// This is a wrapper around signal.NotifyContext() with logging.
//
// Usage:
//
//	ctx, cancel := lifecycle.ShutdownContext(context.Background(), "my-component")
//	defer cancel()
//
//	// Use ctx in your application
//	<-ctx.Done() // Blocks until shutdown signal received
func ShutdownContext(ctx context.Context, component string) (context.Context, context.CancelFunc) {
	logger := logging.NewLogger(component)

	shutdownCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)

	// Log when shutdown signal is received
	go func() {
		<-shutdownCtx.Done()
		logger.Info("Shutdown signal received",
			logging.Operation("shutdown_signal"),
			logging.String("component", component))
	}()

	return shutdownCtx, cancel
}

// ShutdownHTTPServer gracefully shuts down an HTTP server.
// It waits for active connections to finish or times out after the specified duration.
//
// Usage:
//
//	server := &http.Server{Addr: ":8080", Handler: mux}
//	go server.ListenAndServe()
//
//	<-shutdownCtx.Done()
//	if err := lifecycle.ShutdownHTTPServer(shutdownCtx, server, "my-component", 30*time.Second); err != nil {
//	    // Handle error
//	}
func ShutdownHTTPServer(ctx context.Context, server *http.Server, component string, timeout time.Duration) error {
	logger := logging.NewLogger(component)

	if timeout == 0 {
		timeout = DefaultShutdownTimeout
	}

	logger.Info("Shutting down HTTP server...",
		logging.Operation("shutdown"),
		logging.Duration("timeout", timeout))

	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(err, "HTTP server shutdown error",
			logging.Operation("shutdown"),
			logging.ErrorCode("SHUTDOWN_ERROR"))
		return err
	}

	logger.Info("HTTP server shut down gracefully",
		logging.Operation("shutdown_complete"))

	return nil
}

// GRPCServer is an interface for gRPC servers that support graceful shutdown.
// This matches the interface provided by google.golang.org/grpc.Server.
type GRPCServer interface {
	GracefulStop()
	Stop()
}

// ShutdownGRPCServer gracefully shuts down a gRPC server.
// GracefulStop() blocks until all RPCs are finished.
//
// Usage:
//
//	grpcServer := grpc.NewServer()
//	// ... register services ...
//
//	<-shutdownCtx.Done()
//	lifecycle.ShutdownGRPCServer(grpcServer, "my-component")
func ShutdownGRPCServer(server GRPCServer, component string) {
	logger := logging.NewLogger(component)

	logger.Info("Shutting down gRPC server...",
		logging.Operation("shutdown"))

	// GracefulStop() blocks until all RPCs are finished
	server.GracefulStop()

	logger.Info("gRPC server shut down gracefully",
		logging.Operation("shutdown_complete"))
}

// WaitForShutdown waits for a context to be cancelled and optionally runs cleanup.
// This is useful for worker services that need to wait for goroutines to finish.
//
// Usage:
//
//	var wg sync.WaitGroup
//	wg.Add(1)
//	go func() {
//	    defer wg.Done()
//	    runWorker(ctx)
//	}()
//
//	lifecycle.WaitForShutdown(ctx, &wg, func() {
//	    // Cleanup code
//	})
func WaitForShutdown(ctx context.Context, component string, cleanup func()) {
	logger := logging.NewLogger(component)

	<-ctx.Done()

	logger.Info("Waiting for shutdown to complete...",
		logging.Operation("shutdown_wait"))

	if cleanup != nil {
		cleanup()
	}

	logger.Info("Shutdown complete",
		logging.Operation("shutdown_complete"))
}
