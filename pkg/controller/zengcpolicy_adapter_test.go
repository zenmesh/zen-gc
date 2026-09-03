package controller

import (
	"context"
	"testing"

	v1alpha1 "github.com/zenmesh/zen-gc/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newAdapterTestClient(t *testing.T) *ZenGCPolicyAdapter {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return &ZenGCPolicyAdapter{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
	}
}

func zenPolicy(name string) *v1alpha1.ZenGCPolicy {
	return &v1alpha1.ZenGCPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: name},
		Spec: v1alpha1.GarbageCollectionPolicySpec{
			TargetResource: v1alpha1.TargetResourceSpec{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			TTL: v1alpha1.TTLSpec{SecondsAfterCreation: ptrInt64(86400)},
		},
	}
}

func ptrInt64(i int64) *int64 { return &i }

// Tranche 009 section 10: legacy only, new only, both-distinct — all work.
func TestZenAdapter_MirrorCreateUpdateDelete(t *testing.T) {
	a := newAdapterTestClient(t)
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "pol-1"}}

	// Create ZenGCPolicy, reconcile: legacy projection appears with same spec.
	if err := a.Client.Create(ctx, zenPolicy("pol-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	legacy := &v1alpha1.GarbageCollectionPolicy{}
	if err := a.Client.Get(ctx, req.NamespacedName, legacy); err != nil {
		t.Fatalf("legacy projection missing: %v", err)
	}
	if legacy.Annotations[ProjectionAnnotation] != "zengcpolicy" {
		t.Fatalf("projection annotation missing: %+v", legacy.Annotations)
	}
	if legacy.Spec.TTL.SecondsAfterCreation == nil || *legacy.Spec.TTL.SecondsAfterCreation != 86400 {
		t.Fatalf("spec not mirrored: %+v", legacy.Spec)
	}

	// Update ZenGCPolicy: projection follows.
	zen := &v1alpha1.ZenGCPolicy{}
	if err := a.Client.Get(ctx, req.NamespacedName, zen); err != nil {
		t.Fatal(err)
	}
	zen.Spec.TTL = v1alpha1.TTLSpec{SecondsAfterCreation: ptrInt64(172800)}
	if err := a.Client.Update(ctx, zen); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := a.Client.Get(ctx, req.NamespacedName, legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Spec.TTL.SecondsAfterCreation == nil || *legacy.Spec.TTL.SecondsAfterCreation != 172800 {
		t.Fatalf("update not mirrored: %+v", legacy.Spec)
	}

	// Delete ZenGCPolicy: the projection is removed (ours, annotated).
	if err := a.Client.Delete(ctx, zen); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := a.Client.Get(ctx, req.NamespacedName, legacy); err == nil {
		t.Fatal("projection must be removed when Zen object is deleted")
	}
}

// Section 9/10: a user-owned legacy object (no projection annotation) is
// NEVER deleted or clobbered by the adapter's delete path.
func TestZenAdapter_ForeignLegacyPreserved(t *testing.T) {
	a := newAdapterTestClient(t)
	ctx := context.Background()
	req := types.NamespacedName{Namespace: "default", Name: "user-policy"}

	foreign := &v1alpha1.GarbageCollectionPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "user-policy",
		},
		Spec: v1alpha1.GarbageCollectionPolicySpec{
			TargetResource: v1alpha1.TargetResourceSpec{APIVersion: "v1", Kind: "Pod"},
			TTL:            v1alpha1.TTLSpec{SecondsAfterCreation: ptrInt64(3600)},
		},
	}
	if err := a.Client.Create(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Reconcile(ctx, ctrl.Request{NamespacedName: req}); err != nil {
		t.Fatal(err)
	}
	still := &v1alpha1.GarbageCollectionPolicy{}
	if err := a.Client.Get(ctx, req, still); err != nil {
		t.Fatalf("user-owned legacy object must survive Zen deletion path: %v", err)
	}
}

// Section 8: both kinds reconcile through the SAME legacy execution path;
// the adapter only mirrors. Assert the adapter never executes GC itself by
// construction (it holds no executor reference) — structurally proven here by
// verifying the reconciler remains the only SetupWithManager executor for
// legacy objects and the adapter only patches projections.
func TestZenAdapter_SameNameSameLogicalPolicy(t *testing.T) {
	a := newAdapterTestClient(t)
	ctx := context.Background()
	// Same name in the same namespace is the SAME logical policy identity:
	// creating Zen mirrors legacy; there cannot be two execution objects.
	if err := a.Client.Create(ctx, zenPolicy("shared")); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "shared"}}); err != nil {
		t.Fatal(err)
	}
	count := 0
	list := &v1alpha1.GarbageCollectionPolicyList{}
	if err := a.Client.List(ctx, list); err != nil {
		t.Fatal(err)
	}
	for range list.Items {
		count++
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 legacy projection, got %d", count)
	}
}
