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
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestShutdownContext(t *testing.T) {
	ctx := context.Background()
	shutdownCtx, cancel := ShutdownContext(ctx, "test-component")
	defer cancel()

	// Context should not be done initially
	select {
	case <-shutdownCtx.Done():
		t.Fatal("Context should not be done initially")
	default:
		// Expected
	}

	// Cancel should trigger shutdown
	cancel()

	// Context should be done after cancel
	select {
	case <-shutdownCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be done after cancel")
	}
}

func TestShutdownHTTPServer(t *testing.T) {
	// Create a test server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":0", // Let OS choose port
		Handler: mux,
	}

	// Start server in background
	var serverErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		serverErr = server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create shutdown context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Shutdown should succeed
	err := ShutdownHTTPServer(ctx, server, "test-component", 5*time.Second)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Wait for server to stop
	wg.Wait()

	// Server should have stopped with ErrServerClosed
	if serverErr != nil && serverErr != http.ErrServerClosed {
		t.Fatalf("Expected ErrServerClosed, got: %v", serverErr)
	}
}

func TestShutdownHTTPServer_Timeout(t *testing.T) {
	// Create a test server that takes a long time to respond
	mux := http.NewServeMux()
	requestStarted := make(chan bool, 1) // Buffered to prevent blocking
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		select {
		case requestStarted <- true:
		default:
		}
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: mux,
	}

	// Start server
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = server.Serve(listener)
	}()

	// Make a request that will block
	client := &http.Client{Timeout: 5 * time.Second}
	var clientWg sync.WaitGroup
	clientWg.Add(1)
	go func() {
		defer clientWg.Done()
		_, _ = client.Get("http://" + listener.Addr().String() + "/")
	}()

	// Wait for request to start (with timeout)
	select {
	case <-requestStarted:
		// Request started
	case <-time.After(2 * time.Second):
		t.Fatal("Request did not start within timeout")
	}
	time.Sleep(10 * time.Millisecond)

	// Create a context with timeout (not already cancelled)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Shutdown with very short timeout should fail
	err = ShutdownHTTPServer(ctx, server, "test-component", 100*time.Millisecond)
	if err == nil {
		t.Error("Expected error for timeout, got nil")
	}

	// Force stop the server
	_ = server.Close()
	wg.Wait()
	clientWg.Wait()
}

func TestShutdownHTTPServer_DefaultTimeout(t *testing.T) {
	server := &http.Server{
		Addr:    ":0",
		Handler: http.NewServeMux(),
	}

	// Start and immediately stop
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = server.ListenAndServe()
	}()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Shutdown with 0 timeout should use default
	err := ShutdownHTTPServer(ctx, server, "test-component", 0)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	wg.Wait()
}

func TestWaitForShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var cleanupCalled bool
	cleanup := func() {
		cleanupCalled = true
	}

	// Start WaitForShutdown in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		WaitForShutdown(ctx, "test-component", cleanup)
	}()

	// Give it a moment
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for shutdown to complete
	wg.Wait()

	// Cleanup should have been called
	if !cleanupCalled {
		t.Error("Cleanup function should have been called")
	}
}

func TestWaitForShutdown_NoCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Should not panic with nil cleanup
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		WaitForShutdown(ctx, "test-component", nil)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()
}
