// Copyright 2026 The Zen Mesh Authors.
// Licensed under the Apache License, version 2.0.

package controller

import (
	"context"

	v1alpha1 "github.com/zenmesh/zen-gc/pkg/api/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ProjectionAnnotation marks the legacy GarbageCollectionPolicy object as the
// 1:1 execution projection of a ZenGCPolicy object (same namespace, same name).
//
// DOUBLE-EXECUTION SAFETY (tranche 009 section 9): the ZenGCPolicy adapter
// controller does NOT execute GC itself. It projects the policy onto the
// legacy kind, and the EXISTING single GarbageCollectionPolicy reconciler
// remains the only executor. A ZenGCPolicy and its legacy projection are one
// logical policy by construction (same namespace/name), so duplicate
// execution is impossible without bypassing the adapter.
const ProjectionAnnotation = "gc.ops.zen-mesh.io/projected-from"

// ZenGCPolicyAdapter reconciles ZenGCPolicy objects by mirroring them onto
// legacy GarbageCollectionPolicy projections. The legacy kind stays the
// execution authority during the compatibility window; this adapter owns the
// Zen-named surface only.
type ZenGCPolicyAdapter struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager registers the adapter controller. Reconcile is event-driven
// on ZenGCPolicy create/update/delete/generation changes only.
func (a *ZenGCPolicyAdapter) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ZenGCPolicy{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("zengcpolicy-adapter").
		Complete(a)
}

// +kubebuilder:rbac:groups=gc.ops.zen-mesh.io,resources=garbagecollectionpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile mirrors one ZenGCPolicy onto its legacy projection:
//   - create/update: upsert the legacy object with the same spec, annotated
//     as projected (deterministic same-namespace/same-name mapping)
//   - delete: delete the legacy projection (never unrelated objects)
//
// The legacy reconciler continues to own validation, execution, and status.
func (a *ZenGCPolicyAdapter) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	zen := &v1alpha1.ZenGCPolicy{}
	err := a.Client.Get(ctx, req.NamespacedName, zen)
	if k8serrors.IsNotFound(err) {
		// Zen object deleted: remove the legacy projection if it is ours.
		legacy := &v1alpha1.GarbageCollectionPolicy{}
		err = a.Client.Get(ctx, req.NamespacedName, legacy)
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		if err != nil {
			return ctrl.Result{}, err
		}
		if legacy.Annotations[ProjectionAnnotation] == "zengcpolicy" {
			if err := a.Client.Delete(ctx, legacy); err != nil && !k8serrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	if !zen.DeletionTimestamp.IsZero() {
		// Deletion handled by the NotFound branch once finalizers clear.
		return ctrl.Result{}, nil
	}

	legacy := &v1alpha1.GarbageCollectionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gc.ops.zen-mesh.io/v1alpha1",
			Kind:       "GarbageCollectionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: zen.Namespace,
			Name:      zen.Name,
			Annotations: map[string]string{
				ProjectionAnnotation: "zengcpolicy",
			},
		},
		Spec: zen.Spec,
	}
	existing := &v1alpha1.GarbageCollectionPolicy{}
	getErr := a.Client.Get(ctx, req.NamespacedName, existing)
	if getErr == nil {
		existing.Spec = zen.Spec
		if err := a.Client.Update(ctx, existing); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if !k8serrors.IsNotFound(getErr) {
		return ctrl.Result{}, getErr
	}
	if err := a.Client.Create(ctx, legacy); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
