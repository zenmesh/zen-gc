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

package ttl

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCalculateExpirationTime_FixedTTL(t *testing.T) {
	creationTime := time.Now().Add(-2 * time.Hour)
	resource := &unstructured.Unstructured{}
	resource.SetCreationTimestamp(metav1.Time{Time: creationTime})

	ttlSeconds := int64(3600) // 1 hour
	spec := &Spec{
		SecondsAfterCreation: &ttlSeconds,
	}

	expirationTime, err := CalculateExpirationTime(resource, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := creationTime.Add(1 * time.Hour)
	if !expirationTime.Truncate(time.Second).Equal(expected.Truncate(time.Second)) {
		t.Errorf("expected %v, got %v", expected, expirationTime)
	}
}

func TestCalculateExpirationTime_DynamicTTL(t *testing.T) {
	creationTime := time.Now().Add(-30 * time.Minute)
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"ttlSeconds": int64(3600),
			},
		},
	}
	resource.SetCreationTimestamp(metav1.Time{Time: creationTime})

	spec := &Spec{
		FieldPath: "spec.ttlSeconds",
	}

	expirationTime, err := CalculateExpirationTime(resource, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := creationTime.Add(1 * time.Hour)
	if !expirationTime.Truncate(time.Second).Equal(expected.Truncate(time.Second)) {
		t.Errorf("expected %v, got %v", expected, expirationTime)
	}
}

func TestCalculateExpirationTime_MappedTTL(t *testing.T) {
	creationTime := time.Now().Add(-2 * time.Hour)
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"severity": "critical",
			},
		},
	}
	resource.SetCreationTimestamp(metav1.Time{Time: creationTime})

	spec := &Spec{
		FieldPath: "spec.severity",
		Mappings: map[string]int64{
			"critical": 86400, // 24 hours
			"normal":   3600,  // 1 hour
		},
	}

	expirationTime, err := CalculateExpirationTime(resource, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := creationTime.Add(24 * time.Hour)
	if !expirationTime.Truncate(time.Second).Equal(expected.Truncate(time.Second)) {
		t.Errorf("expected %v, got %v", expected, expirationTime)
	}
}

func TestCalculateExpirationTime_MappedTTL_WithDefault(t *testing.T) {
	creationTime := time.Now().Add(-30 * time.Minute)
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"severity": "unknown",
			},
		},
	}
	resource.SetCreationTimestamp(metav1.Time{Time: creationTime})

	defaultTTL := int64(1800) // 30 minutes
	spec := &Spec{
		FieldPath: "spec.severity",
		Mappings: map[string]int64{
			"critical": 86400,
			"normal":   3600,
		},
		Default: &defaultTTL,
	}

	expirationTime, err := CalculateExpirationTime(resource, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := creationTime.Add(30 * time.Minute)
	if !expirationTime.Truncate(time.Second).Equal(expected.Truncate(time.Second)) {
		t.Errorf("expected %v, got %v", expected, expirationTime)
	}
}

func TestCalculateExpirationTime_RelativeTTL(t *testing.T) {
	lastProcessed := time.Now().Add(-30 * time.Minute)
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"lastProcessedAt": lastProcessed.Format(time.RFC3339),
			},
		},
	}

	secondsAfter := int64(3600) // 1 hour after lastProcessedAt
	spec := &Spec{
		RelativeTo:   "status.lastProcessedAt",
		SecondsAfter: &secondsAfter,
	}

	expirationTime, err := CalculateExpirationTime(resource, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := lastProcessed.Add(1 * time.Hour)
	if !expirationTime.Truncate(time.Second).Equal(expected.Truncate(time.Second)) {
		t.Errorf("expected %v, got %v", expected, expirationTime)
	}
}

func TestCalculateExpirationTime_RelativeTTL_Expired(t *testing.T) {
	lastProcessed := time.Now().Add(-2 * time.Hour)
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"lastProcessedAt": lastProcessed.Format(time.RFC3339),
			},
		},
	}

	secondsAfter := int64(3600) // 1 hour after lastProcessedAt (already expired)
	spec := &Spec{
		RelativeTo:   "status.lastProcessedAt",
		SecondsAfter: &secondsAfter,
	}

	_, err := CalculateExpirationTime(resource, spec)
	if err == nil || err.Error() != ErrRelativeTTLExpired.Error() {
		t.Errorf("expected ErrRelativeTTLExpired, got %v", err)
	}
}

func TestIsExpired_NotExpired(t *testing.T) {
	creationTime := time.Now().Add(-30 * time.Minute)
	resource := &unstructured.Unstructured{}
	resource.SetCreationTimestamp(metav1.Time{Time: creationTime})

	ttlSeconds := int64(3600) // 1 hour
	spec := &Spec{
		SecondsAfterCreation: &ttlSeconds,
	}

	expired, err := IsExpired(resource, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if expired {
		t.Error("expected resource to not be expired")
	}
}

func TestIsExpired_Expired(t *testing.T) {
	creationTime := time.Now().Add(-2 * time.Hour)
	resource := &unstructured.Unstructured{}
	resource.SetCreationTimestamp(metav1.Time{Time: creationTime})

	ttlSeconds := int64(3600) // 1 hour
	spec := &Spec{
		SecondsAfterCreation: &ttlSeconds,
	}

	expired, err := IsExpired(resource, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !expired {
		t.Error("expected resource to be expired")
	}
}

func TestCalculateExpirationTime_NoValidConfig(t *testing.T) {
	resource := &unstructured.Unstructured{}

	spec := &Spec{} // No TTL configuration

	_, err := CalculateExpirationTime(resource, spec)
	if err == nil || err.Error() != ErrNoValidTTLConfiguration.Error() {
		t.Errorf("expected ErrNoValidTTLConfiguration, got %v", err)
	}
}

func TestCalculateExpirationTime_FieldPathNotFound(t *testing.T) {
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{},
		},
	}

	spec := &Spec{
		FieldPath: "spec.ttlSeconds",
	}

	_, err := CalculateExpirationTime(resource, spec)
	if err == nil {
		t.Error("expected error for field path not found")
	}
}
