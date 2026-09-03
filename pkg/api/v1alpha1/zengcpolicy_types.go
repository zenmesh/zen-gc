// Copyright 2026 The Zen Mesh Authors.
// Licensed under the Apache License, version 2.0.

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ZenGCPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GarbageCollectionPolicySpec   `json:"spec,omitempty"`
	Status GarbageCollectionPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ZenGCPolicyList is the collection of ZenGCPolicy objects.
type ZenGCPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZenGCPolicy `json:"items"`
}
