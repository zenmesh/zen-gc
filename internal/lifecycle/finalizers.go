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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HasFinalizer checks if an object has the specified finalizer.
func HasFinalizer(obj metav1.Object, finalizer string) bool {
	return ContainsString(obj.GetFinalizers(), finalizer)
}

// AddFinalizer adds a finalizer to an object if it doesn't already exist.
// Returns true if the finalizer was added, false if it already existed.
func AddFinalizer(obj metav1.Object, finalizer string) bool {
	if HasFinalizer(obj, finalizer) {
		return false
	}
	obj.SetFinalizers(append(obj.GetFinalizers(), finalizer))
	return true
}

// RemoveFinalizer removes a finalizer from an object if it exists.
// Returns true if the finalizer was removed, false if it didn't exist.
func RemoveFinalizer(obj metav1.Object, finalizer string) bool {
	if !HasFinalizer(obj, finalizer) {
		return false
	}
	obj.SetFinalizers(RemoveString(obj.GetFinalizers(), finalizer))
	return true
}

// ContainsString checks if a string slice contains a specific string.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveString removes a string from a slice and returns the new slice.
func RemoveString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// IsDeleting checks if an object is being deleted (has DeletionTimestamp set).
func IsDeleting(obj metav1.Object) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

// EnsureFinalizer ensures a finalizer is present on an object and updates it if needed.
// This is a convenience function that combines AddFinalizer with a client update.
func EnsureFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) error {
	if AddFinalizer(obj, finalizer) {
		return c.Update(ctx, obj)
	}
	return nil
}

// RemoveFinalizerAndUpdate removes a finalizer from an object and updates it.
// This is a convenience function that combines RemoveFinalizer with a client update.
func RemoveFinalizerAndUpdate(ctx context.Context, c client.Client, obj client.Object, finalizer string) error {
	if RemoveFinalizer(obj, finalizer) {
		return c.Update(ctx, obj)
	}
	return nil
}
