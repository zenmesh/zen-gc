package controller

import (
	"testing"
)

func TestNewInformerStoreResourceLister(t *testing.T) {
	lister := NewInformerStoreResourceLister(nil)
	if lister == nil {
		t.Fatal("Expected non-nil lister")
	}
}

func TestGCPolicyReconcilerAdapter(t *testing.T) {
	// Test that the adapter can be created with nil (for testing purposes)
	adapter := &GCPolicyReconcilerAdapter{}
	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}
}

func TestNewGCPolicyReconcilerAdapter(t *testing.T) {
	adapter := NewGCPolicyReconcilerAdapter(nil)
	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}
	if adapter.reconciler != nil {
		t.Error("Expected nil reconciler")
	}
}