package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/1mr0-tech/logcloak/pkg/rules"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "logcloak-controller %s starting\n", version)

	ctrl.SetLogger(zap.New())
	scheme := runtime.NewScheme()
	if err := rules.AddToScheme(scheme); err != nil {
		fmt.Fprintf(os.Stderr, "add scheme: %v\n", err)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme})
	if err != nil {
		fmt.Fprintf(os.Stderr, "new manager: %v\n", err)
		os.Exit(1)
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&rules.MaskingPolicy{}).
		Complete(&maskingPolicyReconciler{Client: mgr.GetClient()}); err != nil {
		fmt.Fprintf(os.Stderr, "setup controller: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "logcloak-controller running\n")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Fprintf(os.Stderr, "manager error: %v\n", err)
		os.Exit(1)
	}
}

type maskingPolicyReconciler struct {
	client.Client
}

func (r *maskingPolicyReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var policy rules.MaskingPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("MaskingPolicy reconciled",
		"name", policy.Name,
		"namespace", policy.Namespace,
		"patterns", len(policy.Spec.Patterns),
	)
	return reconcile.Result{}, nil
}
