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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testObject struct {
	metav1.ObjectMeta
}

func (t *testObject) DeepCopyObject() interface{} {
	return &testObject{
		ObjectMeta: *t.ObjectMeta.DeepCopy(),
	}
}

func TestHasFinalizer(t *testing.T) {
	obj := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{"finalizer1", "finalizer2"},
		},
	}

	if !HasFinalizer(obj, "finalizer1") {
		t.Error("Expected finalizer1 to be present")
	}
	if !HasFinalizer(obj, "finalizer2") {
		t.Error("Expected finalizer2 to be present")
	}
	if HasFinalizer(obj, "finalizer3") {
		t.Error("Expected finalizer3 to not be present")
	}
}

func TestAddFinalizer(t *testing.T) {
	obj := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{"finalizer1"},
		},
	}

	// Add new finalizer
	if !AddFinalizer(obj, "finalizer2") {
		t.Error("Expected AddFinalizer to return true when adding new finalizer")
	}
	if !HasFinalizer(obj, "finalizer2") {
		t.Error("Expected finalizer2 to be added")
	}

	// Try to add existing finalizer
	if AddFinalizer(obj, "finalizer1") {
		t.Error("Expected AddFinalizer to return false when finalizer already exists")
	}
	if len(obj.GetFinalizers()) != 2 {
		t.Errorf("Expected 2 finalizers, got %d", len(obj.GetFinalizers()))
	}
}

func TestRemoveFinalizer(t *testing.T) {
	obj := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{"finalizer1", "finalizer2", "finalizer3"},
		},
	}

	// Remove existing finalizer
	if !RemoveFinalizer(obj, "finalizer2") {
		t.Error("Expected RemoveFinalizer to return true when removing existing finalizer")
	}
	if HasFinalizer(obj, "finalizer2") {
		t.Error("Expected finalizer2 to be removed")
	}
	if !HasFinalizer(obj, "finalizer1") {
		t.Error("Expected finalizer1 to still be present")
	}
	if !HasFinalizer(obj, "finalizer3") {
		t.Error("Expected finalizer3 to still be present")
	}

	// Try to remove non-existent finalizer
	if RemoveFinalizer(obj, "finalizer4") {
		t.Error("Expected RemoveFinalizer to return false when finalizer doesn't exist")
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !ContainsString(slice, "a") {
		t.Error("Expected slice to contain 'a'")
	}
	if !ContainsString(slice, "b") {
		t.Error("Expected slice to contain 'b'")
	}
	if ContainsString(slice, "d") {
		t.Error("Expected slice to not contain 'd'")
	}
	if ContainsString([]string{}, "a") {
		t.Error("Expected empty slice to not contain 'a'")
	}
}

func TestRemoveString(t *testing.T) {
	slice := []string{"a", "b", "c"}

	result := RemoveString(slice, "b")
	if ContainsString(result, "b") {
		t.Error("Expected 'b' to be removed")
	}
	if !ContainsString(result, "a") {
		t.Error("Expected 'a' to still be present")
	}
	if !ContainsString(result, "c") {
		t.Error("Expected 'c' to still be present")
	}
	if len(result) != 2 {
		t.Errorf("Expected result length 2, got %d", len(result))
	}

	// Remove non-existent string
	result2 := RemoveString(slice, "d")
	if len(result2) != len(slice) {
		t.Errorf("Expected result length %d, got %d", len(slice), len(result2))
	}

	// Remove from empty slice
	result3 := RemoveString([]string{}, "a")
	if len(result3) != 0 {
		t.Errorf("Expected empty result, got length %d", len(result3))
	}
}

func TestIsDeleting(t *testing.T) {
	now := metav1.Now()
	obj := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &now,
		},
	}

	if !IsDeleting(obj) {
		t.Error("Expected object to be marked as deleting")
	}

	obj2 := &testObject{
		ObjectMeta: metav1.ObjectMeta{},
	}

	if IsDeleting(obj2) {
		t.Error("Expected object without DeletionTimestamp to not be deleting")
	}
}

func TestEnsureFinalizer(t *testing.T) {
	// This test would require a mock client, so we'll test the logic separately
	// The actual integration test would be in components using this
	obj := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{},
		},
	}

	// Test AddFinalizer logic (which EnsureFinalizer uses)
	if !AddFinalizer(obj, "test-finalizer") {
		t.Error("Expected finalizer to be added")
	}
	if !HasFinalizer(obj, "test-finalizer") {
		t.Error("Expected finalizer to be present after adding")
	}
}

func TestRemoveFinalizerAndUpdate(t *testing.T) {
	// This test would require a mock client, so we'll test the logic separately
	obj := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{"test-finalizer"},
		},
	}

	// Test RemoveFinalizer logic (which RemoveFinalizerAndUpdate uses)
	if !RemoveFinalizer(obj, "test-finalizer") {
		t.Error("Expected finalizer to be removed")
	}
	if HasFinalizer(obj, "test-finalizer") {
		t.Error("Expected finalizer to not be present after removing")
	}
}

func TestFinalizerEdgeCases(t *testing.T) {
	// Test with nil finalizers
	obj := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: nil,
		},
	}

	if HasFinalizer(obj, "test") {
		t.Error("Expected nil finalizers to not contain any finalizer")
	}

	if !AddFinalizer(obj, "test") {
		t.Error("Expected to be able to add finalizer to nil slice")
	}
	if !HasFinalizer(obj, "test") {
		t.Error("Expected finalizer to be added to nil slice")
	}

	// Test removing from nil
	obj2 := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: nil,
		},
	}
	if RemoveFinalizer(obj2, "test") {
		t.Error("Expected RemoveFinalizer to return false for nil slice")
	}
}

func TestMultipleFinalizers(t *testing.T) {
	obj := &testObject{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{"finalizer1", "finalizer2", "finalizer3"},
		},
	}

	// Remove middle finalizer
	RemoveFinalizer(obj, "finalizer2")
	expected := []string{"finalizer1", "finalizer3"}
	if len(obj.GetFinalizers()) != len(expected) {
		t.Errorf("Expected %d finalizers, got %d", len(expected), len(obj.GetFinalizers()))
	}

	// Remove first finalizer
	RemoveFinalizer(obj, "finalizer1")
	if !HasFinalizer(obj, "finalizer3") {
		t.Error("Expected finalizer3 to still be present")
	}

	// Remove last finalizer
	RemoveFinalizer(obj, "finalizer3")
	if len(obj.GetFinalizers()) != 0 {
		t.Errorf("Expected no finalizers, got %d", len(obj.GetFinalizers()))
	}
}
