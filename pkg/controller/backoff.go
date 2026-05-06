// Package controller implements the garbage collection controller.
package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/zenmesh/zen-gc/pkg/api/v1alpha1"
	"github.com/zenmesh/zen-gc/internal/ratelimiter"
)

// DeleteResourceWithBackoff deletes a resource with exponential backoff retry logic.
// This is a convenience wrapper for GCPolicyReconciler.
func DeleteResourceWithBackoff(ctx context.Context, reconciler *GCPolicyReconciler, resource *unstructured.Unstructured, policy *v1alpha1.GarbageCollectionPolicy, rateLimiter *ratelimiter.RateLimiter) error {
	return deleteResourceWithBackoff(ctx, reconciler, resource, policy, rateLimiter)
}

// deleteResourceWithBackoff is the internal implementation.
func deleteResourceWithBackoff(ctx context.Context, reconciler *GCPolicyReconciler, resource *unstructured.Unstructured, policy *v1alpha1.GarbageCollectionPolicy, rateLimiter *ratelimiter.RateLimiter) error {
	// Use the deleter from GCPolicyReconciler
	return reconciler.deleteResource(ctx, resource, policy, rateLimiter)
}
