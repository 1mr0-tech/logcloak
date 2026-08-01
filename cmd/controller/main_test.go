package main

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/1mr0-tech/logcloak/pkg/rules"
)

func newScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := rules.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	return scheme
}

func TestReconcileNotFound(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := &maskingPolicyReconciler{Client: c}

	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("expected no error for missing policy, got %v", err)
	}
	if res.Requeue {
		t.Fatalf("expected no requeue for missing policy")
	}
}

func TestReconcileFound(t *testing.T) {
	policy := &rules.MaskingPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"},
		Spec: rules.MaskingPolicySpec{
			Patterns: []rules.PatternSpec{{Name: "email", Builtin: "email"}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(policy).Build()
	r := &maskingPolicyReconciler{Client: c}

	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "example", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Requeue {
		t.Fatalf("expected no requeue")
	}
}
